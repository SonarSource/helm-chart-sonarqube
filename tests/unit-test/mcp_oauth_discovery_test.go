package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
)

// With a web context, RFC 8414 puts the authorization server metadata of the MCP OAuth issuer at the
// host root, so the default Ingress and HTTPRoute route that path too, whether or not the chart runs MCP.

const mcpOAuthDiscoveryPath = "/.well-known/oauth-authorization-server/sonarqube"

func renderRoute(t *testing.T, chart agentChart, template string, setValues map[string]string) string {
	t.Helper()
	opts := &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: []string{chart.valuesDir + "/mcp-enabled.yaml"},
		SetValues:   setValues,
	}
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{template})
	require.NoError(t, err)
	return output
}

func ingressPaths(t *testing.T, chart agentChart, setValues map[string]string) []networkingv1.HTTPIngressPath {
	t.Helper()
	values := map[string]string{"ingress.enabled": "true", "ingress.hosts[0].name": "sonarqube.example.com"}
	for k, v := range setValues {
		values[k] = v
	}
	var ingress networkingv1.Ingress
	helm.UnmarshalK8SYaml(t, renderRoute(t, chart, "templates/ingress.yaml", values), &ingress)
	require.Len(t, ingress.Spec.Rules, 1)
	return ingress.Spec.Rules[0].HTTP.Paths
}

type httpRouteSpec struct {
	Spec struct {
		Rules []struct {
			Matches []struct {
				Path struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"path"`
			} `json:"matches"`
		} `json:"rules"`
	} `json:"spec"`
}

func httpRouteMatches(t *testing.T, chart agentChart, setValues map[string]string) []string {
	t.Helper()
	values := map[string]string{"httproute.enabled": "true", "httproute.gateway": "my-gateway"}
	for k, v := range setValues {
		values[k] = v
	}
	var route httpRouteSpec
	helm.UnmarshalK8SYaml(t, renderRoute(t, chart, "templates/http-route.yaml", values), &route)
	var matches []string
	for _, rule := range route.Spec.Rules {
		for _, match := range rule.Matches {
			matches = append(matches, match.Path.Type+" "+match.Path.Value)
		}
	}
	return matches
}

func TestMcpOAuthDiscoveryIngress(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("web context adds the discovery path to the same backend", func(t *testing.T) {
				paths := ingressPaths(t, chart, map[string]string{"sonarWebContext": "/sonarqube/"})
				require.Len(t, paths, 2)
				assert.Equal(t, "/sonarqube/", paths[0].Path)
				assert.Equal(t, mcpOAuthDiscoveryPath, paths[1].Path)
				assert.Equal(t, networkingv1.PathTypeImplementationSpecific, *paths[1].PathType)
				assert.Equal(t, paths[0].Backend, paths[1].Backend)
			})

			t.Run("discovery path stays ImplementationSpecific with a Prefix host", func(t *testing.T) {
				paths := ingressPaths(t, chart, map[string]string{"sonarWebContext": "/sonarqube/", "ingress.hosts[0].pathType": "Prefix"})
				require.Len(t, paths, 2)
				assert.Equal(t, networkingv1.PathTypePrefix, *paths[0].PathType)
				assert.Equal(t, networkingv1.PathTypeImplementationSpecific, *paths[1].PathType)
			})

			t.Run("no web context adds no path", func(t *testing.T) {
				assert.Len(t, ingressPaths(t, chart, nil), 1)
			})

			for _, rootPath := range []string{"/", "/*"} {
				t.Run("host root path "+rootPath+" adds no path", func(t *testing.T) {
					assert.Len(t, ingressPaths(t, chart, map[string]string{"sonarWebContext": "/sonarqube/", "ingress.hosts[0].path": rootPath}), 1)
				})
			}

			for _, contextPath := range []string{"/sonarqube", "/sonarqube/*"} {
				t.Run("custom path "+contextPath+" adds the discovery path", func(t *testing.T) {
					paths := ingressPaths(t, chart, map[string]string{"sonarWebContext": "/sonarqube/", "ingress.hosts[0].path": contextPath})
					require.Len(t, paths, 2)
					assert.Equal(t, contextPath, paths[0].Path)
					assert.Equal(t, mcpOAuthDiscoveryPath, paths[1].Path)
				})
			}

			t.Run("mcp disabled still adds the discovery path", func(t *testing.T) {
				paths := ingressPaths(t, chart, map[string]string{"sonarWebContext": "/sonarqube/", "mcp.enabled": "false"})
				require.Len(t, paths, 2)
				assert.Equal(t, mcpOAuthDiscoveryPath, paths[1].Path)
			})
		})
	}
}

func TestMcpOAuthDiscoveryHTTPRoute(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("web context adds the discovery match to the default rule", func(t *testing.T) {
				assert.Equal(t, []string{"PathPrefix /sonarqube/", "PathPrefix " + mcpOAuthDiscoveryPath},
					httpRouteMatches(t, chart, map[string]string{"sonarWebContext": "/sonarqube/"}))
			})

			t.Run("no web context adds no match", func(t *testing.T) {
				assert.Equal(t, []string{"PathPrefix /"}, httpRouteMatches(t, chart, nil))
			})

			t.Run("custom rules are left as set", func(t *testing.T) {
				assert.Equal(t, []string{"PathPrefix /bar"}, httpRouteMatches(t, chart, map[string]string{
					"sonarWebContext":                          "/sonarqube/",
					"httproute.rules[0].matches[0].path.type":  "PathPrefix",
					"httproute.rules[0].matches[0].path.value": "/bar",
				}))
			})

			t.Run("mcp disabled still adds the discovery match", func(t *testing.T) {
				assert.Equal(t, []string{"PathPrefix /sonarqube/", "PathPrefix " + mcpOAuthDiscoveryPath},
					httpRouteMatches(t, chart, map[string]string{"sonarWebContext": "/sonarqube/", "mcp.enabled": "false"}))
			})
		})
	}
}
