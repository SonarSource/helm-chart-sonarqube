package tests

import (
	"regexp"
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

// egressProxyCharts is the subset of agentCharts with an Agent Egress Proxy component. Every test
// in this file loops over it instead of pinning to one chart, since the proxy now exists on both.
var egressProxyCharts = func() []agentChart {
	var charts []agentChart
	for _, c := range agentCharts {
		if c.hasEgressProxy {
			charts = append(charts, c)
		}
	}
	return charts
}()

// renderAgentEgressProxyTemplates always merges in the minimum values validation.yaml requires to
// enable a runtime family - community/monitoringPasscode, jdbcOverwrite, a valid
// agentOrchestrator, and a valid vortex (all pre-existing dependencies unrelated to this ticket:
// hunterAgent/remediationAgent both require agentOrchestrator.enabled, and remediationAgent
// additionally requires vortexAnalysis.enabled) - so callers only need to set what's relevant to what
// they're testing, typically just which runtime family(ies) to enable.
func renderAgentEgressProxyTemplates(t *testing.T, chart agentChart, setValues map[string]string, templates []string) (string, error) {
	t.Helper()
	return renderAgentEgressProxyTemplatesAs(t, chart, chart.release, setValues, templates)
}

// renderAgentEgressProxyTemplatesAs is renderAgentEgressProxyTemplates with the release name under
// the caller's control, for the tests that are about how a release name shapes rendered names.
func renderAgentEgressProxyTemplatesAs(t *testing.T, chart agentChart, release string, setValues map[string]string, templates []string) (string, error) {
	t.Helper()
	merged := map[string]string{
		"community.enabled":                  "true",
		"monitoringPasscode":                 "test-passcode",
		"applicationNodes.jwtSecret":         "test-jwt-secret",
		"jdbcOverwrite.enabled":              "true",
		"jdbcOverwrite.jdbcUrl":              "jdbc:postgresql://test-host:5432/testdb",
		"jdbcOverwrite.jdbcUsername":         "test-user",
		"jdbcOverwrite.jdbcPassword":         "test-password",
		"agentOrchestrator.enabled":          "true",
		"agentOrchestrator.image.repository": "example.com/agent-orchestrator",
		"agentOrchestrator.image.tag":        "42",
		"agentOrchestrator.storage.bucket":   "agent-jobs",
		"vortexAnalysis.enabled":             "true",
		"vortexAnalysis.image.repository":    "example.com/vortex",
		"vortexAnalysis.image.tag":           "1",
		"vortexAnalysis.storage.type":        "s3",
		"vortexAnalysis.storage.bucket":      "vortex-artifacts",
		"vortexAnalysis.storage.region":      "eu-west-1",
		// Enabling any runtime family also enables agentic signing, which validation.yaml fails
		// closed on without an instance secret to derive the per-hop keys from.
		"agenticSigningSecret.existingSecret": "test-agentic-instance-secret",
	}
	for k, v := range setValues {
		merged[k] = v
	}
	opts := &helm.Options{
		Logger:    logger.Discard,
		SetValues: merged,
	}
	return helm.RenderTemplateE(t, opts, chart.path, release, templates)
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
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			chart := chart

			t.Run("both disabled renders nothing", func(t *testing.T) {
				for _, tpl := range append(egressProxyTemplates, "templates/agent-egress-proxy-networkpolicy.yaml") {
					opts := &helm.Options{
						Logger:      logger.Discard,
						ValuesFiles: []string{chart.valuesDir + "/agent-all-disabled.yaml"},
					}
					output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{tpl})
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
					ValuesFiles: []string{chart.valuesDir + "/agent-all-disabled.yaml"},
					SetValues:   map[string]string{"agentEgressProxy.serviceAccount.create": "true"},
				}
				output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-serviceaccount.yaml"})
				require.Error(t, err)
				assert.Empty(t, strings.TrimSpace(output))
			})

			for _, family := range []string{"hunterAgent", "remediationAgent"} {
				family := family
				t.Run(family+" alone activates the proxy", func(t *testing.T) {
					setValues := map[string]string{
						family + ".enabled":                      "true",
						family + ".image.repository":             "example.com/" + family,
						family + ".image.tag":                    "1",
						"agentEgressProxy.serviceAccount.create": "true",
					}
					output, err := renderAgentEgressProxyTemplates(t, chart, setValues, egressProxyTemplates)
					require.NoError(t, err)
					assert.Contains(t, output, "kind: Deployment")
					assert.Contains(t, output, "kind: Service")
					assert.Contains(t, output, "kind: ConfigMap")
					assert.Contains(t, output, "kind: PodDisruptionBudget")

					saOutput, err := renderAgentEgressProxyTemplates(t, chart, setValues, []string{"templates/agent-serviceaccount.yaml"})
					require.NoError(t, err)
					assert.Contains(t, saOutput, "agent-egress-proxy")
				})
			}
		})
	}
}

