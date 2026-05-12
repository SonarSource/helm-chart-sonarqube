package sonarqube

import (
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

func TestStatefulSetWithAllValues(t *testing.T) {
	t.Parallel()

	utils.RunChartTest(t, utils.ChartTestSpec{
		Name: "sonarqube",
		Values: map[string]string{
			utils.TESTS_ENABLING_ACTION: "false",
		},
		ValuesFiles:       []string{"all-values.yaml"},
		RequireExternalDB: false,
	})
}
