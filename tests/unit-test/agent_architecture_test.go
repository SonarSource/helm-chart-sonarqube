package tests

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

const architectureFixture = "architecture-agent-enabled.yaml"

func renderArchitecture(t *testing.T, chart agentChart, templates []string, setValues map[string]string, extraArgs ...string) (string, error) {
	t.Helper()
	return helm.RenderTemplateE(t, agentPropertiesOptions(chart, architectureFixture, setValues), chart.path, chart.release, templates, extraArgs...)
}

func mustRenderArchitecture(t *testing.T, chart agentChart, templates []string, setValues map[string]string) string {
	t.Helper()
	output, err := renderArchitecture(t, chart, templates, setValues)
	require.NoError(t, err)
	return output
}

func architectureRuntime(t *testing.T, chart agentChart, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	output := mustRenderArchitecture(t, chart, []string{"templates/agent-runtime.yaml"}, setValues)
	for _, doc := range strings.Split(output, "\n---") {
		if !strings.Contains(doc, "kind: Deployment") {
			continue
		}
		var deployment appsv1.Deployment
		helm.UnmarshalK8SYaml(t, doc, &deployment)
		if deployment.Labels["sonarqube.agent/family"] == "architecture" {
			return deployment
		}
	}
	require.FailNow(t, "no architecture Deployment rendered")
	return appsv1.Deployment{}
}

func envByName(env []corev1.EnvVar) map[string]corev1.EnvVar {
	byName := map[string]corev1.EnvVar{}
	for _, e := range env {
		byName[e.Name] = e
	}
	return byName
}

func architectureSquidConf(t *testing.T, chart agentChart, setValues map[string]string) string {
	t.Helper()
	output := mustRenderArchitecture(t, chart, []string{"templates/agent-egress-proxy-configmap.yaml"}, setValues)
	var cm corev1.ConfigMap
	helm.UnmarshalK8SYaml(t, output, &cm)
	return cm.Data["squid.conf"]
}

func TestArchitectureAgentDisabledByDefault(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := helm.RenderTemplateE(t, agentPropertiesOptions(chart, "agent-runtimes-enabled.yaml", nil), chart.path, chart.release, []string{})
			require.NoError(t, err)
			assert.NotContains(t, output, "architecture")
			assert.NotContains(t, output, "ARCH_AGENT")
		})
	}
}

func TestArchitectureRuntimeDeployment(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			deployment := architectureRuntime(t, chart, nil)
			assert.Equal(t, chart.fullnamePrefix()+"-agent-runtime-architecture", deployment.Name)
			assert.Equal(t, int64(360), *deployment.Spec.Template.Spec.TerminationGracePeriodSeconds)

			container := deployment.Spec.Template.Spec.Containers[0]
			assert.Equal(t, "example.com/architecture-agent:1", container.Image)
			require.Len(t, container.Ports, 1)
			assert.Equal(t, int32(8090), container.Ports[0].ContainerPort)

			env := envByName(container.Env)
			podIP := env["POD_IP"]
			require.NotNil(t, podIP.ValueFrom, "POD_IP must come from the downward API")
			assert.Equal(t, "status.podIP", podIP.ValueFrom.FieldRef.FieldPath)
			assert.Equal(t, "8090", env["PORT"].Value)
			assert.Contains(t, env["SONARQUBE_URL"].Value, chart.fullnamePrefix()+".default.svc.cluster.local:9000")
			assert.Equal(t, "/etc/agentic/keys/architecture-to-sqs", env["ARCH_AGENT_SIGNING_KEY_PATH"].Value)
			assert.Equal(t, "/etc/agentic/keys/orchestrator-to-architecture", env["AGENTIC_VERIFY_KEY_PATH"].Value)
			assert.Equal(t, "orchestrator-to-architecture", env["AGENTIC_VERIFY_KEY_ID"].Value)
			assert.Equal(t, "/etc/agentic/keys/architecture-to-orchestrator", env["AGENT_ORCHESTRATOR_SIGNING_KEY_PATH"].Value)
			assert.Equal(t, "http://$("+strings.ToUpper(strings.ReplaceAll(chart.fullnamePrefix()+"-agent-egress-proxy", "-", "_"))+"_SERVICE_HOST):3130", env["HTTP_PROXY"].Value)
		})
	}
}

