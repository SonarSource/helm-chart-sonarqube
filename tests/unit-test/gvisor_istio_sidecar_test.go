package tests

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// Minimal local shapes for the two Istio CRDs this feature renders - there's no vendored Istio
// Go API in this module, but UnmarshalK8SYaml only needs JSON tags that line up, not a real
// client-go/Istio type.
type istioSidecarPort struct {
	Number   int32  `json:"number"`
	Protocol string `json:"protocol"`
	Name     string `json:"name"`
}

type istioWorkloadSelector struct {
	Labels map[string]string `json:"labels"`
}

type istioSidecar struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		WorkloadSelector istioWorkloadSelector `json:"workloadSelector"`
		Ingress          []struct {
			Port            istioSidecarPort `json:"port"`
			Bind            string           `json:"bind"`
			CaptureMode     string           `json:"captureMode"`
			DefaultEndpoint string           `json:"defaultEndpoint"`
		} `json:"ingress"`
		Egress []struct {
			Port        istioSidecarPort `json:"port"`
			Bind        string           `json:"bind"`
			CaptureMode string           `json:"captureMode"`
			Hosts       []string         `json:"hosts"`
		} `json:"egress"`
	} `json:"spec"`
}

type istioPeerAuthSelector struct {
	MatchLabels map[string]string `json:"matchLabels"`
}

type istioMtlsSettings struct {
	Mode string `json:"mode"`
}

type istioPeerAuthentication struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Selector      istioPeerAuthSelector `json:"selector"`
		Mtls          istioMtlsSettings     `json:"mtls"`
		PortLevelMtls map[string]struct {
			Mode string `json:"mode"`
		} `json:"portLevelMtls"`
	} `json:"spec"`
}

func renderGvisorIstioSidecar(t *testing.T, chart agentChart, fixture string, templates []string, setValues map[string]string) (string, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/" + fixture},
		SetValues:   setValues,
	}
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, templates)
}

// splitGvisorIstioDocs splits a multi-document render into its non-empty documents.
func splitGvisorIstioDocs(manifest string) []string {
	var docs []string
	for _, doc := range strings.Split(manifest, "\n---") {
		if strings.TrimSpace(doc) != "" {
			docs = append(docs, doc)
		}
	}
	return docs
}

func gvisorIstioRuntimeDeployment(t *testing.T, chart agentChart, fixture, family string, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	output, err := renderGvisorIstioSidecar(t, chart, fixture, []string{"templates/agent-runtime.yaml"}, setValues)
	require.NoError(t, err)
	for _, doc := range splitGvisorIstioDocs(output) {
		if !strings.Contains(doc, "kind: Deployment") {
			continue
		}
		var deployment appsv1.Deployment
		helm.UnmarshalK8SYaml(t, doc, &deployment)
		if deployment.Labels["sonarqube.agent/family"] == family {
			return deployment
		}
	}
	require.FailNowf(t, "no Deployment rendered", "family %q", family)
	return appsv1.Deployment{}
}

func gvisorIstioRuntimeService(t *testing.T, chart agentChart, fixture, family string) corev1.Service {
	t.Helper()
	output, err := renderGvisorIstioSidecar(t, chart, fixture, []string{"templates/agent-runtime-service.yaml"}, nil)
	require.NoError(t, err)
	for _, doc := range splitGvisorIstioDocs(output) {
		if !strings.Contains(doc, "kind: Service") {
			continue
		}
		var service corev1.Service
		helm.UnmarshalK8SYaml(t, doc, &service)
		if service.Labels["sonarqube.agent/family"] == family {
			return service
		}
	}
	require.FailNowf(t, "no Service rendered", "family %q", family)
	return corev1.Service{}
}

func gvisorIstioRuntimeSidecar(t *testing.T, chart agentChart, fixture, family string) (istioSidecar, bool) {
	t.Helper()
	// Helm's --show-only errors out (rather than returning empty output) when a template renders
	// no documents at all - the feature-off case here - so a render error means "no Sidecar",
	// not a test failure.
	output, err := renderGvisorIstioSidecar(t, chart, fixture, []string{"templates/agent-runtime-sidecar.yaml"}, nil)
	if err != nil {
		return istioSidecar{}, false
	}
	for _, doc := range splitGvisorIstioDocs(output) {
		if !strings.Contains(doc, "kind: Sidecar") {
			continue
		}
		var sidecar istioSidecar
		helm.UnmarshalK8SYaml(t, doc, &sidecar)
		if strings.HasSuffix(sidecar.Metadata.Name, "-agent-runtime-"+family) {
			return sidecar, true
		}
	}
	return istioSidecar{}, false
}

