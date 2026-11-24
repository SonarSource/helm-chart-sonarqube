package tests

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

func renderGrace(t *testing.T, chart, tpl string, set map[string]string, out interface{}) error {
	t.Helper()
	opts := &helm.Options{Logger: logger.Discard, SetValues: set}
	o, err := helm.RenderTemplateE(t, opts, chart, "grace", []string{tpl})
	if err == nil {
		helm.UnmarshalK8SYaml(t, o, out)
	}
	return err
}

func TestStandaloneTerminationGracePeriod(t *testing.T) {
	var sts appsv1.StatefulSet
	set := map[string]string{"monitoringPasscode": "x", "edition": "developer"}
	require.NoError(t, renderGrace(t, chartPath, "templates/sonarqube-sts.yaml", set, &sts))
	assert.Nil(t, sts.Spec.Template.Spec.TerminationGracePeriodSeconds)

	sts = appsv1.StatefulSet{}
	set["terminationGracePeriodSeconds"] = "120"
	require.NoError(t, renderGrace(t, chartPath, "templates/sonarqube-sts.yaml", set, &sts))
	require.NotNil(t, sts.Spec.Template.Spec.TerminationGracePeriodSeconds)
	assert.Equal(t, int64(120), *sts.Spec.Template.Spec.TerminationGracePeriodSeconds)

	var dep appsv1.Deployment
	set["deploymentType"] = "Deployment"
	require.NoError(t, renderGrace(t, chartPath, "templates/deployment.yaml", set, &dep))
	require.NotNil(t, dep.Spec.Template.Spec.TerminationGracePeriodSeconds)
	assert.Equal(t, int64(120), *dep.Spec.Template.Spec.TerminationGracePeriodSeconds)

	delete(set, "deploymentType")
	set["terminationGracePeriodSeconds"] = "0"
	sts = appsv1.StatefulSet{}
	require.NoError(t, renderGrace(t, chartPath, "templates/sonarqube-sts.yaml", set, &sts))
	require.NotNil(t, sts.Spec.Template.Spec.TerminationGracePeriodSeconds)
	assert.Equal(t, int64(0), *sts.Spec.Template.Spec.TerminationGracePeriodSeconds)

	set["terminationGracePeriodSeconds"] = "-1"
	assert.Error(t, renderGrace(t, chartPath, "templates/sonarqube-sts.yaml", set, &sts))
}

func TestDCETerminationGracePeriod(t *testing.T) {
	grace := func(set map[string]string) (app, search *int64) {
		base := dceEsMajorBaseValues()
		for k, v := range set {
			base[k] = v
		}
		var d appsv1.Deployment
		var s appsv1.StatefulSet
		require.NoError(t, renderGrace(t, dceChartPath, "templates/sonarqube-application.yaml", base, &d))
		require.NoError(t, renderGrace(t, dceChartPath, "templates/sonarqube-search.yaml", base, &s))
		return d.Spec.Template.Spec.TerminationGracePeriodSeconds, s.Spec.Template.Spec.TerminationGracePeriodSeconds
	}

	app, search := grace(nil)
	assert.Nil(t, app)
	assert.Nil(t, search)

	app, search = grace(map[string]string{"applicationNodes.terminationGracePeriodSeconds": "120"})
	require.NotNil(t, app)
	assert.Equal(t, int64(120), *app)
	assert.Nil(t, search)

	app, _ = grace(map[string]string{"applicationNodes.terminationGracePeriodSeconds": "0"})
	require.NotNil(t, app)
	assert.Equal(t, int64(0), *app)

	app, search = grace(map[string]string{"searchNodes.terminationGracePeriodSeconds": "120"})
	assert.Nil(t, app)
	require.NotNil(t, search)
	assert.Equal(t, int64(120), *search)
}