func TestArchitecturePodIPOnlyOnArchitectureRuntime(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			for _, family := range []string{"hunter", "remediation"} {
				deployment, err := renderAgentRuntimeDeploymentFamily(t, chart, family, map[string]string{"architectureAgent.enabled": "true", "architectureAgent.image.tag": "1"})
				require.NoError(t, err)
				assert.NotContains(t, envByName(deployment.Spec.Template.Spec.Containers[0].Env), "POD_IP", family)
			}
		})
	}
}

func TestArchitectureRuntimeService(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output := mustRenderArchitecture(t, chart, []string{"templates/agent-runtime-service.yaml"}, nil)
			var found bool
			for _, doc := range strings.Split(output, "\n---") {
				var service corev1.Service
				helm.UnmarshalK8SYaml(t, doc, &service)
				if service.Labels["sonarqube.agent/family"] != "architecture" {
					continue
				}
				found = true
				require.Len(t, service.Spec.Ports, 1)
				assert.Equal(t, int32(8090), service.Spec.Ports[0].Port)
				assert.Equal(t, "architecture", service.Labels["sonarqube.agent/family"])
				assert.Contains(t, service.Spec.Selector["app"], "agent-runtime-architecture")
			}
			assert.True(t, found)
		})
	}
}

func TestArchitectureKeysMountedAndDerived(t *testing.T) {
	archLabels := []string{"orchestrator-to-architecture", "architecture-to-sqs", "architecture-to-orchestrator"}
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			podSpec := agenticPodSpec(t, chart, "templates/agent-runtime.yaml", architectureFixture, nil, "architecture")
			volume := agentVolumesByName(podSpec.Volumes)[agenticKeyVolumeName]
			require.NotNil(t, volume.Secret)
			assert.Equal(t, chart.keySecretName("architecture"), volume.Secret.SecretName)
			var projected []string
			for _, item := range volume.Secret.Items {
				projected = append(projected, item.Key)
			}
			assert.ElementsMatch(t, archLabels, projected)
			mount := agentVolumeMountsByName(podSpec.Containers[0].VolumeMounts)[agenticKeyVolumeName]
			assert.Equal(t, "/etc/agentic/keys", mount.MountPath)
			assert.True(t, mount.ReadOnly)

			orchestrator := agenticPodSpec(t, chart, "templates/agent-orchestrator.yaml", architectureFixture, nil, "")
			orchestratorVolume := agentVolumesByName(orchestrator.Volumes)[agenticKeyVolumeName]
			var orchestratorLabels []string
			for _, item := range orchestratorVolume.Secret.Items {
				orchestratorLabels = append(orchestratorLabels, item.Key)
			}
			assert.Contains(t, orchestratorLabels, "orchestrator-to-architecture")
			assert.Contains(t, orchestratorLabels, "architecture-to-orchestrator")
			assert.NotContains(t, orchestratorLabels, "architecture-to-sqs")

			job := keyDerivationJob(t, chart, architectureFixture, nil)
			written := writtenSecrets(t, job)
			assert.ElementsMatch(t, archLabels, written[chart.keySecretName("architecture")])
			assert.NotContains(t, written[chart.keySecretName("sqs")], "architecture-to-sqs")
			assert.Contains(t, derivedLabels(t, job), "architecture-to-sqs")
		})
	}
}

func TestArchitectureSessionCapabilityKeyNeverDerivedOrMounted(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderArchitecture(t, chart, []string{}, nil)
			require.NoError(t, err)
			assert.NotContains(t, output, "architecture-session-capability")
		})
	}
}

