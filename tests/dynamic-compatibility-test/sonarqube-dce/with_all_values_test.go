package sonarqube_dce

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

// This test checks the dynamic compatibility test for the sonarqube dce charts
// Specifically, we perform tests that:
// - check if the required (default) number of pods is running
// - check if all the running pods have the containersReady condition set to true
// - check if the sonarqube application is ready (apis are ready to serve new requests)
func TestWithAllValues(t *testing.T) {
	t.Parallel()
	t.Run("DCE chart with All Values", func(t *testing.T) {
		// ******* GIVEN *********

		chartName := "sonarqube-dce"
		values := map[string]string{
			utils.TESTS_ENABLING_ACTION:  "false",
			"ApplicationNodes.jwtSecret": "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=",
		}
		expectedPods := 1
		valuesFilesPaths := []string{"all-values.yaml"}
		namespaceName := utils.NamespaceFor(chartName)
		helmChartPath := "../../../charts/" + chartName
		existingKubectlOptions := k8s.NewKubectlOptions("", "", "default")
		k8s.CreateNamespace(t, existingKubectlOptions, namespaceName)
		kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
		defer utils.TearDownTestSetup(t, kubectlOptions, namespaceName)

		utils.SetupExternalDB(t, kubectlOptions)
		helmOptions := &helm.Options{
			ValuesFiles:    valuesFilesPaths,
			SetValues:      values,
			KubectlOptions: kubectlOptions,
			Logger:         logger.Discard,
		}

		output := helm.RenderTemplate(t, helmOptions, helmChartPath, chartName, []string{})

		// ******* WHEN *********

		k8s.KubectlApplyFromString(t, kubectlOptions, output)

		// ******* THEN *********

		pods := utils.CheckMinimumNumberOfRunningPods(t, kubectlOptions, expectedPods)
		utils.CheckPodsInContainersReadyState(t, kubectlOptions, pods)
		utils.CheckSonarQubeUpAndRunning(t, kubectlOptions, chartName)
	})
}
