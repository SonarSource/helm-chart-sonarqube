package sonarqubedce

import (
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

func TestAgentRuntimeClassCheck(t *testing.T) {
	values := utils.AgentRuntimeClassValues()
	values["applicationNodes.jwtSecret"] = "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo="
	values["jdbcOverwrite.jdbcUrl"] = "jdbc:postgresql://test-host:5432/testdb"
	values["jdbcOverwrite.jdbcUsername"] = "test-user"
	values["jdbcOverwrite.jdbcPassword"] = "test-password"

	utils.RunAgentRuntimeClassCheckTest(t, utils.AgentRuntimeClassSpec{
		ChartName: "sonarqube-dce",
		ChartPath: dceChartPath(),
		SetValues: values,
	})
}
