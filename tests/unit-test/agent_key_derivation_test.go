package tests

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// SONAR-32065: one operator-supplied instance secret is expanded by an HKDF pre-install hook into
// one key per communication hop, and each pod gets only the hops it takes part in. The two things
// worth pinning down are therefore (a) that the hook itself is a well-formed, least-privilege hook,
// and (b) that the derived-key distribution matches the design matrix exactly - a consumer mounting
// a key for a hop it isn't on is the whole failure mode this feature exists to prevent.

const (
	keyDerivationHookTemplate = "templates/agent-key-derivation-hook.yaml"
	keyDerivationRBACTemplate = "templates/agent-key-derivation-rbac.yaml"
	// agenticKeyVolumeName is the volume every agent-side consumer and SQS project its derived
	// keys through.
	agenticKeyVolumeName = "agentic-keys"
)

// agenticKeyCharts is the subset of agentCharts that derive and mount agentic signing keys. Every
// test in this file loops over it, so the whole file starts running against the other chart the
// moment its hasAgenticKeys flag flips.
var agenticKeyCharts = func() []agentChart {
	var charts []agentChart
	for _, c := range agentCharts {
		if c.hasAgenticKeys {
			charts = append(charts, c)
		}
	}
	return charts
}()

// renderKeyDerivation renders the hook Job and its RBAC. Errors are returned rather than asserted:
// several tests here are about validation.yaml failing closed.
func renderKeyDerivation(t *testing.T, chart agentChart, fixture string, setValues map[string]string) (string, error) {
	t.Helper()
	return helm.RenderTemplateE(t, agentPropertiesOptions(chart, fixture, setValues), chart.path, chart.release,
		[]string{keyDerivationHookTemplate, keyDerivationRBACTemplate})
}

// keyDerivationJob renders the hook and returns the Job, failing the test if none rendered.
func keyDerivationJob(t *testing.T, chart agentChart, fixture string, setValues map[string]string) batchv1.Job {
	t.Helper()
	output, err := helm.RenderTemplateE(t, agentPropertiesOptions(chart, fixture, setValues), chart.path, chart.release,
		[]string{keyDerivationHookTemplate})
	require.NoError(t, err)
	require.NotEmpty(t, strings.TrimSpace(output), "the key-derivation hook Job did not render")

	var job batchv1.Job
	helm.UnmarshalK8SYaml(t, output, &job)
	return job
}

// deriveScript is the shell script the hook Job runs - the only place the derived labels and the
// per-consumer Secret writes are observable, since both are computed at render time and then
// embedded in the container command.
func deriveScript(t *testing.T, job batchv1.Job) string {
	t.Helper()
	require.Len(t, job.Spec.Template.Spec.Containers, 1)
	command := job.Spec.Template.Spec.Containers[0].Command
	require.Len(t, command, 3, "expected /bin/sh -c <script>")
	return command[2]
}

// derivedLabels are the labels passed to derive-keys.sh, i.e. the union every enabled consumer
// needs. Parsed out of the script's `--label <name>="$KEYS/<name>"` flags.
func derivedLabels(t *testing.T, job batchv1.Job) []string {
	t.Helper()
	var labels []string
	for _, line := range strings.Split(deriveScript(t, job), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "--label ") {
			continue
		}
		label, _, found := strings.Cut(strings.TrimPrefix(line, "--label "), "=")
		require.True(t, found, "malformed --label flag: %q", line)
		labels = append(labels, label)
	}
	return labels
}

// writtenSecrets maps each Secret the hook writes to the key labels it puts in it, parsed out of
// the script's `write_secret <name> <label>...` calls. This is the derived-key distribution as the
// cluster will actually see it.
func writtenSecrets(t *testing.T, job batchv1.Job) map[string][]string {
	t.Helper()
	written := map[string][]string{}
	for _, line := range strings.Split(deriveScript(t, job), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		// Skip the `write_secret() {` definition; only calls have a name argument.
		if len(fields) < 2 || fields[0] != "write_secret" {
			continue
		}
		written[fields[1]] = fields[2:]
	}
	return written
}

func (c agentChart) keySecretName(consumer string) string {
	return c.fullnamePrefix() + "-agentic-keys-" + consumer
}

// renderedKinds lists the kind of every document in a render, in order. Needed instead of a
// substring check because "kind: ServiceAccount" also appears inside a RoleBinding's subjects.
func renderedKinds(t *testing.T, output string) []string {
	t.Helper()
	var kinds []string
	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var meta struct {
			Kind string `yaml:"kind"`
		}
		helm.UnmarshalK8SYaml(t, doc, &meta)
		kinds = append(kinds, meta.Kind)
	}
	return kinds
}

// The hook exists only when there is something to derive keys for. Nothing agentic enabled means
// no Job, no ServiceAccount, and no RBAC - an install without the agentic pack must not grow a
// Role over Secrets.
func TestAgentKeyDerivationRendersOnlyWhenAgenticEnabled(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			// A template that renders empty is reported by Helm as an error rather than an empty
			// success - the same convention the other agentic no-render tests rely on.
			output, err := renderKeyDerivation(t, chart, "agent-all-disabled.yaml", nil)
			require.Error(t, err, "nothing agentic is enabled, so nothing should render")
			assert.Empty(t, strings.TrimSpace(output))

			output, err = renderKeyDerivation(t, chart, "agent-runtimes-enabled.yaml", nil)
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{"Job", "ServiceAccount", "Role", "RoleBinding"}, renderedKinds(t, output))
		})
	}
}

