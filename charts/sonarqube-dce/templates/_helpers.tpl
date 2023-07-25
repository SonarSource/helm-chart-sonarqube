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
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.webBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder $tempJvm | trim | quote }}
{{- else if .Values.caCerts.enabled -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder $tempJvm | trim | quote }}
{{- else if .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-config.yaml %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.webBeanPort) .Values.sonarqubeFolder $tempJvm | trim | quote }}
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
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml -Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder .Values.sonarqubeFolder $tempJvm | trim | quote }}
{{- else if .Values.caCerts.enabled -}}
{{ printf "-Djavax.net.ssl.trustStore=%s/certs/cacerts %s" .Values.sonarqubeFolder $tempJvm | trim | quote }}
{{- else if .Values.ApplicationNodes.prometheusExporter.enabled -}}
{{ printf "-javaagent:%s/data/jmx_prometheus_javaagent.jar=%d:%s/conf/prometheus-ce-config.yaml %s" .Values.sonarqubeFolder (int .Values.ApplicationNodes.prometheusExporter.ceBeanPort) .Values.sonarqubeFolder $tempJvm | trim | quote }}
{{- else -}}
{{ printf "%s" $tempJvm }}
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
Set search.useInternalKeystoreSecret
*/}}
{{- define "search.useInternalKeystoreSecret" -}}
{{- if .Values.searchNodes.searchAuthentication.keyStorePasswordSecret -}}
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