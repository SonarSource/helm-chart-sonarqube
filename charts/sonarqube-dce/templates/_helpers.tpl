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

{{- define "applicationNodes" -}}
{{- $map1 := .Values.applicationNodes -}}
{{- $map2 := .Values.ApplicationNodes -}}

{{- $applicationNodes := (include "deepMerge" (dict "map1" $map1 "map2" $map2)) -}}
{{- $applicationNodes }}
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

{{- define "accountDeprecation" -}}
{{- $map1 := .Values.setAdminPassword -}}
{{- $map2 := .Values.account -}}

{{- $accountDeprecation := (include "deepMerge" (dict "map1" $map1 "map2" $map2)) -}}
{{- $accountDeprecation }}
{{- end -}}

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
Expand the Application Image name.
*/}}
{{- define "sonarqube.image" -}}
{{- printf "%s:%s" .Values.ApplicationNodes.image.repository .Values.ApplicationNodes.image.tag }}
{{- end -}}

{{- define "searchNodes.endpoints" -}}
  {{- $replicas := int (toString (.Values.searchNodes.replicaCount)) }}
  {{- $uname := (include "sonarqube.fullname" .) }}
  {{- range $i, $e := untilStep 0 $replicas 1 -}}
    {{ $uname}}-search-{{ $i }},
  {{- end -}}
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
{{- else -}}
  {{- template "sonarqube.fullname" . -}}
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
Determine the k8s secretKey containing the JDBC password
*/}}
{{- define "jdbc.secretPasswordKey" -}}
{{- if or .Values.jdbcOverwrite.enabled .Values.jdbcOverwrite.enable -}}
  {{- if and .Values.jdbcOverwrite.jdbcSecretName .Values.jdbcOverwrite.jdbcSecretPasswordKey -}}
  {{- .Values.jdbcOverwrite.jdbcSecretPasswordKey -}}
  {{- else -}}
  {{- "jdbc-password" -}}
  {{- end -}}
{{- else -}}
  {{- "jdbc-password" -}}
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
{{- $tempJvm := .Values.ApplicationNodes.jvmOpts -}}
{{- if and .Values.ApplicationNodes.sonarProperties (hasKey (.Values.ApplicationNodes.sonarProperties) "sonar.web.javaOpts")}}
{{- $tempJvm = (get .Values.ApplicationNodes.sonarProperties "sonar.web.javaOpts") -}}
{{- else if .Values.ApplicationNodes.env -}}
{{- range $index, $val := .Values.ApplicationNodes.env -}}
{{- if eq $val.name "SONAR_WEB_JAVAOPTS" -}}
{{- $tempJvm = $val.value -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- if and .Values.caCerts.enabled .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.webBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder $tempJvm | trim }}
{{- else if .Values.caCerts.enabled -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder $tempJvm | trim }}
{{- else if .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.webBeanPort) .Values.sonarqubeFolder $tempJvm | trim }}
{{- else -}}
{{ printf "%s" $tempJvm }}
{{- end -}}
{{- end -}}

{{/*
Set sonarqube.jvmCEOpts
*/}}
{{- define "sonarqube.jvmCEOpts" -}}
{{- $tempJvm := .Values.ApplicationNodes.jvmCeOpts -}}
{{- if and .Values.ApplicationNodes.sonarProperties (hasKey (.Values.ApplicationNodes.sonarProperties) "sonar.ce.javaOpts")}}
{{- $tempJvm = (get .Values.ApplicationNodes.sonarProperties "sonar.ce.javaOpts") -}}
{{- else if .Values.ApplicationNodes.env -}}
{{- range $index, $val := .Values.ApplicationNodes.env -}}
{{- if eq $val.name "SONAR_CE_JAVAOPTS" -}}
{{- $tempJvm = $val.value -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- if and .Values.caCerts.enabled .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder $tempJvm | trim }}
{{- else if .Values.caCerts.enabled -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder $tempJvm | trim }}
{{- else if .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder $tempJvm | trim }}
{{- else -}}
{{ printf "%s" $tempJvm }}
{{- end -}}
{{- end -}}

