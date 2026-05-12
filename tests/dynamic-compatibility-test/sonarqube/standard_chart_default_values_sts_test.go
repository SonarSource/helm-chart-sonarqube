package sonarqube

import (
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

func TestStatefulSetWithDefaultValues(t *testing.T) {
	t.Parallel()

	utils.RunChartTest(t, utils.ChartTestSpec{
		Name: "sonarqube",
		Values: map[string]string{
			utils.TESTS_ENABLING_ACTION: "false",
			"edition":                   "enterprise",
			"monitoringPasscode":        "monitoringPasscode",
		},
		ValuesFiles:       []string{"../../../charts/sonarqube/values.yaml"},
		RequireExternalDB: false,
	})
}
