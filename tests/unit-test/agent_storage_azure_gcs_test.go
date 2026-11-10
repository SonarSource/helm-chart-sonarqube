package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func renderVortexDeploymentFromFixture(t *testing.T, chart agentChart, setValues map[string]string) (appsv1.Deployment, error) {
	t.Helper()
	return renderDeployment(t, chart, "vortex-enabled.yaml", "templates/vortex.yaml", setValues)
}

func renderStorageSecret(t *testing.T, chart agentChart, fixture, template string, setValues map[string]string) corev1.Secret {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/" + fixture},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{template})
	require.NoError(t, err)
	var secret corev1.Secret
	helm.UnmarshalK8SYaml(t, output, &secret)
	return secret
}

func assertSecretEnv(t *testing.T, container corev1.Container, name, secretName string) {
	t.Helper()
	env := findEnvByName(container, name)
	require.NotNil(t, env, "%s must be set", name)
	require.NotNil(t, env.ValueFrom)
	require.NotNil(t, env.ValueFrom.SecretKeyRef)
	assert.Equal(t, secretName, env.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, name, env.ValueFrom.SecretKeyRef.Key)
}

// Azure and GCS have no region, and Azure addresses a container instead of a bucket, so neither
// may be asked for a region (and Azure not for a bucket) the way S3 is.
func TestVortexAzureAndGCSNeedNoRegion(t *testing.T) {
	cases := map[string]map[string]string{
		"AZURE without region and bucket": {
			"vortexAnalysis.storage.type":                   "AZURE",
			"vortexAnalysis.storage.region":                 "",
			"vortexAnalysis.storage.bucket":                 "",
			"vortexAnalysis.storage.azure.container":        "ctx",
			"vortexAnalysis.storage.azure.connectionString": "cs",
		},
		"GCS without region": {
			"vortexAnalysis.storage.type":   "GCS",
			"vortexAnalysis.storage.region": "",
		},
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for name, setValues := range cases {
				t.Run(name, func(t *testing.T) {
					_, err := renderVortexDeploymentFromFixture(t, chart, setValues)
					require.NoError(t, err)
				})
			}
		})
	}
}

func TestVortexGCSStillRequiresBucket(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderVortexDeploymentFromFixture(t, chart, map[string]string{
				"vortexAnalysis.storage.type":   "GCS",
				"vortexAnalysis.storage.bucket": "",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "vortexAnalysis.storage.bucket is not set")
		})
	}
}

func TestVortexAzureStorage(t *testing.T) {
	setValues := map[string]string{
		"vortexAnalysis.storage.type":                   "AZURE",
		"vortexAnalysis.storage.azure.container":        "analysis-context",
		"vortexAnalysis.storage.azure.connectionString": "UseDevelopmentStorage=true",
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment, err := renderVortexDeploymentFromFixture(t, chart, setValues)
			require.NoError(t, err)
			container := deployment.Spec.Template.Spec.Containers[0]

			assert.Equal(t, "AZURE", findEnvByName(container, "SONAR_AGENTIC_STORAGE_TYPE").Value)
			assert.Equal(t, "analysis-context", findEnvByName(container, "SONAR_AGENTIC_STORAGE_AZURE_CONTAINER").Value)
			assertSecretEnv(t, container, "SONAR_AGENTIC_STORAGE_AZURE_CONNECTION_STRING", chart.fullnamePrefix()+vortexFullnameSuffix)
			assert.Nil(t, findEnvByName(container, "SONAR_AGENTIC_STORAGE_ACCESS_KEY"), "S3 keys are not wired for AZURE")
			assert.Nil(t, findEnvByName(container, "SONAR_AGENTIC_STORAGE_GCS_PROJECT_ID"))

			secret := renderStorageSecret(t, chart, "vortex-enabled.yaml", "templates/vortex-secret.yaml", setValues)
			assert.Equal(t, "UseDevelopmentStorage=true", string(secret.Data["SONAR_AGENTIC_STORAGE_AZURE_CONNECTION_STRING"]))
			assert.NotContains(t, secret.Data, "SONAR_AGENTIC_STORAGE_ACCESS_KEY")
		})
	}
}

