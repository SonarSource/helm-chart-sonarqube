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
