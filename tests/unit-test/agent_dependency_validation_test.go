package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// agentValidationBase returns everything validation.yaml requires regardless of the agent
// toggles (monitoringPasscode, edition/community, jdbcOverwrite), so only the dependency check
// under test can fail. It differs per chart: sonarqube-dce has no edition gate and an
// unconditional jdbcOverwrite, while sonarqube requires community.enabled (or edition) and
// jdbcOverwrite.enabled.
func agentValidationBase(chart agentChart) map[string]string {
	base := map[string]string{
		"monitoringPasscode":         "test-passcode",
		"jdbcOverwrite.jdbcUrl":      "jdbc:postgresql://test-host:5432/testdb",
		"jdbcOverwrite.jdbcUsername": "test-user",
		"jdbcOverwrite.jdbcPassword": "test-password",
	}
	if chart.name == "sonarqube-dce" {
		base["applicationNodes.jwtSecret"] = "test-jwt-secret"
	} else {
		base["community.enabled"] = "true"
		base["jdbcOverwrite.enabled"] = "true"
	}
	return base
}

// renderWithValidation renders a lightweight template with the given SetValues, layered on top of
// the chart's validation base, so only the dependency check under test can fail.
func renderWithValidation(t *testing.T, chart agentChart, setValues map[string]string) (string, error) {
	t.Helper()
	base := agentValidationBase(chart)
	for k, v := range setValues {
		base[k] = v
	}
	opts := &helm.Options{Logger: logger.Discard, SetValues: base}
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator.yaml"})
}

// hunterAgent.enabled=true requires agentOrchestrator.enabled=true (SONAR-31689).
func TestHunterAgentRequiresOrchestrator(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderWithValidation(t, chart, map[string]string{
				"hunterAgent.enabled": "true",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "hunterAgent.enabled is true but agentOrchestrator.enabled is not true")
		})
	}
}

// remediationAgent.enabled=true requires agentOrchestrator.enabled=true (SONAR-31689).
func TestRemediationAgentRequiresOrchestrator(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderWithValidation(t, chart, map[string]string{
				"remediationAgent.enabled": "true",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "remediationAgent.enabled is true but agentOrchestrator.enabled is not true")
		})
	}
}

// remediationAgent.enabled=true also requires vortex.enabled=true, checked independently of
// the orchestrator dependency above (SONAR-31689).
func TestRemediationAgentRequiresVortex(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderWithValidation(t, chart, map[string]string{
				"agentOrchestrator.enabled":          "true",
				"agentOrchestrator.image.repository": "example.com/agent-orchestrator",
				"remediationAgent.enabled":           "true",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "remediationAgent.enabled is true but vortex.enabled is not true")
		})
	}
}

// orchestratorCoreDbBase covers everything agentOrchestrator.enabled needs besides the CORE DB
// settings under test, so only the derivation logic can fail.
func orchestratorCoreDbBase() map[string]string {
	return map[string]string{
		"agentOrchestrator.enabled":          "true",
		"agentOrchestrator.image.repository": "example.com/agent-orchestrator",
		"agentOrchestrator.storage.bucket":   "agent-jobs",
	}
}

// With no agentOrchestrator.coreDb set at all, the endpoint and name are derived from
// jdbcOverwrite.jdbcUrl.
func TestOrchestratorCoreDbDerivedFromJdbcOverwrite(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderWithValidation(t, chart, orchestratorCoreDbBase())
			require.NoError(t, err)
			assert.Contains(t, output, `value: "test-host:5432"`)
			assert.Contains(t, output, `value: "testdb"`)
		})
	}
}

// jdbcOverwrite.jdbcUrl with no database path segment can't yield a name, and
// agentOrchestrator.coreDb.name is not set either, so the render must fail rather than deploy with an
// empty CORE_DB_NAME.
func TestOrchestratorRequiresCoreDbNameWhenNotDerivable(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			values := orchestratorCoreDbBase()
			values["jdbcOverwrite.jdbcUrl"] = "jdbc:postgresql://test-host:5432"
			_, err := renderWithValidation(t, chart, values)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "the CORE DB name could not be derived from jdbcOverwrite.jdbcUrl and agentOrchestrator.coreDb.name is not set")
		})
	}
}

// Explicit agentOrchestrator.coreDb.endpoint/name take precedence and let the render succeed even when
// jdbcOverwrite.jdbcUrl alone wouldn't be derivable.
func TestOrchestratorCoreDbExplicitOverridesTakePrecedence(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			values := orchestratorCoreDbBase()
			values["jdbcOverwrite.jdbcUrl"] = "jdbc:postgresql://test-host:5432"
			values["agentOrchestrator.coreDb.endpoint"] = "explicit-host:5432"
			values["agentOrchestrator.coreDb.name"] = "explicitdb"
			output, err := renderWithValidation(t, chart, values)
			require.NoError(t, err)
			assert.Contains(t, output, `value: "explicit-host:5432"`)
			assert.Contains(t, output, `value: "explicitdb"`)
		})
	}
}

