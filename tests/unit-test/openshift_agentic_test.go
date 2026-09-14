package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	nodev1 "k8s.io/api/node/v1"
)

// SONAR-31373: flipping OpenShift.enabled must be enough to install the agentic pack, so the
// fixture below is the whole pack with every NetworkPolicy on and gVisor left at its defaults.
const openShiftAgenticFixture = "openshift-agentic.yaml"

// renderOpenShift renders the given templates against the OpenShift fixture. Helm executes
// validation.yaml on every render, so a validation `fail` surfaces here regardless of which
// template is the show-only target (used by the negative case below).
func renderOpenShift(t *testing.T, chart agentChart, setValues map[string]string, templates ...string) (string, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/" + openShiftAgenticFixture},
		SetValues:   setValues,
	}
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, templates)
}

// openShiftNetworkPolicies returns every NetworkPolicy the chart renders - the one guarding
// SonarQube itself (two of them in sonarqube-dce: app and search), one per agent runtime family,
// and the Agent Egress Proxy's. All of them carry a DNS egress rule from the shared helper.
func openShiftNetworkPolicies(t *testing.T, chart agentChart, setValues map[string]string) []networkingv1.NetworkPolicy {
	t.Helper()
	output, err := renderOpenShift(t, chart, setValues,
		"templates/networkpolicy.yaml",
		"templates/agent-networkpolicy.yaml",
		"templates/agent-egress-proxy-networkpolicy.yaml",
	)
	require.NoError(t, err)

	var policies []networkingv1.NetworkPolicy
	for _, doc := range splitGvisorDocs(output) {
		if gvisorDocKind(doc) != "NetworkPolicy" {
			continue
		}
		var policy networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, doc, &policy)
		policies = append(policies, policy)
	}
	require.NotEmpty(t, policies)
	return policies
}

// dnsEgressRules picks out the egress rules that are about DNS - the ones naming a port the helper
// can emit. Anything else (the orchestrator, the proxy, the search nodes) is left alone.
func dnsEgressRules(policy networkingv1.NetworkPolicy) []networkingv1.NetworkPolicyEgressRule {
	var rules []networkingv1.NetworkPolicyEgressRule
	for _, rule := range policy.Spec.Egress {
		for _, port := range rule.Ports {
			if port.Port != nil && (port.Port.IntValue() == 53 || port.Port.IntValue() == 5353) {
				rules = append(rules, rule)
				break
			}
		}
	}
	return rules
}

func portProtocols(t *testing.T, rule networkingv1.NetworkPolicyEgressRule) map[string]int {
	t.Helper()
	byProtocol := map[string]int{}
	for _, port := range rule.Ports {
		require.NotNil(t, port.Port)
		protocol := "TCP"
		if port.Protocol != nil {
			protocol = string(*port.Protocol)
		}
		byProtocol[protocol] = port.Port.IntValue()
	}
	return byProtocol
}

// On OpenShift the DNS rule has to name the openshift-dns namespace and port 5353: the platform's
// CoreDNS pods carry no k8s-app label, and OVN-Kubernetes matches egress ACLs after DNAT, so a
// port-53 rule is never hit and every pod loses DNS the moment a policy is applied.
func TestOpenShiftDnsEgressRule(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			policies := openShiftNetworkPolicies(t, chart, nil)
			for _, policy := range policies {
				t.Run(policy.Name, func(t *testing.T) {
					rules := dnsEgressRules(policy)
					require.Len(t, rules, 1, "exactly one DNS egress rule")

					require.Len(t, rules[0].To, 1)
					peer := rules[0].To[0]
					require.NotNil(t, peer.NamespaceSelector)
					assert.Equal(t, map[string]string{"kubernetes.io/metadata.name": "openshift-dns"},
						peer.NamespaceSelector.MatchLabels)
					assert.Nil(t, peer.PodSelector, "openshift-dns pods carry no k8s-app label to select on")

					assert.Equal(t, map[string]int{"UDP": 5353, "TCP": 5353}, portProtocols(t, rules[0]))
				})
			}
		})
	}
}

