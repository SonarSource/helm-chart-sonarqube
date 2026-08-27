package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
)

func renderAgentRuntimeNetworkPolicy(t *testing.T, chart agentChart, setValues map[string]string) (string, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/agent-networkpolicy-enabled.yaml"},
		SetValues:   setValues,
	}
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-networkpolicy.yaml"})
}

// Both hunter and remediation are enabled in the fixture, so the render emits one NetworkPolicy
// document per family; pick out the one for family.
func runtimeNetworkPolicy(t *testing.T, chart agentChart, family string, setValues map[string]string) networkingv1.NetworkPolicy {
	t.Helper()
	output, err := renderAgentRuntimeNetworkPolicy(t, chart, setValues)
	require.NoError(t, err)

	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var policy networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, doc, &policy)
		if policy.Labels["sonarqube.agent/family"] == family {
			return policy
		}
	}
	require.FailNowf(t, "no NetworkPolicy rendered", "family %q", family)
	return networkingv1.NetworkPolicy{}
}

// The ingress rule that lets the Agent Orchestrator reach the application pods must follow
// service.internalPort, not a hardcoded value, or it silently breaks for anyone changing it.
func TestCoreNetworkPolicyOrchestratorIngressPort(t *testing.T) {
	cases := []struct {
		chart              agentChart
		extraSetValues     map[string]string
		corePolicyNameHint string
	}{
		{
			chart: agentCharts[0],
			extraSetValues: map[string]string{
				"applicationNodes.jwtSecret": "test-jwt-secret",
			},
			corePolicyNameHint: "-app\n",
		},
		{
			chart: agentCharts[1],
			extraSetValues: map[string]string{
				"community.enabled":     "true",
				"jdbcOverwrite.enabled": "true",
			},
			corePolicyNameHint: "-network-policy\n",
		},
	}

	for _, c := range cases {
		t.Run(c.chart.name, func(t *testing.T) {
			setValues := map[string]string{
				"monitoringPasscode":                 "test-passcode",
				"jdbcOverwrite.jdbcUrl":              "jdbc:postgresql://test-host:5432/testdb",
				"jdbcOverwrite.jdbcUsername":         "test-user",
				"jdbcOverwrite.jdbcPassword":         "test-password",
				"networkPolicy.enabled":              "true",
				"agentOrchestrator.enabled":          "true",
				"agentOrchestrator.image.repository": "example.com/agent-orchestrator",
				"agentOrchestrator.storage.bucket":   "agent-jobs",
				"service.internalPort":               "9999",
			}
			for k, v := range c.extraSetValues {
				setValues[k] = v
			}

			opts := &helm.Options{
				Logger:    logger.Discard,
				SetValues: setValues,
			}
			output, err := helm.RenderTemplateE(t, opts, c.chart.path, c.chart.release, []string{"templates/networkpolicy.yaml"})
			require.NoError(t, err)

			var policy networkingv1.NetworkPolicy
			for _, doc := range strings.Split(output, "\n---") {
				if strings.Contains(doc, c.corePolicyNameHint) {
					helm.UnmarshalK8SYaml(t, doc, &policy)
				}
			}
			require.NotEmpty(t, policy.Name)

			var fromOrchestrator *networkingv1.NetworkPolicyIngressRule
			for i := range policy.Spec.Ingress {
				rule := policy.Spec.Ingress[i]
				if len(rule.From) == 1 && rule.From[0].PodSelector != nil &&
					rule.From[0].PodSelector.MatchLabels["app"] == c.chart.release+"-agent-orchestrator" {
					fromOrchestrator = &rule
				}
			}
			require.NotNil(t, fromOrchestrator, "expected an ingress rule from the Agent Orchestrator")
			require.Len(t, fromOrchestrator.Ports, 1)
			assert.Equal(t, int32(9999), fromOrchestrator.Ports[0].Port.IntVal)
		})
	}
}

