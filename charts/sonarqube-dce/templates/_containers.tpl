{{/*
Define common initContainer for exporting certificates to Java keystore
{{- include "sonarqube.init.ca-certs" . | nindent 8 }}
*/}}
{{- define "sonarqube.init.ca-certs" -}}
- name: ca-certs
  image: {{ default (include "sonarqube.image" $) .Values.caCerts.image | quote }}
  command:
    - /bin/sh
    - -c
    - |
      cp -f "${JAVA_HOME}/lib/security/cacerts" /tmp/certs/cacerts
      for f in $(ls /tmp/secrets/ca-certs); do
        keytool -importcert -file "/tmp/secrets/ca-certs/${f}" -alias "${f}" -keystore /tmp/certs/cacerts -storepass changeit -trustcacerts -noprompt
      done
  {{- with .Values.initContainers.resources }}
  resources: {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.initContainers.securityContext }}
  securityContext: {{- toYaml . | nindent 4 }}
  {{- end }}
  volumeMounts:
    - mountPath: /tmp/certs
      name: "{{ include "sonarqube.fullname" . }}"
      subPath: certs
    - mountPath: /tmp/secrets/ca-certs
      name: ca-certs
{{- end -}}

{{/*
Define common initContainer for rendering sonar.properties file
{{- include "sonarqube.init.properties" (dict "context" $ "scope" .Values.searchNodes) | nindent 8 }}
{{- include "sonarqube.init.properties" (dict "context" $ "scope" .Values.ApplicationNodes) | nindent 8 }}
*/}}
{{- define "sonarqube.init.properties" -}}
{{- with .context -}}
- name: concat-properties
  image: {{ default (include "sonarqube.image" .) .Values.initContainers.image | quote }}
  command:
    - /bin/sh
    - -c
    - |
      if [ -f /tmp/props/sonar.properties ]; then
        cat /tmp/props/sonar.properties > /tmp/result/sonar.properties
      fi
      if [ -f /tmp/props/secret.properties ]; then
        cat /tmp/props/secret.properties > /tmp/result/sonar.properties
      fi
      if [ -f /tmp/props/sonar.properties -a -f /tmp/props/secret.properties ]; then
        awk 1 /tmp/props/sonar.properties /tmp/props/secret.properties > /tmp/result/sonar.properties
      fi
  {{- with .Values.initContainers.resources }}
  resources: {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.initContainers.securityContext }}
  securityContext: {{- toYaml . | nindent 4 }}
  {{- end }}
  volumeMounts:
    {{- if or $.scope.sonarProperties .Values.sonarSecretKey }}
    - mountPath: /tmp/props/sonar.properties
      name: config
      subPath: sonar.properties
    {{- end }}
    {{- if $.scope.sonarSecretProperties }}
    - mountPath: /tmp/props/secret.properties
      name: secret-config
      subPath: secret.properties
    {{- end }}
    - mountPath: /tmp/result
      name: concat-dir
{{- end -}}
{{- end -}}
