package sonarqube

import (
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

func TestPrometheusExporter(t *testing.T) {
	utils.RunPrometheusExporterTest(t, utils.PrometheusExporterTestSpec{
		ChartName:     "sonarqube",
		PodSelector:   "app=sonarqube,release=sonarqube",
		ContainerName: "sonarqube",
		Values: map[string]string{
			utils.TESTS_ENABLING_ACTION:  "false",
			"edition":                    "enterprise",
			"monitoringPasscode":         "monitoringPasscode",
			"prometheusExporter.enabled": "true",
			"prometheusExporter.sha256":  utils.PrometheusExporterSHA256,
		},
	})
}
