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
	policyv1 "k8s.io/api/policy/v1"
)

// The Agent Egress Proxy only exists on charts/sonarqube - charts/sonarqube-dce has no
// equivalent component, so these tests are not table-driven over agentCharts.
var egressProxyChart = agentCharts[1]

func init() {
	if egressProxyChart.name != "sonarqube" || !egressProxyChart.hasEgressProxy {
		panic("agentCharts[1] is expected to be the sonarqube chart with hasEgressProxy set")
	}
}

// renderAgentEgressProxyTemplates always merges in the minimum values validation.yaml requires to
// enable a runtime family - community/monitoringPasscode, jdbcOverwrite, a valid
// agentOrchestrator, and a valid vortex (all pre-existing dependencies unrelated to this ticket:
// hunterAgent/remediationAgent both require agentOrchestrator.enabled, and remediationAgent
// additionally requires vortex.enabled) - so callers only need to set what's relevant to what
// they're testing, typically just which runtime family(ies) to enable.
func renderAgentEgressProxyTemplates(t *testing.T, setValues map[string]string, templates []string) (string, error) {
	t.Helper()
	merged := map[string]string{
		"community.enabled":                  "true",
		"monitoringPasscode":                 "test-passcode",
		"jdbcOverwrite.enabled":              "true",
		"jdbcOverwrite.jdbcUrl":              "jdbc:postgresql://test-host:5432/testdb",
		"jdbcOverwrite.jdbcUsername":         "test-user",
		"jdbcOverwrite.jdbcPassword":         "test-password",
		"agentOrchestrator.enabled":          "true",
		"agentOrchestrator.image.repository": "example.com/agent-orchestrator",
		"agentOrchestrator.image.tag":        "42",
		"agentOrchestrator.storage.bucket":   "agent-jobs",
		"vortex.enabled":                     "true",
		"vortex.image.repository":            "example.com/vortex",
		"vortex.image.tag":                   "1",
		"vortex.sonarqubeToken.token":        "squ_example000000000000000000000000000000",
		"vortex.storage.type":                "s3",
		"vortex.storage.bucket":              "vortex-artifacts",
		"vortex.storage.region":              "eu-west-1",
	}
	for k, v := range setValues {
		merged[k] = v
	}
	opts := &helm.Options{
		Logger:    logger.Discard,
		SetValues: merged,
	}
	return helm.RenderTemplateE(t, opts, egressProxyChart.path, egressProxyChart.release, templates)
}

var egressProxyTemplates = []string{
	"templates/agent-egress-proxy.yaml",
	"templates/agent-egress-proxy-service.yaml",
	"templates/agent-egress-proxy-configmap.yaml",
	"templates/agent-egress-proxy-poddisruptionbudget.yaml",
}

// D8: the proxy has no enabled toggle of its own - it must render nothing when both runtime
// families are disabled, and render fully when either one is enabled, regardless of the other.
func TestAgentEgressProxyRequired(t *testing.T) {
	t.Run("both disabled renders nothing", func(t *testing.T) {
		for _, tpl := range append(egressProxyTemplates, "templates/agent-egress-proxy-networkpolicy.yaml") {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{egressProxyChart.valuesDir + "/agent-all-disabled.yaml"},
			}
			output, err := helm.RenderTemplateE(t, opts, egressProxyChart.path, egressProxyChart.release, []string{tpl})
			require.Error(t, err, "%s must render nothing when hunterAgent/remediationAgent are both disabled", tpl)
			assert.Empty(t, strings.TrimSpace(output))
		}
	})

	t.Run("serviceaccount renders nothing when both disabled even with create: true", func(t *testing.T) {
		// With agentOrchestrator/hunterAgent/remediationAgent all disabled, every if-block in
		// agent-serviceaccount.yaml is false, so the whole file renders empty - which Helm
		// reports as an error, not an empty success (see the "both disabled renders nothing"
		// subtest above for the same convention).
		opts := &helm.Options{
			Logger:      logger.Discard,
			ValuesFiles: []string{egressProxyChart.valuesDir + "/agent-all-disabled.yaml"},
			SetValues:   map[string]string{"agentEgressProxy.serviceAccount.create": "true"},
		}
		output, err := helm.RenderTemplateE(t, opts, egressProxyChart.path, egressProxyChart.release, []string{"templates/agent-serviceaccount.yaml"})
		require.Error(t, err)
		assert.Empty(t, strings.TrimSpace(output))
	})

	for _, family := range []string{"hunterAgent", "remediationAgent"} {
		t.Run(family+" alone activates the proxy", func(t *testing.T) {
			setValues := map[string]string{
				family + ".enabled":                      "true",
				family + ".image.repository":             "example.com/" + family,
				family + ".image.tag":                    "1",
				"agentEgressProxy.serviceAccount.create": "true",
			}
			output, err := renderAgentEgressProxyTemplates(t, setValues, egressProxyTemplates)
			require.NoError(t, err)
			assert.Contains(t, output, "kind: Deployment")
			assert.Contains(t, output, "kind: Service")
			assert.Contains(t, output, "kind: ConfigMap")
			assert.Contains(t, output, "kind: PodDisruptionBudget")

			saOutput, err := renderAgentEgressProxyTemplates(t, setValues, []string{"templates/agent-serviceaccount.yaml"})
			require.NoError(t, err)
			assert.Contains(t, saOutput, "agent-egress-proxy")
		})
	}
}