func renderAgentEgressProxyDeployment(t *testing.T, chart agentChart, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	merged := map[string]string{
		"hunterAgent.enabled":          "true",
		"hunterAgent.image.repository": "example.com/hunter-agent",
		"hunterAgent.image.tag":        "1",
	}
	for k, v := range setValues {
		merged[k] = v
	}
	output, err := renderAgentEgressProxyTemplates(t, chart, merged, []string{"templates/agent-egress-proxy.yaml"})
	require.NoError(t, err)

	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	require.NotEmpty(t, deployment.Name)
	return deployment
}

func TestAgentEgressProxyDeploymentDefaults(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := renderAgentEgressProxyDeployment(t, chart, nil)

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
		})
	}
}

// squid.conf must reflect allowedDomains, and must not carry a request body cap: HTTPS is
// CONNECT-tunnelled and opaque to Squid, so any such cap would apply only to the cleartext calls
// to SonarQube's always-reachable endpoints - a silent ceiling on a path the chart guarantees.
func TestAgentEgressProxyConfigMapContent(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{
				"hunterAgent.enabled":                "true",
				"hunterAgent.image.repository":       "example.com/hunter-agent",
				"hunterAgent.image.tag":              "1",
				"agentEgressProxy.allowedDomains[0]": ".anthropic.com",
				"agentEgressProxy.allowedDomains[1]": "example.org",
			}
			conf := renderAgentEgressProxySquidConf(t, chart, setValues)

			assert.Contains(t, conf, "acl allowed_domains dstdomain .anthropic.com example.org")
			assert.Contains(t, conf, "http_access allow allowed_domains")
			// Matched per-directive rather than with NotContains: a comment in squid.conf explains why
			// the cap is absent, and naming the directive there keeps it greppable.
			assert.False(t, squidConfHasDirective(conf, "request_body_max_size"),
				"squid.conf must not cap request bodies")
		})
	}
}

// squidConfHasDirective reports whether squid.conf actually sets a directive, ignoring comments.
func squidConfHasDirective(conf, directive string) bool {
	for _, line := range strings.Split(conf, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), directive+" ") {
			return true
		}
	}
	return false
}

// An empty allowedDomains must omit the ACL entirely rather than render a valueless
// `acl ... dstdomain`, which makes Squid log "WARNING: empty ACL" on every start and leaves an
// allow rule that can never match.
func TestAgentEgressProxyOmitsAllowedDomainsAclWhenEmpty(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{
				"hunterAgent.enabled":          "true",
				"hunterAgent.image.repository": "example.com/hunter-agent",
				"hunterAgent.image.tag":        "1",
			}
			conf := renderAgentEgressProxySquidConf(t, chart, setValues)

			assert.NotContains(t, conf, "allowed_domains")
		})
	}
}

// extraSquidConf must land *before* the final `http_access deny all`: Squid evaluates http_access
// top-down and stops at the first match, so anything after that deny is unreachable and a user's
// allow rule would silently do nothing.
func TestAgentEgressProxyExtraSquidConfPrecedesDenyAll(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{
				"hunterAgent.enabled":             "true",
				"hunterAgent.image.repository":    "example.com/hunter-agent",
				"hunterAgent.image.tag":           "1",
				"agentEgressProxy.extraSquidConf": "http_access allow example_marker",
			}
			conf := renderAgentEgressProxySquidConf(t, chart, setValues)

			marker := strings.Index(conf, "http_access allow example_marker")
			denyAll := strings.Index(conf, "http_access deny all")
			require.NotEqual(t, -1, marker, "extraSquidConf was not spliced into squid.conf")
			require.NotEqual(t, -1, denyAll, "squid.conf is missing its final deny")
			assert.Less(t, marker, denyAll, "extraSquidConf must precede `http_access deny all` to have any effect")
		})
	}
}

func renderAgentEgressProxySquidConf(t *testing.T, chart agentChart, setValues map[string]string) string {
	t.Helper()
	output, err := renderAgentEgressProxyTemplates(t, chart, setValues, []string{"templates/agent-egress-proxy-configmap.yaml"})
	require.NoError(t, err)

	var cm corev1.ConfigMap
	helm.UnmarshalK8SYaml(t, output, &cm)
	return cm.Data["squid.conf"]
}

// urlpathRegexMatcher pulls the patterns off a rendered `acl <name> urlpath_regex ...` line and
// returns a predicate that reports whether a request path would match the ACL - i.e. whether Squid
// would let the corresponding http_access allow rule fire. Squid treats each whitespace-separated
// token as its own POSIX ERE and ORs them; Go's RE2 agrees with ERE on the constructs used here, so
// compiling them directly lets a test assert on ACL *behaviour* rather than on the literal string.
func urlpathRegexMatcher(t *testing.T, squidConf string, aclName string) func(path string) bool {
	t.Helper()
	prefix := "acl " + aclName + " urlpath_regex "

	var patterns []string
	for _, line := range strings.Split(squidConf, "\n") {
		line = strings.TrimSpace(line)
		if after, found := strings.CutPrefix(line, prefix); found {
			patterns = strings.Fields(after)
			break
		}
	}
	require.NotEmpty(t, patterns, "no %q line found in the rendered squid.conf", strings.TrimSpace(prefix))

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		require.NoError(t, err, "ACL %s has an uncompilable pattern %q", aclName, pattern)
		compiled = append(compiled, re)
	}

	return func(path string) bool {
		for _, re := range compiled {
			if re.MatchString(path) {
				return true
			}
		}
		return false
	}
}