// The Job has to run before the workloads that mount its Secrets, and it has to be seen to
// terminate. Both are annotation-level properties with no runtime feedback if wrong - a
// post-install hook would deadlock against --wait, and an Istio sidecar would keep the pod alive
// forever - so they get pinned explicitly.
func TestAgentKeyDerivationIsAPreInstallHook(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			job := keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", nil)

			assert.Equal(t, chart.fullnamePrefix()+"-agent-key-derivation", job.Name)
			assert.Equal(t, "pre-install, pre-upgrade", job.Annotations["helm.sh/hook"])
			assert.Equal(t, "0", job.Annotations["helm.sh/hook-weight"])
			assert.Equal(t, "before-hook-creation, hook-succeeded", job.Annotations["helm.sh/hook-delete-policy"])

			podSpec := job.Spec.Template.Spec
			assert.Equal(t, corev1.RestartPolicyOnFailure, podSpec.RestartPolicy)
			assert.Equal(t, "false", job.Spec.Template.Annotations["sidecar.istio.io/inject"],
				"an Istio sidecar would keep the pod running and the hook would never succeed")
			require.NotNil(t, job.Spec.BackoffLimit)
			assert.EqualValues(t, 1, *job.Spec.BackoffLimit)
			require.NotNil(t, job.Spec.ActiveDeadlineSeconds, "the hook must not be able to hang a release forever")
			assert.Positive(t, *job.Spec.ActiveDeadlineSeconds)
		})
	}
}

// The RBAC has to land before the Job (lower hook weight) and must not be torn down while the Job
// is still using it - hook-succeeded on the ServiceAccount would race the pod it authenticates.
func TestAgentKeyDerivationRBACPrecedesTheJob(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderKeyDerivation(t, chart, "agent-runtimes-enabled.yaml", nil)
			require.NoError(t, err)

			var kinds []string
			for _, doc := range strings.Split(output, "\n---") {
				if strings.TrimSpace(doc) == "" {
					continue
				}
				var meta struct {
					Kind     string `yaml:"kind"`
					Metadata struct {
						Annotations map[string]string `yaml:"annotations"`
					} `yaml:"metadata"`
				}
				helm.UnmarshalK8SYaml(t, doc, &meta)
				if meta.Kind == "Job" {
					continue
				}
				kinds = append(kinds, meta.Kind)
				assert.Equal(t, "-5", meta.Metadata.Annotations["helm.sh/hook-weight"], "%s must apply before the Job", meta.Kind)
				assert.Equal(t, "before-hook-creation", meta.Metadata.Annotations["helm.sh/hook-delete-policy"],
					"%s must outlive a succeeding Job", meta.Kind)
			}
			assert.ElementsMatch(t, []string{"ServiceAccount", "Role", "RoleBinding"}, kinds)
		})
	}
}

// The Role is the blast radius of the hook. It upserts Secrets by exact name and never enumerates,
// so it must not have list/watch/delete or a bare get - and it must stay a namespaced Role, never
// a ClusterRole. create can't be scoped by name (the Secret doesn't exist yet on install), but
// update must be pinned to exactly the Secrets the enabled consumers actually write - anything
// wider hands this Role, which survives a helm uninstall, standing write access to every other
// Secret in the namespace.
func TestAgentKeyDerivationRoleIsLeastPrivilege(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderKeyDerivation(t, chart, "agent-runtimes-enabled.yaml", nil)
			require.NoError(t, err)
			assert.NotContains(t, output, "kind: ClusterRole", "the hook only ever writes in the release namespace")

			var role rbacv1.Role
			var found bool
			for _, doc := range strings.Split(output, "\n---") {
				if !strings.Contains(doc, "kind: Role\n") {
					continue
				}
				helm.UnmarshalK8SYaml(t, doc, &role)
				found = true
			}
			require.True(t, found, "no Role rendered")

			require.Len(t, role.Rules, 2)

			createRule := role.Rules[0]
			assert.Equal(t, []string{""}, createRule.APIGroups)
			assert.Equal(t, []string{"secrets"}, createRule.Resources)
			assert.Equal(t, []string{"create"}, createRule.Verbs)
			assert.Empty(t, createRule.ResourceNames, "create can't be scoped by name, but must not be widened to other verbs")

			updateRule := role.Rules[1]
			assert.Equal(t, []string{""}, updateRule.APIGroups)
			assert.Equal(t, []string{"secrets"}, updateRule.Resources)
			assert.Equal(t, []string{"update"}, updateRule.Verbs)
			assert.ElementsMatch(t, []string{
				chart.keySecretName("orchestrator"),
				chart.keySecretName("hunter"),
				chart.keySecretName("remediation"),
				chart.keySecretName("sqs"),
				chart.keySecretName("vortex"),
			}, updateRule.ResourceNames)
		})
	}
}

// A partial agentic install must only be able to update the Secrets it actually writes - the
// resourceNames list has to track the enabled-consumer set exactly, not the full five-consumer
// matrix.
func TestAgentKeyDerivationRoleResourceNamesTrackEnabledConsumers(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderKeyDerivation(t, chart, "gvisor-hunter-only.yaml", nil)
			require.NoError(t, err)

			var role rbacv1.Role
			var found bool
			for _, doc := range strings.Split(output, "\n---") {
				if !strings.Contains(doc, "kind: Role\n") {
					continue
				}
				helm.UnmarshalK8SYaml(t, doc, &role)
				found = true
			}
			require.True(t, found, "no Role rendered")

			require.Len(t, role.Rules, 2)
			assert.ElementsMatch(t, []string{
				chart.keySecretName("orchestrator"),
				chart.keySecretName("hunter"),
				chart.keySecretName("sqs"),
			}, role.Rules[1].ResourceNames, "no remediation or vortex Secret in this fixture")
		})
	}
}