// A runtime reads/writes job artifacts directly against object storage via presigned URLs, so its
// NetworkPolicy needs its own egress to reach it. On charts without an Agent Egress Proxy, the
// chart can't resolve agentOrchestrator.storage to a peer on its own, so
// networkPolicy.egressAllow must cover it - this is documented on
// <hunterAgent|remediationAgent>.networkPolicy.egressAllow in values.yaml, not enforced by the
// render (SONAR-31525). On charts with a proxy, that egress (and everything else) transits it
// instead, and egressAllow no longer exists at all - see TestAgentEgressProxyRequired in
// agent_egress_proxy_test.go for the D8 auto-activation behavior this replaces it with.
func TestAgentRuntimeNetworkPolicyEgressAllow(t *testing.T) {
	valuesKey := map[string]string{"hunter": "hunterAgent", "remediation": "remediationAgent"}
	for _, chart := range agentCharts {
		if chart.hasEgressProxy {
			continue
		}
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family+": empty egressAllow renders no extra egress rule", func(t *testing.T) {
					policy := runtimeNetworkPolicy(t, chart, family, nil)
					require.Len(t, policy.Spec.Egress, 2, "DNS and the orchestrator only")
				})

				t.Run(family+": egressAllow entries render as given", func(t *testing.T) {
					policy := runtimeNetworkPolicy(t, chart, family, map[string]string{
						valuesKey[family] + ".networkPolicy.egressAllow[0].cidr":              "0.0.0.0/0",
						valuesKey[family] + ".networkPolicy.egressAllow[0].ports[0].port":     "443",
						valuesKey[family] + ".networkPolicy.egressAllow[0].ports[0].protocol": "TCP",
					})

					require.Len(t, policy.Spec.Egress, 3, "DNS, the orchestrator, and the one egressAllow entry")
					last := policy.Spec.Egress[2]
					require.Len(t, last.To, 1)
					require.NotNil(t, last.To[0].IPBlock)
					assert.Equal(t, "0.0.0.0/0", last.To[0].IPBlock.CIDR)
					require.Len(t, last.Ports, 1)
					assert.Equal(t, int32(443), last.Ports[0].Port.IntVal)
				})
			}
		})
	}
}

// On a chart with an Agent Egress Proxy, egressAllow no longer exists: the runtime's egress list
// is always exactly DNS + the proxy, and setting the old key has no effect at all.
func TestAgentRuntimeNetworkPolicyEgressProxyMakesEgressAllowInert(t *testing.T) {
	for _, chart := range agentCharts {
		if !chart.hasEgressProxy {
			continue
		}
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					base := runtimeNetworkPolicy(t, chart, family, nil)
					require.Len(t, base.Spec.Egress, 2, "DNS and the proxy only")

					withEgressAllow := runtimeNetworkPolicy(t, chart, family, map[string]string{
						"hunterAgent.networkPolicy.egressAllow[0].cidr": "0.0.0.0/0",
					})
					assert.Equal(t, base.Spec.Egress, withEgressAllow.Spec.Egress, "egressAllow must be fully inert")
				})
			}
		})
	}
}

// A podSelector-only egressAllow entry (no namespaceSelector) must not render a stray
// `namespaceSelector: null` peer alongside it - only the keys actually given. Only applies to
// charts still supporting egressAllow.
func TestAgentRuntimeNetworkPolicyEgressAllowPodSelectorNoNullKeys(t *testing.T) {
	for _, chart := range agentCharts {
		if chart.hasEgressProxy {
			continue
		}
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{
				"hunterAgent.networkPolicy.egressAllow[0].podSelector.matchLabels.app": "some-dependency",
			}

			// Asserted on the rendered YAML, not the parsed peer: `namespaceSelector: null` unmarshals to
			// the same nil pointer as an absent key, so a typed check can't tell the two apart.
			output, err := renderAgentRuntimeNetworkPolicy(t, chart, setValues)
			require.NoError(t, err)
			assert.NotContains(t, output, "namespaceSelector: null")

			policy := runtimeNetworkPolicy(t, chart, "hunter", setValues)
			require.Len(t, policy.Spec.Egress, 3, "DNS, the orchestrator, and the one egressAllow entry")
			last := policy.Spec.Egress[2]
			require.Len(t, last.To, 1)
			require.NotNil(t, last.To[0].PodSelector)
			assert.Equal(t, map[string]string{"app": "some-dependency"}, last.To[0].PodSelector.MatchLabels)
			assert.Nil(t, last.To[0].IPBlock)
		})
	}
}
