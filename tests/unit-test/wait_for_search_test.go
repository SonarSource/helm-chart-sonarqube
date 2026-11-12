package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

func renderWaitForSearchApp(t *testing.T, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	base := dceEsMajorBaseValues()
	for k, v := range setValues {
		base[k] = v
	}
	opts := &helm.Options{Logger: logger.Discard, SetValues: base}
	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/sonarqube-application.yaml"})
	require.NoError(t, err)
	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	return deployment
}

func TestWaitForSearchOffUnlessEnabled(t *testing.T) {
	for _, setValues := range []map[string]string{
		nil,
		{"applicationNodes.waitForSearch.enabled": "false"},
		{
			"applicationNodes.waitForSearch.enabled": "true",
			"istio.enabled":                          "true",
			"applicationNodes.webPort":               "4023",
			"applicationNodes.cePort":                "4024",
		},
	} {
		deployment := renderWaitForSearchApp(t, setValues)
		assert.Nil(t, findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search"))
	}
}

func TestWaitForSearchWaitsForReadyEndpoint(t *testing.T) {
	deployment := renderWaitForSearchApp(t, map[string]string{
		"applicationNodes.waitForSearch.enabled": "true",
	})
	waitFor := findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search")
	require.NotNil(t, waitFor)
	assert.Equal(t, "sonarqube:2026.5.0-datacenter-app", waitFor.Image)
	require.GreaterOrEqual(t, len(waitFor.Command), 3)
	script := waitFor.Command[2]
	assert.Contains(t, script, `host="sonarqube-dce-sonarqube-dce-search"`)
	assert.Contains(t, script, `curl --noproxy "*" --connect-timeout 5 --max-time 5 -so /dev/null "http://${host}:9001"`)
	assert.NotContains(t, script, "https://")
	assert.Empty(t, waitFor.Env)
	assert.Empty(t, waitFor.Resources.Requests)
}

func TestWaitForSearchUsesHttpsWhenNodeEncryptionIsOn(t *testing.T) {
	deployment := renderWaitForSearchApp(t, map[string]string{
		"applicationNodes.waitForSearch.enabled": "true",
		"nodeEncryption.enabled":                 "true",
	})
	waitFor := findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search")
	require.NotNil(t, waitFor)
	script := waitFor.Command[2]
	assert.Contains(t, script, `curl --noproxy "*" --connect-timeout 5 --max-time 5 -sko /dev/null "https://${host}:9001"`)
	assert.NotContains(t, script, `http://${host}:9001`)
}

func TestWaitForSearchAppliesInitContainerSettings(t *testing.T) {
	deployment := renderWaitForSearchApp(t, map[string]string{
		"applicationNodes.waitForSearch.enabled":                "true",
		"initContainers.resources.requests.cpu":                 "10m",
		"initContainers.resources.requests.memory":              "32Mi",
		"initContainers.securityContext.runAsUser":              "1000",
		"initContainers.securityContext.readOnlyRootFilesystem": "true",
	})
	waitFor := findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search")
	require.NotNil(t, waitFor)
	require.NotNil(t, waitFor.SecurityContext)
	require.NotNil(t, waitFor.SecurityContext.RunAsUser)
	assert.Equal(t, int64(1000), *waitFor.SecurityContext.RunAsUser)
	require.NotNil(t, waitFor.SecurityContext.ReadOnlyRootFilesystem)
	assert.True(t, *waitFor.SecurityContext.ReadOnlyRootFilesystem)
	assert.Equal(t, "10m", waitFor.Resources.Requests.Cpu().String())
	assert.Equal(t, "32Mi", waitFor.Resources.Requests.Memory().String())
}

func TestWaitForSearchRejectsZeroSearchReplicas(t *testing.T) {
	base := dceEsMajorBaseValues()
	base["applicationNodes.waitForSearch.enabled"] = "true"
	base["searchNodes.replicaCount"] = "0"
	opts := &helm.Options{Logger: logger.Discard, SetValues: base}
	_, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/validation.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applicationNodes.waitForSearch.enabled is true but searchNodes.replicaCount is 0")
}

func TestWaitForSearchIgnoresInitContainerImageOverride(t *testing.T) {
	deployment := renderWaitForSearchApp(t, map[string]string{
		"applicationNodes.waitForSearch.enabled": "true",
		"initContainers.image":                   "ubuntu:24.04",
	})
	waitFor := findInitContainerByName(deployment.Spec.Template.Spec.InitContainers, "wait-for-search")
	require.NotNil(t, waitFor)
	assert.Equal(t, "sonarqube:2026.5.0-datacenter-app", waitFor.Image)
}
