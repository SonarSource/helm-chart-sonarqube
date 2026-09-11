package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runtimeStorageBase enables one agent runtime family (plus its own dependencies - vortex for
// remediation), so only the storage/mount check under test can fail.
func runtimeStorageBase(family string) map[string]string {
	values := map[string]string{
		"agentOrchestrator.enabled":           "true",
		"agentOrchestrator.image.repository":  "example.com/agent-orchestrator",
		"agentOrchestrator.storage.bucket":    "agent-jobs",
		"agenticSigningSecret.existingSecret": "test-agentic-instance-secret",
		family + "Agent.enabled":              "true",
		family + "Agent.image.repository":     "example.com/" + family + "-agent",
		family + "Agent.image.tag":            "1",
	}
	if family == "remediation" {
		values["vortexAnalysis.enabled"] = "true"
		values["vortexAnalysis.image.repository"] = "example.com/vortex"
		values["vortexAnalysis.image.tag"] = "1"
		values["vortexAnalysis.storage.bucket"] = "vortex-artifacts"
		values["vortexAnalysis.storage.region"] = "eu-west-1"
	}
	return values
}

// A runtime's storage.filesystem.baseDir, when set, must be mounted by that same runtime's
// extraVolumeMounts, or the orchestrator's file:// handoff resolves to nothing on this runtime.
// Left unset, the check doesn't run at all.
func TestRuntimeStorageFilesystemBaseDirRequiresMatchingMount(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					baseDir := "/agentic-storage/" + family

					t.Run("baseDir without a matching mount fails", func(t *testing.T) {
						values := runtimeStorageBase(family)
						values[family+"Agent.storage.filesystem.baseDir"] = baseDir
						_, err := renderWithValidation(t, chart, values)
						require.Error(t, err)
						assert.Contains(t, err.Error(), fmt.Sprintf(
							"%sAgent.storage.filesystem.baseDir is set to %q but no %sAgent.extraVolumeMounts entry has a matching mountPath",
							family, baseDir, family))
					})

					t.Run("baseDir with a matching mount succeeds", func(t *testing.T) {
						values := runtimeStorageBase(family)
						values[family+"Agent.storage.filesystem.baseDir"] = baseDir
						values[family+"Agent.extraVolumeMounts[0].name"] = "agentic-storage"
						values[family+"Agent.extraVolumeMounts[0].mountPath"] = baseDir
						values[family+"Agent.extraVolumeMounts[0].subPath"] = family
						_, err := renderWithValidation(t, chart, values)
						require.NoError(t, err)
					})

					t.Run("a mount at a different path still fails", func(t *testing.T) {
						values := runtimeStorageBase(family)
						values[family+"Agent.storage.filesystem.baseDir"] = baseDir
						values[family+"Agent.extraVolumeMounts[0].name"] = "agentic-storage"
						values[family+"Agent.extraVolumeMounts[0].mountPath"] = "/somewhere/else"
						_, err := renderWithValidation(t, chart, values)
						require.Error(t, err)
						assert.Contains(t, err.Error(), "no "+family+"Agent.extraVolumeMounts entry has a matching mountPath")
					})

					t.Run("no baseDir set stays inert", func(t *testing.T) {
						values := runtimeStorageBase(family)
						_, err := renderWithValidation(t, chart, values)
						require.NoError(t, err)
					})

					// A nulled storage block must not panic with a nil pointer.
					t.Run("storage explicitly nulled stays inert", func(t *testing.T) {
						values := runtimeStorageBase(family)
						values[family+"Agent.storage"] = "null"
						_, err := renderWithValidation(t, chart, values)
						require.NoError(t, err)
					})

					// A trailing slash on either side must not defeat an otherwise-matching mount.
					t.Run("a trailing slash on baseDir still matches a mount without one", func(t *testing.T) {
						values := runtimeStorageBase(family)
						values[family+"Agent.storage.filesystem.baseDir"] = baseDir + "/"
						values[family+"Agent.extraVolumeMounts[0].name"] = "agentic-storage"
						values[family+"Agent.extraVolumeMounts[0].mountPath"] = baseDir
						values[family+"Agent.extraVolumeMounts[0].subPath"] = family
						_, err := renderWithValidation(t, chart, values)
						require.NoError(t, err)
					})

					// Multiple trailing slashes must normalize the same as one, and the failure
					// message must echo the operator's original value, not the normalized one.
					t.Run("multiple trailing slashes on baseDir still match a mount without one", func(t *testing.T) {
						values := runtimeStorageBase(family)
						values[family+"Agent.storage.filesystem.baseDir"] = baseDir + "//"
						values[family+"Agent.extraVolumeMounts[0].name"] = "agentic-storage"
						values[family+"Agent.extraVolumeMounts[0].mountPath"] = baseDir
						values[family+"Agent.extraVolumeMounts[0].subPath"] = family
						_, err := renderWithValidation(t, chart, values)
						require.NoError(t, err)
					})

					// A mount entry with no mountPath key (e.g. a typo) must fall through to the
					// validation message, not crash on a nil value.
					t.Run("a mount without a mountPath key fails cleanly, not with a crash", func(t *testing.T) {
						values := runtimeStorageBase(family)
						values[family+"Agent.storage.filesystem.baseDir"] = baseDir
						values[family+"Agent.extraVolumeMounts[0].name"] = "agentic-storage"
						values[family+"Agent.extraVolumeMounts[0].subPath"] = family
						_, err := renderWithValidation(t, chart, values)
						require.Error(t, err)
						assert.Contains(t, err.Error(), "no "+family+"Agent.extraVolumeMounts entry has a matching mountPath")
						assert.NotContains(t, err.Error(), "wrong type for value")
					})

					// A mount at an ancestor of baseDir (the whole volume, no subPath) genuinely
					// gives the runtime a working path at baseDir, so it must satisfy the check too.
					t.Run("a mount at an ancestor of baseDir satisfies the check", func(t *testing.T) {
						values := runtimeStorageBase(family)
						values[family+"Agent.storage.filesystem.baseDir"] = baseDir
						values[family+"Agent.extraVolumeMounts[0].name"] = "agentic-storage"
						values[family+"Agent.extraVolumeMounts[0].mountPath"] = "/agentic-storage"
						_, err := renderWithValidation(t, chart, values)
						require.NoError(t, err)
					})

					// A mount below baseDir (a subdirectory of it) does not give the runtime a
					// path at baseDir itself, so it must still fail.
					t.Run("a mount below baseDir still fails", func(t *testing.T) {
						values := runtimeStorageBase(family)
						values[family+"Agent.storage.filesystem.baseDir"] = baseDir
						values[family+"Agent.extraVolumeMounts[0].name"] = "agentic-storage"
						values[family+"Agent.extraVolumeMounts[0].mountPath"] = baseDir + "/extra"
						_, err := renderWithValidation(t, chart, values)
						require.Error(t, err)
						assert.Contains(t, err.Error(), "no "+family+"Agent.extraVolumeMounts entry has a matching mountPath")
					})
				})
			}
		})
	}
}

