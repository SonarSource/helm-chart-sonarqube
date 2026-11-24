package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

const (
	ingressNginxHttp2Warning   = "HTTP/2 security vulnerabilities"
	ingressNginxDeprecation    = "The ingress-nginx controller dependency is deprecated."
	ingressNginxOldUpgradeNote = "upgraded to a major version (4.x)"
)

// Values each chart needs to render, on top of the case under test.
var ingressNginxChartArgs = map[string][]string{
	"sonarqube": {
		"--set", "monitoringPasscode=test",
		"--set", "edition=enterprise",
	},
	"sonarqube-dce": {
		"--set", "monitoringPasscode=test",
		"--set", "applicationNodes.jwtSecret=dGVzdA==",
		"--set", "jdbcOverwrite.jdbcUrl=jdbc:postgresql://db/sonar",
		"--set", "jdbcOverwrite.jdbcUsername=sonar",
		"--set", "jdbcOverwrite.jdbcPassword=sonar",
	},
}

type ingressNginxToggle struct {
	name string
	args []string
}

type ingressNginxHttp2Case struct {
	name      string
	args      []string
	configMap string
	http2On   bool
}

// How the controller is turned on: not at all, the current key, the deprecated "nginx" key, or both.
var ingressNginxToggles = []ingressNginxToggle{
	{"disabled", nil},
	{"ingress-nginx", []string{"--set", "ingress-nginx.enabled=true"}},
	{"nginx", []string{"--set", "nginx.enabled=true"}},
	{"both", []string{"--set", "nginx.enabled=true", "--set", "ingress-nginx.enabled=true"}},
}

// How use-http2 is set, and what the controller ConfigMap must then hold ("" = key not rendered).
var ingressNginxHttp2Cases = []ingressNginxHttp2Case{
	{"default", nil, "false", false},
	{"string-true", []string{"--set-string", "ingress-nginx.controller.config.use-http2=true"}, "true", true},
	{"bool-true", []string{"--set", "ingress-nginx.controller.config.use-http2=true"}, "true", true},
	{"string-false", []string{"--set-string", "ingress-nginx.controller.config.use-http2=false"}, "false", false},
	{"bool-false", []string{"--set", "ingress-nginx.controller.config.use-http2=false"}, "false", false},
	{"config-removed", []string{"--set", "ingress-nginx.controller.config=null"}, "", true},
}

// TestIngressNginxHttp2 checks the use-http2 default of the bundled controller and the matching install notes.
func TestIngressNginxHttp2(t *testing.T) {
	t.Parallel()
	for _, chart := range []string{"sonarqube", "sonarqube-dce"} {
		for _, toggle := range ingressNginxToggles {
			for _, http2 := range ingressNginxHttp2Cases {
				chart, toggle, http2 := chart, toggle, http2
				t.Run(fmt.Sprintf("%s/%s/%s", chart, toggle.name, http2.name), func(t *testing.T) {
					t.Parallel()
					testIngressNginxCase(t, chart, toggle, http2)
				})
			}
		}
	}
}

// testIngressNginxCase renders one chart with one toggle and one use-http2 setting, then checks the ConfigMap and the notes.
func testIngressNginxCase(t *testing.T, chart string, toggle ingressNginxToggle, http2 ingressNginxHttp2Case) {
	args := []string{"ingress-nginx-test", "../../charts/" + chart, "--dry-run=client", "--namespace", "default"}
	args = append(args, ingressNginxChartArgs[chart]...)
	args = append(args, toggle.args...)
	args = append(args, http2.args...)
	output, err := helm.RunHelmCommandAndGetStdOutE(t, &helm.Options{Logger: logger.Discard}, "install", args...)
	require.NoError(t, err)

	manifest, notes, found := strings.Cut(output, "\nNOTES:\n")
	require.True(t, found, "no NOTES in the install output")
	configMap, rendered := ingressNginxControllerConfigMap(t, chart, manifest)

	assert.NotContains(t, notes, ingressNginxOldUpgradeNote)
	if toggle.name == "disabled" {
		assert.False(t, rendered, "controller ConfigMap rendered while the controller is disabled")
		assert.NotContains(t, notes, ingressNginxHttp2Warning)
		assert.NotContains(t, notes, ingressNginxDeprecation)
		return
	}
	require.True(t, rendered, "controller ConfigMap not rendered")
	assert.Equal(t, http2.configMap, configMap.Data["use-http2"])
	if http2.http2On {
		assert.Contains(t, notes, ingressNginxHttp2Warning)
		assert.NotContains(t, notes, ingressNginxDeprecation)
	} else {
		assert.Contains(t, notes, ingressNginxDeprecation)
		assert.NotContains(t, notes, ingressNginxHttp2Warning)
	}
}

// ingressNginxControllerConfigMap returns the controller ConfigMap from a rendered manifest, and whether it was there.
func ingressNginxControllerConfigMap(t *testing.T, chart, manifest string) (v1.ConfigMap, bool) {
	var configMap v1.ConfigMap
	source := fmt.Sprintf("# Source: %s/charts/ingress-nginx/templates/controller-configmap.yaml", chart)
	for _, doc := range strings.Split(manifest, "\n---\n") {
		if strings.Contains(doc, source) {
			helm.UnmarshalK8SYaml(t, doc, &configMap)
			return configMap, true
		}
	}
	return configMap, false
}
