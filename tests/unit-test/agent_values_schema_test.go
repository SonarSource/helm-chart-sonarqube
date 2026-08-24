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
	}{
		{"agentOrchestrator.replicaCount", map[string]string{"agentOrchestrator.replicaCount": "notanumber"}},
		{"hunterAgent.enabled", map[string]string{"hunterAgent.enabled": "notabool"}},
		{"gvisor.installer.image.digest", map[string]string{"gvisor.installer.image.digest": "true"}},
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range cases {
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
