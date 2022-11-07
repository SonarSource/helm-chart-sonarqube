package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/stretchr/testify/require"
)

// WaitUntilPodContainersReady will query a given pod until the condition `ContainersReady` becomes true.
// If there are errors, it will raise them.
func WaitUntilPodContainersReady(t *testing.T, options *k8s.KubectlOptions, podName string, retries int, sleepBetweenRetries time.Duration) {
	require.NoError(t, WaitUntilPodContainersReadyE(t, options, podName, retries, sleepBetweenRetries))
}

// WaitUntilPodContainersReadyE will query a given pod until the condition `ContainersReady` becomes true.
func WaitUntilPodContainersReadyE(t *testing.T, options *k8s.KubectlOptions, podName string, retries int, sleepBetweenRetries time.Duration) error {
	statusMsg := fmt.Sprintf("Wait for pod %s to be running.", podName)
	message, err := retry.DoWithRetryE(
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
						return "", k8s.NewPodNotAvailableError(pod) // TODO: Raise an ad-hoc condition here
					}
				}
			}
			return "Pod had now containersReady set to true", nil
		},
	)
	if err != nil {
		logger.Logf(t, "Timedout waiting for Pod to be containersReady: %s", err)
		return err
	}
	logger.Logf(t, message)
	return nil
}