func TestArchitectureKeySecretNameFitsWithLongRelease(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			release := strings.Repeat("x", 63-len(chart.name)-1)
			output, err := helm.RenderTemplateE(t, agentPropertiesOptions(chart, architectureFixture, nil), chart.path, release, []string{keyDerivationHookTemplate})
			require.NoError(t, err)
			var job batchv1.Job
			helm.UnmarshalK8SYaml(t, output, &job)
			written := writtenSecrets(t, job)
			names := map[string]bool{}
			for name := range written {
				assert.LessOrEqual(t, len(name), 63)
				assert.False(t, names[name], "%s collides with another consumer", name)
				names[name] = true
			}
			assert.Len(t, written, 6, "every consumer keeps a distinct Secret name")
			var architecture, remediation string
			for name := range written {
				switch {
				case strings.HasSuffix(name, "-agentic-keys-architecture"):
					architecture = name
				case strings.HasSuffix(name, "-agentic-keys-remediation"):
					remediation = name
				}
			}
			assert.NotEmpty(t, architecture, "the architecture suffix must survive intact")
			assert.NotEmpty(t, remediation)
		})
	}
}

func TestArchitectureOrchestratorTestKnobs(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			env := envByName(renderAgentOrchestratorFromFixture(t, chart, nil).Spec.Template.Spec.Containers[0].Env)
			assert.Equal(t, "600", env["ARCHITECTURE_IDLE_TTL_SECONDS"].Value)
			assert.NotContains(t, env, "ARCHITECTURE_COST_BUDGET_USD")

			env = envByName(renderAgentOrchestratorFromFixture(t, chart, map[string]string{
				"agentOrchestrator.architecture.idleTtlSeconds": "60",
				"agentOrchestrator.architecture.costBudgetUsd":  "0.05",
			}).Spec.Template.Spec.Containers[0].Env)
			assert.Equal(t, "60", env["ARCHITECTURE_IDLE_TTL_SECONDS"].Value)
			assert.Equal(t, "0.05", env["ARCHITECTURE_COST_BUDGET_USD"].Value)
			assert.Equal(t, chart.fullnamePrefix()+"-agent-runtime-architecture", strings.SplitN(strings.TrimPrefix(env["AGENTIC_ARCHITECTURE_RUNTIME_PUSH_URL"].Value, "http://"), ".", 2)[0])

			env = envByName(renderAgentOrchestratorFromFixture(t, chart, map[string]string{"architectureAgent.enabled": "false"}).Spec.Template.Spec.Containers[0].Env)
			assert.NotContains(t, env, "ARCHITECTURE_IDLE_TTL_SECONDS")
			assert.NotContains(t, env, "AGENTIC_ARCHITECTURE_RUNTIME_PUSH_URL")
		})
	}
}

func renderAgentOrchestratorFromFixture(t *testing.T, chart agentChart, setValues map[string]string) appsv1.Deployment {
	t.Helper()
	output := mustRenderArchitecture(t, chart, []string{"templates/agent-orchestrator.yaml"}, setValues)
	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	return deployment
}

