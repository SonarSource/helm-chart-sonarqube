package sonarqubedce

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
	"github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test/dependencies"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	esMajorFromChartRepo = "https://SonarSource.github.io/helm-chart-sonarqube"
	esMajorFromChartRef  = "sonarqube/sonarqube-dce"
	esMajorFromChartVer  = "2026.1.5"
	esMajorUpgradeFail   = "Elasticsearch major upgrade is not a rolling update"
)

func TestDCEEsMajorUpgrade(t *testing.T) {
	if os.Getenv("SONARQUBE_ES_MAJOR_E2E") != "1" {
		t.Skip("set SONARQUBE_ES_MAJOR_E2E=1 to run against a kind cluster")
	}

	namespaceName := utils.NamespaceFor("sonarqube-dce")
	releaseName := "sonarqube-dce"
	helmChartPath := dceChartPath()
	existingKubectlOptions := k8s.NewKubectlOptions("", "", "default")
	k8s.CreateNamespace(t, existingKubectlOptions, namespaceName)
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	defer utils.DeleteNamespaceAndWait(t, kubectlOptions, namespaceName)

	dependencies.SetupDB(t, kubectlOptions)

	installValues := esMajorUpgradeValues()
	repoOpts := &helm.Options{KubectlOptions: kubectlOptions}
	_, err := helm.RunHelmCommandAndGetOutputE(t, repoOpts, "repo", "add", "sonarqube", esMajorFromChartRepo, "--force-update")
	require.NoError(t, err)

	installOpts := &helm.Options{
		SetValues:      installValues,
		KubectlOptions: kubectlOptions,
		ExtraArgs: map[string][]string{
			"install": {"--version", esMajorFromChartVer, "--wait", "--timeout", "25m"},
		},
	}
	helm.Install(t, installOpts, esMajorFromChartRef, releaseName)
	waitSearchReplicas(t, kubectlOptions, 3)
	assertSearchImageContains(t, kubectlOptions, "2026.1")

	upgradeOpts := &helm.Options{
		SetValues:      esMajorUpgradeValues(),
		KubectlOptions: kubectlOptions,
		ExtraArgs: map[string][]string{
			"upgrade": {"--timeout", "5m"},
		},
	}
	err = helm.UpgradeE(t, upgradeOpts, helmChartPath, releaseName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), esMajorUpgradeFail)
	assertSearchImageContains(t, kubectlOptions, "2026.1")
	waitSearchReplicas(t, kubectlOptions, 3)

	scaleDownValues := esMajorUpgradeValues()
	scaleDownValues["searchNodes.replicaCount"] = "0"
	scaleDownOpts := &helm.Options{
		SetValues:      scaleDownValues,
		KubectlOptions: kubectlOptions,
		ExtraArgs: map[string][]string{
			"upgrade": {"--version", esMajorFromChartVer, "--wait", "--timeout", "10m"},
		},
	}
	helm.Upgrade(t, scaleDownOpts, esMajorFromChartRef, releaseName)
	waitSearchReplicas(t, kubectlOptions, 0)

	upgradeOpts.ExtraArgs["upgrade"] = []string{"--wait", "--timeout", "25m"}
	helm.Upgrade(t, upgradeOpts, helmChartPath, releaseName)
	waitSearchReplicas(t, kubectlOptions, 3)
	assertSearchImageContains(t, kubectlOptions, "2026.4")
	assertSearchPVCsRemain(t, kubectlOptions)
	assertAppResponds(t, kubectlOptions, releaseName)

	helm.Upgrade(t, upgradeOpts, helmChartPath, releaseName)
	waitSearchReplicas(t, kubectlOptions, 3)
	assertSearchImageContains(t, kubectlOptions, "2026.4")
	assertAppResponds(t, kubectlOptions, releaseName)
}

func dceChartPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "charts", "sonarqube-dce")
}

