package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const runtimeStoragePrefix = "sonar.agentic.runtime.storage."

func TestServerRuntimeStorageProperties(t *testing.T) {
	for _, chart := range agentCharts {
		checkServerRuntimeStorageProperties(t, chart)
	}
}

func checkServerRuntimeStorageProperties(t *testing.T, chart agentChart) {
	for _, provider := range []string{"S3", "AZURE", "GCS"} {
		t.Run(chart.name+"/"+provider, func(t *testing.T) {
			values := map[string]string{
				"agentOrchestrator.storage.type":                               provider,
				"agentOrchestrator.storage.bucket":                             "job-artifacts",
				"agentOrchestrator.storage.region":                             "eu-west-1",
				"agentOrchestrator.storage.endpoint":                           "https://objects.example.com",
				"agentOrchestrator.storage.pathStyle":                          "false",
				"agentOrchestrator.storage.azure.container":                    "job-container",
				"agentOrchestrator.storage.azure.connectionString":             "test-connection",
				"agentOrchestrator.storage.gcs.projectId":                      "job-project",
				chart.sonarPropertiesPath + `.sonar\.agentic\.storage\.bucket`: "vortex-context",
			}
			props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-orchestrator-enabled.yaml", values))
			assert.Equal(t, provider, props[runtimeStoragePrefix+"type"])
			assert.Equal(t, "job-artifacts", props[runtimeStoragePrefix+"bucket"])
			assert.Equal(t, "eu-west-1", props[runtimeStoragePrefix+"region"])
			assert.Equal(t, "https://objects.example.com", props[runtimeStoragePrefix+"endpoint"])
			assert.Equal(t, "false", props[runtimeStoragePrefix+"path-style-access"])
			assert.Equal(t, "vortex-context", props["sonar.agentic.storage.bucket"])
			if provider == "AZURE" {
				assert.Equal(t, "job-container", props[runtimeStoragePrefix+"azure.container"])
			}
			if provider == "GCS" {
				assert.Equal(t, "job-project", props[runtimeStoragePrefix+"gcs.project-id"])
			}
			for key := range props {
				assert.NotContains(t, key, "sonar.agentic.orchestrator.storage.")
			}
			assert.NotContains(t, props, runtimeStoragePrefix+"azure.connection-string")
			assert.NotContains(t, props, runtimeStoragePrefix+"gcs.credentials-json")
		})
	}
	t.Run(chart.name+"/override", func(t *testing.T) {
		props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-orchestrator-enabled.yaml", map[string]string{
			chart.sonarPropertiesPath + `.sonar\.agentic\.runtime\.storage\.bucket`: "explicit-job-bucket",
		}))
		assert.Equal(t, "explicit-job-bucket", props[runtimeStoragePrefix+"bucket"])
	})
	t.Run(chart.name+"/disabled", func(t *testing.T) {
		props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-all-disabled.yaml", nil))
		for key := range props {
			assert.False(t, strings.HasPrefix(key, runtimeStoragePrefix))
		}
	})
}

func serverStorageInit(t *testing.T, chart agentChart, values map[string]string) corev1.Container {
	t.Helper()
	template := chart.appTemplate
	if values["deploymentType"] == "Deployment" {
		template = "templates/deployment.yaml"
	}
	pod := agenticPodSpec(t, chart, template, "agent-orchestrator-enabled.yaml", values, "")
	for _, init := range pod.InitContainers {
		if init.Name == "concat-properties" {
			return init
		}
	}
	t.Fatal("no properties init container")
	return corev1.Container{}
}

func TestServerRuntimeStorageCredentialReferences(t *testing.T) {
	for _, chart := range agentCharts {
		checkServerRuntimeStorageCredentialReferences(t, chart)
	}
}