func TestArchitectureEgressListenerAndEndpointAllowlist(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			conf := architectureSquidConf(t, chart, nil)
			assert.Contains(t, conf, "http_port 3130")
			assert.Contains(t, conf, "acl architecture_listener localport 3130")
			assert.Contains(t, conf, "acl architecture_read_method method GET")
			assert.Contains(t, conf, "http_access allow sonarqube_host architecture_endpoints architecture_read_method architecture_listener")
			assert.Less(t, strings.Index(conf, "http_access allow sonarqube_host architecture_endpoints"), strings.Index(conf, "http_access deny all"))
			assert.NotContains(t, conf, "request_header_access", "the capability and signature headers must pass through unchanged")
			assert.Equal(t, 1, strings.Count(conf, "acl sonarqube_host dstdomain"))

			matches := urlpathRegexMatcher(t, conf, "architecture_endpoints")
			for _, path := range []string{
				"/api/v2/architecture/graphs",
				"/api/v2/architecture/graphs?projectId=p",
				"/api/v2/architecture/graphs/0123abcd",
				"/api/v2/architecture/graphs/0123abcd?branch=main",
				"/api/v2/architecture/models",
				"/api/v2/architecture/models?projectId=p",
				"/api/v2/architecture/project-configurations",
				"/api/projects/search?q=a",
				"/api/project_branches/list?project=a",
				"/api/project_analyses/search?project=a",
				"/api/components/tree?component=a",
				"/api/components/show?component=a",
				"/api/sources/raw?key=a",
			} {
				assert.True(t, matches(path), "%s must be allowed", path)
			}
			for _, path := range []string{
				"/api/v2/architecture/graphs/a/b",
				"/api/v2/architecture/graphs/",
				"/api/v2/architecture/models/a",
				"/api/v2/architecture/project-configurations/a",
				"/api/v2/architecture/other",
				"/api/v2/architecture-agent/sessions/abc/events",
				"/api/projects/search/extra",
				"/api/projects/create",
				"/api/components/search",
				"/api/sources/show?key=a",
				"/api/rules/show?key=a",
				"/api/v2/a3s/private/analyses",
				"/other/api/projects/search",
				"/x?y=/api/projects/search",
			} {
				assert.False(t, matches(path), "%s must be refused", path)
			}
		})
	}
}

func TestArchitectureEndpointAllowlistFollowsWebContext(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			conf := architectureSquidConf(t, chart, map[string]string{"sonarWebContext": "/sonar.qube"})
			matches := urlpathRegexMatcher(t, conf, "architecture_endpoints")
			assert.True(t, matches("/sonar.qube/api/projects/search?q=a"))
			assert.True(t, matches("/sonar.qube/api/v2/architecture/models"))
			assert.False(t, matches("/api/projects/search"))
			assert.False(t, matches("/sonarXqube/api/projects/search"))
		})
	}
}

func TestArchitectureListenerIsolatedFromOtherRuntimes(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			conf := architectureSquidConf(t, chart, nil)
			listener := regexp.MustCompile(`acl (\w+) localport (\d+)`)
			ports := map[string]string{}
			for _, m := range listener.FindAllStringSubmatch(conf, -1) {
				ports[m[1]] = m[2]
			}
			assert.Equal(t, "3130", ports["architecture_listener"])
			assert.Equal(t, "3129", ports["remediation_listener"])
			assert.NotContains(t, conf, "http_access allow sonarqube_host sonarqube_agentic_endpoints architecture_listener")

			hunterOnly := architectureSquidConf(t, chart, map[string]string{"architectureAgent.enabled": "false", "remediationAgent.enabled": "false", "vortexAnalysis.enabled": "false"})
			assert.NotContains(t, hunterOnly, "architecture")
			assert.NotContains(t, hunterOnly, "sonarqube_host")

			remediationOnly := architectureSquidConf(t, chart, map[string]string{"architectureAgent.enabled": "false"})
			assert.NotContains(t, remediationOnly, "architecture")
		})
	}
}

func TestArchitectureOnlyEgressStillDefinesSonarQubeHost(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			conf := architectureSquidConf(t, chart, map[string]string{"remediationAgent.enabled": "false", "hunterAgent.enabled": "false", "vortexAnalysis.enabled": "false"})
			assert.Equal(t, 1, strings.Count(conf, "acl sonarqube_host dstdomain"))
			assert.Contains(t, conf, "acl Safe_ports port 9000")
			assert.NotContains(t, conf, "sonarqube_agentic_endpoints")
		})
	}
}

