package sonarqube

import (
	"path/filepath"
	"runtime"
	"testing"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

func TestAgentRuntimeClassCheck(t *testing.T) {
	values := utils.AgentRuntimeClassValues()
	// The agentic pack is Community-Build-or-Server only on this chart, and the single-pod
	// deployment needs an external database declared before anything renders.
	values["community.enabled"] = "true"
	values["jdbcOverwrite.enabled"] = "true"
	values["jdbcOverwrite.jdbcUrl"] = "jdbc:postgresql://test-host:5432/testdb"
	values["jdbcOverwrite.jdbcUsername"] = "test-user"
	values["jdbcOverwrite.jdbcPassword"] = "test-password"

	utils.RunAgentRuntimeClassCheckTest(t, utils.AgentRuntimeClassSpec{
		ChartName: "sonarqube",
		ChartPath: chartPath(),
		SetValues: values,
	})
}

func chartPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "charts", "sonarqube")
}
