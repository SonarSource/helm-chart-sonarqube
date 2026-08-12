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

// Vortex analysis is a standalone service deployed alongside SonarQube, like the MCP server, and is
// independent of the Agentic Harness. It exists only on the Data Center chart for now.

const vortexFullnameSuffix = "-vortex-analysis"

func renderVortex(t *testing.T, fixture string, templates ...string) (string, error) {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{"test-cases-values/sonarqube-dce/" + fixture},
	}
	return helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, templates)
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

func vortexDeployment(t *testing.T, fixture string) appsv1.Deployment {
	t.Helper()
	output, err := renderVortex(t, fixture, "templates/vortex-analysis.yaml")
	require.NoError(t, err)

	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	require.NotEmpty(t, deployment.Name, "no Deployment rendered by templates/vortex-analysis.yaml")
	return deployment
}

func vortexContainerEnv(container corev1.Container) map[string]corev1.EnvVar {
	env := map[string]corev1.EnvVar{}
	for _, e := range container.Env {
		env[e.Name] = e
	}
	return env
}

// Off by default, so an existing install picks up nothing new until vortexAnalysis.enabled is set.
func TestVortexAnalysisDisabledByDefault(t *testing.T) {
	for _, tpl := range []string{
		"templates/vortex-analysis.yaml",
		"templates/vortex-analysis-service.yaml",
		"templates/vortex-analysis-secret.yaml",
		"templates/vortex-analysis-networkpolicy.yaml",
	} {
		output, err := renderVortex(t, "vortex-analysis-disabled.yaml", tpl)
		require.Error(t, err, "%s must render nothing when vortexAnalysis.enabled is false", tpl)
		assert.Empty(t, strings.TrimSpace(output))
	}
}

// It must render with no agenticHarness block configured at all.
func TestVortexAnalysisIndependentOfAgenticHarness(t *testing.T) {
	deployment := vortexDeployment(t, "vortex-analysis-enabled.yaml")
	assert.Equal(t, dceReleaseName+"-sonarqube-dce"+vortexFullnameSuffix, deployment.Name)
	assert.Equal(t, "vortex-analysis", deployment.Labels["app.kubernetes.io/component"])

	// It uses the release's own ServiceAccount, like the MCP server.
	_, err := renderVortex(t, "vortex-analysis-enabled.yaml", "templates/agentic-serviceaccount.yaml")
	require.Error(t, err, "the agentic ServiceAccount template must render nothing for a vortex-only install")
}

// The property that activates analysis on the SonarQube side is sonar.vortex.analysis.url, injected
// as SONAR_VORTEX_ANALYSIS_URL. Absent when the service isn't deployed — unset means SonarQube routes
// no analysis requests, rather than failing to boot.
func TestVortexAnalysisAppNodeWiring(t *testing.T) {
	t.Run("set when enabled", func(t *testing.T) {
		output, err := renderVortex(t, "vortex-analysis-enabled.yaml", "templates/sonarqube-application.yaml")
		require.NoError(t, err)

		var app appsv1.Deployment
		helm.UnmarshalK8SYaml(t, output, &app)
		env := vortexContainerEnv(app.Spec.Template.Spec.Containers[0])

		vortex, ok := env["SONAR_VORTEX_ANALYSIS_URL"]
		require.True(t, ok, "app nodes must receive SONAR_VORTEX_ANALYSIS_URL when the service is enabled")
		assert.Equal(t, "http://"+dceReleaseName+"-sonarqube-dce"+vortexFullnameSuffix+":8080", vortex.Value)
	})

	t.Run("absent when disabled", func(t *testing.T) {
		output, err := renderVortex(t, "vortex-analysis-disabled.yaml", "templates/sonarqube-application.yaml")
		require.NoError(t, err)

		var app appsv1.Deployment
		helm.UnmarshalK8SYaml(t, output, &app)
		_, ok := vortexContainerEnv(app.Spec.Template.Spec.Containers[0])["SONAR_VORTEX_ANALYSIS_URL"]
		assert.False(t, ok, "SONAR_VORTEX_ANALYSIS_URL must not be set when the service is disabled")
	})
}