func TestArchitectureEgressProxyServiceDeploymentAndPolicy(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output := mustRenderArchitecture(t, chart, []string{"templates/agent-egress-proxy-service.yaml"}, nil)
			var service corev1.Service
			helm.UnmarshalK8SYaml(t, output, &service)
			var ports []int32
			for _, p := range service.Spec.Ports {
				ports = append(ports, p.Port)
			}
			assert.ElementsMatch(t, []int32{3128, 3129, 3130}, ports)

			output = mustRenderArchitecture(t, chart, []string{"templates/agent-egress-proxy.yaml"}, nil)
			var deployment appsv1.Deployment
			helm.UnmarshalK8SYaml(t, output, &deployment)
			var containerPorts []int32
			for _, p := range deployment.Spec.Template.Spec.Containers[0].Ports {
				containerPorts = append(containerPorts, p.ContainerPort)
			}
			assert.ElementsMatch(t, []int32{3128, 3129, 3130}, containerPorts)

			output = mustRenderArchitecture(t, chart, []string{"templates/agent-egress-proxy-networkpolicy.yaml"}, nil)
			var policy networkingv1.NetworkPolicy
			helm.UnmarshalK8SYaml(t, output, &policy)
			portsByFamily := map[string][]int32{}
			for _, rule := range policy.Spec.Ingress {
				for _, from := range rule.From {
					app := from.PodSelector.MatchLabels["app"]
					for _, p := range rule.Ports {
						portsByFamily[app] = append(portsByFamily[app], int32(p.Port.IntValue()))
					}
				}
			}
			assert.Equal(t, []int32{3130}, portsByFamily[chart.name+"-agent-runtime-architecture"])
			assert.Equal(t, []int32{3128}, portsByFamily[chart.name+"-agent-runtime-hunter"])
			assert.Equal(t, []int32{3129}, portsByFamily[chart.name+"-agent-runtime-remediation"])
		})
	}
}

func TestArchitectureRuntimeNetworkPolicy(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output := mustRenderArchitecture(t, chart, []string{"templates/agent-networkpolicy.yaml"}, nil)
			var policy networkingv1.NetworkPolicy
			for _, doc := range strings.Split(output, "\n---") {
				var candidate networkingv1.NetworkPolicy
				helm.UnmarshalK8SYaml(t, doc, &candidate)
				if candidate.Labels["sonarqube.agent/family"] == "architecture" {
					policy = candidate
				}
			}
			require.NotEmpty(t, policy.Name)

			require.Len(t, policy.Spec.Ingress, 1)
			assert.Equal(t, chart.name+"-agent-orchestrator", policy.Spec.Ingress[0].From[0].PodSelector.MatchLabels["app"])
			assert.Equal(t, 8090, policy.Spec.Ingress[0].Ports[0].Port.IntValue())

			require.Len(t, policy.Spec.Egress, 1)
			assert.Equal(t, chart.name+"-agent-egress-proxy", policy.Spec.Egress[0].To[0].PodSelector.MatchLabels["app"])
			require.Len(t, policy.Spec.Egress[0].Ports, 1)
			assert.Equal(t, 3130, policy.Spec.Egress[0].Ports[0].Port.IntValue())
		})
	}
}