func gvisorIstioPeerAuthentications(t *testing.T, chart agentChart, fixture string, setValues map[string]string) []istioPeerAuthentication {
	t.Helper()
	output, err := renderGvisorIstioSidecar(t, chart, fixture, []string{"templates/peerauthentication.yaml"}, setValues)
	require.NoError(t, err)
	var out []istioPeerAuthentication
	for _, doc := range splitGvisorIstioDocs(output) {
		if !strings.Contains(doc, "kind: PeerAuthentication") {
			continue
		}
		var pa istioPeerAuthentication
		helm.UnmarshalK8SYaml(t, doc, &pa)
		out = append(out, pa)
	}
	return out
}

// findPeerAuthBySuffix returns the last rendered PeerAuthentication whose name ends in suffix, or
// nil if none match. Shared by every PeerAuthentication assertion in this file instead of each
// hand-rolling its own name-matching loop.
func findPeerAuthBySuffix(pas []istioPeerAuthentication, suffix string) *istioPeerAuthentication {
	var match *istioPeerAuthentication
	for i := range pas {
		if strings.HasSuffix(pas[i].Metadata.Name, suffix) {
			match = &pas[i]
		}
	}
	return match
}

func gvisorIstioNetworkPolicy(t *testing.T, chart agentChart, fixture, family string, setValues map[string]string) networkingv1.NetworkPolicy {
	t.Helper()
	output, err := renderGvisorIstioSidecar(t, chart, fixture, []string{"templates/agent-networkpolicy.yaml"}, setValues)
	require.NoError(t, err)
	for _, doc := range splitGvisorIstioDocs(output) {
		var policy networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, doc, &policy)
		if policy.Labels["sonarqube.agent/family"] == family {
			return policy
		}
	}
	require.FailNowf(t, "no NetworkPolicy rendered", "family %q", family)
	return networkingv1.NetworkPolicy{}
}

// networkPolicyIngressPorts flattens every port named across a NetworkPolicy's ingress rules.
func networkPolicyIngressPorts(rules []networkingv1.NetworkPolicyIngressRule) []int32 {
	var ports []int32
	for _, rule := range rules {
		for _, p := range rule.Ports {
			ports = append(ports, p.Port.IntVal)
		}
	}
	return ports
}

// networkPolicyEgressPorts flattens every port named across a NetworkPolicy's egress rules.
func networkPolicyEgressPorts(rules []networkingv1.NetworkPolicyEgressRule) []int32 {
	var ports []int32
	for _, rule := range rules {
		for _, p := range rule.Ports {
			ports = append(ports, p.Port.IntVal)
		}
	}
	return ports
}

// The hand-authored istio-proxy init container: native sidecar, no interception, and the app's
// probes rewritten to go through it rather than hitting the app port directly.
func TestGvisorIstioSidecarContainer(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					assertGvisorIstioSidecarContainer(t, chart, family)
				})
			}
		})
	}
}