{{/*
Set sonarqube.log.jsonoutput
Parameters:
  - ctx: The context to use (required, normally should be '.')
  - node: The node to use (required, ApplicationNodes or searchNodes)
*/}}
{{- define "sonarqube.log.jsonoutput" -}}
  {{- $node := (get .ctx.Values .node) }}
  {{- $tempJsonOutput := default "false" (get .ctx.Values.logging "jsonOutput") -}}
  {{- if and $node.sonarProperties (hasKey $node.sonarProperties "sonar.log.jsonOutput") -}}
    {{- $tempJsonOutput = (get $node.sonarProperties "sonar.log.jsonOutput") -}}
  {{- end -}}
  {{- if .ctx.Values.env -}}
    {{- range $index, $val := .ctx.Values.env -}}
      {{- if eq $val.name "SONAR_LOG_JSONOUTPUT" -}}
        {{- $tempJsonOutput = $val.value -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
  {{- if $node.env -}}
    {{- range $index, $val := $node.env -}}
      {{- if eq $val.name "SONAR_LOG_JSONOUTPUT" -}}
        {{- $tempJsonOutput = $val.value -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
  {{- printf "%s" ($tempJsonOutput | toString) -}}
{{- end -}}

{{/*
Set prometheusExporter.downloadURL
*/}}
{{- define "prometheusExporter.downloadURL" -}}
{{- if .Values.ApplicationNodes.prometheusExporter.downloadURL -}}
{{ printf "%s" .Values.ApplicationNodes.prometheusExporter.downloadURL }}
{{- else -}}
{{ printf "https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/%s/jmx_prometheus_javaagent-%s.jar" .Values.ApplicationNodes.prometheusExporter.version .Values.ApplicationNodes.prometheusExporter.version }}
{{- end -}}
{{- end -}}

{{/*
Set jwtSecret
*/}}
{{- define "jwtSecret" -}}
{{- if .Values.ApplicationNodes.existingJwtSecret -}}
{{- .Values.ApplicationNodes.existingJwtSecret -}}
{{- else -}}
{{- template "sonarqube.fullname" . -}}-jwt
{{- end -}}
{{- end -}}

{{/*
Set jwtSecret.useInternalSecret
*/}}
{{- define "jwtSecret.useInternalSecret" -}}
{{- if .Values.ApplicationNodes.existingJwtSecret -}}
false
{{- else -}}
true
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
Set searchAuthentication.useInternalKeystoreSecret when the searchNodes.searchAuthentication.keyStorePassword is provided instead of relying on an external secret (searchNodes.searchAuthentication.keyStorePasswordSecret)
*/}}
{{- define "searchAuthentication.useInternalKeystoreSecret" -}}
{{- if and .Values.searchNodes.searchAuthentication.keyStorePasswordSecret (not .Values.searchNodes.searchAuthentication.keyStorePassword) -}}
false
{{- else -}}
true
{{- end -}}
{{- end -}}

{{/*
Set search.userPassword
*/}}
{{- define "search.userPassword" -}}
{{- if .Values.searchNodes.searchAuthentication.userPasswordSecret -}}
{{- .Values.searchNodes.searchAuthentication.userPasswordSecret -}}
{{- else -}}
{{- template "sonarqube.fullname" . -}}-user-pass
{{- end -}}
{{- end -}}

{{/*
Set search.useInternalUserSecret
*/}}
{{- define "search.useInternalUserSecret" -}}
{{- if .Values.searchNodes.searchAuthentication.userPasswordSecret -}}
false
{{- else -}}
true
{{- end -}}
{{- end -}}

{{/*
set search.ksPassword
*/}}
{{- define "search.ksPassword" -}}
{{- if .Values.searchNodes.searchAuthentication.keyStorePasswordSecret -}}
{{- .Values.searchNodes.searchAuthentication.keyStorePasswordSecret -}}
{{- else -}}
{{- template "sonarqube.fullname" . -}}-keystore-pass
{{- end -}}
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

{{/*
Set sonarqube.webcontext, ensuring it starts and ends with a slash, in order to ease probes url template
*/}}
{{- define "sonarqube.webcontext" -}}
{{- $tempWebcontext := .Values.sonarWebContext -}}
{{- if and .Values.ApplicationNodes.sonarProperties (hasKey (.Values.ApplicationNodes.sonarProperties) "sonar.web.context") -}}
{{- $tempWebcontext = (get .Values.ApplicationNodes.sonarProperties "sonar.web.context") -}}
{{- end -}}
{{- range $index, $val := .Values.ApplicationNodes.env -}}
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
Set combined_app_env, ensuring we don't have any duplicates with our features and some of the user provided env vars
*/}}
{{- define "sonarqube.combined_app_env" -}}
{{- $filteredEnv := list -}}
{{- range $index,$val := .Values.ApplicationNodes.env -}}
  {{- if not (has $val.name (list "SONAR_WEB_CONTEXT" "SONAR_WEB_JAVAOPTS" "SONAR_CE_JAVAOPTS" "SONAR_LOG_JSONOUTPUT")) -}}
    {{- $filteredEnv = append $filteredEnv $val -}}
  {{- end -}}
{{- end -}}
{{- $filteredEnv = append $filteredEnv (dict "name" "SONAR_WEB_CONTEXT" "value" (include "sonarqube.webcontext" .)) -}}
{{- $filteredEnv = append $filteredEnv (dict "name" "SONAR_WEB_JAVAOPTS" "value" (include "sonarqube.jvmOpts" .)) -}}
{{- $filteredEnv = append $filteredEnv (dict "name" "SONAR_CE_JAVAOPTS" "value" (include "sonarqube.jvmCEOpts" .)) -}}
{{- $filteredEnv = append $filteredEnv (dict "name" "SONAR_LOG_JSONOUTPUT" "value" (include "sonarqube.log.jsonoutput" (dict "ctx" . "node" "ApplicationNodes"))) -}}
{{- toJson $filteredEnv -}}
{{- end -}}


