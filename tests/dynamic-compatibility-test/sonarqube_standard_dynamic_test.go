package tests

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Teardown resources after test execution: namespace deleted
func SetupTest(t *testing.T, options *k8s.KubectlOptions, namespaceName string) func(t *testing.T, options *k8s.KubectlOptions, namespaceName string) {
	log.Println("setup test: Remove leftovers from previous deployments in the same namespace")
	WaitUntilNamespaceDeletedE(t, options, namespaceName)

	return func(t *testing.T, options *k8s.KubectlOptions, namespaceName string) {
		log.Println("teardown test")
		WaitUntilNamespaceDeletedE(t, options, namespaceName)
	}
}

// WaitUntilNamespaceDeletedE checks if the `namespaceName` namespace exists and, in the positive case, tries to delete it.
func WaitUntilNamespaceDeletedE(t *testing.T, options *k8s.KubectlOptions, namespaceName string) {
	namespace, _ := k8s.GetNamespaceE(t, options, namespaceName)
	namespaceActualName := namespace.Name

	for namespaceActualName == namespaceName {
		fmt.Printf("Deleting an existing namespace: %v\n", namespaceActualName)
		k8s.DeleteNamespaceE(t, options, namespaceActualName) // this is an asyncronous call
		// verify if the namespace still exists, it takes some time to delete it
		namespace, _ = k8s.GetNamespaceE(t, options, namespaceName)
		namespaceActualName = namespace.Name
		time.Sleep(10 * time.Second)
	}
}

// This test checks the dynamic compatibility test for the sonarqube standard and dce charts
// Specifically, we perform tests that:
// - check if the required (default) number of pods is running
// - check if all the running pods have the containersReady condition set to true
// - check if the sonarqube application is ready (apis are ready to serve new requests)
func TestSonarQubeChartDynamicCompatibility(t *testing.T) {

	// Input values for our charts' tests
	table := []struct {
		name             string
		chartName        string
		expectedPods     int
		values           map[string]string
		valuesFilesPaths []string
	}{
		{"standard-chart-default-values-deployment", "sonarqube", 2, map[string]string{"tests.enabled": "false",
			"deploymentType":             "Deployment",
			"prometheusExporter.enabled": "false", // only for deployment
		}, []string{"../../charts/sonarqube/values.yaml"}},

		{"standard-chart-default-values-sts", "sonarqube", 2, map[string]string{"tests.enabled": "false"},
			[]string{"../../charts/sonarqube/values.yaml"}},

		{"standard-chart-all-values-deployment", "sonarqube", 2, map[string]string{"tests.enabled": "false",
			"deploymentType":             "Deployment",
			"prometheusExporter.enabled": "false", // only for deployment
		}, []string{"sonarqube/all-values.yaml"}},

		{"standard-chart-all-values-sts", "sonarqube", 2, map[string]string{"tests.enabled": "false"},
			[]string{"sonarqube/all-values.yaml"}},

		{"dce-chart-default-values", "sonarqube-dce", 6, map[string]string{"tests.enabled": "false",
			"ApplicationNodes.jwtSecret": "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=",
		}, []string{"../../charts/sonarqube-dce/values.yaml"}},

		{"dce-chart-all-values", "sonarqube-dce", 6, map[string]string{"tests.enabled": "false",
			"ApplicationNodes.jwtSecret": "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=",
		}, []string{"sonarqube-dce/all-values.yaml"}},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {

			// ******* GIVEN *********

			// gather the inputs required for the test
			chartName := tc.chartName
			values := tc.values
			expectedPods := tc.expectedPods
			valuesFilesPaths := tc.valuesFilesPaths
			namespaceName := chartName + "-dynamic-test"

			// Path to the helm chart we will test
			helmChartPath := "../../charts/" + chartName

			// Setup the kubectl config and context. Here we choose to use the defaults, which is:
			// - HOME/.kube/config for the kubectl config file
			// - Current context of the kubectl config file
			existingKubectlOptions := k8s.NewKubectlOptions("", "", "default")

			teardownTest := SetupTest(t, existingKubectlOptions, namespaceName)

			// Define a new namespace
			k8s.CreateNamespace(t, existingKubectlOptions, namespaceName)
			kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)

			// delete all resources after test execution
			defer teardownTest(t, kubectlOptions, namespaceName)

			// Setup the args
			options := &helm.Options{
				ValuesFiles:    valuesFilesPaths,
				SetValues:      values,
				KubectlOptions: kubectlOptions,
			}

			// Run RenderTemplate to render the template and capture the output.
			output := helm.RenderTemplate(t, options, helmChartPath, chartName, []string{})

			// ******* WHEN *********

			// Now use kubectl to apply the rendered template
			k8s.KubectlApplyFromString(t, kubectlOptions, output)

			// ******* THEN *********

			pods := CheckMinimumNumberOfRunningPods(t, kubectlOptions, expectedPods)
			CheckPodsInContainersReadyState(t, kubectlOptions, pods)
			CheckSonarQubeUpAndRunning(t, kubectlOptions, chartName)

		})
	}
}

// CheckMinimumNumberOfRunningPods will check if the expected number of pods is available (Jobs are excluded)
func CheckMinimumNumberOfRunningPods(t *testing.T, kubectlOptions *k8s.KubectlOptions, expectedPods int) []corev1.Pod {
	// check the number of pods before moving on
	pods := k8s.ListPods(t, kubectlOptions, v1.ListOptions{LabelSelector: "!job-name"})
	lenPods := len(pods)
	for lenPods < expectedPods {
		pods = k8s.ListPods(t, kubectlOptions, v1.ListOptions{LabelSelector: "!job-name"})
		time.Sleep(5 * time.Second)
		fmt.Printf("Currently,  %v pods are running\n", len(pods))
		lenPods = len(pods)
	}

	return pods
}

// CheckPodsInContainersReadyState will check that all pods in the list have the condition `ContainersReady` set to true
func CheckPodsInContainersReadyState(t *testing.T, kubectlOptions *k8s.KubectlOptions, pods []corev1.Pod) {
	for _, pod := range pods {
		fmt.Printf("Checking, %v\n", pod.Name)
		// wait until all pods have the condition "ContainersReady"
		WaitUntilPodContainersReady(t, kubectlOptions, pod.Name, 150, 30*time.Second)
	}
}

// CheckSonarQubeUpAndRunning will check if the deployed sonarqube application is running properly
func CheckSonarQubeUpAndRunning(t *testing.T, kubectlOptions *k8s.KubectlOptions, chartName string) {

	// open a tunnel to the running sonarqube instance using a random port
	pod_name := k8s.ListPods(t, kubectlOptions, v1.ListOptions{LabelSelector: "app=" + chartName + ",release=" + chartName})[0].Name
	fmt.Printf("Opening a tunnel to %v\n", pod_name)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, pod_name, 0, 9000)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	// verify that we get back a 200 OK with the "status UP" message
	retries := 15
	sleep := 5 * time.Second
	endpoint := fmt.Sprintf("http://%s/api/system/status", tunnel.Endpoint())
	tlsConfig := tls.Config{} // empty TLS config
	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		endpoint,
		&tlsConfig,
		retries,
		sleep,
		func(statusCode int, body string) bool {
			isOk := statusCode == 200
			isUp := strings.Contains(body, "\"status\":\"UP\"")
			return isOk && isUp
		},
	)

}
