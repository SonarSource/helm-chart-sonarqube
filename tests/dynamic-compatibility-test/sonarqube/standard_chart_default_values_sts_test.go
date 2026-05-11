package sonarqube

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

// This test checks the dynamic compatibility test for the sonarqube standard
// Specifically, we perform tests that:
// - check if the required (default) number of pods is running
// - check if all the running pods have the containersReady condition set to true
// - check if the sonarqube application is ready (apis are ready to serve new requests)
func TestStatefulSetWithDefaultValues(t *testing.T) {
	t.Parallel()
	t.Run("Standard chart with default values as StatefulSet", func(t *testing.T) {

		// ******* GIVEN *********
		chartName := "sonarqube"
		values := map[string]string{
			utils.TESTS_ENABLING_ACTION: "false",
			"edition":                   "enterprise",
			"monitoringPasscode":        "monitoringPasscode",
		}
		valuesFilesPaths := []string{"../../../charts/sonarqube/values.yaml"}
		namespaceName := utils.NamespaceFor(chartName)
		helmChartPath := "../../../charts/" + chartName
		existingKubectlOptions := k8s.NewKubectlOptions("", "", "default")
		k8s.CreateNamespace(t, existingKubectlOptions, namespaceName)
		kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
		defer utils.TearDownTestSetup(t, kubectlOptions, namespaceName)

		helmOptions := &helm.Options{
			ValuesFiles:    valuesFilesPaths,
			SetValues:      values,
			KubectlOptions: kubectlOptions,
		}

		output := helm.RenderTemplate(t, helmOptions, helmChartPath, chartName, []string{})

		// ******* WHEN *********

		k8s.KubectlApplyFromString(t, kubectlOptions, output)

		// ******* THEN *********

		expectedPods := 1
		pods := utils.CheckMinimumNumberOfRunningPods(t, kubectlOptions, expectedPods)
		utils.CheckPodsInContainersReadyState(t, kubectlOptions, pods)
		utils.CheckSonarQubeUpAndRunning(t, kubectlOptions, chartName)
	})
}
