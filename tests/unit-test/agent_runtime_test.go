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
)

// Both hunter and remediation are enabled in the fixture, so the render emits one Deployment per
// family; pick out the one for family.
func renderAgentRuntime(t *testing.T, family string, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{"test-cases-values/sonarqube-dce/agent-runtimes-enabled.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/agent-runtime.yaml"})
	require.NoError(t, err)

	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" || !strings.Contains(doc, "kind: Deployment") {
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

// renderAgentRuntimeService renders templates/agent-runtime-service.yaml and returns the
// Service for family - both hunter and remediation are enabled in the fixture, so it emits one
// Service document per family.
func renderAgentRuntimeService(t *testing.T, family string, setValues map[string]string) corev1.Service {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{"test-cases-values/sonarqube-dce/agent-runtimes-enabled.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/agent-runtime-service.yaml"})
	require.NoError(t, err)

	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" || !strings.Contains(doc, "kind: Service") {
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

// Off by default, so an existing install picks up nothing new until hunterAgent/remediationAgent
// is enabled.
func TestAgentRuntimeDisabledByDefault(t *testing.T) {
	for _, tpl := range []string{
		"templates/agent-runtime.yaml",
		"templates/agent-runtime-service.yaml",
	} {
		opts := &helm.Options{
			Logger:      logger.Discard,
			ValuesFiles: []string{"test-cases-values/sonarqube-dce/agent-all-disabled.yaml"},
		}
		output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{tpl})
		require.Error(t, err, "%s must render nothing when neither runtime is enabled", tpl)
		assert.Empty(t, strings.TrimSpace(output))
	}
}

// Each runtime's Service must select its own Deployment's pods, not the other family's - a label
// typo or a shared selector here silently yields no endpoints or cross-routes traffic.
func TestAgentRuntimeService(t *testing.T) {
	for _, family := range []string{"hunter", "remediation"} {
		t.Run(family, func(t *testing.T) {
			service := renderAgentRuntimeService(t, family, nil)
			podLabels := renderAgentRuntime(t, family, nil).Spec.Template.Labels

			require.NotEmpty(t, service.Spec.Selector)
			for k, v := range service.Spec.Selector {
				assert.Equal(t, v, podLabels[k], "Service selector %q must match the pod labels", k)
			}
		})
	}
}

// Each runtime can be scheduled apart from the SonarQube pods, and its own scheduling settings
// take precedence over the chart's global ones.
func TestAgentRuntimeSchedulingWinsOverGlobal(t *testing.T) {
	for _, family := range []string{"hunter", "remediation"} {
		t.Run(family, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{"test-cases-values/sonarqube-dce/agent-runtimes-global-scheduling.yaml"},
			}
			output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/agent-runtime.yaml"})
			require.NoError(t, err)

			var podSpec corev1.PodSpec
			for _, doc := range strings.Split(output, "\n---") {
				if strings.TrimSpace(doc) == "" || !strings.Contains(doc, "kind: Deployment") {
					continue
				}
				var deployment appsv1.Deployment
				helm.UnmarshalK8SYaml(t, doc, &deployment)
				if deployment.Labels["sonarqube.agent/family"] == family {
					podSpec = deployment.Spec.Template.Spec
				}
			}

			assert.Equal(t, map[string]string{family: "true"}, podSpec.NodeSelector)

			require.Len(t, podSpec.Tolerations, 1)
			assert.Equal(t, family, podSpec.Tolerations[0].Key)

			require.NotNil(t, podSpec.Affinity)
			require.NotNil(t, podSpec.Affinity.NodeAffinity)
			terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			require.Len(t, terms, 1)
			require.Len(t, terms[0].MatchExpressions, 1)
			assert.Equal(t, family, terms[0].MatchExpressions[0].Key)
		})
	}
}

// Both runtimes' probes default on; hunter's readiness endpoint differs from its liveness one.
func TestAgentRuntimeProbes(t *testing.T) {
	cases := []struct {
		family        string
		readinessPath string
		livenessPath  string
	}{
		{"hunter", "/readyz", "/health"},
		{"remediation", "/readyz", "/health"},
	}

	for _, c := range cases {
		t.Run(c.family, func(t *testing.T) {
			container := renderAgentRuntime(t, c.family, nil).Spec.Template.Spec.Containers[0]

			require.NotNil(t, container.ReadinessProbe)
			require.NotNil(t, container.ReadinessProbe.HTTPGet)
			assert.Equal(t, c.readinessPath, container.ReadinessProbe.HTTPGet.Path)

			require.NotNil(t, container.LivenessProbe)
			require.NotNil(t, container.LivenessProbe.HTTPGet)
			assert.Equal(t, c.livenessPath, container.LivenessProbe.HTTPGet.Path)
		})
	}
}

// Neither runtime sets readOnlyRootFilesystem: the images need their home directory writable.
func TestAgentRuntimeContainerSecurityContext(t *testing.T) {
	cases := []struct {
		family    string
		runAsUser int64
	}{
		{"hunter", 10001},
		{"remediation", 1000},
	}

	for _, c := range cases {
		t.Run(c.family, func(t *testing.T) {
			deployment := renderAgentRuntime(t, c.family, nil)
			require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
			container := deployment.Spec.Template.Spec.Containers[0]

			sc := container.SecurityContext
			require.NotNil(t, sc)
			require.NotNil(t, sc.AllowPrivilegeEscalation)
			assert.False(t, *sc.AllowPrivilegeEscalation)
			require.NotNil(t, sc.RunAsNonRoot)
			assert.True(t, *sc.RunAsNonRoot)
			require.NotNil(t, sc.RunAsUser)
			assert.Equal(t, c.runAsUser, *sc.RunAsUser)
			assert.Nil(t, sc.ReadOnlyRootFilesystem)
			require.NotNil(t, sc.Capabilities)
			assert.Contains(t, sc.Capabilities.Drop, corev1.Capability("ALL"))
		})
	}
}

// Only remediation ships sized resource defaults; hunter's sizing is out of scope for this
// ticket (SONAR-31656) - tracked separately.
func TestAgentRuntimeResources(t *testing.T) {
	t.Run("hunter", func(t *testing.T) {
		deployment := renderAgentRuntime(t, "hunter", nil)
		assert.Empty(t, deployment.Spec.Template.Spec.Containers[0].Resources.Requests)
		assert.Empty(t, deployment.Spec.Template.Spec.Containers[0].Resources.Limits)
	})

	t.Run("remediation", func(t *testing.T) {
		deployment := renderAgentRuntime(t, "remediation", nil)
		container := deployment.Spec.Template.Spec.Containers[0]
		assert.Equal(t, "1", container.Resources.Requests.Cpu().String())
		assert.Equal(t, "2Gi", container.Resources.Requests.Memory().String())
		assert.Equal(t, "4", container.Resources.Limits.Cpu().String())
		assert.Equal(t, "8Gi", container.Resources.Limits.Memory().String())
	})
}
