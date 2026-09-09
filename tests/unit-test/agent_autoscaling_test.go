package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

type scaledObjectScaleTargetRef struct {
	Name string `json:"name"`
}

// scaledObject captures only the fields these tests assert on - there is no k8s.io/api type for a
// KEDA CRD, and the full KEDA client types are not worth adding as a dependency for this. Shared
// with vortex_autoscaling_test.go.
type scaledObject struct {
	Metadata scaledObjectMetadata `json:"metadata"`
	Spec     scaledObjectSpec     `json:"spec"`
}

type scaledObjectMetadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

type scaledObjectSpec struct {
	ScaleTargetRef  scaledObjectScaleTargetRef `json:"scaleTargetRef"`
	MinReplicaCount int64                      `json:"minReplicaCount"`
	MaxReplicaCount int64                      `json:"maxReplicaCount"`
	PollingInterval int64                      `json:"pollingInterval"`
	Advanced        scaledObjectAdvanced       `json:"advanced"`
	Triggers        []scaledObjectTrigger      `json:"triggers"`
}

type scaledObjectAdvanced struct {
	HorizontalPodAutoscalerConfig scaledObjectHPAConfig `json:"horizontalPodAutoscalerConfig"`
}

// Name is the HPA's own overridden object name (SONAR-32220 pins this to stay within the 63-char
// object-name limit); Behavior is Vortex's scale-down stabilization window (SONAR-32198).
type scaledObjectHPAConfig struct {
	Name     string               `json:"name"`
	Behavior scaledObjectBehavior `json:"behavior"`
}

type scaledObjectBehavior struct {
	ScaleDown scaledObjectScaleDown `json:"scaleDown"`
}

type scaledObjectScaleDown struct {
	StabilizationWindowSeconds int64 `json:"stabilizationWindowSeconds"`
}

type scaledObjectTrigger struct {
	Type              string            `json:"type"`
	Metadata          map[string]string `json:"metadata"`
	AuthenticationRef interface{}       `json:"authenticationRef"`
}

func renderAgentOrchestratorHPA(t *testing.T, chart agentChart, setValues map[string]string) (string, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/agent-orchestrator-enabled.yaml"},
		SetValues:   setValues,
	}
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator-hpa.yaml"})
}

func renderAgentOrchestratorDeployment(t *testing.T, chart agentChart, setValues map[string]string, extraArgs ...string) (appsv1.Deployment, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/agent-orchestrator-enabled.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator.yaml"}, extraArgs...)
	if err != nil {
		return appsv1.Deployment{}, err
	}
	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	return deployment, nil
}

// Both hunter and remediation are enabled in the agent-runtimes-enabled.yaml fixture, so the
// render emits one Deployment per family; pick out the one for family.
func renderAgentRuntimeDeploymentFamily(t *testing.T, chart agentChart, family string, setValues map[string]string, extraArgs ...string) (appsv1.Deployment, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/agent-runtimes-enabled.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-runtime.yaml"}, extraArgs...)
	if err != nil {
		return appsv1.Deployment{}, err
	}
	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" || !strings.Contains(doc, "kind: Deployment") {
			continue
		}
		var deployment appsv1.Deployment
		helm.UnmarshalK8SYaml(t, doc, &deployment)
		if deployment.Labels["sonarqube.agent/family"] == family {
			return deployment, nil
		}
	}
	require.FailNowf(t, "no Deployment rendered", "family %q", family)
	return appsv1.Deployment{}, nil
}

func renderAgentRuntimeScaledObject(t *testing.T, chart agentChart, family string, setValues map[string]string) (scaledObject, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/agent-runtimes-enabled.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-runtime-scaledobject.yaml"})
	if err != nil {
		return scaledObject{}, err
	}
	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" || !strings.Contains(doc, "kind: ScaledObject") {
			continue
		}
		var so scaledObject
		helm.UnmarshalK8SYaml(t, doc, &so)
		if so.Metadata.Labels["sonarqube.agent/family"] == family {
			return so, nil
		}
	}
	require.FailNowf(t, "no ScaledObject rendered", "family %q", family)
	return scaledObject{}, nil
}

