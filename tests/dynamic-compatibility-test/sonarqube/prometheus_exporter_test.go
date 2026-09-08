package sonarqube

import (
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	utils "github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test"
)

const prometheusExporterSHA256 = "a95983fd96e865d2bcdf911cc500e7c82808c27ab9fd226bf96732b6c3d8c46e"

func sonarqubeChartPath() string {
	_, testFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(testFile), "..", "..", "..", "charts", "sonarqube")
}

func TestPrometheusExporter(t *testing.T) {
	namespaceName := utils.NamespaceFor("sonarqube-prometheus-exporter")
	existingKubectlOptions := k8s.NewKubectlOptions("", "", "default")
	k8s.CreateNamespace(t, existingKubectlOptions, namespaceName)
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	defer utils.DeleteNamespaceAndWait(t, kubectlOptions, namespaceName)

	helmOptions := &helm.Options{
		SetValues: map[string]string{
			utils.TESTS_ENABLING_ACTION:               "false",
			"edition":                                 "enterprise",
			"monitoringPasscode":                      "monitoringPasscode",
			"prometheusExporter.enabled":              "true",
			"prometheusExporter.sha256":               prometheusExporterSHA256,
		},
		KubectlOptions: kubectlOptions,
		ExtraArgs: map[string][]string{
			"install": {"--wait", "--timeout", "15m"},
		},
	}
	helm.Install(t, helmOptions, sonarqubeChartPath(), "sonarqube")

	utils.WaitForChartReady(t, kubectlOptions, "sonarqube")
	utils.CheckSonarQubeUpAndRunning(t, kubectlOptions, "sonarqube")

	pods := k8s.ListPods(t, kubectlOptions, metav1.ListOptions{LabelSelector: "app=sonarqube,release=sonarqube"})
	require.NotEmpty(t, pods)
	pod := pods[0]
	initContainer, found := findInitContainer(pod.Status.InitContainerStatuses, "inject-prometheus-exporter")
	require.True(t, found, "exporter init container should be present")
	require.NotNil(t, initContainer.State.Terminated)
	require.Equal(t, int32(0), initContainer.State.Terminated.ExitCode)

	_, err := k8s.RunKubectlAndGetOutputE(t, kubectlOptions,
		"exec", pod.Name, "-c", "sonarqube", "--", "test", "-s",
		"/opt/sonarqube/data/jmx_prometheus_javaagent.jar",
	)
	require.NoError(t, err, "the downloaded exporter jar should be non-empty")

	for _, port := range []int{8000, 8001} {
		tunnel := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, pod.Name, 0, port)
		tunnel.ForwardPort(t)
		checkMetricsEndpoint(t, tunnel.Endpoint()+"/metrics", true)
		checkMetricsEndpoint(t, tunnel.Endpoint()+"/", false)
		tunnel.Close()
	}
}

func findInitContainer(statuses []v1.ContainerStatus, name string) (v1.ContainerStatus, bool) {
	for _, status := range statuses {
		if status.Name == name {
			return status, true
		}
	}
	return v1.ContainerStatus{}, false
}

func checkMetricsEndpoint(t *testing.T, endpoint string, expectMetrics bool) {
	t.Helper()
	response, err := http.Get("http://" + endpoint)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	bodyText := string(body)

	require.Equal(t, http.StatusOK, response.StatusCode)
	if expectMetrics {
		require.Contains(t, bodyText, "# TYPE")
		require.Contains(t, bodyText, "jmx_scrape_duration_seconds")
		require.Contains(t, bodyText, "java_lang_")
	} else {
		require.NotContains(t, bodyText, "# TYPE")
		require.False(t, strings.Contains(bodyText, "jmx_scrape_duration_seconds"))
	}
}