func assertGvisorIstioSidecarContainer(t *testing.T, chart agentChart, family string) {
	t.Helper()
	deployment := gvisorIstioRuntimeDeployment(t, chart, "gvisor-istio-sidecar.yaml", family, nil)
	podSpec := deployment.Spec.Template.Spec

	assert.Equal(t, "istio", deployment.Spec.Template.Labels["security.istio.io/tlsMode"])

	// Standard sidecar injection stamps these itself; nothing injects this hand-authored
	// proxy, so without them the pod is invisible to Prometheus's own Kubernetes-SD scrape
	// config despite genuinely enforcing mTLS (confirmed missing on a live cluster).
	annotations := deployment.Spec.Template.Annotations
	assert.Equal(t, "true", annotations["prometheus.io/scrape"])
	assert.Equal(t, "15020", annotations["prometheus.io/port"])
	assert.Equal(t, "/stats/prometheus", annotations["prometheus.io/path"])

	require.Len(t, podSpec.InitContainers, 1)
	proxy := podSpec.InitContainers[0]
	assert.Equal(t, "istio-proxy", proxy.Name)
	require.NotNil(t, proxy.RestartPolicy)
	assert.Equal(t, corev1.ContainerRestartPolicyAlways, *proxy.RestartPolicy)
	assert.Equal(t, "registry.istio.io/release/proxyv2:1.30.4", proxy.Image)

	env := map[string]string{}
	for _, e := range proxy.Env {
		env[e.Name] = e.Value
	}
	assert.Equal(t, "NONE", env["ISTIO_META_INTERCEPTION_MODE"])
	assert.Equal(t, "[\n]", env["ISTIO_META_POD_PORTS"])
	assert.Equal(t, "{\"discoveryAddress\":\"istiod.istio-system.svc:15012\"}\n", env["PROXY_CONFIG"])
	assert.Equal(t, "agent-runtime", env["ISTIO_META_APP_CONTAINERS"])
	assert.Contains(t, env["ISTIO_META_WORKLOAD_NAME"], "agent-runtime-"+family)

	var probers map[string]struct {
		HTTPGet struct {
			Path string `json:"path"`
			Port int    `json:"port"`
		} `json:"httpGet"`
		TimeoutSeconds int `json:"timeoutSeconds"`
	}
	require.NoError(t, json.Unmarshal([]byte(env["ISTIO_KUBE_APP_PROBERS"]), &probers))
	require.Contains(t, probers, "/app-health/agent-runtime/readyz")
	require.Contains(t, probers, "/app-health/agent-runtime/livez")
	assert.Equal(t, "/readyz", probers["/app-health/agent-runtime/readyz"].HTTPGet.Path)
	assert.Equal(t, 8090, probers["/app-health/agent-runtime/readyz"].HTTPGet.Port)
	assert.Equal(t, "/livez", probers["/app-health/agent-runtime/livez"].HTTPGet.Path)

	mounts := agentVolumeMountsByName(proxy.VolumeMounts)
	for name, path := range map[string]string{
		"workload-socket":   "/var/run/secrets/workload-spiffe-uds",
		"credential-socket": "/var/run/secrets/credential-uds",
		"workload-certs":    "/var/run/secrets/workload-spiffe-credentials",
		"istiod-ca-cert":    "/var/run/secrets/istio",
		"istio-ca-crl":      "/var/run/secrets/istio/crl",
		"istio-data":        "/var/lib/istio/data",
		"istio-envoy":       "/etc/istio/proxy",
		"istio-token":       "/var/run/secrets/tokens",
		"istio-podinfo":     "/etc/istio/pod",
	} {
		require.Contains(t, mounts, name)
		assert.Equal(t, path, mounts[name].MountPath, "mount path for %q", name)
	}

	volumes := agentVolumesByName(podSpec.Volumes)
	for _, name := range []string{
		"workload-socket", "credential-socket", "workload-certs", "istiod-ca-cert",
		"istio-ca-crl", "istio-data", "istio-envoy", "istio-token", "istio-podinfo",
	} {
		assert.Contains(t, volumes, name)
	}

	container := deployment.Spec.Template.Spec.Containers[0]
	appEnv := map[string]string{}
	for _, e := range container.Env {
		appEnv[e.Name] = e.Value
	}
	assert.Equal(t, "http://127.0.0.1:3128", appEnv["HTTP_PROXY"])

	require.NotNil(t, container.ReadinessProbe)
	require.NotNil(t, container.ReadinessProbe.HTTPGet)
	assert.Equal(t, "/app-health/agent-runtime/readyz", container.ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, int32(15020), container.ReadinessProbe.HTTPGet.Port.IntVal)

	require.NotNil(t, container.LivenessProbe)
	require.NotNil(t, container.LivenessProbe.HTTPGet)
	assert.Equal(t, "/app-health/agent-runtime/livez", container.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, int32(15020), container.LivenessProbe.HTTPGet.Port.IntVal)
}

