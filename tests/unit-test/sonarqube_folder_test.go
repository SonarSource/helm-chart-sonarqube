package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func folderOptions(chart agentChart) map[string]string {
	values := map[string]string{
		"monitoringPasscode":                  "test-passcode",
		"jdbcOverwrite.enabled":               "true",
		"jdbcOverwrite.jdbcUrl":               "jdbc:oracle:thin:@test-db:1521/test",
		"jdbcOverwrite.jdbcUsername":          "test-user",
		"jdbcOverwrite.jdbcPassword":          "test-password",
		"jdbcOverwrite.oracleJdbcDriver.url":  "https://example.invalid/driver.jar",
		"caCerts.enabled":                     "true",
		"caCerts.secret":                      "test-certificates",
		"sonarSecretKey":                      "test-sonar-key",
		"hunterAgent.enabled":                 "true",
		"agentOrchestrator.enabled":           "true",
		"agenticSigningSecret.existingSecret": "test-signing-key",
	}
	prefix := ""
	if chart.name == "sonarqube-dce" {
		prefix = "applicationNodes."
		values["applicationNodes.jwtSecret"] = "test-jwt-secret"
		values["searchNodes.persistence.enabled"] = "true"
		values["searchNodes.searchAuthentication.enabled"] = "true"
		values["nodeEncryption.enabled"] = "true"
	} else {
		values["edition"] = "enterprise"
		values["persistence.enabled"] = "true"
	}
	values[prefix+"plugins.install[0]"] = "https://example.invalid/plugin.jar"
	values[prefix+"prometheusExporter.enabled"] = "true"
	return values
}

func renderFolder(t *testing.T, chart agentChart, values map[string]string, file string) string {
	t.Helper()
	opts := &helm.Options{Logger: logger.Discard, SetValues: values}
	if file != "" {
		opts.ValuesFiles = []string{file}
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, nil)
	require.NoError(t, err)
	return output
}

func assertFolderConsumers(t *testing.T, output, folder string, dce bool) {
	t.Helper()
	for _, suffix := range []string{
		"/data", "/temp", "/logs", "/certs", "/extensions/plugins",
		"/extensions/jdbc-driver/oracle", "/secret/", "/agentic-keys", "/agentic-secret",
	} {
		assert.Contains(t, output, "mountPath: "+folder+suffix)
	}
	for _, text := range []string{
		"cd " + folder + "/extensions/plugins",
		"cd " + folder + "/extensions/jdbc-driver/oracle",
		"sonar.secretKeyPath=" + folder + "/secret/sonar-secret.txt",
		"-javaagent:" + folder + "/data/jmx_prometheus_javaagent.jar=",
		folder + "/conf/prometheus-config.yaml",
		folder + "/conf/prometheus-ce-config.yaml",
		"-Djavax.net.ssl.trustStore=" + folder + "/certs/cacerts",
		" " + folder + "/data\n",
	} {
		assert.Contains(t, output, text)
	}
	if dce {
		assert.Contains(t, output, folder+"/es-secret/elastic-stack-ca.p12")
		assert.Contains(t, output, "mountPath: "+folder+"/conf/sonar.properties")
	} else {
		assert.Contains(t, output, "mountPath: "+folder+"/conf/")
	}
}

func TestSonarqubeFolder(t *testing.T) {
	for _, chart := range agentCharts {
		workloads := []string{"StatefulSet"}
		if chart.name == "sonarqube" {
			workloads = append(workloads, "Deployment")
		}
		for _, workload := range workloads {
			t.Run(chart.name+"/"+workload, func(t *testing.T) {
				testFolderWorkload(t, chart, workload)
			})
		}
	}
}

func testFolderWorkload(t *testing.T, chart agentChart, workload string) {
	t.Helper()
	values := folderOptions(chart)
	if chart.name == "sonarqube" {
		values["deploymentType"] = workload
	}
	baseline := renderFolder(t, chart, values, "")
	assertFolderConsumers(t, baseline, "/opt/sonarqube", chart.name == "sonarqube-dce")
	for _, tc := range []struct{ name, value, yaml, folder string }{
		{"default", "/opt/sonarqube", "", "/opt/sonarqube"},
		{"empty", "", "", "/opt/sonarqube"},
		{"null", "null", "", "/opt/sonarqube"},
		{"omitted", "", "{}\n", "/opt/sonarqube"},
		{"yaml-null", "", "sonarqubeFolder: null\n", "/opt/sonarqube"},
		{"yaml-empty", "", "sonarqubeFolder: \"\"\n", "/opt/sonarqube"},
		{"custom", "/custom/sonarqube", "", "/custom/sonarqube"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := ""
			delete(values, "sonarqubeFolder")
			if tc.yaml != "" {
				file = filepath.Join(t.TempDir(), "values.yaml")
				require.NoError(t, os.WriteFile(file, []byte(tc.yaml), 0600))
			} else {
				values["sonarqubeFolder"] = tc.value
			}
			output := renderFolder(t, chart, values, file)
			assertFolderConsumers(t, output, tc.folder, chart.name == "sonarqube-dce")
			// ConfigMap path changes also change the pod restart checksums.
			checksums := regexp.MustCompile(`(?m)^.*checksum/.*\n`)
			normalized := strings.ReplaceAll(output, tc.folder, "/opt/sonarqube")
			assert.Equal(t, checksums.ReplaceAllString(baseline, ""), checksums.ReplaceAllString(normalized, ""))
		})
	}
}
