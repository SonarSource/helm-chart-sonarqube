package sonarqubedce

import (
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

func TestDceWithDefaultValues(t *testing.T) {
	t.Parallel()

	utils.RunChartTest(t, utils.ChartTestSpec{
		Name: "sonarqube-dce",
		Values: map[string]string{
			utils.TESTS_ENABLING_ACTION:           "false",
			"ApplicationNodes.jwtSecret":          "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=",
			"monitoringPasscode":                  "monitoringPasscode",
			"jdbcOverwrite.enabled":               "true",
			"jdbcOverwrite.jdbcUrl":               "jdbc:postgresql://" + utils.PostgresHost + ":" + utils.PostgresPort + "/" + utils.PostgresDatabase,
			"jdbcOverwrite.jdbcUsername":          utils.PostgresUsername,
			"jdbcOverwrite.jdbcSecretName":        utils.PostgresSecretName,
			"jdbcOverwrite.jdbcSecretPasswordKey": utils.PostgresSecretPasswordKey,
		},
		ValuesFiles:       []string{"../../../charts/sonarqube-dce/values.yaml"},
		RequireExternalDB: true,
	})
}
