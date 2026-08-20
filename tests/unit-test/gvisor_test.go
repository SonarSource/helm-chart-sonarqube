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
	nodev1 "k8s.io/api/node/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const gvisorInstallerSuffix = "-agentic-gvisor-installer"

// renderGvisor renders only the gVisor template. Helm still executes validation.yaml during the
// render, so a validation `fail` surfaces here even though the gVisor template is the show-only
// target (used by the negative cases below).
func renderGvisor(t *testing.T, fixture string) (string, error) {
	t.Helper()
	opts := &helm.Options{Logger: logger.Discard, ValuesFiles: []string{"test-cases-values/sonarqube-dce/" + fixture}}
	return helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/agentic-gvisor.yaml"})
}

func splitGvisorDocs(manifest string) []string {
	var docs []string
	for _, doc := range strings.Split(manifest, "\n---") {
		if strings.TrimSpace(doc) != "" {
			docs = append(docs, doc)
		}
	}
	return docs
}

func gvisorDocKind(doc string) string {
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "kind:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "kind:"))
		}
	}
	return ""
}

// The gvisor.enabled path always produces the RuntimeClass, wired to the values defaults.
func TestGvisorRuntimeClass(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-installer.yaml")
	require.NoError(t, err)

	var runtimeClass nodev1.RuntimeClass
	for _, doc := range splitGvisorDocs(output) {
		if strings.Contains(doc, "kind: RuntimeClass") {
			helm.UnmarshalK8SYaml(t, doc, &runtimeClass)
		}
	}
	require.NotEmpty(t, runtimeClass.Name)
	assert.Equal(t, "gvisor", runtimeClass.Name)
	assert.Equal(t, "runsc", runtimeClass.Handler)
	require.NotNil(t, runtimeClass.Scheduling)
	assert.Equal(t, map[string]string{"gvisor.enabled": "true"}, runtimeClass.Scheduling.NodeSelector)
}

// gvisor.enabled/installer.enabled default to true but only take effect when at least one of
// hunterAgent.enabled / remediationAgent.enabled is also true (SONAR-31689) — an install with
// neither agent enabled must render nothing.
func TestGvisorDefaultRequiresAnAgent(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-default.yaml")
	require.Error(t, err, "agentic-gvisor.yaml must render nothing without hunterAgent/remediationAgent enabled")
	assert.Empty(t, strings.TrimSpace(output))
}

// With hunterAgent enabled, gVisor and its installer follow their own true defaults.
func TestGvisorFollowsHunterAgentByDefault(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-hunter-only.yaml")
	require.NoError(t, err)

	var runtimeClass nodev1.RuntimeClass
	for _, doc := range splitGvisorDocs(output) {
		if strings.Contains(doc, "kind: RuntimeClass") {
			helm.UnmarshalK8SYaml(t, doc, &runtimeClass)
		}
	}
	assert.Equal(t, "gvisor", runtimeClass.Name)
	assert.Equal(t, "runsc", runtimeClass.Handler)
	assert.Contains(t, output, "kind: DaemonSet")
	assert.Contains(t, output, "kind: ServiceAccount")
	assert.Contains(t, output, "kind: ClusterRole")
	assert.Contains(t, output, "kind: ConfigMap")
}

// remediationAgent alone (no hunterAgent) must also bring gVisor up by default — the render
// condition is an OR across both agents, not hunter-specific (SONAR-31689).
func TestGvisorFollowsRemediationAgentByDefault(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-remediation-only.yaml")
	require.NoError(t, err)

	var runtimeClass nodev1.RuntimeClass
	for _, doc := range splitGvisorDocs(output) {
		if strings.Contains(doc, "kind: RuntimeClass") {
			helm.UnmarshalK8SYaml(t, doc, &runtimeClass)
		}
	}
	assert.Equal(t, "gvisor", runtimeClass.Name)
	assert.Equal(t, "runsc", runtimeClass.Handler)
	assert.Contains(t, output, "kind: DaemonSet")
	assert.Contains(t, output, "kind: ServiceAccount")
	assert.Contains(t, output, "kind: ClusterRole")
	assert.Contains(t, output, "kind: ConfigMap")
}

// off/off/off: no agent enabled, gvisor and installer both explicitly false, must render nothing
// (SONAR-31689).
func TestGvisorAllOffExplicit(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-all-off-explicit.yaml")
	require.Error(t, err)
	assert.Empty(t, strings.TrimSpace(output))
}