func TestArchitectureScaledObject(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			set := map[string]string{"architectureAgent.autoscaling.enabled": "true", "agentKeda.assumeInstalled": "true"}
			output := mustRenderArchitecture(t, chart, []string{"templates/agent-runtime-scaledobject.yaml"}, set)
			var so scaledObject
			for _, doc := range strings.Split(output, "\n---") {
				var candidate scaledObject
				helm.UnmarshalK8SYaml(t, doc, &candidate)
				if candidate.Metadata.Labels["sonarqube.agent/family"] == "architecture" {
					so = candidate
				}
			}
			require.NotEmpty(t, so.Metadata.Name)
			assert.Contains(t, output, `valueLocation: "architecture.unfinished"`)
			assert.Contains(t, output, "minReplicaCount: 2")
			assert.Contains(t, output, "maxReplicaCount: 25")
			assert.Contains(t, output, `targetValue: "1"`)
			assert.Contains(t, output, "stabilizationWindowSeconds: 600")
			assert.NotContains(t, output, `valueLocation: "hunter.unfinished"`)

			set["architectureAgent.autoscaling.minReplicas"] = "0"
			_, err := renderArchitecture(t, chart, []string{"templates/agent-runtime-scaledobject.yaml"}, set)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "architectureAgent.autoscaling.minReplicas must be >= 1")

			set["architectureAgent.autoscaling.minReplicas"] = "1"
			_, err = renderArchitecture(t, chart, []string{"templates/agent-runtime-scaledobject.yaml"}, set)
			require.NoError(t, err, "a single warm replica stays valid for architecture")
			set["hunterAgent.autoscaling.enabled"] = "true"
			set["hunterAgent.autoscaling.minReplicas"] = "1"
			_, err = renderArchitecture(t, chart, []string{"templates/agent-runtime-scaledobject.yaml"}, set)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "hunterAgent.autoscaling.minReplicas must be >= 2")
		})
	}
}

func TestArchitectureValidation(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			cases := []struct {
				name string
				set  map[string]string
				want string
			}{
				{"needs the orchestrator", map[string]string{"agentOrchestrator.enabled": "false", "hunterAgent.enabled": "false", "remediationAgent.enabled": "false"}, "architectureAgent.enabled is true but agentOrchestrator.enabled is not true"},
				{"needs an image tag", map[string]string{"architectureAgent.image.repository": "sonarsource/sonarqube-architecture-agent", "architectureAgent.image.tag": ""}, "architectureAgent.image.tag is empty"},
				{"listener port differs from the shared one", map[string]string{"agentEgressProxy.architecturePort": "3128"}, "architecturePort (3128) must differ from agentEgressProxy.port"},
				{"listener port differs from remediation's", map[string]string{"agentEgressProxy.architecturePort": "3129"}, "architecturePort (3129) must differ from agentEgressProxy.remediationPort"},
				{"needs the signing secret", map[string]string{"agenticSigningSecret.existingSecret": ""}, "agenticSigningSecret.existingSecret is not set"},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					_, err := renderArchitecture(t, chart, []string{"templates/agent-runtime.yaml"}, tc.set)
					require.Error(t, err)
					assert.Contains(t, err.Error(), tc.want)
				})
			}
		})
	}
}

func TestArchitectureAloneSatisfiesOrchestratorRuntimeRequirement(t *testing.T) {
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			output, err := renderArchitecture(t, chart, []string{"templates/agent-orchestrator.yaml"}, map[string]string{"hunterAgent.enabled": "false", "remediationAgent.enabled": "false"})
			require.NoError(t, err)
			assert.Contains(t, output, "AGENTIC_ARCHITECTURE_RUNTIME_PUSH_URL")
		})
	}
}

func TestArchitectureOrchestratorURLOnApplication(t *testing.T) {
	const property = "sonar.architectureagent.orchestrator.url"
	const envName = "SONAR_ARCHITECTUREAGENT_ORCHESTRATOR_URL"
	for _, chart := range agentCharts {
		t.Run(chart.name, func(t *testing.T) {
			props := appSonarProperties(t, chart, agentPropertiesOptions(chart, architectureFixture, nil))
			assert.Equal(t, chart.orchestratorURL(), props[property])
			podSpec := agenticPodSpec(t, chart, chart.appTemplate, architectureFixture, nil, "")
			assert.Equal(t, chart.orchestratorURL(), envByName(podSpec.Containers[0].Env)[envName].Value)

			off := map[string]string{"architectureAgent.enabled": "false"}
			props = appSonarProperties(t, chart, agentPropertiesOptions(chart, architectureFixture, off))
			assert.NotContains(t, props, property)
			podSpec = agenticPodSpec(t, chart, chart.appTemplate, architectureFixture, off, "")
			assert.NotContains(t, envByName(podSpec.Containers[0].Env), envName)
		})
	}
}
