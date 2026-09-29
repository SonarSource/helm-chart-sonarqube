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

// remediationAgent.enabled=true still fails when vortexAnalysis.enabled is explicitly set to
// false, checked independently of the orchestrator dependency above (SONAR-31689).
func TestRemediationAgentRequiresVortex(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderWithValidation(t, chart, map[string]string{
				"agentOrchestrator.enabled":          "true",
				"agentOrchestrator.image.repository": "example.com/agent-orchestrator",
				"remediationAgent.enabled":           "true",
				"remediationAgent.image.repository":  "example.com/remediation-agent",
				"remediationAgent.image.tag":         "1",
				"vortexAnalysis.enabled":             "false",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "remediationAgent.enabled is true but vortexAnalysis.enabled is explicitly set to false")
		})
	}
}

// remediationAgent.enabled=true defaults vortexAnalysis.enabled to true when left unset, since
// remediation is not useful without it - so a render with vortex config supplied (but
// vortexAnalysis.enabled itself left unset) must succeed (SONAR-31689).
func TestRemediationAgentDefaultsVortexOn(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderWithValidation(t, chart, map[string]string{
				"agentOrchestrator.enabled":           "true",
				"agentOrchestrator.image.repository":  "example.com/agent-orchestrator",
				"remediationAgent.enabled":            "true",
				"remediationAgent.image.repository":   "example.com/remediation-agent",
				"remediationAgent.image.tag":          "1",
				"agenticSigningSecret.existingSecret": "agentic-instance-secret",
				"vortexAnalysis.image.repository":     "example.com/vortex",
				"vortexAnalysis.image.tag":            "1",
				"vortexAnalysis.storage.bucket":       "vortex-artifacts",
				"vortexAnalysis.storage.region":       "eu-west-1",
			})
			require.NoError(t, err)
		})
	}
}

// orchestratorCoreDbBase covers everything agentOrchestrator.enabled needs besides the CORE DB
// settings under test, so only the derivation logic can fail. hunterAgent.enabled satisfies the
// orchestrator's own requirement of having an agent runtime to dispatch jobs to, which in turn
// requires agenticSigningSecret.existingSecret to be set.
func orchestratorCoreDbBase() map[string]string {
	return map[string]string{
		"agentOrchestrator.enabled":           "true",
		"agentOrchestrator.image.repository":  "example.com/agent-orchestrator",
		"agentOrchestrator.storage.bucket":    "agent-jobs",
		"hunterAgent.enabled":                 "true",
		"hunterAgent.image.repository":        "example.com/hunter-agent",
		"hunterAgent.image.tag":               "1",
		"agenticSigningSecret.existingSecret": "test-agentic-instance-secret",
	}
}

// With no agentOrchestrator.coreDb set at all, CORE_DB_JDBC_URL is jdbcOverwrite.jdbcUrl as is, and
// for PostgreSQL the legacy endpoint/name pair older orchestrator images read is derived from it
// too (SONAR-32565).
func TestOrchestratorCoreDbDerivedFromJdbcOverwrite(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			values := orchestratorCoreDbBase()
			values["jdbcOverwrite.jdbcUrl"] = "jdbc:postgresql://test-host:5432/testdb?sslmode=require"
			output, err := renderWithValidation(t, chart, values)
			require.NoError(t, err)
			assert.Contains(t, output, "- name: CORE_DB_JDBC_URL\n              value: \"jdbc:postgresql://test-host:5432/testdb?sslmode=require\"")
			assert.Contains(t, output, "- name: CORE_DB_READ_WRITE_ENDPOINT\n              value: \"test-host:5432\"")
			assert.Contains(t, output, "- name: CORE_DB_NAME\n              value: \"testdb\"")
		})
	}
}

// Oracle and Microsoft SQL Server URLs are passed through as CORE_DB_JDBC_URL, without the
// PostgreSQL-only endpoint/name pair (SONAR-32565).
func TestOrchestratorCoreDbNonPostgresqlUrlPassedThrough(t *testing.T) {
	urls := map[string]string{
		"oracle":    "jdbc:oracle:thin:@//oracle-host:1521/FREEPDB1",
		"sqlserver": "jdbc:sqlserver://mssql-host:1433;databaseName=sonar;encrypt=true",
	}
	for _, chart := range agentCharts {
		for vendor, url := range urls {
			t.Run(chart.name+"/"+vendor, func(t *testing.T) {
				values := orchestratorCoreDbBase()
				values["jdbcOverwrite.jdbcUrl"] = url
				output, err := renderWithValidation(t, chart, values)
				require.NoError(t, err)
				assert.Contains(t, output, "- name: CORE_DB_JDBC_URL\n              value: \""+url+"\"")
				assert.NotContains(t, output, "CORE_DB_READ_WRITE_ENDPOINT")
				assert.NotContains(t, output, "CORE_DB_NAME")
			})
		}
	}
}