// on + off = off: hunterAgent being enabled does not force gVisor on — an explicit
// gvisor.enabled=false still renders nothing at all (SONAR-31689).
func TestGvisorExplicitDisableOverridesAgent(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-disabled-hunter-enabled.yaml")
	require.Error(t, err)
	assert.Empty(t, strings.TrimSpace(output))
}

// The opt-out: explicitly setting enabled=false must render nothing at all, not just skip the
// installer.
func TestGvisorDisabledRendersNothing(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-disabled.yaml")
	require.Error(t, err, "agentic-gvisor.yaml must render nothing when gvisor.enabled is false")
	assert.Empty(t, strings.TrimSpace(output))
}

// gvisor.enabled WITHOUT installer.enabled must render only the RuntimeClass — the privileged,
// host-mutating installer resources are a deliberately separate trust tier gated on installer.enabled.
func TestGvisorRuntimeClassOnlyWithoutInstaller(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-enabled-no-installer.yaml")
	require.NoError(t, err)

	assert.Contains(t, output, "kind: RuntimeClass")
	assert.NotContains(t, output, "kind: ServiceAccount")
	assert.NotContains(t, output, "kind: ClusterRole")
	assert.NotContains(t, output, "kind: ClusterRoleBinding")
	assert.NotContains(t, output, "kind: ConfigMap")
	assert.NotContains(t, output, "kind: DaemonSet")
}

// The installer only ever needs to get/patch its own Node to apply the readiness label — no broader
// cluster-scoped access — and every installer resource shares one fully-qualified name.
func TestGvisorInstallerRBAC(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-installer.yaml")
	require.NoError(t, err)

	var serviceAccount corev1.ServiceAccount
	var clusterRole rbacv1.ClusterRole
	var clusterRoleBinding rbacv1.ClusterRoleBinding
	for _, doc := range splitGvisorDocs(output) {
		switch gvisorDocKind(doc) {
		case "ServiceAccount":
			helm.UnmarshalK8SYaml(t, doc, &serviceAccount)
		case "ClusterRole":
			helm.UnmarshalK8SYaml(t, doc, &clusterRole)
		case "ClusterRoleBinding":
			helm.UnmarshalK8SYaml(t, doc, &clusterRoleBinding)
		}
	}

	require.NotEmpty(t, serviceAccount.Name)
	assert.True(t, strings.HasSuffix(serviceAccount.Name, gvisorInstallerSuffix),
		"service account %q should end with %q", serviceAccount.Name, gvisorInstallerSuffix)

	require.NotEmpty(t, clusterRole.Name)
	assert.Equal(t, serviceAccount.Name, clusterRole.Name)
	require.Len(t, clusterRole.Rules, 1)
	assert.Equal(t, []string{""}, clusterRole.Rules[0].APIGroups)
	assert.Equal(t, []string{"nodes"}, clusterRole.Rules[0].Resources)
	assert.ElementsMatch(t, []string{"get", "patch"}, clusterRole.Rules[0].Verbs)

	require.NotEmpty(t, clusterRoleBinding.Name)
	assert.Equal(t, serviceAccount.Name, clusterRoleBinding.RoleRef.Name)
	assert.Equal(t, "ClusterRole", clusterRoleBinding.RoleRef.Kind)
	require.Len(t, clusterRoleBinding.Subjects, 1)
	assert.Equal(t, "ServiceAccount", clusterRoleBinding.Subjects[0].Kind)
	assert.Equal(t, serviceAccount.Name, clusterRoleBinding.Subjects[0].Name)
}