func TestVortexGCSStorage(t *testing.T) {
	setValues := map[string]string{
		"vortexAnalysis.storage.type":                "GCS",
		"vortexAnalysis.storage.gcs.projectId":       "my-project",
		"vortexAnalysis.storage.gcs.credentialsJson": "service-account-key",
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment, err := renderVortexDeploymentFromFixture(t, chart, setValues)
			require.NoError(t, err)
			container := deployment.Spec.Template.Spec.Containers[0]

			assert.Equal(t, "my-project", findEnvByName(container, "SONAR_AGENTIC_STORAGE_GCS_PROJECT_ID").Value)
			assertSecretEnv(t, container, "SONAR_AGENTIC_STORAGE_GCS_CREDENTIALS_JSON", chart.fullnamePrefix()+vortexFullnameSuffix)

			secret := renderStorageSecret(t, chart, "vortex-enabled.yaml", "templates/vortex-secret.yaml", setValues)
			assert.Equal(t, "service-account-key", string(secret.Data["SONAR_AGENTIC_STORAGE_GCS_CREDENTIALS_JSON"]))
		})
	}
}

// With no credentials JSON the env var must be absent, so the client falls back to Application
// Default Credentials (Workload Identity).
func TestVortexGCSWithoutCredentialsUsesApplicationDefaults(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment, err := renderVortexDeploymentFromFixture(t, chart, map[string]string{"vortexAnalysis.storage.type": "GCS"})
			require.NoError(t, err)
			container := deployment.Spec.Template.Spec.Containers[0]
			assert.Nil(t, findEnvByName(container, "SONAR_AGENTIC_STORAGE_GCS_CREDENTIALS_JSON"))
		})
	}
}

func TestAgentOrchestratorAzureStorage(t *testing.T) {
	setValues := map[string]string{
		"agentOrchestrator.storage.type":                   "AZURE",
		"agentOrchestrator.storage.azure.container":        "agent-jobs",
		"agentOrchestrator.storage.azure.connectionString": "UseDevelopmentStorage=true",
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := renderAgentOrchestrator(t, chart, setValues).Spec.Template.Spec.Containers[0]

			assert.Equal(t, "AZURE", findEnvByName(container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_TYPE").Value)
			assert.Equal(t, "agent-jobs", findEnvByName(container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_AZURE_CONTAINER").Value)
			assertSecretEnv(t, container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_AZURE_CONNECTION_STRING", chart.fullnamePrefix()+"-agent-orchestrator")
			assert.Nil(t, findEnvByName(container, "AGENTIC_STORAGE_ACCESS_KEY"), "S3 keys are not wired for AZURE")

			secret := renderStorageSecret(t, chart, "agent-orchestrator-enabled.yaml", "templates/agent-orchestrator-secret.yaml", setValues)
			assert.Equal(t, "UseDevelopmentStorage=true", string(secret.Data["SONAR_AGENTIC_ORCHESTRATOR_STORAGE_AZURE_CONNECTION_STRING"]))
		})
	}
}

// An existingSecret supplies the Azure connection string under its long env var name, and the
// chart renders no Secret of its own for it.
func TestAgentOrchestratorAzureStorageExistingSecret(t *testing.T) {
	setValues := map[string]string{
		"agentOrchestrator.storage.type":            "AZURE",
		"agentOrchestrator.storage.azure.container": "agent-jobs",
		"agentOrchestrator.storage.existingSecret":  "my-azure-secret",
	}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := renderAgentOrchestrator(t, chart, setValues).Spec.Template.Spec.Containers[0]
			assertSecretEnv(t, container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_AZURE_CONNECTION_STRING", "my-azure-secret")
			assert.Nil(t, findEnvByName(container, "AGENTIC_STORAGE_ACCESS_KEY"))

			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/agent-orchestrator-enabled.yaml"},
				SetValues:   setValues,
			}
			_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-orchestrator-secret.yaml"})
			require.Error(t, err, "no chart-managed Secret is rendered when existingSecret is set")
		})
	}
}

