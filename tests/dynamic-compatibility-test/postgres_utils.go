package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

// External Postgres install constants. These match the values hardcoded in
// sonarqube-dce/postgres-values.yaml and the JDBC config passed to the DCE chart.
const (
	postgresReleaseName = "external-postgres"
	postgresChart       = "oci://registry-1.docker.io/bitnamicharts/postgresql"
	postgresChartVer    = "18.2.3"
	postgresValuesFile  = "sonarqube-dce/postgres-values.yaml"

	// Service hostname produced by the Bitnami chart with the release name above.
	// Resolves within the same namespace via cluster DNS.
	PostgresHost     = "external-postgres-postgresql"
	PostgresPort     = "5432"
	PostgresDatabase = "sonar"
	PostgresUsername = "sonar"

	// Reference the Bitnami-generated secret instead of passing the password
	// inline — the deprecated jdbcOverwrite.jdbcPassword field should not be
	// used. The Bitnami chart names the secret <release>-postgresql and stores
	// the application user's password under the key "password".
	PostgresSecretName        = "external-postgres-postgresql"
	PostgresSecretPasswordKey = "password"
)

// setupExternalDB installs a Bitnami PostgreSQL release into the test
// namespace and waits for the primary pod to be ready. The release is torn
// down automatically when the namespace is deleted in the test teardown.
func setupExternalDB(t *testing.T, kubectlOptions *k8s.KubectlOptions) {
	options := &helm.Options{
		ValuesFiles:    []string{postgresValuesFile},
		KubectlOptions: kubectlOptions,
		ExtraArgs: map[string][]string{
			"install": {"--version", postgresChartVer, "--wait", "--timeout", "5m"},
		},
	}
	helm.Install(t, options, postgresChart, postgresReleaseName)

	// Belt-and-braces: --wait above already blocks on readiness, but the chart
	// produces a StatefulSet so we double-check the primary pod directly.
	k8s.RunKubectl(t, kubectlOptions,
		"wait", "--for=condition=ready", "pod",
		"-l", "app.kubernetes.io/name=postgresql",
		"--timeout=300s",
	)
}
