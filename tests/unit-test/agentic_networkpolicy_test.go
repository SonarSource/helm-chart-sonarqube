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

func renderAgenticRuntimeNetworkPolicy(t *testing.T, setValues map[string]string) (string, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{"test-cases-values/sonarqube-dce/agentic-networkpolicy-enabled.yaml"},
		SetValues:   setValues,
	}
	return helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/agentic-networkpolicy.yaml"})
}

// Both hunter and remediation are enabled in the fixture, so the render emits one NetworkPolicy
// document per family; pick out the one for family.
func runtimeNetworkPolicy(t *testing.T, family string, setValues map[string]string) networkingv1.NetworkPolicy {
	t.Helper()
	output, err := renderAgenticRuntimeNetworkPolicy(t, setValues)
	require.NoError(t, err)

	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var policy networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, doc, &policy)
		if policy.Labels["sonarqube.agentic/family"] == family {
			return policy
		}
	}
	require.FailNowf(t, "no NetworkPolicy rendered", "family %q", family)
	return networkingv1.NetworkPolicy{}
}

// A runtime reads/writes job artifacts directly against object storage via presigned URLs, so its
// NetworkPolicy needs its own egress to reach it. The chart can't resolve
// orchestrator.storage to a peer on its own, so egressAllow must cover it - this is
// documented on <hunterAgent|remediationAgent>.egressAllow in values.yaml, not enforced by the
// render (SONAR-31525).
func TestAgenticRuntimeNetworkPolicyEgressAllow(t *testing.T) {
	valuesKey := map[string]string{"hunter": "hunterAgent", "remediation": "remediationAgent"}
	for _, family := range []string{"hunter", "remediation"} {
		t.Run(family+": empty egressAllow renders no extra egress rule", func(t *testing.T) {
			policy := runtimeNetworkPolicy(t, family, nil)
			require.Len(t, policy.Spec.Egress, 2, "DNS and the orchestrator only")
		})

		t.Run(family+": egressAllow entries render as given", func(t *testing.T) {
			policy := runtimeNetworkPolicy(t, family, map[string]string{
				valuesKey[family] + ".egressAllow[0].cidr":              "0.0.0.0/0",
				valuesKey[family] + ".egressAllow[0].ports[0].port":     "443",
				valuesKey[family] + ".egressAllow[0].ports[0].protocol": "TCP",
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
}