{{/*
Set combined_search_env, ensuring we don't have any duplicates with our features and some of the user provided env vars
*/}}
{{- define "sonarqube.combined_search_env" -}}
{{- $filteredEnv := list -}}
{{- range $index,$val := .Values.searchNodes.env -}}
  {{- if not (has $val.name (list "SONAR_LOG_JSONOUTPUT")) -}}
    {{- $filteredEnv = append $filteredEnv $val -}}
  {{- end -}}
{{- end -}}
{{- $filteredEnv = append $filteredEnv (dict "name" "SONAR_LOG_JSONOUTPUT" "value" (include "sonarqube.log.jsonoutput" (dict "ctx" . "node" "searchNodes"))) -}}
{{- toJson $filteredEnv -}}
{{- end -}}

{{/*
Node address for exec probes: SONAR_CLUSTER_NODE_HOST (the pod IP), IPv6 bracket-wrapped for the
URL, falling back to `hostname -i` then `localhost`.
*/}}
{{- define "sonarqube.probeHostScript" -}}
host="${SONAR_CLUSTER_NODE_HOST:-$(hostname -i | awk '{print $1}')}"
host="${host:-localhost}"
case "${host}" in *:*) host="[${host}]" ;; esac
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
{{- define "sonarqube.ApplicationNodes.securityContext" -}}
{{- $adaptedApplicationNodesSecurityContext := .Values.ApplicationNodes.securityContext -}}
  {{- if .Values.OpenShift.enabled -}}
    {{- $adaptedApplicationNodesSecurityContext = omit $adaptedApplicationNodesSecurityContext "fsGroup" "runAsUser" "runAsGroup" -}}
  {{- end -}}
  {{- toYaml $adaptedApplicationNodesSecurityContext -}}
{{- end -}}


{{/*
Remove incompatible user/group values that do not work in Openshift out of the box
*/}}
{{- define "sonarqube.ApplicationNodes.containerSecurityContext" -}}
{{- $adaptedApplicationNodesContainerSecurityContext := .Values.ApplicationNodes.containerSecurityContext -}}
  {{- if .Values.OpenShift.enabled -}}
    {{- $adaptedApplicationNodesContainerSecurityContext = omit $adaptedApplicationNodesContainerSecurityContext "fsGroup" "runAsUser" "runAsGroup" -}}
  {{- end -}}
{{- toYaml $adaptedApplicationNodesContainerSecurityContext -}}
{{- end -}}

{{/*
Remove incompatible user/group values that do not work in Openshift out of the box
*/}}
{{- define "sonarqube.searchNodes.securityContext" -}}
{{- $adaptedsearchNodesSecurityContext := .Values.searchNodes.securityContext -}}
  {{- if .Values.OpenShift.enabled -}}
    {{- $adaptedsearchNodesSecurityContext = omit $adaptedsearchNodesSecurityContext "fsGroup" "runAsUser" "runAsGroup" -}}
  {{- end -}}
  {{- toYaml $adaptedsearchNodesSecurityContext -}}
{{- end -}}


