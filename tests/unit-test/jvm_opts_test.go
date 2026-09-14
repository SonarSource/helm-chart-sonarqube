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

// jvmOptsValueKey returns the values path for the deprecated jvmOpts/jvmCeOpts knob, which sits
// under applicationNodes in sonarqube-dce but at the chart root in sonarqube.
func jvmOptsValueKey(chart agentChart, field string) string {
	if chart.name == "sonarqube-dce" {
		return "applicationNodes." + field
	}
	return field
}

// sonarPropertyValueKey escapes the dots of a sonar.properties key so it survives Helm's --set
// dotted-path parsing as a single map key instead of being split into nested maps.
func sonarPropertyValueKey(chart agentChart, property string) string {
	return chart.sonarPropertiesPath + "." + strings.ReplaceAll(property, ".", `\.`)
}

func jvmOptsAppContainer(t *testing.T, chart agentChart, fixture string, setValues map[string]string) corev1.PodSpec {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/" + fixture},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{chart.appTemplate})
	require.NoError(t, err)

	if chart.name == "sonarqube-dce" {
		var deployment appsv1.Deployment
		helm.UnmarshalK8SYaml(t, output, &deployment)
		return deployment.Spec.Template.Spec
	}

	var sts appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &sts)
	return sts.Spec.Template.Spec
}

// Setting only the deprecated jvmOpts/jvmCeOpts (no conflicting sonarProperties key) must keep
// behaving exactly as before: it becomes the whole SONAR_*_JAVAOPTS value.
func TestJvmOptsAloneIsUsedAsIs(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := jvmOptsAppContainer(t, chart, "jvm-opts-baseline.yaml", map[string]string{
				jvmOptsValueKey(chart, "jvmOpts"):   "-Xmx2g",
				jvmOptsValueKey(chart, "jvmCeOpts"): "-Xmx1g",
			}).Containers[0]

			assert.Equal(t, "-Xmx2g", findEnvByName(container, "SONAR_WEB_JAVAOPTS").Value)
			assert.Equal(t, "-Xmx1g", findEnvByName(container, "SONAR_CE_JAVAOPTS").Value)
		})
	}
}

// Setting only sonar.web.javaOpts/sonar.ce.javaOpts (no jvmOpts/jvmCeOpts) must also keep behaving
// exactly as before: it becomes the whole SONAR_*_JAVAOPTS value.
func TestSonarPropertiesJavaOptsAloneIsUsedAsIs(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := jvmOptsAppContainer(t, chart, "jvm-opts-baseline.yaml", map[string]string{
				sonarPropertyValueKey(chart, "sonar.web.javaOpts"): "-Dsome.other=true",
				sonarPropertyValueKey(chart, "sonar.ce.javaOpts"):  "-Dce.other=true",
			}).Containers[0]

			assert.Equal(t, "-Dsome.other=true", findEnvByName(container, "SONAR_WEB_JAVAOPTS").Value)
			assert.Equal(t, "-Dce.other=true", findEnvByName(container, "SONAR_CE_JAVAOPTS").Value)
		})
	}
}

// SONAR-32320: when both the deprecated jvmOpts/jvmCeOpts and their sonarProperties replacement are
// set, the resulting env var must combine both instead of silently dropping jvmOpts/jvmCeOpts -
// losing a heap setting like -Xmx this way could leave SonarQube running on an unbounded default
// heap.
func TestJvmOptsMergesWithSonarPropertiesJavaOpts(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := jvmOptsAppContainer(t, chart, "jvm-opts-baseline.yaml", map[string]string{
				jvmOptsValueKey(chart, "jvmOpts"):                  "-Xmx2g",
				jvmOptsValueKey(chart, "jvmCeOpts"):                "-Xmx1g",
				sonarPropertyValueKey(chart, "sonar.web.javaOpts"): "-Dsome.other=true",
				sonarPropertyValueKey(chart, "sonar.ce.javaOpts"):  "-Dce.other=true",
			}).Containers[0]

			assert.Equal(t, "-Xmx2g -Dsome.other=true", findEnvByName(container, "SONAR_WEB_JAVAOPTS").Value)
			assert.Equal(t, "-Xmx1g -Dce.other=true", findEnvByName(container, "SONAR_CE_JAVAOPTS").Value)
		})
	}
}
