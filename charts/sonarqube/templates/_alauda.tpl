{{- define "sonarqube.image" -}}
{{- printf "%s/%s:%s" .Values.global.registry.address .Values.global.images.sonarqube.repository .Values.global.images.sonarqube.tag -}}
{{- end -}}

{{- define "initSysctl.image" -}}
{{- printf "%s/%s:%s" .Values.global.registry.address .Values.global.images.busybox.repository .Values.global.images.busybox.tag -}}
{{- end -}}

{{- define "plugins.image" -}}
{{- printf "%s/%s:%s" .Values.global.registry.address .Values.global.images.pluginPackage.repository .Values.global.images.pluginPackage.tag -}}
{{- end -}}

{{- define "busybox.image" -}}
{{- printf "%s/%s:%s" .Values.global.registry.address .Values.global.images.busybox.repository .Values.global.images.busybox.tag -}}
{{- end -}}

{{- define "preupgrade.name" -}}
{{ (printf "%s-pre-migration" (include "sonarqube.fullname" .) | trunc 63) | trimSuffix "-" }}
{{- end -}}