{{/*
Remove incompatible user/group values that do not work in Openshift out of the box
*/}}
{{- define "sonarqube.searchNodes.containerSecurityContext" -}}
{{- $adaptedsearchNodesContainerSecurityContext := .Values.searchNodes.containerSecurityContext -}}
  {{- if .Values.OpenShift.enabled -}}
    {{- $adaptedsearchNodesContainerSecurityContext = omit $adaptedsearchNodesContainerSecurityContext "fsGroup" "runAsUser" "runAsGroup" -}}
  {{- end -}}
{{- toYaml $adaptedsearchNodesContainerSecurityContext -}}
{{- end -}}

{{/*
Remove incompatible user/group values that do not work in Openshift out of the box
*/}}
{{- define "sonarqube.initContainersSecurityContext" -}}
{{- $adaptedinitContainersSecurityContext := .Values.initContainers.securityContext -}}
  {{- if .Values.OpenShift.enabled -}}
    {{- $adaptedinitContainersSecurityContext = omit $adaptedinitContainersSecurityContext "fsGroup" "runAsUser" "runAsGroup" -}}
  {{- end -}}
{{- toYaml $adaptedinitContainersSecurityContext -}}
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
    items:
      - key: {{ .Values.caCerts.configMap.key }}
        path: {{ .Values.caCerts.configMap.path }}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Generate required Hazelcast cluster properties when custom ports are configured.
This helper automatically sets the sonar.cluster.node properties when any of the
Hazelcast ports (port, webPort, cePort) are configured, ensuring proper cluster communication.
*/}}
{{- define "sonarqube.hazelcastProperties" -}}
{{- $props := dict -}}
{{- if .Values.ApplicationNodes.port -}}
{{- $_ := set $props "sonar.cluster.node.port" (.Values.ApplicationNodes.port | toString) -}}
{{- end -}}
{{- if .Values.ApplicationNodes.webPort -}}
{{- $_ := set $props "sonar.cluster.node.web.port" (.Values.ApplicationNodes.webPort | toString) -}}
{{- end -}}
{{- if .Values.ApplicationNodes.cePort -}}
{{- $_ := set $props "sonar.cluster.node.ce.port" (.Values.ApplicationNodes.cePort | toString) -}}
{{- end -}}
{{- toYaml $props -}}
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
{{- if eq (include "sonarqube.agentic.enabled" .) "true" -}}
{{- $_ := set $props "sonar.agentic.signing.secretFile" (include "sonarqube.agentic.sqsSecretFile" .) -}}
{{- /* Inbound and outbound are two different settings. The secret above is what SQS *verifies*
       with: it derives remediation-to-sqs and agentic-shared from it in-process, which is why it
       mounts neither. This one is what it *signs* with when it calls the orchestrator itself
       (health, supported models, the hunter/remediation web configs) - a path, since the key is
       mounted rather than derived. Unset leaves those calls unsigned. */}}
{{- $_ := set $props "sonar.agentic.orchestrator.signingKeyPath" (printf "%s/agentic-shared" (include "sonarqube.agentic.sqsKeyDir" .)) -}}
{{- end -}}
{{- toYaml $props -}}
{{- end -}}

