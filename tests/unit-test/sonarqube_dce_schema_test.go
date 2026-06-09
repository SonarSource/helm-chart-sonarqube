package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

var dceChartPath string = "../../charts/sonarqube-dce"
var dceReleaseName string = "sonarqube-dce"
var dceHelmOptions *helm.Options = &helm.Options{
	Logger: logger.Discard,
}

func findInitContainerByName(containers []v1.Container, name string) *v1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

func renderDCESearchTemplate(t *testing.T, valuesFile string) appsv1.StatefulSet {
	t.Helper()
	dceHelmOptions.ValuesFiles = []string{valuesFile}
	output, err := helm.RenderTemplateE(t, dceHelmOptions, dceChartPath, dceReleaseName, []string{"templates/sonarqube-search.yaml"})
	assert.NoError(t, err)
	var rendered appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &rendered)
	return rendered
}

func renderDCEAppTemplate(t *testing.T, valuesFile string) appsv1.Deployment {
	t.Helper()
	dceHelmOptions.ValuesFiles = []string{valuesFile}
	output, err := helm.RenderTemplateE(t, dceHelmOptions, dceChartPath, dceReleaseName, []string{"templates/sonarqube-application.yaml"})
	assert.NoError(t, err)
	var rendered appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &rendered)
	return rendered
}

// TestSearchNodeExtraInitContainers verifies that extra init containers defined under
// searchNodes.extraInitContainers are rendered in the search StatefulSet.
func TestSearchNodeExtraInitContainers(t *testing.T) {
	rendered := renderDCESearchTemplate(t, "test-cases-values/sonarqube-dce/test-search-extra-init-containers.yaml")
	container := findInitContainerByName(rendered.Spec.Template.Spec.InitContainers, "extra-search-init")
	assert.NotNil(t, container, "Expected 'extra-search-init' init container to be present in search pod")
}

// TestSearchNodeBackwardCompatExtraInitContainers verifies that the root-level extraInitContainers
// still appears in the search StatefulSet when searchNodes.extraInitContainers is not set,
// preserving backward compatibility for existing users.
func TestSearchNodeBackwardCompatExtraInitContainers(t *testing.T) {
	rendered := renderDCESearchTemplate(t, "test-cases-values/sonarqube-dce/test-backward-compat-extra-init-containers.yaml")
	container := findInitContainerByName(rendered.Spec.Template.Spec.InitContainers, "extra-init-backward-compat")
	assert.NotNil(t, container, "Expected root-level extraInitContainers to still appear in search pod for backward compatibility")
}

// TestApplicationNodeExtraInitContainers verifies that extra init containers defined under
// applicationNodes.extraInitContainers are rendered in the application Deployment.
func TestApplicationNodeExtraInitContainers(t *testing.T) {
	rendered := renderDCEAppTemplate(t, "test-cases-values/sonarqube-dce/test-app-extra-init-containers.yaml")
	container := findInitContainerByName(rendered.Spec.Template.Spec.InitContainers, "extra-app-init")
	assert.NotNil(t, container, "Expected 'extra-app-init' init container to be present in application pod")
}
