package tests

import (
	"io"
	"net/http"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/helm-chart-sonarqube/tests/dynamic-compatibility-test/dependencies"
)

const PrometheusExporterSHA256 = "a95983fd96e865d2bcdf911cc500e7c82808c27ab9fd226bf96732b6c3d8c46e"

type PrometheusExporterTestSpec struct {
	ChartName         string
	ChartPath         string
	Values            map[string]string
	PodSelector       string
	ContainerName     string
	RequireExternalDB bool
}

func RunPrometheusExporterTest(t *testing.T, spec PrometheusExporterTestSpec) {
	t.Helper()
	namespaceName := NamespaceFor(spec.ChartName + "-prometheus-exporter")
	existingKubectlOptions := k8s.NewKubectlOptions("", "", "default")
	k8s.CreateNamespace(t, existingKubectlOptions, namespaceName)
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	defer DeleteNamespaceAndWait(t, kubectlOptions, namespaceName)

	if spec.RequireExternalDB {
		dependencies.SetupDB(t, kubectlOptions)
	}

	helmOptions := &helm.Options{
		SetValues:      spec.Values,
		KubectlOptions: kubectlOptions,
		ExtraArgs: map[string][]string{
			"install": {"--wait", "--timeout", "15m"},
		},
	}
	helm.Install(t, helmOptions, spec.ChartPath, spec.ChartName)

	WaitForChartReady(t, kubectlOptions, spec.ChartName)
	CheckSonarQubeUpAndRunning(t, kubectlOptions, spec.ChartName)

	pods := k8s.ListPods(t, kubectlOptions, metav1.ListOptions{LabelSelector: spec.PodSelector})
	require.NotEmpty(t, pods)
	for _, pod := range pods {
		initContainer, found := findPrometheusExporterInitContainer(pod.Status.InitContainerStatuses)
		require.True(t, found, "exporter init container should be present in pod %s", pod.Name)
		require.NotNil(t, initContainer.State.Terminated)
		require.Equal(t, int32(0), initContainer.State.Terminated.ExitCode)

		_, err := k8s.RunKubectlAndGetOutputE(t, kubectlOptions,
			"exec", pod.Name, "-c", spec.ContainerName, "--", "test", "-s",
			"/opt/sonarqube/data/jmx_prometheus_javaagent.jar",
		)
		require.NoError(t, err, "the downloaded exporter jar should be non-empty in pod %s", pod.Name)

		for _, port := range []int{8000, 8001} {
			tunnel := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, pod.Name, 0, port)
			tunnel.ForwardPort(t)
			checkPrometheusExporterEndpoint(t, tunnel.Endpoint()+"/metrics", true)
			checkPrometheusExporterEndpoint(t, tunnel.Endpoint()+"/", false)
			tunnel.Close()
		}
	}
}

func findPrometheusExporterInitContainer(statuses []v1.ContainerStatus) (v1.ContainerStatus, bool) {
	for _, status := range statuses {
		if status.Name == "inject-prometheus-exporter" {
			return status, true
		}
	}
	return v1.ContainerStatus{}, false
}

func checkPrometheusExporterEndpoint(t *testing.T, endpoint string, expectMetrics bool) {
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
		require.NotContains(t, bodyText, "jmx_scrape_duration_seconds")
	}
}
