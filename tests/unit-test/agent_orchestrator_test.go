package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func renderAgentOrchestrator(t *testing.T, chart agentChart, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/agent-orchestrator-enabled.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator.yaml"})
	require.NoError(t, err)

	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	require.NotEmpty(t, deployment.Name, "no Deployment rendered by templates/agent-orchestrator.yaml")
	return deployment
}

func TestAgentOrchestratorContainerSecurityContext(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := renderAgentOrchestrator(t, chart, nil)
			podSpec := deployment.Spec.Template.Spec
			require.Len(t, podSpec.Containers, 1)
			container := podSpec.Containers[0]

			sc := container.SecurityContext
			require.NotNil(t, sc)
			require.NotNil(t, sc.AllowPrivilegeEscalation)
			assert.False(t, *sc.AllowPrivilegeEscalation)
			require.NotNil(t, sc.RunAsNonRoot)
			assert.True(t, *sc.RunAsNonRoot)
			require.NotNil(t, sc.RunAsUser)
			assert.Equal(t, int64(900), *sc.RunAsUser)
			require.NotNil(t, sc.ReadOnlyRootFilesystem)
			assert.True(t, *sc.ReadOnlyRootFilesystem)
			require.NotNil(t, sc.Capabilities)
			assert.Contains(t, sc.Capabilities.Drop, corev1.Capability("ALL"))

			require.Len(t, container.VolumeMounts, 1)
			assert.Equal(t, "tmp", container.VolumeMounts[0].Name)
			assert.Equal(t, "/tmp", container.VolumeMounts[0].MountPath)

			require.Len(t, podSpec.Volumes, 1)
			assert.Equal(t, "tmp", podSpec.Volumes[0].Name)
			require.NotNil(t, podSpec.Volumes[0].EmptyDir)
		})
	}
}

// Clearing readOnlyRootFilesystem drops the securityContext key and the chart-managed /tmp
// emptyDir, since neither is needed without it.
func TestAgentOrchestratorNoReadOnlyRootFilesystemNoTmpVolume(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("readOnlyRootFilesystem: false", func(t *testing.T) {
				deployment := renderAgentOrchestrator(t, chart, map[string]string{
					"agentOrchestrator.containerSecurityContext.readOnlyRootFilesystem": "false",
				})
				podSpec := deployment.Spec.Template.Spec
				require.Len(t, podSpec.Containers, 1)
				assert.Empty(t, podSpec.Containers[0].VolumeMounts)
				assert.Empty(t, podSpec.Volumes)
			})

			// containerSecurityContext: null is the chart's documented convention for disabling a
			// default map entirely (see gvisor.nodeSelector) - it must not nil-pointer.
			t.Run("containerSecurityContext: null", func(t *testing.T) {
				deployment := renderAgentOrchestrator(t, chart, map[string]string{
					"agentOrchestrator.containerSecurityContext": "null",
				})
				podSpec := deployment.Spec.Template.Spec
				require.Len(t, podSpec.Containers, 1)
				assert.Nil(t, podSpec.Containers[0].SecurityContext)
				assert.Empty(t, podSpec.Containers[0].VolumeMounts)
				assert.Empty(t, podSpec.Volumes)
			})
		})
	}
}

func TestAgentOrchestratorResources(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := renderAgentOrchestrator(t, chart, nil)
			container := deployment.Spec.Template.Spec.Containers[0]
			assert.Equal(t, "250m", container.Resources.Requests.Cpu().String())
			assert.Equal(t, "512Mi", container.Resources.Requests.Memory().String())
			assert.Equal(t, "512Mi", container.Resources.Requests.StorageEphemeral().String())
			assert.Equal(t, "1", container.Resources.Limits.Cpu().String())
			assert.Equal(t, "1Gi", container.Resources.Limits.Memory().String())
			assert.Equal(t, "2Gi", container.Resources.Limits.StorageEphemeral().String())
		})
	}
}

// Off by default, so an existing install picks up nothing new until agentOrchestrator.enabled is set.
func TestAgentOrchestratorDisabledByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tpl := range []string{
				"templates/agent-orchestrator.yaml",
				"templates/agent-orchestrator-service.yaml",
			} {
				opts := &helm.Options{
					Logger:      logger.Discard,
					ValuesFiles: []string{chart.valuesDir + "/agent-all-disabled.yaml"},
				}
				output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{tpl})
				require.Error(t, err, "%s must render nothing when agentOrchestrator.enabled is false", tpl)
				assert.Empty(t, strings.TrimSpace(output))
			}
		})
	}
}