// assertReplicasOmittedOnUpgrade renders a Deployment via render on a fresh install and again on
// upgrade with the same values, and asserts replicas is set on install but omitted on upgrade -
// the pattern shared by the orchestrator, each agent runtime family, and Vortex once their
// respective autoscaler owns spec.replicas. Used by both this file and
// vortex_autoscaling_test.go.
func assertReplicasOmittedOnUpgrade(t *testing.T, render func(extraArgs ...string) (appsv1.Deployment, error)) {
	t.Helper()
	install, err := render()
	require.NoError(t, err)
	require.NotNil(t, install.Spec.Replicas, "replicas should render on a fresh install")

	upgrade, err := render("--is-upgrade")
	require.NoError(t, err)
	assert.Nil(t, upgrade.Spec.Replicas, "replicas should be omitted on upgrade once the autoscaler owns it")
}

// assertReplicasOmittedWhenManageReplicasFalse is the GitOps-escape-hatch counterpart to
// assertReplicasOmittedOnUpgrade: Release.IsInstall is always true under `helm template` (Argo CD,
// Flux, --dry-run=client), so without manageReplicas=false those consumers would always hit the
// "fresh install" branch above and keep resetting replicas on every sync. Used by both this file
// and vortex_autoscaling_test.go.
func assertReplicasOmittedWhenManageReplicasFalse(t *testing.T, render func() (appsv1.Deployment, error)) {
	t.Helper()
	install, err := render()
	require.NoError(t, err)
	assert.Nil(t, install.Spec.Replicas, "replicas should be omitted even on install when manageReplicas=false")
}

// assertReplicasRenderedWhenAutoscalingDisabled checks that manageReplicas only gates the
// autoscaling-enabled branch, not replicas rendering whenever a shared values layer sets it false
// regardless of autoscaling.enabled. Used by both this file and vortex_autoscaling_test.go.
func assertReplicasRenderedWhenAutoscalingDisabled(t *testing.T, render func() (appsv1.Deployment, error)) {
	t.Helper()
	deployment, err := render()
	require.NoError(t, err)
	require.NotNil(t, deployment.Spec.Replicas, "replicas must still render when autoscaling is disabled, regardless of manageReplicas")
}

// assertAutoscalingRequiresKedaCRDOrOverride exercises the KEDA CRD guard in validation.yaml,
// shared by every autoscaled component: enabling autoscaling without the KEDA CRDs present (and
// no explicit agentKeda.assumeInstalled override) fails; --api-versions simulates the CRD being
// registered on a real cluster (Capabilities.APIVersions is otherwise empty under `helm
// template`). Used by both this file and vortex_autoscaling_test.go.
func assertAutoscalingRequiresKedaCRDOrOverride(t *testing.T, opts *helm.Options, chart agentChart, template string, errSubstring string) {
	t.Helper()
	t.Run("no KEDA CRD, no override: fails", func(t *testing.T) {
		_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{template})
		require.Error(t, err)
		assert.Contains(t, err.Error(), errSubstring)
	})

	t.Run("KEDA CRD present via --api-versions: succeeds", func(t *testing.T) {
		_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{template}, "--api-versions=keda.sh/v1alpha1")
		require.NoError(t, err)
	})
}

// Helm's `--show-only` errors ("could not find template ... in chart") rather than returning empty
// when the named template renders zero documents, so "not rendered" must be asserted against a full
// chart render instead of a --show-only'd one.
func TestAgentOrchestratorHPANotRenderedByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/agent-orchestrator-enabled.yaml"},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{})
			require.NoError(t, err)
			assert.NotContains(t, output, "agent-orchestrator-hpa.yaml")
		})
	}
}