// istio.namespace must actually drive discoveryAddress/CA_ADDR - the default value asserted above
// is also pilot-agent's own hardcoded fallback, so pinning only the default can't catch a
// regression back to the bug this fixed (PROXY_CONFIG left empty, silently defaulting to
// istio-system regardless of istio.namespace).
func TestGvisorIstioSidecarContainerHonoursNamespace(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := gvisorIstioRuntimeDeployment(t, chart, "gvisor-istio-sidecar.yaml", "hunter", map[string]string{"istio.namespace": "custom-istio"})
			env := map[string]string{}
			for _, e := range deployment.Spec.Template.Spec.InitContainers[0].Env {
				env[e.Name] = e.Value
			}
			assert.Equal(t, "{\"discoveryAddress\":\"istiod.custom-istio.svc:15012\"}\n", env["PROXY_CONFIG"])
			assert.Equal(t, "istiod.custom-istio.svc:15012", env["CA_ADDR"])
		})
	}
}

// The Sidecar resource is what closes both hops for a NONE-mode proxy: an ingress listener on the
// mesh port forwarding to the app over loopback, and an egress listener scoped to the egress proxy
// alone.
func TestGvisorIstioSidecarResource(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					sidecar, found := gvisorIstioRuntimeSidecar(t, chart, "gvisor-istio-sidecar.yaml", family)
					require.True(t, found, "expected a Sidecar for family %q", family)

					// Istio's WorkloadSelector field is "labels", not the Kubernetes-selector-style
					// "matchLabels" PeerAuthentication.spec.selector uses - verified against the real
					// Sidecar CRD on a live cluster; the wrong key renders fine locally (no schema to
					// check against) but is rejected by the API server.
					assert.Equal(t, chart.name+"-agent-runtime-"+family, sidecar.Spec.WorkloadSelector.Labels["app"])

					require.Len(t, sidecar.Spec.Ingress, 1)
					ingress := sidecar.Spec.Ingress[0]
					assert.Equal(t, int32(18080), ingress.Port.Number)
					assert.Equal(t, "0.0.0.0", ingress.Bind)
					assert.Equal(t, "NONE", ingress.CaptureMode)
					assert.Equal(t, "127.0.0.1:8090", ingress.DefaultEndpoint)

					require.Len(t, sidecar.Spec.Egress, 1)
					egress := sidecar.Spec.Egress[0]
					assert.Equal(t, int32(3128), egress.Port.Number)
					assert.Equal(t, "127.0.0.1", egress.Bind)
					assert.Equal(t, "NONE", egress.CaptureMode)
					require.Len(t, egress.Hosts, 1)
					assert.Contains(t, egress.Hosts[0], "agent-egress-proxy")
				})
			}
		})
	}
}

// The Service must forward to the pod's own Envoy (meshPort), not the app's own un-intercepted
// port - the app-facing port number itself is unchanged, so the orchestrator needs no change.
func TestGvisorIstioSidecarServiceTargetPort(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					service := gvisorIstioRuntimeService(t, chart, "gvisor-istio-sidecar.yaml", family)
					require.Len(t, service.Spec.Ports, 1)
					assert.Equal(t, int32(8090), service.Spec.Ports[0].Port)
					assert.Equal(t, int32(18080), service.Spec.Ports[0].TargetPort.IntVal)
				})
			}
		})
	}
}

// Each enabled runtime family gets its own STRICT PeerAuthentication, and the egress proxy's
// PERMISSIVE exception is removed outright rather than merely narrowed.
func TestGvisorIstioSidecarPeerAuthentication(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			assertGvisorIstioSidecarPeerAuthentication(t, chart)
		})
	}
}

func assertGvisorIstioSidecarPeerAuthentication(t *testing.T, chart agentChart) {
	t.Helper()
	pas := gvisorIstioPeerAuthentications(t, chart, "gvisor-istio-sidecar.yaml", nil)

	for _, suffix := range []string{"agent-runtime-hunter", "agent-runtime-remediation"} {
		match := findPeerAuthBySuffix(pas, suffix)
		require.NotNil(t, match, "expected a PeerAuthentication ending in %q", suffix)
		assert.Equal(t, "STRICT", match.Spec.Mtls.Mode)
	}

	egressProxyPA := findPeerAuthBySuffix(pas, "agent-egress-proxy")
	require.NotNil(t, egressProxyPA, "expected the agent-egress-proxy PeerAuthentication")
	assert.Equal(t, "STRICT", egressProxyPA.Spec.Mtls.Mode)
	assert.Empty(t, egressProxyPA.Spec.PortLevelMtls, "portLevelMtls exception must be removed once the mesh sidecar is enabled")
}

