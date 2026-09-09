package sonarqubedce

import (
	"path/filepath"
	"runtime"
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
	"github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test/dependencies"
)

func sonarqubeDCEChartPath() string {
	_, testFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(testFile), "..", "..", "..", "charts", "sonarqube-dce")
}

func TestPrometheusExporter(t *testing.T) {
	utils.RunPrometheusExporterTest(t, utils.PrometheusExporterTestSpec{
		ChartName:         "sonarqube-dce",
		ChartPath:         sonarqubeDCEChartPath(),
		PodSelector:       "app=sonarqube-dce,release=sonarqube-dce,sonarqube.datacenter/type=app",
		ContainerName:     "sonarqube-dce",
		RequireExternalDB: true,
		Values: map[string]string{
			utils.TESTS_ENABLING_ACTION:                   "false",
			"ApplicationNodes.jwtSecret":                  "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=",
			"ApplicationNodes.prometheusExporter.enabled": "true",
			"ApplicationNodes.prometheusExporter.sha256":  utils.PrometheusExporterSHA256,
			"monitoringPasscode":                          "monitoringPasscode",
			"jdbcOverwrite.enabled":                       "true",
			"jdbcOverwrite.jdbcUrl":                       "jdbc:postgresql://" + dependencies.PostgresHost + ":" + dependencies.PostgresPort + "/" + dependencies.PostgresDatabase,
			"jdbcOverwrite.jdbcUsername":                  dependencies.PostgresUsername,
			"jdbcOverwrite.jdbcSecretName":                dependencies.PostgresSecretName,
			"jdbcOverwrite.jdbcSecretPasswordKey":         dependencies.PostgresSecretPasswordKey,
		},
	})
}
