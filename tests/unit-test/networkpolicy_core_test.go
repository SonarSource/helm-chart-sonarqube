package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
)

const (
	coreNetworkPolicyNamespace = "sq-test"
	coreNetworkPolicyPort      = 9999
)

// coreNetworkPolicyCharts pairs each chart with the name hint of its SonarQube application policy:
// sonarqube-dce names it "<fullname>-app", sonarqube "<fullname>-network-policy".
var coreNetworkPolicyCharts = []struct {
	chart              agentChart
	extraSetValues     map[string]string
	corePolicyNameHint string
}{
	{
		chart:              agentCharts[0],
		extraSetValues:     map[string]string{"applicationNodes.jwtSecret": "test-jwt-secret"},
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

// renderCoreNetworkPolicy renders the SonarQube application NetworkPolicy with networkPolicy on,
// service.internalPort moved off its default and no agentic component enabled; overrides layers
// the scenario under test on top. jsonValues go through --set-json, for nested selectors.
func renderCoreNetworkPolicy(t *testing.T, i int, overrides, jsonValues map[string]string) networkingv1.NetworkPolicy {
	t.Helper()
	output, err := renderCoreNetworkPolicyOutput(t, i, overrides, jsonValues)
	require.NoError(t, err)
	return parseCorePolicy(t, i, output)
}

// renderCoreNetworkPolicyE returns only the render error, for scenarios the chart rejects.
func renderCoreNetworkPolicyE(t *testing.T, i int, overrides, jsonValues map[string]string) error {
	t.Helper()
	_, err := renderCoreNetworkPolicyOutput(t, i, overrides, jsonValues)
	return err
}

func renderCoreNetworkPolicyOutput(t *testing.T, i int, overrides, jsonValues map[string]string) (string, error) {
	t.Helper()
	c := coreNetworkPolicyCharts[i]
	setValues := map[string]string{
		"monitoringPasscode":         "test-passcode",
		"jdbcOverwrite.jdbcUrl":      "jdbc:postgresql://test-host:5432/testdb",
		"jdbcOverwrite.jdbcUsername": "test-user",
		"jdbcOverwrite.jdbcPassword": "test-password",
		"networkPolicy.enabled":      "true",
		"service.internalPort":       "9999",
	}
	for k, v := range c.extraSetValues {
		setValues[k] = v
	}
	for k, v := range overrides {
		setValues[k] = v
	}

	opts := &helm.Options{
		Logger:         logger.Discard,
		SetValues:      setValues,
		SetJsonValues:  jsonValues,
		KubectlOptions: k8s.NewKubectlOptions("", "", coreNetworkPolicyNamespace),
	}
	return helm.RenderTemplateE(t, opts, c.chart.path, c.chart.release, []string{"templates/networkpolicy.yaml"})
}

func parseCorePolicy(t *testing.T, i int, output string) networkingv1.NetworkPolicy {
	t.Helper()
	c := coreNetworkPolicyCharts[i]
	var policy networkingv1.NetworkPolicy
	for _, doc := range strings.Split(output, "\n---") {
		if strings.Contains(doc, c.corePolicyNameHint) {
			helm.UnmarshalK8SYaml(t, doc, &policy)
		}
	}
	require.NotEmpty(t, policy.Name)
	return policy
}

func remediationAgentValues() map[string]string {
	return map[string]string{
		"agentOrchestrator.enabled":           "true",
		"agentOrchestrator.image.repository":  "example.com/agent-orchestrator",
		"agentOrchestrator.storage.bucket":    "agent-jobs",
		"remediationAgent.enabled":            "true",
		"remediationAgent.image.repository":   "example.com/remediation-agent",
		"remediationAgent.image.tag":          "1",
		"vortexAnalysis.image.repository":     "example.com/vortex",
		"vortexAnalysis.image.tag":            "1",
		"vortexAnalysis.storage.bucket":       "vortex-artifacts",
		"vortexAnalysis.storage.region":       "eu-west-1",
		"agenticSigningSecret.existingSecret": "test-agentic-instance-secret",
	}
}

func hunterAgentValues() map[string]string {
	return map[string]string{
		"agentOrchestrator.enabled":           "true",
		"agentOrchestrator.image.repository":  "example.com/agent-orchestrator",
		"agentOrchestrator.storage.bucket":    "agent-jobs",
		"hunterAgent.enabled":                 "true",
		"hunterAgent.image.repository":        "example.com/hunter-agent",
		"hunterAgent.image.tag":               "1",
		"agenticSigningSecret.existingSecret": "test-agentic-instance-secret",
	}
}

// ingressRulesFromPod returns the ingress rules whose single peer is a pod in the policy's own
// namespace carrying the given app label.
func ingressRulesFromPod(policy networkingv1.NetworkPolicy, app string) []networkingv1.NetworkPolicyIngressRule {
	var rules []networkingv1.NetworkPolicyIngressRule
	for _, rule := range policy.Spec.Ingress {
		if len(rule.From) == 1 && rule.From[0].NamespaceSelector == nil && rule.From[0].PodSelector != nil &&
			rule.From[0].PodSelector.MatchLabels["app"] == app {
			rules = append(rules, rule)
		}
	}
	return rules
}

// entryPointRules returns the ingress rules that admit traffic on the SonarQube port from a
// namespace selector: the rule for an Ingress controller or Gateway data plane. The Prometheus rule
// also selects a namespace, but on the exporter ports.
func entryPointRules(policy networkingv1.NetworkPolicy) []networkingv1.NetworkPolicyIngressRule {
	var rules []networkingv1.NetworkPolicyIngressRule
	for _, rule := range policy.Spec.Ingress {
		if len(rule.From) != 1 || rule.From[0].NamespaceSelector == nil {
			continue
		}
		if len(rule.Ports) == 1 && rule.Ports[0].Port != nil && rule.Ports[0].Port.IntVal == coreNetworkPolicyPort {
			rules = append(rules, rule)
		}
	}
	return rules
}

// The egress proxy carries the Remediation runtime's SonarQube calls (rule details, fix
// verification), so the SonarQube policy has to admit it. Hunter must stay unable to reach SonarQube
// (SONAR-32432), so the rule follows remediationAgent, not the proxy being deployed.
func TestCoreNetworkPolicyAdmitsEgressProxy(t *testing.T) {
	for i, c := range coreNetworkPolicyCharts {
		proxyApp := c.chart.release + "-agent-egress-proxy"

		t.Run(c.chart.name+": remediation enabled admits the proxy on service.internalPort", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, remediationAgentValues(), nil)

			rules := ingressRulesFromPod(policy, proxyApp)
			require.Len(t, rules, 1, "expected one ingress rule from the egress proxy")
			assert.Equal(t, c.chart.release, rules[0].From[0].PodSelector.MatchLabels["release"])
			require.Len(t, rules[0].Ports, 1)
			assert.Equal(t, int32(coreNetworkPolicyPort), rules[0].Ports[0].Port.IntVal)
			require.NotNil(t, rules[0].Ports[0].Protocol)
			assert.Equal(t, "TCP", string(*rules[0].Ports[0].Protocol))
		})

		t.Run(c.chart.name+": hunter only does not admit the proxy", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, hunterAgentValues(), nil)
			assert.Empty(t, ingressRulesFromPod(policy, proxyApp))
		})

		t.Run(c.chart.name+": no agent does not admit the proxy", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, nil, nil)
			assert.Empty(t, ingressRulesFromPod(policy, proxyApp))
		})
	}
}

