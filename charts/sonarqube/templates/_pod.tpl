{{- define "sonarqube.pod" -}}
metadata:
  annotations:
    checksum/config: {{ include (print $.Template.BasePath "/config.yaml") . | sha256sum }}
    {{- if and .Values.persistence.enabled .Values.initFs.enabled (not .Values.OpenShift.enabled) }}
    checksum/init-fs: {{ include (print $.Template.BasePath "/init-fs.yaml") . | sha256sum }}
    {{- end }}
    {{- if and .Values.initSysctl.enabled (not .Values.OpenShift.enabled) }}
    checksum/init-sysctl: {{ include (print $.Template.BasePath "/init-sysctl.yaml") . | sha256sum }}
    {{- end }}
    checksum/plugins: {{ include (print $.Template.BasePath "/install-plugins.yaml") . | sha256sum }}
    checksum/secret: {{ include (print $.Template.BasePath "/secret.yaml") . | sha256sum }}
    {{- if .Values.prometheusExporter.enabled }}
    checksum/prometheus-config: {{ include (print $.Template.BasePath "/prometheus-config.yaml") . | sha256sum }}
    checksum/prometheus-ce-config: {{ include (print $.Template.BasePath "/prometheus-ce-config.yaml") . | sha256sum }}
    {{- end }}
    {{- with .Values.annotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  labels:
    {{- include "sonarqube.selectorLabels" . | nindent 4 }}
    {{- with .Values.podLabels }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
spec:
  automountServiceAccountToken: {{ .Values.serviceAccount.automountToken }}
  {{- with .Values.schedulerName }}
  schedulerName: {{ . }}
  {{- end }}
  {{- with (include "sonarqube.securityContext" .) }}
  securityContext: {{- . | nindent 4 }}
  {{- end }}
  {{- if or .Values.image.pullSecrets .Values.image.pullSecret }}
  imagePullSecrets:
    {{- if .Values.image.pullSecret }}
    - name: {{ .Values.image.pullSecret }}
    {{- end }}
    {{- with .Values.image.pullSecrets }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- end }}
  initContainers:
    {{- if .Values.extraInitContainers }}
    {{- toYaml .Values.extraInitContainers | nindent 4 }}
    {{- end }}
    {{- if .Values.postgresql.enabled }}
    - name: "wait-for-db"
      image: {{ default (include "sonarqube.image" $) .Values.initContainers.image }}
      imagePullPolicy: {{ .Values.image.pullPolicy  }}
      {{- with (include "sonarqube.initContainerSecurityContext" .) }}
      securityContext: {{- . | nindent 8 }}
      {{- end }}
      {{- with .Values.initContainers.resources }}
      resources: {{- toYaml . | nindent 8 }}
      {{- end }}
      command: ["/bin/bash", "-c"]
      args: ['set -o pipefail;for i in {1..200};do (echo > /dev/tcp/{{ .Release.Name }}-postgresql/5432) && exit 0; sleep 2;done; exit 1']
    {{- end }}
    {{- if .Values.caCerts.enabled }}
    - name: ca-certs
      image: {{ default (include "sonarqube.image" $) .Values.caCerts.image }}
      imagePullPolicy: {{ .Values.image.pullPolicy  }}
      command: ["sh"]
      args: ["-c", "cp -f \"${JAVA_HOME}/lib/security/cacerts\" /tmp/certs/cacerts; if [ \"$(ls /tmp/secrets/ca-certs)\" ]; then for f in /tmp/secrets/ca-certs/*; do keytool -importcert -file \"${f}\" -alias \"$(basename \"${f}\")\" -keystore /tmp/certs/cacerts -storepass changeit -trustcacerts -noprompt; done; fi;"]
      {{- with (include "sonarqube.initContainerSecurityContext" .) }}
      securityContext: {{- . | nindent 8 }}
      {{- end }}
      {{- with .Values.initContainers.resources }}
      resources: {{- toYaml . | nindent 8 }}
      {{- end }}
      volumeMounts:
        - mountPath: /tmp/certs
          name: sonarqube
          subPath: certs
        - mountPath: /tmp/secrets/ca-certs
          name: ca-certs
      env:
        {{- (include "sonarqube.combined_env" . | fromJsonArray) | toYaml | trim | nindent 8 }}
    {{- end }}
    {{- if and (or .Values.initSysctl.enabled .Values.elasticsearch.configureNode) (not .Values.OpenShift.enabled) }}
    - name: init-sysctl
      image: {{ default (include "sonarqube.image" $) .Values.initSysctl.image }}
      imagePullPolicy: {{ .Values.image.pullPolicy  }}
      {{- with (default (fromYaml (include "sonarqube.initContainerSecurityContext" .)) (.Values.initSysctl.securityContext )) }}
      securityContext: {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with (default .Values.initContainers.resources .Values.initSysctl.resources) }}
      resources:  {{- toYaml . | nindent 8 }}
      {{- end }}
      command: ["/bin/bash", "-e", "/tmp/scripts/init_sysctl.sh"]
      volumeMounts:
        - name: init-sysctl
          mountPath: /tmp/scripts/
      env:
        {{- (include "sonarqube.combined_env" . | fromJsonArray) | toYaml | trim | nindent 8 }}
    {{- end }}
    {{- if or .Values.sonarProperties .Values.sonarSecretProperties .Values.sonarSecretKey (not .Values.elasticsearch.bootstrapChecks) }}
    - name: concat-properties
      image: {{ default (include "sonarqube.image" $) .Values.initContainers.image }}
      imagePullPolicy: {{ .Values.image.pullPolicy  }}
      command:
        - sh
        - -c
        - |
          #!/bin/sh
          if [ -f /tmp/props/sonar.properties ]; then
            cat /tmp/props/sonar.properties > /tmp/result/sonar.properties
          fi
          if [ -f /tmp/props/secret.properties ]; then
            cat /tmp/props/secret.properties > /tmp/result/sonar.properties
          fi
          if [ -f /tmp/props/sonar.properties -a -f /tmp/props/secret.properties ]; then
            awk 1 /tmp/props/sonar.properties /tmp/props/secret.properties > /tmp/result/sonar.properties
          fi
      volumeMounts:
        - mountPath: /tmp/result
          name: concat-dir
        {{- if or .Values.sonarProperties .Values.sonarSecretKey (not .Values.elasticsearch.bootstrapChecks) }}
        - mountPath: /tmp/props/sonar.properties
          name: config
          subPath: sonar.properties
        {{- end }}
        {{- if .Values.sonarSecretProperties }}
        - mountPath: /tmp/props/secret.properties
          name: secret-config
          subPath: secret.properties
        {{- end }}
      {{- with (include "sonarqube.initContainerSecurityContext" .) }}
      securityContext: {{- . | nindent 8 }}
      {{- end }}
      {{- with .Values.initContainers.resources }}
      resources: {{- toYaml . | nindent 8 }}
      {{- end }}
      env:
        {{- (include "sonarqube.combined_env" . | fromJsonArray) | toYaml | trim | nindent 8 }}
    {{- end }}
    {{- if .Values.prometheusExporter.enabled }}
    - name: inject-prometheus-exporter
      image: {{ default (include "sonarqube.image" $) .Values.prometheusExporter.image }}
      imagePullPolicy: {{ .Values.image.pullPolicy  }}
      {{- with (default (fromYaml (include "sonarqube.initContainerSecurityContext" .)) .Values.prometheusExporter.securityContext) }}
      securityContext: {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with (default .Values.initContainers.resources .Values.prometheusExporter.resources)}}
      resources: {{- toYaml . | nindent 8 }}
      {{- end }}
      command: ["/bin/sh", "-c"]
      args: ["curl -s '{{ include "prometheusExporter.downloadURL" . }}' {{ if $.Values.prometheusExporter.noCheckCertificate }}--insecure{{ end }} --output /data/jmx_prometheus_javaagent.jar -v"]
      volumeMounts:
        - mountPath: /data
          name: sonarqube
          subPath: data
      env:
        {{- with (include "sonarqube.prometheusExporterProxy.env" .) }}
        {{- . | nindent 8 }}
        {{- end }}
        {{- (include "sonarqube.combined_env" . | fromJsonArray) | toYaml | trim | nindent 8 }}
    {{- end }}
    {{- if and .Values.persistence.enabled .Values.initFs.enabled (not .Values.OpenShift.enabled) }}
    - name: init-fs
      image: {{ default (include "sonarqube.image" $) .Values.initFs.image }}
      imagePullPolicy: {{ .Values.image.pullPolicy  }}
      {{- with (default (fromYaml (include "sonarqube.initContainerSecurityContext" .)) .Values.initFs.securityContext) }}
      securityContext: {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with (default .Values.initContainers.resources .Values.initFs.resources) }}
      resources: {{- toYaml . | nindent 8 }}
      {{- end }}
      command: ["sh", "-e", "/tmp/scripts/init_fs.sh"]
      volumeMounts:
        - name: init-fs
          mountPath: /tmp/scripts/
        - mountPath: {{ .Values.sonarqubeFolder }}/data
          name: sonarqube
          subPath: data
        - mountPath: {{ .Values.sonarqubeFolder }}/temp
          name: sonarqube
          subPath: temp
        - mountPath: {{ .Values.sonarqubeFolder }}/logs
          name: sonarqube
          subPath: logs
        - mountPath: /tmp
          name: tmp-dir
        {{- if .Values.caCerts.enabled }}
        - mountPath: {{ .Values.sonarqubeFolder }}/certs
          name: sonarqube
          subPath: certs
        {{- end }}
        - mountPath: {{ .Values.sonarqubeFolder }}/extensions
          name: sonarqube
          subPath: extensions
        {{- with .Values.persistence.mounts }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
    {{- end }}
    {{- if .Values.plugins.install }}
    - name: install-plugins
      image: {{ default (include "sonarqube.image" $) .Values.plugins.image }}
      imagePullPolicy: {{ .Values.image.pullPolicy  }}
      command: ["sh", "-e", "/tmp/scripts/install_plugins.sh"]
      {{- with (default (fromYaml (include "sonarqube.initContainerSecurityContext" .)) .Values.plugins.securityContext) }}
      securityContext: {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with (default .Values.initContainers.resources .Values.plugins.resource) }}
      resources: {{- toYaml . | nindent 8 }}
      {{- end }}
      volumeMounts:
        - mountPath: {{ .Values.sonarqubeFolder }}/extensions/plugins
          name: sonarqube
          subPath: extensions/plugins
        - name: install-plugins
          mountPath: /tmp/scripts/
        {{- if .Values.plugins.netrcCreds }}
        - name: plugins-netrc-file
          mountPath: /root
        {{- end }}
      env:
        {{- with (include "sonarqube.install-plugins-proxy.env" .) }}
        {{- . | nindent 8 }}
        {{- end }}
        {{- (include "sonarqube.combined_env" . | fromJsonArray) | toYaml | trim | nindent 8 }}
    {{- end }}
    {{- if and .Values.jdbcOverwrite.oracleJdbcDriver .Values.jdbcOverwrite.oracleJdbcDriver.url }}
    - name: install-oracle-jdbc-driver
      image: {{ default (include "sonarqube.image" $) .Values.initContainers.image }}
      imagePullPolicy: {{ .Values.image.pullPolicy  }}
      command: ["sh", "-e", "/tmp/scripts/install_oracle_jdbc_driver.sh"]
      {{- with (default (fromYaml (include "sonarqube.initContainerSecurityContext" .)) .Values.initContainers.securityContext) }}
      securityContext: {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.initContainers.resources }}
      resources: {{- toYaml . | nindent 8 }}
      {{- end }}
      volumeMounts:
        - mountPath: {{ .Values.sonarqubeFolder }}/extensions/jdbc-driver/oracle
          name: sonarqube
          subPath: extensions/jdbc-driver/oracle 
        - name: install-oracle-jdbc-driver
          mountPath: /tmp/scripts/
        {{- if .Values.jdbcOverwrite.oracleJdbcDriver.netrcCreds }}
        - name: oracle-jdbc-driver-netrc-file
          mountPath: /root
        {{- end }}
      {{- if .Values.caCerts.enabled }} 
        - mountPath: /tmp/secrets/ca-certs
          name: ca-certs
      {{- end }}
      env:
        {{- (include "sonarqube.combined_env" . | fromJsonArray) | toYaml | trim | nindent 8 }}
    {{- end }}
  containers:
    {{- with .Values.extraContainers }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
    - name: {{ .Chart.Name }}
      image: {{ include "sonarqube.image" . }}
      imagePullPolicy: {{ .Values.image.pullPolicy }}
      ports:
        - name: http
          containerPort: {{ .Values.service.internalPort }}
          protocol: TCP
        {{- if .Values.prometheusExporter.enabled }}
        - name: monitoring-web
          containerPort: {{ .Values.prometheusExporter.webBeanPort }}
          protocol: TCP
        - name: monitoring-ce
          containerPort: {{ .Values.prometheusExporter.ceBeanPort }}
          protocol: TCP
        {{- end }}
      resources: {{- toYaml .Values.resources | nindent 8 }}
      env:
        - name: SONAR_HELM_CHART_VERSION
          value: {{ .Chart.Version | replace "+" "_" }}
        {{- if .Values.OpenShift.enabled }}
        - name: IS_HELM_OPENSHIFT_ENABLED
          value: "true"
        {{- end }}
        - name: SONAR_JDBC_PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{ include "jdbc.secret" . }}
              key: {{ include "jdbc.secretPasswordKey" . }}
        - name: SONAR_WEB_SYSTEMPASSCODE
          valueFrom:
            secretKeyRef:
            {{- if and .Values.monitoringPasscodeSecretName .Values.monitoringPasscodeSecretKey }}
              name: {{ .Values.monitoringPasscodeSecretName }}
              key: {{ .Values.monitoringPasscodeSecretKey }}
            {{- else }}
              name: {{ include "sonarqube.fullname" . }}-monitoring-passcode
              key: SONAR_WEB_SYSTEMPASSCODE
            {{- end }}
        {{- (include "sonarqube.combined_env" . | fromJsonArray) | toYaml | trim | nindent 8 }}
      envFrom:
        - configMapRef:
            name: {{ include "sonarqube.fullname" . }}-jdbc-config
        {{- range .Values.extraConfig.secrets }}
        - secretRef:
            name: {{ . }}
        {{- end }}
        {{- range .Values.extraConfig.configmaps }}
        - configMapRef:
            name: {{ . }}
        {{- end }}
      livenessProbe:
        {{- tpl (omit .Values.livenessProbe "sonarWebContext" | toYaml) . | nindent 8 }}
      readinessProbe:
        {{- tpl (omit .Values.readinessProbe "sonarWebContext" | toYaml) . | nindent 8 }}
      startupProbe:
        httpGet:
          scheme: HTTP
          path: {{ .Values.startupProbe.sonarWebContext | default (include "sonarqube.webcontext" .) }}api/system/status
          port: http
        initialDelaySeconds: {{ .Values.startupProbe.initialDelaySeconds }}
        periodSeconds: {{ .Values.startupProbe.periodSeconds }}
        failureThreshold: {{ .Values.startupProbe.failureThreshold }}
        timeoutSeconds: {{ .Values.startupProbe.timeoutSeconds }}
      {{- with (include "sonarqube.containerSecurityContext" .) }}
      securityContext: {{- . | nindent 8 }}
      {{- end }}
      volumeMounts:
        - mountPath: {{ .Values.sonarqubeFolder }}/data
          name: sonarqube
          subPath: data
        - mountPath: {{ .Values.sonarqubeFolder }}/temp
          name: sonarqube
          subPath: temp
        - mountPath: {{ .Values.sonarqubeFolder }}/logs
          name: sonarqube
          subPath: logs
        - mountPath: /tmp
          name: tmp-dir
        {{- if or .Values.sonarProperties .Values.sonarSecretProperties .Values.sonarSecretKey (not .Values.elasticsearch.bootstrapChecks) }}
        - mountPath: {{ .Values.sonarqubeFolder }}/conf/
          name: concat-dir
        {{- end }}
        {{- if .Values.sonarSecretKey }}
        - mountPath: {{ .Values.sonarqubeFolder }}/secret/
          name: secret
        {{- end }}
        {{- if .Values.caCerts.enabled }}
        - mountPath: {{ .Values.sonarqubeFolder }}/certs
          name: sonarqube
          subPath: certs
        {{- end }}
        - mountPath: {{ .Values.sonarqubeFolder }}/extensions
          name: sonarqube
          subPath: extensions
        {{- if .Values.prometheusExporter.enabled }}
        - mountPath: {{ .Values.sonarqubeFolder }}/conf/prometheus-config.yaml
          subPath: prometheus-config.yaml
          name: prometheus-config
        - mountPath: {{ .Values.sonarqubeFolder }}/conf/prometheus-ce-config.yaml
          subPath: prometheus-ce-config.yaml
          name: prometheus-ce-config
        {{- end }}
        {{- with .Values.persistence.mounts }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
        {{- with .Values.extraVolumeMounts }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
  {{- with .Values.priorityClassName }}
  priorityClassName: {{ . }}
  {{- end }}
  {{- with .Values.nodeSelector }}
  nodeSelector: {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.hostAliases }}
  hostAliases: {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.tolerations }}
  tolerations: {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.affinity }}
  affinity: {{- toYaml . | nindent 4 }}
  {{- end }}
  serviceAccountName: {{ include "sonarqube.serviceAccountName" . }}
  volumes:
    {{- with .Values.persistence.volumes }}
    {{- tpl (toYaml . | nindent 4) $ }}
    {{- end }}
    {{- with .Values.extraVolumes }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
    {{- if or .Values.sonarProperties .Values.sonarSecretKey ( not .Values.elasticsearch.bootstrapChecks) }}
    - name: config
      configMap:
        name: {{ include "sonarqube.fullname" . }}-config
        items:
        - key: sonar.properties
          path: sonar.properties
    {{- end }}
    {{- if .Values.sonarSecretProperties }}
    - name: secret-config
      secret:
        secretName: {{ .Values.sonarSecretProperties }}
        items:
        - key: secret.properties
          path: secret.properties
    {{- end }}
    {{- if .Values.sonarSecretKey }}
    - name: secret
      secret:
        secretName: {{ .Values.sonarSecretKey }}
        items:
        - key: sonar-secret.txt
          path: sonar-secret.txt
    {{- end }}
    {{- include "sonarqube.volumes.caCerts" . | nindent 4 }}
    {{- if .Values.plugins.netrcCreds }}
    - name: plugins-netrc-file
      secret:
        secretName: {{ .Values.plugins.netrcCreds }}
        items:
        - key: netrc
          path: .netrc
    {{- end }}
    {{- if and .Values.jdbcOverwrite.oracleJdbcDriver .Values.jdbcOverwrite.oracleJdbcDriver.netrcCreds }}
    - name: oracle-jdbc-driver-netrc-file
      secret:
        secretName: {{ .Values.jdbcOverwrite.oracleJdbcDriver.netrcCreds }}
        items:
        - key: netrc
          path: .netrc
    {{- end }}
    {{- if and .Values.initSysctl.enabled (not .Values.OpenShift.enabled) }}
    - name: init-sysctl
      configMap:
        name: {{ include "sonarqube.fullname" . }}-init-sysctl
        items:
          - key: init_sysctl.sh
            path: init_sysctl.sh
    {{- end }}
    {{- if and .Values.persistence.enabled .Values.initFs.enabled (not .Values.OpenShift.enabled) }}
    - name: init-fs
      configMap:
        name: {{ include "sonarqube.fullname" . }}-init-fs
        items:
          - key: init_fs.sh
            path: init_fs.sh
    {{- end }}
    {{- if .Values.plugins.install }}
    - name: install-plugins
      configMap:
        name: {{ include "sonarqube.fullname" . }}-install-plugins
        items:
          - key: install_plugins.sh
            path: install_plugins.sh
    {{- end }}
    {{- if and .Values.jdbcOverwrite.oracleJdbcDriver .Values.jdbcOverwrite.oracleJdbcDriver.url }}
    - name: install-oracle-jdbc-driver
      configMap:
        name: {{ include "sonarqube.fullname" . }}-install-oracle-jdbc-driver
        items:
          - key: install_oracle_jdbc_driver.sh
            path: install_oracle_jdbc_driver.sh
    {{- end }}
    {{- if .Values.prometheusExporter.enabled }}
    - name: prometheus-config
      configMap:
        name: {{ include "sonarqube.fullname" . }}-prometheus-config
        items:
          - key: prometheus-config.yaml
            path: prometheus-config.yaml
    - name: prometheus-ce-config
      configMap:
        name: {{ include "sonarqube.fullname" . }}-prometheus-ce-config
        items:
          - key: prometheus-ce-config.yaml
            path: prometheus-ce-config.yaml
    {{- end }}
    - name: sonarqube
      {{- if .Values.persistence.enabled }}
      persistentVolumeClaim:
        claimName: {{ if .Values.persistence.existingClaim }}{{ .Values.persistence.existingClaim }}{{- else }}{{ include "sonarqube.fullname" . }}{{- end }}
      {{- else }}
      emptyDir: {{- toYaml .Values.emptyDir | nindent 8 }}
      {{- end  }}
    - name : tmp-dir
      emptyDir: {{- toYaml .Values.emptyDir | nindent 8 }}
      {{- if or .Values.sonarProperties .Values.sonarSecretProperties .Values.sonarSecretKey ( not .Values.elasticsearch.bootstrapChecks) }}
    - name : concat-dir
      emptyDir: {{- toYaml .Values.emptyDir | nindent 8 }}
      {{- end }}

{{- end -}}
