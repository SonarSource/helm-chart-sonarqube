package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

const esMajorAnnotation = "sonarqube.datacenter/elasticsearch-major"

func dceEsMajorBaseValues() map[string]string {
	return map[string]string{
		"monitoringPasscode":         "test-passcode",
		"applicationNodes.jwtSecret": "test-jwt-secret",
		"jdbcOverwrite.jdbcUrl":      "jdbc:postgresql://test-host:5432/testdb",
		"jdbcOverwrite.jdbcUsername": "test-user",
		"jdbcOverwrite.jdbcPassword": "test-password",
	}
}

func renderDCESearchWithValues(t *testing.T, setValues map[string]string) (appsv1.StatefulSet, error) {
	t.Helper()
	base := dceEsMajorBaseValues()
	for k, v := range setValues {
		base[k] = v
	}
	opts := &helm.Options{Logger: logger.Discard, SetValues: base}
	output, err := helm.RenderTemplateE(t, opts, dceChartPath, dceReleaseName, []string{"templates/sonarqube-search.yaml"})
	if err != nil {
		return appsv1.StatefulSet{}, err
	}
	var rendered appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &rendered)
	return rendered, nil
}

func TestSearchEsMajorAnnotationFromImageTag(t *testing.T) {
	cases := []struct {
		tag  string
		want string
	}{
		{"2026.1.5-datacenter-search", "8"},
		{"2026.3.1-datacenter-search", "8"},
		{"2026.4.0-datacenter-search", "9"},
		{"2026.5.0-datacenter-search", "9"},
		{"2025.1.0-datacenter-search", "8"},
		{"9.9.0-datacenter-search", ""},
		{"custom-build", ""},
	}
	for _, tc := range cases {
		t.Run(tc.tag, func(t *testing.T) {
			sts, err := renderDCESearchWithValues(t, map[string]string{
				"searchNodes.image.tag": tc.tag,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.want, sts.Annotations[esMajorAnnotation])
		})
	}
}

func TestSearchEsMajorUpgradeCheckSkippedOnInstall(t *testing.T) {
	_, err := renderDCESearchWithValues(t, map[string]string{
		"searchNodes.image.tag": "2026.1.5-datacenter-search",
	})
	require.NoError(t, err)
}
