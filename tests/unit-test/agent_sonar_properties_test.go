package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// The agent URLs are wired twice on purpose: as SONAR_* pod env vars (covered by vortex_test.go and
// agent_orchestrator_test.go) and as real conf/sonar.properties lines. Only the second one reaches
// plain Configuration.get() consumers - SonarQube honours a SONAR_* override for a property that
// already has a properties-file line, so the env var alone is silently ignored (SONAR-31416).
// These tests cover the properties-file half, which is the part that actually fixes the bug.

const (
	hunterOrchestratorURLProperty      = "sonar.hunteragent.orchestrator.url"
	remediationOrchestratorURLProperty = "sonar.remediationagent.orchestrator.url"
	vortexAnalysisURLProperty          = "sonar.vortex.analysis.url"
	agenticSigningSecretFileProperty   = "sonar.agentic.signing.secretFile"
)

// sonarPropertiesFromConfigMap parses the conf/sonar.properties body of the ConfigMap named
// configMapName out of a rendered templates/config.yaml output into key/value pairs.
func sonarPropertiesFromConfigMap(t *testing.T, output string, configMapName string) map[string]string {
	t.Helper()
	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var cm corev1.ConfigMap
		helm.UnmarshalK8SYaml(t, doc, &cm)
		if cm.Name != configMapName {
			continue
		}
		props := map[string]string{}
		for _, line := range strings.Split(cm.Data["sonar.properties"], "\n") {
			if key, value, found := strings.Cut(strings.TrimSpace(line), "="); found {
				props[key] = value
			}
		}
		return props
	}

	t.Fatalf("templates/config.yaml rendered no ConfigMap named %q", configMapName)
	return nil
}

// appSonarProperties renders templates/config.yaml and parses the conf/sonar.properties body of the
// application nodes' ConfigMap into key/value pairs.
func appSonarProperties(t *testing.T, chart agentChart, opts *helm.Options) map[string]string {
	t.Helper()
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/config.yaml"})
	require.NoError(t, err)
	return sonarPropertiesFromConfigMap(t, output, chart.fullnamePrefix()+chart.appConfigMapSuffix)
}

func agentPropertiesOptions(chart agentChart, fixture string, setValues map[string]string) *helm.Options {
	return &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/" + fixture},
		SetValues:   setValues,
	}
}

// With the orchestrator enabled, both orchestrator URL properties must be real properties-file
// lines pointing at its Service.
func TestOrchestratorUrlsInSonarProperties(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-orchestrator-enabled.yaml", nil))
			url := chart.orchestratorURL()
			assert.Equal(t, url, props[hunterOrchestratorURLProperty])
			assert.Equal(t, url, props[remediationOrchestratorURLProperty])
		})
	}
}

// Same for Vortex, which is toggled independently of the orchestrator.
func TestVortexUrlInSonarProperties(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "vortex-enabled.yaml", nil))
			assert.Equal(t, "http://"+chart.fullnamePrefix()+vortexFullnameSuffix+":8080", props[vortexAnalysisURLProperty])

			// Vortex on its own must not advertise an orchestrator that isn't deployed.
			assert.NotContains(t, props, hunterOrchestratorURLProperty)
			assert.NotContains(t, props, remediationOrchestratorURLProperty)
		})
	}
}

// With every agent off, none of the properties may appear - an existing install's sonar.properties
// is unchanged until a toggle is flipped.
func TestNoAgentUrlsInSonarPropertiesWhenDisabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-all-disabled.yaml", nil))
			for _, key := range []string{hunterOrchestratorURLProperty, remediationOrchestratorURLProperty, vortexAnalysisURLProperty} {
				assert.NotContains(t, props, key)
			}
		})
	}
}

// The generated properties are merged into the user's sonarProperties rather than replacing them,
// and the user's value wins on a collision - so an operator can still point SonarQube at an
// orchestrator the chart didn't deploy.
func TestUserSonarPropertiesOverrideAgentUrls(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-orchestrator-enabled.yaml", map[string]string{
				chart.sonarPropertiesPath + `.sonar\.hunteragent\.orchestrator\.url`: "http://external-orchestrator:9000",
				chart.sonarPropertiesPath + `.sonar\.forceAuthentication`:            "false",
			}))
			assert.Equal(t, "http://external-orchestrator:9000", props[hunterOrchestratorURLProperty])
			assert.Equal(t, "false", props["sonar.forceAuthentication"])
			// Untouched keys still come from the chart.
			assert.Equal(t, chart.orchestratorURL(), props[remediationOrchestratorURLProperty])
		})
	}
}

// The agentic signing secret (.Values.agenticSigningSecretKey) follows the same sonar.properties-file
// pattern as sonarSecretKey (SONAR-31416): a SONAR_* env var override is silently ignored unless the
// property already has a properties-file line, so sonar.agentic.signing.secretFile must be written
// here too rather than only exposed as an env var (SONAR-32028). It must reach every node that reads
// conf/sonar.properties directly - the single ConfigMap in sonarqube, and both the app and search
// ConfigMaps in sonarqube-dce.
func TestAgenticSigningSecretFileInSonarProperties(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("unset renders no property", func(t *testing.T) {
				props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-all-disabled.yaml", nil))
				assert.NotContains(t, props, agenticSigningSecretFileProperty)
			})

			t.Run("set renders the property pointing at the mounted file", func(t *testing.T) {
				props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-all-disabled.yaml", map[string]string{
					"agenticSigningSecretKey": "agentic-signing-secret",
				}))
				assert.Equal(t, "/opt/sonarqube/secret-agentic-signing/sonar-agentic-signing-secret.txt", props[agenticSigningSecretFileProperty])
			})
		})
	}

	t.Run("sonarqube-dce search-config", func(t *testing.T) {
		dce := agentCharts[0]
		require.Equal(t, "sonarqube-dce", dce.name)
		searchConfigMap := dce.fullnamePrefix() + "-search-config"

		t.Run("unset renders no property", func(t *testing.T) {
			opts := agentPropertiesOptions(dce, "agent-all-disabled.yaml", nil)
			output, err := helm.RenderTemplateE(t, opts, dce.path, dce.release, []string{"templates/config.yaml"})
			require.NoError(t, err)
			props := sonarPropertiesFromConfigMap(t, output, searchConfigMap)
			assert.NotContains(t, props, agenticSigningSecretFileProperty)
		})

		t.Run("set renders the property pointing at the mounted file", func(t *testing.T) {
			opts := agentPropertiesOptions(dce, "agent-all-disabled.yaml", map[string]string{
				"agenticSigningSecretKey": "agentic-signing-secret",
			})
			output, err := helm.RenderTemplateE(t, opts, dce.path, dce.release, []string{"templates/config.yaml"})
			require.NoError(t, err)
			props := sonarPropertiesFromConfigMap(t, output, searchConfigMap)
			assert.Equal(t, "/opt/sonarqube/secret-agentic-signing/sonar-agentic-signing-secret.txt", props[agenticSigningSecretFileProperty])
		})
	})
}