// An Ingress controller or Gateway data plane normally runs in another namespace, so its rule is a
// namespace selector. Without one, users must write a policy of their own to open the UI.
func TestCoreNetworkPolicyAdmitsEntryPoint(t *testing.T) {
	overrideJSON := map[string]string{
		"networkPolicy.ingressController": `{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"traefik"}},` +
			`"podSelector":{"matchLabels":{"app.kubernetes.io/name":"traefik"}}}`,
	}

	for i, c := range coreNetworkPolicyCharts {
		t.Run(c.chart.name+": ingress.enabled alone admits no entry point", func(t *testing.T) {
			// The controller's namespace is unknown; admitting every namespace would let in-cluster
			// callers skip the Ingress. NOTES.txt tells the user to set ingressController.
			policy := renderCoreNetworkPolicy(t, i, map[string]string{"ingress.enabled": "true"}, nil)
			assert.Empty(t, entryPointRules(policy))
		})

		t.Run(c.chart.name+": httproute.enabled admits the gateway namespace", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, map[string]string{
				"httproute.enabled":          "true",
				"httproute.gateway":          "my-gateway",
				"httproute.gatewayNamespace": "gw-ns",
			}, nil)

			rules := entryPointRules(policy)
			require.Len(t, rules, 1)
			peer := rules[0].From[0]
			assert.Equal(t, map[string]string{"kubernetes.io/metadata.name": "gw-ns"}, peer.NamespaceSelector.MatchLabels)
			assert.Nil(t, peer.PodSelector)
		})

		t.Run(c.chart.name+": httproute without gatewayNamespace admits no entry point, with or without hunter", func(t *testing.T) {
			// Admitting the release namespace would also admit sibling pods such as Hunter.
			for name, values := range map[string]map[string]string{
				"hunter off": {},
				"hunter on":  hunterAgentValues(),
			} {
				values["httproute.enabled"] = "true"
				values["httproute.gateway"] = "my-gateway"
				policy := renderCoreNetworkPolicy(t, i, values, nil)
				assert.Empty(t, entryPointRules(policy), name)
			}
		})

		t.Run(c.chart.name+": httproute gatewayNamespace equal to the release namespace admits no entry point", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, map[string]string{
				"httproute.enabled":          "true",
				"httproute.gateway":          "my-gateway",
				"httproute.gatewayNamespace": coreNetworkPolicyNamespace,
			}, nil)
			assert.Empty(t, entryPointRules(policy))
		})

		t.Run(c.chart.name+": ingress and httproute together admit only the gateway namespace", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, map[string]string{
				"ingress.enabled":            "true",
				"httproute.enabled":          "true",
				"httproute.gateway":          "my-gateway",
				"httproute.gatewayNamespace": "gw-ns",
			}, nil)

			rules := entryPointRules(policy)
			require.Len(t, rules, 1)
			assert.Equal(t, map[string]string{"kubernetes.io/metadata.name": "gw-ns"},
				rules[0].From[0].NamespaceSelector.MatchLabels)
		})

		t.Run(c.chart.name+": httproute in the release namespace renders with an ingressController podSelector", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, map[string]string{
				"httproute.enabled": "true",
				"httproute.gateway": "my-gateway",
			}, map[string]string{
				"networkPolicy.ingressController": `{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"sq-test"}},` +
					`"podSelector":{"matchLabels":{"gateway.networking.k8s.io/gateway-name":"my-gateway"}}}`,
			})

			rules := entryPointRules(policy)
			require.Len(t, rules, 1)
			require.NotNil(t, rules[0].From[0].PodSelector, "namespace-wide admission would let Hunter in")
		})

		t.Run(c.chart.name+": ingressController with only a podSelector fails", func(t *testing.T) {
			err := renderCoreNetworkPolicyE(t, i, nil, map[string]string{
				"networkPolicy.ingressController": `{"podSelector":{"matchLabels":{"app":"gw"}}}`,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "namespaceSelector is required")
		})

		t.Run(c.chart.name+": ingressController with an unknown key is rejected by the schema", func(t *testing.T) {
			err := renderCoreNetworkPolicyE(t, i, nil, map[string]string{
				"networkPolicy.ingressController": `{"namespaceSelectr":{}}`,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "namespaceSelectr")
		})

		t.Run(c.chart.name+": ingressController alone admits it, without ingress or httproute", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, nil, overrideJSON)

			rules := entryPointRules(policy)
			require.Len(t, rules, 1)
			peer := rules[0].From[0]
			assert.Equal(t, map[string]string{"kubernetes.io/metadata.name": "traefik"}, peer.NamespaceSelector.MatchLabels)
			require.NotNil(t, peer.PodSelector, "namespace and pod selectors belong in the same peer so they AND")
			assert.Equal(t, map[string]string{"app.kubernetes.io/name": "traefik"}, peer.PodSelector.MatchLabels)
		})

		t.Run(c.chart.name+": ingressController with ingress.enabled admits only the selected controller", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, map[string]string{"ingress.enabled": "true"}, overrideJSON)

			rules := entryPointRules(policy)
			require.Len(t, rules, 1)
			assert.Equal(t, map[string]string{"kubernetes.io/metadata.name": "traefik"},
				rules[0].From[0].NamespaceSelector.MatchLabels)
		})

		t.Run(c.chart.name+": ingressController replaces the httproute default rather than adding to it", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, map[string]string{
				"httproute.enabled":          "true",
				"httproute.gateway":          "my-gateway",
				"httproute.gatewayNamespace": "gw-ns",
			}, overrideJSON)

			rules := entryPointRules(policy)
			require.Len(t, rules, 1)
			assert.Equal(t, map[string]string{"kubernetes.io/metadata.name": "traefik"},
				rules[0].From[0].NamespaceSelector.MatchLabels)
		})

		t.Run(c.chart.name+": nothing exposing SonarQube admits no entry point", func(t *testing.T) {
			policy := renderCoreNetworkPolicy(t, i, nil, nil)
			assert.Empty(t, entryPointRules(policy))
		})
	}
}