// agenticEndpointCase is one request path run through the sonarqube_agentic_endpoints ACL, with
// the verdict the ACL must reach for it.
type agenticEndpointCase struct {
	path  string
	allow bool
	why   string
}

// assertAgenticEndpointRegexMatches compiles the rendered sonarqube_agentic_endpoints ACL and runs
// real request paths through it, for one value of sonarWebContext (pass "" for the default).
func assertAgenticEndpointRegexMatches(t *testing.T, chart agentChart, webContext string) {
	t.Helper()

	setValues := map[string]string{
		"remediationAgent.enabled":          "true",
		"remediationAgent.image.repository": "example.com/remediation-agent",
		"remediationAgent.image.tag":        "1",
	}
	if webContext != "" {
		setValues["sonarWebContext"] = webContext
	}
	conf := renderAgentEgressProxySquidConf(t, chart, setValues)
	matches := urlpathRegexMatcher(t, conf, "sonarqube_agentic_endpoints")

	for _, tc := range []agenticEndpointCase{
		{webContext + "/api/rules/show?key=java:S1481", true, "the real v1 rule-lookup path"},
		{webContext + "/api/v2/a3s/private/analyses", true, "the real v2 analysis-creation path"},
		{webContext + "/rules/show?key=java:S1481", true, "unversioned form must keep matching too"},
		{webContext + "/a3s/private/analyses", true, "unversioned analysis-creation form"},
		{webContext + "/a3s/private/analyses/123", true, "a sub-resource of analyses"},
		{"/anything?x=/rules/show", false, "a query string must not smuggle the path past the scoping"},
		{"/anything?x=/a3s/private/analyses", false, "same, for a3s/private/analyses"},
		{"/rules/showdown", false, "an unrelated path that merely shares a prefix"},
		{webContext + "/api/v2/a3s/analyses", false, "missing the private/ segment is not the real endpoint"},
		{webContext + "/api/v2/a3s/contexts", false, "scoping is per-endpoint, not the whole a3s surface"},
	} {
		assert.Equal(t, tc.allow, matches(tc.path), "%s: %s", tc.path, tc.why)
	}
}

// SonarQube's rule-lookup (api/rules/show) and analysis-creation (api/v2/a3s/private/analyses)
// endpoints must always be reachable through the proxy for the Remediation runtime, hardcoded
// independently of allowedDomains - there is no values key that can remove this allow rule, unlike
// everything in allowedDomains. Hunter must never reach these endpoints (SONAR-32432): the rule is
// scoped to the remediation-only listener, so enabling Hunter alone can never open this path no
// matter what allowedDomains or extraSquidConf contains.
func TestAgentEgressProxyAlwaysAllowsSonarQubeAgenticEndpoints(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			chart := chart

			t.Run("present regardless of allowedDomains, scoped to the remediation listener", func(t *testing.T) {
				setValues := map[string]string{
					"remediationAgent.enabled":          "true",
					"remediationAgent.image.repository": "example.com/remediation-agent",
					"remediationAgent.image.tag":        "1",
					// allowedDomains left empty on purpose: the SonarQube allow rule must not depend on it.
				}
				output, err := renderAgentEgressProxyTemplates(t, chart, setValues, []string{"templates/agent-egress-proxy-configmap.yaml"})
				require.NoError(t, err)

				var cm corev1.ConfigMap
				helm.UnmarshalK8SYaml(t, output, &cm)
				conf := cm.Data["squid.conf"]

				// Fully qualified, not the bare short name - Squid's own DNS resolver doesn't honour
				// /etc/resolv.conf's search list, so a bare Service name can never resolve for it.
				assert.Contains(t, conf, "acl sonarqube_host dstdomain "+chart.fullnamePrefix()+".default.svc.cluster.local")
				assert.Contains(t, conf, "acl sonarqube_agentic_endpoints urlpath_regex ^[^?]*/rules/show(\\?|$) ^[^?]*/a3s/private/analyses(/|\\?|$)")
				assert.Contains(t, conf, "acl remediation_listener localport 3129")
				assert.Contains(t, conf, "http_access allow sonarqube_host sonarqube_agentic_endpoints remediation_listener")
				assert.Contains(t, conf, "acl Safe_ports port 9000", "SonarQube's default externalPort must be reachable too")
			})

			t.Run("absent when only Hunter is enabled", func(t *testing.T) {
				setValues := map[string]string{
					"hunterAgent.enabled":          "true",
					"hunterAgent.image.repository": "example.com/hunter-agent",
					"hunterAgent.image.tag":        "1",
				}
				conf := renderAgentEgressProxySquidConf(t, chart, setValues)

				assert.NotContains(t, conf, "http_access allow sonarqube_host sonarqube_agentic_endpoints", "Hunter must never reach SonarQube through the proxy (SONAR-32432)")
				assert.NotContains(t, conf, "acl remediation_listener", "the remediation-only listener ACL must not render when Remediation is disabled")
			})

			// Prefix tolerance is load-bearing twice over: the two endpoints sit on different API
			// versions (api/rules/show is v1, api/v2/a3s/private/analyses is v2), and the chart hands the
			// agents a web-context-aware base URL (AGENTIC_SONARQUBE_URL is built from
			// sonarqube.webcontext) so the path Squid sees also carries whatever sonarWebContext is
			// set to. Asserting on the rendered literal alone can't catch a regex that no longer
			// matches those paths, so compile it and run real request paths through it.
			t.Run("regex tolerates path prefixes without allowing query-string smuggling", func(t *testing.T) {
				for _, wc := range []struct {
					name       string
					webContext string
				}{
					{"default web context", ""},
					{"sonarWebContext=/sonarqube", "/sonarqube"},
				} {
					wc := wc
					t.Run(wc.name, func(t *testing.T) {
						assertAgenticEndpointRegexMatches(t, chart, wc.webContext)
					})
				}
			})

			t.Run("dstdomain tracks the fullname prefix and service.externalPort overrides", func(t *testing.T) {
				setValues := map[string]string{
					"remediationAgent.enabled":          "true",
					"remediationAgent.image.repository": "example.com/remediation-agent",
					"remediationAgent.image.tag":        "1",
					"service.externalPort":              "9001",
				}
				output, err := renderAgentEgressProxyTemplates(t, chart, setValues, []string{"templates/agent-egress-proxy-configmap.yaml"})
				require.NoError(t, err)

				var cm corev1.ConfigMap
				helm.UnmarshalK8SYaml(t, output, &cm)
				conf := cm.Data["squid.conf"]

				assert.Contains(t, conf, "acl sonarqube_host dstdomain "+chart.fullnamePrefix()+".default.svc.cluster.local")
				assert.Contains(t, conf, "acl Safe_ports port 9001")
			})
		})
	}
}

