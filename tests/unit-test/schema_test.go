package tests

import (
	"testing"

	"fmt"
	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

// Global variables
// chart path
var chartPath string = "../../charts/sonarqube"

// release name
var releaseName string = "sonarqube"

// Ensure we are using the dry-run flag
var helmOptions *helm.Options = &helm.Options{
	ExtraArgs: map[string][]string{"install": {"--dry-run"}},
	SetValues: map[string]string{"monitoringPasscode": "monitoring-test-passcode"}, // See SONAR-24068
	Logger:    logger.Discard,
}

// Templates to be tested
var sqStsTemplate []string = []string{"templates/sonarqube-sts.yaml"}

// TestValidSchema tests the schema of the values.yaml file using the validations
// inside values.schema.json
func TestInvalidSchema(t *testing.T) {
	table := []struct {
		testCaseName     string
		valuesFilesPaths []string
		expectedError    string
	}{
		{
			testCaseName:     "invalid-edition",
			valuesFilesPaths: []string{"test-cases-values/sonarqube/invalid-edition.yaml"},
			expectedError:    "The 'edition' must be either 'developer' or 'enterprise'.",
		},
		{
			testCaseName:     "no-default-edition",
			valuesFilesPaths: []string{"test-cases-values/sonarqube/invalid-no-default-edition.yaml"},
			expectedError:    "You must choose an 'edition' to install: 'developer' or 'enterprise'. If you want to use SonarQube Community Build, unset 'edition' and set 'community.enabled=true' instead.",
		},
		{
			testCaseName:     "community-disabled-no-edition",
			valuesFilesPaths: []string{"test-cases-values/sonarqube/invalid-community-disabled-no-edition.yaml"},
			expectedError:    "You must choose an 'edition' to install: 'developer' or 'enterprise'. If you want to use SonarQube Community Build, unset 'edition' and set 'community.enabled=true' instead.",
		},
		{
			testCaseName:     "invalid-community-edition",
			valuesFilesPaths: []string{"test-cases-values/sonarqube/invalid-community-edition.yaml"},
			expectedError:    "'community' is not a valid edition. If you want to use SonarQube Community Build, unset 'edition' and set 'community.enabled=true' instead.",
		},
	}

	for _, test := range table {
		t.Run(test.testCaseName, func(t *testing.T) {
			helmOptions.ValuesFiles = test.valuesFilesPaths
			_, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, []string{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.expectedError)
		})
	}
}

func TestNoTagLatestCommunity(t *testing.T) {
	helmOptions.ValuesFiles = []string{"test-cases-values/sonarqube/test-notag-latest-cb.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	// Now we use kubernetes/client-go library to render the template output into the Deployment struct. This will
	// ensure the Deployment resource is rendered correctly.
	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	expectedContainerImage := "sonarqube:25.5.0.107428-community"
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}

func TestNoTagAppVersionDeveloper(t *testing.T) {
	helmOptions.ValuesFiles = []string{"test-cases-values/sonarqube/test-notag-appVersion-developer.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	// Get the appVersion from the Chart.yaml file
	var chartMetadata *chart.Metadata
	chartMetadata, err = chartutil.LoadChartfile(fmt.Sprintf("%s/Chart.yaml", chartPath))
	if err != nil {
		t.Fatalf("Error loading Chart.yaml file: %s", err)
	}
	var appVersion string = chartMetadata.AppVersion

	expectedContainerImage := fmt.Sprintf("sonarqube:%s-developer", appVersion)
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}

func TestShouldUseBuildNumber(t *testing.T) {
	helmOptions.ValuesFiles = []string{"test-cases-values/sonarqube/test-build-number.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	expectedContainerImage := "sonarqube:25.5.0.107428-community"
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}

func TestShouldUseImageTag(t *testing.T) {
	helmOptions.ValuesFiles = []string{"test-cases-values/sonarqube/test-image-tag-developer.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	expectedContainerImage := "sonarqube:2025.3.0-enterprise"
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}

func TestCustomCommunityTag(t *testing.T) {
	helmOptions.ValuesFiles = []string{"test-cases-values/sonarqube/test-custom-image-values.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	expectedContainerImage := "sonarqube:lts-community@sha256:3596d14feb065a31ce84cef60cc3ecfb7b47233ef860fd85c0d4e465f676c9f7"
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}

func TestCommunityBuildNumberEmptyTag(t *testing.T) {
	helmOptions.ValuesFiles = []string{"test-cases-values/sonarqube/test-community-build-empty-tag.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	expectedContainerImage := "sonarqube:12345-community"
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}

func TestCommunityEmptyBuildNumberEmptyTag(t *testing.T) {
	helmOptions.ValuesFiles = []string{"test-cases-values/sonarqube/test-community-empty-build-empty-tag.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	expectedContainerImage := "sonarqube:community"
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}

// This test loads the values.yaml used by Cirrus at runtime.
func TestCiCirrusValues(t *testing.T) {
	helmOptions.ValuesFiles = []string{chartPath + "/ci/cirrus-values.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	expectedContainerImage := "sonarsource/sonarqube:25.5.0.107428-community"
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}

// This test loads the values.yaml used by the OpenShift Verifier at runtime.
func TestCiOpenshiftVerifierValues(t *testing.T) {
	helmOptions.ValuesFiles = []string{chartPath + "/openshift-verifier/values.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	expectedContainerImage := "sonarsource/sonarqube:25.5.0.107428-community"
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}

func TestDeveloperEdition(t *testing.T) {
	helmOptions.ValuesFiles = []string{"test-cases-values/sonarqube/test-developer-edition.yaml"}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)

	var renderedTemplate appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &renderedTemplate)

	expectedContainerImage := "sonarqube:2025.3.0-developer"
	actualContainers := renderedTemplate.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage, actualContainers[0].Image)
}
