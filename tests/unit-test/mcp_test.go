package tests

import (
	"sort"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

// MCP is a standalone service deployed alongside SonarQube, like Vortex, and is independent of
// the orchestrator/hunterAgent/remediationAgent.

func renderMcp(t *testing.T, chart agentChart, fixture string, setValues map[string]string) (string, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/" + fixture},
		SetValues:   setValues,
	}
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/mcp.yaml"})
}

func mcpDeployment(t *testing.T, chart agentChart, fixture string, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	output, err := renderMcp(t, chart, fixture, setValues)
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
			podSpec := mcpDeployment(t, chart, "mcp-global-scheduling.yaml", nil).Spec.Template.Spec

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

// The built-in wait-for-sonarqube init container follows initContainers.resources, like every
// other init container, for clusters that require resources on all of them (SONAR-33186).
func TestMcpWaitForSonarqubeResources(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("unset renders no resources", func(t *testing.T) {
				initContainers := mcpDeployment(t, chart, "mcp-enabled.yaml", nil).Spec.Template.Spec.InitContainers
				waitFor := findInitContainerByName(initContainers, "wait-for-sonarqube")
				require.NotNil(t, waitFor)
				assert.Empty(t, waitFor.Resources.Requests)
				assert.Empty(t, waitFor.Resources.Limits)
			})

			t.Run("set renders initContainers.resources", func(t *testing.T) {
				initContainers := mcpDeployment(t, chart, "mcp-enabled.yaml", initContainersResources).Spec.Template.Spec.InitContainers
				waitFor := findInitContainerByName(initContainers, "wait-for-sonarqube")
				require.NotNil(t, waitFor)
				assertInitContainersResources(t, waitFor.Resources)
			})
		})
	}
}

// mcp.podLabels lands on the pod template only; the selector stays fixed so upgrades keep working.
func TestMcpPodLabels(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("unset renders only the selector labels", func(t *testing.T) {
				deployment := mcpDeployment(t, chart, "mcp-enabled.yaml", nil)
				assert.Equal(t, []string{"app", "release"}, sortedKeys(deployment.Spec.Selector.MatchLabels))
				assert.Equal(t, deployment.Spec.Selector.MatchLabels, deployment.Spec.Template.Labels)
			})

			t.Run("set renders podLabels on the pod template", func(t *testing.T) {
				deployment := mcpDeployment(t, chart, "mcp-enabled.yaml", map[string]string{
					"mcp.podLabels.team":        "platform",
					"mcp.podLabels.cost-center": "cc-4711",
				})
				labels := deployment.Spec.Template.Labels
				assert.Equal(t, "platform", labels["team"])
				assert.Equal(t, "cc-4711", labels["cost-center"])
				assert.Equal(t, []string{"app", "release"}, sortedKeys(deployment.Spec.Selector.MatchLabels))
				for k, v := range deployment.Spec.Selector.MatchLabels {
					assert.Equal(t, v, labels[k], "pod template must keep selector label %q", k)
				}
			})
		})
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
