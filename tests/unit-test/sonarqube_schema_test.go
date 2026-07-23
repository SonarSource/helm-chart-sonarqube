package tests

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

var chartPath string = "../../charts/sonarqube"
var releaseName string = "sonarqube"
var expectedContainerImage string = "sonarqube:26.7.0.124771"
var sqStsTemplate []string = []string{"templates/sonarqube-sts.yaml"}

func newSQHelmOptions() *helm.Options {
	return &helm.Options{
		ExtraArgs: map[string][]string{"install": {"--dry-run"}},
		SetValues: map[string]string{"monitoringPasscode": "monitoring-test-passcode"}, // See SONAR-24068
		Logger:    logger.Discard,
	}
}

func renderSQStsTemplate(t *testing.T, valuesFile string, helmOptions *helm.Options) appsv1.StatefulSet {
	t.Helper()
	helmOptions.ValuesFiles = []string{valuesFile}
	output, err := helm.RenderTemplateE(t, helmOptions, chartPath, releaseName, sqStsTemplate)
	assert.NoError(t, err)
	var rendered appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &rendered)
	return rendered
}

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
			opts := newSQHelmOptions()
			opts.ValuesFiles = test.valuesFilesPaths
			_, err := helm.RenderTemplateE(t, opts, chartPath, releaseName, []string{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.expectedError)
		})
	}
}

func TestNoTagLatestCommunity(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-notag-latest-cb.yaml", newSQHelmOptions())

	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage+"-community", actualContainers[0].Image)
}

func TestNoTagAppVersionDeveloper(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-notag-appVersion-developer.yaml", newSQHelmOptions())

	chartMetadata, err := chartutil.LoadChartfile(fmt.Sprintf("%s/Chart.yaml", chartPath))
	if err != nil {
		t.Fatalf("Error loading Chart.yaml file: %s", err)
	}

	expectedImage := fmt.Sprintf("sonarqube:%s-developer", chartMetadata.AppVersion)
	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedImage, actualContainers[0].Image)
}

func TestShouldUseBuildNumber(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-build-number.yaml", newSQHelmOptions())

	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, expectedContainerImage+"-community", actualContainers[0].Image)
}

func TestShouldUseImageTag(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-image-tag-developer.yaml", newSQHelmOptions())

	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, "sonarqube:2026.4.0-enterprise", actualContainers[0].Image)
}

func TestCustomCommunityTag(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-custom-image-values.yaml", newSQHelmOptions())

	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, "sonarqube:lts-community@sha256:3596d14feb065a31ce84cef60cc3ecfb7b47233ef860fd85c0d4e465f676c9f7", actualContainers[0].Image)
}

func TestCommunityBuildNumberEmptyTag(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-community-build-empty-tag.yaml", newSQHelmOptions())

	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, "sonarqube:12345-community", actualContainers[0].Image)
}

func TestCommunityEmptyBuildNumberEmptyTag(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-community-empty-build-empty-tag.yaml", newSQHelmOptions())
	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, "sonarqube:community", actualContainers[0].Image)
}

// TestCiValues loads the values.yaml used by CI at runtime.
func TestCiValues(t *testing.T) {
	rendered := renderSQStsTemplate(t, chartPath+"/ci/ci-values.yaml", newSQHelmOptions())
	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, "sonarsource/"+expectedContainerImage+"-master-community", actualContainers[0].Image)
}

// TestCiOpenshiftVerifierValues loads the values.yaml used by the OpenShift Verifier at runtime.
func TestCiOpenshiftVerifierValues(t *testing.T) {
	rendered := renderSQStsTemplate(t, chartPath+"/openshift-verifier/values.yaml", newSQHelmOptions())
	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, "sonarsource/"+expectedContainerImage+"-master-community", actualContainers[0].Image)
}

func TestDeveloperEdition(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-developer-edition.yaml", newSQHelmOptions())
	actualContainers := rendered.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(actualContainers))
	assert.Equal(t, "sonarqube:2026.4.0-developer", actualContainers[0].Image)
}

func findVolumeByName(volumes []v1.Volume, name string) *v1.Volume {
	for _, volume := range volumes {
		if volume.Name == name {
			return &volume
		}
	}
	return nil
}

func TestHostPath(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-persistence-hostpath.yaml", newSQHelmOptions())
	volume := findVolumeByName(rendered.Spec.Template.Spec.Volumes, "sonarqube")
	assert.NotNil(t, volume, "Volume 'sonarqube' should exist but was not found")
	assert.NotNil(t, volume.HostPath, "sonarqube volume should have a HostPath source")
	assert.Equal(t, "/hostPath/path", volume.HostPath.Path)
	assert.Equal(t, v1.HostPathDirectoryOrCreate, *volume.HostPath.Type)
}

func TestPersistenceWithoutHostpath(t *testing.T) {
	rendered := renderSQStsTemplate(t, "test-cases-values/sonarqube/test-persistence-without-hostpath.yaml", newSQHelmOptions())
	volume := findVolumeByName(rendered.Spec.Template.Spec.Volumes, "sonarqube")
	assert.NotNil(t, volume, "Volume 'sonarqube' should exist but was not found")
	assert.Nil(t, volume.HostPath, "sonarqube volume should NOT have a HostPath source")
	assert.Equal(t, "sonarqube-sonarqube", volume.PersistentVolumeClaim.ClaimName)
}
