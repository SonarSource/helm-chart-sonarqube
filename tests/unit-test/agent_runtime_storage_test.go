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
		values["vortex.enabled"] = "true"
		values["vortex.image.repository"] = "example.com/vortex"
		values["vortex.image.tag"] = "1"
		values["vortex.storage.bucket"] = "vortex-artifacts"
		values["vortex.storage.region"] = "eu-west-1"
	}
	return values
}

// A runtime's storage.filesystem.baseDir, when set, must be mounted by that same runtime's
// extraVolumeMounts - otherwise the orchestrator's file:// handoff resolves to nothing on this
// runtime, since Kubernetes gives it no mount at that path (SONAR-32022). Left unset (the
// default), the check does not run at all: today's flat, unscoped layout keeps working.
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
