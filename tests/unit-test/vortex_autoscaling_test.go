package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

// Vortex has no shared queue like the agent runtimes' orchestrator - each replica only knows its
// own peak concurrency, so the ScaledObject aggregates across replicas itself (SONAR-32198). These
// tests reuse the scaledObject struct and vortexDeployment/renderVortex helpers from
// agent_autoscaling_test.go and vortex_test.go.

func renderVortexScaledObject(t *testing.T, chart agentChart, setValues map[string]string) (scaledObject, error) {
	t.Helper()
	base := map[string]string{
		"vortex.autoscaling.enabled": "true",
		"vortex.strategy.type":       "RollingUpdate",
		"agentKeda.assumeInstalled":  "true",
	}
	for k, v := range setValues {
		base[k] = v
	}
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/vortex-enabled.yaml"},
		SetValues:   base,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex-scaledobject.yaml"})
	if err != nil {
		return scaledObject{}, err
	}
	var so scaledObject
	helm.UnmarshalK8SYaml(t, output, &so)
	return so, nil
}

func renderVortexDeploymentWithValues(t *testing.T, chart agentChart, setValues map[string]string, extraArgs ...string) (appsv1.Deployment, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"}, extraArgs...)
	if err != nil {
		return appsv1.Deployment{}, err
	}
	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	return deployment, nil
}

// Helm's `--show-only` errors on a template that renders zero documents rather than returning
// empty, so "not rendered" must be asserted against a full chart render - same pattern as the
// agent runtime ScaledObject tests.
func TestVortexScaledObjectNotRenderedByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-enabled.yaml"},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{})
			require.NoError(t, err)
			assert.NotContains(t, output, "vortex-scaledobject.yaml")
		})
	}
}

func TestVortexScaledObjectNotRenderedWhenVortexDisabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-disabled.yaml"},
				SetValues: map[string]string{
					"vortex.autoscaling.enabled": "true",
					"agentKeda.assumeInstalled":  "true",
				},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{})
			require.NoError(t, err)
			assert.NotContains(t, output, "vortex-scaledobject.yaml")
		})
	}
}

func TestVortexScaledObjectNotRenderedWhenAutoscalingDisabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
				SetValues:   map[string]string{"vortex.autoscaling.enabled": "false"},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{})
			require.NoError(t, err)
			assert.NotContains(t, output, "vortex-scaledobject.yaml")
		})
	}
}

func TestVortexScaledObjectRendersWhenEnabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			so, err := renderVortexScaledObject(t, chart, map[string]string{
				"vortex.autoscaling.minReplicas":                   "3",
				"vortex.autoscaling.maxReplicas":                   "7",
				"vortex.autoscaling.pollingInterval":               "20",
				"vortex.autoscaling.scaleDownStabilizationSeconds": "1200",
				"vortex.autoscaling.targetConcurrentRequests":      "12",
			})
			require.NoError(t, err)

			expectedName := chart.fullnamePrefix() + vortexFullnameSuffix
			assert.Equal(t, expectedName, so.Metadata.Name)
			assert.Equal(t, expectedName, so.Spec.ScaleTargetRef.Name)
			assert.EqualValues(t, 3, so.Spec.MinReplicaCount)
			assert.EqualValues(t, 7, so.Spec.MaxReplicaCount)
			assert.EqualValues(t, 20, so.Spec.PollingInterval)
			assert.EqualValues(t, 1200, so.Spec.Advanced.HorizontalPodAutoscalerConfig.Behavior.ScaleDown.StabilizationWindowSeconds)

			require.Len(t, so.Spec.Triggers, 1)
			trigger := so.Spec.Triggers[0]
			assert.Equal(t, "metrics-api", trigger.Type)
			assert.Equal(t, "value", trigger.Metadata["valueLocation"])
			assert.Equal(t, "12", trigger.Metadata["targetValue"])
			// Unauthenticated by design, matching the agent runtimes' own orchestrator trigger.
			assert.Nil(t, trigger.AuthenticationRef)
		})
	}
}