func checkServerRuntimeStorageCredentialReferences(t *testing.T, chart agentChart) {
	cases := []struct{ name, provider, field, env, key string }{
		{"s3", "S3", "accessKey", "RUNTIME_STORAGE_ACCESS_KEY", "AGENTIC_STORAGE_ACCESS_KEY"},
		{"azure", "AZURE", "azure.connectionString", "RUNTIME_STORAGE_AZURE_CONNECTION_STRING", "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_AZURE_CONNECTION_STRING"},
		{"gcs", "GCS", "gcs.credentialsJson", "RUNTIME_STORAGE_GCS_CREDENTIALS_JSON", "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_GCS_CREDENTIALS_JSON"},
	}
	for _, tc := range cases {
		for _, existing := range []bool{false, true} {
			t.Run(chart.name+"/"+tc.name+"/existing="+map[bool]string{false: "false", true: "true"}[existing], func(t *testing.T) {
				values := map[string]string{"agentOrchestrator.storage.type": tc.provider, "agentOrchestrator.storage.azure.container": "jobs"}
				secret := chart.fullnamePrefix() + "-agent-orchestrator"
				if existing {
					values["agentOrchestrator.storage.existingSecret"] = "external-job-store"
					secret = "external-job-store"
				} else {
					values["agentOrchestrator.storage."+tc.field] = "test-value"
					if tc.provider == "S3" {
						values["agentOrchestrator.storage.secretKey"] = "test-value"
					}
				}
				init := serverStorageInit(t, chart, values)
				env := findEnvByName(init, tc.env)
				require.NotNil(t, env)
				require.NotNil(t, env.ValueFrom)
				require.NotNil(t, env.ValueFrom.SecretKeyRef)
				assert.Equal(t, secret, env.ValueFrom.SecretKeyRef.Name)
				assert.Equal(t, tc.key, env.ValueFrom.SecretKeyRef.Key)
				assert.NotContains(t, strings.Join(init.Command, " "), "test-value")
			})
		}
	}
	checkServerRuntimeDefaultIdentity(t, chart)
}

func checkServerRuntimeDefaultIdentity(t *testing.T, chart agentChart) {
	for _, provider := range []string{"S3", "GCS"} {
		t.Run(chart.name+"/default-identity/"+provider, func(t *testing.T) {
			init := serverStorageInit(t, chart, map[string]string{"agentOrchestrator.storage.type": provider})
			for _, env := range init.Env {
				assert.False(t, strings.HasPrefix(env.Name, "RUNTIME_STORAGE_"))
			}
		})
	}
}

func TestServerRuntimeStoragePropertiesScript(t *testing.T) {
	for _, chart := range agentCharts {
		for _, deployment := range []bool{false, true} {
			if chart.name == "sonarqube-dce" && deployment {
				continue
			}
			t.Run(chart.name+map[bool]string{false: "/default", true: "/deployment"}[deployment], func(t *testing.T) {
				values := map[string]string{"agentOrchestrator.storage.existingSecret": "job-store"}
				if deployment {
					values["deploymentType"] = "Deployment"
				}
				init := serverStorageInit(t, chart, values)
				dir := t.TempDir()
				props := filepath.Join(dir, "props")
				result := filepath.Join(dir, "result")
				require.NoError(t, os.Mkdir(props, 0700))
				require.NoError(t, os.Mkdir(result, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(props, "sonar.properties"), []byte("sonar.agentic.runtime.storage.bucket=jobs\n"), 0600))
				script := strings.ReplaceAll(init.Command[2], "/tmp/props", props)
				script = strings.ReplaceAll(script, "/tmp/result", result)
				command := exec.Command("sh", "-c", script)
				command.Env = append(os.Environ(), "RUNTIME_STORAGE_ACCESS_KEY= test\\key\nnext\r\t", "RUNTIME_STORAGE_SECRET_KEY=test-secret")
				output, err := command.CombinedOutput()
				require.NoError(t, err, string(output))
				actual, err := os.ReadFile(filepath.Join(result, "sonar.properties"))
				require.NoError(t, err)
				assert.Contains(t, string(actual), "sonar.agentic.runtime.storage.access-key=\\ test\\\\key\\nnext\\r\\t\n")
				assert.Contains(t, string(actual), "sonar.agentic.runtime.storage.secret-key=test-secret\n")
				// User Secret properties must be last, so they override generated credentials.
				require.NoError(t, os.WriteFile(filepath.Join(props, "secret.properties"), []byte("sonar.agentic.runtime.storage.access-key=user-override\n"), 0600))
				command = exec.Command("sh", "-c", script)
				command.Env = append(os.Environ(), "RUNTIME_STORAGE_ACCESS_KEY=generated", "RUNTIME_STORAGE_SECRET_KEY=generated")
				output, err = command.CombinedOutput()
				require.NoError(t, err, string(output))
				actual, err = os.ReadFile(filepath.Join(result, "sonar.properties"))
				require.NoError(t, err)
				assert.True(t, strings.HasSuffix(string(actual), "sonar.agentic.runtime.storage.access-key=user-override\n"))
			})
		}
	}
}

