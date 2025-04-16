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
  Create a default fully qualified mysql/postgresql name.
  We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "postgresql.fullname" -}}
{{- printf "%s-%s" .Release.Name "postgresql" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
  Determine the hostname to use for PostgreSQL/mySQL.
*/}}
{{- define "postgresql.hostname" -}}
{{- if .Values.postgresql.enabled -}}
{{- printf "%s-%s" .Release.Name "postgresql" | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s" .Values.postgresql.postgresqlServer -}}
{{- end -}}
{{- end -}}

{{/*
Determine the k8s secret containing the JDBC credentials
*/}}
{{- define "jdbc.secret" -}}
{{- if .Values.postgresql.enabled -}}
  {{- if .Values.postgresql.existingSecret -}}
  {{- .Values.postgresql.existingSecret -}}
  {{- else -}}
  {{- template "postgresql.fullname" . -}}
  {{- end -}}
{{- else if or .Values.jdbcOverwrite.enabled .Values.jdbcOverwrite.enable -}}
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
{{- if and .Values.postgresql.enabled .Values.postgresql.postgresqlUsername -}}
  {{- .Values.postgresql.postgresqlUsername | quote -}}
{{- else if and (or .Values.jdbcOverwrite.enabled .Values.jdbcOverwrite.enable) .Values.jdbcOverwrite.jdbcUsername -}}
  {{- .Values.jdbcOverwrite.jdbcUsername | quote -}}
{{- else -}}
  {{- .Values.postgresql.postgresqlUsername -}}
{{- end -}}
{{- end -}}

{{/*
Determine the k8s secretKey containing the JDBC password
*/}}
{{- define "jdbc.secretPasswordKey" -}}
{{- if .Values.postgresql.enabled -}}
  {{- if and .Values.postgresql.existingSecret .Values.postgresql.existingSecretPasswordKey -}}
  {{- .Values.postgresql.existingSecretPasswordKey -}}
  {{- else -}}
  {{- "postgresql-password" -}}
  {{- end -}}
{{- else if or .Values.jdbcOverwrite.enabled .Values.jdbcOverwrite.enable -}}
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
{{- else -}}
  {{- .Values.postgresql.postgresqlPassword | b64enc | quote -}}
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

{{- define "If_any_proxy_var_exists" -}}
  {{- if or .Values.httpProxy .Values.ApplicationNodes.plugins.httpProxy .Values.httpsProxy .Values.ApplicationNodes.plugins.httpsProxy .Values.noProxy .Values.ApplicationNodes.plugins.noProxy .Values.ApplicationNodes.prometheusExporter.httpProxy .Values.ApplicationNodes.prometheusExporter.httpsProxy .Values.ApplicationNodes.prometheusExporter.noProxy }}
    {{- true -}}
  {{- else -}}
    {{- false -}}
  {{- end -}}
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
{{- else if eq (include "If_any_proxy_var_exists" .) "true" -}}
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
{{- else if eq (include "If_any_proxy_var_exists" .) "true" -}}
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