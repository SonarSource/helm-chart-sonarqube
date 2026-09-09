package sonarqube

import (
	"path/filepath"
	"runtime"
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

func sonarqubeChartPath() string {
	_, testFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(testFile), "..", "..", "..", "charts", "sonarqube")
}

func TestPrometheusExporter(t *testing.T) {
	utils.RunPrometheusExporterTest(t, utils.PrometheusExporterTestSpec{
		ChartName:     "sonarqube",
		ChartPath:     sonarqubeChartPath(),
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