func renderAgentEgressProxyDeployment(t *testing.T, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	merged := map[string]string{
		"hunterAgent.enabled":          "true",
		"hunterAgent.image.repository": "example.com/hunter-agent",
		"hunterAgent.image.tag":        "1",
	}
	for k, v := range setValues {
		merged[k] = v
	}
	output, err := renderAgentEgressProxyTemplates(t, merged, []string{"templates/agent-egress-proxy.yaml"})
	require.NoError(t, err)

	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	require.NotEmpty(t, deployment.Name)
	return deployment
}

func TestAgentEgressProxyDeploymentDefaults(t *testing.T) {
	deployment := renderAgentEgressProxyDeployment(t, nil)

	assert.Equal(t, int32(2), *deployment.Spec.Replicas, "D7: two replicas by default")

	podSpec := deployment.Spec.Template.Spec
	require.Len(t, podSpec.Containers, 1)
	container := podSpec.Containers[0]

	require.NotNil(t, container.ReadinessProbe)
	require.NotNil(t, container.ReadinessProbe.TCPSocket, "Squid has no HTTP health endpoint")
	require.NotNil(t, container.LivenessProbe)
	require.NotNil(t, container.LivenessProbe.TCPSocket)

	volumeNames := map[string]bool{}
	for _, v := range podSpec.Volumes {
		volumeNames[v.Name] = true
	}
	for _, want := range []string{"squid-conf", "squid-spool", "squid-run", "tmp"} {
		assert.True(t, volumeNames[want], "expected volume %q", want)
	}
}

// squid.conf must reflect allowedDomains, extraSafePorts, and requestBodyMaxSizeMb.
func TestAgentEgressProxyConfigMapContent(t *testing.T) {
	setValues := map[string]string{
		"hunterAgent.enabled":                   "true",
		"hunterAgent.image.repository":          "example.com/hunter-agent",
		"hunterAgent.image.tag":                 "1",
		"agentEgressProxy.allowedDomains[0]":    ".anthropic.com",
		"agentEgressProxy.allowedDomains[1]":    "example.org",
		"agentEgressProxy.extraSafePorts[0]":    "8443",
		"agentEgressProxy.requestBodyMaxSizeMb": "42",
	}
	output, err := renderAgentEgressProxyTemplates(t, setValues, []string{"templates/agent-egress-proxy-configmap.yaml"})
	require.NoError(t, err)

	var cm corev1.ConfigMap
	helm.UnmarshalK8SYaml(t, output, &cm)
	conf := cm.Data["squid.conf"]

	assert.Contains(t, conf, "acl allowed_domains dstdomain .anthropic.com example.org")
	assert.Contains(t, conf, "acl Safe_ports port 8443")
	assert.Contains(t, conf, "request_body_max_size 42 MB")
}