// The Agent Egress Proxy's PERMISSIVE exception must apply only when the runtimes are actually
// excluded from injection (gVisor active, mesh sidecar off) - not whenever the mesh sidecar
// merely happens to be off. gvisor.enabled=false takes the runtimes out of that branch entirely:
// agent-runtime.yaml no longer opts them out of injection, so they get a standard sidecar and a
// real client cert, and the exception must not linger for them.
func TestGvisorIstioSidecarPeerAuthenticationPermissiveRequiresGvisor(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			pas := gvisorIstioPeerAuthentications(t, chart, "gvisor-istio-sidecar.yaml", map[string]string{"gvisor.enabled": "false"})

			egressProxyPA := findPeerAuthBySuffix(pas, "agent-egress-proxy")
			require.NotNil(t, egressProxyPA, "expected the agent-egress-proxy PeerAuthentication")
			assert.Equal(t, "STRICT", egressProxyPA.Spec.Mtls.Mode)
			assert.Empty(t, egressProxyPA.Spec.PortLevelMtls, "no PERMISSIVE exception should remain once gVisor is disabled - the runtimes get standard sidecar injection and a real client cert")
		})
	}
}

// A runtime that gets standard injection (gVisor off) must be under the same STRICT
// PeerAuthentication as every other injected workload, not left relying on the mesh's
// PERMISSIVE default.
func TestGvisorIstioSidecarPeerAuthenticationRuntimeWithoutGvisor(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			pas := gvisorIstioPeerAuthentications(t, chart, "gvisor-istio-sidecar-off.yaml", map[string]string{"gvisor.enabled": "false"})
			match := findPeerAuthBySuffix(pas, "agent-runtime-hunter")
			require.NotNil(t, match, "expected a PeerAuthentication for the standard-injected runtime")
			assert.Equal(t, "STRICT", match.Spec.Mtls.Mode)
		})
	}
}

// The runtime NetworkPolicy must expose only meshPort plus the proxy's own status ports on
// ingress - never the app's own, un-intercepted port - and must reach istiod on egress.
func TestGvisorIstioSidecarNetworkPolicy(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					assertGvisorIstioSidecarNetworkPolicy(t, chart, family)
				})
			}
		})
	}
}

func assertGvisorIstioSidecarNetworkPolicy(t *testing.T, chart agentChart, family string) {
	t.Helper()
	policy := gvisorIstioNetworkPolicy(t, chart, "gvisor-istio-sidecar.yaml", family, nil)

	ingressPorts := networkPolicyIngressPorts(policy.Spec.Ingress)
	assert.Contains(t, ingressPorts, int32(18080))
	assert.Contains(t, ingressPorts, int32(15020))
	assert.Contains(t, ingressPorts, int32(15021))
	assert.Contains(t, ingressPorts, int32(15090))
	assert.NotContains(t, ingressPorts, int32(8090), "the app's own un-intercepted port must not be exposed")

	egressPorts := networkPolicyEgressPorts(policy.Spec.Egress)
	assert.Contains(t, egressPorts, int32(15012), "expected an egress rule reaching istiod on 15012")
}

// The same sidecar status ports must be admitted on ingress for the standard-injection path
// (gVisor off), not only when the hand-authored gVisor mesh sidecar is enabled.
func TestGvisorIstioSidecarNetworkPolicyRuntimeWithoutGvisor(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			policy := gvisorIstioNetworkPolicy(t, chart, "gvisor-istio-sidecar-off.yaml", "hunter", map[string]string{"gvisor.enabled": "false"})
			ingressPorts := networkPolicyIngressPorts(policy.Spec.Ingress)
			assert.Contains(t, ingressPorts, int32(15020))
			assert.Contains(t, ingressPorts, int32(15021))
			assert.Contains(t, ingressPorts, int32(15090))
		})
	}
}

