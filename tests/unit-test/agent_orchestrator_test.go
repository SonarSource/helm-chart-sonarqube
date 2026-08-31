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

func findEnvByName(container corev1.Container, name string) *corev1.EnvVar {
	for i := range container.Env {
		if container.Env[i].Name == name {
			return &container.Env[i]
		}
	}
	return nil
}

func findVolumeMountByName(container corev1.Container, name string) *corev1.VolumeMount {
	for i := range container.VolumeMounts {
		if container.VolumeMounts[i].Name == name {
			return &container.VolumeMounts[i]
		}
	}
	return nil
}

// When the chart-wide encryption key secret (.Values.sonarSecretKey) is set, it must be mounted
// into the orchestrator too, mirroring how it's mounted into the SonarQube application/search
// pods, with its path exposed via AGENTIC_SECRET_KEY_PATH (SONAR-31746).
func TestAgentOrchestratorEncryptionKeyMount(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("unset renders no encryption key env var, volume, or volumeMount", func(t *testing.T) {
				deployment := renderAgentOrchestrator(t, chart, nil)
				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Nil(t, findEnvByName(container, "AGENTIC_SECRET_KEY_PATH"))
				assert.Nil(t, findVolumeMountByName(container, "secret"))
				assert.Nil(t, findVolumeByName(deployment.Spec.Template.Spec.Volumes, "secret"))
			})

			t.Run("set mounts the secret and exposes its path", func(t *testing.T) {
				deployment := renderAgentOrchestrator(t, chart, map[string]string{
					"sonarSecretKey": "settings-encryption-secret",
				})
				podSpec := deployment.Spec.Template.Spec
				container := podSpec.Containers[0]

				env := findEnvByName(container, "AGENTIC_SECRET_KEY_PATH")
				require.NotNil(t, env)
				assert.Equal(t, "/sonarcloud/secret/sonar-secret.txt", env.Value)

				volumeMount := findVolumeMountByName(container, "secret")
				require.NotNil(t, volumeMount)
				assert.Equal(t, "/sonarcloud/secret/", volumeMount.MountPath)

				volume := findVolumeByName(podSpec.Volumes, "secret")
				require.NotNil(t, volume)
				require.NotNil(t, volume.Secret)
				assert.Equal(t, "settings-encryption-secret", volume.Secret.SecretName)
				require.Len(t, volume.Secret.Items, 1)
				assert.Equal(t, "sonar-secret.txt", volume.Secret.Items[0].Key)
				assert.Equal(t, "sonar-secret.txt", volume.Secret.Items[0].Path)
			})

			// The secret volume/volumeMount must still render even without readOnlyRootFilesystem,
			// since the two are independently gated (SONAR-31746 must not regress SONAR-31523's
			// tmp-volume fix).
			t.Run("set still mounts the secret without readOnlyRootFilesystem", func(t *testing.T) {
				deployment := renderAgentOrchestrator(t, chart, map[string]string{
					"sonarSecretKey": "settings-encryption-secret",
					"agentOrchestrator.containerSecurityContext.readOnlyRootFilesystem": "false",
				})
				podSpec := deployment.Spec.Template.Spec
				container := podSpec.Containers[0]

				assert.NotNil(t, findVolumeMountByName(container, "secret"))
				assert.NotNil(t, findVolumeByName(podSpec.Volumes, "secret"))
				assert.Nil(t, findVolumeMountByName(container, "tmp"))
				assert.Nil(t, findVolumeByName(podSpec.Volumes, "tmp"))
			})
		})
	}
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