// SonarQube's /rules/show and /agentic-analysis endpoints must always be reachable through the
// proxy, hardcoded independently of allowedDomains - there is no values key that can remove this
// allow rule, unlike everything in allowedDomains.
func TestAgentEgressProxyAlwaysAllowsSonarQubeAgenticEndpoints(t *testing.T) {
	t.Run("present regardless of allowedDomains", func(t *testing.T) {
		setValues := map[string]string{
			"hunterAgent.enabled":          "true",
			"hunterAgent.image.repository": "example.com/hunter-agent",
			"hunterAgent.image.tag":        "1",
			// allowedDomains left empty on purpose: the SonarQube allow rule must not depend on it.
		}
		output, err := renderAgentEgressProxyTemplates(t, setValues, []string{"templates/agent-egress-proxy-configmap.yaml"})
		require.NoError(t, err)

		var cm corev1.ConfigMap
		helm.UnmarshalK8SYaml(t, output, &cm)
		conf := cm.Data["squid.conf"]

		// Fully qualified, not the bare short name - Squid's own DNS resolver doesn't honour
		// /etc/resolv.conf's search list, so a bare Service name can never resolve for it.
		assert.Contains(t, conf, "acl sonarqube_host dstdomain "+egressProxyChart.fullnamePrefix()+".default.svc.cluster.local")
		assert.Contains(t, conf, "acl sonarqube_agentic_endpoints urlpath_regex /rules/show(\\?|$) /agentic-analysis(/|\\?|$)")
		assert.Contains(t, conf, "http_access allow sonarqube_host sonarqube_agentic_endpoints")
		assert.Contains(t, conf, "acl Safe_ports port 9000", "SonarQube's default externalPort must be reachable too")
	})

	t.Run("dstdomain tracks the fullname prefix and service.externalPort overrides", func(t *testing.T) {
		setValues := map[string]string{
			"hunterAgent.enabled":          "true",
			"hunterAgent.image.repository": "example.com/hunter-agent",
			"hunterAgent.image.tag":        "1",
			"service.externalPort":         "9001",
		}
		output, err := renderAgentEgressProxyTemplates(t, setValues, []string{"templates/agent-egress-proxy-configmap.yaml"})
		require.NoError(t, err)

		var cm corev1.ConfigMap
		helm.UnmarshalK8SYaml(t, output, &cm)
		conf := cm.Data["squid.conf"]

		assert.Contains(t, conf, "acl sonarqube_host dstdomain "+egressProxyChart.fullnamePrefix()+".default.svc.cluster.local")
		assert.Contains(t, conf, "acl Safe_ports port 9001")
	})
}

func TestAgentEgressProxyPodDisruptionBudget(t *testing.T) {
	setValues := map[string]string{
		"hunterAgent.enabled":                               "true",
		"hunterAgent.image.repository":                      "example.com/hunter-agent",
		"hunterAgent.image.tag":                             "1",
		"agentEgressProxy.podDisruptionBudget.minAvailable": "3",
	}
	output, err := renderAgentEgressProxyTemplates(t, setValues, []string{"templates/agent-egress-proxy-poddisruptionbudget.yaml"})
	require.NoError(t, err)

	var pdb policyv1.PodDisruptionBudget
	helm.UnmarshalK8SYaml(t, output, &pdb)
	require.NotNil(t, pdb.Spec.MinAvailable)
	assert.Equal(t, "3", pdb.Spec.MinAvailable.String())
}