// The URL must be a Kubernetes-DNS FQDN, not the short Service name sonarqube.vortex.url builds
// elsewhere: KEDA parses the host as <service>.<namespace> to list the Service's EndpointSlices,
// and rejects a bare short name outright.
func TestVortexScaledObjectTriggerURLIsFQDN(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			so, err := renderVortexScaledObject(t, chart, nil)
			require.NoError(t, err)

			require.Len(t, so.Spec.Triggers, 1)
			url := so.Spec.Triggers[0].Metadata["url"]
			expectedHost := chart.fullnamePrefix() + vortexFullnameSuffix + ".default.svc.cluster.local"
			assert.Equal(t, "http://"+expectedHost+":8080/metrics/max-concurrent-requests", url)
		})
	}
}

// The port in the URL is matched against the pod's target port, not just any port on the Service -
// a mismatch makes KEDA silently fall back to inferring the port from the scheme (80 for http)
// instead of erroring. vortex.port drives both the Service port and the containerPort, so a custom
// value must still produce a URL KEDA can resolve to the right pod port.
func TestVortexScaledObjectTriggerURLTracksCustomPort(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			so, err := renderVortexScaledObject(t, chart, map[string]string{"vortex.port": "9090"})
			require.NoError(t, err)

			require.Len(t, so.Spec.Triggers, 1)
			url := so.Spec.Triggers[0].Metadata["url"]
			expectedHost := chart.fullnamePrefix() + vortexFullnameSuffix + ".default.svc.cluster.local"
			assert.Equal(t, "http://"+expectedHost+":9090/metrics/max-concurrent-requests", url)
		})
	}
}

// A custom metricPath must be reflected verbatim - the endpoint's final path isn't settled yet
// upstream (SONAR-32198 plan Q6), so this has to stay a values knob.
func TestVortexScaledObjectTriggerURLTracksCustomMetricPath(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			so, err := renderVortexScaledObject(t, chart, map[string]string{"vortex.autoscaling.metricPath": "/metrics-api"})
			require.NoError(t, err)

			require.Len(t, so.Spec.Triggers, 1)
			assert.True(t, strings.HasSuffix(so.Spec.Triggers[0].Metadata["url"], "/metrics-api"))
		})
	}
}

// aggregationType must be "sum", not the metrics-api scaler's own "average" default: KEDA's
// metricType defaults to AverageValue, so the HPA computes ceil(metric/targetValue) - an averaged
// metric would cap the whole fleet at ceil(mean/target) regardless of replica count.
func TestVortexScaledObjectAggregatesAcrossReplicasByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			so, err := renderVortexScaledObject(t, chart, nil)
			require.NoError(t, err)

			require.Len(t, so.Spec.Triggers, 1)
			trigger := so.Spec.Triggers[0]
			assert.Equal(t, "true", trigger.Metadata["aggregateFromKubeServiceEndpoints"])
			assert.Equal(t, "sum", trigger.Metadata["aggregationType"])
		})
	}
}

// aggregateAcrossReplicas=false is the escape hatch for a pre-existing cluster-wide KEDA older
// than 2.20.0 (a version this chart has no way to detect) - both keys must then be omitted rather
// than sent with a value the scaler might reject or ignore.
func TestVortexScaledObjectAggregationOptOut(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			so, err := renderVortexScaledObject(t, chart, map[string]string{"vortex.autoscaling.aggregateAcrossReplicas": "false"})
			require.NoError(t, err)

			require.Len(t, so.Spec.Triggers, 1)
			trigger := so.Spec.Triggers[0]
			_, hasAggregate := trigger.Metadata["aggregateFromKubeServiceEndpoints"]
			_, hasType := trigger.Metadata["aggregationType"]
			assert.False(t, hasAggregate, "aggregateFromKubeServiceEndpoints must be omitted, not sent as \"false\"")
			assert.False(t, hasType)
		})
	}
}

// Once the ScaledObject owns replicas, a fresh install still gets a sane initial count
// (Release.IsInstall is true by default under `helm template`), but a later `helm upgrade` must
// not fight it - same pattern as the agent runtimes and orchestrator.
func TestVortexReplicasOmittedOnUpgradeWhenAutoscalingEnabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			assertReplicasOmittedOnUpgrade(t, func(extraArgs ...string) (appsv1.Deployment, error) {
				return renderVortexDeploymentWithValues(t, chart, nil, extraArgs...)
			})
		})
	}
}

