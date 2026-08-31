{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "sonarqube.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "sonarqube.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "sonarqube.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "sonarqube.labels" -}}
app: {{ include "sonarqube.name" . }}
chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
release: {{ .Release.Name }}
heritage: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "sonarqube.selectorLabels" -}}
app: {{ include "sonarqube.name" . }}
release: {{ .Release.Name }}
{{- end -}}

{{/*
Workload labels (Deployment or StatefulSet)
*/}}
{{- define "sonarqube.workloadLabels" -}}
{{- include "sonarqube.labels" . }}
app.kubernetes.io/name: {{ .Release.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: sonarqube
app.kubernetes.io/component: {{ include "sonarqube.fullname" . }}
app.kubernetes.io/version: {{ (tpl (include "image.tag" .) . ) | trunc 63 | trimSuffix "-" | quote }}
{{- end -}}

{{/*
Expand the Application Image name.
*/}}
{{- define "sonarqube.image" -}}
{{- if and .Values.global .Values.global.azure .Values.global.azure.images .Values.global.azure.images.sonarqube }}
{{- printf "%s/%s:%s" .Values.global.azure.images.sonarqube.registry .Values.global.azure.images.sonarqube.image .Values.global.azure.images.sonarqube.tag }}
{{- else }}
{{- printf "%s:%s" .Values.image.repository (tpl (include "image.tag" .) .) }}
{{- end -}}
{{- end -}}

{{/*
Check if Azure configuration is complete
*/}}
{{- define "sonarqube.azure.enabled" -}}
{{- if and .Values.global .Values.global.azure -}}
  {{- with .Values.global.azure -}}
    {{- if and .identity .extension .marketplace -}}
      {{- if and .identity.clientId .extension.resourceId .marketplace.planId -}}
        {{- true -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
  Define the image.tag value that computes the right tag to be used as `sonarqube.image`
  The tag is derived from the following parameters:
  - .Values.image.tag
  - .Values.community.enabled
  - .Values.community.buildNumber
  - .Values.edition
  - .Chart.AppVersion

  The logic to generate the tag is as follows:
  There should not be a default edition, with users that specify it.
  The edition must be one of these values: developer/enterprise.
  When “edition“ is used and “image.tag” is not, we use “appVersion” for paid editions and the latest release of SQ-CB for the community.
  The CI supports the release of the Server edition.
*/}}
{{- define "image.tag" -}}
  {{- $imageTag := "" -}}
  {{- if not (empty .Values.edition) -}}
    {{- if or (empty .Values.image) (empty .Values.image.tag) -}}
      {{- $imageTag = printf "%s-%s" .Chart.AppVersion .Values.edition -}}
    {{- else -}}
      {{- $imageTag = printf "%s" .Values.image.tag -}}
    {{- end -}}
  {{- else if (and (.Values.community) .Values.community.enabled) -}}
    {{- if or (empty .Values.image) (empty .Values.image.tag) -}}
      {{- if not (empty .Values.community.buildNumber) -}}
        {{- $imageTag = printf "%s-%s" .Values.community.buildNumber "community" -}}
      {{- else -}}
        {{- $imageTag = printf "community" -}}
      {{- end -}}
    {{- else -}}
      {{- $imageTag = printf "%s" .Values.image.tag -}}
    {{- end -}}
  {{- end -}}
  {{- printf "%s" $imageTag -}}
{{- end -}}

{{/*
Determine the k8s secret containing the JDBC credentials
*/}}
{{- define "jdbc.secret" -}}
{{- if or .Values.jdbcOverwrite.enabled .Values.jdbcOverwrite.enable -}}
  {{- if .Values.jdbcOverwrite.jdbcSecretName -}}
  {{- .Values.jdbcOverwrite.jdbcSecretName -}}
  {{- else -}}
  {{- template "sonarqube.fullname" . -}}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Determine JDBC username
*/}}
{{- define "jdbc.username" -}}
{{- if and (or .Values.jdbcOverwrite.enabled .Values.jdbcOverwrite.enable) .Values.jdbcOverwrite.jdbcUsername -}}
{{- .Values.jdbcOverwrite.jdbcUsername | quote -}}
{{- end -}}
{{- end -}}

{{/*
Determine the k8s secretKey contrining the JDBC password
*/}}
{{- define "jdbc.secretPasswordKey" -}}
{{- if or .Values.jdbcOverwrite.enabled .Values.jdbcOverwrite.enable -}}
  {{- if and .Values.jdbcOverwrite.jdbcSecretName .Values.jdbcOverwrite.jdbcSecretPasswordKey -}}
  {{- .Values.jdbcOverwrite.jdbcSecretPasswordKey -}}
  {{- else -}}
  {{- "jdbc-password" -}}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Determine JDBC password if internal secret is used
