package tests

import corev1 "k8s.io/api/core/v1"

// agentVolumeMountsByName / agentVolumesByName index a pod's mounts and volumes by name, so a test
// can assert on the one entry it is about without also pinning every unrelated entry another
// feature legitimately adds to the same pod.
func agentVolumeMountsByName(mounts []corev1.VolumeMount) map[string]corev1.VolumeMount {
	byName := map[string]corev1.VolumeMount{}
	for _, m := range mounts {
		byName[m.Name] = m
	}
	return byName
}

func agentVolumesByName(volumes []corev1.Volume) map[string]corev1.Volume {
	byName := map[string]corev1.Volume{}
	for _, v := range volumes {
		byName[v.Name] = v
	}
	return byName
}

// agentChart describes one chart the agentic pack test suite runs against.
type agentChart struct {
	name      string
	path      string
	release   string
	valuesDir string
	// appTemplate is the path of the template rendering the SonarQube application pod - the
	// two charts name it differently.
	appTemplate string
	// appConfigMapSuffix is appended to fullnamePrefix() to get the name of the ConfigMap holding
	// the application nodes' conf/sonar.properties. templates/config.yaml renders a single
	// ConfigMap in sonarqube, but an app one plus a search one in sonarqube-dce.
	appConfigMapSuffix string
	// sonarPropertiesPath is the values path holding the user's own conf/sonar.properties entries -
	// top-level in sonarqube, under the application nodes in sonarqube-dce.
	sonarPropertiesPath string
	// hasEgressProxy is true for charts that route agent runtime egress through a dedicated Agent
	// Egress Proxy component instead of the legacy networkPolicy.egressAllow escape hatch.
	hasEgressProxy bool
	// hasAgenticKeys is true for charts that derive per-hop agentic signing keys from
	// agenticSigningSecret and mount them on the consuming pods (SONAR-32065). Both charts have
	// the feature; the flag stays so a chart that ever drops it can opt out of the whole file.
	hasAgenticKeys bool
}

// fullnamePrefix is "<release>-<chart>", e.g. "sonarqube-dce-sonarqube-dce" or
// "sonarqube-sonarqube" - the prefix `sonarqube.fullname` produces for every agentic resource name.
func (c agentChart) fullnamePrefix() string {
	return c.release + "-" + c.name
}

// orchestratorURL is the expected value of the sonar.*.orchestrator.url properties once the
// orchestrator is enabled. Charts with an Agent Egress Proxy (hasEgressProxy) route runtime
// traffic through Squid, whose own DNS resolver ignores /etc/resolv.conf's search list and so can
// never resolve a bare Service name - those charts render a fully qualified in-cluster hostname
// instead of the short one the other chart still uses. None of these tests set an explicit
// namespace, so Helm's "default" applies.
func (c agentChart) orchestratorURL() string {
	host := c.fullnamePrefix() + "-agent-orchestrator"
	if c.hasEgressProxy {
		host += ".default.svc.cluster.local"
	}
	return "http://" + host + ":8080"
}

var agentCharts = []agentChart{
	{"sonarqube-dce", dceChartPath, dceReleaseName, "test-cases-values/sonarqube-dce", "templates/sonarqube-application.yaml", "-app-config", "applicationNodes.sonarProperties", true, true},
	{"sonarqube", chartPath, releaseName, "test-cases-values/sonarqube", "templates/sonarqube-sts.yaml", "-config", "sonarProperties", true, true},
}
