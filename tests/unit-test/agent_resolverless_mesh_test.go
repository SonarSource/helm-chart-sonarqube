package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// runtimeHostAliases returns the hostAliases on an agent runtime's pod template.
func runtimeHostAliases(t *testing.T, chart agentChart, fixture, family string, setValues map[string]string) []corev1.HostAlias {
	t.Helper()
	spec := agenticPodSpec(t, chart, "templates/agent-runtime.yaml", fixture, setValues, family)
	return spec.HostAliases
}

// istio.istiodClusterIP exists so that an agent runtime carrying an Envoy can be denied kube-dns
// egress outright, which matters because NetworkPolicy selects pods and not containers: every
// egress rule on a runtime's policy is equally a capability of the untrusted agent container
// sharing that pod, and reaching a recursive resolver is a bidirectional channel out of the
// cluster (payload in the query name, answer in a TXT record) that neither the Agent Egress
// Proxy's allowedDomains ACL nor any L3/L4 rule can see. Envoy is not in that path either -
// Istio's iptables rules are TCP-only unless DNS capture is explicitly enabled.
//
// What makes it possible is that the pod needs exactly one name, istiod.<istio.namespace>.svc,
// and it is pilot-agent - Go, and so /etc/hosts-aware - that resolves it rather than Envoy, whose
// xds-grpc cluster is STATIC over a Unix socket. So the mapping belongs in hostAliases, keeping
// the hostname (and with it istiod's TLS SAN validation) intact.
func TestAgentRuntimeResolverlessMesh(t *testing.T) {
	const clusterIP = "10.96.0.42"

	for _, chart := range agentCharts {
		if !chart.hasEgressProxy {
			continue
		}
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				// The hand-authored mesh sidecar path: sandboxed pod, Envoy present.
				t.Run(family+": mesh sidecar gets the istiod mapping", func(t *testing.T) {
					aliases := runtimeHostAliases(t, chart, "gvisor-istio-sidecar.yaml", family,
						map[string]string{"istio.istiodClusterIP": clusterIP})
					require.Len(t, aliases, 1)
					assert.Equal(t, clusterIP, aliases[0].IP)
					// Both spellings: the two injection paths write discoveryAddress differently.
					assert.ElementsMatch(t,
						[]string{"istiod.istio-system.svc", "istiod.istio-system.svc.cluster.local"},
						aliases[0].Hostnames)
				})

				// Standard injection: not sandboxed, so the injector supplies the Envoy.
				t.Run(family+": standard injection gets the istiod mapping", func(t *testing.T) {
					aliases := runtimeHostAliases(t, chart, "gvisor-istio-sidecar.yaml", family,
						map[string]string{
							"gvisor.enabled":        "false",
							"istio.istiodClusterIP": clusterIP,
						})
					require.Len(t, aliases, 1)
					assert.Equal(t, clusterIP, aliases[0].IP)
				})

				// istio.namespace is not assumed to be istio-system.
				t.Run(family+": the mapping follows istio.namespace", func(t *testing.T) {
					aliases := runtimeHostAliases(t, chart, "gvisor-istio-sidecar.yaml", family,
						map[string]string{
							"istio.namespace":       "mesh-control",
							"istio.istiodClusterIP": clusterIP,
						})
					require.Len(t, aliases, 1)
					assert.ElementsMatch(t,
						[]string{"istiod.mesh-control.svc", "istiod.mesh-control.svc.cluster.local"},
						aliases[0].Hostnames)
				})

				// Nothing to pin when the pod has no Envoy, and nothing to pin when it has no
				// mesh at all - in both cases the runtime resolves nothing, reaching the Agent
				// Egress Proxy by the ClusterIP kubelet hands it as a service-link variable.
				t.Run(family+": no hostAliases without an Envoy", func(t *testing.T) {
					aliases := runtimeHostAliases(t, chart, "gvisor-istio-sidecar-off.yaml", family,
						map[string]string{"istio.istiodClusterIP": clusterIP})
					assert.Empty(t, aliases)
				})

				t.Run(family+": no hostAliases with istio disabled", func(t *testing.T) {
					aliases := runtimeHostAliases(t, chart, "gvisor-istio-sidecar.yaml", family,
						map[string]string{
							"istio.enabled":         "false",
							"istio.istiodClusterIP": clusterIP,
						})
					assert.Empty(t, aliases)
				})

				// Default: opt-in, so leaving the value unset must change nothing.
				t.Run(family+": no hostAliases by default", func(t *testing.T) {
					aliases := runtimeHostAliases(t, chart, "gvisor-istio-sidecar.yaml", family, nil)
					assert.Empty(t, aliases)
				})
			}
		})
	}
}

// A hostname in hostAliases is rejected by the API server, but only at apply time - which for a
// value whose whole point is closing a DNS channel is far too late to find out. The schema
// pattern catches the likely mistake (pasting the name this value replaces) at template time.
func TestAgentRuntimeIstiodClusterIPRejectsHostname(t *testing.T) {
	for _, chart := range agentCharts {
		if !chart.hasEgressProxy {
			continue
		}
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderAgentRuntimeNetworkPolicy(t, chart, map[string]string{
				"istio.enabled":         "true",
				"istio.istiodClusterIP": "istiod.istio-system.svc",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "istiodClusterIP")
		})
	}
}