// agentOrchestrator.coreDb.jdbcUrl overrides jdbcOverwrite.jdbcUrl, and wins over the legacy
// endpoint/name fields.
func TestOrchestratorCoreDbJdbcUrlOverride(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			values := orchestratorCoreDbBase()
			values["agentOrchestrator.coreDb.jdbcUrl"] = "jdbc:sqlserver://explicit-host:1433;databaseName=explicitdb"
			values["agentOrchestrator.coreDb.endpoint"] = "ignored-host:5432"
			output, err := renderWithValidation(t, chart, values)
			require.NoError(t, err)
			assert.Contains(t, output, "- name: CORE_DB_JDBC_URL\n              value: \"jdbc:sqlserver://explicit-host:1433;databaseName=explicitdb\"")
			assert.NotContains(t, output, "test-host")
			assert.NotContains(t, output, "ignored-host")
		})
	}
}

// The legacy, PostgreSQL-only agentOrchestrator.coreDb.endpoint/name still take precedence over
// what jdbcOverwrite.jdbcUrl holds, each falling back to it on its own.
func TestOrchestratorCoreDbExplicitOverridesTakePrecedence(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("both", func(t *testing.T) {
				values := orchestratorCoreDbBase()
				values["agentOrchestrator.coreDb.endpoint"] = "explicit-host:5432"
				values["agentOrchestrator.coreDb.name"] = "explicitdb"
				output, err := renderWithValidation(t, chart, values)
				require.NoError(t, err)
				assert.Contains(t, output, "- name: CORE_DB_JDBC_URL\n              value: \"jdbc:postgresql://explicit-host:5432/explicitdb\"")
				assert.Contains(t, output, "- name: CORE_DB_READ_WRITE_ENDPOINT\n              value: \"explicit-host:5432\"")
				assert.Contains(t, output, "- name: CORE_DB_NAME\n              value: \"explicitdb\"")
			})

			t.Run("name only", func(t *testing.T) {
				values := orchestratorCoreDbBase()
				values["agentOrchestrator.coreDb.name"] = "explicitdb"
				output, err := renderWithValidation(t, chart, values)
				require.NoError(t, err)
				assert.Contains(t, output, "- name: CORE_DB_JDBC_URL\n              value: \"jdbc:postgresql://test-host:5432/explicitdb\"")
			})
		})
	}
}

// A CORE DB URL for a database the orchestrator doesn't support must fail the render.
func TestOrchestratorRejectsUnsupportedCoreDbUrl(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("from jdbcOverwrite", func(t *testing.T) {
				values := orchestratorCoreDbBase()
				values["jdbcOverwrite.jdbcUrl"] = "jdbc:h2:tcp://test-host/testdb"
				_, err := renderWithValidation(t, chart, values)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "the CORE DB JDBC URL 'jdbc:h2:tcp://test-host/testdb'")
			})

			t.Run("from coreDb.jdbcUrl", func(t *testing.T) {
				values := orchestratorCoreDbBase()
				values["agentOrchestrator.coreDb.jdbcUrl"] = "postgresql://test-host:5432/testdb"
				_, err := renderWithValidation(t, chart, values)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "is not a PostgreSQL, Oracle or Microsoft SQL Server one")
			})
		})
	}
}

// The legacy CORE_DB_READ_WRITE_ENDPOINT/CORE_DB_NAME pair must be derivable from a PostgreSQL URL,
// as orchestrator images before multi-database support only read it.
func TestOrchestratorRejectsIncompletePostgresqlCoreDbUrl(t *testing.T) {
	cases := map[string]map[string]string{
		"no database name":    {"jdbcOverwrite.jdbcUrl": "jdbc:postgresql://test-host:5432"},
		"empty database name": {"agentOrchestrator.coreDb.jdbcUrl": "jdbc:postgresql://test-host:5432/"},
		"endpoint only":       {"jdbcOverwrite.jdbcUrl": "jdbc:postgresql://", "agentOrchestrator.coreDb.endpoint": "test-host:5432"},
		"database name only":  {"jdbcOverwrite.jdbcUrl": "jdbc:postgresql://", "agentOrchestrator.coreDb.name": "testdb"},
	}
	for _, chart := range agentCharts {
		for name, extraValues := range cases {
			t.Run(chart.name+"/"+name, func(t *testing.T) {
				values := orchestratorCoreDbBase()
				for k, v := range extraValues {
					values[k] = v
				}
				_, err := renderWithValidation(t, chart, values)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "the CORE DB host or database name could not be parsed from the PostgreSQL JDBC URL")
			})
		}
	}
}