// manageReplicas=false is the GitOps escape hatch: Release.IsInstall is always true under `helm
// template` (Argo CD, Flux, --dry-run=client), so without it those consumers would always hit the
// "fresh install" branch above and keep resetting replicas on every sync.
func TestVortexReplicasSuppressedByManageReplicasFalse(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			assertReplicasOmittedWhenManageReplicasFalse(t, func() (appsv1.Deployment, error) {
				return renderVortexDeploymentWithValues(t, chart, map[string]string{
					"vortex.autoscaling.manageReplicas": "false",
				})
			})
		})
	}
}

// manageReplicas must only gate the autoscaling-enabled branch, not suppress replicas whenever a
// shared values layer sets it false regardless of autoscaling.enabled.
func TestVortexReplicasRenderedWhenAutoscalingDisabledEvenIfManageReplicasFalse(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			assertReplicasRenderedWhenAutoscalingDisabled(t, func() (appsv1.Deployment, error) {
				return renderVortexDeploymentWithValues(t, chart, map[string]string{
					"vortex.autoscaling.enabled":        "false",
					"vortex.autoscaling.manageReplicas": "false",
				})
			})
		})
	}
}

// The sliding window Vortex reports its peak concurrency over is only set when overridden, so an
// unset value leaves the image's own default (30s) in place, and a user-supplied vortex.env entry
// of the same name still wins - matching every other auto-generated env var on this Deployment.
func TestVortexMetricsWindowEnvVar(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("present when autoscaling enabled", func(t *testing.T) {
				deployment := vortexDeployment(t, chart, "vortex-autoscaling.yaml")
				env := vortexContainerEnv(deployment.Spec.Template.Spec.Containers[0])
				window, ok := env["METRICS_CONCURRENT_REQUESTS_WINDOW_SECONDS"]
				require.True(t, ok, "windowSeconds defaults to 30, so the env var must be set once autoscaling is on")
				assert.Equal(t, "30", window.Value)
			})

			// The var is gated on autoscaling.enabled, not just windowSeconds: the latter
			// defaults to a non-empty 30, so gating on it alone would pin every Vortex pod to
			// this env var even on an install that never opts into autoscaling.
			t.Run("absent when autoscaling disabled", func(t *testing.T) {
				deployment := vortexDeployment(t, chart, "vortex-enabled.yaml")
				_, ok := vortexContainerEnv(deployment.Spec.Template.Spec.Containers[0])["METRICS_CONCURRENT_REQUESTS_WINDOW_SECONDS"]
				assert.False(t, ok, "vortex-enabled.yaml has autoscaling disabled by default")
			})

			t.Run("absent when blank", func(t *testing.T) {
				opts := &helm.Options{
					Logger:      logger.Discard,
					ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
					SetValues:   map[string]string{"vortex.autoscaling.windowSeconds": ""},
				}
				output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
				require.NoError(t, err)
				var d appsv1.Deployment
				helm.UnmarshalK8SYaml(t, output, &d)
				_, ok := vortexContainerEnv(d.Spec.Template.Spec.Containers[0])["METRICS_CONCURRENT_REQUESTS_WINDOW_SECONDS"]
				assert.False(t, ok)
			})

			t.Run("overridable via vortex.env", func(t *testing.T) {
				opts := &helm.Options{
					Logger:      logger.Discard,
					ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
					SetValues: map[string]string{
						"vortex.env[0].name": "METRICS_CONCURRENT_REQUESTS_WINDOW_SECONDS",
					},
					// SetStrValues, not SetValues: an unquoted 45 renders as a YAML number, which
					// EnvVar.Value (a string) fails to unmarshal.
					SetStrValues: map[string]string{
						"vortex.env[0].value": "45",
					},
				}
				output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
				require.NoError(t, err)
				var d appsv1.Deployment
				helm.UnmarshalK8SYaml(t, output, &d)
				container := d.Spec.Template.Spec.Containers[0]
				window, ok := vortexContainerEnv(container)["METRICS_CONCURRENT_REQUESTS_WINDOW_SECONDS"]
				require.True(t, ok)
				assert.Equal(t, "45", window.Value)
				assert.Equal(t, "METRICS_CONCURRENT_REQUESTS_WINDOW_SECONDS", container.Env[len(container.Env)-1].Name,
					"user-supplied env must come last so it overrides the auto-wired var")
			})
		})
	}
}