// The distribution matrix from the design (EA-791/ADR-10): each hop's key goes to exactly the two
// components on that hop, and the shared key only to the components that use it. Any extra entry
// here is a component holding a key it has no business being able to sign with.
func TestAgentKeyDerivationDistributesKeysPerHop(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			cases := []struct {
				name    string
				fixture string
				// want maps consumer name to the labels its Secret must hold, exactly.
				want map[string][]string
			}{
				{
					name:    "every component enabled",
					fixture: "agent-runtimes-enabled.yaml",
					want: map[string][]string{
						"orchestrator": {"orchestrator-to-hunter", "orchestrator-to-remediation", "agentic-shared"},
						"hunter":       {"orchestrator-to-hunter"},
						"remediation":  {"orchestrator-to-remediation", "remediation-to-sqs"},
						// SQS verifies the remediation-to-sqs hop but does not mount its key: it
						// holds the instance secret and re-derives that one in-process.
						"sqs":    {"agentic-shared"},
						"vortex": {"agentic-shared"},
					},
				},
				{
					// Hunter never talks to SQS, so remediation-to-sqs must not be derived at all
					// here - and with no Vortex there is no Vortex Secret to write.
					name:    "hunter only",
					fixture: "gvisor-hunter-only.yaml",
					want: map[string][]string{
						"orchestrator": {"orchestrator-to-hunter", "agentic-shared"},
						"hunter":       {"orchestrator-to-hunter"},
						"sqs":          {"agentic-shared"},
					},
				},
				{
					// Vortex is deployable without the orchestrator, and then only the shared key
					// exists - notably no orchestrator Secret, since there is no orchestrator pod
					// to mount one.
					name:    "vortex only",
					fixture: "vortex-enabled.yaml",
					want: map[string][]string{
						"sqs":    {"agentic-shared"},
						"vortex": {"agentic-shared"},
					},
				},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					job := keyDerivationJob(t, chart, tc.fixture, nil)

					want := map[string][]string{}
					var wantLabels []string
					for consumer, labels := range tc.want {
						want[chart.keySecretName(consumer)] = labels
						wantLabels = append(wantLabels, labels...)
					}
					assert.Equal(t, want, writtenSecrets(t, job))

					// Deriving a label nothing mounts is wasted work; missing one is a pod that
					// can't start, so the union must match the distribution exactly.
					assert.ElementsMatch(t, uniqueStrings(wantLabels), derivedLabels(t, job))
				})
			}
		})
	}
}

// The per-consumer suffix is what makes the five Secret names distinct, so it has to survive the
// 63-character limit that a long release name pushes them against. If it didn't, the hook's five
// write_secret calls would overwrite one another and every pod's items: projection would ask the
// surviving Secret for labels it doesn't hold - ContainerCreating across the whole agentic pack.
func TestAgenticKeySecretNamesSurviveALongReleaseName(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			// Long enough that truncating the composed name instead of the prefix would collapse
			// the consumers, but still inside what Helm accepts as a release name.
			longRelease := "sonarqube-agentic-release-with-a-long-name"

			output, err := helm.RenderTemplateE(t, agentPropertiesOptions(chart, "agent-runtimes-enabled.yaml", nil),
				chart.path, longRelease, []string{keyDerivationHookTemplate})
			require.NoError(t, err)
			var job batchv1.Job
			helm.UnmarshalK8SYaml(t, output, &job)

			var consumers []string
			for name := range writtenSecrets(t, job) {
				assert.LessOrEqual(t, len(name), 63, "%s is not a valid Secret name", name)
				_, consumer, found := strings.Cut(name, "-agentic-keys-")
				require.True(t, found, "%s lost its -agentic-keys- infix", name)
				consumers = append(consumers, consumer)
			}
			assert.ElementsMatch(t, []string{"orchestrator", "hunter", "remediation", "sqs", "vortex"}, consumers)
		})
	}
}