// Vortex analysis calls the SonarQube Web API, so it needs the SonarQube URL and a token. User
// supplied env comes last, so it can override the values the chart wires automatically.
func TestVortexAnalysisContainerEnv(t *testing.T) {
	container := vortexDeployment(t, "vortex-analysis-enabled.yaml").Spec.Template.Spec.Containers[0]
	env := vortexContainerEnv(container)

	url, ok := env["VORTEX_ANALYSIS_SONARQUBE_URL"]
	require.True(t, ok)
	assert.Equal(t, "http://"+dceReleaseName+"-sonarqube-dce:9000", url.Value)

	token, ok := env["VORTEX_ANALYSIS_SONARQUBE_TOKEN"]
	require.True(t, ok, "the callback token must be wired when a token is configured")
	require.NotNil(t, token.ValueFrom)
	require.NotNil(t, token.ValueFrom.SecretKeyRef)
	assert.Equal(t, dceReleaseName+"-sonarqube-dce"+vortexFullnameSuffix, token.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "VORTEX_ANALYSIS_SONARQUBE_TOKEN", token.ValueFrom.SecretKeyRef.Key)

	extra, ok := env["MY_VAR"]
	require.True(t, ok, "extra env must reach the container")
	assert.Equal(t, "my-value", extra.Value)
	assert.Equal(t, "MY_VAR", container.Env[len(container.Env)-1].Name,
		"user-supplied env must come last so it overrides the auto-wired vars")
}

// The pod talks to the Web API with its own token and never to the Kubernetes API, so it must not
// get a ServiceAccount token mounted.
func TestVortexAnalysisDoesNotAutomountServiceAccountToken(t *testing.T) {
	podSpec := vortexDeployment(t, "vortex-analysis-enabled.yaml").Spec.Template.Spec
	require.NotNil(t, podSpec.AutomountServiceAccountToken)
	assert.False(t, *podSpec.AutomountServiceAccountToken)
}

// The token reaches the container through secretKeyRef, which is only read at startup. Without a
// checksum annotation, rotating it would leave the running pod on the old token.
func TestVortexAnalysisRollsOnTokenChange(t *testing.T) {
	annotations := vortexDeployment(t, "vortex-analysis-enabled.yaml").Spec.Template.Annotations
	checksum, ok := annotations["checksum/secret"]
	require.True(t, ok, "the pod template must carry a checksum of the token Secret")

	// A different token must produce a different pod template, so the upgrade rolls the pod.
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{"test-cases-values/sonarqube-dce/vortex-analysis-enabled.yaml"},
		SetValues:   map[string]string{"vortexAnalysis.sonarqubeToken.token": "squ_rotated0000000000000000000000000000000"},
	}
	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/vortex-analysis.yaml"})
	require.NoError(t, err)

	var rotated appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &rotated)
	assert.NotEqual(t, checksum, rotated.Spec.Template.Annotations["checksum/secret"])
}

// Recreate rather than the default RollingUpdate, so an upgrade never runs two pods at once.
func TestVortexAnalysisUsesRecreateStrategy(t *testing.T) {
	deployment := vortexDeployment(t, "vortex-analysis-enabled.yaml")
	assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deployment.Spec.Strategy.Type)
}

// Only the Vortex analysis image pull secrets are applied. The application nodes' ones are not
// reused, so the pod never references registry credentials that were not configured for it.
func TestVortexAnalysisPullSecrets(t *testing.T) {
	podSpec := vortexDeployment(t, "vortex-analysis-enabled.yaml").Spec.Template.Spec

	var names []string
	for _, s := range podSpec.ImagePullSecrets {
		names = append(names, s.Name)
	}
	assert.Equal(t, []string{"vortex-secret", "vortex-secret-list"}, names,
		"only the Vortex analysis pull secrets may be applied")
}

