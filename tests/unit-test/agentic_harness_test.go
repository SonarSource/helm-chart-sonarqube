package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// splitK8sYamlDocs splits terratest's raw multi-document `helm template` output into individual
// YAML documents, since a single agentic-* template file can render more than one resource
// (e.g. one Deployment+Service per agentic job family, or one NetworkPolicy per component).
func splitK8sYamlDocs(output string) []string {
	var docs []string
	for _, doc := range strings.Split(output, "\n---\n") {
		trimmed := strings.TrimSpace(doc)
		if trimmed != "" {
			docs = append(docs, trimmed)
		}
	}
	return docs
}

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestAgenticHarnessDisabledByDefault(t *testing.T) {
	opts := newSQHelmOptions()
	opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-community-build-empty-tag.yaml"}
	output, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{})
	require.NoError(t, err)
	assert.NotContains(t, output, "agentic-orchestrator")
	assert.NotContains(t, output, "agentic-runtime")
}

func TestAgenticHarnessRequiresJdbcOverwrite(t *testing.T) {
	opts := newSQHelmOptions()
	opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-missing-jdbc.yaml"}
	_, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agenticHarness.enabled requires jdbcOverwrite.enabled=true")
}

func TestAgenticOrchestratorDeployment(t *testing.T) {
	opts := newSQHelmOptions()
	opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-orchestrator.yaml"}
	output, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/agentic-orchestrator.yaml"})
	require.NoError(t, err)

	var deployment appsv1.Deployment
	for _, doc := range splitK8sYamlDocs(output) {
		if strings.Contains(doc, "kind: Deployment") {
			helm.UnmarshalK8SYaml(t, doc, &deployment)
		}
	}
	require.NotEmpty(t, deployment.Name)

	container := deployment.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "example/agentic-orchestrator:1.0.0", container.Image)

	env := map[string]string{}
	envFrom := map[string]corev1.EnvVarSource{}
	for _, e := range container.Env {
		if e.ValueFrom != nil {
			envFrom[e.Name] = *e.ValueFrom
		} else {
			env[e.Name] = e.Value
		}
	}
	// jdbcUrl is jdbc:postgresql://postgres.example.svc:5432/sonarqube?currentSchema=public —
	// the orchestrator needs the bare host:port and db name, not the full JDBC URL.
	assert.Equal(t, "postgres.example.svc:5432", env["CORE_DB_READ_WRITE_ENDPOINT"])
	assert.Equal(t, "sonarqube", env["CORE_DB_NAME"])
	assert.Equal(t, "sonarqube", env["CORE_DB_USERNAME"])
	// The orchestrator has no family-aware routing today, so it always pushes to the hunter runtime.
	assert.Equal(t, "http://sonarqube-sonarqube-agentic-runtime-hunter:8081/jobs", env["AGENTIC_RUNTIME_PUSH_URL"])
	// Storage credentials must always come from a Secret, never a plaintext value.
	_, hasPlainAccessKey := env["AGENTIC_STORAGE_ACCESS_KEY"]
	assert.False(t, hasPlainAccessKey, "AGENTIC_STORAGE_ACCESS_KEY must not be a plaintext env value")
	require.Contains(t, envFrom, "AGENTIC_STORAGE_ACCESS_KEY")
	assert.Equal(t, "sonarqube-sonarqube-agentic-orchestrator-storage", envFrom["AGENTIC_STORAGE_ACCESS_KEY"].SecretKeyRef.Name)
	assert.Equal(t, "access-key", envFrom["AGENTIC_STORAGE_ACCESS_KEY"].SecretKeyRef.Key)
}

func TestAgenticOrchestratorStorageSecret(t *testing.T) {
	t.Run("plain values are wrapped into an internal Secret", func(t *testing.T) {
		opts := newSQHelmOptions()
		opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-orchestrator.yaml"}
		output, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/agentic-orchestrator-secret.yaml"})
		require.NoError(t, err)

		var secret corev1.Secret
		helm.UnmarshalK8SYaml(t, output, &secret)
		assert.Equal(t, "mykey", string(secret.Data["access-key"]))
		assert.Equal(t, "mysecret", string(secret.Data["secret-key"]))
	})

	t.Run("existingSecret suppresses the internal Secret entirely", func(t *testing.T) {
		opts := newSQHelmOptions()
		opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-existing-secret.yaml"}
		_, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/agentic-orchestrator-secret.yaml"})
		assert.Error(t, err, "expected no internal storage Secret when existingSecret is set")

		output, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/agentic-orchestrator.yaml"})
		require.NoError(t, err)
		assert.Contains(t, output, "name: my-s3-secret")
	})
}

func TestAgenticRuntimeDeployment(t *testing.T) {
	opts := newSQHelmOptions()
	opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-orchestrator.yaml"}
	output, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/agentic-runtime.yaml"})
	require.NoError(t, err)

	// Only "hunter" ships by default — no remediation-family Deployment should exist at all.
	assert.NotContains(t, output, "agentic-runtime-remediation")

	var deployment appsv1.Deployment
	for _, doc := range splitK8sYamlDocs(output) {
		if strings.Contains(doc, "kind: Deployment") {
			helm.UnmarshalK8SYaml(t, doc, &deployment)
		}
	}
	require.NotEmpty(t, deployment.Name)
	assert.Equal(t, "example/hunter-runtime:1.0.0", deployment.Spec.Template.Spec.Containers[0].Image)
	for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "AGENT_FAMILY" {
			assert.Equal(t, "hunter", e.Value)
		}
	}
}

