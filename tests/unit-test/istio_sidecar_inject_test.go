package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type istioInjectCase struct {
	name      string
	template  string
	setValues map[string]string
}

// renderWorkloadMeta renders one template off the gvisor-istio-sidecar fixture (already wired
// with agentOrchestrator/hunterAgent/remediationAgent/vortex/istio enabled) and returns the pod
// template's ObjectMeta (labels and annotations). Deployment and StatefulSet share the same
// Spec.Template.ObjectMeta shape, so unmarshalling either into appsv1.Deployment is safe (see
// agenticPodSpec).
func renderWorkloadMeta(t *testing.T, chart agentChart, template string, setValues map[string]string) metav1.ObjectMeta {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/gvisor-istio-sidecar.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{template})
	require.NoError(t, err)
	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var workload appsv1.Deployment
		helm.UnmarshalK8SYaml(t, doc, &workload)
		return workload.Spec.Template.ObjectMeta
	}
	t.Fatalf("%s rendered no workload", template)
	return metav1.ObjectMeta{}
}

// Every workload this chart puts under STRICT mTLS (peerauthentication.yaml) must also request
// its own sidecar explicitly - namespace-wide auto-injection is an external cluster policy this
// chart can't see, and without a sidecar STRICT mode doesn't degrade to "mTLS unenforced", it
// rejects all the now-plaintext-only traffic the pod can ever receive. Required as BOTH an
// annotation and a label: the sidecar-injector webhook's objectSelector is a LabelSelector and
// can only pre-filter on labels, never annotations (see sonarqube.istio.sidecarInjectAnnotation).
func TestIstioSidecarInjectAnnotation(t *testing.T) {
	shared := []istioInjectCase{
		{"agentOrchestrator", "templates/agent-orchestrator.yaml", nil},
		{"vortex", "templates/vortex.yaml", nil},
		{"agentEgressProxy", "templates/agent-egress-proxy.yaml", nil},
		{"mcp", "templates/mcp.yaml", map[string]string{"mcp.enabled": "true"}},
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			cases := append([]istioInjectCase{{"mainWorkload", chart.appTemplate, nil}}, shared...)
			if chart.name == "sonarqube-dce" {
				cases = append(cases, istioInjectCase{"searchNodes", "templates/sonarqube-search.yaml", nil})
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					meta := renderWorkloadMeta(t, chart, tc.template, tc.setValues)
					assert.Equal(t, "true", meta.Annotations["sidecar.istio.io/inject"])
					assert.Equal(t, "true", meta.Labels["sidecar.istio.io/inject"])
				})
			}
		})
	}
}

// istio.enabled=false must render no sidecar-inject annotation or label at all - it's a no-op
// outside the feature, not a default stance on injection.
func TestIstioSidecarInjectAnnotationOffWhenIstioDisabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			meta := renderWorkloadMeta(t, chart, "templates/agent-orchestrator.yaml", map[string]string{"istio.enabled": "false"})
			assert.NotContains(t, meta.Annotations, "sidecar.istio.io/inject")
			assert.NotContains(t, meta.Labels, "sidecar.istio.io/inject")
		})
	}
}

// A gVisor-sandboxed runtime keeps the explicit "false" it needs (istio-init can't run under
// runsc) rather than picking up "true" just because istio.enabled is also set - the two
// annotations are mutually exclusive per pod. Checked as both an annotation and a label, same
// reasoning as TestIstioSidecarInjectAnnotation.
func TestIstioSidecarInjectAnnotationGvisorRuntimeStaysExcluded(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := gvisorIstioRuntimeDeployment(t, chart, "gvisor-istio-sidecar-off.yaml", "hunter", nil)
			assert.Equal(t, "false", deployment.Spec.Template.Annotations["sidecar.istio.io/inject"])
			assert.Equal(t, "false", deployment.Spec.Template.Labels["sidecar.istio.io/inject"])
		})
	}
}

// With gVisor off the runtime gets standard injection, so it must ask for a sidecar like every
// other chart-owned workload - as both an annotation and a label.
func TestIstioSidecarInjectAnnotationRuntimeInjectedWithoutGvisor(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := gvisorIstioRuntimeDeployment(t, chart, "gvisor-istio-sidecar-off.yaml", "hunter",
				map[string]string{"gvisor.enabled": "false"})
			assert.Equal(t, "true", deployment.Spec.Template.Annotations["sidecar.istio.io/inject"])
			assert.Equal(t, "true", deployment.Spec.Template.Labels["sidecar.istio.io/inject"])
		})
	}
}
