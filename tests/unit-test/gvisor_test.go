package tests

import (
	"os"
	"os/exec"
	"path/filepath"
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

const gvisorInstallerSuffix = "-gvisor-installer"

// renderGvisor renders only the gVisor template. Helm still executes validation.yaml during the
// render, so a validation `fail` surfaces here even though the gVisor template is the show-only
// target (used by the negative cases below).
func renderGvisor(t *testing.T, chart agentChart, fixture string) (string, error) {
	t.Helper()
	opts := &helm.Options{Logger: logger.Discard, ValuesFiles: []string{chart.valuesDir + "/" + fixture}}
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/gvisor.yaml"})
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

// gvisorInstallerScript renders the installer fixture and returns the named script from its ConfigMap.
func gvisorInstallerScript(t *testing.T, chart agentChart, key string) string {
	t.Helper()
	output, err := renderGvisor(t, chart, "gvisor-installer.yaml")
	require.NoError(t, err)

	var configMap corev1.ConfigMap
	for _, doc := range splitGvisorDocs(output) {
		if strings.Contains(doc, "kind: ConfigMap") {
			helm.UnmarshalK8SYaml(t, doc, &configMap)
		}
	}
	require.NotEmpty(t, configMap.Name)

	script, ok := configMap.Data[key]
	require.True(t, ok, "expected a %s key in the installer ConfigMap", key)
	return script
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
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-installer.yaml")
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

// gvisor.enabled/installer.enabled default to true but only take effect when at least one of
// hunterAgent.enabled / remediationAgent.enabled is also true (SONAR-31689) — an install with
// neither agent enabled must render nothing.
func TestGvisorDefaultRequiresAnAgent(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-default.yaml")
			require.Error(t, err, "gvisor.yaml must render nothing without hunterAgent/remediationAgent enabled")
			assert.Empty(t, strings.TrimSpace(output))
		})
	}
}

// With hunterAgent enabled, gVisor and its installer follow their own true defaults.
func TestGvisorFollowsHunterAgentByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-hunter-only.yaml")
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
		})
	}
}

// remediationAgent alone (no hunterAgent) must also bring gVisor up by default — the render
// condition is an OR across both agents, not hunter-specific (SONAR-31689).
func TestGvisorFollowsRemediationAgentByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-remediation-only.yaml")
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
		})
	}
}

// gvisor and installer both explicitly false, no agent enabled either: must render nothing.
func TestGvisorAllOffExplicit(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-all-off-explicit.yaml")
			require.Error(t, err)
			assert.Empty(t, strings.TrimSpace(output))
		})
	}
}

// hunterAgent being enabled has no bearing on gVisor's own toggle — an explicit
// gvisor.enabled=false still renders nothing at all.
func TestGvisorExplicitDisableOverridesAgent(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-disabled-hunter-enabled.yaml")
			require.Error(t, err)
			assert.Empty(t, strings.TrimSpace(output))
		})
	}
}

// The opt-out: explicitly setting enabled=false must render nothing at all, not just skip the
// installer.
func TestGvisorDisabledRendersNothing(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-disabled.yaml")
			require.Error(t, err, "gvisor.yaml must render nothing when gvisor.enabled is false")
			assert.Empty(t, strings.TrimSpace(output))
		})
	}
}

// gvisor.enabled WITHOUT installer.enabled must render only the RuntimeClass — the privileged,
// host-mutating installer resources are a deliberately separate trust tier gated on installer.enabled.
func TestGvisorRuntimeClassOnlyWithoutInstaller(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-enabled-no-installer.yaml")
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
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-installer.yaml")
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
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-installer.yaml")
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
			assert.Equal(t, "alpine@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf", container.Image)
			// alpine has no bash: run.sh is POSIX sh, the host scripts use the node's bash through chroot.
			assert.Equal(t, []string{"/bin/sh", "/scripts/run.sh"}, container.Command)
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
		})
	}
}

// Clearing installer.image.digest falls back to repository:tag (SONAR-31656).
func TestGvisorInstallerImageDigestFallback(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderGvisor(t, chart, "gvisor-installer-tag-fallback.yaml")
			require.NoError(t, err)

			var daemonSet appsv1.DaemonSet
			for _, doc := range splitGvisorDocs(output) {
				if strings.Contains(doc, "kind: DaemonSet") {
					helm.UnmarshalK8SYaml(t, doc, &daemonSet)
				}
			}
			require.Len(t, daemonSet.Spec.Template.Spec.Containers, 1)
			assert.Equal(t, "alpine:3.24.4", daemonSet.Spec.Template.Spec.Containers[0].Image)
		})
	}
}

