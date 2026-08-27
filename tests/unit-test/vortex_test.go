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
	networkingv1 "k8s.io/api/networking/v1"
)

// Vortex is a standalone service deployed alongside SonarQube, like the MCP server, and is
// independent of the orchestrator/hunterAgent/remediationAgent.

const vortexFullnameSuffix = "-vortex"

func renderVortex(t *testing.T, chart agentChart, fixture string, templates ...string) (string, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/" + fixture},
	}
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, templates)
}

func vortexDocs(manifest string) []string {
	var docs []string
	for _, doc := range strings.Split(manifest, "\n---") {
		if strings.TrimSpace(doc) != "" {
			docs = append(docs, doc)
		}
	}
	return docs
}

func vortexDocKind(doc string) string {
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "kind:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "kind:"))
		}
	}
	return ""
}

func vortexDeployment(t *testing.T, chart agentChart, fixture string) appsv1.Deployment {
	t.Helper()
	output, err := renderVortex(t, chart, fixture, "templates/vortex.yaml")
	require.NoError(t, err)

	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	require.NotEmpty(t, deployment.Name, "no Deployment rendered by templates/vortex.yaml")
	return deployment
}

func vortexContainerEnv(container corev1.Container) map[string]corev1.EnvVar {
	env := map[string]corev1.EnvVar{}
	for _, e := range container.Env {
		env[e.Name] = e
	}
	return env
}

// Off by default, so an existing install picks up nothing new until vortex.enabled is set.
func TestVortexDisabledByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tpl := range []string{
				"templates/vortex.yaml",
				"templates/vortex-service.yaml",
				"templates/vortex-secret.yaml",
			} {
				output, err := renderVortex(t, chart, "vortex-disabled.yaml", tpl)
				require.Error(t, err, "%s must render nothing when vortex.enabled is false", tpl)
				assert.Empty(t, strings.TrimSpace(output))
			}
		})
	}
}

// It must render with no orchestrator/hunterAgent/remediationAgent configured at all.
func TestVortexIndependentOfAgentComponents(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := vortexDeployment(t, chart, "vortex-enabled.yaml")
			assert.Equal(t, chart.fullnamePrefix()+vortexFullnameSuffix, deployment.Name)
			assert.Equal(t, "vortex", deployment.Labels["app.kubernetes.io/component"])

			// It uses the release's own ServiceAccount, like the MCP server.
			_, err := renderVortex(t, chart, "vortex-enabled.yaml", "templates/agent-serviceaccount.yaml")
			require.Error(t, err, "the agent ServiceAccount template must render nothing for a vortex-only install")
		})
	}
}

// The property that activates analysis on the SonarQube side is sonar.vortex.analysis.url, injected
// as SONAR_VORTEX_ANALYSIS_URL. Absent when the service isn't deployed — unset means SonarQube routes
// no analysis requests, rather than failing to boot.
func TestVortexAppNodeWiring(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("set when enabled", func(t *testing.T) {
				output, err := renderVortex(t, chart, "vortex-enabled.yaml", chart.appTemplate)
				require.NoError(t, err)

				var app appsv1.Deployment
				helm.UnmarshalK8SYaml(t, output, &app)
				env := vortexContainerEnv(app.Spec.Template.Spec.Containers[0])

				vortex, ok := env["SONAR_VORTEX_ANALYSIS_URL"]
				require.True(t, ok, "app nodes must receive SONAR_VORTEX_ANALYSIS_URL when the service is enabled")
				assert.Equal(t, "http://"+chart.fullnamePrefix()+vortexFullnameSuffix+":8080", vortex.Value)
			})

			t.Run("absent when disabled", func(t *testing.T) {
				output, err := renderVortex(t, chart, "vortex-disabled.yaml", chart.appTemplate)
				require.NoError(t, err)

				var app appsv1.Deployment
				helm.UnmarshalK8SYaml(t, output, &app)
				_, ok := vortexContainerEnv(app.Spec.Template.Spec.Containers[0])["SONAR_VORTEX_ANALYSIS_URL"]
				assert.False(t, ok, "SONAR_VORTEX_ANALYSIS_URL must not be set when the service is disabled")
			})
		})
	}
}