// The same rule off OpenShift must keep targeting kube-dns on 53 - this is the regression guard for
// every non-OpenShift user of both charts.
func TestDnsEgressRuleOffOpenShift(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			policies := openShiftNetworkPolicies(t, chart, map[string]string{"OpenShift.enabled": "false"})
			for _, policy := range policies {
				t.Run(policy.Name, func(t *testing.T) {
					rules := dnsEgressRules(policy)
					require.Len(t, rules, 1)

					require.Len(t, rules[0].To, 1)
					peer := rules[0].To[0]
					require.NotNil(t, peer.NamespaceSelector)
					assert.Empty(t, peer.NamespaceSelector.MatchLabels)
					require.NotNil(t, peer.PodSelector)
					assert.Equal(t, map[string]string{"k8s-app": "kube-dns"}, peer.PodSelector.MatchLabels)

					assert.Equal(t, map[string]int{"UDP": 53, "TCP": 53}, portProtocols(t, rules[0]))
				})
			}
		})
	}
}

// gvisor.enabled and gvisor.installer.enabled both default to true, but on OpenShift the feature
// cannot work: the installer needs containerd plus privileged/hostPID (no SCC grants that) and
// CRI-O has no runsc handler. Rendering it anyway produced a RuntimeClass the runtimes could never
// be scheduled with, while Helm still reported success - so the whole feature stays off.
func TestOpenShiftGvisorOffByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			// helm --show-only errors out when its target renders nothing at all, which is
			// exactly the expected outcome here; assert on the message so a validation
			// failure (also Error + empty output) cannot pass for it.
			output, err := renderOpenShift(t, chart, nil, "templates/gvisor.yaml")
			require.Error(t, err, "gvisor.yaml must render nothing on OpenShift")
			assert.Contains(t, err.Error(), "could not find template")
			assert.Empty(t, strings.TrimSpace(output), "no RuntimeClass and no installer DaemonSet")

			for _, deployment := range openShiftAgentRuntimes(t, chart, nil) {
				assert.Nil(t, deployment.Spec.Template.Spec.RuntimeClassName,
					"%s must not reference a RuntimeClass that is never created", deployment.Name)
			}
		})
	}
}

// Opting in (having provisioned runsc out of band) brings the RuntimeClass and the runtime wiring
// back, but never the installer.
func TestOpenShiftGvisorOptIn(t *testing.T) {
	optIn := map[string]string{
		"gvisor.openShiftOptIn":    "true",
		"gvisor.installer.enabled": "false",
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderOpenShift(t, chart, optIn, "templates/gvisor.yaml")
			require.NoError(t, err)

			kinds := map[string]int{}
			for _, doc := range splitGvisorDocs(output) {
				kinds[gvisorDocKind(doc)]++
			}
			assert.Equal(t, 1, kinds["RuntimeClass"])
			assert.Zero(t, kinds["DaemonSet"], "the installer cannot run on OpenShift")

			var runtimeClass nodev1.RuntimeClass
			for _, doc := range splitGvisorDocs(output) {
				if gvisorDocKind(doc) == "RuntimeClass" {
					helm.UnmarshalK8SYaml(t, doc, &runtimeClass)
				}
			}
			assert.Equal(t, "gvisor", runtimeClass.Name)

			for _, deployment := range openShiftAgentRuntimes(t, chart, optIn) {
				require.NotNil(t, deployment.Spec.Template.Spec.RuntimeClassName, deployment.Name)
				assert.Equal(t, "gvisor", *deployment.Spec.Template.Spec.RuntimeClassName)
			}
		})
	}
}