{{/*
Merge user-provided sonarProperties with automatically generated properties (Hazelcast, agent health).
User-provided properties take precedence.
*/}}
{{- define "sonarqube.mergedSonarProperties" -}}
{{- $hazelcastProps := fromYaml (include "sonarqube.hazelcastProperties" .) | default dict -}}
{{- $agentHealthProps := fromYaml (include "sonarqube.agentHealthProperties" .) | default dict -}}
{{- $userProps := .Values.ApplicationNodes.sonarProperties | default dict -}}
{{- $merged := dict -}}
{{- /* Start with automatically generated properties */}}
{{- range $key, $val := $hazelcastProps -}}
  {{- $_ := set $merged $key $val -}}
{{- end -}}
{{- range $key, $val := $agentHealthProps -}}
  {{- $_ := set $merged $key $val -}}
{{- end -}}
{{- /* User properties override automatic ones */}}
{{- range $key, $val := $userProps -}}
  {{- $_ := set $merged $key $val -}}
{{- end -}}
{{- toYaml $merged -}}
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
Whether any agentic component that needs derived signing/verification keys is enabled: hunterAgent,
remediationAgent, or vortex. Broader than sonarqube.agentEgressProxy.required (which excludes
vortex - vortex doesn't route through the egress proxy) - gates the key-derivation hook Job, its
RBAC/ServiceAccount, the fail-closed agenticSigningSecret validation, and the "agentic-shared" key
mount everywhere it's needed.
Usage: {{- if include "sonarqube.agentic.enabled" . }}
*/}}
{{- define "sonarqube.agentic.enabled" -}}
{{- if or .Values.hunterAgent.enabled .Values.remediationAgent.enabled .Values.vortex.enabled -}}
true
{{- end -}}
{{- end -}}

{{/*
Name of the per-consumer derived signing/verification key Secret produced by the key-derivation
hook Job.
Parameters (dict): ctx (required, the root context '.'), consumer (required, one of "orchestrator",
"hunter", "remediation", "sqs", "vortex").
*/}}
{{- define "sonarqube.agentic.keySecretName" -}}
{{- /* Truncate the fullname prefix, not the consumer suffix. Truncating the whole thing at 63
       would collapse every consumer onto one name once the fullname passes 49 characters (a
       release name of ~39 is well inside the fullname budget): the hook's five write_secret calls
       would overwrite each other, and every pod's items: projection would then ask the surviving
       Secret for labels it doesn't hold, wedging them in ContainerCreating. 36 + len
       "-agentic-keys-" + the longest consumer ("remediation") is 61, so the suffix always
       survives intact. */}}
{{- printf "%s-agentic-keys-%s" (include "sonarqube.fullname" .ctx | trunc 36 | trimSuffix "-") .consumer | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Which derived key labels a given consumer needs mounted. Single source of truth: the
key-derivation hook Job derives and writes exactly these per consumer, and every pod template
projects exactly these out of its own Secret - so the Secret's data keys and the volume's
items: projection can never drift apart (a projected key that doesn't exist wedges the pod in
ContainerCreating).

Least privilege: each pod only ever holds the keys for the hops it actually participates in.

  orchestrator -> orchestrator-to-hunter (hunter), orchestrator-to-remediation (remediation),
                  agentic-shared (SQS <-> orchestrator)
  hunter       -> orchestrator-to-hunter
  remediation  -> orchestrator-to-remediation, remediation-to-sqs
  sqs          -> agentic-shared
  vortex       -> agentic-shared (Vortex -> SQS)

Note SQS does *not* mount remediation-to-sqs even though it verifies that hop: it has the
instance secret itself (sonarqube.agentic.sqsSecretFile) and re-derives the key in-process.

Parameters (dict): ctx (required, the root context '.'), consumer (required).
Returns a YAML list; "[]" when the consumer needs none.
*/}}
{{- define "sonarqube.agentic.keyLabels" -}}
{{- $v := .ctx.Values -}}
{{- $any := eq (include "sonarqube.agentic.enabled" .ctx) "true" -}}
{{- $labels := list -}}
{{- if eq .consumer "orchestrator" -}}
{{- if $v.hunterAgent.enabled }}{{- $labels = append $labels "orchestrator-to-hunter" }}{{- end }}
{{- if $v.remediationAgent.enabled }}{{- $labels = append $labels "orchestrator-to-remediation" }}{{- end }}
{{- if and $any $v.agentOrchestrator.enabled }}{{- $labels = append $labels "agentic-shared" }}{{- end }}
{{- else if eq .consumer "hunter" -}}
{{- if $v.hunterAgent.enabled }}{{- $labels = append $labels "orchestrator-to-hunter" }}{{- end }}
{{- else if eq .consumer "remediation" -}}
{{- if $v.remediationAgent.enabled }}{{- $labels = concat $labels (list "orchestrator-to-remediation" "remediation-to-sqs") }}{{- end }}
{{- else if eq .consumer "sqs" -}}
{{- if $any }}{{- $labels = append $labels "agentic-shared" }}{{- end }}
{{- else if eq .consumer "vortex" -}}
{{- if and $any $v.vortex.enabled }}{{- $labels = append $labels "agentic-shared" }}{{- end }}
{{- end -}}
{{- toYaml $labels -}}
{{- end -}}