func TestAgentOrchestratorHPARendersWhenEnabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderAgentOrchestratorHPA(t, chart, map[string]string{
				"agentOrchestrator.autoscaling.enabled":     "true",
				"agentOrchestrator.autoscaling.minReplicas": "3",
				"agentOrchestrator.autoscaling.maxReplicas": "7",
			})
			require.NoError(t, err)

			var hpa autoscalingv2.HorizontalPodAutoscaler
			helm.UnmarshalK8SYaml(t, output, &hpa)
			assert.Equal(t, chart.fullnamePrefix()+"-agent-orchestrator", hpa.Name)
			assert.Equal(t, chart.fullnamePrefix()+"-agent-orchestrator", hpa.Spec.ScaleTargetRef.Name)
			require.NotNil(t, hpa.Spec.MinReplicas)
			assert.EqualValues(t, 3, *hpa.Spec.MinReplicas)
			assert.EqualValues(t, 7, hpa.Spec.MaxReplicas)
			require.Len(t, hpa.Spec.Metrics, 1)
			assert.Equal(t, autoscalingv2.ResourceMetricSourceType, hpa.Spec.Metrics[0].Type)
			require.NotNil(t, hpa.Spec.Behavior)
			require.NotNil(t, hpa.Spec.Behavior.ScaleDown)
			assert.EqualValues(t, 300, *hpa.Spec.Behavior.ScaleDown.StabilizationWindowSeconds)
		})
	}
}

// Once the HPA owns replicas, a fresh install still gets a sane initial count (Release.IsInstall is
// true by default under `helm template`), but a later `helm upgrade` must not fight the HPA.
func TestAgentOrchestratorReplicasOmittedOnUpgradeWhenAutoscalingEnabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{"agentOrchestrator.autoscaling.enabled": "true"}
			assertReplicasOmittedOnUpgrade(t, func(extraArgs ...string) (appsv1.Deployment, error) {
				return renderAgentOrchestratorDeployment(t, chart, setValues, extraArgs...)
			})
		})
	}
}

// manageReplicas=false is the GitOps escape hatch: Release.IsInstall is always true under `helm
// template` (Argo CD, Flux, --dry-run=client), so without it those consumers would always hit the
// "fresh install" branch above and keep resetting replicas on every sync.
func TestAgentOrchestratorReplicasSuppressedByManageReplicasFalse(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{
				"agentOrchestrator.autoscaling.enabled":        "true",
				"agentOrchestrator.autoscaling.manageReplicas": "false",
			}
			assertReplicasOmittedWhenManageReplicasFalse(t, func() (appsv1.Deployment, error) {
				return renderAgentOrchestratorDeployment(t, chart, setValues)
			})
		})
	}
}

// manageReplicas must only gate the autoscaling-enabled branch, not suppress replicas whenever a
// shared values layer sets it false regardless of autoscaling.enabled.
func TestAgentOrchestratorReplicasRenderedWhenAutoscalingDisabledEvenIfManageReplicasFalse(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			assertReplicasRenderedWhenAutoscalingDisabled(t, func() (appsv1.Deployment, error) {
				return renderAgentOrchestratorDeployment(t, chart, map[string]string{
					"agentOrchestrator.autoscaling.enabled":        "false",
					"agentOrchestrator.autoscaling.manageReplicas": "false",
				})
			})
		})
	}
}

func TestAgentRuntimeScaledObjectNotRenderedByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/agent-runtimes-enabled.yaml"},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{})
			require.NoError(t, err)
			assert.NotContains(t, output, "agent-runtime-scaledobject.yaml")
		})
	}
}