// sonarqube.agentKeyDerivation.fullname appends its suffix to sonarqube.fullname and only then
// truncates to 63 - so once sonarqube.fullname itself already sits at the cap, appending and
// re-truncating must not collapse right back onto it. If it did, the hook's Job/ServiceAccount/
// Role/RoleBinding would collide with the chart's own primary resources instead of just with each
// other.
func TestAgentKeyDerivationFullnameSurvivesALongReleaseName(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			// Chosen so release name + "-" + chart name lands exactly on sonarqube.fullname's
			// 63-character cap for this chart.
			longRelease := strings.Repeat("x", 63-len(chart.name)-1)
			mainFullname := longRelease + "-" + chart.name
			require.Len(t, mainFullname, 63)

			output, err := helm.RenderTemplateE(t, agentPropertiesOptions(chart, "agent-runtimes-enabled.yaml", nil),
				chart.path, longRelease, []string{keyDerivationHookTemplate})
			require.NoError(t, err)
			var job batchv1.Job
			helm.UnmarshalK8SYaml(t, output, &job)

			assert.LessOrEqual(t, len(job.Name), 63, "%s is not a valid resource name", job.Name)
			assert.NotEqual(t, mainFullname, job.Name,
				"the hook's name must not collapse onto the chart's own fullname")
			assert.True(t, strings.HasSuffix(job.Name, "-agent-key-derivation"),
				"truncating the prefix must not eat the suffix that makes this name distinct: got %q", job.Name)
		})
	}
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// derive-keys.sh ships inside the Agent Orchestrator image, so an orchestrator install needs no
// image configuration of its own - but a Vortex-only install has no orchestrator to borrow from
// and must be able to name an image without also pulling and deploying an orchestrator.
func TestAgentKeyDerivationImage(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("defaults to the orchestrator image", func(t *testing.T) {
				job := keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", nil)
				assert.Equal(t, "example.com/agent-orchestrator:42", job.Spec.Template.Spec.Containers[0].Image)
			})

			t.Run("own image wins", func(t *testing.T) {
				job := keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", map[string]string{
					"agentKeyDerivation.image.repository": "example.com/derive",
					"agentKeyDerivation.image.tag":        "7",
					"agentKeyDerivation.image.pullPolicy": "Always",
				})
				container := job.Spec.Template.Spec.Containers[0]
				assert.Equal(t, "example.com/derive:7", container.Image)
				assert.Equal(t, corev1.PullAlways, container.ImagePullPolicy)
			})

			t.Run("vortex without an orchestrator uses its own image", func(t *testing.T) {
				job := keyDerivationJob(t, chart, "vortex-enabled.yaml", nil)
				assert.Equal(t, "example.com/agent-orchestrator:42", job.Spec.Template.Spec.Containers[0].Image,
					"the vortex fixture sets agentKeyDerivation.image, not agentOrchestrator.image")
			})

			// Fail closed: keys that never get derived would surface as pods stuck on a missing
			// Secret, or - worse, on upgrade - as silently stale keys.
			t.Run("fails closed with no image anywhere", func(t *testing.T) {
				_, err := renderKeyDerivation(t, chart, "vortex-enabled.yaml", map[string]string{
					"agentKeyDerivation.image.repository": "",
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "derive-keys.sh")
			})

			// The fallback is all-or-nothing, so an override repository without a tag renders
			// "repo:" - an invalid reference no node can pull. Left unchecked the hook would hang
			// until activeDeadlineSeconds and fail the release with a timeout instead of this.
			t.Run("fails closed on an override repository with no tag", func(t *testing.T) {
				_, err := renderKeyDerivation(t, chart, "vortex-enabled.yaml", map[string]string{
					"agentKeyDerivation.image.tag": "",
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "agentKeyDerivation.image.tag")
			})

			// The orchestrator fallback is a different case: whether that image carries a tag is
			// the orchestrator's own concern, and fixtures legitimately leave it unset.
			t.Run("no tag on the fallback image is not our business", func(t *testing.T) {
				job := keyDerivationJob(t, chart, "gvisor-hunter-only.yaml", nil)
				assert.Equal(t, "example.com/agent-orchestrator:", job.Spec.Template.Spec.Containers[0].Image)
			})
		})
	}
}

// Enabling anything agentic without an instance secret has no safe interpretation: there is
// nothing to derive from, so the components would start unable to sign or verify anything.
func TestAgenticSigningRequiresAnInstanceSecret(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			_, err := renderKeyDerivation(t, chart, "agent-runtimes-enabled.yaml", map[string]string{
				"agenticSigningSecret.existingSecret": "",
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "agenticSigningSecret.existingSecret")
		})
	}
}

// The escape hatch for operators who derive and distribute the keys themselves (external secret
// managers, stricter RBAC policies): no Job and no Role, but the consumers still mount the Secrets,
// since the operator is expected to have provided them under the same names.
func TestAgentKeyDerivationDisabledStillMountsKeys(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			setValues := map[string]string{"agentKeyDerivation.enabled": "false"}

			output, err := renderKeyDerivation(t, chart, "agent-runtimes-enabled.yaml", setValues)
			require.Error(t, err, "the operator owns key derivation, so the chart must not run it")
			assert.Empty(t, strings.TrimSpace(output))

			podSpec := agenticPodSpec(t, chart, "templates/agent-orchestrator.yaml", "agent-runtimes-enabled.yaml", setValues, "")
			assert.Contains(t, agentVolumesByName(podSpec.Volumes), agenticKeyVolumeName,
				"the keys are still expected to exist, just not to be created by us")
		})
	}
}

