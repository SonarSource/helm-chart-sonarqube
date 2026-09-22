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
				"monitoringPasscode":                  "test-passcode",
				"jdbcOverwrite.jdbcUrl":               "jdbc:postgresql://test-host:5432/testdb",
				"jdbcOverwrite.jdbcUsername":          "test-user",
				"jdbcOverwrite.jdbcPassword":          "test-password",
				"networkPolicy.enabled":               "true",
				"agentOrchestrator.enabled":           "true",
				"agentOrchestrator.image.repository":  "example.com/agent-orchestrator",
				"agentOrchestrator.storage.bucket":    "agent-jobs",
				"service.internalPort":                "9999",
				"hunterAgent.enabled":                 "true",
				"hunterAgent.image.repository":        "example.com/hunter-agent",
				"hunterAgent.image.tag":               "1",
				"agenticSigningSecret.existingSecret": "test-agentic-instance-secret",
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

// On a chart with an Agent Egress Proxy, egressAllow no longer exists: with no sidecar injected
// the runtime's egress list is exactly one rule - the proxy - and setting the old key has no
// effect at all.
func TestAgentRuntimeNetworkPolicyEgressProxyMakesEgressAllowInert(t *testing.T) {
	for _, chart := range agentCharts {
		if !chart.hasEgressProxy {
			continue
		}
		t.Run(chart.name, func(t *testing.T) {
			valuesKey := map[string]string{"hunter": "hunterAgent", "remediation": "remediationAgent"}
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					base := runtimeNetworkPolicy(t, chart, family, nil)
					require.Len(t, base.Spec.Egress, 1, "the proxy only")

					withEgressAllow := runtimeNetworkPolicy(t, chart, family, map[string]string{
						valuesKey[family] + ".networkPolicy.egressAllow[0].cidr": "0.0.0.0/0",
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

// classifyEgressPeer folds one egress peer into the running shape accumulators.
func classifyEgressPeer(peer networkingv1.NetworkPolicyPeer, hasIPBlock, hasKubeDNS *bool, podSelectorApps *[]string) {
	if peer.IPBlock != nil {
		*hasIPBlock = true
	}
	if peer.PodSelector == nil {
		return
	}
	if peer.PodSelector.MatchLabels["k8s-app"] == "kube-dns" {
		*hasKubeDNS = true
	}
	if app, ok := peer.PodSelector.MatchLabels["app"]; ok {
		*podSelectorApps = append(*podSelectorApps, app)
	}
}

// egressPeerShapes flattens a policy's egress peers into a comparable form, so a test can assert
// what a rule reaches without pinning the order of the rule list.
func egressPeerShapes(rules []networkingv1.NetworkPolicyEgressRule) (hasIPBlock bool, hasKubeDNS bool, podSelectorApps []string) {
	for _, rule := range rules {
		for _, peer := range rule.To {
			classifyEgressPeer(peer, &hasIPBlock, &hasKubeDNS, &podSelectorApps)
		}
	}
	return hasIPBlock, hasKubeDNS, podSelectorApps
}

// The posture this whole design exists to produce: an agent runtime that gets no sidecar reaches
// the Agent Egress Proxy and nothing else - one egress rule, no resolver, no ipBlock. It matters
// that the resolver is absent rather than merely unused: NetworkPolicy selects pods and not
// containers, so every rule on this policy is a capability handed to the untrusted agent
// container alongside the runtime. The runtime does not need DNS because it addresses the proxy by
// the ClusterIP kubelet injects as a service-link variable (see agentChart.agentProxyURL), so
// there is no name left to look up.
//
// A runtime that does get an Envoy is the one exception, and needs exactly one name -
// istiod.<istio.namespace>.svc - so kube-dns egress comes back on exactly those paths, once an
// operator has explicitly accepted it via istio.istiodClusterIP: "". Setting it to a real address
// instead pins that one name through hostAliases and takes the resolver away again; see
// TestAgentRuntimeResolverlessMesh. Left at the shipped "auto" with no live cluster to resolve it
// against, the render fails closed rather than falling back to kube-dns - also covered there.
type runtimeResolverCase struct {
	name        string
	setValues   map[string]string
	wantEgress  int
	wantKubeDNS bool
}

var runtimeResolverCases = []runtimeResolverCase{
	{
		name:        "no istio: the proxy and nothing else",
		setValues:   nil,
		wantEgress:  1,
		wantKubeDNS: false,
	},
	{
		// The shipped default for istiodClusterIP is "auto", which resolves by reading the
		// istiod Service - unreachable here, where `helm template` has no cluster to read, so
		// that combination fails the render instead (see
		// TestAgentRuntimeIstiodClusterIPAutoFailsWithoutLiveCluster); "" is the only way to
		// keep the resolver on purpose, and must not read as "unset".
		name: "istio, standard injection, istiodClusterIP opted out: resolver stays",
		setValues: map[string]string{
			"istio.enabled":         "true",
			"gvisor.enabled":        "false",
			"istio.istiodClusterIP": "",
		},
		wantEgress:  3,
		wantKubeDNS: true,
	},
	{
		name:        "istio but sandboxed with no mesh sidecar: no Envoy, so still no resolver",
		setValues:   map[string]string{"istio.enabled": "true", "gvisor.enabled": "true"},
		wantEgress:  1,
		wantKubeDNS: false,
	},
	{
		name: "istio, standard injection, istiodClusterIP set: hostAliases replaces the resolver",
		setValues: map[string]string{
			"istio.enabled":         "true",
			"gvisor.enabled":        "false",
			"istio.istiodClusterIP": "10.96.0.42",
		},
		wantEgress:  2,
		wantKubeDNS: false,
	},
	{
		name: "istiodClusterIP set but no Envoy: nothing to pin, and still no resolver",
		setValues: map[string]string{
			"istio.enabled":         "true",
			"gvisor.enabled":        "true",
			"istio.istiodClusterIP": "10.96.0.42",
		},
		wantEgress:  1,
		wantKubeDNS: false,
	},
}

// egressRuleToApp returns the egress rule reaching the given app label, or nil if none does.
func egressRuleToApp(rules []networkingv1.NetworkPolicyEgressRule, app string) *networkingv1.NetworkPolicyEgressRule {
	for i := range rules {
		rule := rules[i]
		if len(rule.To) == 1 && rule.To[0].PodSelector != nil && rule.To[0].PodSelector.MatchLabels["app"] == app {
			return &rule
		}
	}
	return nil
}

// assertRuntimeResolverCase renders family's NetworkPolicy under c.setValues and checks the
// resulting egress shape - see TestAgentRuntimeNetworkPolicyResolverOnlyWhenInjected.
func assertRuntimeResolverCase(t *testing.T, chart agentChart, family string, c runtimeResolverCase) {
	t.Helper()
	setValues := map[string]string{}
	for k, v := range c.setValues {
		setValues[k] = v
	}
	// sonarqube-dce refuses to render under Istio unless the Hazelcast channels are pinned;
	// irrelevant here, but the gate runs first.
	if len(setValues) > 0 && chart.name == "sonarqube-dce" {
		setValues["applicationNodes.webPort"] = "4023"
		setValues["applicationNodes.cePort"] = "4024"
	}

	policy := runtimeNetworkPolicy(t, chart, family, setValues)
	require.Len(t, policy.Spec.Egress, c.wantEgress)

	hasIPBlock, hasKubeDNS, apps := egressPeerShapes(policy.Spec.Egress)
	assert.False(t, hasIPBlock, "a runtime must never hold a cidr-based egress rule")
	assert.Equal(t, c.wantKubeDNS, hasKubeDNS)
	assert.Contains(t, apps, chart.release+"-agent-egress-proxy")

	proxyRule := egressRuleToApp(policy.Spec.Egress, chart.release+"-agent-egress-proxy")
	require.NotNil(t, proxyRule, "expected an egress rule reaching the Agent Egress Proxy")
	require.Len(t, proxyRule.Ports, 1)
	// Family-scoped: remediation dials the proxy's second listener (remediationPort), the only
	// one whose Squid ACL admits SonarQube's own agentic endpoints; every other family dials port.
	wantPort := int32(3128)
	if family == "remediation" {
		wantPort = 3129
	}
	assert.Equal(t, wantPort, proxyRule.Ports[0].Port.IntVal)
}

// assertMeshSidecarResolver covers the second way a runtime ends up resolving istiod by name: the
// hand-authored mesh sidecar puts an Envoy in a sandboxed pod, and istiodClusterIP is the only
// thing that takes the resolver away again.
func assertMeshSidecarResolver(t *testing.T, chart agentChart, family string) {
	t.Helper()
	// Left at the shipped "auto" default with no live cluster to resolve it against, the render
	// fails closed rather than falling back to kube-dns; see
	// TestAgentRuntimeIstiodClusterIPAutoFailsWithoutLiveCluster.
	t.Run(family+": istio with the mesh sidecar, auto with no cluster to read: fails the render", func(t *testing.T) {
		_, err := renderGvisorIstioSidecar(t, chart, "gvisor-istio-sidecar.yaml",
			[]string{"templates/agent-networkpolicy.yaml"},
			map[string]string{"istio.istiodClusterIP": "auto"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "istiodClusterIP")
	})

	t.Run(family+": istio with the mesh sidecar, istiodClusterIP opted out: resolver stays", func(t *testing.T) {
		policy := gvisorIstioNetworkPolicy(t, chart, "gvisor-istio-sidecar.yaml", family,
			map[string]string{"istio.istiodClusterIP": ""})
		_, hasKubeDNS, _ := egressPeerShapes(policy.Spec.Egress)
		assert.True(t, hasKubeDNS, "a sandboxed runtime running Envoy still resolves istiod by name")
	})

	t.Run(family+": mesh sidecar with istiodClusterIP: no resolver", func(t *testing.T) {
		policy := gvisorIstioNetworkPolicy(t, chart, "gvisor-istio-sidecar.yaml", family,
			map[string]string{"istio.istiodClusterIP": "10.96.0.42"})
		_, hasKubeDNS, _ := egressPeerShapes(policy.Spec.Egress)
		assert.False(t, hasKubeDNS, "hostAliases makes the mesh sidecar's resolver rule unnecessary")
	})
}

func TestAgentRuntimeNetworkPolicyResolverOnlyWhenInjected(t *testing.T) {
	for _, chart := range agentCharts {
		if !chart.hasEgressProxy {
			continue
		}
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				for _, c := range runtimeResolverCases {
					t.Run(family+": "+c.name, func(t *testing.T) {
						assertRuntimeResolverCase(t, chart, family, c)
					})
				}
				assertMeshSidecarResolver(t, chart, family)
			}
		})
	}
}