func TestAgentRuntimeScaledObjectRendersWhenEnabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					so, err := renderAgentRuntimeScaledObject(t, chart, family, map[string]string{
						family + "Agent.autoscaling.enabled":         "true",
						family + "Agent.autoscaling.minReplicas":     "4",
						family + "Agent.autoscaling.maxReplicas":     "9",
						family + "Agent.autoscaling.pollingInterval": "20",
						"agentKeda.assumeInstalled":                  "true",
					})
					require.NoError(t, err)

					expectedName := chart.fullnamePrefix() + "-agent-runtime-" + family
					assert.Equal(t, expectedName, so.Metadata.Name)
					assert.Equal(t, expectedName, so.Spec.ScaleTargetRef.Name)
					assert.Equal(t, expectedName, so.Spec.Advanced.HorizontalPodAutoscalerConfig.Name)
					assert.EqualValues(t, 4, so.Spec.MinReplicaCount)
					assert.EqualValues(t, 9, so.Spec.MaxReplicaCount)
					assert.EqualValues(t, 20, so.Spec.PollingInterval)

					require.Len(t, so.Spec.Triggers, 1)
					trigger := so.Spec.Triggers[0]
					assert.Equal(t, "metrics-api", trigger.Type)
					assert.Contains(t, trigger.Metadata["url"], "/metrics/queue")
					assert.Equal(t, family+".unfinished", trigger.Metadata["valueLocation"])
				})
			}
		})
	}
}

// KEDA defaults the underlying HPA's name to "keda-hpa-<scaledobject name>" - that 9-char prefix
// can push the *HPA's* name past the 63-char Kubernetes name limit even when the ScaledObject's
// own (already-truncated-to-63) name fits, and KEDA's admission webhook rejects the ScaledObject
// outright when that happens. horizontalPodAutoscalerConfig.name must be pinned to the
// ScaledObject's own name so no extra prefix is ever added.
func TestAgentRuntimeScaledObjectHPANameStaysWithinLimit(t *testing.T) {
	const longRelease = "next-prod-linas"
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			so, err := renderAgentRuntimeScaledObject(t, agentChart{
				name:      chart.name,
				path:      chart.path,
				release:   longRelease,
				valuesDir: chart.valuesDir,
			}, "remediation", map[string]string{
				"remediationAgent.autoscaling.enabled": "true",
				"agentKeda.assumeInstalled":            "true",
			})
			require.NoError(t, err)

			assert.LessOrEqual(t, len(so.Spec.Advanced.HorizontalPodAutoscalerConfig.Name), 63)
			assert.Equal(t, so.Metadata.Name, so.Spec.Advanced.HorizontalPodAutoscalerConfig.Name)
		})
	}
}

// The KEDA CRD guard in validation.yaml: enabling a family's autoscaling without the KEDA CRDs
// present (and no explicit agentKeda.assumeInstalled override) fails; --api-versions simulates the
// CRD being registered on a real cluster (Capabilities.APIVersions is otherwise empty under `helm
// template`).
func TestAgentRuntimeAutoscalingRequiresKedaCRDOrOverride(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			base := agentValidationBase(chart)
			base["agentOrchestrator.enabled"] = "true"
			base["agentOrchestrator.image.repository"] = "example.com/agent-orchestrator"
			base["hunterAgent.enabled"] = "true"
			base["hunterAgent.image.repository"] = "example.com/hunter-agent"
			base["hunterAgent.autoscaling.enabled"] = "true"
			base["agenticSigningSecret.existingSecret"] = "test-agentic-instance-secret"
			opts := &helm.Options{Logger: logger.Discard, SetValues: base}
			assertAutoscalingRequiresKedaCRDOrOverride(t, opts, chart, "templates/agent-orchestrator.yaml",
				"hunterAgent.autoscaling.enabled is true but the KEDA CRDs")
		})
	}
}