// terminationGracePeriodSeconds is unconditional hardening (protects any scale-down, not just an
// autoscaler-driven one), mirroring the agent runtimes' own rule.
func TestVortexTerminationGraceUnconditional(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := vortexDeployment(t, chart, "vortex-enabled.yaml")
			podSpec := deployment.Spec.Template.Spec
			require.NotNil(t, podSpec.TerminationGracePeriodSeconds)
			assert.EqualValues(t, 180, *podSpec.TerminationGracePeriodSeconds)

			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-enabled.yaml"},
				SetValues:   map[string]string{"vortex.terminationGracePeriodSeconds": "120"},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
			require.NoError(t, err)
			var overridden appsv1.Deployment
			helm.UnmarshalK8SYaml(t, output, &overridden)
			require.NotNil(t, overridden.Spec.Template.Spec.TerminationGracePeriodSeconds)
			assert.EqualValues(t, 120, *overridden.Spec.Template.Spec.TerminationGracePeriodSeconds)
		})
	}
}

// assertVortexAutoscalingRejected renders templates/vortex.yaml (which always renders when
// vortex.enabled=true, regardless of whether these specific checks pass) with the given SetValues
// layered onto vortex-autoscaling.yaml, and asserts the render fails containing errSubstring.
func assertVortexAutoscalingRejected(t *testing.T, setValues map[string]string, errSubstring string) {
	t.Helper()
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
				SetValues:   setValues,
			}
			_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), errSubstring)
		})
	}
}

func TestVortexAutoscalingMinReplicasFloor(t *testing.T) {
	assertVortexAutoscalingRejected(t,
		map[string]string{"vortex.autoscaling.minReplicas": "1"},
		"vortex.autoscaling.minReplicas must be >= 2")
}

func TestVortexAutoscalingMaxReplicasBelowMinReplicas(t *testing.T) {
	assertVortexAutoscalingRejected(t,
		map[string]string{"vortex.autoscaling.minReplicas": "4", "vortex.autoscaling.maxReplicas": "3"},
		"vortex.autoscaling.maxReplicas must be >= minReplicas")
}

// Without cross-replica aggregation, KEDA samples a single random pod through the Service, so
// the sum-vs-target arithmetic is only valid for exactly one replica - the normal >= 2 floor would
// otherwise make that documented fallback impossible to configure at all.
func TestVortexAutoscalingMinReplicasFloorRelaxedWhenAggregationDisabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
				SetValues: map[string]string{
					"vortex.autoscaling.aggregateAcrossReplicas": "false",
					"vortex.autoscaling.minReplicas":             "1",
				},
			}
			_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
			require.NoError(t, err)
		})
	}

	// The default (aggregateAcrossReplicas: true) still enforces the >= 2 floor.
	assertVortexAutoscalingRejected(t,
		map[string]string{"vortex.autoscaling.minReplicas": "1"},
		"vortex.autoscaling.minReplicas must be >= 2")
}

// The KEDA CRD guard: enabling autoscaling without the KEDA CRDs present (and no explicit
// agentKeda.assumeInstalled override) fails; --api-versions simulates the CRD being registered on
// a real cluster (Capabilities.APIVersions is otherwise empty under `helm template`). The fixture
// itself sets assumeInstalled: true, so it's reset to null here (not false, which the chart reads
// as a real "assume absent" override that --api-versions can never outweigh) to exercise the
// actual auto-detection path.
func TestVortexAutoscalingRequiresKedaCRDOrOverride(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
				SetValues:   map[string]string{"agentKeda.assumeInstalled": "null"},
			}
			assertAutoscalingRequiresKedaCRDOrOverride(t, opts, chart, "templates/vortex.yaml",
				"vortex.autoscaling.enabled is true but the KEDA CRDs")
		})
	}
}

