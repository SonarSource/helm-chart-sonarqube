package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentOrchestratorPrivateCATruststore(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, customCA := range []bool{false, true} {
				name := "without CA"
				values := map[string]string{}
				if customCA {
					name = "with CA and extra volume"
					values["agentOrchestrator.extraVolumes[0].name"] = "custom-ca"
					values["agentOrchestrator.extraVolumes[0].secret.secretName"] = "private-ca"
					values["agentOrchestrator.extraVolumeMounts[0].name"] = "custom-ca"
					values["agentOrchestrator.extraVolumeMounts[0].mountPath"] = "/custom-ca"
					values["agentOrchestrator.extraVolumeMounts[0].readOnly"] = "true"
				}
				t.Run(name, func(t *testing.T) {
					pod := renderAgentOrchestrator(t, chart, values).Spec.Template.Spec
					container := pod.Containers[0]
					mounts := agentVolumeMountsByName(container.VolumeMounts)
					require.Contains(t, mounts, "orchestrator-truststore")
					mount := mounts["orchestrator-truststore"]
					assert.Equal(t, "/opt/java/openjdk/lib/security/cacerts", mount.MountPath)
					assert.Equal(t, "cacerts", mount.SubPath)
					assert.False(t, mount.ReadOnly)
					require.NotNil(t, container.SecurityContext.ReadOnlyRootFilesystem)
					assert.True(t, *container.SecurityContext.ReadOnlyRootFilesystem)
					volume := agentVolumesByName(pod.Volumes)["orchestrator-truststore"]
					require.NotNil(t, volume.EmptyDir)
					assert.Equal(t, "10Mi", volume.EmptyDir.SizeLimit.String())
					require.NotEmpty(t, pod.InitContainers)
					init := pod.InitContainers[0]
					assert.Equal(t, "init-orchestrator-truststore", init.Name)
					assert.Equal(t, container.Image, init.Image)
					assert.Equal(t, container.ImagePullPolicy, init.ImagePullPolicy)
					assert.Equal(t, container.SecurityContext, init.SecurityContext)
					require.Len(t, init.Command, 4)
					assert.Equal(t, []string{"sh", "-e", "-c"}, init.Command[:3])
					assert.Contains(t, init.Command[3], `cp -L "$JAVA_HOME/lib/security/cacerts" /orchestrator-truststore/cacerts`)
					require.Len(t, init.VolumeMounts, 1)
					assert.Equal(t, mount.Name, init.VolumeMounts[0].Name)
					assert.Equal(t, "/orchestrator-truststore", init.VolumeMounts[0].MountPath)
					if customCA {
						assert.Equal(t, "/custom-ca", mounts["custom-ca"].MountPath)
						assert.True(t, mounts["custom-ca"].ReadOnly)
					}
				})
			}
		})
	}
}

func TestAgentOrchestratorWritableRootSkipsTruststoreCopy(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, values := range []map[string]string{
				{"agentOrchestrator.containerSecurityContext.readOnlyRootFilesystem": "false"},
				{"agentOrchestrator.containerSecurityContext": "null"},
			} {
				pod := renderAgentOrchestrator(t, chart, values).Spec.Template.Spec
				assert.NotContains(t, agentVolumeMountsByName(pod.Containers[0].VolumeMounts), "orchestrator-truststore")
				assert.NotContains(t, agentVolumesByName(pod.Volumes), "orchestrator-truststore")
				for _, init := range pod.InitContainers {
					assert.NotEqual(t, "init-orchestrator-truststore", init.Name)
				}
			}
		})
	}
}
