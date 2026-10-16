package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// renderHookTestPodCommand renders either the change-admin-password hook Job or the helm-test
// Pod off the istio-hook-test-pods fixture and returns the single container's command.
func renderHookTestPodCommand(t *testing.T, chart agentChart, template string, setValues map[string]string) []string {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/istio-hook-test-pods.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{template})
	require.NoError(t, err)
	if template == "templates/change-admin-password-hook.yaml" {
		var job batchv1.Job
		helm.UnmarshalK8SYaml(t, output, &job)
		require.Len(t, job.Spec.Template.Spec.Containers, 1)
		return job.Spec.Template.Spec.Containers[0].Command
	}
	var pod corev1.Pod
	helm.UnmarshalK8SYaml(t, output, &pod)
	require.Len(t, pod.Spec.Containers, 1)
	return pod.Spec.Containers[0].Command
}

var istioHookAndTestPodTemplates = []string{
	"templates/change-admin-password-hook.yaml",
	"templates/tests/sonarqube-test.yaml",
}

// Without a native sidecar (Istio >= 1.27 + Kubernetes >= 1.29), istio-proxy never exits on its
// own, so these short-lived pods must ask it to shut down (quitquitquit) once their own work is
// done - but must still exit with their own captured status, not silently report success
// regardless of what the real command did.
func TestIstioSidecarHookAndTestPodShutdownProxy(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tpl := range istioHookAndTestPodTemplates {
				t.Run(tpl, func(t *testing.T) {
					command := renderHookTestPodCommand(t, chart, tpl, nil)
					require.Len(t, command, 3)
					script := command[2]
					assert.Contains(t, script, "127.0.0.1:15020/quitquitquit")
					assert.True(t, strings.HasSuffix(strings.TrimSpace(script), "exit $rc"),
						"script must end by exiting with the captured status: %q", script)
				})
			}
		})
	}
}

// istio.enabled=false must not request any sidecar shutdown - there is no sidecar. The hook Job
// keeps its pre-existing shell wrapper (needed for its own retry loop regardless of Istio) and
// still exits with its real command's captured status; the test Pod has no such pre-existing
// wrapper, so it reverts to a plain, unwrapped curl exec entirely rather than carrying a shell
// layer that exists only to make room for the shutdown call.
func TestIstioSidecarHookAndTestPodNoShutdownWhenIstioDisabled(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("templates/change-admin-password-hook.yaml", func(t *testing.T) {
				command := renderHookTestPodCommand(t, chart, "templates/change-admin-password-hook.yaml", map[string]string{"istio.enabled": "false"})
				require.Len(t, command, 3)
				script := command[2]
				assert.NotContains(t, script, "quitquitquit")
				assert.True(t, strings.HasSuffix(strings.TrimSpace(script), "exit $rc"))
			})
			t.Run("templates/tests/sonarqube-test.yaml", func(t *testing.T) {
				command := renderHookTestPodCommand(t, chart, "templates/tests/sonarqube-test.yaml", map[string]string{"istio.enabled": "false"})
				assert.Equal(t, []string{"curl"}, command, "no shell wrapper needed once there's no shutdown call to make")
			})
		})
	}
}