// Vortex analysis calls the SonarQube Web API, so it needs the SonarQube URL and a token. User
// supplied env comes last, so it can override the values the chart wires automatically.
func TestVortexContainerEnv(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := vortexDeployment(t, chart, "vortex-enabled.yaml").Spec.Template.Spec.Containers[0]
			env := vortexContainerEnv(container)

			url, ok := env["VORTEX_ANALYSIS_SONARQUBE_URL"]
			require.True(t, ok)
			assert.Equal(t, "http://"+chart.fullnamePrefix()+":9000", url.Value)

			token, ok := env["VORTEX_ANALYSIS_SONARQUBE_TOKEN"]
			require.True(t, ok, "the callback token must be wired when a token is configured")
			require.NotNil(t, token.ValueFrom)
			require.NotNil(t, token.ValueFrom.SecretKeyRef)
			assert.Equal(t, chart.fullnamePrefix()+vortexFullnameSuffix, token.ValueFrom.SecretKeyRef.Name)
			assert.Equal(t, "VORTEX_ANALYSIS_SONARQUBE_TOKEN", token.ValueFrom.SecretKeyRef.Key)

			extra, ok := env["MY_VAR"]
			require.True(t, ok, "extra env must reach the container")
			assert.Equal(t, "my-value", extra.Value)
			assert.Equal(t, "MY_VAR", container.Env[len(container.Env)-1].Name,
				"user-supplied env must come last so it overrides the auto-wired vars")
		})
	}
}

// Context restoration reads from an S3-compatible object store, wired via the SONAR_AGENTIC_STORAGE_*
// env vars (SONAR-31587). They must resolve to the same bucket SonarQube Server writes the context
// items to, or restoration silently finds nothing.
func TestVortexStorageEnv(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := vortexDeployment(t, chart, "vortex-enabled.yaml").Spec.Template.Spec.Containers[0]
			env := vortexContainerEnv(container)

			assert.Equal(t, "S3", env["SONAR_AGENTIC_STORAGE_TYPE"].Value)
			assert.Equal(t, "agent-artifacts", env["SONAR_AGENTIC_STORAGE_BUCKET"].Value)
			assert.Equal(t, "eu-west-1", env["SONAR_AGENTIC_STORAGE_REGION"].Value)
			assert.Equal(t, "false", env["SONAR_AGENTIC_STORAGE_PATH_STYLE_ACCESS"].Value)
			assert.Equal(t, "", env["SONAR_AGENTIC_STORAGE_ENDPOINT"].Value)

			_, hasAccessKey := env["SONAR_AGENTIC_STORAGE_ACCESS_KEY"]
			_, hasSecretKey := env["SONAR_AGENTIC_STORAGE_SECRET_KEY"]
			assert.False(t, hasAccessKey, "no static credentials configured, so the AWS SDK must fall back to its default credential chain")
			assert.False(t, hasSecretKey)
		})
	}
}

// A custom endpoint (e.g. MinIO) with inline static credentials becomes a chart-managed Secret.
func TestVortexStorageCredentials(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := vortexDeployment(t, chart, "vortex-storage-credentials.yaml").Spec.Template.Spec.Containers[0]
			env := vortexContainerEnv(container)

			assert.Equal(t, "http://minio.speedboat-test-xx.svc:9000", env["SONAR_AGENTIC_STORAGE_ENDPOINT"].Value)

			accessKey := env["SONAR_AGENTIC_STORAGE_ACCESS_KEY"]
			require.NotNil(t, accessKey.ValueFrom)
			require.NotNil(t, accessKey.ValueFrom.SecretKeyRef)
			assert.Equal(t, chart.fullnamePrefix()+vortexFullnameSuffix, accessKey.ValueFrom.SecretKeyRef.Name)
			assert.Equal(t, "SONAR_AGENTIC_STORAGE_ACCESS_KEY", accessKey.ValueFrom.SecretKeyRef.Key)

			secretKey := env["SONAR_AGENTIC_STORAGE_SECRET_KEY"]
			require.NotNil(t, secretKey.ValueFrom)
			assert.Equal(t, "SONAR_AGENTIC_STORAGE_SECRET_KEY", secretKey.ValueFrom.SecretKeyRef.Key)

			output, err := renderVortex(t, chart, "vortex-storage-credentials.yaml", "templates/vortex-secret.yaml")
			require.NoError(t, err)
			var secret corev1.Secret
			helm.UnmarshalK8SYaml(t, output, &secret)
			assert.Equal(t, "minioadmin", string(secret.Data["SONAR_AGENTIC_STORAGE_ACCESS_KEY"]))
			assert.Equal(t, "miniosecret", string(secret.Data["SONAR_AGENTIC_STORAGE_SECRET_KEY"]))
		})
	}
}

