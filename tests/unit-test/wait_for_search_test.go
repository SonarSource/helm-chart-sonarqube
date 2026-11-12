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

func renderWaitForSearchApp(t *testing.T, valuesFiles []string, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	base := dceEsMajorBaseValues()
	for k, v := range setValues {
		base[k] = v
	}
	opts := &helm.Options{Logger: logger.Discard, ValuesFiles: valuesFiles, SetValues: base}
	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/sonarqube-application.yaml"})
	require.NoError(t, err)
	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	return deployment
}

func waitForSearchScript(t *testing.T, container *corev1.Container) string {
	t.Helper()
	require.NotNil(t, container)
	require.GreaterOrEqual(t, len(container.Command), 3)
	return container.Command[2]
}

func TestWaitForSearchDefault(t *testing.T) {
	deployment := renderWaitForSearchApp(t, nil, nil)
	initContainers := deployment.Spec.Template.Spec.InitContainers
	waitFor := findInitContainerByName(initContainers, "wait-for-search")
	script := waitForSearchScript(t, waitFor)

	assert.Equal(t, "wait-for-search", initContainers[len(initContainers)-1].Name)
	assert.Contains(t, script, "Waiting for search nodes at ${host} to report yellow or green")
	assert.Contains(t, script, "Connection errors while search is still starting are expected.")
	assert.Contains(t, script, `http://sonarqube-dce-sonarqube-dce-search:9001/_cluster/health`)
	assert.NotContains(t, script, "https://")
	assert.NotContains(t, script, "-u ")
	assert.Empty(t, waitFor.Env)
	require.NotNil(t, waitFor.SecurityContext)
	require.NotNil(t, waitFor.SecurityContext.RunAsNonRoot)
	assert.True(t, *waitFor.SecurityContext.RunAsNonRoot)
	assert.Empty(t, waitFor.Resources.Requests)
}

func TestWaitForSearchDisabled(t *testing.T) {
	deployment := renderWaitForSearchApp(t, nil, map[string]string{
		"applicationNodes.waitForSearch.enabled": "false",
	})
	assert.Nil(t, findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search"))
}

func TestWaitForSearchMissingValueDefaultsToEnabled(t *testing.T) {
	deployment := renderWaitForSearchApp(t, nil, map[string]string{
		"applicationNodes.waitForSearch": "null",
	})
	assert.NotNil(t, findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search"))
}

func TestWaitForSearchAuthentication(t *testing.T) {
	deployment := renderWaitForSearchApp(t, nil, map[string]string{
		"searchNodes.searchAuthentication.enabled":      "true",
		"searchNodes.searchAuthentication.userPassword": "search-test-password",
	})
	waitFor := findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search")
	script := waitForSearchScript(t, waitFor)

	assert.Contains(t, script, `-u "elastic:${SONAR_CLUSTER_SEARCH_PASSWORD}"`)
	assert.Contains(t, script, `http://sonarqube-dce-sonarqube-dce-search:9001/_cluster/health`)
	assert.NotContains(t, script, "search-test-password")
	require.Len(t, waitFor.Env, 1)
	assert.Equal(t, "SONAR_CLUSTER_SEARCH_PASSWORD", waitFor.Env[0].Name)
	require.NotNil(t, waitFor.Env[0].ValueFrom)
	assert.Equal(t, "sonarqube-dce-sonarqube-dce-user-pass", waitFor.Env[0].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "SONAR_CLUSTER_SEARCH_PASSWORD", waitFor.Env[0].ValueFrom.SecretKeyRef.Key)
}

func TestWaitForSearchAuthenticationAndTls(t *testing.T) {
	deployment := renderWaitForSearchApp(t, nil, map[string]string{
		"nodeEncryption.enabled":                        "true",
		"searchNodes.searchAuthentication.enabled":      "true",
		"searchNodes.searchAuthentication.userPassword": "search-test-password",
	})
	script := waitForSearchScript(t, findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search"))
	assert.Contains(t, script, `-k -u "elastic:${SONAR_CLUSTER_SEARCH_PASSWORD}"`)
	assert.Contains(t, script, `https://sonarqube-dce-sonarqube-dce-search:9001/_cluster/health`)
	assert.NotContains(t, script, "search-test-password")
}

func TestWaitForSearchSkippedWhenIstioEnabled(t *testing.T) {
	deployment := renderWaitForSearchApp(t, nil, map[string]string{
		"istio.enabled":            "true",
		"applicationNodes.webPort": "4023",
		"applicationNodes.cePort":  "4024",
	})
	assert.Nil(t, findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search"))
}

func TestWaitForSearchKeepsExtraInitContainersAndResources(t *testing.T) {
	setValues := map[string]string{
		"initContainers.resources.requests.cpu":    "10m",
		"initContainers.resources.requests.memory": "32Mi",
		"initContainers.resources.limits.cpu":      "50m",
		"initContainers.resources.limits.memory":   "64Mi",
	}
	deployment := renderWaitForSearchApp(t, []string{"test-cases-values/sonarqube-dce/test-app-extra-init-containers.yaml"}, setValues)
	initContainers := deployment.Spec.Template.Spec.InitContainers
	extra := findInitContainerByName(initContainers, "extra-app-init")
	waitFor := findInitContainerByName(initContainers, "wait-for-search")

	require.NotNil(t, extra)
	require.NotNil(t, waitFor)
	assert.Greater(t, indexOfInitContainer(initContainers, "wait-for-search"), indexOfInitContainer(initContainers, "extra-app-init"))
	assertInitContainersResources(t, waitFor.Resources)
}

func indexOfInitContainer(containers []corev1.Container, name string) int {
	for i := range containers {
		if containers[i].Name == name {
			return i
		}
	}
	return -1
}
