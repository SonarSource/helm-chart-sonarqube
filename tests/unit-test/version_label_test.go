package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const versionLabel = "app.kubernetes.io/version"

const testDigest = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// versionLabelCases maps an image tag to the app.kubernetes.io/version label it must produce.
// Kubernetes label values are at most 63 chars of [A-Za-z0-9-_.], starting and ending alphanumeric.
var versionLabelCases = []struct {
	name     string
	tag      string
	expected string
}{
	{"plain tag", "2026.1.0-enterprise", "2026.1.0-enterprise"},
	{"tag with digest", "2026.1.0-enterprise@" + testDigest, "2026.1.0-enterprise"},
	{"truncation ending on dash", strings.Repeat("a", 62) + "-b", strings.Repeat("a", 62)},
	{"truncation ending on dot", strings.Repeat("a", 62) + ".b", strings.Repeat("a", 62)},
	{"truncation ending on underscores", strings.Repeat("a", 60) + "___b", strings.Repeat("a", 60)},
}

type versionLabelTarget struct {
	name      string
	chartPath string
	release   string
	template  string
	values    map[string]string
	tagKey    string
}

var versionLabelTargets = []versionLabelTarget{
	{
		name:      "sonarqube/sts",
		chartPath: chartPath,
		release:   releaseName,
		template:  "templates/sonarqube-sts.yaml",
		values:    map[string]string{"community.enabled": "true", "monitoringPasscode": "test-passcode"},
		tagKey:    "image.tag",
	},
	{
		name:      "sonarqube-dce/application",
		chartPath: dceChartPath,
		release:   dceReleaseName,
		template:  "templates/sonarqube-application.yaml",
		values:    dceEsMajorBaseValues(),
		tagKey:    "applicationNodes.image.tag",
	},
	{
		name:      "sonarqube-dce/search",
		chartPath: dceChartPath,
		release:   dceReleaseName,
		template:  "templates/sonarqube-search.yaml",
		values:    dceEsMajorBaseValues(),
		tagKey:    "searchNodes.image.tag",
	},
}

func renderWorkloadLabels(t *testing.T, target versionLabelTarget, tag string) map[string]string {
	t.Helper()
	setValues := map[string]string{target.tagKey: tag}
	for k, v := range target.values {
		setValues[k] = v
	}
	opts := &helm.Options{Logger: logger.Discard, SetValues: setValues}
	output, err := helm.RenderTemplateE(t, opts, target.chartPath, target.release, []string{target.template})
	require.NoError(t, err)

	var workload struct {
		Metadata metav1.ObjectMeta `json:"metadata"`
	}
	helm.UnmarshalK8SYaml(t, output, &workload)
	return workload.Metadata.Labels
}

func TestVersionLabelIsValidLabelValue(t *testing.T) {
	for _, target := range versionLabelTargets {
		for _, tc := range versionLabelCases {
			t.Run(target.name+"/"+tc.name, func(t *testing.T) {
				label := renderWorkloadLabels(t, target, tc.tag)[versionLabel]
				assert.Equal(t, tc.expected, label)
				assert.Empty(t, validation.IsValidLabelValue(label))
			})
		}
	}
}