// The install-gvisor.sh the installer runs must keep its safety nets: config is syntax-checked before
// any restart, a bounded health poll gates the readiness label, and every failure path rolls back and
// fails closed.
func TestGvisorInstallerSafetyNets(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			script := gvisorInstallerScript(t, chart, "install-gvisor.sh")

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

// A pre-flight check must skip the whole install, before touching anything on disk, when containerd
// already exposes the target handler (e.g. GKE Sandbox's own runsc) (SONAR-31721).
func TestGvisorInstallerPreflightSkipsExistingRuntime(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			script := gvisorInstallerScript(t, chart, "install-gvisor.sh")

			preflightIdx := strings.Index(script, "skipping installation")
			downloadIdx := strings.Index(script, "already present, skipping download")
			require.NotEqual(t, -1, preflightIdx, "expected a pre-flight skip check in install-gvisor.sh")
			require.NotEqual(t, -1, downloadIdx)
			assert.Less(t, preflightIdx, downloadIdx, "pre-flight check must run before the download step")
			assert.Contains(t, script, "crictl info")
			assert.Contains(t, script, `"${RUNSC_HANDLER}"`)
		})
	}
}

// The installer DaemonSet must roll when its scripts can change (chart version or gvisor.* values),
// so an upgrade replaces the running pods with ones that run the new scripts. User-supplied pod
// annotations must still be rendered next to the checksum (SONAR-33343).
func TestGvisorInstallerRollsOnScriptChange(t *testing.T) {
	scriptsChecksum := func(t *testing.T, chart agentChart, setValues map[string]string) string {
		t.Helper()
		opts := &helm.Options{
			Logger:      logger.Discard,
			ValuesFiles: []string{chart.valuesDir + "/gvisor-installer.yaml"},
			SetValues:   setValues,
		}
		output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/gvisor.yaml"})
		require.NoError(t, err)

		var daemonSet appsv1.DaemonSet
		for _, doc := range splitGvisorDocs(output) {
			if strings.Contains(doc, "kind: DaemonSet") {
				helm.UnmarshalK8SYaml(t, doc, &daemonSet)
			}
		}
		require.NotEmpty(t, daemonSet.Name)
		checksum := daemonSet.Spec.Template.Annotations["checksum/scripts"]
		require.NotEmpty(t, checksum, "expected a checksum/scripts annotation on the installer pod template")
		return checksum
	}

	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			base := scriptsChecksum(t, chart, nil)
			assert.Equal(t, base, scriptsChecksum(t, chart, nil), "checksum must be stable across renders")
			assert.NotEqual(t, base, scriptsChecksum(t, chart, map[string]string{"gvisor.installer.runscVersion": "20990101"}),
				"checksum must change when the gvisor values change")

			output, err := helm.RenderTemplateE(t, &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/gvisor-installer.yaml"},
				SetValues:   map[string]string{"gvisor.installer.annotations.example\\.com/owner": "platform"},
			}, chart.path, chart.release, []string{"templates/gvisor.yaml"})
			require.NoError(t, err)
			var daemonSet appsv1.DaemonSet
			for _, doc := range splitGvisorDocs(output) {
				if strings.Contains(doc, "kind: DaemonSet") {
					helm.UnmarshalK8SYaml(t, doc, &daemonSet)
				}
			}
			assert.Equal(t, "platform", daemonSet.Spec.Template.Annotations["example.com/owner"])
			assert.NotEmpty(t, daemonSet.Spec.Template.Annotations["checksum/scripts"])
		})
	}
}