// Vortex analysis can be scheduled apart from the SonarQube pods through its own nodeSelector,
// affinity and tolerations.
func TestVortexAnalysisScheduling(t *testing.T) {
	podSpec := vortexDeployment(t, "vortex-analysis-enabled.yaml").Spec.Template.Spec

	assert.Equal(t, map[string]string{"vortex": "true"}, podSpec.NodeSelector)

	require.Len(t, podSpec.Tolerations, 1)
	assert.Equal(t, "vortex", podSpec.Tolerations[0].Key)

	require.NotNil(t, podSpec.Affinity)
	require.NotNil(t, podSpec.Affinity.NodeAffinity)
	terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	require.Len(t, terms, 1)
	require.Len(t, terms[0].MatchExpressions, 1)
	assert.Equal(t, "vortex", terms[0].MatchExpressions[0].Key)
}

// The global settings take precedence over the Vortex analysis ones, matching how the application
// and search nodes behave.
func TestVortexAnalysisGlobalSchedulingWins(t *testing.T) {
	podSpec := vortexDeployment(t, "vortex-analysis-global-scheduling.yaml").Spec.Template.Spec

	assert.Equal(t, map[string]string{"global": "true"}, podSpec.NodeSelector)

	require.Len(t, podSpec.Tolerations, 1)
	assert.Equal(t, "global", podSpec.Tolerations[0].Key)

	require.NotNil(t, podSpec.Affinity)
	terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	require.Len(t, terms, 1)
	require.Len(t, terms[0].MatchExpressions, 1)
	assert.Equal(t, "global", terms[0].MatchExpressions[0].Key)
}

// The pod can take several minutes to become ready on a first start, so the startup probe must allow
// for that before the liveness probe begins.
func TestVortexAnalysisProbes(t *testing.T) {
	container := vortexDeployment(t, "vortex-analysis-enabled.yaml").Spec.Template.Spec.Containers[0]

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
}

// The Service must select the Deployment's pods — a label typo here silently yields no endpoints.
func TestVortexAnalysisService(t *testing.T) {
	output, err := renderVortex(t, "vortex-analysis-enabled.yaml", "templates/vortex-analysis-service.yaml")
	require.NoError(t, err)

	var service corev1.Service
	helm.UnmarshalK8SYaml(t, output, &service)
	assert.Equal(t, dceReleaseName+"-sonarqube-dce"+vortexFullnameSuffix, service.Name)
	require.Len(t, service.Spec.Ports, 1)
	assert.Equal(t, int32(8080), service.Spec.Ports[0].Port)

	podLabels := vortexDeployment(t, "vortex-analysis-enabled.yaml").Spec.Template.Labels
	for k, v := range service.Spec.Selector {
		assert.Equal(t, v, podLabels[k], "Service selector %q must match the pod labels", k)
	}
}

// An inline token becomes a chart-managed Secret; an existingSecret must not produce one.
func TestVortexAnalysisSecret(t *testing.T) {
	t.Run("inline token renders a Secret", func(t *testing.T) {
		output, err := renderVortex(t, "vortex-analysis-enabled.yaml", "templates/vortex-analysis-secret.yaml")
		require.NoError(t, err)

		var secret corev1.Secret
		helm.UnmarshalK8SYaml(t, output, &secret)
		assert.Equal(t, dceReleaseName+"-sonarqube-dce"+vortexFullnameSuffix, secret.Name)
		assert.Equal(t,
			"squ_example000000000000000000000000000000",
			string(secret.Data["VORTEX_ANALYSIS_SONARQUBE_TOKEN"]))
	})

	t.Run("existingSecret renders none and is referenced directly", func(t *testing.T) {
		output, err := renderVortex(t, "vortex-analysis-existing-secret.yaml", "templates/vortex-analysis-secret.yaml")
		require.Error(t, err, "no Secret may be rendered when the token comes from an existingSecret")
		assert.Empty(t, strings.TrimSpace(output))

		token := vortexContainerEnv(vortexDeployment(t, "vortex-analysis-existing-secret.yaml").
			Spec.Template.Spec.Containers[0])["VORTEX_ANALYSIS_SONARQUBE_TOKEN"]
		require.NotNil(t, token.ValueFrom)
		assert.Equal(t, "my-vortex-token", token.ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, "TOKEN", token.ValueFrom.SecretKeyRef.Key)
	})
}

// Enabling the service without an image must fail validation rather than deploy an invalid image
// reference.
func TestVortexAnalysisRequiresImageRepository(t *testing.T) {
	_, err := renderVortex(t, "vortex-analysis-no-image.yaml", "templates/vortex-analysis.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vortexAnalysis.image.repository is not set")
}

