package tests

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"

	"github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test/dependencies"
)

type ChartTestSpec struct {
	Name              string
	Values            map[string]string
	ValuesFiles       []string
	RequireExternalDB bool
}

// chartPath returns the absolute path of <repo>/charts/<name>. Resolved via
// runtime.Caller so the result is independent of the test process's working
// directory — each test package runs from its own subdir, so a fixed relative
// path would break the moment the tree is restructured.
func chartPath(name string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "charts", name)
}

// This function checks the dynamic compatibility test for the sonarqube standard and DCE chart
// Specifically, the tests confirm that:
// - the resources part of the Helm chart are ready
// - the sonarqube application is ready (apis are ready to serve new requests)
func RunChartTest(t *testing.T, chart ChartTestSpec) {
	// ******* GIVEN *********

	namespaceName := NamespaceFor(chart.Name)
	helmChartPath := chartPath(chart.Name)
	existingKubectlOptions := k8s.NewKubectlOptions("", "", "default")
	k8s.CreateNamespace(t, existingKubectlOptions, namespaceName)
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	defer DeleteNamespaceAndWait(t, kubectlOptions, namespaceName)

	if chart.RequireExternalDB {
		dependencies.SetupDB(t, kubectlOptions)
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