// renderOracleCoreDb renders the orchestrator against an Oracle jdbcOverwrite.jdbcUrl, with the
// given extra (driver-related) values layered on top.
func renderOracleCoreDb(t *testing.T, chart agentChart, extraValues map[string]string) string {
	t.Helper()
	values := orchestratorCoreDbBase()
	values["jdbcOverwrite.jdbcUrl"] = "jdbc:oracle:thin:@//oracle-host:1521/FREEPDB1"
	for k, v := range extraValues {
		values[k] = v
	}
	output, err := renderWithValidation(t, chart, values)
	require.NoError(t, err)
	return output
}

// jdbcOverwrite.oracleJdbcDriver.url installs the Oracle driver in the orchestrator pod the same way
// as in SonarQube's, into an emptyDir the orchestrator reads it from (SONAR-32565).
func TestOrchestratorInstallsOracleJdbcDriver(t *testing.T) {
	const driverURL = "https://repo.example.com/ojdbc11.jar"
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("absent without a driver URL", func(t *testing.T) {
				output := renderOracleCoreDb(t, chart, nil)
				assert.NotContains(t, output, "oracle-jdbc-driver")
				assert.NotContains(t, output, "CORE_DB_ORACLE_DRIVER_DIR")
			})

			t.Run("present with a driver URL", func(t *testing.T) {
				output := renderOracleCoreDb(t, chart, map[string]string{
					"jdbcOverwrite.oracleJdbcDriver.url": driverURL,
				})
				assert.Contains(t, output, "- name: install-oracle-jdbc-driver\n          image:")
				assert.Contains(t, output, "sh -e /tmp/scripts/install_oracle_jdbc_driver.sh")
				assert.Contains(t, output, "- name: oracle-jdbc-driver\n              mountPath: /opt/sonarqube/extensions/jdbc-driver/oracle")
				assert.Contains(t, output, "- name: oracle-jdbc-driver\n              mountPath: /app/jdbc-driver/oracle\n              readOnly: true")
				assert.Contains(t, output, "- name: oracle-jdbc-driver\n          emptyDir: {}")
				assert.Contains(t, output, "name: "+chart.release+"-"+chart.name+"-install-oracle-jdbc-driver")
				assert.Contains(t, output, "- name: CORE_DB_ORACLE_DRIVER_DIR\n              value: /app/jdbc-driver/oracle")
				assert.NotContains(t, output, "oracle-jdbc-driver-netrc-file")
				assert.NotContains(t, output, "ca-bundle")
			})

			t.Run("with netrc credentials and CA certificates", func(t *testing.T) {
				output := renderOracleCoreDb(t, chart, map[string]string{
					"jdbcOverwrite.oracleJdbcDriver.url":        driverURL,
					"jdbcOverwrite.oracleJdbcDriver.netrcCreds": "oracle-netrc",
					"caCerts.enabled":                           "true",
					"caCerts.secret":                            "test-ca-certs",
				})
				assert.Contains(t, output, "- name: oracle-jdbc-driver-netrc-file\n              mountPath: /root")
				assert.Contains(t, output, "- name: oracle-jdbc-driver-netrc-file\n          secret:\n            secretName: oracle-netrc")
				assert.Contains(t, output, "> /tmp/certs/ca-bundle.pem")
				assert.Contains(t, output, "- name: ca-certs\n              mountPath: /tmp/secrets/ca-certs")
				assert.Contains(t, output, "- name: ca-certs\n          secret:\n            secretName: test-ca-certs")
				assert.Contains(t, output, "- name: ca-bundle\n              mountPath: /tmp/certs")
			})
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
		values["hunterAgent.enabled"] = "true"
		values["hunterAgent.image.repository"] = "example.com/hunter-agent"
		values["hunterAgent.image.tag"] = "1"
		values["agenticSigningSecret.existingSecret"] = "test-agentic-instance-secret"
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
