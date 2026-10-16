package sonarqubedce

import (
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
	"github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test/dependencies"
)

func TestDCEWithAllValues(t *testing.T) {
	t.Parallel()

	utils.RunChartTest(t, utils.ChartTestSpec{
		Name: "sonarqube-dce",
		Values: map[string]string{
			utils.TESTS_ENABLING_ACTION:           "false",
			"ApplicationNodes.jwtSecret":          "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=",
			"jdbcOverwrite.jdbcUrl":               "jdbc:postgresql://" + dependencies.PostgresHost + ":" + dependencies.PostgresPort + "/" + dependencies.PostgresDatabase,
			"jdbcOverwrite.jdbcUsername":          dependencies.PostgresUsername,
			"jdbcOverwrite.jdbcSecretName":        dependencies.PostgresSecretName,
			"jdbcOverwrite.jdbcSecretPasswordKey": dependencies.PostgresSecretPasswordKey,
		},
		ValuesFiles:       []string{"all-values.yaml"},
		RequireExternalDB: true,
	})
}