// Artifact-locator renewal (EA-968) is the one orchestrator call a runtime makes, and it makes it
// through the proxy like everything else: HTTP_PROXY is forced to Squid with NO_PROXY empty, so
// without an allow rule the renewal POST lands on `http_access deny all` and a job whose presigned
// URLs expired mid-run fails its upload. Hardcoded rather than sourced from allowedDomains, on the
// same reasoning as the SonarQube rule - an operator tightening allowedDomains must not be able to
// break renewal by omission.
//
// There is no "orchestrator disabled" counterpart to this test because that state is unreachable
// here: the proxy only renders when a runtime family is enabled, and validation.yaml fails a
// runtime family without agentOrchestrator.enabled. The template still gates the block, so the
// allowlist can never name a destination the release doesn't deploy if that ever changes.
func TestAgentEgressProxyAllowsOrchestratorArtifactLocatorRenewal(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			chart := chart
			setValues := map[string]string{
				"remediationAgent.enabled":          "true",
				"remediationAgent.image.repository": "example.com/remediation-agent",
				"remediationAgent.image.tag":        "1",
				// Left empty on purpose: renewal must not depend on allowedDomains.
			}

			t.Run("present regardless of allowedDomains", func(t *testing.T) {
				conf := renderAgentEgressProxySquidConf(t, chart, setValues)

				// Fully qualified for the same reason sonarqube_host is - Squid's resolver ignores
				// /etc/resolv.conf's search list - and it must be the exact host the runtime's
				// AGENT_ORCHESTRATOR_URL names, or the dstdomain ACL never matches.
				assert.Contains(t, conf, "acl agent_orchestrator_host dstdomain "+chart.fullnamePrefix()+"-agent-orchestrator.default.svc.cluster.local")
				assert.Contains(t, conf, "acl agent_orchestrator_endpoints urlpath_regex ^[^?]*/artifact-locators(\\?|$)")
				// Both ACLs on one rule: the host alone would make the whole orchestrator API
				// reachable from a runtime, which is the opposite of what this rule is for.
				assert.Contains(t, conf, "http_access allow agent_orchestrator_host agent_orchestrator_endpoints")
				// Load-bearing: `http_access deny !Safe_ports` is evaluated before every allow
				// rule, and the orchestrator's port is neither 80, 443 nor SonarQube's.
				assert.Contains(t, conf, "acl Safe_ports port 8080", "the renewal POST is denied before any allow rule without this")
			})

			// The allow rule is only as good as its regex: assert on ACL behaviour, not on the
			// rendered literal, for the same reasons as the SonarQube endpoints ACL.
			t.Run("regex is scoped to the renewal endpoint", func(t *testing.T) {
				matches := urlpathRegexMatcher(t, renderAgentEgressProxySquidConf(t, chart, setValues), "agent_orchestrator_endpoints")

				for _, tc := range []agenticEndpointCase{
					{"/artifact-locators", true, "the bare renewal path"},
					{"/artifact-locators?jobId=abc", true, "the real call carries a query string"},
					{"/api/v2/artifact-locators?jobId=abc", true, "prefix-agnostic, so an API-version change needs no chart change"},
					{"/anything?x=/artifact-locators", false, "a query string must not smuggle the path past the scoping"},
					{"/artifact-locators-debug", false, "an unrelated path that merely shares a prefix"},
					{"/jobs/abc", false, "the rest of the orchestrator API stays unreachable from a runtime"},
				} {
					assert.Equal(t, tc.allow, matches(tc.path), "%s: %s", tc.path, tc.why)
				}
			})

			// A non-default agentOrchestrator.port has to reach both the Safe_ports entry and the
			// NetworkPolicy rule; drift between them fails closed but only at runtime.
			t.Run("Safe_ports tracks agentOrchestrator.port", func(t *testing.T) {
				withPort := map[string]string{"agentOrchestrator.port": "8181"}
				for k, v := range setValues {
					withPort[k] = v
				}
				conf := renderAgentEgressProxySquidConf(t, chart, withPort)
				assert.Contains(t, conf, "acl Safe_ports port 8181")
				assert.NotContains(t, conf, "acl Safe_ports port 8080")
			})
		})
	}
}