// An existingSecret for the storage credentials is referenced directly, with no chart-managed Secret.
func TestVortexStorageExistingSecret(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := vortexDeployment(t, chart, "vortex-storage-existing-secret.yaml")
			env := vortexContainerEnv(deployment.Spec.Template.Spec.Containers[0])

			accessKey := env["SONAR_AGENTIC_STORAGE_ACCESS_KEY"]
			require.NotNil(t, accessKey.ValueFrom)
			assert.Equal(t, "my-minio-credentials", accessKey.ValueFrom.SecretKeyRef.Name)
			assert.Equal(t, "SONAR_AGENTIC_STORAGE_ACCESS_KEY", accessKey.ValueFrom.SecretKeyRef.Key)

			output, err := renderVortex(t, chart, "vortex-storage-existing-secret.yaml", "templates/vortex-secret.yaml")
			require.Error(t, err, "no Secret may be rendered when both the token and the storage credentials come from an existingSecret")
			assert.Empty(t, strings.TrimSpace(output))
		})
	}
}

// By default the pod talks to the Web API with its own token and never to the Kubernetes API, so
// it must not get a ServiceAccount token mounted, and it uses "default" rather than a pinned name.
func TestVortexDoesNotAutomountServiceAccountToken(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			podSpec := vortexDeployment(t, chart, "vortex-enabled.yaml").Spec.Template.Spec
			require.NotNil(t, podSpec.AutomountServiceAccountToken)
			assert.False(t, *podSpec.AutomountServiceAccountToken)
			assert.Equal(t, "default", podSpec.ServiceAccountName)
		})
	}
}

// A dedicated, IRSA-annotated ServiceAccount (SONAR-31587) lets the pod reach real S3 for context
// restoration instead of falling back to the node's own AWS identity.
func TestVortexServiceAccount(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			podSpec := vortexDeployment(t, chart, "vortex-serviceaccount.yaml").Spec.Template.Spec
			assert.Equal(t, chart.fullnamePrefix()+vortexFullnameSuffix, podSpec.ServiceAccountName)
			require.NotNil(t, podSpec.AutomountServiceAccountToken)
			assert.True(t, *podSpec.AutomountServiceAccountToken)

			output, err := renderVortex(t, chart, "vortex-serviceaccount.yaml", "templates/vortex-serviceaccount.yaml")
			require.NoError(t, err)

			var sa corev1.ServiceAccount
			helm.UnmarshalK8SYaml(t, output, &sa)
			assert.Equal(t, podSpec.ServiceAccountName, sa.Name)
			assert.Equal(t, "arn:aws:iam::123456789012:role/vortex", sa.Annotations["eks.amazonaws.com/role-arn"])

			_, err = renderVortex(t, chart, "vortex-enabled.yaml", "templates/vortex-serviceaccount.yaml")
			require.Error(t, err, "no ServiceAccount may render when vortex.serviceAccount.create is false")
		})
	}
}

// The token reaches the container through secretKeyRef, which is only read at startup. Without a
// checksum annotation, rotating it would leave the running pod on the old token.
func TestVortexRollsOnTokenChange(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			annotations := vortexDeployment(t, chart, "vortex-enabled.yaml").Spec.Template.Annotations
			checksum, ok := annotations["checksum/secret"]
			require.True(t, ok, "the pod template must carry a checksum of the token Secret")

			// A different token must produce a different pod template, so the upgrade rolls the pod.
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-enabled.yaml"},
				SetValues:   map[string]string{"vortex.sonarqubeToken.token": "squ_rotated0000000000000000000000000000000"},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
			require.NoError(t, err)

			var rotated appsv1.Deployment
			helm.UnmarshalK8SYaml(t, output, &rotated)
			assert.NotEqual(t, checksum, rotated.Spec.Template.Annotations["checksum/secret"])
		})
	}
}

