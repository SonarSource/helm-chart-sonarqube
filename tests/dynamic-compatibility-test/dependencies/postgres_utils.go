package dependencies

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

// valuesFile returns the absolute path of postgres-values.yaml that sits next to this
// Go source file. Using runtime.Caller makes the path independent of the test
// process's working directory — each test package runs from its own dir
// (sonarqube/, sonarqube-dce/, …) and helm resolves --values relative to CWD,
// so a fixed relative path would only work from one caller.
func valuesFile() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "postgres-values.yaml")
}

// External Postgres install constants. These match the Bitnami chart's
// defaults (no auth.* overrides in postgres-values.yaml) and the JDBC config
// passed to the DCE chart.
const (
	postgresReleaseName = "external-postgres"
	postgresChart       = "oci://registry-1.docker.io/bitnamicharts/postgresql"
	postgresChartVer    = "18.2.3"

	// Service hostname produced by the Bitnami chart with the release name above.
	// Resolves within the same namespace via cluster DNS.
	PostgresHost     = "external-postgres-postgresql"
	PostgresPort     = "5432"
	PostgresDatabase = "postgres"
	PostgresUsername = "postgres"

	// Reference the Bitnami-generated secret instead of passing the password
	// inline — the deprecated jdbcOverwrite.jdbcPassword field should not be
	// used. The Bitnami chart names the secret <release>-postgresql and stores
	// the postgres superuser's password under the key "postgres-password".
	PostgresSecretName        = "external-postgres-postgresql"
	PostgresSecretPasswordKey = "postgres-password"
)

// SetupDB installs a Bitnami PostgreSQL release into the test
// namespace and waits for the primary pod to be ready. The release is torn
// down automatically when the namespace is deleted in the test teardown.
func SetupDB(t *testing.T, kubectlOptions *k8s.KubectlOptions) {
	options := &helm.Options{
		ValuesFiles:    []string{valuesFile()},
		KubectlOptions: kubectlOptions,
		ExtraArgs: map[string][]string{
			"install": {"--version", postgresChartVer, "--wait", "--timeout", "5m"},
		},
	}
	helm.Install(t, options, postgresChart, postgresReleaseName)
}