// Opting in while leaving the installer on would render a DaemonSet every node rejects, with Helm
// reporting success - so the combination is refused up front instead.
func TestOpenShiftGvisorOptInRejectsInstaller(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderOpenShift(t, chart, map[string]string{
				"gvisor.openShiftOptIn":    "true",
				"gvisor.installer.enabled": "true",
			}, "templates/gvisor.yaml")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "gvisor.openShiftOptIn=true requires gvisor.installer.enabled=false")
		})
	}
}

func openShiftAgentRuntimes(t *testing.T, chart agentChart, setValues map[string]string) []appsv1.Deployment {
	t.Helper()
	output, err := renderOpenShift(t, chart, setValues, "templates/agent-runtime.yaml")
	require.NoError(t, err)

	var deployments []appsv1.Deployment
	for _, doc := range splitGvisorDocs(output) {
		var deployment appsv1.Deployment
		helm.UnmarshalK8SYaml(t, doc, &deployment)
		deployments = append(deployments, deployment)
	}
	require.Len(t, deployments, 2, "hunter and remediation")
	return deployments
}

// Every agentic workload must leave the UID/GID choice to OpenShift: restricted-v2 admits pods
// under the namespace's own uid-range and rejects any explicit runAsUser outside it, which is what
// used to keep the whole pack from ever starting.
func TestOpenShiftAgenticWorkloadsDropUids(t *testing.T) {
	templates := []string{
		"templates/agent-orchestrator.yaml",
		"templates/agent-runtime.yaml",
		"templates/vortex.yaml",
		"templates/agent-egress-proxy.yaml",
		"templates/agent-key-derivation-hook.yaml",
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, template := range templates {
				t.Run(template, func(t *testing.T) {
					output, err := renderOpenShift(t, chart, nil, template)
					require.NoError(t, err)
					docs := splitGvisorDocs(output)
					require.NotEmpty(t, docs)

					for _, doc := range docs {
						// Deployment, Job and bare Pod all carry the pod template at a
						// different path, so assert on the raw manifest instead of picking a
						// typed struct per kind.
						assert.NotContains(t, doc, "runAsUser:")
						assert.NotContains(t, doc, "runAsGroup:")
						assert.NotContains(t, doc, "fsGroup:")
						// The hardening that must survive the UID stripping.
						assert.Contains(t, doc, "runAsNonRoot: true")
					}
				})
			}
		})
	}
}

// Off OpenShift the explicit UIDs must stay: they are what makes the images run as a known
// non-root user on clusters that do not assign one.
func TestAgenticWorkloadsKeepUidsOffOpenShift(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderOpenShift(t, chart,
				map[string]string{"OpenShift.enabled": "false"}, "templates/agent-orchestrator.yaml")
			require.NoError(t, err)
			assert.Contains(t, output, "runAsUser:")
		})
	}
}

// Guard against the pod-level securityContext coming back as an empty map: `{}` is not the same as
// absent to some admission plugins, and the helper relies on `with` skipping it entirely.
func TestOpenShiftOrchestratorPodSecurityContextAbsent(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderOpenShift(t, chart, nil, "templates/agent-orchestrator.yaml")
			require.NoError(t, err)

			var deployment appsv1.Deployment
			helm.UnmarshalK8SYaml(t, output, &deployment)
			assert.Nil(t, deployment.Spec.Template.Spec.SecurityContext)

			for _, container := range deployment.Spec.Template.Spec.Containers {
				assertNoExplicitUser(t, container.Name, container.SecurityContext)
			}
			for _, container := range deployment.Spec.Template.Spec.InitContainers {
				assertNoExplicitUser(t, container.Name, container.SecurityContext)
			}
		})
	}
}

func assertNoExplicitUser(t *testing.T, name string, securityContext *corev1.SecurityContext) {
	t.Helper()
	require.NotNil(t, securityContext, name)
	assert.Nil(t, securityContext.RunAsUser, name)
	assert.Nil(t, securityContext.RunAsGroup, name)
}