// The remaining settings the service cannot start without: an image tag, since the reference is
// built verbatim, and a token, since every Web API call needs one.
func TestVortexAnalysisRequiresTagAndToken(t *testing.T) {
	for name, tc := range map[string]struct {
		unset    map[string]string
		expected string
	}{
		"image tag": {map[string]string{"vortexAnalysis.image.tag": ""}, "vortexAnalysis.image.tag is not set"},
		"token":     {map[string]string{"vortexAnalysis.sonarqubeToken.token": ""}, "no token is set"},
	} {
		t.Run(name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{"test-cases-values/sonarqube-dce/vortex-analysis-enabled.yaml"},
				SetValues:   tc.unset,
			}
			_, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/vortex-analysis.yaml"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expected)
		})
	}
}

// Reachable only from the application nodes, and allowed out only to DNS, the application nodes and
// any configured egressAllow entries.
func TestVortexAnalysisNetworkPolicy(t *testing.T) {
	output, err := renderVortex(t, "vortex-analysis-enabled.yaml", "templates/vortex-analysis-networkpolicy.yaml")
	require.NoError(t, err)

	var policy networkingv1.NetworkPolicy
	helm.UnmarshalK8SYaml(t, output, &policy)
	require.NotEmpty(t, policy.Name)

	podLabels := vortexDeployment(t, "vortex-analysis-enabled.yaml").Spec.Template.Labels
	for k, v := range policy.Spec.PodSelector.MatchLabels {
		assert.Equal(t, v, podLabels[k], "policy podSelector %q must match the pod labels", k)
	}

	require.Len(t, policy.Spec.Ingress, 1, "only the app nodes may reach the analysis pod")
	require.Len(t, policy.Spec.Ingress[0].From, 1)
	assert.Equal(t, "app",
		policy.Spec.Ingress[0].From[0].PodSelector.MatchLabels["sonarqube.datacenter/type"])
	require.Len(t, policy.Spec.Ingress[0].Ports, 1)
	assert.Equal(t, int32(8080), policy.Spec.Ingress[0].Ports[0].Port.IntVal)

	// DNS, the app nodes on 9000 for the callback, and the one egressAllow entry.
	require.Len(t, policy.Spec.Egress, 3)
	assert.Equal(t, "app",
		policy.Spec.Egress[1].To[0].PodSelector.MatchLabels["sonarqube.datacenter/type"])
	assert.Equal(t, int32(9000), policy.Spec.Egress[1].Ports[0].Port.IntVal)
	require.NotNil(t, policy.Spec.Egress[2].To[0].IPBlock)
	assert.Equal(t, "10.0.0.0/8", policy.Spec.Egress[2].To[0].IPBlock.CIDR)
}

// An egressAllow entry may carry a podSelector, a namespaceSelector or both; each shape must become
// one peer, since two peers would widen the rule instead of narrowing it.
func TestVortexAnalysisEgressAllowSelectors(t *testing.T) {
	output, err := renderVortex(t, "vortex-analysis-existing-secret.yaml", "templates/vortex-analysis-networkpolicy.yaml")
	require.NoError(t, err)

	var policy networkingv1.NetworkPolicy
	helm.UnmarshalK8SYaml(t, output, &policy)
	require.Len(t, policy.Spec.Egress, 4, "DNS, the app nodes and the two egressAllow entries")

	namespaceOnly := policy.Spec.Egress[2]
	require.Len(t, namespaceOnly.To, 1)
	assert.Nil(t, namespaceOnly.To[0].PodSelector, "a namespaceSelector entry must not emit an empty podSelector")
	assert.Equal(t, "egress-gateway", namespaceOnly.To[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])

	both := policy.Spec.Egress[3]
	require.Len(t, both.To, 1, "both selectors must combine into a single peer")
	assert.Equal(t, "proxy", both.To[0].PodSelector.MatchLabels["app"])
	assert.Equal(t, "egress-gateway", both.To[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])
	require.Len(t, both.Ports, 1)
	assert.Equal(t, int32(8443), both.Ports[0].Port.IntVal)
}