// The runtime's own subPath mount (extraVolumeMounts/extraVolumes) renders through as plain
// pass-through YAML on templates/agent-runtime.yaml, the same way any other extraVolumes entry
// does - confirming the values shown in values.yaml's comments actually render as documented.
func TestRuntimeStorageSubPathMountRenders(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				t.Run(family, func(t *testing.T) {
					baseDir := "/agentic-storage/" + family
					deployment := renderAgentRuntime(t, chart, family, map[string]string{
						family + "Agent.storage.filesystem.baseDir":                      baseDir,
						family + "Agent.extraVolumeMounts[0].name":                       "agentic-storage",
						family + "Agent.extraVolumeMounts[0].mountPath":                  baseDir,
						family + "Agent.extraVolumeMounts[0].subPath":                    family,
						family + "Agent.extraVolumes[0].name":                            "agentic-storage",
						family + "Agent.extraVolumes[0].persistentVolumeClaim.claimName": "agentic-storage",
					})

					container := deployment.Spec.Template.Spec.Containers[0]
					mounts := agentVolumeMountsByName(container.VolumeMounts)
					require.Contains(t, mounts, "agentic-storage")
					assert.Equal(t, baseDir, mounts["agentic-storage"].MountPath)
					assert.Equal(t, family, mounts["agentic-storage"].SubPath)

					volumes := agentVolumesByName(deployment.Spec.Template.Spec.Volumes)
					require.Contains(t, volumes, "agentic-storage")
					require.NotNil(t, volumes["agentic-storage"].PersistentVolumeClaim)
					assert.Equal(t, "agentic-storage", volumes["agentic-storage"].PersistentVolumeClaim.ClaimName)
				})
			}
		})
	}
}