// Recreate rather than the default RollingUpdate, so an upgrade never runs two pods at once.
func TestVortexUsesRecreateStrategy(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := vortexDeployment(t, chart, "vortex-enabled.yaml")
			assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deployment.Spec.Strategy.Type)
		})
	}
}

// Only the Vortex image pull secrets are applied. The application nodes' ones are not
// reused, so the pod never references registry credentials that were not configured for it.
func TestVortexPullSecrets(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			podSpec := vortexDeployment(t, chart, "vortex-enabled.yaml").Spec.Template.Spec

			var names []string
			for _, s := range podSpec.ImagePullSecrets {
				names = append(names, s.Name)
			}
			assert.Equal(t, []string{"vortex-secret", "vortex-secret-list"}, names,
				"only the Vortex pull secrets may be applied")
		})
	}
}

// Vortex can be scheduled apart from the SonarQube pods through its own nodeSelector,
// affinity and tolerations.
func TestVortexScheduling(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			podSpec := vortexDeployment(t, chart, "vortex-enabled.yaml").Spec.Template.Spec

			assert.Equal(t, map[string]string{"vortex": "true"}, podSpec.NodeSelector)

			require.Len(t, podSpec.Tolerations, 1)
			assert.Equal(t, "vortex", podSpec.Tolerations[0].Key)

			require.NotNil(t, podSpec.Affinity)
			require.NotNil(t, podSpec.Affinity.NodeAffinity)
			terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			require.Len(t, terms, 1)
			require.Len(t, terms[0].MatchExpressions, 1)
			assert.Equal(t, "vortex", terms[0].MatchExpressions[0].Key)
		})
	}
}

// Vortex's own scheduling settings take precedence over the chart's global ones.
func TestVortexSchedulingWinsOverGlobal(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			podSpec := vortexDeployment(t, chart, "vortex-global-scheduling.yaml").Spec.Template.Spec

			assert.Equal(t, map[string]string{"vortex": "true"}, podSpec.NodeSelector)

			require.Len(t, podSpec.Tolerations, 1)
			assert.Equal(t, "vortex", podSpec.Tolerations[0].Key)

			require.NotNil(t, podSpec.Affinity)
			terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			require.Len(t, terms, 1)
			require.Len(t, terms[0].MatchExpressions, 1)
			assert.Equal(t, "vortex", terms[0].MatchExpressions[0].Key)
		})
	}
}

// The pod can take several minutes to become ready on a first start, so the startup probe must allow
// for that before the liveness probe begins.
func TestVortexProbes(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			container := vortexDeployment(t, chart, "vortex-enabled.yaml").Spec.Template.Spec.Containers[0]

			for name, probe := range map[string]*corev1.Probe{
				"startup":   container.StartupProbe,
				"readiness": container.ReadinessProbe,
				"liveness":  container.LivenessProbe,
			} {
				require.NotNil(t, probe, "%s probe must be set", name)
				require.NotNil(t, probe.HTTPGet, "%s probe must be an HTTP GET", name)
				assert.Equal(t, "/health", probe.HTTPGet.Path)
				assert.Equal(t, "http", probe.HTTPGet.Port.StrVal)
			}

			startupWindow := container.StartupProbe.PeriodSeconds * container.StartupProbe.FailureThreshold
			assert.GreaterOrEqual(t, startupWindow, int32(300),
				"the startup window must tolerate a multi-minute cold start")
		})
	}
}

// The Service must select the Deployment's pods — a label typo here silently yields no endpoints.
func TestVortexService(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderVortex(t, chart, "vortex-enabled.yaml", "templates/vortex-service.yaml")
			require.NoError(t, err)

			var service corev1.Service
			helm.UnmarshalK8SYaml(t, output, &service)
			assert.Equal(t, chart.fullnamePrefix()+vortexFullnameSuffix, service.Name)
			require.Len(t, service.Spec.Ports, 1)
			assert.Equal(t, int32(8080), service.Spec.Ports[0].Port)

			podLabels := vortexDeployment(t, chart, "vortex-enabled.yaml").Spec.Template.Labels
			for k, v := range service.Spec.Selector {
				assert.Equal(t, v, podLabels[k], "Service selector %q must match the pod labels", k)
			}
		})
	}
}