// The default S3 backend is unaffected by the FILESYSTEM/NFS addition (SONAR-31980): the
// long, SONAR_-prefixed vars must still render with their StoragePropertiesParser defaults, since
// the orchestrator's own application.yml has no placeholder for either of them.
func TestAgentOrchestratorStorageEnv(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("defaults to S3, no filesystem base dir", func(t *testing.T) {
				container := renderAgentOrchestrator(t, chart, nil).Spec.Template.Spec.Containers[0]

				typeEnv := findEnvByName(container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_TYPE")
				require.NotNil(t, typeEnv)
				assert.Equal(t, "S3", typeEnv.Value)

				baseDirEnv := findEnvByName(container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_FILESYSTEM_BASE_DIR")
				require.NotNil(t, baseDirEnv)
				assert.Equal(t, "", baseDirEnv.Value)
			})

			// A FILESYSTEM backend needs no bucket/region (see agent_dependency_validation_test.go
			// for the matching validation-relaxation case) and hands the base dir through instead.
			t.Run("FILESYSTEM sets the base dir, no bucket needed", func(t *testing.T) {
				container := renderAgentOrchestrator(t, chart, map[string]string{
					"agentOrchestrator.storage.bucket":             "",
					"agentOrchestrator.storage.type":               "FILESYSTEM",
					"agentOrchestrator.storage.filesystem.baseDir": "/agentic-storage",
				}).Spec.Template.Spec.Containers[0]

				typeEnv := findEnvByName(container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_TYPE")
				require.NotNil(t, typeEnv)
				assert.Equal(t, "FILESYSTEM", typeEnv.Value)

				baseDirEnv := findEnvByName(container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_FILESYSTEM_BASE_DIR")
				require.NotNil(t, baseDirEnv)
				assert.Equal(t, "/agentic-storage", baseDirEnv.Value)
			})
		})
	}
}

// User-supplied env comes last, so it can override the values the chart wires automatically -
// same contract as agent-runtime.yaml's hunterAgent.env/remediationAgent.env (SONAR-31980).
func TestAgentOrchestratorExtraEnv(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := renderAgentOrchestrator(t, chart, map[string]string{
				"agentOrchestrator.env[0].name":  "MY_VAR",
				"agentOrchestrator.env[0].value": "my-value",
			}).Spec.Template.Spec.Containers[0]

			extra := findEnvByName(container, "MY_VAR")
			require.NotNil(t, extra, "extra env must reach the container")
			assert.Equal(t, "my-value", extra.Value)
			assert.Equal(t, "MY_VAR", container.Env[len(container.Env)-1].Name,
				"user-supplied env must come last so it overrides the auto-wired vars")
		})
	}
}

// extraVolumes/extraVolumeMounts merge into the pod's existing volumes/volumeMounts (the tmp
// volume from readOnlyRootFilesystem, the secret volume from sonarSecretKey) rather than
// replacing them - mirrors agent-runtime.yaml's hunterAgent/remediationAgent equivalents
// (SONAR-31980). This is the mechanism a shared FILESYSTEM storage mount relies on.
func TestAgentOrchestratorExtraVolumes(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("merges alongside the tmp volume", func(t *testing.T) {
				deployment := renderAgentOrchestrator(t, chart, map[string]string{
					"agentOrchestrator.extraVolumes[0].name":                            "agentic-storage",
					"agentOrchestrator.extraVolumes[0].persistentVolumeClaim.claimName": "agentic-storage",
					"agentOrchestrator.extraVolumeMounts[0].name":                       "agentic-storage",
					"agentOrchestrator.extraVolumeMounts[0].mountPath":                  "/agentic-storage",
				})
				podSpec := deployment.Spec.Template.Spec
				container := podSpec.Containers[0]

				require.NotNil(t, findVolumeMountByName(container, "tmp"), "readOnlyRootFilesystem's tmp mount must survive")
				extraMount := findVolumeMountByName(container, "agentic-storage")
				require.NotNil(t, extraMount)
				assert.Equal(t, "/agentic-storage", extraMount.MountPath)

				require.NotNil(t, findVolumeByName(podSpec.Volumes, "tmp"))
				extraVolume := findVolumeByName(podSpec.Volumes, "agentic-storage")
				require.NotNil(t, extraVolume)
				require.NotNil(t, extraVolume.PersistentVolumeClaim)
				assert.Equal(t, "agentic-storage", extraVolume.PersistentVolumeClaim.ClaimName)
			})

			// Independent of readOnlyRootFilesystem/sonarSecretKey, same as the encryption key
			// mount (SONAR-31746 must not regress; see TestAgentOrchestratorEncryptionKeyMount).
			t.Run("renders without readOnlyRootFilesystem or sonarSecretKey", func(t *testing.T) {
				deployment := renderAgentOrchestrator(t, chart, map[string]string{
					"agentOrchestrator.containerSecurityContext.readOnlyRootFilesystem": "false",
					"agentOrchestrator.extraVolumes[0].name":                            "agentic-storage",
					"agentOrchestrator.extraVolumes[0].persistentVolumeClaim.claimName": "agentic-storage",
					"agentOrchestrator.extraVolumeMounts[0].name":                       "agentic-storage",
					"agentOrchestrator.extraVolumeMounts[0].mountPath":                  "/agentic-storage",
				})
				podSpec := deployment.Spec.Template.Spec
				container := podSpec.Containers[0]

				require.Nil(t, findVolumeMountByName(container, "tmp"))
				extraMount := findVolumeMountByName(container, "agentic-storage")
				require.NotNil(t, extraMount)
				assert.Equal(t, "/agentic-storage", extraMount.MountPath)
			})
		})
	}
}
