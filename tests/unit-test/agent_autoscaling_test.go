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
// KEDA CRD, and the full KEDA client types are not worth adding as a dependency for this.
type scaledObject struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		ScaleTargetRef  scaledObjectScaleTargetRef `json:"scaleTargetRef"`
		MinReplicaCount int64                      `json:"minReplicaCount"`
		MaxReplicaCount int64                      `json:"maxReplicaCount"`
		PollingInterval int64                      `json:"pollingInterval"`
		Triggers        []struct {
			Type     string            `json:"type"`
			Metadata map[string]string `json:"metadata"`
		} `json:"triggers"`
	} `json:"spec"`
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

			install, err := renderAgentOrchestratorDeployment(t, chart, setValues)
			require.NoError(t, err)
			require.NotNil(t, install.Spec.Replicas, "replicas should render on a fresh install")

			upgrade, err := renderAgentOrchestratorDeployment(t, chart, setValues, "--is-upgrade")
			require.NoError(t, err)
			assert.Nil(t, upgrade.Spec.Replicas, "replicas should be omitted on upgrade once the HPA owns it")
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

			install, err := renderAgentOrchestratorDeployment(t, chart, setValues)
			require.NoError(t, err)
			assert.Nil(t, install.Spec.Replicas, "replicas should be omitted even on install when manageReplicas=false")
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
			opts := &helm.Options{Logger: logger.Discard, SetValues: base}

			t.Run("no KEDA CRD, no override: fails", func(t *testing.T) {
				_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator.yaml"})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "hunterAgent.autoscaling.enabled is true but the KEDA CRDs")
			})

			t.Run("KEDA CRD present via --api-versions: succeeds", func(t *testing.T) {
				_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator.yaml"}, "--api-versions=keda.sh/v1alpha1")
				require.NoError(t, err)
			})
		})
	}
}

func TestAgentAutoscalingMinReplicasFloor(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("orchestrator", func(t *testing.T) {
				_, err := renderWithValidation(t, chart, map[string]string{
					"agentOrchestrator.enabled":                 "true",
					"agentOrchestrator.image.repository":        "example.com/agent-orchestrator",
					"agentOrchestrator.autoscaling.enabled":     "true",
					"agentOrchestrator.autoscaling.minReplicas": "1",
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "agentOrchestrator.autoscaling.minReplicas must be >= 2")
			})

			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					_, err := renderWithValidation(t, chart, map[string]string{
						"agentOrchestrator.enabled":              "true",
						"agentOrchestrator.image.repository":     "example.com/agent-orchestrator",
						family + "Agent.enabled":                 "true",
						family + "Agent.image.repository":        "example.com/" + family + "-agent",
						family + "Agent.autoscaling.enabled":     "true",
						family + "Agent.autoscaling.minReplicas": "1",
						"agentKeda.assumeInstalled":              "true",
						"vortex.enabled":                         "true",
						"vortex.image.repository":                "example.com/vortex",
						"vortex.image.tag":                       "1",
						"vortex.sonarqubeToken.token":            "squ_example",
						"vortex.storage.bucket":                  "vortex-artifacts",
						"vortex.storage.region":                  "eu-west-1",
					})
					require.Error(t, err)
					assert.Contains(t, err.Error(), family+"Agent.autoscaling.minReplicas must be >= 2")
				})
			}
		})
	}
}

// maxReplicas must not be below minReplicas - a template-time check giving an actionable error
// instead of an opaque HPA/ScaledObject rejection at apply time.
func TestAgentAutoscalingMaxReplicasBelowMinReplicas(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("orchestrator", func(t *testing.T) {
				_, err := renderWithValidation(t, chart, map[string]string{
					"agentOrchestrator.enabled":                 "true",
					"agentOrchestrator.image.repository":        "example.com/agent-orchestrator",
					"agentOrchestrator.autoscaling.enabled":     "true",
					"agentOrchestrator.autoscaling.minReplicas": "4",
					"agentOrchestrator.autoscaling.maxReplicas": "3",
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "agentOrchestrator.autoscaling.maxReplicas must be >= minReplicas")
			})

			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					_, err := renderWithValidation(t, chart, map[string]string{
						"agentOrchestrator.enabled":              "true",
						"agentOrchestrator.image.repository":     "example.com/agent-orchestrator",
						family + "Agent.enabled":                 "true",
						family + "Agent.image.repository":        "example.com/" + family + "-agent",
						family + "Agent.autoscaling.enabled":     "true",
						family + "Agent.autoscaling.minReplicas": "4",
						family + "Agent.autoscaling.maxReplicas": "3",
						"agentKeda.assumeInstalled":              "true",
						"vortex.enabled":                         "true",
						"vortex.image.repository":                "example.com/vortex",
						"vortex.image.tag":                       "1",
						"vortex.sonarqubeToken.token":            "squ_example",
						"vortex.storage.bucket":                  "vortex-artifacts",
						"vortex.storage.region":                  "eu-west-1",
					})
					require.Error(t, err)
					assert.Contains(t, err.Error(), family+"Agent.autoscaling.maxReplicas must be >= minReplicas")
				})
			}
		})
	}
}

// The autoscaling validation must be gated on the component's own enabled flag, matching the
// render guards in agent-orchestrator-hpa.yaml/agent-runtime-scaledobject.yaml - otherwise a
// values layer that sets autoscaling defaults once and toggles components per environment (e.g.
// hunterAgent.enabled=false while hunterAgent.autoscaling.enabled stays true) would hard-fail even
// though no HPA/ScaledObject would ever render.
func TestAgentAutoscalingValidationSkippedWhenComponentDisabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("orchestrator disabled", func(t *testing.T) {
				// agent-orchestrator.yaml itself renders zero documents when disabled, and
				// --show-only errors rather than returning empty for that (see the comment on
				// TestAgentOrchestratorHPANotRenderedByDefault) - target secret.yaml instead,
				// which always renders given agentValidationBase's monitoringPasscode.
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

// terminationGracePeriodSeconds/SHUTDOWN_GRACE_SECONDS are unconditional hardening - they must be
// present even with autoscaling off (protects any scale-down: manual, rollout, or node drain).
// Defaults must clear each runtime image's own fixed, non-overridable `uvicorn
// --timeout-graceful-shutdown` (14400s hunter-agent-runtime, 3600s remediation-agent-runtime) - see
// those Dockerfiles. The chart does not set SHUTDOWN_GRACE_SECONDS itself; the image already sizes
// that env var correctly against its own ceiling, so asserting its absence here guards against
// reintroducing an override that would clobber it with a too-small value.
func TestAgentRuntimeTerminationGraceUnconditional(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range []struct {
				family       string
				graceSeconds int64
			}{
				{"hunter", 14430},
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

			install, err := renderAgentRuntimeDeploymentFamily(t, chart, "hunter", setValues)
			require.NoError(t, err)
			require.NotNil(t, install.Spec.Replicas, "replicas should render on a fresh install")

			upgrade, err := renderAgentRuntimeDeploymentFamily(t, chart, "hunter", setValues, "--is-upgrade")
			require.NoError(t, err)
			assert.Nil(t, upgrade.Spec.Replicas, "replicas should be omitted on upgrade once the ScaledObject owns it")

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

			install, err := renderAgentRuntimeDeploymentFamily(t, chart, "hunter", setValues)
			require.NoError(t, err)
			assert.Nil(t, install.Spec.Replicas, "replicas should be omitted even on install when manageReplicas=false")
		})
	}
}
