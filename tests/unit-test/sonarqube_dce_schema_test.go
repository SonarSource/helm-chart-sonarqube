package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

var dceChartPath string = "../../charts/sonarqube-dce"
var dceReleaseName string = "sonarqube-dce"

func findInitContainerByName(containers []v1.Container, name string) *v1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

func renderDCESearchTemplate(t *testing.T, valuesFile string, helmOptions *helm.Options) appsv1.StatefulSet {
	t.Helper()
	helmOptions.ValuesFiles = []string{valuesFile}
	output, err := helm.RenderTemplateE(t, helmOptions, dceChartPath, dceReleaseName, []string{"templates/sonarqube-search.yaml"})
	assert.NoError(t, err)
	var rendered appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &rendered)
	return rendered
}

func renderDCEAppTemplate(t *testing.T, valuesFile string, helmOptions *helm.Options) appsv1.Deployment {
	t.Helper()
	helmOptions.ValuesFiles = []string{valuesFile}
	output, err := helm.RenderTemplateE(t, helmOptions, dceChartPath, dceReleaseName, []string{"templates/sonarqube-application.yaml"})
	assert.NoError(t, err)
	var rendered appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &rendered)
	return rendered
}

// TestSearchNodeExtraInitContainers verifies that extra init containers defined under
// searchNodes.extraInitContainers are rendered in the search StatefulSet.
func TestSearchNodeExtraInitContainers(t *testing.T) {
	var dceHelmOptions *helm.Options = &helm.Options{
		Logger: logger.Discard,
	}
	rendered := renderDCESearchTemplate(t, "test-cases-values/sonarqube-dce/test-search-extra-init-containers.yaml", dceHelmOptions)
	container := findInitContainerByName(rendered.Spec.Template.Spec.InitContainers, "extra-search-init")
	assert.NotNil(t, container, "Expected 'extra-search-init' init container to be present in search pod")
}

// TestSearchNodeBackwardCompatExtraInitContainers verifies that the root-level extraInitContainers
// still appears in the search StatefulSet when searchNodes.extraInitContainers is not set,
// preserving backward compatibility for existing users.
func TestSearchNodeBackwardCompatExtraInitContainers(t *testing.T) {
	var dceHelmOptions *helm.Options = &helm.Options{
		Logger: logger.Discard,
	}
	rendered := renderDCESearchTemplate(t, "test-cases-values/sonarqube-dce/test-backward-compat-extra-init-containers.yaml", dceHelmOptions)
	container := findInitContainerByName(rendered.Spec.Template.Spec.InitContainers, "extra-init-backward-compat")
	assert.NotNil(t, container, "Expected root-level extraInitContainers to still appear in search pod for backward compatibility")
}

// TestSearchNodeExtraInitContainersPrecedence verifies that searchNodes.extraInitContainers takes
// precedence over the deprecated root-level extraInitContainers when both are set: only the
// searchNodes container should appear, the deprecated one should be absent.
func TestSearchNodeExtraInitContainersPrecedence(t *testing.T) {
	var dceHelmOptions *helm.Options = &helm.Options{
		Logger: logger.Discard,
	}
	rendered := renderDCESearchTemplate(t, "test-cases-values/sonarqube-dce/test-search-extra-init-containers-precedence.yaml", dceHelmOptions)
	initContainers := rendered.Spec.Template.Spec.InitContainers

	searchNodeContainer := findInitContainerByName(initContainers, "extra-init-search-node")
	assert.NotNil(t, searchNodeContainer, "Expected 'extra-init-search-node' from searchNodes.extraInitContainers to be present")

	deprecatedContainer := findInitContainerByName(initContainers, "extra-init-deprecated")
	assert.Nil(t, deprecatedContainer, "Expected root-level extraInitContainers to be ignored when searchNodes.extraInitContainers is set")
}

// TestApplicationNodeExtraInitContainers verifies that extra init containers defined under
// applicationNodes.extraInitContainers are rendered in the application Deployment.
func TestApplicationNodeExtraInitContainers(t *testing.T) {
	var dceHelmOptions *helm.Options = &helm.Options{
		Logger: logger.Discard,
	}
	rendered := renderDCEAppTemplate(t, "test-cases-values/sonarqube-dce/test-app-extra-init-containers.yaml", dceHelmOptions)
	container := findInitContainerByName(rendered.Spec.Template.Spec.InitContainers, "extra-app-init")
	assert.NotNil(t, container, "Expected 'extra-app-init' init container to be present in application pod")
}

