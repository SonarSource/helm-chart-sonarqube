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

// renderDCEJsonOutputContainers renders the DCE application and search workloads and returns their
// main containers, so SONAR_LOG_JSONOUTPUT can be checked on both node types.
func renderDCEJsonOutputContainers(t *testing.T, valuesFiles []string, setValues map[string]string) map[string]corev1.Container {
	t.Helper()
	base := dceEsMajorBaseValues()
	for k, v := range setValues {
		base[k] = v
	}
	opts := &helm.Options{Logger: logger.Discard, ValuesFiles: valuesFiles, SetValues: base}

	appOutput, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/sonarqube-application.yaml"})
	require.NoError(t, err)
	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, appOutput, &deployment)

	searchOutput, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/sonarqube-search.yaml"})
	require.NoError(t, err)
	var sts appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, searchOutput, &sts)

	return map[string]corev1.Container{
		"application": deployment.Spec.Template.Spec.Containers[0],
		"search":      sts.Spec.Template.Spec.Containers[0],
	}
}

// Users who can't pull the chart from the Helm repo copy its values.yaml and drop the deprecated
// logging.jsonOutput, leaving no default to fall back to. The chart must still render, keep the
// "false" default, and honour SONAR_LOG_JSONOUTPUT.
func TestDCELoggingJsonOutputMissing(t *testing.T) {
	cases := []struct {
		name        string
		valuesFiles []string
		setValues   map[string]string
		expected    string
	}{
		{"logging removed", nil, map[string]string{"logging": "null"}, "false"},
		{"logging.jsonOutput removed", nil, map[string]string{"logging.jsonOutput": "null"}, "false"},
		{"logging removed with env override", []string{"test-cases-values/sonarqube-dce/logging-json-output-env.yaml"}, map[string]string{"logging": "null"}, "true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for node, container := range renderDCEJsonOutputContainers(t, tc.valuesFiles, tc.setValues) {
				env := findEnvByName(container, "SONAR_LOG_JSONOUTPUT")
				require.NotNil(t, env, "%s container must set SONAR_LOG_JSONOUTPUT", node)
				assert.Equal(t, tc.expected, env.Value, "%s container", node)
			}
		})
	}
}