// The hook needs a token that can write Secrets, which the release's own ServiceAccount generally
// isn't allowed to do - hence a dedicated one by default. serviceAccount.create=false hands that
// choice back to the operator, and must then not render an SA of its own.
func TestAgentKeyDerivationServiceAccount(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			t.Run("dedicated by default", func(t *testing.T) {
				job := keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", nil)
				assert.Equal(t, chart.fullnamePrefix()+"-agent-key-derivation", job.Spec.Template.Spec.ServiceAccountName)
			})

			t.Run("create=false reuses the named release ServiceAccount", func(t *testing.T) {
				setValues := map[string]string{
					"agentKeyDerivation.serviceAccount.create": "false",
					"serviceAccount.name":                      "sonarqube-release",
				}
				output, err := renderKeyDerivation(t, chart, "agent-runtimes-enabled.yaml", setValues)
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"Job", "Role", "RoleBinding"}, renderedKinds(t, output),
					"no ServiceAccount of our own, but the reused account still has to be granted the Role")

				job := keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", setValues)
				assert.Equal(t, "sonarqube-release", job.Spec.Template.Spec.ServiceAccountName)
			})

			// The name override is what the create=false failure message tells operators to
			// set, so it has to be read on that path too - otherwise following the message
			// returns the identical failure.
			t.Run("create=false honours agentKeyDerivation.serviceAccount.name", func(t *testing.T) {
				setValues := map[string]string{
					"agentKeyDerivation.serviceAccount.create": "false",
					"agentKeyDerivation.serviceAccount.name":   "operator-bound-sa",
					// Deliberately different, to pin down which of the two wins.
					"serviceAccount.name": "sonarqube-release",
				}
				output, err := renderKeyDerivation(t, chart, "agent-runtimes-enabled.yaml", setValues)
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"Job", "Role", "RoleBinding"}, renderedKinds(t, output),
					"the operator owns the account; we only bind the Role to it")

				job := keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", setValues)
				assert.Equal(t, "operator-bound-sa", job.Spec.Template.Spec.ServiceAccountName)
			})

			// The Role grants get/create/update on every Secret in the namespace. Binding that to
			// "default" - which is what the reuse resolves to when the release names no account -
			// would hand Secret-write access to every unrelated pod running as default.
			t.Run("create=false without a name fails closed", func(t *testing.T) {
				_, err := renderKeyDerivation(t, chart, "agent-runtimes-enabled.yaml", map[string]string{
					"agentKeyDerivation.serviceAccount.create": "false",
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "'default' ServiceAccount")
			})

			// The script reads the projected token under `set -eu`, so a pod without one fails the
			// hook and takes the release with it. The reused account's automountToken is the
			// operator's business and defaults to false, hence the pod-level override.
			t.Run("forces token projection on the pod", func(t *testing.T) {
				for name, setValues := range map[string]map[string]string{
					"dedicated": nil,
					"reused": {
						"agentKeyDerivation.serviceAccount.create": "false",
						"serviceAccount.name":                      "sonarqube-release",
					},
				} {
					t.Run(name, func(t *testing.T) {
						job := keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", setValues)
						automount := job.Spec.Template.Spec.AutomountServiceAccountToken
						require.NotNil(t, automount, "the pod must not inherit the account's setting")
						assert.True(t, *automount)
					})
				}
			})
		})
	}
}

// The instance secret is mounted, never passed as an env var or an argument: env vars leak through
// `kubectl describe`, pod specs and crash dumps, and the root secret is the one value from which
// every derived key can be recomputed.
func TestAgentKeyDerivationMountsTheInstanceSecret(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			job := keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", nil)
			podSpec := job.Spec.Template.Spec

			volume, ok := agentVolumesByName(podSpec.Volumes)["instance-secret"]
			require.True(t, ok, "the instance secret must reach the Job as a volume")
			require.NotNil(t, volume.Secret)
			assert.Equal(t, "test-agentic-instance-secret", volume.Secret.SecretName)

			mount, ok := agentVolumeMountsByName(podSpec.Containers[0].VolumeMounts)["instance-secret"]
			require.True(t, ok)
			assert.True(t, mount.ReadOnly)

			for _, env := range podSpec.Containers[0].Env {
				assert.NotContains(t, strings.ToLower(env.Name), "secret",
					"the instance secret must not be exposed through the environment")
			}
		})
	}
}

// agenticPodSpec renders one template and returns the pod spec of the workload it contains. When
// familyLabel is set, the workload is picked by its sonarqube.agent/family label - templates/
// agent-runtime.yaml renders one Deployment per runtime family.
func agenticPodSpec(t *testing.T, chart agentChart, template, fixture string, setValues map[string]string, familyLabel string) corev1.PodSpec {
	t.Helper()
	output, err := helm.RenderTemplateE(t, agentPropertiesOptions(chart, fixture, setValues), chart.path, chart.release, []string{template})
	require.NoError(t, err)

	for _, doc := range strings.Split(output, "\n---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		// StatefulSet and Deployment differ only in fields this helper doesn't touch.
		var workload appsv1.Deployment
		helm.UnmarshalK8SYaml(t, doc, &workload)
		if familyLabel != "" && workload.Labels["sonarqube.agent/family"] != familyLabel {
			continue
		}
		return workload.Spec.Template.Spec
	}

	t.Fatalf("%s rendered no workload (family %q)", template, familyLabel)
	return corev1.PodSpec{}
}

// The other half of the matrix: what each pod actually mounts. The Secret name pins which consumer
// the pod is, and the projected items pin which hops it can sign for - a pod projecting a key for
// a hop it isn't on would be able to forge messages on that hop.
func TestAgenticKeysMountedPerConsumer(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			cases := []struct {
				consumer    string
				template    string
				familyLabel string
				mountPath   string
				labels      []string
			}{
				{consumer: "orchestrator", template: "templates/agent-orchestrator.yaml", mountPath: "/etc/agentic/keys",
					labels: []string{"orchestrator-to-hunter", "orchestrator-to-remediation", "agentic-shared"}},
				{consumer: "hunter", template: "templates/agent-runtime.yaml", familyLabel: "hunter", mountPath: "/etc/agentic/keys",
					labels: []string{"orchestrator-to-hunter"}},
				{consumer: "remediation", template: "templates/agent-runtime.yaml", familyLabel: "remediation", mountPath: "/etc/agentic/keys",
					labels: []string{"orchestrator-to-remediation", "remediation-to-sqs"}},
				{consumer: "vortex", template: "templates/vortex.yaml", mountPath: "/etc/agentic/keys",
					labels: []string{"agentic-shared"}},
				{consumer: "sqs", template: chart.appTemplate, mountPath: "/opt/sonarqube/agentic-keys",
					labels: []string{"agentic-shared"}},
			}
			for _, tc := range cases {
				t.Run(tc.consumer, func(t *testing.T) {
					podSpec := agenticPodSpec(t, chart, tc.template, "agent-runtimes-enabled.yaml", nil, tc.familyLabel)

					volume, ok := agentVolumesByName(podSpec.Volumes)[agenticKeyVolumeName]
					require.True(t, ok, "%s does not mount its derived keys", tc.consumer)
					require.NotNil(t, volume.Secret)
					assert.Equal(t, chart.keySecretName(tc.consumer), volume.Secret.SecretName)

					var projected []string
					for _, item := range volume.Secret.Items {
						assert.Equal(t, item.Key, item.Path, "the file name must match the label the key was derived under")
						projected = append(projected, item.Key)
					}
					assert.ElementsMatch(t, tc.labels, projected, "%s must project exactly the hops it is on", tc.consumer)

					mount, ok := agentVolumeMountsByName(podSpec.Containers[0].VolumeMounts)[agenticKeyVolumeName]
					require.True(t, ok)
					assert.Equal(t, tc.mountPath, mount.MountPath)
					assert.True(t, mount.ReadOnly, "signing keys must never be writable by their consumer")
				})
			}
		})
	}
}

