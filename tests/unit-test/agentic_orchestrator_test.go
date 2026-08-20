package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func renderAgenticOrchestrator(t *testing.T, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{"test-cases-values/sonarqube-dce/agentic-orchestrator-enabled.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/agentic/orchestrator.yaml"})
	require.NoError(t, err)

	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	require.NotEmpty(t, deployment.Name, "no Deployment rendered by templates/agentic/orchestrator.yaml")
	return deployment
}

func TestAgenticOrchestratorContainerSecurityContext(t *testing.T) {
	deployment := renderAgenticOrchestrator(t, nil)
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
}

// Clearing readOnlyRootFilesystem drops the securityContext key and the chart-managed /tmp
// emptyDir, since neither is needed without it.
func TestAgenticOrchestratorNoReadOnlyRootFilesystemNoTmpVolume(t *testing.T) {
	t.Run("readOnlyRootFilesystem: false", func(t *testing.T) {
		deployment := renderAgenticOrchestrator(t, map[string]string{
			"orchestrator.containerSecurityContext.readOnlyRootFilesystem": "false",
		})
		podSpec := deployment.Spec.Template.Spec
		require.Len(t, podSpec.Containers, 1)
		assert.Empty(t, podSpec.Containers[0].VolumeMounts)
		assert.Empty(t, podSpec.Volumes)
	})

	// containerSecurityContext: null is the chart's documented convention for disabling a
	// default map entirely (see gvisor.nodeSelector) - it must not nil-pointer.
	t.Run("containerSecurityContext: null", func(t *testing.T) {
		deployment := renderAgenticOrchestrator(t, map[string]string{
			"orchestrator.containerSecurityContext": "null",
		})
		podSpec := deployment.Spec.Template.Spec
		require.Len(t, podSpec.Containers, 1)
		assert.Nil(t, podSpec.Containers[0].SecurityContext)
		assert.Empty(t, podSpec.Containers[0].VolumeMounts)
		assert.Empty(t, podSpec.Volumes)
	})
}

func TestAgenticOrchestratorResources(t *testing.T) {
	deployment := renderAgenticOrchestrator(t, nil)
	container := deployment.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "250m", container.Resources.Requests.Cpu().String())
	assert.Equal(t, "512Mi", container.Resources.Requests.Memory().String())
	assert.Equal(t, "512Mi", container.Resources.Requests.StorageEphemeral().String())
	assert.Equal(t, "1", container.Resources.Limits.Cpu().String())
	assert.Equal(t, "1Gi", container.Resources.Limits.Memory().String())
	assert.Equal(t, "2Gi", container.Resources.Limits.StorageEphemeral().String())
}

func agenticOrchestratorPullSecretNames(deployment appsv1.Deployment) []string {
	var names []string
	for _, s := range deployment.Spec.Template.Spec.ImagePullSecrets {
		names = append(names, s.Name)
	}
	return names
}

// The wait-for-sonarqube init container pulls the SonarQube image, so it needs the application
// nodes' pull secrets, not just the orchestrator's own (SONAR-31523).
func TestAgenticOrchestratorPullSecrets(t *testing.T) {
	t.Run("none configured renders no imagePullSecrets", func(t *testing.T) {
		assert.Empty(t, agenticOrchestratorPullSecretNames(renderAgenticOrchestrator(t, nil)))
	})

	t.Run("application nodes' pull secret and list reach the pod", func(t *testing.T) {
		names := agenticOrchestratorPullSecretNames(renderAgenticOrchestrator(t, map[string]string{
			"applicationNodes.image.pullSecret":          "app-secret",
			"applicationNodes.image.pullSecrets[0].name": "app-secret-list",
		}))
		assert.Equal(t, []string{"app-secret", "app-secret-list"}, names)
	})

	t.Run("orchestrator's own pull secret and list still apply", func(t *testing.T) {
		names := agenticOrchestratorPullSecretNames(renderAgenticOrchestrator(t, map[string]string{
			"orchestrator.image.pullSecret":          "orch-secret",
			"orchestrator.image.pullSecrets[0].name": "orch-secret-list",
		}))
		assert.Equal(t, []string{"orch-secret", "orch-secret-list"}, names)
	})

	t.Run("both combine, application nodes first", func(t *testing.T) {
		names := agenticOrchestratorPullSecretNames(renderAgenticOrchestrator(t, map[string]string{
			"applicationNodes.image.pullSecret": "app-secret",
			"orchestrator.image.pullSecret":     "orch-secret",
		}))
		assert.Equal(t, []string{"app-secret", "orch-secret"}, names)
	})
}
