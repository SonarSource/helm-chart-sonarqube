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
func renderAgenticRuntime(t *testing.T, family string, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{"test-cases-values/sonarqube-dce/agentic-runtimes-enabled.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/agentic/runtime.yaml"})
	require.NoError(t, err)

	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" || !strings.Contains(doc, "kind: Deployment") {
			continue
		}
		var deployment appsv1.Deployment
		helm.UnmarshalK8SYaml(t, doc, &deployment)
		if deployment.Labels["sonarqube.agentic/family"] == family {
			return deployment
		}
	}
	require.FailNowf(t, "no Deployment rendered", "family %q", family)
	return appsv1.Deployment{}
}

// Neither runtime sets readOnlyRootFilesystem: the images need their home directory writable.
func TestAgenticRuntimeContainerSecurityContext(t *testing.T) {
	cases := []struct {
		family    string
		runAsUser int64
	}{
		{"hunter", 10001},
		{"remediation", 1000},
	}

	for _, c := range cases {
		t.Run(c.family, func(t *testing.T) {
			deployment := renderAgenticRuntime(t, c.family, nil)
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
func TestAgenticRuntimeResources(t *testing.T) {
	t.Run("hunter", func(t *testing.T) {
		deployment := renderAgenticRuntime(t, "hunter", nil)
		assert.Empty(t, deployment.Spec.Template.Spec.Containers[0].Resources.Requests)
		assert.Empty(t, deployment.Spec.Template.Spec.Containers[0].Resources.Limits)
	})

	t.Run("remediation", func(t *testing.T) {
		deployment := renderAgenticRuntime(t, "remediation", nil)
		container := deployment.Spec.Template.Spec.Containers[0]
		assert.Equal(t, "1", container.Resources.Requests.Cpu().String())
		assert.Equal(t, "2Gi", container.Resources.Requests.Memory().String())
		assert.Equal(t, "4", container.Resources.Limits.Cpu().String())
		assert.Equal(t, "8Gi", container.Resources.Limits.Memory().String())
	})
}
