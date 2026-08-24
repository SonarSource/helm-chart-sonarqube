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
}

// fullnamePrefix is "<release>-<chart>", e.g. "sonarqube-dce-sonarqube-dce" or
// "sonarqube-sonarqube" - the prefix `sonarqube.fullname` produces for every agentic resource name.
func (c agentChart) fullnamePrefix() string {
	return c.release + "-" + c.name
}

var agentCharts = []agentChart{
	{"sonarqube-dce", dceChartPath, dceReleaseName, "test-cases-values/sonarqube-dce", "templates/sonarqube-application.yaml"},
	{"sonarqube", chartPath, releaseName, "test-cases-values/sonarqube", "templates/sonarqube-sts.yaml"},
}