func TestAgentEgressProxyPodDisruptionBudget(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{
				"hunterAgent.enabled":                               "true",
				"hunterAgent.image.repository":                      "example.com/hunter-agent",
				"hunterAgent.image.tag":                             "1",
				"agentEgressProxy.podDisruptionBudget.minAvailable": "3",
			}
			output, err := renderAgentEgressProxyTemplates(t, chart, setValues, []string{"templates/agent-egress-proxy-poddisruptionbudget.yaml"})
			require.NoError(t, err)

			var pdb policyv1.PodDisruptionBudget
			helm.UnmarshalK8SYaml(t, output, &pdb)
			require.NotNil(t, pdb.Spec.MinAvailable)
			assert.Equal(t, "3", pdb.Spec.MinAvailable.String())
		})
	}
}

// findEgressRuleTo returns the first egress rule matching predicate, or nil if none match.
func findEgressRuleTo(rules []networkingv1.NetworkPolicyEgressRule, predicate func(networkingv1.NetworkPolicyEgressRule) bool) *networkingv1.NetworkPolicyEgressRule {
	for i := range rules {
		if predicate(rules[i]) {
			return &rules[i]
		}
	}
	return nil
}

// findIngressRuleFrom returns the first ingress rule matching predicate, or nil if none match.
func findIngressRuleFrom(rules []networkingv1.NetworkPolicyIngressRule, predicate func(networkingv1.NetworkPolicyIngressRule) bool) *networkingv1.NetworkPolicyIngressRule {
	for i := range rules {
		if predicate(rules[i]) {
			return &rules[i]
		}
	}
	return nil
}

func isRuntimeFamilyPodRule(chart agentChart, family string) func(networkingv1.NetworkPolicyIngressRule) bool {
	return func(rule networkingv1.NetworkPolicyIngressRule) bool {
		return len(rule.From) == 1 && rule.From[0].PodSelector != nil &&
			rule.From[0].PodSelector.MatchLabels["app"] == chart.name+"-agent-runtime-"+family
	}
}

func isBroadIPBlockRule(rule networkingv1.NetworkPolicyEgressRule) bool {
	return len(rule.To) == 1 && rule.To[0].IPBlock != nil
}

func isSonarQubePodRule(chart agentChart) func(networkingv1.NetworkPolicyEgressRule) bool {
	return func(rule networkingv1.NetworkPolicyEgressRule) bool {
		return len(rule.To) == 1 && rule.To[0].PodSelector != nil && rule.To[0].PodSelector.MatchLabels["app"] == chart.name
	}
}

func isOrchestratorPodRule(chart agentChart) func(networkingv1.NetworkPolicyEgressRule) bool {
	return func(rule networkingv1.NetworkPolicyEgressRule) bool {
		return len(rule.To) == 1 && rule.To[0].PodSelector != nil && rule.To[0].PodSelector.MatchLabels["app"] == chart.name+"-agent-orchestrator"
	}
}

// The proxy's own NetworkPolicy defaults to enabled (SONAR-32432: egress must be structurally
// enforced, not opt-in), independent of D8's auto-activation of the Deployment/Service/ConfigMap.
func TestAgentEgressProxyNetworkPolicy(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			chart := chart
			t.Run("enabled by default when the proxy is active", func(t *testing.T) {
				testAgentEgressProxyNetworkPolicyEnabledByDefault(t, chart)
			})
			t.Run("enabled splits ingress per runtime family and allows 0.0.0.0/0 on egress", func(t *testing.T) {
				testAgentEgressProxyNetworkPolicyEnabled(t, chart)
			})
			t.Run("SonarQube pod egress rule is unconditional, not gated behind networkPolicy.egressPorts", func(t *testing.T) {
				testAgentEgressProxyNetworkPolicySonarQubePortUnconditional(t, chart)
			})
			t.Run("istio.enabled adds the sidecar's probe/scrape ports on ingress", func(t *testing.T) {
				testAgentEgressProxyNetworkPolicyIstioSidecarPorts(t, chart)
			})
		})
	}
}