{{/*
The union of every derived key label any enabled consumer needs - i.e. exactly what the hook Job
has to run derive-keys.sh for. Returns a YAML list.
*/}}
{{- define "sonarqube.agentic.allKeyLabels" -}}
{{- $all := list -}}
{{- range $consumer := (list "orchestrator" "hunter" "remediation" "sqs" "vortex") -}}
{{- $all = concat $all (fromYamlArray (include "sonarqube.agentic.keyLabels" (dict "ctx" $ "consumer" $consumer))) -}}
{{- end -}}
{{- toYaml (uniq $all) -}}
{{- end -}}

{{/*
Whether a given consumer mounts any derived keys at all. Emits "true" or "".
Parameters (dict): ctx, consumer.
*/}}
{{- define "sonarqube.agentic.hasKeys" -}}
{{- if (fromYamlArray (include "sonarqube.agentic.keyLabels" .)) -}}
true
{{- end -}}
{{- end -}}

{{/*
Where the derived keys are mounted. Agent-side components (orchestrator, runtimes, Vortex) share
one neutral path since they're three different base images; SQS keeps its material under
sonarqubeFolder alongside the existing secret/ mount.
*/}}
{{- define "sonarqube.agentic.keyDir" -}}/etc/agentic/keys{{- end -}}
{{- define "sonarqube.agentic.sqsKeyDir" -}}{{ .Values.sonarqubeFolder }}/agentic-keys{{- end -}}

{{/*
The key directory for one consumer - sqsKeyDir on the application nodes, the neutral keyDir
everywhere else.
Parameters (dict): ctx, consumer.
*/}}
{{- define "sonarqube.agentic.consumerKeyDir" -}}
{{- if eq .consumer "sqs" -}}
{{- include "sonarqube.agentic.sqsKeyDir" .ctx -}}
{{- else -}}
{{- include "sonarqube.agentic.keyDir" .ctx -}}
{{- end -}}
{{- end -}}

{{/*
Env vars pointing each consumer at the individual key *files* it mounts. The contract is one
variable per key path, and the same key is named differently on either side of a hop - it is the
consumer's role in the hop that picks the name, not the key:

  orchestrator  orchestrator-to-hunter       -> AGENTIC_HUNTER_RUNTIME_SIGNING_KEY_PATH
                orchestrator-to-remediation  -> AGENTIC_REMEDIATION_RUNTIME_SIGNING_KEY_PATH
                agentic-shared               -> AGENTIC_SONARQUBE_SIGNING_KEY_PATH
  hunter        orchestrator-to-hunter       -> AGENTIC_VERIFY_KEY_PATH
  remediation   orchestrator-to-remediation  -> AGENTIC_VERIFY_KEY_PATH
                remediation-to-sqs           -> REMEDIATION_AGENTIC_SIGNING_KEY_PATH
  vortex        agentic-shared               -> AGENTIC_ORCHESTRATOR_SIGNING_KEY_PATH

AGENTIC_VERIFY_KEY_PATH is reused across the two runtimes without colliding: each pod mounts
exactly one verification key. Because the name is the same on both, it is paired with
AGENTIC_VERIFY_KEY_ID, whose value is the label itself (orchestrator-to-hunter or
orchestrator-to-remediation) - that is what tells the runtime which hop the key belongs to.

SQS deliberately has no entry here even though it mounts agentic-shared too (see keyLabels): what
the JVM reads to sign its own outbound calls to the orchestrator is the
sonar.agentic.orchestrator.signingKeyPath property (SONAR-31996), set by
sonarqube.agentHealthProperties - Configuration.get() does not fall back to plain env vars
(SONAR-31416), so an env var here would just go unread. Vortex has no properties mechanism (it's a
standalone Deployment, not a SonarQube JVM process), so AGENTIC_ORCHESTRATOR_SIGNING_KEY_PATH
remains its only way to find the key.

AGENTIC_SONARQUBE_SIGNING_KEY_PATH fills in the previously-unnamed use of agentic-shared on the
orchestrator: the file was already mounted (sonarqube.agentic.keyLabels), it just had no variable
pointing at it.

Driven off sonarqube.agentic.keyLabels, so a variable can only appear when the file behind it is
actually projected into the pod. Returns a YAML list of env entries; "[]" when there are none.
Parameters (dict): ctx, consumer.
*/}}
{{- define "sonarqube.agentic.keyPathEnv" -}}
{{- $names := dict
  "orchestrator" (dict "orchestrator-to-hunter" "AGENTIC_HUNTER_RUNTIME_SIGNING_KEY_PATH" "orchestrator-to-remediation" "AGENTIC_REMEDIATION_RUNTIME_SIGNING_KEY_PATH" "agentic-shared" "AGENTIC_SONARQUBE_SIGNING_KEY_PATH")
  "hunter" (dict "orchestrator-to-hunter" "AGENTIC_VERIFY_KEY_PATH")
  "remediation" (dict "orchestrator-to-remediation" "AGENTIC_VERIFY_KEY_PATH" "remediation-to-sqs" "REMEDIATION_AGENTIC_SIGNING_KEY_PATH")
  "vortex" (dict "agentic-shared" "AGENTIC_ORCHESTRATOR_SIGNING_KEY_PATH")
-}}
{{- /* Path variables that need a companion variable naming *which* key the file holds, keyed by
       the path variable they accompany. Only the runtimes need one: AGENTIC_VERIFY_KEY_PATH is
       the same name on both, so the label is what tells the runtime which hop it is verifying. */}}
{{- $idNames := dict "AGENTIC_VERIFY_KEY_PATH" "AGENTIC_VERIFY_KEY_ID" -}}
{{- $forConsumer := get $names .consumer | default dict -}}
{{- $dir := include "sonarqube.agentic.consumerKeyDir" . -}}
{{- $env := list -}}
{{- range $label := (fromYamlArray (include "sonarqube.agentic.keyLabels" .)) -}}
{{- with (get $forConsumer $label) -}}
{{- $env = append $env (dict "name" . "value" (printf "%s/%s" $dir $label)) -}}
{{- with (get $idNames .) -}}
{{- $env = append $env (dict "name" . "value" $label) -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- toYaml $env -}}
{{- end -}}

