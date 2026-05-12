package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

type ChartTestSpec struct {
	Name              string
	Values            map[string]string
	ValuesFiles       []string
	RequireExternalDB bool
}

// This function checks the dynamic compatibility test for the sonarqube standard and DCE chart
// Specifically, the tests confirm that:
// - the resources part of the Helm chart are ready
// - the sonarqube application is ready (apis are ready to serve new requests)
func RunChartTest(t *testing.T, chart ChartTestSpec) {
	namespaceName := NamespaceFor(chart.Name)
	helmChartPath := "../../../charts/" + chart.Name
	existingKubectlOptions := k8s.NewKubectlOptions("", "", "default")
	k8s.CreateNamespace(t, existingKubectlOptions, namespaceName)
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	defer TearDownTestSetup(t, kubectlOptions, namespaceName)

	if chart.RequireExternalDB {
		SetupExternalDB(t, kubectlOptions)
	}

	helmOptions := &helm.Options{
		ValuesFiles:    chart.ValuesFiles,
		SetValues:      chart.Values,
		KubectlOptions: kubectlOptions,
	}
	output := helm.RenderTemplate(t, helmOptions, helmChartPath, chart.Name, []string{})

	// ******* WHEN *********

	k8s.KubectlApplyFromString(t, kubectlOptions, output)

	// ******* THEN *********

	WaitForChartReady(t, kubectlOptions, chart.Name)
	CheckSonarQubeUpAndRunning(t, kubectlOptions, chart.Name)
}