// The Service must select the Deployment's pods — a label typo here silently yields no endpoints.
func TestAgentOrchestratorService(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/agent-orchestrator-enabled.yaml"},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator-service.yaml"})
			require.NoError(t, err)

			var service corev1.Service
			helm.UnmarshalK8SYaml(t, output, &service)
			require.NotEmpty(t, service.Name)

			podLabels := renderAgentOrchestrator(t, chart, nil).Spec.Template.Labels
			require.NotEmpty(t, service.Spec.Selector)
			for k, v := range service.Spec.Selector {
				assert.Equal(t, v, podLabels[k], "Service selector %q must match the pod labels", k)
			}
		})
	}
}

// The Agent Orchestrator can be scheduled apart from the SonarQube pods, and its own scheduling
// settings take precedence over the chart's global ones.
func TestAgentOrchestratorSchedulingWinsOverGlobal(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/agent-orchestrator-global-scheduling.yaml"},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator.yaml"})
			require.NoError(t, err)
			var deployment appsv1.Deployment
			helm.UnmarshalK8SYaml(t, output, &deployment)
			podSpec := deployment.Spec.Template.Spec

			assert.Equal(t, map[string]string{"orchestrator": "true"}, podSpec.NodeSelector)

			require.Len(t, podSpec.Tolerations, 1)
			assert.Equal(t, "orchestrator", podSpec.Tolerations[0].Key)

			require.NotNil(t, podSpec.Affinity)
			require.NotNil(t, podSpec.Affinity.NodeAffinity)
			terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			require.Len(t, terms, 1)
			require.Len(t, terms[0].MatchExpressions, 1)
			assert.Equal(t, "orchestrator", terms[0].MatchExpressions[0].Key)
		})
	}
}

// Both probes default on, against the health endpoint the orchestrator's own source exposes.
func TestAgentOrchestratorProbes(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := renderAgentOrchestrator(t, chart, nil).Spec.Template.Spec.Containers[0]

			for name, probe := range map[string]*corev1.Probe{
				"readiness": container.ReadinessProbe,
				"liveness":  container.LivenessProbe,
			} {
				require.NotNil(t, probe, "%s probe must be set", name)
				require.NotNil(t, probe.HTTPGet, "%s probe must be an HTTP GET", name)
				assert.Equal(t, "/health", probe.HTTPGet.Path)
				assert.Equal(t, "http", probe.HTTPGet.Port.StrVal)
			}
		})
	}
}

// When the chart-wide encryption key secret (.Values.sonarSecretKey) is set, it must be mounted
// into the orchestrator too, mirroring how it's mounted into the SonarQube application/search
// pods, with its path exposed via AGENTIC_SECRET_KEY_PATH (SONAR-31746).
func TestAgentOrchestratorEncryptionKeyMount(t *testing.T) {
	findEnv := func(container corev1.Container, name string) *corev1.EnvVar {
		for i := range container.Env {
			if container.Env[i].Name == name {
				return &container.Env[i]
			}
		}
		return nil
	}
	findVolume := func(podSpec corev1.PodSpec, name string) *corev1.Volume {
		for i := range podSpec.Volumes {
			if podSpec.Volumes[i].Name == name {
				return &podSpec.Volumes[i]
			}
		}
		return nil
	}
	findVolumeMount := func(container corev1.Container, name string) *corev1.VolumeMount {
		for i := range container.VolumeMounts {
			if container.VolumeMounts[i].Name == name {
				return &container.VolumeMounts[i]
			}
		}
		return nil
	}

	t.Run("unset renders no encryption key env var, volume, or volumeMount", func(t *testing.T) {
		deployment := renderAgentOrchestrator(t, nil)
		container := deployment.Spec.Template.Spec.Containers[0]
		assert.Nil(t, findEnv(container, "AGENTIC_SECRET_KEY_PATH"))
		assert.Nil(t, findVolumeMount(container, "secret"))
		assert.Nil(t, findVolume(deployment.Spec.Template.Spec, "secret"))
	})

	t.Run("set mounts the secret and exposes its path", func(t *testing.T) {
		deployment := renderAgentOrchestrator(t, map[string]string{
			"sonarSecretKey": "settings-encryption-secret",
		})
		podSpec := deployment.Spec.Template.Spec
		container := podSpec.Containers[0]

		env := findEnv(container, "AGENTIC_SECRET_KEY_PATH")
		require.NotNil(t, env)
		assert.Equal(t, "/sonarcloud/secret/sonar-secret.txt", env.Value)

		volumeMount := findVolumeMount(container, "secret")
		require.NotNil(t, volumeMount)
		assert.Equal(t, "/sonarcloud/secret/", volumeMount.MountPath)

		volume := findVolume(podSpec, "secret")
		require.NotNil(t, volume)
		require.NotNil(t, volume.Secret)
		assert.Equal(t, "settings-encryption-secret", volume.Secret.SecretName)
		require.Len(t, volume.Secret.Items, 1)
		assert.Equal(t, "sonar-secret.txt", volume.Secret.Items[0].Key)
		assert.Equal(t, "sonar-secret.txt", volume.Secret.Items[0].Path)
	})

	// The secret volume/volumeMount must still render even without readOnlyRootFilesystem, since
	// the two are independently gated (SONAR-31746 must not regress SONAR-31523's tmp-volume fix).
	t.Run("set still mounts the secret without readOnlyRootFilesystem", func(t *testing.T) {
		deployment := renderAgentOrchestrator(t, map[string]string{
			"sonarSecretKey": "settings-encryption-secret",
			"agentOrchestrator.containerSecurityContext.readOnlyRootFilesystem": "false",
		})
		podSpec := deployment.Spec.Template.Spec
		container := podSpec.Containers[0]

		assert.NotNil(t, findVolumeMount(container, "secret"))
		assert.NotNil(t, findVolume(podSpec, "secret"))
		assert.Nil(t, findVolumeMount(container, "tmp"))
		assert.Nil(t, findVolume(podSpec, "tmp"))
	})
}

