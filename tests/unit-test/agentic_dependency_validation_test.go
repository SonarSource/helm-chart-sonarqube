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