// Only one of agentOrchestrator.storage.accessKey / secretKey set must fail rather than deploy with a
// silently empty credential.
func TestOrchestratorRequiresBothStorageCredentialsOrNeither(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			values := orchestratorCoreDbBase()
			values["agentOrchestrator.storage.accessKey"] = "only-access-key"
			_, err := renderWithValidation(t, chart, values)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "only one of agentOrchestrator.storage.accessKey / agentOrchestrator.storage.secretKey is set")
		})
	}
}

// agentOrchestrator.storage.bucket is required for the default S3 type (an unset bucket must not
// silently fall back to StoragePropertiesParser's "agentic-jobs" placeholder), but meaningless -
// and so not required - for a file-based backend that hands the runtime a direct file:// path
// instead; that backend requires filesystem.baseDir instead, for the same reason: an unset baseDir
// must not silently fall back to each service's own default directory, which would break the
// shared mount the orchestrator and runtimes rely on (SONAR-31980).
func TestOrchestratorStorageBucketRequiredUnlessFileBased(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("S3 without a bucket fails", func(t *testing.T) {
				values := orchestratorCoreDbBase()
				values["agentOrchestrator.storage.bucket"] = ""
				_, err := renderWithValidation(t, chart, values)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "agentOrchestrator.storage.bucket is not set")
			})

			for _, storageType := range []string{"FILESYSTEM", "NFS"} {
				t.Run(storageType+" without a bucket but with a baseDir succeeds", func(t *testing.T) {
					values := orchestratorCoreDbBase()
					values["agentOrchestrator.storage.bucket"] = ""
					values["agentOrchestrator.storage.type"] = storageType
					values["agentOrchestrator.storage.filesystem.baseDir"] = "/agentic-storage"
					_, err := renderWithValidation(t, chart, values)
					require.NoError(t, err)
				})

				t.Run(storageType+" without a baseDir fails", func(t *testing.T) {
					values := orchestratorCoreDbBase()
					values["agentOrchestrator.storage.bucket"] = ""
					values["agentOrchestrator.storage.type"] = storageType
					_, err := renderWithValidation(t, chart, values)
					require.Error(t, err)
					assert.Contains(t, err.Error(), "agentOrchestrator.storage.filesystem.baseDir is not set")
				})
			}
		})
	}
}

// The orchestrator shares SonarQube's database, so it cannot run against the embedded one that
// jdbcOverwrite.enabled=false selects - not even with an explicit agentOrchestrator.coreDb, since
// the CORE_DB_USERNAME/PASSWORD defaults still come from the (then empty) jdbcOverwrite helpers.
// Specific to this chart: sonarqube-dce has no jdbcOverwrite.enabled gate - jdbcOverwrite is
// always active there.
func TestAgentOrchestratorRequiresJdbcOverwrite(t *testing.T) {
	sonarqube := agentCharts[1]

	baseValues := func() map[string]string {
		values := agentValidationBase(sonarqube)
		values["jdbcOverwrite.enabled"] = "false"
		values["agentOrchestrator.enabled"] = "true"
		values["agentOrchestrator.image.repository"] = "example.com/agent-orchestrator"
		values["agentOrchestrator.storage.bucket"] = "agent-jobs"
		return values
	}

	t.Run("fails without jdbcOverwrite", func(t *testing.T) {
		opts := &helm.Options{Logger: logger.Discard, SetValues: baseValues()}
		_, err := helm.RenderTemplateE(t, opts, sonarqube.path, sonarqube.release, []string{"templates/agent-orchestrator.yaml"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agentOrchestrator.enabled is true but jdbcOverwrite is not enabled")
	})

	t.Run("fails without jdbcOverwrite even with explicit coreDb", func(t *testing.T) {
		values := baseValues()
		values["agentOrchestrator.coreDb.endpoint"] = "explicit-host:5432"
		values["agentOrchestrator.coreDb.name"] = "explicitdb"
		opts := &helm.Options{Logger: logger.Discard, SetValues: values}
		_, err := helm.RenderTemplateE(t, opts, sonarqube.path, sonarqube.release, []string{"templates/agent-orchestrator.yaml"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agentOrchestrator.enabled is true but jdbcOverwrite is not enabled")
	})

	t.Run("succeeds with jdbcOverwrite.enabled=true", func(t *testing.T) {
		values := baseValues()
		values["jdbcOverwrite.enabled"] = "true"
		opts := &helm.Options{Logger: logger.Discard, SetValues: values}
		_, err := helm.RenderTemplateE(t, opts, sonarqube.path, sonarqube.release, []string{"templates/agent-orchestrator.yaml"})
		require.NoError(t, err)
	})
}