// run.sh must clean up /host/tmp/gvisor-sa on every exit path: an EXIT trap registered before the
// install step, plus an explicit removal after labelling and before it execs into the idle loop
// (SONAR-33336).
func TestGvisorInstallerCleansUpAfterLabelling(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			script := gvisorInstallerScript(t, chart, "run.sh")

			trapIdx := strings.Index(script, "trap 'rm -rf /host/tmp/gvisor-sa \"${DOWNLOAD_DIR}\"' EXIT")
			installIdx := strings.Index(script, "chroot /host /bin/bash /opt/gvisor/install-gvisor.sh")
			require.NotEqual(t, -1, trapIdx, "expected run.sh to register a cleanup EXIT trap")
			require.NotEqual(t, -1, installIdx)
			assert.Less(t, trapIdx, installIdx, "cleanup EXIT trap must be registered before the install step")

			labelIdx := strings.Index(script, "chroot /host /bin/bash /opt/gvisor/label-node.sh")
			execIdx := strings.Index(script, "exec tail -f /dev/null")
			require.NotEqual(t, -1, labelIdx, "expected run.sh to label the node")
			require.NotEqual(t, -1, execIdx)
			cleanupIdx := strings.Index(script[labelIdx:execIdx], "\nrm -rf /host/tmp/gvisor-sa\n")
			assert.NotEqual(t, -1, cleanupIdx, "run.sh must remove /host/tmp/gvisor-sa after labelling and before exec")
		})
	}
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
		{
			name:          "runtime nodeSelector conflicts with gvisor.nodeSelector",
			fixture:       "gvisor-nodeselector-conflict.yaml",
			expectedError: `hunterAgent's effective nodeSelector conflicts with gvisor.nodeSelector on key "gvisor.enabled" ("false" vs "true")`,
		},
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					_, err := renderGvisor(t, chart, tc.fixture)
					require.Error(t, err)
					assert.Contains(t, err.Error(), tc.expectedError)
				})
			}
		})
	}
}

// renderGvisorScripts renders the installer fixture with extra values and returns its ConfigMap scripts.
func renderGvisorScripts(t *testing.T, chart agentChart, setValues, setStrValues map[string]string) (map[string]string, error) {
	t.Helper()
	output, err := helm.RenderTemplateE(t, &helm.Options{
		Logger:       logger.Discard,
		ValuesFiles:  []string{chart.valuesDir + "/gvisor-installer.yaml"},
		SetValues:    setValues,
		SetStrValues: setStrValues,
	}, chart.path, chart.release, []string{"templates/gvisor.yaml"})
	if err != nil {
		return nil, err
	}
	var configMap corev1.ConfigMap
	for _, doc := range splitGvisorDocs(output) {
		if strings.Contains(doc, "kind: ConfigMap") {
			helm.UnmarshalK8SYaml(t, doc, &configMap)
		}
	}
	require.NotEmpty(t, configMap.Name)
	return configMap.Data, nil
}

// From release 20260721, gVisor ships gvisor.tar.bz2 (with gvisor-bin/, which runsc needs next to it);
// from 20260831 it no longer ships the separate files (SONAR-33332). The host downloads and verifies
// the archive, the pod unpacks it, then the host configures containerd.
func TestGvisorInstallerArchiveLayout(t *testing.T) {
	for _, chart := range agentCharts {
		for _, release := range []string{"", "20260721", "20261116.0"} {
			t.Run(chart.name+"/"+release, func(t *testing.T) {
				setStr := map[string]string{}
				if release != "" {
					setStr["gvisor.installer.runscVersion"] = release
				}
				scripts, err := renderGvisorScripts(t, chart, nil, setStr)
				require.NoError(t, err)

				install := scripts["install-gvisor.sh"]
				assert.Contains(t, install, `for f in gvisor.tar.bz2 gvisor.tar.bz2.sha512; do`)
				assert.Contains(t, install, "sha512sum -c gvisor.tar.bz2.sha512")
				assert.NotContains(t, install, "containerd-shim-runsc-v1.sha512")
				assert.Contains(t, install, `if [[ "${PHASE}" == fetch ]]; then`)
				if release != "" {
					assert.Contains(t, install, `GVISOR_RELEASE="`+release+`"`)
				}

				run := scripts["run.sh"]
				assert.True(t, strings.HasPrefix(run, "#!/bin/sh\n"), "run.sh runs on the installer image's sh")
				assert.NotContains(t, run, "pipefail", "dash (Debian bookworm, Ubuntu 24.04 /bin/sh) rejects set -o pipefail")
				fetchIdx := strings.Index(run, "install-gvisor.sh fetch")
				unpackIdx := strings.Index(run, `tar -xjf "${DOWNLOAD_DIR}/gvisor.tar.bz2"`)
				binIdx := strings.Index(run, `mv "${unpacked}/gvisor-bin" /host/usr/local/bin/gvisor-bin`)
				runscIdx := strings.Index(run, `mv "${unpacked}/runsc" /host/usr/local/bin/runsc`)
				configureIdx := strings.Index(run, "install-gvisor.sh configure")
				for name, idx := range map[string]int{"fetch": fetchIdx, "unpack": unpackIdx, "gvisor-bin": binIdx, "runsc": runscIdx, "configure": configureIdx} {
					require.NotEqual(t, -1, idx, "expected the %s step in run.sh", name)
				}
				assert.Less(t, fetchIdx, unpackIdx)
				assert.Less(t, binIdx, runscIdx, "runsc is moved last: its presence marks a complete install")
				assert.Less(t, runscIdx, configureIdx)
				assert.Contains(t, run, "gvisor.installer.image needs tar with bzip2 support")
			})
		}
	}
}

