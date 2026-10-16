package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

// MCP is a standalone service deployed alongside SonarQube, like Vortex, and is independent of
// the orchestrator/hunterAgent/remediationAgent.

func renderMcp(t *testing.T, chart agentChart, fixture string) (string, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/" + fixture},
	}
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/mcp.yaml"})
}

func mcpDeployment(t *testing.T, chart agentChart, fixture string) appsv1.Deployment {
	t.Helper()
	output, err := renderMcp(t, chart, fixture)
	require.NoError(t, err)

	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	require.NotEmpty(t, deployment.Name, "no Deployment rendered by templates/mcp.yaml")
	return deployment
}

// MCP's own scheduling settings take precedence over the chart's global ones.
func TestMcpSchedulingWinsOverGlobal(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			podSpec := mcpDeployment(t, chart, "mcp-global-scheduling.yaml").Spec.Template.Spec

			assert.Equal(t, map[string]string{"mcp": "true"}, podSpec.NodeSelector)

			require.Len(t, podSpec.Tolerations, 1)
			assert.Equal(t, "mcp", podSpec.Tolerations[0].Key)

			require.NotNil(t, podSpec.Affinity)
			require.NotNil(t, podSpec.Affinity.NodeAffinity)
			terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			require.Len(t, terms, 1)
			require.Len(t, terms[0].MatchExpressions, 1)
			assert.Equal(t, "mcp", terms[0].MatchExpressions[0].Key)
		})
	}
}