func testAgentEgressProxyNetworkPolicyEnabledByDefault(t *testing.T, chart agentChart) {
	setValues := map[string]string{
		"hunterAgent.enabled":          "true",
		"hunterAgent.image.repository": "example.com/hunter-agent",
		"hunterAgent.image.tag":        "1",
	}
	output, err := renderAgentEgressProxyTemplates(t, chart, setValues, []string{"templates/agent-egress-proxy-networkpolicy.yaml"})
	require.NoError(t, err)

	var policy networkingv1.NetworkPolicy
	helm.UnmarshalK8SYaml(t, output, &policy)

	hunterIngress := findIngressRuleFrom(policy.Spec.Ingress, isRuntimeFamilyPodRule(chart, "hunter"))
	require.NotNil(t, hunterIngress, "expected an ingress rule selecting Hunter's own pods")
	require.Len(t, hunterIngress.Ports, 1)
	assert.EqualValues(t, 3128, hunterIngress.Ports[0].Port.IntVal)
}

func testAgentEgressProxyNetworkPolicyEnabled(t *testing.T, chart agentChart) {
	setValues := map[string]string{
		"hunterAgent.enabled":                    "true",
		"hunterAgent.image.repository":           "example.com/hunter-agent",
		"hunterAgent.image.tag":                  "1",
		"remediationAgent.enabled":               "true",
		"remediationAgent.image.repository":      "example.com/remediation-agent",
		"remediationAgent.image.tag":             "1",
		"agentEgressProxy.networkPolicy.enabled": "true",
	}
	output, err := renderAgentEgressProxyTemplates(t, chart, setValues, []string{"templates/agent-egress-proxy-networkpolicy.yaml"})
	require.NoError(t, err)

	var policy networkingv1.NetworkPolicy
	helm.UnmarshalK8SYaml(t, output, &policy)

	// Split in two so Hunter can only ever dial agentEgressProxy.port and Remediation only
	// agentEgressProxy.remediationPort (SONAR-32432) - a single merged rule selecting both
	// families on either port would let Hunter reach the remediation-only Squid listener and,
	// through it, the SonarQube allow rule that listener backs.
	require.Len(t, policy.Spec.Ingress, 2)
	hunterIngress := findIngressRuleFrom(policy.Spec.Ingress, isRuntimeFamilyPodRule(chart, "hunter"))
	remediationIngress := findIngressRuleFrom(policy.Spec.Ingress, isRuntimeFamilyPodRule(chart, "remediation"))

	require.NotNil(t, hunterIngress, "expected an ingress rule selecting Hunter's own pods")
	require.Len(t, hunterIngress.Ports, 1)
	assert.EqualValues(t, 3128, hunterIngress.Ports[0].Port.IntVal)

	require.NotNil(t, remediationIngress, "expected an ingress rule selecting Remediation's own pods")
	require.Len(t, remediationIngress.Ports, 1)
	assert.EqualValues(t, 3129, remediationIngress.Ports[0].Port.IntVal)

	require.Len(t, policy.Spec.Egress, 4, "DNS, the SonarQube pod rule, the orchestrator pod rule, plus the broad 0.0.0.0/0 rule")
	broad := findEgressRuleTo(policy.Spec.Egress, isBroadIPBlockRule)
	sonarqube := findEgressRuleTo(policy.Spec.Egress, isSonarQubePodRule(chart))
	orchestrator := findEgressRuleTo(policy.Spec.Egress, isOrchestratorPodRule(chart))

	require.NotNil(t, broad, "expected a broad ipBlock egress rule")
	assert.Equal(t, "0.0.0.0/0", broad.To[0].IPBlock.CIDR)
	require.Len(t, broad.Ports, 2)
	ports := []int32{broad.Ports[0].Port.IntVal, broad.Ports[1].Port.IntVal}
	assert.ElementsMatch(t, []int32{80, 443}, ports)

	require.NotNil(t, sonarqube, "expected a rule allowing egress to the SonarQube pod - "+
		"this backs the hardcoded sonarqube_host allow rule in agent-egress-proxy-configmap.yaml")
	require.Len(t, sonarqube.Ports, 1)
	assert.EqualValues(t, 9000, sonarqube.Ports[0].Port.IntVal)

	// Backs the hardcoded agent_orchestrator_host allow rule, which is the path a runtime's
	// artifact-locator renewal takes (EA-968). Its own rule rather than an entry in
	// networkPolicy.egressPorts: that list applies to the broad rule above, so putting the
	// orchestrator's port there would open it to the whole internet to reach one in-cluster
	// Service.
	require.NotNil(t, orchestrator, "expected a rule allowing egress to the orchestrator pod - "+
		"this backs the hardcoded agent_orchestrator_host allow rule in agent-egress-proxy-configmap.yaml")
	require.Len(t, orchestrator.Ports, 1)
	assert.EqualValues(t, 8080, orchestrator.Ports[0].Port.IntVal)
	assert.NotContains(t, ports, int32(8080), "the orchestrator's port must not be reachable on the broad 0.0.0.0/0 rule")
}

