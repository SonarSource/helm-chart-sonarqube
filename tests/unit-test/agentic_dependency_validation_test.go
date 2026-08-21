package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderWithValidation renders a lightweight template with the given SetValues, layered on top of
// the base fields validation.yaml requires regardless of the agentic toggles (monitoringPasscode,
// jwtSecret, jdbcOverwrite), so only the dependency check under test can fail.
func renderWithValidation(t *testing.T, setValues map[string]string) (string, error) {
	t.Helper()
	base := map[string]string{
		"monitoringPasscode":         "test-passcode",
		"applicationNodes.jwtSecret": "test-jwt-secret",
		"jdbcOverwrite.jdbcUrl":      "jdbc:postgresql://test-host:5432/testdb",
		"jdbcOverwrite.jdbcUsername": "test-user",
		"jdbcOverwrite.jdbcPassword": "test-password",
	}
	for k, v := range setValues {
		base[k] = v
	}
	opts := &helm.Options{Logger: logger.Discard, SetValues: base}
	return helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/agentic-orchestrator.yaml"})
}

// hunterAgent.enabled=true requires orchestrator.enabled=true (SONAR-31689).
func TestHunterAgentRequiresOrchestrator(t *testing.T) {
	_, err := renderWithValidation(t, map[string]string{
		"hunterAgent.enabled": "true",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hunterAgent.enabled is true but orchestrator.enabled is not true")
}

// remediationAgent.enabled=true requires orchestrator.enabled=true (SONAR-31689).
func TestRemediationAgentRequiresOrchestrator(t *testing.T) {
	_, err := renderWithValidation(t, map[string]string{
		"remediationAgent.enabled": "true",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remediationAgent.enabled is true but orchestrator.enabled is not true")
}

// remediationAgent.enabled=true also requires vortexAnalysis.enabled=true, checked independently of
// the orchestrator dependency above (SONAR-31689).
func TestRemediationAgentRequiresVortexAnalysis(t *testing.T) {
	_, err := renderWithValidation(t, map[string]string{
		"orchestrator.enabled":          "true",
		"orchestrator.image.repository": "example.com/agentic/orchestrator",
		"remediationAgent.enabled":      "true",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remediationAgent.enabled is true but vortexAnalysis.enabled is not true")
}

// orchestratorCoreDbBase covers everything orchestrator.enabled needs besides the CORE DB
// settings under test, so only the derivation logic can fail.
func orchestratorCoreDbBase() map[string]string {
	return map[string]string{
		"orchestrator.enabled":          "true",
		"orchestrator.image.repository": "example.com/agentic/orchestrator",
		"orchestrator.storage.bucket":   "agentic-jobs",
	}
}

// With no orchestrator.coreDb set at all, the endpoint and name are derived from
// jdbcOverwrite.jdbcUrl.
func TestOrchestratorCoreDbDerivedFromJdbcOverwrite(t *testing.T) {
	output, err := renderWithValidation(t, orchestratorCoreDbBase())
	require.NoError(t, err)
	assert.Contains(t, output, `value: "test-host:5432"`)
	assert.Contains(t, output, `value: "testdb"`)
}

// jdbcOverwrite.jdbcUrl with no database path segment can't yield a name, and
// orchestrator.coreDb.name is not set either, so the render must fail rather than deploy with an
// empty CORE_DB_NAME.
func TestOrchestratorRequiresCoreDbNameWhenNotDerivable(t *testing.T) {
	values := orchestratorCoreDbBase()
	values["jdbcOverwrite.jdbcUrl"] = "jdbc:postgresql://test-host:5432"
	_, err := renderWithValidation(t, values)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "the CORE DB name could not be derived from jdbcOverwrite.jdbcUrl and orchestrator.coreDb.name is not set")
}

// Explicit orchestrator.coreDb.endpoint/name take precedence and let the render succeed even when
// jdbcOverwrite.jdbcUrl alone wouldn't be derivable.
func TestOrchestratorCoreDbExplicitOverridesTakePrecedence(t *testing.T) {
	values := orchestratorCoreDbBase()
	values["jdbcOverwrite.jdbcUrl"] = "jdbc:postgresql://test-host:5432"
	values["orchestrator.coreDb.endpoint"] = "explicit-host:5432"
	values["orchestrator.coreDb.name"] = "explicitdb"
	output, err := renderWithValidation(t, values)
	require.NoError(t, err)
	assert.Contains(t, output, `value: "explicit-host:5432"`)
	assert.Contains(t, output, `value: "explicitdb"`)
}

// Only one of orchestrator.storage.accessKey / secretKey set must fail rather than deploy with a
// silently empty credential.
func TestOrchestratorRequiresBothStorageCredentialsOrNeither(t *testing.T) {
	values := orchestratorCoreDbBase()
	values["orchestrator.storage.accessKey"] = "only-access-key"
	_, err := renderWithValidation(t, values)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one of orchestrator.storage.accessKey / orchestrator.storage.secretKey is set")
}
