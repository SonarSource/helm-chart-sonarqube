package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AGENT_RUNTIME_CLASS_E2E gates RunAgentRuntimeClassCheckTest: it needs a reachable cluster and
// creates a cluster-scoped RuntimeClass, so it is opt-in like the ES major-upgrade test.
const AGENT_RUNTIME_CLASS_E2E = "SONARQUBE_RUNTIME_CLASS_E2E"

// The distinctive part of the failure the guard raises, from
// sonarqube.openshift.assertAgentRuntimeClass.
const agentRuntimeClassFail = "does not exist in this cluster"

// The values key every subtest below overrides.
const agentRuntimeClassNameKey = "OpenShift.agentRuntimeClassName"

// AgentRuntimeClassSpec describes one chart's take on the OpenShift agent RuntimeClass check.
type AgentRuntimeClassSpec struct {
	ChartName string
	ChartPath string
	// SetValues must be enough for the chart to render with OpenShift.enabled and at least one
	// agent runtime turned on - that is what makes the check run at all.
	SetValues map[string]string
}

// RunAgentRuntimeClassCheckTest proves that OpenShift.agentRuntimeClassName has to name a
// RuntimeClass the cluster actually has. Sandboxing the agent runtimes is opt-out, so the name is
// set by default while the chart never creates the object (the OpenShift sandboxed containers
// operator does); without the check Helm reports success and the API server then rejects every
// agent runtime pod with `RuntimeClass "kata" not found`, leaving both Deployments at 0 available.
//
// This cannot be a unit test: the check is a `lookup`, and `lookup` only reaches the API server on
// a real install/upgrade or `--dry-run=server`. `helm template` and client-side `--dry-run` both
// return nothing from it, which the chart deliberately treats as "cannot tell" rather than
// "absent" (see tests/unit-test/openshift_agentic_test.go for that half). `--dry-run=server` is
// what makes the lookup live, so this needs no database and installs nothing.
func RunAgentRuntimeClassCheckTest(t *testing.T, spec AgentRuntimeClassSpec) {
	if os.Getenv(AGENT_RUNTIME_CLASS_E2E) != "1" {
		t.Skipf("set %s=1 to run against a live cluster", AGENT_RUNTIME_CLASS_E2E)
	}

	namespaceName := NamespaceFor(spec.ChartName + "-runtimeclass")
	k8s.CreateNamespace(t, k8s.NewKubectlOptions("", "", "default"), namespaceName)
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	defer DeleteNamespaceAndWait(t, kubectlOptions, namespaceName)

	// The check only trusts an empty RuntimeClass lookup once it has proved lookup is live, and it
	// does that by reading this ServiceAccount. The control plane creates it a moment after the
	// namespace, so waiting here is the difference between testing the check and testing nothing.
	waitForDefaultServiceAccount(t, kubectlOptions)

	// A name no cluster would have, so the failure cannot come from something real.
	missingName := "no-such-runtimeclass-" + namespaceName

	t.Run("missing RuntimeClass fails the install", func(t *testing.T) {
		err := dryRunServer(t, spec, kubectlOptions, map[string]string{
			agentRuntimeClassNameKey: missingName,
		})
		require.Error(t, err, "the install must not succeed with a RuntimeClass that does not exist")
		assert.Contains(t, err.Error(), agentRuntimeClassFail)
		// The message has to point at the operator and the two ways out, not just say "no".
		assert.Contains(t, err.Error(), "sandboxed containers operator")
	})

	t.Run("an existing RuntimeClass passes", func(t *testing.T) {
		// Named after the test rather than "kata": the check must validate whatever the value
		// holds, so passing on a name it cannot have hardcoded is the point.
		existingName := "sonarqube-test-" + namespaceName
		manifest := fmt.Sprintf(`apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: %s
handler: %s
`, existingName, existingName)
		k8s.KubectlApplyFromString(t, kubectlOptions, manifest)
		defer k8s.KubectlDeleteFromString(t, kubectlOptions, manifest)

		err := dryRunServer(t, spec, kubectlOptions, map[string]string{
			agentRuntimeClassNameKey: existingName,
		})
		assert.NoError(t, err)
	})

	t.Run("opting out of sandboxing passes", func(t *testing.T) {
		err := dryRunServer(t, spec, kubectlOptions, map[string]string{
			agentRuntimeClassNameKey: "",
		})
		assert.NoError(t, err, "an empty name is the documented opt-out, not a missing RuntimeClass")
	})

	t.Run("skipAgentRuntimeClassCheck bypasses the check", func(t *testing.T) {
		err := dryRunServer(t, spec, kubectlOptions, map[string]string{
			agentRuntimeClassNameKey:               missingName,
			"OpenShift.skipAgentRuntimeClassCheck": "true",
		})
		assert.NoError(t, err)
	})

	t.Run("the check is scoped to OpenShift", func(t *testing.T) {
		err := dryRunServer(t, spec, kubectlOptions, map[string]string{
			"OpenShift.enabled":      "false",
			agentRuntimeClassNameKey: missingName,
		})
		assert.NoError(t, err, "off OpenShift the name is never applied to a pod")
	})
}

// dryRunServer renders the chart against the live API server - which is what makes the `lookup` in
// the check return real data - without creating anything.
func dryRunServer(t *testing.T, spec AgentRuntimeClassSpec, kubectlOptions *k8s.KubectlOptions, overrides map[string]string) error {
	t.Helper()
	values := map[string]string{TESTS_ENABLING_ACTION: "false"}
	for key, value := range spec.SetValues {
		values[key] = value
	}
	for key, value := range overrides {
		values[key] = value
	}
	options := &helm.Options{
		SetValues:      values,
		KubectlOptions: kubectlOptions,
		ExtraArgs:      map[string][]string{"install": {"--dry-run=server"}},
	}
	// A fresh release name per call: --dry-run=server still refuses a name already in use.
	releaseName := spec.ChartName + "-" + strings.ToLower(random.UniqueId())
	return helm.InstallE(t, options, spec.ChartPath, releaseName)
}

func waitForDefaultServiceAccount(t *testing.T, kubectlOptions *k8s.KubectlOptions) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if _, err := k8s.GetServiceAccountE(t, kubectlOptions, "default"); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("namespace %s never got its default ServiceAccount; the RuntimeClass check "+
				"would skip itself and this test would prove nothing", kubectlOptions.Namespace)
		}
		time.Sleep(2 * time.Second)
	}
}

// AgentRuntimeClassValues is the shared half of both charts' values: OpenShift on, the agent
// runtimes on (what the check is gated on), and the agentic pack's mandatory settings.
//
// hunterAgent alone, deliberately: the check is gated on `or hunterAgent.enabled
// remediationAgent.enabled`, so one runtime family is enough to reach it, while remediationAgent
// additionally requires vortexAnalysis.enabled (validation.yaml, and that rule runs *before* the
// check) - so enabling it here means configuring a whole second subsystem only to render it and
// throw it away.
func AgentRuntimeClassValues() map[string]string {
	return map[string]string{
		"OpenShift.enabled":                   "true",
		"monitoringPasscode":                  "test-passcode",
		"agentOrchestrator.enabled":           "true",
		"agentOrchestrator.image.repository":  "example.com/agent-orchestrator",
		"agentOrchestrator.image.tag":         "42",
		"agentOrchestrator.storage.bucket":    "agent-jobs",
		"hunterAgent.enabled":                 "true",
		"agenticSigningSecret.existingSecret": "test-agentic-instance-secret",
	}
}