// The dedicated toggle renders the pod's policy without the top-level networkPolicy.enabled.
func TestVortexAnalysisStandaloneNetworkPolicyToggle(t *testing.T) {
	output, err := renderVortex(t, "vortex-analysis-existing-secret.yaml", "templates/vortex-analysis-networkpolicy.yaml")
	require.NoError(t, err)

	var policy networkingv1.NetworkPolicy
	helm.UnmarshalK8SYaml(t, output, &policy)
	assert.Equal(t, dceReleaseName+"-sonarqube-dce"+vortexFullnameSuffix, policy.Name)

	// The app-node policy is not rendered at all here (global networkPolicy.enabled is false).
	_, err = renderVortex(t, "vortex-analysis-existing-secret.yaml", "templates/networkpolicy.yaml")
	require.Error(t, err)
}

// vortexAppNodePolicy picks the application nodes' policy out of a networkpolicy.yaml render.
func vortexAppNodePolicy(t *testing.T, manifest string) networkingv1.NetworkPolicy {
	t.Helper()
	for _, doc := range vortexDocs(manifest) {
		if vortexDocKind(doc) != "NetworkPolicy" {
			continue
		}
		var candidate networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, doc, &candidate)
		if candidate.Spec.PodSelector.MatchLabels["sonarqube.datacenter/type"] == "app" {
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

// The application nodes' own policy needs matching rules, otherwise they could neither reach Vortex
// analysis nor accept its calls back to the Web API.
func TestVortexAnalysisAppNodeNetworkPolicy(t *testing.T) {
	output, err := renderVortex(t, "vortex-analysis-enabled.yaml", "templates/networkpolicy.yaml")
	require.NoError(t, err)

	policy := vortexAppNodePolicy(t, output)
	vortexPodLabel := "sonarqube-dce" + vortexFullnameSuffix

	ingress := vortexIngressPorts(policy.Spec.Ingress, vortexPodLabel)
	require.Len(t, ingress, 1, "the app nodes must accept the analysis callback")
	assert.Equal(t, int32(9000), ingress[0].Port.IntVal)

	egress := vortexEgressPorts(policy.Spec.Egress, vortexPodLabel)
	require.Len(t, egress, 1, "the app nodes must be allowed to reach the service on its port")
	assert.Equal(t, int32(8080), egress[0].Port.IntVal)
}

// Two different ports are in play and mixing them up silently breaks the callback: the URL goes
// through the SonarQube Service, which listens on service.externalPort, while the NetworkPolicy
// rules are evaluated after the Service DNAT and so must use the container's service.internalPort.
func TestVortexAnalysisFollowsServicePorts(t *testing.T) {
	ports := map[string]string{"service.externalPort": "80", "service.internalPort": "9500"}
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{"test-cases-values/sonarqube-dce/vortex-analysis-enabled.yaml"},
		SetValues:   ports,
	}

	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/vortex-analysis.yaml"})
	require.NoError(t, err)
	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	url := vortexContainerEnv(deployment.Spec.Template.Spec.Containers[0])["VORTEX_ANALYSIS_SONARQUBE_URL"]
	assert.Equal(t, "http://"+dceReleaseName+"-sonarqube-dce:80", url.Value,
		"the callback URL must use the Service port")

	output, err = helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/vortex-analysis-networkpolicy.yaml"})
	require.NoError(t, err)
	var policy networkingv1.NetworkPolicy
	helm.UnmarshalK8SYaml(t, output, &policy)
	egress := vortexEgressPorts(policy.Spec.Egress, "sonarqube-dce")
	require.Len(t, egress, 1)
	assert.Equal(t, int32(9500), egress[0].Port.IntVal, "egress must use the app container port")

	output, err = helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/networkpolicy.yaml"})
	require.NoError(t, err)
	ingress := vortexIngressPorts(vortexAppNodePolicy(t, output).Spec.Ingress, "sonarqube-dce"+vortexFullnameSuffix)
	require.Len(t, ingress, 1)
	assert.Equal(t, int32(9500), ingress[0].Port.IntVal, "ingress must use the app container port")
}