// Components that mount no keys must not mount an empty agentic-keys volume either: a Secret named
// after a consumer that has no hops is never written, so the pod would hang on a missing Secret.
func TestAgenticKeysNotMountedWithoutHops(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			// Vortex on its own: no orchestrator, no runtimes, so the SQS <-> remediation hop
			// doesn't exist and the orchestrator has no keys of its own.
			podSpec := agenticPodSpec(t, chart, chart.appTemplate, "vortex-enabled.yaml", nil, "")
			volume, ok := agentVolumesByName(podSpec.Volumes)[agenticKeyVolumeName]
			require.True(t, ok, "SQS still shares the agentic-shared key with Vortex")
			require.NotNil(t, volume.Secret)
			require.Len(t, volume.Secret.Items, 1)
			assert.Equal(t, "agentic-shared", volume.Secret.Items[0].Key)

			// Nothing agentic at all: no key material anywhere on the SQS pod.
			podSpec = agenticPodSpec(t, chart, chart.appTemplate, "agent-all-disabled.yaml", nil, "")
			assert.NotContains(t, agentVolumesByName(podSpec.Volumes), agenticKeyVolumeName)
			assert.NotContains(t, agentVolumesByName(podSpec.Volumes), "agentic-instance-secret")
		})
	}
}

// SQS reads both signing settings through conf/sonar.properties. secretFile also has a redundant
// env var carrying the same value (AGENTIC_SIGNING_SECRET_FILE), but the properties file is what
// actually takes effect - Configuration.get() doesn't fall back to plain env vars (SONAR-31416).
// signingKeyPath has no env-var companion at all (see sonarqube.agentic.keyPathEnv); the property
// is its only path in. secretFile is what SQS verifies incoming calls with; signingKeyPath is what
// it signs its own calls to the orchestrator with (SONAR-31996), and unset leaves those unsigned
// rather than failing.
func TestAgenticSigningPathsInSonarProperties(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-runtimes-enabled.yaml", nil))
			assert.Equal(t, "/opt/sonarqube/agentic-secret/instance-secret", props["sonar.agentic.signing.secretFile"])
			assert.Equal(t, "/opt/sonarqube/agentic-keys/agentic-shared", props["sonar.agentic.orchestrator.signingKeyPath"])

			props = appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-all-disabled.yaml", nil))
			assert.NotContains(t, props, "sonar.agentic.signing.secretFile")
			assert.NotContains(t, props, "sonar.agentic.orchestrator.signingKeyPath")
		})
	}
}

// Both properties are absolute paths assembled from helpers that are separate from the ones wiring
// the mounts, so nothing but this stops them drifting apart: the properties file is generated by
// sonarqube.agentHealthProperties, the mounts by keyVolumeMount/the instance-secret block. Drift
// points the JVM at a path that isn't in the container, and the only symptom is at runtime - an
// ERROR on startup for secretFile, an unsigned first orchestrator call for signingKeyPath. Each
// property is therefore resolved against what the pod actually projects; secretFile is additionally
// cross-checked against the redundant env var that carries the same value, since signingKeyPath has
// no such env var to cross-check against (see sonarqube.agentic.keyPathEnv).
func TestAgenticSigningPathsResolveToProjectedFiles(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			props := appSonarProperties(t, chart, agentPropertiesOptions(chart, "agent-runtimes-enabled.yaml", nil))
			podSpec := agenticPodSpec(t, chart, chart.appTemplate, "agent-runtimes-enabled.yaml", nil, "")
			container := podSpec.Containers[0]

			env := map[string]string{}
			for _, e := range container.Env {
				env[e.Name] = e.Value
			}
			mounts := agentVolumeMountsByName(container.VolumeMounts)
			volumes := agentVolumesByName(podSpec.Volumes)

			t.Run("sonar.agentic.signing.secretFile", func(t *testing.T) {
				property, envName := "sonar.agentic.signing.secretFile", "AGENTIC_SIGNING_SECRET_FILE"
				path := props[property]
				require.NotEmpty(t, path, "%s is not in conf/sonar.properties", property)
				assert.Equal(t, path, env[envName],
					"the redundant env var and the property that takes effect must name the same file")

				assertPathIsProjected(t, property, path, mounts, volumes)
			})

			t.Run("sonar.agentic.orchestrator.signingKeyPath", func(t *testing.T) {
				property := "sonar.agentic.orchestrator.signingKeyPath"
				path := props[property]
				require.NotEmpty(t, path, "%s is not in conf/sonar.properties", property)
				assert.NotContains(t, env, "AGENTIC_ORCHESTRATOR_SIGNING_KEY_PATH",
					"SQS must read this path only through the property, not a plain env var (SONAR-31416)")

				assertPathIsProjected(t, property, path, mounts, volumes)
			})
		})
	}
}