*/}}
{{- define "jdbc.internalSecretPasswd" -}}
{{- if or .Values.jdbcOverwrite.enabled .Values.jdbcOverwrite.enable -}}
  {{- .Values.jdbcOverwrite.jdbcPassword | b64enc | quote -}}
{{- end -}}
{{- end -}}

{{/*
Set sonarqube.jvmOpts
*/}}
{{- define "sonarqube.jvmOpts" -}}
{{- $tempJvm := .Values.jvmOpts -}}
{{- if and .Values.sonarProperties (hasKey (.Values.sonarProperties) "sonar.web.javaOpts")}}
{{- $tempJvm = (get .Values.sonarProperties "sonar.web.javaOpts") -}}
{{- else if .Values.env -}}
{{- range $index, $val := .Values.env -}}
{{- if eq $val.name "SONAR_WEB_JAVAOPTS" -}}
{{- $tempJvm = $val.value -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- if and .Values.caCerts.enabled .Values.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.prometheusExporter.webBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder $tempJvm | trim }}
{{- else if .Values.caCerts.enabled -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder $tempJvm | trim }}
{{- else if .Values.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml %s" .Values.sonarqubeFolder (int .Values.prometheusExporter.webBeanPort) .Values.sonarqubeFolder $tempJvm | trim }}
{{- else -}}
{{ printf "%s" $tempJvm }}
{{- end -}}
{{- end -}}

{{/*
Set sonarqube.jvmCEOpts
*/}}
{{- define "sonarqube.jvmCEOpts" -}}
{{- $tempJvm := .Values.jvmCeOpts -}}
{{- if and .Values.sonarProperties (hasKey (.Values.sonarProperties) "sonar.ce.javaOpts")}}
{{- $tempJvm = (get .Values.sonarProperties "sonar.ce.javaOpts") -}}
{{- else if .Values.env -}}
{{- range $index, $val := .Values.env -}}
{{- if eq $val.name "SONAR_CE_JAVAOPTS" -}}
{{- $tempJvm = $val.value -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- if and .Values.caCerts.enabled .Values.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder $tempJvm | trim }}
{{- else if .Values.caCerts.enabled -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder $tempJvm | trim }}
{{- else if .Values.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml %s" .Values.sonarqubeFolder (int .Values.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder $tempJvm | trim }}
{{- else -}}
{{ printf "%s" $tempJvm }}
{{- end -}}
{{- end -}}

{{/*
Set prometheusExporter.downloadURL
*/}}
{{- define "prometheusExporter.downloadURL" -}}
{{- if .Values.prometheusExporter.downloadURL -}}
{{ printf "%s" .Values.prometheusExporter.downloadURL }}
{{- else -}}
{{ printf "https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/%s/jmx_prometheus_javaagent-%s.jar" .Values.prometheusExporter.version .Values.prometheusExporter.version }}
{{- end -}}
{{- end -}}


{{/*
Create the name of the service account to use
*/}}
{{- define "sonarqube.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "sonarqube.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Set sonarqube.webcontext, ensuring it starts and ends with a slash, in order to ease probes url template
*/}}
{{- define "sonarqube.webcontext" -}}
{{- $tempWebcontext := .Values.sonarWebContext -}}
{{- if and .Values.sonarProperties (hasKey (.Values.sonarProperties) "sonar.web.context") -}}
{{- $tempWebcontext = (get .Values.sonarProperties "sonar.web.context") -}}
{{- end -}}
{{- range $index, $val := .Values.env -}}
{{- if eq $val.name "SONAR_WEB_CONTEXT" -}}
{{- $tempWebcontext = $val.value -}}
{{- end -}}
{{- end -}}
{{- if not (hasPrefix "/" $tempWebcontext) -}}
{{- $tempWebcontext = print "/" $tempWebcontext -}}
{{- end -}}
{{- if not (hasSuffix "/" $tempWebcontext) -}}
{{- $tempWebcontext = print $tempWebcontext "/" -}}
{{- end -}}
{{ printf "%s" $tempWebcontext }}
{{- end -}}

{{/*
Set combined_env, ensuring we dont have any duplicates with our features and some of the user provided env vars
*/}}
{{- define "sonarqube.combined_env" -}}
{{- $filteredEnv := list -}}
{{- range $index,$val := .Values.env -}}
  {{- if not (has $val.name (list "SONAR_WEB_CONTEXT" "SONAR_WEB_JAVAOPTS" "SONAR_CE_JAVAOPTS")) -}}
    {{- $filteredEnv = append $filteredEnv $val -}}
  {{- end -}}
{{- end -}}
{{- $filteredEnv = append $filteredEnv (dict "name" "SONAR_WEB_CONTEXT" "value" (include "sonarqube.webcontext" .)) -}}
{{- $filteredEnv = append $filteredEnv (dict "name" "SONAR_WEB_JAVAOPTS" "value" (include "sonarqube.jvmOpts" .)) -}}
{{- $filteredEnv = append $filteredEnv (dict "name" "SONAR_CE_JAVAOPTS" "value" (include "sonarqube.jvmCEOpts" .)) -}}
{{- toJson $filteredEnv -}}
{{- end -}}


{{/*
  generate Proxy env var from httpProxySecret
*/}}
{{- define "sonarqube.proxyFromSecret" -}}
- name: http_proxy
  valueFrom:
    secretKeyRef:
      name: {{ .Values.httpProxySecret }}
      key: http_proxy
- name: https_proxy
  valueFrom:
    secretKeyRef:
      name: {{ .Values.httpProxySecret }}
      key: https_proxy
- name: no_proxy
  valueFrom:
    secretKeyRef:
      name: {{ .Values.httpProxySecret }}
      key: no_proxy
{{- end -}}

{{/*
  generate prometheusExporter proxy env var
*/}}
{{- define "sonarqube.prometheusExporterProxy.env" -}}
{{- if .Values.httpProxySecret -}}
{{- include "sonarqube.proxyFromSecret" . }}
{{- else -}}
- name: http_proxy
  valueFrom:
    secretKeyRef:
      name: {{ template "sonarqube.fullname" . }}-http-proxies
      key: PROMETHEUS-EXPORTER-HTTP-PROXY
- name: https_proxy
  valueFrom:
    secretKeyRef:
      name: {{ template "sonarqube.fullname" . }}-http-proxies
      key: PROMETHEUS-EXPORTER-HTTPS-PROXY
- name: no_proxy
  valueFrom:
    secretKeyRef:
      name: {{ template "sonarqube.fullname" . }}-http-proxies
      key: PROMETHEUS-EXPORTER-NO-PROXY
{{- end -}}
{{- end -}}

{{/*
  generate install-plugins proxy env var
*/}}
{{- define "sonarqube.install-plugins-proxy.env" -}}
{{- if .Values.httpProxySecret -}}
{{- include "sonarqube.proxyFromSecret" . }}
{{- else -}}
- name: http_proxy
  valueFrom:
    secretKeyRef:
      name: {{ template "sonarqube.fullname" . }}-http-proxies
      key: PLUGINS-HTTP-PROXY
- name: https_proxy
  valueFrom:
    secretKeyRef:
      name: {{ template "sonarqube.fullname" . }}-http-proxies
      key: PLUGINS-HTTPS-PROXY
- name: no_proxy
  valueFrom:
    secretKeyRef:
      name: {{ template "sonarqube.fullname" . }}-http-proxies
      key: PLUGINS-NO-PROXY
{{- end -}}
{{- end -}}

{{/*
Remove incompatible user/group values that do not work in Openshift out of the box
*/}}
{{- define "sonarqube.securityContext" -}}
{{- $adaptedSecurityContext := .Values.securityContext -}}
  {{- if .Values.OpenShift.enabled -}}
    {{- $adaptedSecurityContext = omit $adaptedSecurityContext "fsGroup" "runAsUser" "runAsGroup" -}}
  {{- end -}}
  {{- toYaml $adaptedSecurityContext -}}
{{- end -}}


{{/*
Remove incompatible user/group values that do not work in Openshift out of the box
*/}}
{{- define "sonarqube.containerSecurityContext" -}}
{{- $adaptedContainerSecurityContext := .Values.containerSecurityContext -}}
  {{- if .Values.OpenShift.enabled -}}
    {{- $adaptedContainerSecurityContext = omit $adaptedContainerSecurityContext "fsGroup" "runAsUser" "runAsGroup" -}}
  {{- end -}}
{{- toYaml $adaptedContainerSecurityContext -}}
{{- end -}}

{{/*
Remove incompatible user/group values that do not work in Openshift out of the box
*/}}
{{- define "sonarqube.initContainerSecurityContext" -}}
{{- $adaptedInitContainerSecurityContext := .Values.initContainers.securityContext -}}
  {{- if .Values.OpenShift.enabled -}}
    {{- $adaptedInitContainerSecurityContext = omit $adaptedInitContainerSecurityContext "fsGroup" "runAsUser" "runAsGroup" -}}
  {{- end -}}
{{- toYaml $adaptedInitContainerSecurityContext -}}
{{- end -}}

{{/*
  generate caCerts volume
*/}}
{{- define "sonarqube.volumes.caCerts" -}}
{{- if .Values.caCerts.enabled -}}
- name: ca-certs
  {{- if .Values.caCerts.secret }}
  secret:
    secretName: {{ .Values.caCerts.secret }}
  {{- else if .Values.caCerts.configMap }}
  configMap:
    name: {{ .Values.caCerts.configMap.name }}
    {{- if or .Values.caCerts.configMap.key .Values.caCerts.configMap.path }}
    items:
      - key: {{ required "caCerts.configMap.key is required when caCerts.configMap.path is set" .Values.caCerts.configMap.key }}
        path: {{ default .Values.caCerts.configMap.key .Values.caCerts.configMap.path }}
    {{- end }}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
  This helper deeply merges two maps (structs). It recursively merges nested maps and takes the values from `map2` when keys overlap.
*/}}
{{- define "deepMerge" -}}
{{- $map1 := .map1 -}}
{{- $map2 := .map2 -}}

{{- $result := dict -}}

{{- /* Merge keys from map1 */}}
{{- range $key, $value := $map1 -}}
  {{- $_ := set $result $key $value -}}
{{- end -}}

{{- /* Merge keys from map2 (overriding map1 if the key exists) */}}
{{- range $key, $value := $map2 -}}
  {{- if hasKey $map1 $key -}}
    {{- /* If both maps have the same key and the value is a map, we need to merge recursively */}}
    {{- if and (kindIs "map" $value) (kindIs "map" (index $map1 $key)) -}}
      {{- $_ := set $result $key (fromYaml (include "deepMerge" (dict "map1" (index $map1 $key) "map2" $value))) -}}
    {{- else -}}
      {{- /* Otherwise, just take the value from map2 */}}
      {{- $_ := set $result $key $value -}}
    {{- end -}}
  {{- else -}}
    {{- /* If map2 has a key not in map1, just add it to the result */}}
    {{- $_ := set $result $key $value -}}
  {{- end -}}
{{- end -}}

{{- toYaml $result -}}
{{- end -}}

{{/*
Create the fully qualified name for the MCP service.
*/}}
{{- define "sonarqube.mcp.fullname" -}}
{{- printf "%s-mcp" (include "sonarqube.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the fully qualified name for Vortex.
Usage: {{ include "sonarqube.vortex.fullname" . }}
*/}}
{{- define "sonarqube.vortex.fullname" -}}
{{- printf "%s-vortex" (include "sonarqube.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
URL the application nodes use to reach Vortex (the sonar.vortex.analysis.url property).
*/}}
{{- define "sonarqube.vortex.url" -}}
{{- printf "http://%s:%d" (include "sonarqube.vortex.fullname" .) (int .Values.vortex.port) -}}
{{- end -}}

{{/*
Name of the ServiceAccount for the Vortex pod. Same create / pinned-name / "default"
fallback logic as sonarqube.serviceAccountName, but independent of it.
*/}}
{{- define "sonarqube.vortex.serviceAccountName" -}}
{{- if .Values.vortex.serviceAccount.create -}}
    {{ default (include "sonarqube.vortex.fullname" .) .Values.vortex.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.vortex.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Create the fully qualified name for the gVisor installer.
Usage: {{ include "sonarqube.gvisor.fullname" . }}
*/}}
{{- define "sonarqube.gvisor.fullname" -}}
{{- printf "%s-gvisor-installer" (include "sonarqube.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Effective gvisor.enabled: requires at least one of hunterAgent.enabled / remediationAgent.enabled
too, so gVisor only ever renders when there's a runtime to sandbox.
Usage: {{ include "sonarqube.gvisor.enabled" . }}
*/}}
{{- define "sonarqube.gvisor.enabled" -}}
{{- and .Values.gvisor.enabled (or .Values.hunterAgent.enabled .Values.remediationAgent.enabled) -}}
{{- end -}}

{{/*
Return the target Kubernetes version
*/}}
{{- define "common.capabilities.kubeVersion" -}}
{{- print .Capabilities.KubeVersion.Version -}}
{{- end -}}

{{/*
Return the appropriate apiVersion for poddisruptionbudget.
*/}}
{{- define "common.capabilities.policy.apiVersion" -}}
{{- if semverCompare "<1.21-0" (include "common.capabilities.kubeVersion" .) -}}
{{- print "policy/v1beta1" -}}
{{- else -}}
{{- print "policy/v1" -}}
{{- end -}}
{{- end -}}

{{- define "accountDeprecation" -}}
{{- $map1 := .Values.setAdminPassword -}}
{{- $map2 := .Values.account -}}

{{- $accountDeprecation := (include "deepMerge" (dict "map1" $map1 "map2" $map2)) -}}
{{- $accountDeprecation }}
{{- end -}}

{{/*
Create the fully qualified name for the Agent Orchestrator.
*/}}
{{- define "sonarqube.agentOrchestrator.fullname" -}}
{{- printf "%s-agent-orchestrator" (include "sonarqube.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the fully qualified name for an agent runtime family.
Parameters (dict): ctx (required, the root context '.'), family (required, the runtime family name)
*/}}
{{- define "sonarqube.agentRuntime.fullname" -}}
{{- printf "%s-agent-runtime-%s" (include "sonarqube.fullname" .ctx) .family | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Selector labels for the Agent Orchestrator: app: <chart>-agent-orchestrator + release, matching
the Vortex convention so the value doesn't collide with the SonarQube app Deployment's own
selector (app: <chart> + release).
*/}}
{{- define "sonarqube.agentOrchestrator.selectorLabels" -}}
app: {{ include "sonarqube.name" . }}-agent-orchestrator
release: {{ .Release.Name }}
{{- end -}}

{{/*
Selector labels for one agent runtime family: app: <chart>-agent-runtime-<family> + release.
Parameters (dict): ctx (required, the root context '.'), family (required, the runtime family name)
*/}}
{{- define "sonarqube.agentRuntime.selectorLabels" -}}
app: {{ include "sonarqube.name" .ctx }}-agent-runtime-{{ .family }}
release: {{ .ctx.Release.Name }}
{{- end -}}

{{/*
The two agent runtime families, keyed by name, for templates that iterate over both.
Usage: {{- range $family, $cfg := fromYaml (include "sonarqube.agentRuntimes" .) }}
*/}}
{{- define "sonarqube.agentRuntimes" -}}
hunter: {{- .Values.hunterAgent | toYaml | nindent 2 }}
remediation: {{- .Values.remediationAgent | toYaml | nindent 2 }}
{{- end -}}

{{/*
Name of the ServiceAccount for the Agent Orchestrator.
When agentOrchestrator.serviceAccount.create is true, use the pinned agentOrchestrator.serviceAccount.name
(defaulting to the orchestrator fullname). Otherwise fall back to the main SonarQube ServiceAccount,
so deployments that don't opt in to dedicated agent SAs are unaffected.
*/}}
{{- define "sonarqube.agentOrchestrator.serviceAccountName" -}}
{{- if .Values.agentOrchestrator.serviceAccount.create -}}
{{- default (include "sonarqube.agentOrchestrator.fullname" .) .Values.agentOrchestrator.serviceAccount.name -}}
{{- else -}}
{{- include "sonarqube.serviceAccountName" . -}}
{{- end -}}
{{- end -}}

{{/*
Name of the ServiceAccount for an agent runtime family.
Parameters (dict): ctx (required, the root context '.'), family (required, the runtime family name)
Same create / pinned-name / fallback logic as the orchestrator helper above.
*/}}
{{- define "sonarqube.agentRuntime.serviceAccountName" -}}
{{- $ctx := .ctx -}}
{{- $family := .family -}}
{{- $cfg := get (fromYaml (include "sonarqube.agentRuntimes" $ctx)) $family -}}
{{- if $cfg.serviceAccount.create -}}
{{- default (include "sonarqube.agentRuntime.fullname" (dict "ctx" $ctx "family" $family)) $cfg.serviceAccount.name -}}
{{- else -}}
{{- include "sonarqube.serviceAccountName" $ctx -}}
{{- end -}}
{{- end -}}

{{/*
Fully qualified in-cluster DNS name for a Service, given its short name. Used for every in-cluster
URL this chart builds (not just proxied ones, for consistency): Squid's own DNS resolver only
reads `nameserver`/`ndots` from /etc/resolv.conf, not the `search` list, so it can never resolve a
bare short Service name the way normal container DNS resolution does - any address a runtime
reaches through the Agent Egress Proxy must be fully qualified or Squid fails with ERR_DNS_FAIL.
Parameters (dict): name (required, the Service's short name), ctx (required, the root context '.')
*/}}
{{- define "sonarqube.svcFQDN" -}}
{{- printf "%s.%s.svc.cluster.local" .name .ctx.Release.Namespace -}}
{{- end -}}

{{/*
URL the app nodes use to reach the shared Agent Orchestrator.
*/}}
{{- define "sonarqube.agentOrchestrator.url" -}}
{{- printf "http://%s:%d" (include "sonarqube.svcFQDN" (dict "name" (include "sonarqube.agentOrchestrator.fullname" .) "ctx" .)) (int .Values.agentOrchestrator.port) -}}
{{- end -}}

{{/*
In-cluster URL of the SonarQube service the Agent Orchestrator talks to (AGENTIC_SONARQUBE_URL
and the wait-for-sonarqube init container). Always the co-deployed SonarQube service; honours the
web context path.
*/}}
{{- define "sonarqube.agent.sonarqube.url" -}}
{{- printf "http://%s:%d%s" (include "sonarqube.svcFQDN" (dict "name" (include "sonarqube.fullname" .) "ctx" .)) (int .Values.service.externalPort) (trimSuffix "/" (include "sonarqube.webcontext" .)) -}}
{{- end -}}

{{/*
Push URL the orchestrator uses to dispatch jobs to one agent runtime family.
Parameters (dict): ctx (required, the root context '.'), family (required, the runtime family name)
*/}}
{{- define "sonarqube.agentRuntime.pushUrl" -}}
{{- $port := (get (fromYaml (include "sonarqube.agentRuntimes" .ctx)) .family).port -}}
{{- printf "http://%s:%d/jobs" (include "sonarqube.svcFQDN" (dict "name" (include "sonarqube.agentRuntime.fullname" .) "ctx" .ctx)) (int $port) -}}
{{- end -}}

{{/*
Create the fully qualified name for the Agent Egress Proxy.
*/}}
{{- define "sonarqube.agentEgressProxy.fullname" -}}
{{- printf "%s-agent-egress-proxy" (include "sonarqube.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Selector labels for the Agent Egress Proxy: app: <chart>-agent-egress-proxy + release.
*/}}
{{- define "sonarqube.agentEgressProxy.selectorLabels" -}}
app: {{ include "sonarqube.name" . }}-agent-egress-proxy
release: {{ .Release.Name }}
{{- end -}}

{{/*
Name of the ServiceAccount for the Agent Egress Proxy.
Same create / pinned-name / fallback logic as the orchestrator helper above.
*/}}
{{- define "sonarqube.agentEgressProxy.serviceAccountName" -}}
{{- if .Values.agentEgressProxy.serviceAccount.create -}}
{{- default (include "sonarqube.agentEgressProxy.fullname" .) .Values.agentEgressProxy.serviceAccount.name -}}
{{- else -}}
{{- include "sonarqube.serviceAccountName" . -}}
{{- end -}}
{{- end -}}

{{/*
In-cluster URL the agent runtimes reach the Agent Egress Proxy through. Plain http:// even for
HTTPS_PROXY - the runtime talks plain HTTP to Squid itself, which then CONNECT-tunnels the actual
HTTPS session (no TLS interception between the runtime and the proxy).
*/}}
{{- define "sonarqube.agentEgressProxy.url" -}}
{{- printf "http://%s:%d" (include "sonarqube.agentEgressProxy.fullname" .) (int .Values.agentEgressProxy.port) -}}
{{- end -}}

{{/*
Whether the Agent Egress Proxy must be rendered: it has no independent enable/disable switch of
its own (unlike every other agentic component) - it auto-activates whenever hunterAgent.enabled
or remediationAgent.enabled is true, and renders nothing when both are false. Every Agent Egress
Proxy template gates on this helper instead of a values flag.
Usage: {{- if include "sonarqube.agentEgressProxy.required" . }}
*/}}
{{- define "sonarqube.agentEgressProxy.required" -}}
{{- if or .Values.hunterAgent.enabled .Values.remediationAgent.enabled -}}
true
{{- end -}}
{{- end -}}

{{/*
Scheduling block (nodeSelector/tolerations/affinity) for the Agent Egress Proxy.

Identical to sonarqube.agent.scheduling except for the affinity default: with replicaCount 2 and a
podDisruptionBudget of minAvailable 1, nothing otherwise stops both replicas landing on the same
node - which would make a single node drain sever every runtime's only egress path *and* be
blocked by the PDB. So when neither agentEgressProxy.affinity nor the global affinity is set, fall
back to a soft (preferred, not required) anti-affinity on hostname: spreads across nodes when the
cluster has them, still schedulable on a single-node cluster such as kind.

Setting agentEgressProxy.affinity replaces this default wholesale - it is not merged.
*/}}
{{- define "sonarqube.agentEgressProxy.scheduling" -}}
{{- $proxy := .Values.agentEgressProxy -}}
{{- $affinity := default .Values.affinity $proxy.affinity -}}
{{- if not $affinity -}}
{{- $affinity = fromYaml (include "sonarqube.agentEgressProxy.defaultAntiAffinity" .) -}}
{{- end -}}
{{- include "sonarqube.agent.scheduling" (dict "ctx" . "component" (dict "nodeSelector" $proxy.nodeSelector "tolerations" $proxy.tolerations "affinity" $affinity)) -}}
{{- end -}}

{{- define "sonarqube.agentEgressProxy.defaultAntiAffinity" -}}
podAntiAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        topologyKey: kubernetes.io/hostname
        labelSelector:
          matchLabels:
{{ include "sonarqube.agentEgressProxy.selectorLabels" . | indent 12 }}
{{- end -}}

{{/*
Selector labels common to both agent runtime families (family-agnostic), so a single selector can
match every runtime pod regardless of family - used by the Agent Egress Proxy's own NetworkPolicy
ingress rule.
*/}}
{{- define "sonarqube.agentRuntime.commonSelectorLabels" -}}
sonarqube.agent/component: runtime
release: {{ .Release.Name }}
{{- end -}}

{{/*
The DNS-to-kube-dns egress rule shared by every agent NetworkPolicy (runtime and proxy alike).
Output is unindented; callers should pipe through `indent`/`nindent` to place it under `egress:`.
Usage: {{ include "sonarqube.agent.dnsEgressRule" $ | indent 4 }}
*/}}
{{- define "sonarqube.agent.dnsEgressRule" -}}
- to:
    - namespaceSelector: {}
      podSelector:
        matchLabels:
          k8s-app: kube-dns
  ports:
    - port: 53
      protocol: UDP
    - port: 53
      protocol: TCP
{{- end -}}

{{/*
Parse the host:port endpoint out of jdbcOverwrite.jdbcUrl (jdbc:postgresql://host:port/db[?params]),
for the Agent Orchestrator's CORE_DB_READ_WRITE_ENDPOINT env, since it reuses SonarQube's own DB.
*/}}
{{- define "sonarqube.agent.jdbc.endpoint" -}}
{{- $stripped := regexReplaceAll "^jdbc:[a-zA-Z0-9]+://" .Values.jdbcOverwrite.jdbcUrl "" -}}
{{- (splitn "/" 2 $stripped)._0 -}}
{{- end -}}

{{/*
Parse the database name out of jdbcOverwrite.jdbcUrl (jdbc:postgresql://host:port/db[?params]),
for the Agent Orchestrator's CORE_DB_NAME env.
*/}}
{{- define "sonarqube.agent.jdbc.dbname" -}}
{{- $stripped := regexReplaceAll "^jdbc:[a-zA-Z0-9]+://" .Values.jdbcOverwrite.jdbcUrl "" -}}
{{- $rest := (splitn "/" 2 $stripped)._1 | default "" -}}
{{- regexReplaceAll "\\?.*$" $rest "" -}}
{{- end -}}

{{/*
Render the `imagePullSecrets` list items for one or more image blocks, each optionally providing
`pullSecret` (string) and/or `pullSecrets` (list), in the order given - callers combine e.g. the
application nodes' image with their own. Nil-safe against an image block the chart doesn't define
(e.g. a top-level .Values.image). Output is unindented and empty when no source has anything.
Usage: {{- $pullSecrets := include "sonarqube.agent.imagePullSecrets" (list .Values.a.image .Values.b.image) }}
*/}}
{{- define "sonarqube.agent.imagePullSecrets" -}}
{{- $refs := list -}}
{{- range . -}}
{{- $img := . | default dict -}}
{{- with $img.pullSecret -}}
{{- $refs = append $refs (dict "name" .) -}}
{{- end -}}
{{- range ($img.pullSecrets | default list) -}}
{{- $refs = append $refs . -}}
{{- end -}}
{{- end -}}
{{- with $refs -}}
{{- toYaml . -}}
{{- end -}}
{{- end -}}

{{/*
Whether to automount the ServiceAccount token: the component's own value when it creates a
dedicated ServiceAccount, else the top-level serviceAccount.automountToken.
Parameters (dict): ctx (required, the root context '.'), serviceAccount (required, the component's
own serviceAccount block)
*/}}
{{- define "sonarqube.agent.automountServiceAccountToken" -}}
{{- if .serviceAccount.create -}}
{{- .serviceAccount.automountToken -}}
{{- else -}}
{{- .ctx.Values.serviceAccount.automountToken -}}
{{- end -}}
{{- end -}}

{{/*
The SonarQube application image config: repository/tag/pullPolicy/pullSecret(s).
*/}}
{{- define "sonarqube.agent.sonarqubeImage" -}}
{{- .Values.image | toYaml -}}
{{- end -}}

{{/*
Security context for the wait-for-sonarqube init container.
*/}}
{{- define "sonarqube.agent.initContainerSecurityContext" -}}
{{- include "sonarqube.initContainerSecurityContext" . -}}
{{- end -}}

{{/*
Render nodeSelector/tolerations/affinity for an agent workload: the component's own value wins
when set (whole field, not merged), else the chart's global one.
Parameters (dict): ctx (required, the root context '.'), component (required, the component's own
values block, providing nodeSelector/tolerations/affinity)
Usage: {{- with (include "sonarqube.agent.scheduling" (dict "ctx" $ "component" .Values.agentOrchestrator)) }}
{{ . | indent 6 }}
      {{- end }}
*/}}
{{- define "sonarqube.agent.scheduling" -}}
{{- trimPrefix "\n" (include "sonarqube.agent.scheduling.render" .) -}}
{{- end -}}

{{- define "sonarqube.agent.scheduling.render" -}}
{{- $ctx := .ctx -}}
{{- $component := .component -}}
{{- with default $ctx.Values.nodeSelector $component.nodeSelector }}
nodeSelector:
{{ toYaml . | indent 2 }}
{{- end }}
{{- with default $ctx.Values.tolerations $component.tolerations }}
tolerations:
{{ toYaml . | indent 2 }}
{{- end }}
{{- with default $ctx.Values.affinity $component.affinity }}
affinity:
{{ toYaml . | indent 2 }}
{{- end }}
{{- end -}}

{{/*
Render one HTTP probe (readiness or liveness) from a probes.<kind> block.
Parameters (dict): probe (required, the probes.<kind> values block)
Usage: {{- with (include "sonarqube.agent.probe" (dict "probe" .Values.agentOrchestrator.probes.readiness)) }}
          readinessProbe:
{{ . | indent 12 }}
          {{- end }}
*/}}
{{- define "sonarqube.agent.probe" -}}
{{- if .probe.enabled -}}
httpGet:
  path: {{ .probe.path }}
  port: http
{{- with .probe.initialDelaySeconds }}
initialDelaySeconds: {{ . }}
{{- end }}
{{- with .probe.periodSeconds }}
periodSeconds: {{ . }}
{{- end }}
{{- with .probe.timeoutSeconds }}
timeoutSeconds: {{ . }}
{{- end }}
{{- with .probe.successThreshold }}
successThreshold: {{ . }}
{{- end }}
{{- with .probe.failureThreshold }}
failureThreshold: {{ . }}
{{- end }}
{{- end -}}
{{- end -}}

{{/*
Render one tcpSocket probe (readiness or liveness) for the Agent Egress Proxy - Squid has no HTTP
health endpoint worth curling, so unlike sonarqube.agent.probe above this checks the proxy port is
accepting TCP connections.
Parameters: the agentEgressProxy.probes.<kind> values block.
Usage: {{- with (include "sonarqube.agent.egressProxy.probe" .Values.agentEgressProxy.probes.readiness) }}
          readinessProbe:
{{ . | indent 12 }}
          {{- end }}
*/}}
{{- define "sonarqube.agent.egressProxy.probe" -}}
tcpSocket:
  port: http-proxy
{{- with .periodSeconds }}
periodSeconds: {{ . }}
{{- end }}
{{- with .timeoutSeconds }}
timeoutSeconds: {{ . }}
{{- end }}
{{- end -}}

{{/*
Also give these properties a real conf/sonar.properties line — the pod env vars set further down
aren't picked up by plain Configuration.get() consumers (SONAR-31416). Additive: env vars stay too,
for consumers like hunter-agent-unified-app that read them via Spring instead.
*/}}
{{- define "sonarqube.agentHealthProperties" -}}
{{- $props := dict -}}
{{- if .Values.agentOrchestrator.enabled -}}
{{- $_ := set $props "sonar.hunteragent.orchestrator.url" (include "sonarqube.agentOrchestrator.url" .) -}}
{{- $_ := set $props "sonar.remediationagent.orchestrator.url" (include "sonarqube.agentOrchestrator.url" .) -}}
{{- end -}}
{{- if .Values.vortex.enabled -}}
{{- $_ := set $props "sonar.vortex.analysis.url" (include "sonarqube.vortex.url" .) -}}
{{- end -}}
{{- toYaml $props -}}
{{- end -}}

{{/*
Merge user-provided sonarProperties with automatically generated agent health properties.
User-provided properties take precedence.
*/}}
{{- define "sonarqube.mergedSonarProperties" -}}
{{- $agentHealthProps := fromYaml (include "sonarqube.agentHealthProperties" .) | default dict -}}
{{- $userProps := .Values.sonarProperties | default dict -}}
{{- $merged := dict -}}
{{- range $key, $val := $agentHealthProps }}{{- $_ := set $merged $key $val }}{{- end -}}
{{- range $key, $val := $userProps }}{{- $_ := set $merged $key $val }}{{- end -}}
{{- toYaml $merged -}}
{{- end -}}
