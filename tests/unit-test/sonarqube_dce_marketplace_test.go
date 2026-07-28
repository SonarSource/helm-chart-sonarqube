package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var marketplaceTestSchemaPath string = "../../google-cloud-marketplace-k8s-app/data-test/schema.yaml"

// Placeholder rejected by charts/sonarqube-dce/templates/validation.yaml.
var rejectedJdbcPlaceholder string = "jdbc:postgresql://myPostgres/myDatabase"

// marketplaceTestSchemaDefaults returns the property defaults declared in the GCP
// Marketplace data-test schema. These are the values fed to the deployer during the
// automated functionality test, so the chart must render with them.
func marketplaceTestSchemaDefaults(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(marketplaceTestSchemaPath)
	require.NoError(t, err)

	var schema struct {
		Properties map[string]struct {
			Default interface{} `yaml:"default"`
		} `yaml:"properties"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &schema))

	defaults := map[string]string{}
	for name, prop := range schema.Properties {
		if prop.Default != nil {
			defaults[name] = fmt.Sprintf("%v", prop.Default)
		}
	}
	return defaults
}

// TestMarketplaceDataTestSchemaRenders guards the regression where the data-test schema
// defaulted jdbcOverwrite.jdbcUrl to the placeholder rejected by validation.yaml, making
// the Marketplace deployer fail at `helm template`. The chart must render cleanly with the
// data-test defaults, and jdbcOverwrite.jdbcUrl must never be the rejected placeholder.
func TestMarketplaceDataTestSchemaRenders(t *testing.T) {
	defaults := marketplaceTestSchemaDefaults(t)

	require.Contains(t, defaults, "jdbcOverwrite.jdbcUrl")
	assert.NotEqual(t, rejectedJdbcPlaceholder, defaults["jdbcOverwrite.jdbcUrl"],
		"data-test jdbcOverwrite.jdbcUrl must not be the placeholder rejected by validation.yaml")

	setValues := map[string]string{"gcp_marketplace": "true"}
	for k, v := range defaults {
		setValues[k] = v
	}

	helmOptions := &helm.Options{
		Logger:    logger.Discard,
		SetValues: setValues,
	}
	_, err := helm.RenderTemplateE(t, helmOptions, dceChartPath, dceReleaseName, []string{"templates/sonarqube-application.yaml"})
	assert.NoError(t, err, "DCE chart must render with the GCP Marketplace data-test schema defaults")
}
