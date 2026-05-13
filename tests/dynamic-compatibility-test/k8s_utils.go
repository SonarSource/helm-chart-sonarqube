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
	"github.com/stretchr/testify/require"

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
func NamespaceFor(chartName string) string {
	base := nonNamespaceChar.ReplaceAllString(strings.ToLower(chartName), "-")
	base = strings.Trim(base, "-")

	name := base + "-" + strings.ToLower(random.UniqueId())
	if len(name) > 63 {
		name = strings.TrimRight(name[:63], "-")
	}
	return name
}

// WaitForChartReady blocks until every Deployment and StatefulSet created by
// the given Helm release is fully rolled out — i.e. its replica count matches
// and every replica is Ready. Resources are discovered by the standard Helm
// `release` label rather than hardcoded names, so this works for charts with
// multiple workloads (e.g. DCE's separate app/search tiers).
//
// `kubectl rollout status` is the canonical Kubernetes primitive for this and
// handles pod recreations, slow image pulls, and readiness gates without the
// snapshot-staleness and false-positive problems of the previous custom waits.
func WaitForChartReady(t *testing.T, options *k8s.KubectlOptions, chartName string) {
	out, err := k8s.RunKubectlAndGetOutputE(t, options,
		"get", "statefulset,deployment",
		"-l", "release="+chartName,
		"-o", "name",
	)
	require.NoError(t, err, "failed to list workloads for release %q", chartName)

	resources := strings.Fields(strings.TrimSpace(out))
	require.NotEmpty(t, resources, "no Deployments or StatefulSets found for release %q", chartName)

	for _, resource := range resources {
		fmt.Printf("Waiting for rollout of %s\n", resource)
		k8s.RunKubectl(t, options,
			"rollout", "status", resource, "--timeout=15m")
	}
}

// DeleteNamespaceAndWait checks if the namespaceName namespace exists and,
// in the positive case, tries to delete it.
func DeleteNamespaceAndWait(t *testing.T, options *k8s.KubectlOptions, namespaceName string) {
	log.Println("Delete the namespace")
	namespace, err := k8s.GetNamespaceE(t, options, namespaceName)
	if err != nil || namespace == nil {
		return //namespace doesn't exist, nothing to delete
	}

	namespaceActualName := namespace.Name

	for namespaceActualName == namespaceName {
		fmt.Printf("Deleting an existing namespace: %v\n", namespaceActualName)
		k8s.DeleteNamespaceE(t, options, namespaceActualName) // asynchronous call
		// verify if the namespace still exists, it takes some time to delete it
		namespace, err = k8s.GetNamespaceE(t, options, namespaceName)
		if err != nil || namespace == nil {
			return //namespace doesn't exist, nothing to delete
		}

		namespaceActualName = namespace.Name
		time.Sleep(10 * time.Second)
	}
}

// CheckSonarQubeUpAndRunning will check if the deployed sonarqube application
// is running properly by hitting /api/system/status through a port-forward.
//
// For the DCE chart the label selector is narrowed to the app tier — search
// pods carry the same `app=`/`release=` labels but don't listen on 9000.
func CheckSonarQubeUpAndRunning(t *testing.T, kubectlOptions *k8s.KubectlOptions, chartName string) {
	selector := "app=" + chartName + ",release=" + chartName
	if chartName == "sonarqube-dce" {
		selector += ",sonarqube.datacenter/type=app"
	}
	pods := k8s.ListPods(t, kubectlOptions, v1.ListOptions{LabelSelector: selector})
	require.NotEmpty(t, pods, "no pods match selector %q", selector)
	podName := pods[0].Name
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
