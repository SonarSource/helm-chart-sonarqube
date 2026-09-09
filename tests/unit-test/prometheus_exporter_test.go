package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	defaultPrometheusExporterVersion = "1.6.0"
	prometheusExporterChecksum       = "a95983fd96e865d2bcdf911cc500e7c82808c27ab9fd226bf96732b6c3d8c46e"
)

type prometheusPodMonitor struct {
	Spec struct {
		PodMetricsEndpoints []struct {
			Port string `yaml:"port"`
			Path string `yaml:"path"`
		} `yaml:"podMetricsEndpoints"`
	} `yaml:"spec"`
}

func prometheusExporterValueKey(chart agentChart, field string) string {
	if chart.name == "sonarqube-dce" {
		return "ApplicationNodes.prometheusExporter." + field
	}
	return "prometheusExporter." + field
}

func prometheusExporterOptions(chart agentChart, fixtures []string, setValues map[string]string) *helm.Options {
	valuesFiles := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		valuesFiles = append(valuesFiles, chart.valuesDir+"/"+fixture)
	}
	return &helm.Options{
		Logger:      logger.Discard,
		ValuesFiles: valuesFiles,
		SetValues:   setValues,
	}
}

func renderPrometheusExporterWorkload(t *testing.T, chart agentChart, fixtures []string, setValues map[string]string) (string, error) {
	t.Helper()
	opts := prometheusExporterOptions(chart, fixtures, setValues)
	return helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{chart.appTemplate})
}

func prometheusExporterArgs(t *testing.T, chart agentChart, fixtures []string, setValues map[string]string) string {
	t.Helper()
	output, err := renderPrometheusExporterWorkload(t, chart, fixtures, setValues)
	require.NoError(t, err)

	if chart.name == "sonarqube-dce" {
		var rendered appsv1.Deployment
		helm.UnmarshalK8SYaml(t, output, &rendered)
		container := findInitContainerByName(rendered.Spec.Template.Spec.InitContainers, "inject-prometheus-exporter")
		require.NotNil(t, container)
		require.NotEmpty(t, container.Args)
		return strings.Join(container.Args, " ")
	}

	var rendered appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &rendered)
	container := findInitContainerByName(rendered.Spec.Template.Spec.InitContainers, "inject-prometheus-exporter")
	require.NotNil(t, container)
	require.NotEmpty(t, container.Args)
	return strings.Join(container.Args, " ")
}

func renderPrometheusPodMonitor(t *testing.T, chart agentChart, fixtures []string, setValues map[string]string) prometheusPodMonitor {
	t.Helper()
	opts := prometheusExporterOptions(chart, fixtures, setValues)
	output, err := helm.RenderTemplateE(t, opts, chart.path, chart.release, []string{"templates/prometheus-podmonitor.yaml"})
	require.NoError(t, err)

	var rendered prometheusPodMonitor
	helm.UnmarshalK8SYaml(t, output, &rendered)
	return rendered
}

func prometheusExporterURL(version string) string {
	if version == "1.1.0" || version == defaultPrometheusExporterVersion {
		return fmt.Sprintf("https://github.com/prometheus/jmx_exporter/releases/download/%s/jmx_prometheus_javaagent-%s.jar", version, version)
	}
	return fmt.Sprintf("https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/%s/jmx_prometheus_javaagent-%s.jar", version, version)
}

func TestPrometheusExporterDefaultRendering(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			args := prometheusExporterArgs(t, chart, []string{"prometheus-exporter-enabled.yaml"}, nil)

			assert.Contains(t, args, "curl -s -L --fail")
			assert.Contains(t, args, prometheusExporterURL(defaultPrometheusExporterVersion))
			assert.NotContains(t, args, "sha256sum")
		})
	}
}

func TestPrometheusExporterVersionAwareDownloadURL(t *testing.T) {
	cases := []struct {
		name    string
		version string
	}{
		{name: "maven", version: "0.17.2"},
		{name: "legacy-two-part", version: "0.10"},
		{name: "github-boundary", version: "1.1.0"},
	}

	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					args := prometheusExporterArgs(t, chart, []string{"prometheus-exporter-enabled.yaml"}, map[string]string{
						prometheusExporterValueKey(chart, "version"): tc.version,
					})

					assert.Contains(t, args, prometheusExporterURL(tc.version))
				})
			}
		})
	}
}

func TestPrometheusExporterCustomURLChecksumAndCertificateFlag(t *testing.T) {
	const customURL = "https://downloads.example.test/jmx_prometheus_javaagent-custom.jar"

	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			args := prometheusExporterArgs(t, chart, []string{"prometheus-exporter-enabled.yaml"}, map[string]string{
				prometheusExporterValueKey(chart, "version"):            "0.17.2",
				prometheusExporterValueKey(chart, "downloadURL"):        customURL,
				prometheusExporterValueKey(chart, "sha256"):             prometheusExporterChecksum,
				prometheusExporterValueKey(chart, "noCheckCertificate"): "true",
			})

			assert.Contains(t, args, customURL)
			assert.NotContains(t, args, prometheusExporterURL("0.17.2"))
			assert.Contains(t, args, "--insecure")
			assert.Contains(t, args, prometheusExporterChecksum)
			assert.Contains(t, args, "sha256sum -c -")
		})
	}
}

func TestPrometheusExporterChecksumDisabled(t *testing.T) {
	cases := []struct {
		name     string
		fixtures []string
	}{
		{name: "unset", fixtures: []string{"prometheus-exporter-enabled.yaml"}},
		{name: "empty", fixtures: []string{"prometheus-exporter-enabled.yaml", "prometheus-exporter-sha256-empty.yaml"}},
		{name: "null", fixtures: []string{"prometheus-exporter-enabled.yaml", "prometheus-exporter-sha256-null.yaml"}},
	}

	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					args := prometheusExporterArgs(t, chart, tc.fixtures, nil)
					assert.NotContains(t, args, "sha256sum")
				})
			}
		})
	}
}

func TestPrometheusPodMonitorMetricsPath(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range []struct {
				name string
				path string
			}{
				{name: "default", path: "/metrics"},
				{name: "custom", path: "/custom-metrics"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					setValues := map[string]string(nil)
					if tc.name == "custom" {
						setValues = map[string]string{
							prometheusExporterValueKey(chart, "metricsPath"): tc.path,
						}
					}

					monitor := renderPrometheusPodMonitor(t, chart, []string{"prometheus-exporter-enabled.yaml"}, setValues)
					pathsByPort := make(map[string]string, len(monitor.Spec.PodMetricsEndpoints))
					for _, endpoint := range monitor.Spec.PodMetricsEndpoints {
						pathsByPort[endpoint.Port] = endpoint.Path
					}

					assert.Equal(t, tc.path, pathsByPort["monitoring-web"])
					assert.Equal(t, tc.path, pathsByPort["monitoring-ce"])
				})
			}
		})
	}
}