func TestAgenticNetworkPolicyEnabledByDefault(t *testing.T) {
	opts := newSQHelmOptions()
	// This fixture doesn't touch networkPolicy at all — it should still be on, and the runtime
	// should get Anthropic's documented CIDR without any egressAllow override.
	opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-orchestrator.yaml"}
	output, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/agentic-networkpolicy.yaml"})
	require.NoError(t, err)

	policies := map[string]networkingv1.NetworkPolicy{}
	for _, doc := range splitK8sYamlDocs(output) {
		var p networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, doc, &p)
		policies[p.Name] = p
	}

	orchestratorPolicy, ok := policies["sonarqube-sonarqube-agentic-orchestrator-network-policy"]
	require.True(t, ok, "expected an orchestrator NetworkPolicy, got: %v", mapKeys(policies))
	// SonarQube + the hunter runtime (podSelector rules); orchestrator.egressAllow is empty by
	// default, so no CIDR rule beyond DNS.
	assert.Len(t, orchestratorPolicy.Spec.Egress, 3)

	runtimePolicy, ok := policies["sonarqube-sonarqube-agentic-runtime-hunter-network-policy"]
	require.True(t, ok, "expected a hunter runtime NetworkPolicy, got: %v", mapKeys(policies))
	// DNS + Anthropic's documented IPv4/IPv6 CIDR, present without any values override.
	require.Len(t, runtimePolicy.Spec.Egress, 3)
	assert.Equal(t, "160.79.104.0/23", runtimePolicy.Spec.Egress[1].To[0].IPBlock.CIDR)
	assert.Equal(t, "2607:6bc0::/48", runtimePolicy.Spec.Egress[2].To[0].IPBlock.CIDR)
}

func TestAgenticNetworkPolicyEgressAllowPodSelector(t *testing.T) {
	// ipBlock can't target a Service's ClusterIP on any CNI (kube-proxy DNATs to the backing pod
	// IP before policy enforcement sees the packet), so egressAllow also accepts a podSelector
	// entry for in-cluster destinations like a self-hosted DB or S3 gateway.
	opts := newSQHelmOptions()
	opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-egressallow-podselector.yaml"}
	output, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/agentic-networkpolicy.yaml"})
	require.NoError(t, err)

	policies := map[string]networkingv1.NetworkPolicy{}
	for _, doc := range splitK8sYamlDocs(output) {
		var p networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, doc, &p)
		policies[p.Name] = p
	}

	orchestratorPolicy, ok := policies["sonarqube-sonarqube-agentic-orchestrator-network-policy"]
	require.True(t, ok, "expected an orchestrator NetworkPolicy, got: %v", mapKeys(policies))

	var minioRule, dbRule *networkingv1.NetworkPolicyEgressRule
	for i := range orchestratorPolicy.Spec.Egress {
		rule := &orchestratorPolicy.Spec.Egress[i]
		if len(rule.To) == 0 || rule.To[0].PodSelector == nil {
			continue
		}
		switch rule.To[0].PodSelector.MatchLabels["app"] {
		case "my-in-cluster-minio":
			minioRule = rule
		case "my-in-cluster-db":
			dbRule = rule
		}
	}

	require.NotNil(t, minioRule, "expected a podSelector egress rule for my-in-cluster-minio")
	assert.Nil(t, minioRule.To[0].NamespaceSelector, "no namespaceSelector override means same-namespace only")
	assert.Equal(t, int32(9000), minioRule.Ports[0].Port.IntVal)

	require.NotNil(t, dbRule, "expected a podSelector egress rule for my-in-cluster-db")
	require.NotNil(t, dbRule.To[0].NamespaceSelector)
	assert.Equal(t, "other-ns", dbRule.To[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])
	assert.Equal(t, int32(5432), dbRule.Ports[0].Port.IntVal)
}

func TestAgenticNetworkPolicyCanBeDisabled(t *testing.T) {
	opts := newSQHelmOptions()
	opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-networkpolicy-disabled.yaml"}
	_, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/agentic-networkpolicy.yaml"})
	assert.Error(t, err, "expected no NetworkPolicy resources when agenticHarness.networkPolicy.enabled=false")
}

func TestAgenticOrchestratorUrlAutoInjection(t *testing.T) {
	t.Run("auto-injected when unset", func(t *testing.T) {
		opts := newSQHelmOptions()
		opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-orchestrator.yaml"}
		output, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/config.yaml"})
		require.NoError(t, err)
		assert.Contains(t, output, "sonar.hunteragent.orchestrator.url=http://sonarqube-sonarqube-agentic-orchestrator:8080")
	})

	t.Run("operator override wins", func(t *testing.T) {
		opts := newSQHelmOptions()
		opts.ValuesFiles = []string{"test-cases-values/sonarqube/test-agentic-harness-custom-orchestrator-url.yaml"}
		output, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{"templates/config.yaml"})
		require.NoError(t, err)
		assert.Contains(t, output, "sonar.hunteragent.orchestrator.url=http://custom-orchestrator:9999")
		assert.NotContains(t, output, "agentic-orchestrator:8080")
	})
}