func TestServerRuntimeFilesystemStore(t *testing.T) {
	for _, chart := range agentCharts {
		for _, provider := range []string{"FILESYSTEM", "NFS"} {
			t.Run(chart.name+"/"+provider, func(t *testing.T) {
				server := ""
				if chart.name == "sonarqube-dce" {
					server = "applicationNodes."
				}
				values := map[string]string{
					"agentOrchestrator.storage.type":                                    provider,
					"agentOrchestrator.storage.filesystem.baseDir":                      "/artifacts/jobs",
					"agentOrchestrator.extraVolumes[0].name":                            "jobs",
					"agentOrchestrator.extraVolumes[0].persistentVolumeClaim.claimName": "shared-jobs",
					"agentOrchestrator.extraVolumeMounts[0].name":                       "jobs",
					"agentOrchestrator.extraVolumeMounts[0].mountPath":                  "/artifacts",
					server + "extraVolumes[0].name":                                     "server-jobs",
					server + "extraVolumes[0].persistentVolumeClaim.claimName":          "shared-jobs",
					server + "extraVolumeMounts[0].name":                                "server-jobs",
					server + "extraVolumeMounts[0].mountPath":                           "/artifacts/jobs",
					server + "extraVolumeMounts[0].subPath":                             "jobs",
				}
				opts := agentPropertiesOptions(chart, "agent-orchestrator-enabled.yaml", values)
				_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{chart.appTemplate})
				require.NoError(t, err)
				props := appSonarProperties(t, chart, opts)
				assert.Equal(t, "/artifacts/jobs", props[runtimeStoragePrefix+"filesystem.base-dir"])
				for _, invalid := range []struct{ key, value string }{
					{server + "extraVolumes[0].persistentVolumeClaim.claimName", "different-store"},
					{server + "extraVolumeMounts[0].mountPath", "/elsewhere"},
					{server + "extraVolumeMounts[0].subPath", "hunter"},
					{server + "extraVolumeMounts[0].name", "missing-volume"},
					{"agentOrchestrator.extraVolumeMounts[0].readOnly", "true"},
				} {
					old := values[invalid.key]
					values[invalid.key] = invalid.value
					_, err = helm.RenderTemplateE(t, opts, chart.path, chart.release, nil)
					require.Error(t, err)
					assert.Contains(t, err.Error(), "Server job log access requires matching")
					values[invalid.key] = old
				}
			})
		}
	}
}

func withServerRuntimeFilesystem(chart agentChart, values map[string]string) map[string]string {
	server := ""
	if chart.name == "sonarqube-dce" {
		server = "applicationNodes."
	}
	for _, prefix := range []string{"agentOrchestrator.", server} {
		values[prefix+"extraVolumes[0].name"] = "job-store"
		values[prefix+"extraVolumes[0].persistentVolumeClaim.claimName"] = "shared-jobs"
		values[prefix+"extraVolumeMounts[0].name"] = "job-store"
		values[prefix+"extraVolumeMounts[0].mountPath"] = values["agentOrchestrator.storage.filesystem.baseDir"]
	}
	return values
}

func runtimeStorageChecksum(t *testing.T, chart agentChart, values map[string]string) string {
	t.Helper()
	template := chart.appTemplate
	if values["deploymentType"] == "Deployment" {
		template = "templates/deployment.yaml"
	}
	rendered := helm.RenderTemplate(t, agentPropertiesOptions(chart, "agent-orchestrator-enabled.yaml", values), chart.path, chart.release, []string{template})
	var workload appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, rendered, &workload)
	return workload.Spec.Template.Annotations["checksum/runtime-storage-secret"]
}

func TestServerRuntimeStorageCredentialRotation(t *testing.T) {
	for _, chart := range agentCharts {
		checkRuntimeStorageCredentialRotation(t, chart, "StatefulSet")
		if chart.name == "sonarqube" {
			checkRuntimeStorageCredentialRotation(t, chart, "Deployment")
		}
	}
}

func checkRuntimeStorageCredentialRotation(t *testing.T, chart agentChart, deployment string) {
	for _, provider := range []struct{ name, field string }{
		{"S3", "secretKey"}, {"AZURE", "azure.connectionString"}, {"GCS", "gcs.credentialsJson"},
	} {
		t.Run(chart.name+"/"+deployment+"/"+provider.name, func(t *testing.T) {
			values := map[string]string{
				"agentOrchestrator.storage.type":              provider.name,
				"agentOrchestrator.storage.azure.container":   "job-container",
				"agentOrchestrator.storage." + provider.field: "before-rotation",
			}
			if chart.name == "sonarqube" {
				values["deploymentType"] = deployment
			}
			if provider.name == "S3" {
				values["agentOrchestrator.storage.accessKey"] = "rotation-access"
			}
			before := runtimeStorageChecksum(t, chart, values)
			require.NotEmpty(t, before)
			values["agentOrchestrator.storage."+provider.field] = "after-rotation"
			assert.NotEqual(t, before, runtimeStorageChecksum(t, chart, values))
			values["agentOrchestrator.storage.existingSecret"] = "external-job-store"
			assert.Empty(t, runtimeStorageChecksum(t, chart, values))
		})
	}
}
