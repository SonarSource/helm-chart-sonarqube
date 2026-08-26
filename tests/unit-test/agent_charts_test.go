package tests

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
}

// fullnamePrefix is "<release>-<chart>", e.g. "sonarqube-dce-sonarqube-dce" or
// "sonarqube-sonarqube" - the prefix `sonarqube.fullname` produces for every agentic resource name.
func (c agentChart) fullnamePrefix() string {
	return c.release + "-" + c.name
}

var agentCharts = []agentChart{
	{"sonarqube-dce", dceChartPath, dceReleaseName, "test-cases-values/sonarqube-dce", "templates/sonarqube-application.yaml", "-app-config", "applicationNodes.sonarProperties"},
	{"sonarqube", chartPath, releaseName, "test-cases-values/sonarqube", "templates/sonarqube-sts.yaml", "-config", "sonarProperties"},
}