// An inline token becomes a chart-managed Secret; an existingSecret must not produce one.
func TestVortexSecret(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("inline token renders a Secret", func(t *testing.T) {
				output, err := renderVortex(t, chart, "vortex-enabled.yaml", "templates/vortex-secret.yaml")
				require.NoError(t, err)

				var secret corev1.Secret
				helm.UnmarshalK8SYaml(t, output, &secret)
				assert.Equal(t, chart.fullnamePrefix()+vortexFullnameSuffix, secret.Name)
				assert.Equal(t,
					"squ_example000000000000000000000000000000",
					string(secret.Data["VORTEX_ANALYSIS_SONARQUBE_TOKEN"]))
			})

			t.Run("existingSecret renders none and is referenced directly", func(t *testing.T) {
				output, err := renderVortex(t, chart, "vortex-existing-secret.yaml", "templates/vortex-secret.yaml")
				require.Error(t, err, "no Secret may be rendered when the token comes from an existingSecret")
				assert.Empty(t, strings.TrimSpace(output))

				token := vortexContainerEnv(vortexDeployment(t, chart, "vortex-existing-secret.yaml").
					Spec.Template.Spec.Containers[0])["VORTEX_ANALYSIS_SONARQUBE_TOKEN"]
				require.NotNil(t, token.ValueFrom)
				assert.Equal(t, "my-vortex-token", token.ValueFrom.SecretKeyRef.Name)
				assert.Equal(t, "TOKEN", token.ValueFrom.SecretKeyRef.Key)
			})
		})
	}
}

// Enabling the service without an image must fail validation rather than deploy an invalid image
// reference.
func TestVortexRequiresImageRepository(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderVortex(t, chart, "vortex-no-image.yaml", "templates/vortex.yaml")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "vortex.image.repository is not set")
		})
	}
}

// The remaining settings the service cannot start without: an image tag, since the reference is
// built verbatim, a token, since every Web API call needs one, and the storage settings backing
// context restoration, which have to resolve to the store SonarQube Server writes the items to.
func TestVortexRequiresTagAndToken(t *testing.T) {
	cases := map[string]struct {
		unset    map[string]string
		expected string
	}{
		"image tag":      {map[string]string{"vortex.image.tag": ""}, "vortex.image.tag is not set"},
		"token":          {map[string]string{"vortex.sonarqubeToken.token": ""}, "no token is set"},
		"storage type":   {map[string]string{"vortex.storage.type": ""}, "vortex.storage.type is not set"},
		"storage bucket": {map[string]string{"vortex.storage.bucket": ""}, "vortex.storage.bucket is not set"},
		"storage region": {map[string]string{"vortex.storage.region": ""}, "vortex.storage.region is not set"},
		"storage partial credentials": {
			map[string]string{"vortex.storage.accessKey": "only-access-key"},
			"only one of vortex.storage.accessKey / vortex.storage.secretKey is set",
		},
	}

	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for name, tc := range cases {
				t.Run(name, func(t *testing.T) {
					opts := &helm.Options{
						Logger:      logger.Discard,
						ValuesFiles: []string{chart.valuesDir + "/vortex-enabled.yaml"},
						SetValues:   tc.unset,
					}
					_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
					require.Error(t, err)
					assert.Contains(t, err.Error(), tc.expected)
				})
			}
		})
	}
}

// vortexAppNodePolicy picks the application nodes' policy out of a networkpolicy.yaml render. The
// two charts use different pod-selector labels for their app pods.
func vortexAppNodePolicy(t *testing.T, chart agentChart, manifest string) networkingv1.NetworkPolicy {
	t.Helper()
	for _, doc := range vortexDocs(manifest) {
		if vortexDocKind(doc) != "NetworkPolicy" {
			continue
		}
		var candidate networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, doc, &candidate)
		if chart.name == "sonarqube-dce" {
			if candidate.Spec.PodSelector.MatchLabels["sonarqube.datacenter/type"] == "app" {
				return candidate
			}
			continue
		}
		if strings.HasSuffix(candidate.Name, "-network-policy") {
			return candidate
		}
	}
	require.FailNow(t, "the app-node NetworkPolicy must render")
	return networkingv1.NetworkPolicy{}
}