func agentOrchestratorPullSecretNames(deployment appsv1.Deployment) []string {
	var names []string
	for _, s := range deployment.Spec.Template.Spec.ImagePullSecrets {
		names = append(names, s.Name)
	}
	return names
}

// The wait-for-sonarqube init container pulls the SonarQube image, so it needs the application
// nodes' pull secrets, not just the orchestrator's own (SONAR-31523).
//
// This test cannot be table-driven: it exercises applicationNodes.image.pullSecret, a DCE-only
// field. See TestAgentOrchestratorPullSecretsSonarqube for the sonarqube-chart equivalent.
func TestAgentOrchestratorPullSecrets(t *testing.T) {
	dce := agentCharts[0]

	t.Run("none configured renders no imagePullSecrets", func(t *testing.T) {
		assert.Empty(t, agentOrchestratorPullSecretNames(renderAgentOrchestrator(t, dce, nil)))
	})

	t.Run("application nodes' pull secret and list reach the pod", func(t *testing.T) {
		names := agentOrchestratorPullSecretNames(renderAgentOrchestrator(t, dce, map[string]string{
			"applicationNodes.image.pullSecret":          "app-secret",
			"applicationNodes.image.pullSecrets[0].name": "app-secret-list",
		}))
		assert.Equal(t, []string{"app-secret", "app-secret-list"}, names)
	})

	t.Run("orchestrator's own pull secret and list still apply", func(t *testing.T) {
		names := agentOrchestratorPullSecretNames(renderAgentOrchestrator(t, dce, map[string]string{
			"agentOrchestrator.image.pullSecret":          "orch-secret",
			"agentOrchestrator.image.pullSecrets[0].name": "orch-secret-list",
		}))
		assert.Equal(t, []string{"orch-secret", "orch-secret-list"}, names)
	})

	t.Run("both combine, application nodes first", func(t *testing.T) {
		names := agentOrchestratorPullSecretNames(renderAgentOrchestrator(t, dce, map[string]string{
			"applicationNodes.image.pullSecret":  "app-secret",
			"agentOrchestrator.image.pullSecret": "orch-secret",
		}))
		assert.Equal(t, []string{"app-secret", "orch-secret"}, names)
	})
}

// Sibling of TestAgentOrchestratorPullSecrets for the sonarqube chart, which has no
// applicationNodes indirection: the wait-for-sonarqube init container instead reuses the
// top-level image.pullSecret(s) via the sonarqube.agent.sonarqubeImage shim.
func TestAgentOrchestratorPullSecretsSonarqube(t *testing.T) {
	sonarqube := agentCharts[1]

	t.Run("none configured renders no imagePullSecrets", func(t *testing.T) {
		assert.Empty(t, agentOrchestratorPullSecretNames(renderAgentOrchestrator(t, sonarqube, nil)))
	})

	t.Run("application image's pull secret and list reach the pod", func(t *testing.T) {
		names := agentOrchestratorPullSecretNames(renderAgentOrchestrator(t, sonarqube, map[string]string{
			"image.pullSecret":          "app-secret",
			"image.pullSecrets[0].name": "app-secret-list",
		}))
		assert.Equal(t, []string{"app-secret", "app-secret-list"}, names)
	})

	t.Run("orchestrator's own pull secret and list still apply", func(t *testing.T) {
		names := agentOrchestratorPullSecretNames(renderAgentOrchestrator(t, sonarqube, map[string]string{
			"agentOrchestrator.image.pullSecret":          "orch-secret",
			"agentOrchestrator.image.pullSecrets[0].name": "orch-secret-list",
		}))
		assert.Equal(t, []string{"orch-secret", "orch-secret-list"}, names)
	})

	t.Run("both combine, application image first", func(t *testing.T) {
		names := agentOrchestratorPullSecretNames(renderAgentOrchestrator(t, sonarqube, map[string]string{
			"image.pullSecret":                   "app-secret",
			"agentOrchestrator.image.pullSecret": "orch-secret",
		}))
		assert.Equal(t, []string{"app-secret", "orch-secret"}, names)
	})
}