// assertAgentAutoscalingRangeRejected renders each chart/family with autoscalingValues layered onto
// an otherwise-valid enable of that component, and asserts the render fails containing
// "<component>.autoscaling.<errSuffix>" - shared by the min-floor and max-below-min checks below,
// which only differ in which fields are invalid and what the error names.
func assertAgentAutoscalingRangeRejected(t *testing.T, autoscalingValues map[string]string, errSuffix string) {
	t.Helper()
	prefixed := func(prefix string) map[string]string {
		values := make(map[string]string, len(autoscalingValues))
		for k, v := range autoscalingValues {
			values[prefix+".autoscaling."+k] = v
		}
		return values
	}

	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("orchestrator", func(t *testing.T) {
				values := prefixed("agentOrchestrator")
				values["agentOrchestrator.enabled"] = "true"
				values["agentOrchestrator.image.repository"] = "example.com/agent-orchestrator"
				_, err := renderWithValidation(t, chart, values)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "agentOrchestrator.autoscaling."+errSuffix)
			})

			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					values := prefixed(family + "Agent")
					values["agentOrchestrator.enabled"] = "true"
					values["agentOrchestrator.image.repository"] = "example.com/agent-orchestrator"
					values[family+"Agent.enabled"] = "true"
					values[family+"Agent.image.repository"] = "example.com/" + family + "-agent"
					values["agentKeda.assumeInstalled"] = "true"
					values["vortex.enabled"] = "true"
					values["vortex.image.repository"] = "example.com/vortex"
					values["vortex.image.tag"] = "1"
					values["vortex.sonarqubeToken.token"] = "squ_example"
					values["vortex.storage.bucket"] = "vortex-artifacts"
					values["vortex.storage.region"] = "eu-west-1"
					_, err := renderWithValidation(t, chart, values)
					require.Error(t, err)
					assert.Contains(t, err.Error(), family+"Agent.autoscaling."+errSuffix)
				})
			}
		})
	}
}

func TestAgentAutoscalingMinReplicasFloor(t *testing.T) {
	assertAgentAutoscalingRangeRejected(t,
		map[string]string{"enabled": "true", "minReplicas": "1"},
		"minReplicas must be >= 2")
}

// maxReplicas must not be below minReplicas - a template-time check giving an actionable error
// instead of an opaque HPA/ScaledObject rejection at apply time.
func TestAgentAutoscalingMaxReplicasBelowMinReplicas(t *testing.T) {
	assertAgentAutoscalingRangeRejected(t,
		map[string]string{"enabled": "true", "minReplicas": "4", "maxReplicas": "3"},
		"maxReplicas must be >= minReplicas")
}

// Validation must be gated on the component's own enabled flag too, matching the render guards -
// otherwise hunterAgent.enabled=false with hunterAgent.autoscaling.enabled=true would hard-fail
// even though no HPA/ScaledObject would ever render.
func TestAgentAutoscalingValidationSkippedWhenComponentDisabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("orchestrator disabled", func(t *testing.T) {
				// agent-orchestrator.yaml renders nothing when disabled, and --show-only errors
				// on that rather than returning empty - target secret.yaml instead.
				base := agentValidationBase(chart)
				base["agentOrchestrator.enabled"] = "false"
				base["agentOrchestrator.autoscaling.enabled"] = "true"
				base["agentOrchestrator.autoscaling.minReplicas"] = "1"
				opts := &helm.Options{Logger: logger.Discard, SetValues: base}
				_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/secret.yaml"})
				require.NoError(t, err)
			})

			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family+" disabled", func(t *testing.T) {
					_, err := renderWithValidation(t, chart, map[string]string{
						"agentOrchestrator.enabled":              "true",
						"agentOrchestrator.image.repository":     "example.com/agent-orchestrator",
						family + "Agent.enabled":                 "false",
						family + "Agent.autoscaling.enabled":     "true",
						family + "Agent.autoscaling.minReplicas": "1",
					})
					require.NoError(t, err)
				})
			}
		})
	}
}

