package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// agentRuntimeSandbox.enabled is the generic, non-gVisor alternative to gvisor.enabled: any
// sandboxing RuntimeClass that can't grant istio-init NET_ADMIN (e.g. Kata Containers under AKS
// Pod Sandboxing) can opt into the exact same mesh-sidecar treatment gVisor gets. The
// sandbox-istio-sidecar.yaml fixture proves this with gvisor.enabled: false throughout, so any
// pass here can only be explained by the generic path, not by gVisor quietly still being active.

// The runtime Deployment must actually run on the caller-supplied RuntimeClass, not silently stay
// on the cluster's default runtime while still being excluded from Istio injection - that
// combination would be worse than doing nothing (no sandbox isolation, and the mTLS story no
// longer matches reality).
func TestAgentRuntimeSandboxUsesCallerRuntimeClass(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					deployment := gvisorIstioRuntimeDeployment(t, chart, "sandbox-istio-sidecar.yaml", family, nil)
					require.NotNil(t, deployment.Spec.Template.Spec.RuntimeClassName)
					assert.Equal(t, "kata-mshv-vm-isolation", *deployment.Spec.Template.Spec.RuntimeClassName)
					assert.Equal(t, "false", deployment.Spec.Template.Annotations["sidecar.istio.io/inject"])
					assert.Equal(t, "false", deployment.Spec.Template.Labels["sidecar.istio.io/inject"])
				})
			}
		})
	}
}

// Same hand-authored istio-proxy/Sidecar CR/Service/NetworkPolicy/PeerAuthentication shape as the
// gVisor path - the mechanism genuinely doesn't care which flag turned sandboxing on.
func TestAgentRuntimeSandboxMeshSidecarShape(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					deployment := gvisorIstioRuntimeDeployment(t, chart, "sandbox-istio-sidecar.yaml", family, nil)
					require.Len(t, deployment.Spec.Template.Spec.InitContainers, 1)
					assert.Equal(t, "istio-proxy", deployment.Spec.Template.Spec.InitContainers[0].Name)
					assert.Equal(t, "istio", deployment.Spec.Template.Labels["security.istio.io/tlsMode"])

					sidecar, found := gvisorIstioRuntimeSidecar(t, chart, "sandbox-istio-sidecar.yaml", family)
					require.True(t, found, "expected a Sidecar CR for family %q", family)
					assert.Equal(t, chart.name+"-agent-runtime-"+family, sidecar.Spec.WorkloadSelector.Labels["app"])

					service := gvisorIstioRuntimeService(t, chart, "sandbox-istio-sidecar.yaml", family)
					assert.Equal(t, int32(18080), service.Spec.Ports[0].TargetPort.IntVal)

					policy := gvisorIstioNetworkPolicy(t, chart, "sandbox-istio-sidecar.yaml", family, nil)
					ingressPorts := networkPolicyIngressPorts(policy.Spec.Ingress)
					assert.Contains(t, ingressPorts, int32(18080))
					assert.NotContains(t, ingressPorts, int32(8090), "the app's own un-intercepted port must not be exposed")

					pas := gvisorIstioPeerAuthentications(t, chart, "sandbox-istio-sidecar.yaml", nil)
					match := findPeerAuthBySuffix(pas, "agent-runtime-"+family)
					require.NotNil(t, match, "expected a PeerAuthentication for family %q", family)
					assert.Equal(t, "STRICT", match.Spec.Mtls.Mode)
				})
			}
		})
	}
}

// The egress proxy's PERMISSIVE exception must track istio.meshSidecar.enabled under the generic
// sandbox path exactly as it does under gVisor: absent while the mesh sidecar gives the runtime a
// real client cert, present when it doesn't.
func TestAgentRuntimeSandboxEgressProxyPermissiveException(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("absent when meshSidecar is enabled", func(t *testing.T) {
				pas := gvisorIstioPeerAuthentications(t, chart, "sandbox-istio-sidecar.yaml", nil)
				egressProxyPA := findPeerAuthBySuffix(pas, "agent-egress-proxy")
				require.NotNil(t, egressProxyPA)
				assert.Equal(t, "STRICT", egressProxyPA.Spec.Mtls.Mode)
				assert.Empty(t, egressProxyPA.Spec.PortLevelMtls, "no PERMISSIVE exception should remain once the mesh sidecar gives the runtime a real client cert")
			})

			t.Run("present when meshSidecar is disabled", func(t *testing.T) {
				pas := gvisorIstioPeerAuthentications(t, chart, "sandbox-istio-sidecar.yaml", map[string]string{"istio.meshSidecar.enabled": "false"})
				egressProxyPA := findPeerAuthBySuffix(pas, "agent-egress-proxy")
				require.NotNil(t, egressProxyPA)
				require.NotEmpty(t, egressProxyPA.Spec.PortLevelMtls, "the PERMISSIVE exception must appear once the runtime is sandboxed with no mesh identity")
				assert.Equal(t, "PERMISSIVE", egressProxyPA.Spec.PortLevelMtls["3128"].Mode)
			})
		})
	}
}

// With agentRuntimeSandbox.enabled: false (and gvisor.enabled already false in the fixture), the
// runtime reverts to plain, unsandboxed standard injection - proving sonarqube.agentRuntime.sandboxed
// correctly returns false when neither flag is set.
func TestAgentRuntimeSandboxDisabledRevertsToStandardInjection(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := gvisorIstioRuntimeDeployment(t, chart, "sandbox-istio-sidecar.yaml", "hunter",
				map[string]string{"agentRuntimeSandbox.enabled": "false"})
			assert.Empty(t, deployment.Spec.Template.Spec.RuntimeClassName)
			assert.Empty(t, deployment.Spec.Template.Spec.InitContainers)
			assert.Equal(t, "true", deployment.Spec.Template.Annotations["sidecar.istio.io/inject"])
			assert.Equal(t, "true", deployment.Spec.Template.Labels["sidecar.istio.io/inject"])
		})
	}
}

// agentRuntimeSandbox.enabled=true with no runtimeClassName must fail closed rather than exclude
// the pod from injection while leaving it on no RuntimeClass at all.
func TestAgentRuntimeSandboxRequiresRuntimeClassName(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/sandbox-istio-sidecar.yaml"},
				SetValues:   map[string]string{"agentRuntimeSandbox.runtimeClassName": ""},
			}
			_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-runtime.yaml"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "agentRuntimeSandbox.enabled=true requires a non-empty agentRuntimeSandbox.runtimeClassName")
		})
	}
}
