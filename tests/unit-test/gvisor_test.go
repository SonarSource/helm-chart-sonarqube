package tests

import (
	"os"
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

// The gVisor feature is identical across the community/commercial (sonarqube) and Data Center
// (sonarqube-dce) charts, so every render assertion runs against both. Only the generated installer
// name differs (it derives from each chart's own sonarqube.fullname), so name checks assert a shared
// suffix + internal consistency rather than a hard-coded string.

type gvisorChartCase struct {
	name        string
	chartPath   string
	releaseName string
	baseOptions func() *helm.Options // fresh options carrying the chart's minimal base values
	fixturesDir string
}

func gvisorChartCases() []gvisorChartCase {
	return []gvisorChartCase{
		{
			name:        "sonarqube",
			chartPath:   chartPath,   // defined in sonarqube_schema_test.go
			releaseName: releaseName, // defined in sonarqube_schema_test.go
			baseOptions: newSQHelmOptions,
			fixturesDir: "test-cases-values/sonarqube",
		},
		{
			name:        "sonarqube-dce",
			chartPath:   dceChartPath,   // defined in sonarqube_dce_schema_test.go
			releaseName: dceReleaseName, // defined in sonarqube_dce_schema_test.go
			baseOptions: func() *helm.Options { return &helm.Options{Logger: logger.Discard} },
			fixturesDir: "test-cases-values/sonarqube-dce",
		},
	}
}

const gvisorInstallerSuffix = "-agentic-gvisor-installer"

// The gVisor template is intentionally duplicated verbatim across the two charts (the repo has no
// shared library chart, mirroring the existing mcp-pvc.yaml / mcp-service.yaml duplication). This
// guard fails if the two copies drift, so any change to one must be mirrored in the other.
func TestGvisorTemplateIdenticalAcrossCharts(t *testing.T) {
	sq, err := os.ReadFile(chartPath + "/templates/agentic-gvisor.yaml")
	require.NoError(t, err)
	dce, err := os.ReadFile(dceChartPath + "/templates/agentic-gvisor.yaml")
	require.NoError(t, err)
	assert.Equal(t, string(sq), string(dce),
		"charts/sonarqube and charts/sonarqube-dce templates/agentic-gvisor.yaml must stay "+
			"byte-identical — update both copies when changing the gVisor template")
}

// renderGvisor renders only the gVisor template. Helm still executes validation.yaml during the
// render, so a validation `fail` surfaces here even though the gVisor template is the show-only
// target (used by the negative cases below).
func renderGvisor(t *testing.T, c gvisorChartCase, fixture string) (string, error) {
	t.Helper()
	opts := c.baseOptions()
	opts.ValuesFiles = []string{c.fixturesDir + "/" + fixture}
	return helm.RenderTemplateE(t, opts, c.chartPath, c.releaseName, []string{"templates/agentic-gvisor.yaml"})
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

// The gvisor.enabled path always produces the RuntimeClass, wired to the values defaults, on both
// charts.
func TestGvisorRuntimeClass(t *testing.T) {
	for _, c := range gvisorChartCases() {
		t.Run(c.name, func(t *testing.T) {
			output, err := renderGvisor(t, c, "gvisor-installer.yaml")
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
		})
	}
}

// gvisor.enabled WITHOUT installer.enabled must render only the RuntimeClass — the privileged,
// host-mutating installer resources are a deliberately separate trust tier gated on installer.enabled.
func TestGvisorRuntimeClassOnlyWithoutInstaller(t *testing.T) {
	for _, c := range gvisorChartCases() {
		t.Run(c.name, func(t *testing.T) {
			output, err := renderGvisor(t, c, "gvisor-enabled-no-installer.yaml")
			require.NoError(t, err)

			assert.Contains(t, output, "kind: RuntimeClass")
			assert.NotContains(t, output, "kind: ServiceAccount")
			assert.NotContains(t, output, "kind: ClusterRole")
			assert.NotContains(t, output, "kind: ClusterRoleBinding")
			assert.NotContains(t, output, "kind: ConfigMap")
			assert.NotContains(t, output, "kind: DaemonSet")
		})
	}
}

// The installer only ever needs to get/patch its own Node to apply the readiness label — no broader
// cluster-scoped access — and every installer resource shares one fully-qualified name.
func TestGvisorInstallerRBAC(t *testing.T) {
	for _, c := range gvisorChartCases() {
		t.Run(c.name, func(t *testing.T) {
			output, err := renderGvisor(t, c, "gvisor-installer.yaml")
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
		})
	}
}

// The installer DaemonSet needs hostPID (so nsenter -t 1 reaches the host systemd bus), a privileged
// root container, the host root mount, and its scripts ConfigMap.
func TestGvisorInstallerDaemonSet(t *testing.T) {
	for _, c := range gvisorChartCases() {
		t.Run(c.name, func(t *testing.T) {
			output, err := renderGvisor(t, c, "gvisor-installer.yaml")
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
			assert.Equal(t, "debian:stable-slim", container.Image)
			require.NotNil(t, container.SecurityContext)
			require.NotNil(t, container.SecurityContext.Privileged)
			assert.True(t, *container.SecurityContext.Privileged)
			require.NotNil(t, container.SecurityContext.RunAsUser)
			assert.Equal(t, int64(0), *container.SecurityContext.RunAsUser)

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
		})
	}
}

// The install-gvisor.sh the installer runs must keep its safety nets: config is syntax-checked before
// any restart, a bounded health poll gates the readiness label, and every failure path rolls back and
// fails closed.
func TestGvisorInstallerSafetyNets(t *testing.T) {
	for _, c := range gvisorChartCases() {
		t.Run(c.name, func(t *testing.T) {
			output, err := renderGvisor(t, c, "gvisor-installer.yaml")
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
		})
	}
}

// The four invalid toggle combinations must each fail closed at template time with a specific message.
func TestGvisorValidationToggles(t *testing.T) {
	cases := []struct {
		name          string
		fixture       string
		expectedError string
	}{
		{
			name:          "installer without gvisor.enabled",
			fixture:       "gvisor-installer-without-gvisor.yaml",
			expectedError: "agenticHarness.gvisor.installer.enabled=true requires agenticHarness.gvisor.enabled=true",
		},
		{
			name:          "empty runtimeClassName",
			fixture:       "gvisor-empty-runtimeclassname.yaml",
			expectedError: "agenticHarness.gvisor.enabled=true requires a non-empty agenticHarness.gvisor.runtimeClassName",
		},
		{
			name:          "empty handler",
			fixture:       "gvisor-empty-handler.yaml",
			expectedError: "agenticHarness.gvisor.enabled=true requires a non-empty agenticHarness.gvisor.handler",
		},
		{
			name:          "installer with empty image repository",
			fixture:       "gvisor-installer-empty-image-repo.yaml",
			expectedError: "agenticHarness.gvisor.installer.enabled=true requires a non-empty agenticHarness.gvisor.installer.image.repository",
		},
	}
	for _, c := range gvisorChartCases() {
		for _, tc := range cases {
			t.Run(c.name+"/"+tc.name, func(t *testing.T) {
				_, err := renderGvisor(t, c, tc.fixture)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			})
		}
	}
}