// terminationGracePeriodSeconds is unconditional hardening (protects any scale-down, not just an
// autoscaler-driven one) and must exceed each image's own fixed `uvicorn --timeout-graceful-
// shutdown` (44400s hunter, 3600s remediation) - never set SHUTDOWN_GRACE_SECONDS or
// LIVENESS_JOB_MAX_SECONDS from the chart, the image already sizes both against that ceiling.
func TestAgentRuntimeTerminationGraceUnconditional(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range []struct {
				family       string
				graceSeconds int64
			}{
				{"hunter", 44430},
				{"remediation", 3630},
			} {
				t.Run(tc.family, func(t *testing.T) {
					deployment, err := renderAgentRuntimeDeploymentFamily(t, chart, tc.family, nil)
					require.NoError(t, err)

					podSpec := deployment.Spec.Template.Spec
					require.NotNil(t, podSpec.TerminationGracePeriodSeconds)
					assert.Equal(t, tc.graceSeconds, *podSpec.TerminationGracePeriodSeconds)

					require.Len(t, podSpec.Containers, 1)
					for _, env := range podSpec.Containers[0].Env {
						assert.NotEqual(t, "SHUTDOWN_GRACE_SECONDS", env.Name,
							"chart must not override SHUTDOWN_GRACE_SECONDS - the image sizes it against its own fixed uvicorn timeout")
						assert.NotEqual(t, "LIVENESS_JOB_MAX_SECONDS", env.Name,
							"chart must not override LIVENESS_JOB_MAX_SECONDS - the image sizes it against its own fixed uvicorn timeout")
					}
				})
			}
		})
	}
}

func TestAgentRuntimeReplicasOmittedOnUpgradeWhenAutoscalingEnabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{
				"hunterAgent.autoscaling.enabled": "true",
				"agentKeda.assumeInstalled":       "true",
			}
			assertReplicasOmittedOnUpgrade(t, func(extraArgs ...string) (appsv1.Deployment, error) {
				return renderAgentRuntimeDeploymentFamily(t, chart, "hunter", setValues, extraArgs...)
			})

			// The other family, with autoscaling untouched, is unaffected either way.
			remediationInstall, err := renderAgentRuntimeDeploymentFamily(t, chart, "remediation", setValues)
			require.NoError(t, err)
			require.NotNil(t, remediationInstall.Spec.Replicas)
			remediationUpgrade, err := renderAgentRuntimeDeploymentFamily(t, chart, "remediation", setValues, "--is-upgrade")
			require.NoError(t, err)
			require.NotNil(t, remediationUpgrade.Spec.Replicas)
		})
	}
}

// manageReplicas=false is the GitOps escape hatch: Release.IsInstall is always true under `helm
// template` (Argo CD, Flux, --dry-run=client), so without it those consumers would always hit the
// "fresh install" branch above and keep resetting replicas on every sync.
func TestAgentRuntimeReplicasSuppressedByManageReplicasFalse(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{
				"hunterAgent.autoscaling.enabled":        "true",
				"hunterAgent.autoscaling.manageReplicas": "false",
				"agentKeda.assumeInstalled":              "true",
			}
			assertReplicasOmittedWhenManageReplicasFalse(t, func() (appsv1.Deployment, error) {
				return renderAgentRuntimeDeploymentFamily(t, chart, "hunter", setValues)
			})

			// The other family, with autoscaling untouched, is unaffected.
			remediationInstall, err := renderAgentRuntimeDeploymentFamily(t, chart, "remediation", setValues)
			require.NoError(t, err)
			require.NotNil(t, remediationInstall.Spec.Replicas)
		})
	}
}

// manageReplicas must only gate the autoscaling-enabled branch, not suppress replicas whenever a
// shared values layer sets it false regardless of autoscaling.enabled.
func TestAgentRuntimeReplicasRenderedWhenAutoscalingDisabledEvenIfManageReplicasFalse(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					assertReplicasRenderedWhenAutoscalingDisabled(t, func() (appsv1.Deployment, error) {
						return renderAgentRuntimeDeploymentFamily(t, chart, family, map[string]string{
							family + "Agent.autoscaling.enabled":        "false",
							family + "Agent.autoscaling.manageReplicas": "false",
						})
					})
				})
			}
		})
	}
}