// Releases before 20260721 keep the separate-file download, in one host call.
func TestGvisorInstallerLegacyLayout(t *testing.T) {
	for _, chart := range agentCharts {
		for _, release := range []string{"20260706", "20260714"} {
			t.Run(chart.name+"/"+release, func(t *testing.T) {
				scripts, err := renderGvisorScripts(t, chart, nil, map[string]string{"gvisor.installer.runscVersion": release})
				require.NoError(t, err)

				install := scripts["install-gvisor.sh"]
				assert.Contains(t, install, "for f in runsc runsc.sha512 containerd-shim-runsc-v1 containerd-shim-runsc-v1.sha512; do")
				assert.Contains(t, install, "mv runsc containerd-shim-runsc-v1 /usr/local/bin")
				assert.NotContains(t, install, "gvisor.tar.bz2.sha512")
				assert.NotContains(t, install, "PHASE}\" == fetch")

				run := scripts["run.sh"]
				assert.Contains(t, run, "chroot /host /bin/bash /opt/gvisor/install-gvisor.sh\n")
				assert.NotContains(t, run, "tar -xjf")
			})
		}
	}
}

// An unquoted release (a YAML number) renders as the release name, not in float notation.
func TestGvisorInstallerNumericRelease(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			scripts, err := renderGvisorScripts(t, chart, map[string]string{"gvisor.installer.runscVersion": "20261116"}, nil)
			require.NoError(t, err)
			assert.Contains(t, scripts["install-gvisor.sh"], `GVISOR_RELEASE="20261116"`)
		})
	}
}

// Anything but a gVisor release name fails at render time instead of on every node.
func TestGvisorInstallerRejectsInvalidRelease(t *testing.T) {
	for _, chart := range agentCharts {
		for _, release := range []string{"latest", "2026-11-16", "20261116-rc1"} {
			t.Run(chart.name+"/"+release, func(t *testing.T) {
				_, err := renderGvisorScripts(t, chart, nil, map[string]string{"gvisor.installer.runscVersion": release})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "runscVersion")
			})
		}
	}
}

// runGvisorPreflight runs the rendered install-gvisor.sh in the given phase with a PATH holding only
// the given mock commands (plus the system tools), against a containerd config at configPath.
func runGvisorPreflight(t *testing.T, chart agentChart, phase string, mocks map[string]string, configContent string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))
	scripts, err := renderGvisorScripts(t, chart, map[string]string{"gvisor.installer.containerdConfigPath": configPath}, nil)
	require.NoError(t, err)
	scriptPath := filepath.Join(dir, "install-gvisor.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte(scripts["install-gvisor.sh"]), 0o755))
	binDir := filepath.Join(dir, "bin")
	require.NoError(t, os.Mkdir(binDir, 0o755))
	for name, body := range mocks {
		require.NoError(t, os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755))
	}
	bash, err := exec.LookPath("bash")
	require.NoError(t, err)
	cmd := exec.Command(bash, scriptPath, phase)
	cmd.Env = []string{"PATH=" + binDir + ":/usr/bin:/bin"}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// When containerd still exposes the handler but runsc or its shim is gone (the README upgrade steps