func TestVortexAutoscalingPollingIntervalMustNotExceedWindow(t *testing.T) {
	assertVortexAutoscalingRejected(t,
		map[string]string{"vortex.autoscaling.pollingInterval": "60", "vortex.autoscaling.windowSeconds": "30"},
		"vortex.autoscaling.pollingInterval must be <= vortex.autoscaling.windowSeconds")
}

// windowSeconds blank means "leave the image's own default", which this chart can't compare
// against - so the check must not fire when it's unset, however large pollingInterval is.
func TestVortexAutoscalingPollingIntervalCheckSkippedWhenWindowBlank(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
				SetValues: map[string]string{
					"vortex.autoscaling.pollingInterval": "9999",
					"vortex.autoscaling.windowSeconds":   "",
				},
			}
			_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
			require.NoError(t, err)
		})
	}
}

func TestVortexAutoscalingMetricPathMustBeAbsolute(t *testing.T) {
	assertVortexAutoscalingRejected(t,
		map[string]string{"vortex.autoscaling.metricPath": "relative-path"},
		`vortex.autoscaling.metricPath must start with "/"`)
}

// Recreate with N replicas takes the whole Vortex fleet down on every rollout, which is actively
// at odds with autoscaling - fail rather than silently switching the operator's strategy for them.
func TestVortexAutoscalingRejectsRecreateStrategy(t *testing.T) {
	assertVortexAutoscalingRejected(t,
		map[string]string{"vortex.strategy.type": "Recreate"},
		`vortex.strategy.type is "Recreate"`)
}

// Recreate stays the shipped default, so it must keep rendering fine whenever autoscaling itself
// is off - the check above must only fire when both conditions hold together.
func TestVortexRecreateStrategyAllowedWhenAutoscalingDisabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := vortexDeployment(t, chart, "vortex-enabled.yaml")
			assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deployment.Spec.Strategy.Type)
		})
	}
}

// vortex.yaml itself tolerates an absent strategy ({{- with $vortex.strategy }}, falling back to
// Kubernetes' own RollingUpdate default), so vortex.strategy: null is a supported input. The
// Recreate check must read strategy.type nil-safely rather than panicking on that nil.
func TestVortexAutoscalingAllowsNilStrategy(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
				SetValues:   map[string]string{"vortex.strategy": "null"},
			}
			_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
			require.NoError(t, err)
		})
	}
}

// Both charts share byte-identical vortex.yaml/vortex-scaledobject.yaml templates, so identical
// values must render an identical ScaledObject - a parity guard against the two charts drifting.
func TestVortexScaledObjectIdenticalAcrossCharts(t *testing.T) {
	var rendered []scaledObject
	for _, chart := range agentCharts {
		opts := &helm.Options{
			Logger:      logger.Discard,
			ValuesFiles: []string{chart.valuesDir + "/vortex-autoscaling.yaml"},
			SetValues:   map[string]string{"vortex.autoscaling.minReplicas": "2"},
		}
		output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex-scaledobject.yaml"})
		require.NoError(t, err)
		var so scaledObject
		helm.UnmarshalK8SYaml(t, output, &so)
		rendered = append(rendered, so)
	}

	require.Len(t, rendered, 2)
	assert.Equal(t, rendered[0].Spec.MinReplicaCount, rendered[1].Spec.MinReplicaCount)
	assert.Equal(t, rendered[0].Spec.MaxReplicaCount, rendered[1].Spec.MaxReplicaCount)
	assert.Equal(t, rendered[0].Spec.PollingInterval, rendered[1].Spec.PollingInterval)
	require.Len(t, rendered[0].Spec.Triggers, 1)
	require.Len(t, rendered[1].Spec.Triggers, 1)
	assert.Equal(t, rendered[0].Spec.Triggers[0].Metadata["valueLocation"], rendered[1].Spec.Triggers[0].Metadata["valueLocation"])
	assert.Equal(t, rendered[0].Spec.Triggers[0].Metadata["targetValue"], rendered[1].Spec.Triggers[0].Metadata["targetValue"])
}
