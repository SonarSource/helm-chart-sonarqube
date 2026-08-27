package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// With every agentic value at its default, the whole chart must render with zero agentic
// resources: the pack must be strictly opt-in, never appearing on an install that never asked
// for it.
func TestAgenticDisabledByDefaultRendersNoAgenticResources(t *testing.T) {
	agenticKinds := map[string]bool{
		"Deployment":   true,
		"Service":      true,
		"Secret":       true,
		"DaemonSet":    true,
		"RuntimeClass": true,
	}

	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{Logger: logger.Discard, SetValues: agentValidationBase(chart)}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{})
			require.NoError(t, err)

			for _, doc := range vortexDocs(output) {
				var meta struct {
					metav1.TypeMeta   `json:",inline"`
					metav1.ObjectMeta `json:"metadata,omitempty"`
				}
				helm.UnmarshalK8SYaml(t, doc, &meta)
				if !agenticKinds[meta.Kind] {
					continue
				}
				assert.False(t,
					strings.Contains(meta.Name, "-agent-") ||
						strings.HasSuffix(meta.Name, "-vortex") ||
						strings.Contains(meta.Name, "-gvisor-"),
					"unexpected agentic resource rendered with every agentic value at its default: %s %q", meta.Kind, meta.Name)
			}
		})
	}
}