// When the agentic signing secret (.Values.agenticSigningSecretKey) is set, it must be mounted
// into the search node, using its own volume/mount distinct from the sonarSecretKey one, mirroring
// that mount's own readOnly: true (SONAR-32028).
func TestSearchNodeSigningSecretMount(t *testing.T) {
	t.Run("unset renders no signing secret volume or volumeMount", func(t *testing.T) {
		dceHelmOptions := &helm.Options{Logger: logger.Discard}
		rendered := renderDCESearchTemplate(t, "test-cases-values/sonarqube-dce/agent-all-disabled.yaml", dceHelmOptions)
		container := rendered.Spec.Template.Spec.Containers[0]
		assert.Nil(t, findVolumeMountByName(container, "agentic-signing-secret"))
		assert.Nil(t, findVolumeByName(rendered.Spec.Template.Spec.Volumes, "agentic-signing-secret"))
	})

	t.Run("set mounts the secret read-only", func(t *testing.T) {
		dceHelmOptions := &helm.Options{
			Logger:    logger.Discard,
			SetValues: map[string]string{"agenticSigningSecretKey": "agentic-signing-secret"},
		}
		rendered := renderDCESearchTemplate(t, "test-cases-values/sonarqube-dce/agent-all-disabled.yaml", dceHelmOptions)
		podSpec := rendered.Spec.Template.Spec
		container := podSpec.Containers[0]

		volumeMount := findVolumeMountByName(container, "agentic-signing-secret")
		require.NotNil(t, volumeMount)
		assert.Equal(t, "/opt/sonarqube/secret-agentic-signing/", volumeMount.MountPath)
		assert.True(t, volumeMount.ReadOnly)

		volume := findVolumeByName(podSpec.Volumes, "agentic-signing-secret")
		require.NotNil(t, volume)
		require.NotNil(t, volume.Secret)
		assert.Equal(t, "agentic-signing-secret", volume.Secret.SecretName)
		require.Len(t, volume.Secret.Items, 1)
		assert.Equal(t, "sonar-agentic-signing-secret.txt", volume.Secret.Items[0].Key)
		assert.Equal(t, "sonar-agentic-signing-secret.txt", volume.Secret.Items[0].Path)
	})
}

// Same as TestSearchNodeSigningSecretMount, but for the application node, which mirrors its own
// sonarSecretKey mount by NOT setting readOnly (SONAR-32028).
func TestApplicationNodeSigningSecretMount(t *testing.T) {
	t.Run("unset renders no signing secret volume or volumeMount", func(t *testing.T) {
		dceHelmOptions := &helm.Options{Logger: logger.Discard}
		rendered := renderDCEAppTemplate(t, "test-cases-values/sonarqube-dce/agent-all-disabled.yaml", dceHelmOptions)
		container := rendered.Spec.Template.Spec.Containers[0]
		assert.Nil(t, findVolumeMountByName(container, "agentic-signing-secret"))
		assert.Nil(t, findVolumeByName(rendered.Spec.Template.Spec.Volumes, "agentic-signing-secret"))
	})

	t.Run("set mounts the secret", func(t *testing.T) {
		dceHelmOptions := &helm.Options{
			Logger:    logger.Discard,
			SetValues: map[string]string{"agenticSigningSecretKey": "agentic-signing-secret"},
		}
		rendered := renderDCEAppTemplate(t, "test-cases-values/sonarqube-dce/agent-all-disabled.yaml", dceHelmOptions)
		podSpec := rendered.Spec.Template.Spec
		container := podSpec.Containers[0]

		volumeMount := findVolumeMountByName(container, "agentic-signing-secret")
		require.NotNil(t, volumeMount)
		assert.Equal(t, "/opt/sonarqube/secret-agentic-signing/", volumeMount.MountPath)
		assert.False(t, volumeMount.ReadOnly)

		volume := findVolumeByName(podSpec.Volumes, "agentic-signing-secret")
		require.NotNil(t, volume)
		require.NotNil(t, volume.Secret)
		assert.Equal(t, "agentic-signing-secret", volume.Secret.SecretName)
		require.Len(t, volume.Secret.Items, 1)
		assert.Equal(t, "sonar-agentic-signing-secret.txt", volume.Secret.Items[0].Key)
		assert.Equal(t, "sonar-agentic-signing-secret.txt", volume.Secret.Items[0].Path)
	})
}