func vortexSelectsApp(peers []networkingv1.NetworkPolicyPeer, appLabel string) bool {
	for _, peer := range peers {
		if peer.PodSelector != nil && peer.PodSelector.MatchLabels["app"] == appLabel {
			return true
		}
	}
	return false
}

// Ports of the ingress rule admitting appLabel, or nil when no rule does.
func vortexIngressPorts(rules []networkingv1.NetworkPolicyIngressRule, appLabel string) []networkingv1.NetworkPolicyPort {
	for _, rule := range rules {
		if vortexSelectsApp(rule.From, appLabel) {
			return rule.Ports
		}
	}
	return nil
}

// Ports of the egress rule targeting appLabel, or nil when no rule does.
func vortexEgressPorts(rules []networkingv1.NetworkPolicyEgressRule, appLabel string) []networkingv1.NetworkPolicyPort {
	for _, rule := range rules {
		if vortexSelectsApp(rule.To, appLabel) {
			return rule.Ports
		}
	}
	return nil
}

// True if rules already let the app nodes reach anywhere with no port restriction. The sonarqube
// chart has no per-target egress rules like DCE's - it egresses unrestricted to 0.0.0.0/0 and
// ::/0, so a Vortex-specific rule would be redundant (see networkpolicy.yaml).
func vortexHasUnrestrictedEgress(rules []networkingv1.NetworkPolicyEgressRule) bool {
	for _, rule := range rules {
		if len(rule.Ports) != 0 {
			continue
		}
		for _, to := range rule.To {
			if to.IPBlock != nil && to.IPBlock.CIDR == "0.0.0.0/0" {
				return true
			}
		}
	}
	return false
}

// The application nodes' own policy needs a matching ingress rule, otherwise Vortex's analysis
// callback would be dropped. Egress to Vortex differs per chart: DCE scopes it to a dedicated rule,
// while the sonarqube chart already egresses unrestricted, so no dedicated rule is expected there.
func TestVortexAppNodeNetworkPolicy(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderVortex(t, chart, "vortex-enabled.yaml", "templates/networkpolicy.yaml")
			require.NoError(t, err)

			policy := vortexAppNodePolicy(t, chart, output)
			vortexPodLabel := chart.name + vortexFullnameSuffix

			ingress := vortexIngressPorts(policy.Spec.Ingress, vortexPodLabel)
			require.Len(t, ingress, 1, "the app nodes must accept the analysis callback")
			assert.Equal(t, int32(9000), ingress[0].Port.IntVal)

			if chart.name == "sonarqube-dce" {
				egress := vortexEgressPorts(policy.Spec.Egress, vortexPodLabel)
				require.Len(t, egress, 1, "the app nodes must be allowed to reach the service on its port")
				assert.Equal(t, int32(8080), egress[0].Port.IntVal)
				return
			}
			assert.True(t, vortexHasUnrestrictedEgress(policy.Spec.Egress),
				"the app nodes must already be allowed to reach the service unrestricted")
		})
	}
}

// Two different ports are in play and mixing them up silently breaks the callback: the URL goes
// through the SonarQube Service, which listens on service.externalPort, while the NetworkPolicy
// rules are evaluated after the Service DNAT and so must use the container's service.internalPort.
func TestVortexFollowsServicePorts(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			ports := map[string]string{"service.externalPort": "80", "service.internalPort": "9500"}
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/vortex-enabled.yaml"},
				SetValues:   ports,
			}

			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/vortex.yaml"})
			require.NoError(t, err)
			var deployment appsv1.Deployment
			helm.UnmarshalK8SYaml(t, output, &deployment)
			url := vortexContainerEnv(deployment.Spec.Template.Spec.Containers[0])["VORTEX_ANALYSIS_SONARQUBE_URL"]
			assert.Equal(t, "http://"+chart.fullnamePrefix()+":80", url.Value,
				"the callback URL must use the Service port")

			output, err = helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/networkpolicy.yaml"})
			require.NoError(t, err)
			ingress := vortexIngressPorts(vortexAppNodePolicy(t, chart, output).Spec.Ingress, chart.name+vortexFullnameSuffix)
			require.Len(t, ingress, 1)
			assert.Equal(t, int32(9500), ingress[0].Port.IntVal, "ingress must use the app container port")
		})
	}
}