{{/*
Where the operator-provided instance secret is mounted on SQS, and the file it's projected to.
Projected to a fixed filename so the mount path doesn't change with agenticSigningSecret.key.
*/}}
{{- define "sonarqube.agentic.sqsSecretDir" -}}{{ .Values.sonarqubeFolder }}/agentic-secret{{- end -}}
{{- define "sonarqube.agentic.sqsSecretFile" -}}{{ include "sonarqube.agentic.sqsSecretDir" . }}/instance-secret{{- end -}}

{{/*
Key within .Values.agenticSigningSecret.existingSecret holding the instance secret.
*/}}
{{- define "sonarqube.agentic.signingSecretKey" -}}
{{- .Values.agenticSigningSecret.key | default "instance-secret" -}}
{{- end -}}

{{/*
volumeMount for a consumer's derived-key Secret. Renders nothing when the consumer needs no keys.
Parameters (dict): ctx, consumer, mountPath.
*/}}
{{- define "sonarqube.agentic.keyVolumeMount" -}}
{{- if eq (include "sonarqube.agentic.hasKeys" (dict "ctx" .ctx "consumer" .consumer)) "true" }}
- name: agentic-keys
  mountPath: {{ .mountPath }}
  readOnly: true
{{- end }}
{{- end -}}

{{/*
volume for a consumer's derived-key Secret, projecting only that consumer's labels (one file per
label, named after the label). Renders nothing when the consumer needs no keys.
Parameters (dict): ctx, consumer.
*/}}
{{- define "sonarqube.agentic.keyVolume" -}}
{{- $labels := fromYamlArray (include "sonarqube.agentic.keyLabels" (dict "ctx" .ctx "consumer" .consumer)) }}
{{- if $labels }}
- name: agentic-keys
  secret:
    secretName: {{ include "sonarqube.agentic.keySecretName" (dict "ctx" .ctx "consumer" .consumer) }}
    items:
    {{- range $label := $labels }}
    - key: {{ $label }}
      path: {{ $label }}
    {{- end }}
{{- end }}
{{- end -}}

