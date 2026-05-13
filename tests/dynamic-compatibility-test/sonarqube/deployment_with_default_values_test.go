package sonarqube

import (
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

func TestDeploymentWithDefaultValues(t *testing.T) {
	t.Parallel()

	utils.RunChartTest(t, utils.ChartTestSpec{
		Name: "sonarqube",
		Values: map[string]string{
			utils.TESTS_ENABLING_ACTION:  "false",
			"deploymentType":             "Deployment",
			"prometheusExporter.enabled": "false", // only for deployment
			"edition":                    "enterprise",
			"monitoringPasscode":         "monitoringPasscode",
		},
		RequireExternalDB: false,
	})
}