func testAgentEgressProxyNetworkPolicySonarQubePortUnconditional(t *testing.T, chart agentChart) {
	setValues := map[string]string{
		"hunterAgent.enabled":          "true",
		"hunterAgent.image.repository": "example.com/hunter-agent",
		"hunterAgent.image.tag":        "1",
		// networkPolicy is enabled so the resource renders; egressPorts and service.externalPort
		// are pinned to non-default values to prove the SonarQube rule's port comes from
		// service.internalPort and is not gated behind networkPolicy.egressPorts.
		"agentEgressProxy.networkPolicy.enabled":        "true",
		"agentEgressProxy.networkPolicy.egressPorts[0]": "8080",
		"service.internalPort":                          "9001",
		"service.externalPort":                          "9002",
		"agentOrchestrator.port":                        "8181",
	}
	output, err := renderAgentEgressProxyTemplates(t, chart, setValues, []string{"templates/agent-egress-proxy-networkpolicy.yaml"})
	require.NoError(t, err)

	var policy networkingv1.NetworkPolicy
	helm.UnmarshalK8SYaml(t, output, &policy)

	sonarqube := findEgressRuleTo(policy.Spec.Egress, isSonarQubePodRule(chart))
	require.NotNil(t, sonarqube)
	require.Len(t, sonarqube.Ports, 1)
	assert.EqualValues(t, 9001, sonarqube.Ports[0].Port.IntVal, "tracks service.internalPort, not networkPolicy.egressPorts or service.externalPort")

	// Same for the orchestrator rule: its port comes from agentOrchestrator.port, which must be
	// the same value the Squid ConfigMap puts in Safe_ports - a mismatch drops the renewal POST at
	// the NetworkPolicy, before Squid ever logs it.
	orchestrator := findEgressRuleTo(policy.Spec.Egress, isOrchestratorPodRule(chart))
	require.NotNil(t, orchestrator)
	require.Len(t, orchestrator.Ports, 1)
	assert.EqualValues(t, 8181, orchestrator.Ports[0].Port.IntVal, "tracks agentOrchestrator.port")
}

// The proxy is always injected when istio.enabled (never excluded by gVisor, unlike the
// runtimes) - its own NetworkPolicy must always admit the sidecar's status ports on ingress.
func testAgentEgressProxyNetworkPolicyIstioSidecarPorts(t *testing.T, chart agentChart) {
	setValues := map[string]string{
		"hunterAgent.enabled":                    "true",
		"hunterAgent.image.repository":           "example.com/hunter-agent",
		"hunterAgent.image.tag":                  "1",
		"agentEgressProxy.networkPolicy.enabled": "true",
		"istio.enabled":                          "true",
	}
	if chart.name == "sonarqube-dce" {
		// istio.enabled requires the Hazelcast web/CE channels to be pinned (validation.yaml);
		// unrelated to this policy, but the render fails without them.
		setValues["applicationNodes.webPort"] = "4023"
		setValues["applicationNodes.cePort"] = "4024"
	}
	output, err := renderAgentEgressProxyTemplates(t, chart, setValues, []string{"templates/agent-egress-proxy-networkpolicy.yaml"})
	require.NoError(t, err)

	var policy networkingv1.NetworkPolicy
	helm.UnmarshalK8SYaml(t, output, &policy)

	ingressPorts := networkPolicyIngressPorts(policy.Spec.Ingress)
	assert.Contains(t, ingressPorts, int32(15020))
	assert.Contains(t, ingressPorts, int32(15021))
	assert.Contains(t, ingressPorts, int32(15090))
}

// Regression test: HTTP_PROXY/HTTPS_PROXY/NO_PROXY (and lowercase variants) must not be
// overridable via hunterAgent.env/remediationAgent.env - D1/D2 require the runtime to have no
// way to bypass the proxy or carve out a NO_PROXY exception.
func TestAgentEgressProxyEnvVarsNotOverridable(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/agent-runtimes-enabled.yaml"},
				SetValues: map[string]string{
					"hunterAgent.env[0].name":  "NO_PROXY",
					"hunterAgent.env[0].value": "attacker.example.com",
					"hunterAgent.env[1].name":  "HTTP_PROXY",
					"hunterAgent.env[1].value": "http://bypass.example.com:8080",
				},
			}
			output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-runtime.yaml"})
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

			assert.Equal(t, chart.agentProxyURL(), lastValueByName["HTTP_PROXY"], "attacker-supplied env must not win")
			assert.Equal(t, "", lastValueByName["NO_PROXY"], "NO_PROXY must stay forced empty")
		})
	}
}

// reservedServiceHostVar renders the proxy's own Service and derives the kubelet-published
// service-link variable name a runtime must never be able to set - see
// TestAgentEgressProxyServiceHostVarRejectedInRuntimeEnv.
func reservedServiceHostVar(t *testing.T, chart agentChart) string {
	t.Helper()
	svcOut, err := renderAgentEgressProxyTemplates(t, chart, map[string]string{
		"hunterAgent.enabled":          "true",
		"hunterAgent.image.repository": "example.com/hunter-agent",
		"hunterAgent.image.tag":        "1",
	}, []string{"templates/agent-egress-proxy-service.yaml"})
	require.NoError(t, err)
	var service corev1.Service
	helm.UnmarshalK8SYaml(t, svcOut, &service)
	require.NotEmpty(t, service.Name)
	return strings.ToUpper(strings.ReplaceAll(service.Name, "-", "_")) + "_SERVICE_HOST"
}

