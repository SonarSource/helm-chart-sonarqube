package tests

import (
	"crypto/tls"
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"
	"time"

	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TESTS_ENABLING_ACTION is the helm value key that toggles the chart's
// in-cluster `helm test` pods. The dynamic compatibility tests disable it.
const TESTS_ENABLING_ACTION = "tests.enabled"

// nonNamespaceChar matches anything outside the Kubernetes namespace charset
// (RFC 1123: lowercase alphanumeric and dashes).
var nonNamespaceChar = regexp.MustCompile(`[^a-z0-9-]+`)

// NamespaceFor returns an RFC 1123-compliant Kubernetes namespace name derived
// from the running test's name plus a short random suffix. The test name keeps
// the namespace easy to spot in `kubectl get ns`; the random suffix means
// concurrent runs and leftover namespaces from a crashed run never collide.
//
// Usage:
//
//	namespaceName := dyntest.NamespaceFor(t)
func NamespaceFor(chartName string) string {
	base := nonNamespaceChar.ReplaceAllString(strings.ToLower(chartName), "-")
	base = strings.Trim(base, "-")

	name := base + "-" + strings.ToLower(random.UniqueId())
	if len(name) > 63 {
		name = strings.TrimRight(name[:63], "-")
	}
	return name
}

// WaitUntilPodContainersReady will query a given pod until the condition `ContainersReady` becomes true.
// If there are errors, it will raise them.
func WaitUntilPodContainersReady(t *testing.T, options *k8s.KubectlOptions, podName string, retries int, sleepBetweenRetries time.Duration) {
	require.NoError(t, WaitUntilPodContainersReadyE(t, options, podName, retries, sleepBetweenRetries))
}

// WaitUntilPodContainersReadyE will query a given pod until the condition `ContainersReady` becomes true.
func WaitUntilPodContainersReadyE(t *testing.T, options *k8s.KubectlOptions, podName string, retries int, sleepBetweenRetries time.Duration) error {
	statusMsg := fmt.Sprintf("Wait for pod %s to be running.", podName)
	_, err := retry.DoWithRetryE(
		t,
		statusMsg,
		retries,
		sleepBetweenRetries,
		func() (string, error) {
			pod, err := k8s.GetPodE(t, options, podName)
			if err != nil {
				return "", err
			}
			if !k8s.IsPodAvailable(pod) {
				return "", k8s.NewPodNotAvailableError(pod)
			}
			conditions := pod.Status.Conditions
			for _, condition := range conditions {
				if condition.Type == "ContainersReady" {
					//fmt.Printf("condition: %v\n", condition.Status)
					if condition.Status == "False" {
						return "", k8s.NewPodNotAvailableError(pod)
					}
				}
			}
			return "Pod had now containersReady set to true", nil
		},
	)
	if err != nil {
		// logger.Logf(t, "Timedout waiting for Pod to be containersReady: %s", err)
		return err
	}
	// logger.Logf(t, "%s", message)
	return nil
}

// TearDownTestSetup deletes the test namespace and waits for it to disappear.
func TearDownTestSetup(t *testing.T, options *k8s.KubectlOptions, namespaceName string) {
	log.Println("setup test: Remove leftovers from previous deployments in the same namespace")
	WaitUntilNamespaceDeletedE(t, options, namespaceName)
}

// WaitUntilNamespaceDeletedE checks if the namespaceName namespace exists and,
// in the positive case, tries to delete it.
func WaitUntilNamespaceDeletedE(t *testing.T, options *k8s.KubectlOptions, namespaceName string) {
	namespace, _ := k8s.GetNamespaceE(t, options, namespaceName)
	namespaceActualName := namespace.Name

	for namespaceActualName == namespaceName {
		fmt.Printf("Deleting an existing namespace: %v\n", namespaceActualName)
		k8s.DeleteNamespaceE(t, options, namespaceActualName) // asynchronous call
		// verify if the namespace still exists, it takes some time to delete it
		namespace, _ = k8s.GetNamespaceE(t, options, namespaceName)
		namespaceActualName = namespace.Name
		time.Sleep(10 * time.Second)
	}
}

// CheckMinimumNumberOfRunningPods will check if the expected number of pods is available (Jobs are excluded)
func CheckMinimumNumberOfRunningPods(t *testing.T, kubectlOptions *k8s.KubectlOptions, expectedPods int) []corev1.Pod {
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

// CheckPodsInContainersReadyState will check that all pods in the list have
// the condition `ContainersReady` set to true.
func CheckPodsInContainersReadyState(t *testing.T, kubectlOptions *k8s.KubectlOptions, pods []corev1.Pod) {
	for _, pod := range pods {
		fmt.Printf("Checking, %v\n", pod.Name)
		WaitUntilPodContainersReady(t, kubectlOptions, pod.Name, 150, 30*time.Second)
	}
}

// CheckSonarQubeUpAndRunning will check if the deployed sonarqube application
// is running properly by hitting /api/system/status through a port-forward.
func CheckSonarQubeUpAndRunning(t *testing.T, kubectlOptions *k8s.KubectlOptions, chartName string) {
	podName := k8s.ListPods(t, kubectlOptions, v1.ListOptions{LabelSelector: "app=" + chartName + ",release=" + chartName})[0].Name
	fmt.Printf("Opening a tunnel to %v\n", podName)
	tunnel := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, podName, 0, 9000)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	retries := 15
	sleep := 5 * time.Second
	endpoint := fmt.Sprintf("http://%s/api/system/status", tunnel.Endpoint())
	tlsConfig := tls.Config{}
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
