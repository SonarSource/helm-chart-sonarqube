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
{{- printf "%s:%s" .Values.image.repository (tpl (include "image.tag" .) .) }}
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
Determine the k8s secretKey contrining the JDBC password
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
    items:
      - key: {{ .Values.caCerts.configMap.key }}
        path: {{ .Values.caCerts.configMap.path }}
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

{{- define "accountDeprecation" -}}
{{- $map1 := .Values.setAdminPassword -}}
{{- $map2 := .Values.account -}}

{{- $accountDeprecation := (include "deepMerge" (dict "map1" $map1 "map2" $map2)) -}}
{{- $accountDeprecation }}
{{- end -}}