// TestAgentEgressProxyEnvVarsNotOverridable only proves the last-write-wins merge can't be used to
// smuggle a bypass through a literal HTTP_PROXY/NO_PROXY name. It can't prove anything about the
// reserved kubelet service-link variable HTTP_PROXY itself expands to ($(<SVC>_SERVICE_HOST)):
// kubelet's tmpEnv map poisons that key on first sight, so no later entry - regardless of
// declaration order - can ever override it back once a runtime's own env sets it. Ordering-based
// defenses are structurally unable to close that, which is why templates/agent-runtime.yaml fails
// the render outright instead.
func TestAgentEgressProxyServiceHostVarRejectedInRuntimeEnv(t *testing.T) {
	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			reservedVar := reservedServiceHostVar(t, chart)

			opts := &helm.Options{
				Logger:      logger.Discard,
				ValuesFiles: []string{chart.valuesDir + "/agent-runtimes-enabled.yaml"},
				SetValues: map[string]string{
					"hunterAgent.env[0].name":  reservedVar,
					"hunterAgent.env[0].value": "attacker.example.com:9999",
				},
			}
			_, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/agent-runtime.yaml"})
			require.Error(t, err, "setting the egress proxy's own kubelet-published service-link variable must fail the render, not silently merge")
			assert.Contains(t, err.Error(), reservedVar)
			assert.Contains(t, err.Error(), "kubelet-published address of the Agent Egress Proxy")
		})
	}
}

// The service-link variable a runtime expands is derived in-template from the proxy's Service
// name, and the two must never drift: a wrong variable name expands to nothing, leaving
// HTTP_PROXY pointing at a port on no host, and since the runtime has no resolver and no other
// egress rule the failure is total rather than degraded. Release names are where drift would
// show up - `sonarqube.agentEgressProxy.fullname` truncates to 63 characters, and a release name
// may contain dashes or start with a digit - so this renders the Service and the runtime
// together for each shape and compares one against the other rather than against a literal.
func TestAgentEgressProxyServiceLinkVariableTracksServiceName(t *testing.T) {
	releases := []struct {
		name    string
		release string
	}{
		{"dashes", "sq-test-release"},
		{"leading digit", "1sq"},
		// 53 characters is Helm's own ceiling on a release name, and it is already long
		// enough that appending the chart name and "-agent-egress-proxy" overruns 63.
		{"long enough to truncate the 63-char fullname", "sq-" + strings.Repeat("a", 50)},
	}

	for _, chart := range egressProxyCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, r := range releases {
				t.Run(r.name, func(t *testing.T) {
					assertServiceLinkVariableTracksServiceName(t, chart, r.release)
				})
			}
		})
	}
}

// assertServiceLinkVariableTracksServiceName renders the proxy's Service and the runtime
// Deployments for one chart/release combination and checks that HTTP_PROXY et al. point at the
// service-link variable derived from the Service's actual (possibly truncated) name.
func assertServiceLinkVariableTracksServiceName(t *testing.T, chart agentChart, release string) {
	setValues := map[string]string{
		"hunterAgent.enabled":               "true",
		"hunterAgent.image.repository":      "example.com/hunter-agent",
		"hunterAgent.image.tag":             "1",
		"remediationAgent.enabled":          "true",
		"remediationAgent.image.repository": "example.com/remediation-agent",
		"remediationAgent.image.tag":        "1",
	}

	svcOut, err := renderAgentEgressProxyTemplatesAs(t, chart, release, setValues,
		[]string{"templates/agent-egress-proxy-service.yaml"})
	require.NoError(t, err)
	var service corev1.Service
	helm.UnmarshalK8SYaml(t, svcOut, &service)
	require.NotEmpty(t, service.Name)
	require.LessOrEqual(t, len(service.Name), 63)

	hostVar := strings.ToUpper(strings.ReplaceAll(service.Name, "-", "_")) + "_SERVICE_HOST"

	runtimeOut, err := renderAgentEgressProxyTemplatesAs(t, chart, release, setValues,
		[]string{"templates/agent-runtime.yaml"})
	require.NoError(t, err)

	var sawRuntime bool
	for _, doc := range strings.Split(runtimeOut, "\n---") {
		if !strings.Contains(doc, "kind: Deployment") {
			continue
		}
		var deployment appsv1.Deployment
		helm.UnmarshalK8SYaml(t, doc, &deployment)
		sawRuntime = true
		// Family-scoped: remediation dials the proxy's second listener (3129), the only one
		// whose Squid ACL admits SonarQube's own agentic endpoints; hunter dials 3128.
		port := "3128"
		if deployment.Labels["sonarqube.agent/family"] == "remediation" {
			port = "3129"
		}
		want := "http://$(" + hostVar + "):" + port
		assertRuntimeProxyEnv(t, deployment, want)
	}
	require.True(t, sawRuntime, "expected at least one runtime Deployment")
}

func assertRuntimeProxyEnv(t *testing.T, deployment appsv1.Deployment, want string) {
	env := map[string]string{}
	for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
		env[e.Name] = e.Value
	}
	for _, key := range []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy"} {
		assert.Equal(t, want, env[key], "%s on %s", key, deployment.Name)
	}
}