// The installer DaemonSet needs hostPID (so nsenter -t 1 reaches the host systemd bus), a privileged
// root container, the host root mount, and its scripts ConfigMap.
func TestGvisorInstallerDaemonSet(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-installer.yaml")
	require.NoError(t, err)

	var daemonSet appsv1.DaemonSet
	for _, doc := range splitGvisorDocs(output) {
		if strings.Contains(doc, "kind: DaemonSet") {
			helm.UnmarshalK8SYaml(t, doc, &daemonSet)
		}
	}
	require.NotEmpty(t, daemonSet.Name)

	podSpec := daemonSet.Spec.Template.Spec
	assert.True(t, podSpec.HostPID)
	assert.True(t, strings.HasSuffix(podSpec.ServiceAccountName, gvisorInstallerSuffix))
	require.Len(t, podSpec.Tolerations, 1)
	assert.Equal(t, corev1.TolerationOpExists, podSpec.Tolerations[0].Operator)

	require.Len(t, podSpec.Containers, 1)
	container := podSpec.Containers[0]
	assert.Equal(t, "debian@sha256:1710bde34461551a19a47c787885ec9ad7058d9a5bead2affb8d088fa2f8502b", container.Image)
	require.NotNil(t, container.SecurityContext)
	require.NotNil(t, container.SecurityContext.Privileged)
	assert.True(t, *container.SecurityContext.Privileged)
	require.NotNil(t, container.SecurityContext.RunAsUser)
	assert.Equal(t, int64(0), *container.SecurityContext.RunAsUser)
	assert.Equal(t, "100m", container.Resources.Requests.Cpu().String())
	assert.Equal(t, "500m", container.Resources.Limits.Cpu().String())

	volumes := map[string]corev1.Volume{}
	for _, v := range podSpec.Volumes {
		volumes[v.Name] = v
	}
	require.Contains(t, volumes, "host-root")
	require.NotNil(t, volumes["host-root"].HostPath)
	assert.Equal(t, "/", volumes["host-root"].HostPath.Path)
	require.Contains(t, volumes, "scripts")
	require.NotNil(t, volumes["scripts"].ConfigMap)
	assert.True(t, strings.HasSuffix(volumes["scripts"].ConfigMap.Name, gvisorInstallerSuffix))
}

// Clearing installer.image.digest falls back to repository:tag (SONAR-31656).
func TestGvisorInstallerImageDigestFallback(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-installer-tag-fallback.yaml")
	require.NoError(t, err)

	var daemonSet appsv1.DaemonSet
	for _, doc := range splitGvisorDocs(output) {
		if strings.Contains(doc, "kind: DaemonSet") {
			helm.UnmarshalK8SYaml(t, doc, &daemonSet)
		}
	}
	require.Len(t, daemonSet.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "debian:stable-slim", daemonSet.Spec.Template.Spec.Containers[0].Image)
}

// The install-gvisor.sh the installer runs must keep its safety nets: config is syntax-checked before
// any restart, a bounded health poll gates the readiness label, and every failure path rolls back and
// fails closed.
func TestGvisorInstallerSafetyNets(t *testing.T) {
	output, err := renderGvisor(t, "gvisor-installer.yaml")
	require.NoError(t, err)

	var configMap corev1.ConfigMap
	for _, doc := range splitGvisorDocs(output) {
		if strings.Contains(doc, "kind: ConfigMap") {
			helm.UnmarshalK8SYaml(t, doc, &configMap)
		}
	}
	require.NotEmpty(t, configMap.Name)

	script, ok := configMap.Data["install-gvisor.sh"]
	require.True(t, ok, "expected an install-gvisor.sh key in the installer ConfigMap")

	assert.Contains(t, script, "containerd --config \"${CONTAINERD_CONFIG}\" config dump")
	assert.Contains(t, script, "wait_for_containerd_healthy")
	assert.Contains(t, script, "systemctl is-active --quiet containerd")
	assert.Contains(t, script, "CONTAINERD_BACKUP")
	assert.Contains(t, script, "rollback_config")
	assert.Contains(t, script, "rollback_and_recover")
	assert.Contains(t, script, "CRITICAL")
}

// The three invalid toggle combinations must each fail closed at template time with a specific
// message. (installer.enabled without gvisor.enabled is intentionally NOT here: gvisor.enabled=false
// renders nothing, so the leftover installer flag is inert and must not fail — see validation.yaml.)
func TestGvisorValidationToggles(t *testing.T) {
	cases := []struct {
		name          string
		fixture       string
		expectedError string
	}{
		{
			name:          "empty runtimeClassName",
			fixture:       "gvisor-empty-runtimeclassname.yaml",
			expectedError: "gvisor.enabled=true requires a non-empty gvisor.runtimeClassName",
		},
		{
			name:          "empty handler",
			fixture:       "gvisor-empty-handler.yaml",
			expectedError: "gvisor.enabled=true requires a non-empty gvisor.handler",
		},
		{
			name:          "installer with empty image repository",
			fixture:       "gvisor-installer-empty-image-repo.yaml",
			expectedError: "gvisor.installer.enabled=true requires a non-empty gvisor.installer.image.repository",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := renderGvisor(t, tc.fixture)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}