// The proxy's own NetworkPolicy is opt-in (agentEgressProxy.networkPolicy.enabled), independent
// of D8's auto-activation of the Deployment/Service/ConfigMap.
func TestAgentEgressProxyNetworkPolicy(t *testing.T) {
	t.Run("disabled by default even when the proxy is active", func(t *testing.T) {
		setValues := map[string]string{
			"hunterAgent.enabled":          "true",
			"hunterAgent.image.repository": "example.com/hunter-agent",
			"hunterAgent.image.tag":        "1",
		}
		output, err := renderAgentEgressProxyTemplates(t, setValues, []string{"templates/agent-egress-proxy-networkpolicy.yaml"})
		require.Error(t, err)
		assert.Empty(t, strings.TrimSpace(output))
	})

	t.Run("enabled selects both runtime families on ingress and 0.0.0.0/0 on egress", func(t *testing.T) {
		setValues := map[string]string{
			"hunterAgent.enabled":                    "true",
			"hunterAgent.image.repository":           "example.com/hunter-agent",
			"hunterAgent.image.tag":                  "1",
			"remediationAgent.enabled":               "true",
			"remediationAgent.image.repository":      "example.com/remediation-agent",
			"remediationAgent.image.tag":             "1",
			"agentEgressProxy.networkPolicy.enabled": "true",
		}
		output, err := renderAgentEgressProxyTemplates(t, setValues, []string{"templates/agent-egress-proxy-networkpolicy.yaml"})
		require.NoError(t, err)

		var policy networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, output, &policy)

		require.Len(t, policy.Spec.Ingress, 1)
		ingress := policy.Spec.Ingress[0]
		require.Len(t, ingress.From, 1)
		require.NotNil(t, ingress.From[0].PodSelector)
		assert.Equal(t, "runtime", ingress.From[0].PodSelector.MatchLabels["sonarqube.agent/component"])
		assert.NotContains(t, ingress.From[0].PodSelector.MatchLabels, "sonarqube.agent/family",
			"must select both families, not just one")

		require.Len(t, policy.Spec.Egress, 3, "DNS, the SonarQube pod rule, plus the broad 0.0.0.0/0 rule")
		var broad, sonarqube *networkingv1.NetworkPolicyEgressRule
		for i := range policy.Spec.Egress {
			rule := policy.Spec.Egress[i]
			if len(rule.To) == 1 && rule.To[0].IPBlock != nil {
				broad = &rule
			}
			if len(rule.To) == 1 && rule.To[0].PodSelector != nil && rule.To[0].PodSelector.MatchLabels["app"] == "sonarqube" {
				sonarqube = &rule
			}
		}
		require.NotNil(t, broad, "expected a broad ipBlock egress rule")
		assert.Equal(t, "0.0.0.0/0", broad.To[0].IPBlock.CIDR)
		require.Len(t, broad.Ports, 2)
		ports := []int32{broad.Ports[0].Port.IntVal, broad.Ports[1].Port.IntVal}
		assert.ElementsMatch(t, []int32{80, 443}, ports)

		require.NotNil(t, sonarqube, "expected a rule allowing egress to the SonarQube pod - "+
			"this backs the hardcoded sonarqube_host allow rule in agent-egress-proxy-configmap.yaml")
		require.Len(t, sonarqube.Ports, 1)
		assert.EqualValues(t, 9000, sonarqube.Ports[0].Port.IntVal)
	})

	t.Run("SonarQube pod egress rule is unconditional, not gated behind networkPolicy.egressPorts", func(t *testing.T) {
		setValues := map[string]string{
			"hunterAgent.enabled":          "true",
			"hunterAgent.image.repository": "example.com/hunter-agent",
			"hunterAgent.image.tag":        "1",
			// networkPolicy left disabled - the whole NetworkPolicy resource doesn't render then,
			// so this instead pins the port to a non-default value to prove it isn't read from
			// networkPolicy.egressPorts.
			"agentEgressProxy.networkPolicy.enabled":        "true",
			"agentEgressProxy.networkPolicy.egressPorts[0]": "8080",
			"service.internalPort":                          "9001",
			"service.externalPort":                          "9002",
		}
		output, err := renderAgentEgressProxyTemplates(t, setValues, []string{"templates/agent-egress-proxy-networkpolicy.yaml"})
		require.NoError(t, err)

		var policy networkingv1.NetworkPolicy
		helm.UnmarshalK8SYaml(t, output, &policy)

		var sonarqube *networkingv1.NetworkPolicyEgressRule
		for i := range policy.Spec.Egress {
			rule := policy.Spec.Egress[i]
			if len(rule.To) == 1 && rule.To[0].PodSelector != nil && rule.To[0].PodSelector.MatchLabels["app"] == "sonarqube" {
				sonarqube = &rule
			}
		}
		require.NotNil(t, sonarqube)
		require.Len(t, sonarqube.Ports, 1)
		assert.EqualValues(t, 9001, sonarqube.Ports[0].Port.IntVal, "tracks service.internalPort, not networkPolicy.egressPorts or service.externalPort")
	})
}

// Regression test: HTTP_PROXY/HTTPS_PROXY/NO_PROXY (and lowercase variants) must not be
// overridable via hunterAgent.env/remediationAgent.env - D1/D2 require the runtime to have no
// way to bypass the proxy or carve out a NO_PROXY exception.
func TestAgentEgressProxyEnvVarsNotOverridable(t *testing.T) {
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{egressProxyChart.valuesDir + "/agent-runtimes-enabled.yaml"},
		SetValues: map[string]string{
			"hunterAgent.env[0].name":  "NO_PROXY",
			"hunterAgent.env[0].value": "attacker.example.com",
			"hunterAgent.env[1].name":  "HTTP_PROXY",
			"hunterAgent.env[1].value": "http://bypass.example.com:8080",
		},
	}
	output, err := helm.RenderTemplateE(t, opts, egressProxyChart.path, egressProxyChart.release, []string{"templates/agent-runtime.yaml"})
	require.NoError(t, err)

	var deployments []appsv1.Deployment
	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var d appsv1.Deployment
		helm.UnmarshalK8SYaml(t, doc, &d)
		if strings.Contains(d.Labels["sonarqube.agent/family"], "hunter") {
			deployments = append(deployments, d)
		}
	}
	require.Len(t, deployments, 1)
	env := deployments[0].Spec.Template.Spec.Containers[0].Env

	lastValueByName := map[string]string{}
	for _, e := range env {
		lastValueByName[e.Name] = e.Value
	}

	expectedProxyURL := "http://" + egressProxyChart.fullnamePrefix() + "-agent-egress-proxy:3128"
	assert.Equal(t, expectedProxyURL, lastValueByName["HTTP_PROXY"], "attacker-supplied env must not win")
	assert.Equal(t, "", lastValueByName["NO_PROXY"], "NO_PROXY must stay forced empty")
}