// assertPathIsProjected resolves an absolute in-container path against what the pod actually
// projects: the directory has to be a read-only mount, and the Secret behind it has to carry an
// item of that name - a Secret volume only surfaces the items it lists.
func assertPathIsProjected(t *testing.T, what, path string, mounts map[string]corev1.VolumeMount, volumes map[string]corev1.Volume) {
	t.Helper()

	cut := strings.LastIndex(path, "/")
	require.Positive(t, cut, "%s must be a file inside a mount, not a bare path", what)
	dir, file := path[:cut], path[cut+1:]

	var mountName string
	for name, mount := range mounts {
		if mount.MountPath == dir {
			mountName = name
		}
	}
	require.NotEmpty(t, mountName, "%s points into %s, which the pod does not mount", what, dir)
	assert.True(t, mounts[mountName].ReadOnly, "%s must be mounted read-only", mountName)

	volume, ok := volumes[mountName]
	require.True(t, ok)
	require.NotNil(t, volume.Secret, "%s must come from a Secret", mountName)
	var projected []string
	for _, item := range volume.Secret.Items {
		projected = append(projected, item.Path)
	}
	assert.Contains(t, projected, file, "%s names %s, which %s does not project", what, path, mountName)
}

// Each consumer is pointed at its keys one env var per key *file*, and the same key is named
// differently on either side of a hop - the name follows the consumer's role, not the key. Every
// such variable has to resolve to a file the pod actually projects, or the consumer opens a path
// that isn't there.
//
// AGENTIC_VERIFY_KEY_PATH appearing on both runtimes is not a collision: each holds exactly one
// verification key.
func TestAgenticKeyPathEnvPointsAtProjectedFiles(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, tc := range agenticKeyEnvCases(chart.appTemplate) {
				t.Run(tc.name, func(t *testing.T) {
					assertKeyPathEnvResolves(t, chart, tc)
				})
			}
		})
	}
}

// agenticKeyEnvCase is one consumer's expected view of the derived keys.
type agenticKeyEnvCase struct {
	name        string
	template    string
	familyLabel string
	// want maps the env var the consumer reads to the label it must resolve to.
	want map[string]string
	// wantIDs maps the companion *_KEY_ID vars to the label they must name. Unlike want, the
	// value is the bare label, not a path under the mount.
	wantIDs map[string]string
}

func agenticKeyEnvCases(appTemplate string) []agenticKeyEnvCase {
	return []agenticKeyEnvCase{
		{name: "orchestrator", template: "templates/agent-orchestrator.yaml", want: map[string]string{
			"AGENTIC_HUNTER_RUNTIME_SIGNING_KEY_PATH":      "orchestrator-to-hunter",
			"AGENTIC_REMEDIATION_RUNTIME_SIGNING_KEY_PATH": "orchestrator-to-remediation",
			"AGENTIC_SONARQUBE_SIGNING_KEY_PATH":           "agentic-shared",
		}},
		{name: "hunter", template: "templates/agent-runtime.yaml", familyLabel: "hunter", want: map[string]string{
			"AGENTIC_VERIFY_KEY_PATH": "orchestrator-to-hunter",
		}, wantIDs: map[string]string{
			"AGENTIC_VERIFY_KEY_ID": "orchestrator-to-hunter",
		}},
		{name: "remediation", template: "templates/agent-runtime.yaml", familyLabel: "remediation", want: map[string]string{
			"AGENTIC_VERIFY_KEY_PATH":              "orchestrator-to-remediation",
			"REMEDIATION_AGENTIC_SIGNING_KEY_PATH": "remediation-to-sqs",
		}, wantIDs: map[string]string{
			"AGENTIC_VERIFY_KEY_ID": "orchestrator-to-remediation",
		}},
		// SQS mounts agentic-shared (see keyLabels) but gets no path env var for it: it reads the
		// path via the sonar.agentic.orchestrator.signingKeyPath property instead. Kept as an
		// explicit case, with an empty want, so a regression that reintroduces the env var fails
		// here rather than going unnoticed.
		{name: "sqs", template: appTemplate, want: map[string]string{}},
		// Vortex has no properties mechanism (it's a standalone Deployment, not a SonarQube JVM
		// process), so it's the only consumer that still reads this key through an env var. No
		// _KEY_ID companion: it uses the key for one thing, signing its own outbound calls, so the
		// name is unambiguous.
		{name: "vortex", template: "templates/vortex.yaml", want: map[string]string{
			"AGENTIC_ORCHESTRATOR_SIGNING_KEY_PATH": "agentic-shared",
		}},
	}
}

func assertKeyPathEnvResolves(t *testing.T, chart agentChart, tc agenticKeyEnvCase) {
	t.Helper()

	podSpec := agenticPodSpec(t, chart, tc.template, "agent-runtimes-enabled.yaml", nil, tc.familyLabel)
	container := podSpec.Containers[0]

	mount, ok := agentVolumeMountsByName(container.VolumeMounts)[agenticKeyVolumeName]
	require.True(t, ok, "%s does not mount its derived keys", tc.name)

	projected := map[string]bool{}
	for _, item := range agentVolumesByName(podSpec.Volumes)[agenticKeyVolumeName].Secret.Items {
		projected[item.Path] = true
	}

	got, gotIDs := keyEnvByKind(t, container.Env, mount.MountPath)

	want := map[string]string{}
	for name, label := range tc.want {
		want[name] = mount.MountPath + "/" + label
		assert.True(t, projected[label], "%s resolves to %s, which this pod does not project", name, label)
	}
	assert.Equal(t, want, got)

	wantIDs := tc.wantIDs
	if wantIDs == nil {
		wantIDs = map[string]string{}
	}
	for name, label := range wantIDs {
		assert.True(t, projected[label], "%s names %s, which this pod does not project", name, label)
	}
	assert.Equal(t, wantIDs, gotIDs)
}

