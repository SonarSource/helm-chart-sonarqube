package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// Port 9002 ("es") is Elasticsearch's raw binary transport, not HTTP - an unmarked Service port
// falls back to Istio's protocol sniffing (discouraged, adds latency), so pin it to tcp like the
// "hazelcast" ports. search-port (9001) stays unmarked: it's HTTP or HTTPS depending on
// nodeEncryption.enabled, so it can't be pinned to one protocol.
func TestSearchServiceEsPortIsRawTCP(t *testing.T) {
	opts := &helm.Options{Logger: logger.Discard, SetValues: dceEsMajorBaseValues()}
	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/service.yaml"})
	require.NoError(t, err)

	services := map[string]corev1.Service{}
	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" || !strings.Contains(doc, "kind: Service") {
			continue
		}
		var service corev1.Service
		helm.UnmarshalK8SYaml(t, doc, &service)
		switch {
		case strings.HasSuffix(service.Name, "-search-headless"):
			services["headless"] = service
		case strings.HasSuffix(service.Name, "-search"):
			services["clusterip"] = service
		}
	}
	require.Len(t, services, 2, "expected both the search and search-headless Services to render")

	for kind, service := range services {
		t.Run(kind, func(t *testing.T) {
			for _, port := range service.Spec.Ports {
				if port.Name != "es" {
					continue
				}
				require.NotNil(t, port.AppProtocol, "es port must set appProtocol so Istio does not fall back to protocol sniffing")
				assert.Equal(t, "tcp", *port.AppProtocol)
				return
			}
			t.Fatalf("service %q has no port named %q", service.Name, "es")
		})
	}
}