{{/*
Whether the key-derivation hook Job (and its ServiceAccount/RBAC) renders. Emits "true" or "".
It has no enable switch of its own beyond the explicit agentKeyDerivation.enabled opt-out: it
follows the components that consume its output. validation.yaml guarantees agenticSigningSecret
is set whenever any of them is enabled, so the check here is belt-and-braces for `helm template`
runs that bypass validation.
*/}}
{{- define "sonarqube.agentKeyDerivation.required" -}}
{{- if and .Values.agentKeyDerivation.enabled (eq (include "sonarqube.agentic.enabled" .) "true") .Values.agenticSigningSecret .Values.agenticSigningSecret.existingSecret -}}
true
{{- end -}}
{{- end -}}

{{- define "sonarqube.agentKeyDerivation.fullname" -}}
{{- /* Truncate the fullname prefix, not the "-agent-key-derivation" suffix (21 chars) - the same
       collision sonarqube.agentic.keySecretName guards against. Truncating the concatenation at 63
       would render this exactly as sonarqube.fullname once that name reaches 63 characters,
       colliding the Job/Role/RoleBinding/ServiceAccount with the chart's own primary resources. */}}
{{- printf "%s-agent-key-derivation" (include "sonarqube.fullname" . | trunc 41 | trimSuffix "-") | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "sonarqube.agentKeyDerivation.serviceAccountName" -}}
{{- if .Values.agentKeyDerivation.serviceAccount.create -}}
{{- default (include "sonarqube.agentKeyDerivation.fullname" .) .Values.agentKeyDerivation.serviceAccount.name -}}
{{- else -}}
{{- /* An explicit name still wins when create is false: that's the "I bound the Role myself"
       path, and it's the one the validation error points operators at. Without the override it
       would fall through to the release's top-level ServiceAccount - "default" unless that one is
       named too - and the same failure would come straight back. */ -}}
{{- default (include "sonarqube.serviceAccountName" .) .Values.agentKeyDerivation.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
The image the hook Job runs. What it actually needs is *an* image carrying /derive-keys.sh, which
today is the Agent Orchestrator image (baked in there so air-gapped installs don't need a second
pull, per EA-791/ADR-10) - hence the fallback to agentOrchestrator.image. It is a separate value
rather than a hard reference to agentOrchestrator.image because vortex.enabled alone still needs
derived keys while not otherwise deploying, or even pulling, the orchestrator: such an install sets
agentKeyDerivation.image.* and keeps agentOrchestrator untouched.
Returns the image block as a dict, so callers can read .repository/.tag/.pullPolicy/.pullSecrets.
*/}}
{{- define "sonarqube.agentKeyDerivation.image" -}}
{{- $own := .Values.agentKeyDerivation.image | default dict -}}
{{- if $own.repository -}}
{{- toYaml $own -}}
{{- else -}}
{{- toYaml (.Values.agentOrchestrator.image | default dict) -}}
{{- end -}}
{{- end -}}

{{/*
The repository of the image above, or "" when neither it nor agentOrchestrator.image sets one.
validation.yaml fails closed on the empty case.
*/}}
{{- define "sonarqube.agentKeyDerivation.imageRepository" -}}
{{- (fromYaml (include "sonarqube.agentKeyDerivation.image" .)).repository | default "" -}}
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
{{- (fromYaml (include "applicationNodes" .)).image | toYaml -}}
{{- end -}}

{{/*
Security context for the wait-for-sonarqube init container.
*/}}
{{- define "sonarqube.agent.initContainerSecurityContext" -}}
{{- include "sonarqube.initContainersSecurityContext" . -}}
{{- end -}}

{{/*
Container/pod securityContext for an agent workload, omitting runAsUser/runAsGroup/fsGroup under
OpenShift's restricted-v2 SCC (same reasoning as sonarqube.containerSecurityContext) - unlike that
helper, this one is parameterized per-component since agent workloads hardcode different UIDs.
Parameters (dict): ctx (required, the root context '.'), securityContext (required, the
component's own securityContext block)
*/}}
{{- define "sonarqube.agent.containerSecurityContext" -}}
{{- $securityContext := .securityContext -}}
{{- if .ctx.Values.OpenShift.enabled -}}
{{- $securityContext = omit $securityContext "runAsUser" "runAsGroup" "fsGroup" -}}
{{- end -}}
{{- toYaml $securityContext -}}
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