// keyEnvByKind splits a container's environment into the signing-key path vars and the companion
// key-ID vars, and asserts along the way that no path var names the mount directory itself.
func keyEnvByKind(t *testing.T, env []corev1.EnvVar, mountPath string) (paths, ids map[string]string) {
	t.Helper()

	paths, ids = map[string]string{}, map[string]string{}
	for _, e := range env {
		switch {
		// AGENTIC_SECRET_KEY_PATH is the chart-wide encryption key (sonarSecretKey), not a
		// signing key - it happens to share the suffix.
		case strings.HasSuffix(e.Name, "_KEY_PATH") && e.Name != "AGENTIC_SECRET_KEY_PATH":
			paths[e.Name] = e.Value
			// A directory variable would silently pass the caller's checks; the contract is one
			// variable per file, so guard the shape too.
			assert.NotEqual(t, mountPath, e.Value, "%s points at the key directory, not a key file", e.Name)
		case strings.HasSuffix(e.Name, "_KEY_ID"):
			ids[e.Name] = e.Value
		}
	}
	return paths, ids
}

// Regression guard on the least-privilege property itself, stated once as an invariant rather than
// per fixture: no component may ever hold a key for a hop it is not an endpoint of.
func TestAgenticKeysNeverCrossHops(t *testing.T) {
	// hopEndpoints lists, for each derived label, the only consumers allowed to hold it.
	hopEndpoints := map[string][]string{
		"orchestrator-to-hunter":      {"orchestrator", "hunter"},
		"orchestrator-to-remediation": {"orchestrator", "remediation"},
		"remediation-to-sqs":          {"remediation"},
		"agentic-shared":              {"orchestrator", "sqs", "vortex"},
	}
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, fixture := range []string{"agent-runtimes-enabled.yaml", "gvisor-hunter-only.yaml", "gvisor-remediation-only.yaml", "vortex-enabled.yaml"} {
				t.Run(fixture, func(t *testing.T) {
					job := keyDerivationJob(t, chart, fixture, nil)
					for secretName, labels := range writtenSecrets(t, job) {
						consumer := strings.TrimPrefix(secretName, chart.keySecretName(""))
						for _, label := range labels {
							allowed, known := hopEndpoints[label]
							require.True(t, known, "unknown derived key label %q", label)
							assert.Contains(t, allowed, consumer, "%s must not hold the %s key", consumer, label)
						}
					}
				})
			}
		})
	}
}

// Re-running derive-keys.sh must reproduce the same keys, otherwise every upgrade would rotate
// every key and restart every agentic pod. The determinism lives in derive-keys.sh (HKDF over the
// instance secret), so what the chart has to get right is not passing anything release-specific
// into it - no timestamps, no random suffixes, nothing that differs between two renders.
func TestAgentKeyDerivationIsDeterministic(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			first := deriveScript(t, keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", nil))
			second := deriveScript(t, keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", nil))
			assert.Equal(t, first, second)
		})
	}
}

// Pod-level overrides have to reach the Job: a cluster with a restricted PodSecurity policy or a
// tight LimitRange would otherwise have no way to make the hook - and therefore the release -
// admissible.
func TestAgentKeyDerivationHonoursPodOverrides(t *testing.T) {
	for _, chart := range agenticKeyCharts {
		t.Run(chart.name, func(t *testing.T) {
			job := keyDerivationJob(t, chart, "agent-runtimes-enabled.yaml", map[string]string{
				"agentKeyDerivation.securityContext.runAsUser":                       "1234",
				"agentKeyDerivation.containerSecurityContext.readOnlyRootFilesystem": "true",
				"agentKeyDerivation.resources.limits.memory":                         "256Mi",
				"agentKeyDerivation.activeDeadlineSeconds":                           "120",
			})

			podSpec := job.Spec.Template.Spec
			require.NotNil(t, podSpec.SecurityContext)
			require.NotNil(t, podSpec.SecurityContext.RunAsUser)
			assert.EqualValues(t, 1234, *podSpec.SecurityContext.RunAsUser)

			container := podSpec.Containers[0]
			require.NotNil(t, container.SecurityContext)
			require.NotNil(t, container.SecurityContext.ReadOnlyRootFilesystem)
			assert.True(t, *container.SecurityContext.ReadOnlyRootFilesystem)
			assert.Equal(t, "256Mi", container.Resources.Limits.Memory().String())

			require.NotNil(t, job.Spec.ActiveDeadlineSeconds)
			assert.EqualValues(t, 120, *job.Spec.ActiveDeadlineSeconds)

			// readOnlyRootFilesystem is the default, so the script's scratch space must already be
			// a writable volume rather than the container filesystem.
			mount, ok := agentVolumeMountsByName(container.VolumeMounts)["tmp"]
			require.True(t, ok, "derive-keys.sh writes to $TMPDIR, which needs a writable volume")
			assert.Equal(t, "/tmp", mount.MountPath)
		})
	}
}