// istio.meshSidecar.enabled=false must restore today's behaviour exactly: no init container, no
// Sidecar, the named Service port, the Service-DNS HTTP_PROXY, and the egress proxy's PERMISSIVE
// exception.
func TestGvisorIstioSidecarOffPath(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := gvisorIstioRuntimeDeployment(t, chart, "gvisor-istio-sidecar-off.yaml", "hunter", nil)
			assert.Empty(t, deployment.Spec.Template.Spec.InitContainers)
			assert.NotContains(t, deployment.Spec.Template.Labels, "security.istio.io/tlsMode")
			assert.NotContains(t, deployment.Spec.Template.Annotations, "prometheus.io/scrape")

			container := deployment.Spec.Template.Spec.Containers[0]
			env := map[string]string{}
			for _, e := range container.Env {
				env[e.Name] = e.Value
			}
			assert.Contains(t, env["HTTP_PROXY"], "agent-egress-proxy")
			assert.NotEqual(t, "http://127.0.0.1:3128", env["HTTP_PROXY"])

			require.NotNil(t, container.ReadinessProbe.HTTPGet)
			assert.Equal(t, "http", container.ReadinessProbe.HTTPGet.Port.StrVal)

			service := gvisorIstioRuntimeService(t, chart, "gvisor-istio-sidecar-off.yaml", "hunter")
			assert.Equal(t, "http", service.Spec.Ports[0].TargetPort.StrVal)

			_, found := gvisorIstioRuntimeSidecar(t, chart, "gvisor-istio-sidecar-off.yaml", "hunter")
			assert.False(t, found, "no Sidecar should render when the feature is off")

			pas := gvisorIstioPeerAuthentications(t, chart, "gvisor-istio-sidecar-off.yaml", nil)
			egressProxyPA := findPeerAuthBySuffix(pas, "agent-egress-proxy")
			var sawRuntimePA bool
			for _, pa := range pas {
				if strings.Contains(pa.Metadata.Name, "agent-runtime-") {
					sawRuntimePA = true
				}
			}

			require.NotNil(t, egressProxyPA)
			require.NotEmpty(t, egressProxyPA.Spec.PortLevelMtls, "the PERMISSIVE exception must still be present when the feature is off")
			assert.Equal(t, "PERMISSIVE", egressProxyPA.Spec.PortLevelMtls["3128"].Mode)
			assert.False(t, sawRuntimePA, "no runtime PeerAuthentication should render when the feature is off")
		})
	}
}

// The hand-authored istio-proxy is a native sidecar (initContainers entry with restartPolicy:
// Always), which needs Kubernetes >= 1.29 - a pre-1.29 cluster must fail closed at render time
// rather than admit a Deployment whose pods never progress past Init.
func TestGvisorIstioSidecarRequiresKubernetes129(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/gvisor-istio-sidecar.yaml"},
			}
			_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-runtime.yaml"}, "--kube-version=1.28.0")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "istio.meshSidecar.enabled requires Kubernetes >= 1.29")
		})
	}
}

// meshPort colliding with an enabled runtime's own port must fail closed at render time rather
// than produce a silently broken listener.
func TestGvisorIstioSidecarPortConflict(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderGvisorIstioSidecar(t, chart, "gvisor-istio-sidecar-port-conflict.yaml", []string{"templates/agent-runtime.yaml"}, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "istio.meshSidecar.meshPort (8090) must differ from the hunter runtime port")
		})
	}
}

// The feature must stay off unless all three of istio.enabled, gvisor.enabled and
// istio.meshSidecar.enabled are true - any one missing must reproduce today's behaviour.
func TestGvisorIstioSidecarGatingMatrix(t *testing.T) {
	cases := []struct {
		name      string
		setValues map[string]string
	}{
		{"istio.enabled=false", map[string]string{"istio.enabled": "false"}},
		{"gvisor.enabled=false", map[string]string{"gvisor.enabled": "false"}},
		{"istio.meshSidecar.enabled=false", map[string]string{"istio.meshSidecar.enabled": "false"}},
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					deployment := gvisorIstioRuntimeDeployment(t, chart, "gvisor-istio-sidecar.yaml", "hunter", tc.setValues)
					assert.Empty(t, deployment.Spec.Template.Spec.InitContainers)
					assert.NotContains(t, deployment.Spec.Template.Labels, "security.istio.io/tlsMode")
				})
			}
		})
	}
}