// remove them), the pre-flight must not skip: it reinstalls on nodes whose handler this installer
// registered, and fails on the others rather than labeling a node that cannot run gVisor pods
// (SONAR-33514).
func TestGvisorInstallerPreflightRequiresBinaries(t *testing.T) {
	crictl := `echo '{"config":{"containerd":{"runtimes":{"runc":{},"runsc":{}}}}}'`
	ownedConfig := "version = 2\n\n# gVisor sandbox runtime (managed by SonarQube Helm chart)\n" +
		"[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.runsc]\n  runtime_type = \"io.containerd.runsc.v1\"\n" +
		"[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.runsc.options]\n  TypeUrl = \"io.containerd.runsc.v1.options\"\n" +
		"  ConfigPath = \"/etc/containerd/runsc.toml\"\n"
	foreignConfig := "version = 2\n"

	for _, chart := range agentCharts {
		t.Run(chart.name+"/binaries present", func(t *testing.T) {
			output, err := runGvisorPreflight(t, chart, "fetch",
				map[string]string{"crictl": crictl, "runsc": "", "containerd-shim-runsc-v1": ""}, ownedConfig)
			require.NoError(t, err, output)
			assert.Contains(t, output, "skipping installation")
		})
		for _, missing := range []string{"runsc", "containerd-shim-runsc-v1"} {
			present := map[string]string{"runsc": "containerd-shim-runsc-v1", "containerd-shim-runsc-v1": "runsc"}[missing]
			t.Run(chart.name+"/"+missing+" missing, installer-owned", func(t *testing.T) {
				output, err := runGvisorPreflight(t, chart, "configure",
					map[string]string{"crictl": crictl, present: ""}, ownedConfig)
				require.Error(t, err, output)
				assert.NotContains(t, output, "skipping installation")
				assert.Contains(t, output, "but "+missing+" is missing; reinstalling")
				// The configure phase runs after run.sh unpacked the archive: reaching this check
				// proves the pre-flight let the install go ahead.
				assert.Contains(t, output, "not installed after unpacking gvisor.tar.bz2")
			})
			t.Run(chart.name+"/"+missing+" missing, not installer-owned", func(t *testing.T) {
				output, err := runGvisorPreflight(t, chart, "fetch",
					map[string]string{"crictl": crictl, present: ""}, foreignConfig)
				require.Error(t, err, output)
				assert.NotContains(t, output, "skipping installation")
				assert.Contains(t, output, "but "+missing+" is not on the node; not labeling it")
			})
		}
	}
}

// install-gvisor.sh must fail, so run.sh never labels the node, when runsc or its shim is missing at
// the end of the install (SONAR-33514).
func TestGvisorInstallerChecksBinariesBeforeLabelling(t *testing.T) {
	for _, chart := range agentCharts {
		for _, release := range []string{"", "20260706"} {
			t.Run(chart.name+"/"+release, func(t *testing.T) {
				setStr := map[string]string{}
				if release != "" {
					setStr["gvisor.installer.runscVersion"] = release
				}
				scripts, err := renderGvisorScripts(t, chart, nil, setStr)
				require.NoError(t, err)
				install := scripts["install-gvisor.sh"]

				assert.Contains(t, install, "for f in runsc containerd-shim-runsc-v1; do")
				restartIdx := strings.Index(install, "containerd already serving runsc; nothing to do")
				checkIdx := strings.Index(install, "not found on the node; not labeling it")
				doneIdx := strings.Index(install, `echo "[install-gvisor] done"`)
				for name, idx := range map[string]int{"restart": restartIdx, "binary check": checkIdx, "done": doneIdx} {
					require.NotEqual(t, -1, idx, "expected the %s step in install-gvisor.sh", name)
				}
				assert.Less(t, restartIdx, checkIdx)
				assert.Less(t, checkIdx, doneIdx)
				assert.Contains(t, install, `if [[ "${reinstall}" != true ]] && command -v runsc >/dev/null 2>&1; then`)

				run := scripts["run.sh"]
				installIdx := strings.LastIndex(run, "/opt/gvisor/install-gvisor.sh")
				labelIdx := strings.Index(run, "chroot /host /bin/bash /opt/gvisor/label-node.sh")
				require.NotEqual(t, -1, labelIdx)
				assert.Less(t, installIdx, labelIdx, "run.sh labels the node only after install-gvisor.sh succeeded")
				assert.Contains(t, run, "set -eu")
			})
		}
	}
}
