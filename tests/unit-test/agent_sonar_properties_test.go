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
)

// appSonarProperties renders templates/config.yaml and parses the conf/sonar.properties body of the
// application nodes' ConfigMap into key/value pairs.
func appSonarProperties(t *testing.T, chart agentChart, opts *helm.Options) map[string]string {
	t.Helper()
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/config.yaml"})
	require.NoError(t, err)

	want := chart.fullnamePrefix() + chart.appConfigMapSuffix
	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var cm corev1.ConfigMap
		helm.UnmarshalK8SYaml(t, doc, &cm)
		if cm.Name != want {
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

	t.Fatalf("templates/config.yaml rendered no ConfigMap named %q", want)
	return nil
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
