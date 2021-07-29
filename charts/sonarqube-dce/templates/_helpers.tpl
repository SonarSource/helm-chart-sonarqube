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
Set postgresql.secret
*/}}
{{- define "postgresql.secret" -}}
{{- if .Values.postgresql.existingSecret -}}
{{- .Values.postgresql.existingSecret -}}
{{- else if .Values.postgresql.enabled -}}
{{- template "postgresql.fullname" . -}}
{{- else -}}
{{- template "sonarqube.fullname" . -}}
{{- end -}}
{{- end -}}

{{/*
Set postgresql.secretKey
*/}}
{{- define "postgresql.secretPasswordKey" -}}
{{- if and .Values.postgresql.existingSecretPasswordKey .Values.postgresql.existingSecret -}}
{{- .Values.postgresql.existingSecretPasswordKey -}}
{{- else -}}
{{- "postgresql-password" -}}
{{- end -}}
{{- end -}}

{{/*
Set postgresql.useInternalSecret
*/}}
{{- define "postgresql.useInternalSecret" -}}
{{- if or .Values.postgresql.enabled .Values.postgresql.existingSecret -}}
false
{{- else -}}
true
{{- end -}}
{{- end -}}

{{/*
Set sonarqube.jvmOpts
*/}}
{{- define "sonarqube.jvmOpts" -}}
{{- if and .Values.caCerts .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.webBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder .Values.ApplicationNodes.jvmOpts | trim | quote }}
{{- else if .Values.caCerts -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder .Values.ApplicationNodes.jvmOpts | trim | quote }}
{{- else if .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.webBeanPort) .Values.sonarqubeFolder .Values.ApplicationNodes.jvmOpts | trim | quote }}
{{- else -}}
{{ printf "%s" .Values.ApplicationNodes.jvmOpts }}
{{- end -}}
{{- end -}}

{{/*
Set sonarqube.jvmCEOpts
*/}}
{{- define "sonarqube.jvmCEOpts" -}}
{{- if and .Values.caCerts .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder .Values.ApplicationNodes.jvmCeOpts | trim | quote }}
{{- else if .Values.caCerts -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder .Values.ApplicationNodes.jvmCeOpts | trim | quote }}
{{- else if .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder .Values.ApplicationNodes.jvmCeOpts | trim | quote }}
{{- else -}}
{{ printf "%s" .Values.ApplicationNodes.jvmCeOpts }}
{{- end -}}
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