func TestAgentOrchestratorGCSStorage(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("service account key", func(t *testing.T) {
				setValues := map[string]string{
					"agentOrchestrator.storage.type":                "GCS",
					"agentOrchestrator.storage.bucket":              "agent-jobs",
					"agentOrchestrator.storage.gcs.projectId":       "my-project",
					"agentOrchestrator.storage.gcs.credentialsJson": "service-account-key",
				}
				container := renderAgentOrchestrator(t, chart, setValues).Spec.Template.Spec.Containers[0]
				assert.Equal(t, "my-project", findEnvByName(container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_GCS_PROJECT_ID").Value)
				assertSecretEnv(t, container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_GCS_CREDENTIALS_JSON", chart.fullnamePrefix()+"-agent-orchestrator")

				secret := renderStorageSecret(t, chart, "agent-orchestrator-enabled.yaml", "templates/agent-orchestrator-secret.yaml", setValues)
				assert.Equal(t, "service-account-key", string(secret.Data["SONAR_AGENTIC_ORCHESTRATOR_STORAGE_GCS_CREDENTIALS_JSON"]))
			})
			t.Run("application default credentials", func(t *testing.T) {
				container := renderAgentOrchestrator(t, chart, map[string]string{
					"agentOrchestrator.storage.type": "GCS",
				}).Spec.Template.Spec.Containers[0]
				assert.Nil(t, findEnvByName(container, "SONAR_AGENTIC_ORCHESTRATOR_STORAGE_GCS_CREDENTIALS_JSON"))
			})
		})
	}
}

// The Azure and GCS env vars are scoped to their own type, so an S3 install renders none of them.
func TestAgentStorageAzureGCSEnvAbsentForS3(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			orchestrator := renderAgentOrchestrator(t, chart, nil).Spec.Template.Spec.Containers[0]
			for _, name := range []string{
				"SONAR_AGENTIC_ORCHESTRATOR_STORAGE_AZURE_CONTAINER",
				"SONAR_AGENTIC_ORCHESTRATOR_STORAGE_AZURE_CONNECTION_STRING",
				"SONAR_AGENTIC_ORCHESTRATOR_STORAGE_GCS_PROJECT_ID",
				"SONAR_AGENTIC_ORCHESTRATOR_STORAGE_GCS_CREDENTIALS_JSON",
			} {
				assert.Nil(t, findEnvByName(orchestrator, name), name)
			}
			vortex, err := renderVortexDeploymentFromFixture(t, chart, nil)
			require.NoError(t, err)
			for _, name := range []string{
				"SONAR_AGENTIC_STORAGE_AZURE_CONTAINER",
				"SONAR_AGENTIC_STORAGE_AZURE_CONNECTION_STRING",
				"SONAR_AGENTIC_STORAGE_GCS_PROJECT_ID",
				"SONAR_AGENTIC_STORAGE_GCS_CREDENTIALS_JSON",
			} {
				assert.Nil(t, findEnvByName(vortex.Spec.Template.Spec.Containers[0], name), name)
			}
		})
	}
}

// Azure addresses its container, so an empty bucket is fine but an empty container is not.
func TestAgentOrchestratorAzureRequiresContainerNotBucket(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderAgentOrchestratorDeployment(t, chart, map[string]string{
				"agentOrchestrator.storage.type":                   "AZURE",
				"agentOrchestrator.storage.bucket":                 "",
				"agentOrchestrator.storage.azure.container":        "agent-jobs",
				"agentOrchestrator.storage.azure.connectionString": "cs",
			})
			require.NoError(t, err)

			_, err = renderAgentOrchestratorDeployment(t, chart, map[string]string{
				"agentOrchestrator.storage.type":                   "AZURE",
				"agentOrchestrator.storage.azure.container":        "",
				"agentOrchestrator.storage.azure.connectionString": "cs",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "agentOrchestrator.storage.azure.container is not set")
		})
	}
}

func TestVortexAzureRequiresContainer(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderVortexDeploymentFromFixture(t, chart, map[string]string{
				"vortexAnalysis.storage.type":                   "AZURE",
				"vortexAnalysis.storage.azure.container":        "",
				"vortexAnalysis.storage.azure.connectionString": "cs",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "vortexAnalysis.storage.azure.container is not set")
		})
	}
}
