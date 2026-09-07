package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// values.schema.json type-checks the agent pack's values at render time, catching e.g. a
// replicaCount typed as a string before it ever reaches a Deployment spec.
func TestAgentValuesSchemaRejectsWrongTypes(t *testing.T) {
	cases := []struct {
		name string
		set  map[string]string
		// requiresEgressProxy restricts this case to charts with an Agent Egress Proxy - an
		// unknown key like agentEgressProxy.* on a chart without one is silently ignored by the
		// schema (no additionalProperties: false), not rejected, so it wouldn't error there.
		requiresEgressProxy bool
		// requiresAgenticKeys restricts this case to charts that derive agentic signing keys,
		// for the same reason.
		requiresAgenticKeys bool
		// sonarqubeOnly restricts this case to the sonarqube chart - istio.gvisorSidecar has no
		// equivalent block on sonarqube-dce at all, so an unknown key there is silently ignored
		// by the schema (no additionalProperties: false), not rejected.
		sonarqubeOnly bool
	}{
		{name: "agentOrchestrator.replicaCount", set: map[string]string{"agentOrchestrator.replicaCount": "notanumber"}},
		{name: "hunterAgent.enabled", set: map[string]string{"hunterAgent.enabled": "notabool"}},
		{name: "hunterAgent.storage.pathStyle", set: map[string]string{"hunterAgent.storage.pathStyle": "notabool"}},
		{name: "remediationAgent.storage.pathStyle", set: map[string]string{"remediationAgent.storage.pathStyle": "notabool"}},
		{name: "gvisor.installer.image.digest", set: map[string]string{"gvisor.installer.image.digest": "true"}},
		{name: "agentEgressProxy.replicaCount", set: map[string]string{"agentEgressProxy.replicaCount": "notanumber"}, requiresEgressProxy: true},
		{name: "agentKeyDerivation.enabled", set: map[string]string{"agentKeyDerivation.enabled": "notabool"}, requiresAgenticKeys: true},
		{name: "agenticSigningSecret.existingSecret", set: map[string]string{"agenticSigningSecret.existingSecret": "true"}, requiresAgenticKeys: true},
		{name: "istio.gvisorSidecar.meshPort below minimum", set: map[string]string{"istio.gvisorSidecar.meshPort": "80"}, sonarqubeOnly: true},
		{name: "istio.gvisorSidecar.meshPort above maximum", set: map[string]string{"istio.gvisorSidecar.meshPort": "70000"}, sonarqubeOnly: true},
		{name: "istio.gvisorSidecar.meshPort inside Istio's reserved range", set: map[string]string{"istio.gvisorSidecar.meshPort": "15050"}, sonarqubeOnly: true},
		{name: "istio.gvisorSidecar.enabled", set: map[string]string{"istio.gvisorSidecar.enabled": "notabool"}, sonarqubeOnly: true},
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range cases {
				if tc.requiresEgressProxy && !chart.hasEgressProxy {
					continue
				}
				if tc.requiresAgenticKeys && !chart.hasAgenticKeys {
					continue
				}
				if tc.sonarqubeOnly && chart.name != "sonarqube" {
					continue
				}
				t.Run(tc.name, func(t *testing.T) {
					opts := &helm.Options{Logger: logger.Discard, SetValues: tc.set}
					_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator.yaml"})
					require.Error(t, err)
					assert.Contains(t, err.Error(), "don't meet the specifications")
				})
			}
		})
	}
}