func esMajorUpgradeValues() map[string]string {
	return map[string]string{
		utils.TESTS_ENABLING_ACTION:                  "false",
		"ApplicationNodes.jwtSecret":                 "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=",
		"applicationNodes.jwtSecret":                 "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=",
		"applicationNodes.replicaCount":              "1",
		"monitoringPasscode":                         "monitoringPasscode",
		"jdbcOverwrite.enabled":                      "true",
		"jdbcOverwrite.jdbcUrl":                      "jdbc:postgresql://" + dependencies.PostgresHost + ":" + dependencies.PostgresPort + "/" + dependencies.PostgresDatabase,
		"jdbcOverwrite.jdbcUsername":                 dependencies.PostgresUsername,
		"jdbcOverwrite.jdbcSecretName":               dependencies.PostgresSecretName,
		"jdbcOverwrite.jdbcSecretPasswordKey":        dependencies.PostgresSecretPasswordKey,
		"searchNodes.replicaCount":                   "3",
		"searchNodes.persistence.size":               "1Gi",
		"searchNodes.resources.limits.memory":        "3072M",
		"searchNodes.resources.requests.memory":      "3072M",
		"applicationNodes.resources.limits.memory":   "4096M",
		"applicationNodes.resources.requests.memory": "4096M",
	}
}

func waitSearchReplicas(t *testing.T, kubectlOptions *k8s.KubectlOptions, want int) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Minute)
	for {
		pods := k8s.ListPods(t, kubectlOptions, v1.ListOptions{LabelSelector: "sonarqube.datacenter/type=search"})
		ready := 0
		for _, pod := range pods {
			if isPodReady(pod) {
				ready++
			}
		}
		if want == 0 && len(pods) == 0 {
			return
		}
		if want > 0 && ready == want && len(pods) == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d search pods (have %d listed, %d ready)", want, len(pods), ready)
		}
		time.Sleep(5 * time.Second)
	}
}

func assertSearchImageContains(t *testing.T, kubectlOptions *k8s.KubectlOptions, needle string) {
	t.Helper()
	pods := k8s.ListPods(t, kubectlOptions, v1.ListOptions{LabelSelector: "sonarqube.datacenter/type=search"})
	require.NotEmpty(t, pods)
	for _, pod := range pods {
		require.NotEmpty(t, pod.Spec.Containers)
		assert.Contains(t, pod.Spec.Containers[0].Image, needle, "pod %s image", pod.Name)
	}
}

func assertSearchPVCsRemain(t *testing.T, kubectlOptions *k8s.KubectlOptions) {
	t.Helper()
	out, err := k8s.RunKubectlAndGetOutputE(t, kubectlOptions, "get", "pvc", "-o", "name")
	require.NoError(t, err)
	var searchPVCs []string
	for _, name := range strings.Fields(strings.TrimSpace(out)) {
		if strings.Contains(name, "search") {
			searchPVCs = append(searchPVCs, name)
		}
	}
	assert.GreaterOrEqual(t, len(searchPVCs), 3)
}

func isPodReady(pod corev1.Pod) bool {
	if pod.DeletionTimestamp != nil {
		return false
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func assertAppResponds(t *testing.T, kubectlOptions *k8s.KubectlOptions, releaseName string) {
	t.Helper()
	selector := "app=" + releaseName + ",release=" + releaseName + ",sonarqube.datacenter/type=app"
	pods := k8s.ListPods(t, kubectlOptions, v1.ListOptions{LabelSelector: selector})
	require.NotEmpty(t, pods, "no app pods match %q", selector)
	podName := pods[0].Name
	fmt.Printf("Opening a tunnel to %v\n", podName)
	tunnel := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, podName, 0, 9000)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	endpoint := fmt.Sprintf("http://%s/api/system/status", tunnel.Endpoint())
	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		endpoint,
		&tls.Config{},
		30,
		5*time.Second,
		func(statusCode int, body string) bool {
			if statusCode != 200 {
				return false
			}
			return strings.Contains(body, `"status":"UP"`) ||
				strings.Contains(body, `"status":"DB_MIGRATION_NEEDED"`) ||
				strings.Contains(body, `"status":"STARTING"`)
		},
	)
}
