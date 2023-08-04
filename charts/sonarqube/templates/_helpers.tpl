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
{{- else if .Values.jdbcOverwrite.enable -}}
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
{{- else if and .Values.jdbcOverwrite.enable .Values.jdbcOverwrite.jdbcUsername -}}
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
{{- else if .Values.jdbcOverwrite.enable -}}
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
{{- if .Values.jdbcOverwrite.enable -}}
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
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.prometheusExporter.webBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder $tempJvm | trim | quote }}
{{- else if .Values.caCerts.enabled -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder $tempJvm | trim | quote }}
{{- else if .Values.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml %s" .Values.sonarqubeFolder (int .Values.prometheusExporter.webBeanPort) .Values.sonarqubeFolder $tempJvm | trim | quote }}
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
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder $tempJvm | trim | quote }}
{{- else if .Values.caCerts.enabled -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder $tempJvm | trim | quote }}
{{- else if .Values.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml %s" .Values.sonarqubeFolder (int .Values.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder $tempJvm | trim | quote }}
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
Common labels
*/}}
{{- define "sonarqube.labels" -}}
{{ include "sonarqube.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.extraLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Extra Annotations
*/}}
{{- define "extraAnnotations" -}}
{{- with .Values.extraAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "sonarqube.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sonarqube.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

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