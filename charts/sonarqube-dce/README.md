# SonarQube

Code better in up to 27 languages. Improve Code Quality and Code Security throughout your workflow. [SonarQube](https://www.sonarqube.org/) can detect Bugs, Vulnerabilities, Security Hotspots and Code Smells and give you the guidance to fix them.

## Introduction

This helm chart bootstraps a SonarQube Data Center Edition cluster with a PostgreSQL database.

The latest version of the chart installs the latest SonarQube version.

To install the version of the chart for SonarQube 9.9 LTS, please read the section [below](#installing-the-lts-chart). Deciding between LTS and Latest? [This may help](https://www.sonarsource.com/products/sonarqube/downloads/lts/)

Please note that this chart does NOT support SonarQube Community, Developer, and Enterprise Editions.

## Compatibility

Compatible SonarQube Version: `9.9.0`

Supported Kubernetes Versions: From `1.23` to `1.26`

## Installing the chart

> **_NOTE:_**  Please refer to [the official page](https://docs.sonarqube.org/latest/setup/sonarqube-cluster-on-kubernetes/) for further information on how to install and tune the helm chart specifications.

Prior to installing the chart, please ensure that the `ApplicationNodes.jwtSecret` value is set properly with a HS256 key encoded with base64. In the following, an example on how to generate this key on a Unix system:
```bash
echo -n "your_secret" | openssl dgst -sha256 -hmac "your_key" -binary | base64
```

To install the chart:

```bash
helm repo add sonarqube https://SonarSource.github.io/helm-chart-sonarqube
helm repo update
kubectl create namespace sonarqube-dce
export JWT_SECRET=$(echo -n "your_secret" | openssl dgst -sha256 -hmac "your_key" -binary | base64)
helm upgrade --install -n sonarqube-dce sonarqube sonarqube/sonarqube-dce --set ApplicationNodes.jwtSecret=$JWT_SECRET
```

The above command deploys SonarQube on the Kubernetes cluster in the default configuration in the sonarqube namespace. The [configuration](#configuration) section lists the parameters that can be configured during installation.

The default login is admin/admin.

## Installing the SonarQube 9.9 LTS chart

The version of the chart for the SonarQube 9.9 LTS is being distributed as the `7.x.x` version of this chart.

In order to use it, please set the version constraint `~7`, which is equivalent to `>=7.0.0 && <= 8.0.0`. That version parameter **must** be used in every helm related command including `install`, `upgrade`, `template`, and `diff` (don't treat this as an exhaustive list).

Example:
```
helm upgrade --install -n sonarqube-dce --version ~7 sonarqube sonarqube/sonarqube-dce --set ApplicationNodes.jwtSecret=$JWT_SECRET
```

## How to use it

Take some time to read the Deploy [SonarQube on Kubernetes](https://docs.sonarqube.org/latest/setup/sonarqube-cluster-on-kubernetes/) page.
SonarQube deployment on Kubernetes has been tested with the recommendations and constraints documented there, and deployment has some limitations.

## Uninstalling the chart

To uninstall/delete the deployment:

```bash
$ helm list
NAME        REVISION    UPDATED                     STATUS      CHART            NAMESPACE
kindly-newt 1           Mon Oct  2 15:05:44 2017    DEPLOYED    sonarqube-0.1.0  sonarqube
$ helm delete kindly-newt
```

## Ingress

### Path

Some cloud may need the path to be `/*` instead of `/.` Try this first if you are having issues getting traffic through the ingress.

### Default Backend

if you use GCP as a cloud provider you need to set a default backend to avoid useless default backend created by the gce controller. To add this default backend you must set "ingress.class" annotation with "gce" or "gce-internal" value.

Example:

```yaml
---
ingress:
  enabled: true
  hosts:
    - name: sonarqube.example.com
      path: "/*"
  annotations:
    kubernetes.io/ingress.class: "gce-internal"
    kubernetes.io/ingress.allow-http: "false"
```

## Monitoring

This Helm chart offers the possibility to monitor SonarQube with Prometheus.

### Export JMX metrics

The prometheus exporter (`ApplicationNodes.prometheusExporter.enabled=true`) converts the JMX metrics into a format that Prometheus can understand. After the metrics are exported, you can connect your Prometheus instance and scrape them.

Per default the JMX metrics for the Web Bean and the CE Bean are exposed on port 8000 and 8001. These values can be configured with `ApplicationNodes.prometheusExporter.webBeanPort` and `ApplicationNodes.prometheusExporter.ceBeanPort`.

### PodMonitor

If a Prometheus Operator is deployed in your cluster, you can enable a PodMonitor resource with `ApplicationNodes.prometheusMonitoring.podMonitor.enabled`. It scrapes the Prometheus endpoint `/api/monitoring/metrics` exposed by the SonarQube application.


## Configuration

The following table lists the configurable parameters of the SonarQube chart and their default values.

### Search Nodes Configuration

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `searchNodes.image.repository` | search image repository | `sonarqube` |
| `searchNodes.image.tag` | search image tag | `9.9.0-datacenter-search` |
| `searchNodes.image.pullPolicy` | search image pull policy | `IfNotPresent` |
| `searchNodes.image.pullSecret` | (DEPRECATED) search imagePullSecret to use for private repository | `nil` |
| `searchNodes.image.pullSecrets` | search imagePullSecrets to use for private repository | `nil` |
| `searchNodes.env` | Environment variables to attach to the search pods | `nil` |
| `searchNodes.sonarProperties` | Custom `sonar.properties` file for Search Nodes | `None` |
| `searchNodes.sonarSecretProperties` | Additional `sonar.properties` file for Search Nodes to load from a secret | `None` |
| `searchNodes.sonarSecretKey` | Name of existing secret used for settings encryption | `None` |
| `searchNodes.searchAuthentication.enabled` | Securing the Search Cluster with basic authentication and TLS in between search nodes | `false` |
| `searchNodes.searchAuthentication.keyStoreSecret` | Existing PKCS#12 Container as Keystore/Truststore to be used | `""` |
| `searchNodes.searchAuthentication.keyStorePassword` | Password to Keystore/Truststore used in search nodes (optional) | `""` |
| `searchNodes.searchAuthentication.keyStorePasswordSecret` | Existing secret for Password to Keystore/Truststore used in search nodes (optional) | `nil` |
| `searchNodes.searchAuthentication.userPassword` | A User Password that will be used to authenticate against the Search Cluster | `""` |
| `searchNodes.replicaCount` | Replica count of the Search Nodes | `3` |
| `searchNodes.podDistributionBudget` | PodDisctributionBudget for the Search Nodes | `maxUnavailable: "33%"` |
| `searchNodes.securityContext.fsGroup` | Group applied to mounted directories/files on search nodes | `1000` |
| `searchNodes.containerSecurityContext.runAsUser` | User to run search container in sonarqube pod as | `1000` |
| `searchNodes.readinessProbe.initialDelaySeconds` | ReadinessProbe initial delay for Search Node checking| `60` |
| `searchNodes.readinessProbe.periodSeconds` | ReadinessProbe period between checking Search Node | `30` |
| `searchNodes.readinessProbe.failureThreshold`| ReadinessProbe thresold for marking as failed | `6` |
| `searchNodes.readinessProbe.timeoutSeconds`| ReadinessProbe timeout delay | `1` |
| `searchNodes.livenessProbe.initialDelaySeconds`| LivenessProbe initial delay for Search Node checking | `60` |
| `searchNodes.livenessProbe.periodSeconds`| LivenessProbe period between checking Search Node | `30` |
| `searchNodes.livenessProbe.failureThreshold`| LivenessProbe thresold for marking as dead | `6` |
| `searchNodes.livenessProbe.timeoutSeconds`| LivenessProbe timeout delay | `1` |
| `searchNodes.startupProbe.initialDelaySeconds`| StartupProbe initial delay for Search Node checking | `30` |
| `searchNodes.startupProbe.periodSeconds`| StartupProbe period between checking Search Node | `10` |
| `searchNodes.startupProbe.failureThreshold`| StartupProbe thresold for marking as failed | `24` |
| `searchNodes.startupProbe.timeoutSeconds`| StartupProbe timeout delay | `1` |
| `searchNodes.resources.requests.memory` | memory request for Search Nodes | `2Gi` |
| `searchNodes.resources.requests.cpu` | cpu request for Search Nodes | `400m` |
| `searchNodes.resources.limits.memory` | memory limit for Search Nodes. should not be under 4G | `4096M` |
| `searchNodes.resources.limits.cpu` | cpu limit for Search Nodes | `800m` |
| `searchNodes.persistence.enabled` | enabled or disables the creation of VPCs for the Search Nodes | `true` |
| `searchNodes.persistence.annotations` | PVC annotations for the Search Nodes | `{}` |
| `searchNodes.persistence.storageClass` | Storage class to be used | `""` |
| `searchNodes.persistence.accessMode` | Volumes access mode to be set | `ReadWriteOnce` |
| `searchNodes.persistence.size` | Size of the PVC | `5G` |
| `searchNodes.persistence.uid` | UID used for init-fs container | `1000` |
| `searchNodes.extraContainers` | Array of extra containers to run alongside | `[]` |

### App Nodes Configuration

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `ApplicationNodes.image.repository` | app image repository | `sonarqube` |
| `ApplicationNodes.image.tag` | app image tag | `9.9.0-datacenter-app` |
| `ApplicationNodes.image.pullPolicy` | app image pull policy | `IfNotPresent` |
| `ApplicationNodes.image.pullSecret` | (DEPRECATED) app imagePullSecret to use for private repository | `nil` |
| `ApplicationNodes.image.pullSecrets` | app imagePullSecrets to use for private repository | `nil` |
| `ApplicationNodes.env` | Environment variables to attach to the app pods | `nil` |
| `ApplicationNodes.sonarProperties` | Custom `sonar.properties` key-value pairs for App Nodes (e.g., "ApplicationNodes.sonarProperties.sonar.forceAuthentication=true") | `None` |
| `ApplicationNodes.sonarSecretProperties` | Additional `sonar.properties` key-value pairs for App Nodes to load from a secret | `None` |
| `ApplicationNodes.sonarSecretKey` | Name of existing secret used for settings encryption | `None` |
| `ApplicationNodes.replicaCount` | Replica count of the app Nodes | `2` |
| `ApplicationNodes.podDistributionBudget` | PodDisctributionBudget for the App Nodes | `minAvailable: "50%"` |
| `ApplicationNodes.securityContext.fsGroup` | Group applied to mounted directories/files on app nodes | `1000` |
| `ApplicationNodes.containerSecurityContext.runAsUser` | User to run app container in sonarqube pod as | `1000` |
| `ApplicationNodes.readinessProbe.initialDelaySeconds` | ReadinessProbe initial delay for app Node checking| `60` |
| `ApplicationNodes.readinessProbe.periodSeconds` | ReadinessProbe period between checking app Node | `30` |
| `ApplicationNodes.readinessProbe.failureThreshold`| ReadinessProbe thresold for marking as failed | `6` |
| `ApplicationNodes.readinessProbe.timeoutSeconds`| ReadinessProbe timeout delay | `1` |
| `ApplicationNodes.readinessProbe.sonarWebContext`| SonarQube web context for readinessProbe | `/` |
| `ApplicationNodes.livenessProbe.initialDelaySeconds`| LivenessProbe initial delay for app Node checking | `60` |
| `ApplicationNodes.livenessProbe.periodSeconds`| LivenessProbe period between checking app Node | `30` |
| `ApplicationNodes.livenessProbe.failureThreshold`| LivenessProbe thresold for marking as dead | `6` |
| `ApplicationNodes.livenessProbe.timeoutSeconds`| LivenessProbe timeout delay | `1` |
| `ApplicationNodes.readinessProbe.sonarWebContext`| SonarQube web context for StartupProbe | `/` |
| `ApplicationNodes.startupProbe.initialDelaySeconds`| StartupProbe initial delay for app Node checking | `30` |
| `ApplicationNodes.startupProbe.periodSeconds`| StartupProbe period between checking app Node | `10` |
| `ApplicationNodes.startupProbe.failureThreshold`| StartupProbe thresold for marking as failed | `24` |
| `ApplicationNodes.startupProbe.timeoutSeconds`| StartupProbe timeout delay | `1` |
| `ApplicationNodes.readinessProbe.sonarWebContext`| SonarQube web context for StartupProbe | `/` |
| `ApplicationNodes.resources.requests.memory` | memory request for app Nodes | `2Gi` |
| `ApplicationNodes.resources.requests.cpu` | cpu request for app Nodes | `400m` |
| `ApplicationNodes.resources.limits.memory` | memory limit for app Nodes. should not be under 4G | `4096M` |
| `ApplicationNodes.resources.limits.cpu` | cpu limit for app Nodes | `800m` |
| `ApplicationNodes.prometheusExporter.enabled` | Use the Prometheus JMX exporter | `false` |
| `ApplicationNodes.prometheusExporter.version` | jmx_prometheus_javaagent version to download from Maven Central | `0.17.2`|
| `ApplicationNodes.prometheusExporter.noCheckCertificate` | Flag to not check server's certificate when downloading jmx_prometheus_javaagent | `false`|
| `ApplicationNodes.prometheusExporter.webBeanPort` | Port where the jmx_prometheus_javaagent exposes the metrics for the webBean | `8000`|
| `ApplicationNodes.prometheusExporter.ceBeanPort` | Port where the jmx_prometheus_javaagent exposes the metrics for the ceBean | `8001`|
| `ApplicationNodes.prometheusExporter.downloadURL` | Alternative full download URL for the jmx_prometheus_javaagent.jar (overrides `prometheusExporter.version`) | `""` |
| `ApplicationNodes.prometheusExporter.config` | Prometheus JMX exporter config yaml for the web process, and the CE process if `prometheusExporter.ceConfig` is not set | see `values.yaml`|
| `ApplicationNodes.prometheusExporter.ceConfig` | Prometheus JMX exporter config yaml for the CE process (by default, `prometheusExporter.config` is used | `None` |
| `ApplicationNodes.prometheusExporter.httpProxy` | HTTP proxy for downloading JMX agent | `""` |
| `ApplicationNodes.prometheusExporter.httpsProxy` | HTTPS proxy for downloading JMX agent | `""` |
| `ApplicationNodes.prometheusExporter.noProxy` | No proxy for downloading JMX agent | `""` |
| `ApplicationNodes.prometheusExporter.securityContext` | Security context for downloading the jmx agent | see `values.yaml`|
| `ApplicationNodes.prometheusMonitoring.podMonitor.enabled` | Enable Prometheus PodMonitor | `false` |
| `ApplicationNodes.prometheusMonitoring.podMonitor.namespace` | Specify a custom namespace where the PodMonitor will be created | `default` |
| `ApplicationNodes.prometheusMonitoring.podMonitor.interval` | Specify the interval how often metrics should be scraped | `30s` |
| `ApplicationNodes.prometheusMonitoring.podMonitor.scrapeTimeout` | Specify the timeout after a scrape is ended | `None` |
| `ApplicationNodes.prometheusMonitoring.podMonitor.jobLabel` |  Name of the label on target services that prometheus uses as job name | `None` |
| `ApplicationNodes.plugins.install` | Link(s) to the plugin JARs to download and install | `[]` |
| `ApplicationNodes.plugins.resources` | Plugin Pod resource requests & limits | `{}` |
| `ApplicationNodes.plugins.httpProxy` | For use behind a corporate proxy when downloading plugins | `""` |
| `ApplicationNodes.plugins.httpsProxy` | For use behind a corporate proxy when downloading plugins | `""` |
| `ApplicationNodes.plugins.noProxy` | For use behind a corporate proxy when downloading plugins | `""` |
| `ApplicationNodes.plugins.image` | Image for plugins container | `""` |
| `ApplicationNodes.plugins.resources` | Resources for plugins container | `""` |
| `ApplicationNodes.plugins.netrcCreds` | Name of the secret containing .netrc file to use creds when downloading plugins | `""` |
| `ApplicationNodes.plugins.noCheckCertificate` | Flag to not check server's certificate when downloading plugins | `false |
| `ApplicationNodes.plugins.securityContext` | Security context for the container to download plugins | see `values.yaml |
| `ApplicationNodes.jvmOpts` | (DEPRECATED) Values to add to SONARQUBE_WEB_JVM_OPTS | `""` |
| `ApplicationNodes.jvmCeOpts` | (DEPRECATED) Values to add to SONAR_CE_JAVAOPTS | `""` |
| `ApplicationNodes.jwtSecret` | A HS256 key encoded with base64 (*This value must be set before installing the chart, see [the documentation](https://docs.sonarqube.org/latest/setup/sonarqube-cluster-on-kubernetes/)*) | `""` |
| `ApplicationNodes.existingJwtSecret` | secret that contains the `jwtSecret` | `nil` |
| `ApplicationNodes.resources.requests.memory` | memory request for app Nodes | `2Gi` |
| `ApplicationNodes.resources.requests.cpu` | cpu request for app Nodes | `400m` |
| `ApplicationNodes.resources.limits.memory` | memory limit for app Nodes. should not be under 4G | `4096M` |
| `ApplicationNodes.resources.limits.cpu` | cpu limit for app Nodes | `800m` |
| `ApplicationNodes.extraContainers` | Array of extra containers to run alongside | `[]` |

### Generic Configuration

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `affinity` | Node / Pod affinities | `{}` |
| `tolerations` | List of node taints to tolerate | `[]` |
| `priorityClassName` | Schedule pods on priority (e.g. `high-priority`) | `None` |
| `nodeSelector` | Node labels for pod assignment | `{}` |
| `hostAliases` | Aliases for IPs in /etc/hosts | `[]` |
| `podLabels` | Map of labels to add to the pods | `{}` |
| `env` | Environment variables to attach to the pods | `{}`|
| `annotations` | SonarQube Pod annotations | `{}` |


### NetworkPolicies

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `networkPolicy.enabled` | Create NetworkPolicies | `false` |
| `networkPolicy.prometheusNamespace` | Allow incoming traffic to monitoring ports from this namespace | `nil` |
| `networkPolicy.additionalNetworkPolicys` | User defined NetworkPolicies (usefull for external database) | `nil` |

### OpenShift

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `OpenShift.enabled` | Define if this deployment is for OpenShift | `false` |
| `OpenShift.createSCC` | If this deployment is for OpenShift, define if SCC should be created for sonarqube pod | `true` |

### Elasticsearch

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `elasticsearch.bootstrapChecks` | Enables/disables Elasticsearch bootstrap checks | `true` |

### Service

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.externalPort` | Kubernetes service port | `9000` |
| `service.internalPort` | Kubernetes container port | `9000` |
| `service.labels` | Kubernetes service labels | `None` |
| `service.annotations` | Kubernetes service annotations | `None` |
| `service.loadBalancerSourceRanges` | Kubernetes service LB Allowed inbound IP addresses | `None` |
| `service.loadBalancerIP` | Kubernetes service LB Optional fixed external IP | `None` |

### Ingress

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `nginx.enabled` | Also install Nginx Ingress Helm | `false` |
| `ingress.enabled` | Flag to enable Ingress | `false` |
| `ingress.labels` | Ingress additional labels | `{}` |
| `ingress.hosts[0].name` | Hostname to your SonarQube installation | `sonarqube.your-org.com` |
| `ingress.hosts[0].path` | Path within the URL structure | `/` |
| `ingress.hosts[0].serviceName` | Optional field to override the default serviceName of a path | `None` |
| `ingress.hosts[0].servicePort` | Optional field to override the default servicePort of a path | `None` |
| `ingress.tls` | Ingress secrets for TLS certificates | `[]` |
| `ingress.ingressClassName` | Optional field to configure ingress class name | `None` |
| `ingress.annotations` | Field to add extra annotations to the ingress | {`nginx.ingress.kubernetes.io/proxy-body-size=64m`} |
| `ingress.annotations.nginx.ingress.kubernetes.io/proxy-body-size` | Field to set the maximum allowed size of the client request body  | `64m` |

### InitContainers

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `initContainers.image` | Change init container image | `busybox:1.32` |
| `initContainers.securityContext` | SecurityContext for init containers | `None` |
| `initContainers.resources` | Resources for init containers | `{}` |
| `extraInitContainers` | Extra init containers to e.g. download required artifacts | `{}` |
| `caCerts.enabled` | Flag for enabling additional CA certificates | `false` |
| `caCerts.image` | Change init CA certificates container image | `adoptopenjdk/openjdk11:alpine` |
| `caCerts.secret` | Name of the secret containing additional CA certificates | `None` |
| `initSysctl.enabled` | Modify k8s worker to conform to system requirements | `true` |
| `initSysctl.vmMaxMapCount` | Set init sysctl container vm.max_map_count | `524288` |
| `initSysctl.fsFileMax` | Set init sysctl container fs.file-max | `131072` |
| `initSysctl.nofile` | Set init sysctl container open file descriptors limit | `131072` |
| `initSysctl.nproc` | Set init sysctl container open threads limit | `8192 ` |
| `initSysctl.image` | Change init sysctl container image | `busybox:1.32` |
| `initSysctl.securityContext` | InitSysctl container security context | `{privileged: true}` |
| `initSysctl.resources` | InitSysctl container resource requests & limits | `{}` |
| `initFs.enabled` | Enable file permission change with init container | `true` |
| `initFs.image` | InitFS container image | `busybox:1.32` |
| `initFs.securityContext.privileged` | InitFS container needs to run privileged | `true` |

### Persistence

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `persistence.enabled` | Flag for enabling persistent storage | `false` |
| `persistence.annotations` | Kubernetes pvc annotations | `{}` |
| `persistence.existingClaim` | Do not create a new PVC but use this one | `None` |
| `persistence.storageClass` | Storage class to be used | `""` |
| `persistence.accessMode` | Volumes access mode to be set | `ReadWriteOnce` |
| `persistence.size` | Size of the volume | `5Gi` |
| `persistence.volumes` | Specify extra volumes. Refer to ".spec.volumes" specification | `[]` |
| `persistence.mounts` | Specify extra mounts. Refer to ".spec.containers.volumeMounts" specification | `[]` |
| `emptyDir` | Configuration of resources for `emptyDir` | `{}` |

### SonarQube Specific

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `sonarqubeFolder` | Directory name of SonarQube | `/opt/sonarqube` |
| `monitoringPasscode` | Value for sonar.web.systemPasscode needed for LivenessProbes (encoded to Base64 format) | `define_it` |
| `monitoringPasscodeSecretName` | Name of the secret where to load `monitoringPasscode` | `None` |
| `monitoringPasscodeSecretKey` | Key of an existing secret containing `monitoringPasscode` | `None` |
| `extraContainers` | Array of extra containers to run alongside the `sonarqube` container (aka. Sidecars) | `[]` |

### JDBC Overwrite

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `jdbcOverwrite.enable` | Enable JDBC overwrites for external Databases (disable `postgresql.enabled`) | `false` |
| `jdbcOverwrite.jdbcUrl` | The JDBC url to connect the external DB | `jdbc:postgresql://myPostgress/myDatabase?socketTimeout=1500` |
| `jdbcOverwrite.jdbcUsername` | The DB user that should be used for the JDBC connection | `sonarUser` |
| `jdbcOverwrite.jdbcPassword` | The DB password that should be used for the JDBC connection (Use this if you don't mind the DB password getting stored in plain text within the values file) | `sonarPass` |
| `jdbcOverwrite.jdbcSecretName` | Alternatively, use a pre-existing k8s secret containing the DB password | `None` |
| `jdbcOverwrite.jdbcSecretPasswordKey` | If the pre-existing k8s secret is used this allows the user to overwrite the 'key' of the password property in the secret | `None` |

### Bundled Postgres Chart

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `postgresql.enabled` | Set to `false` to use external database server | `true` |
| `postgresql.existingSecret` | existingSecret Name of existing secret to use for PostgreSQL passwords | `nil` |
| `postgresql.postgresqlUsername` | Postgresql database user | `sonarUser` |
| `postgresql.postgresqlPassword` | Postgresql database password | `sonarPass` |
| `postgresql.postgresqlDatabase` | Postgresql database name | `sonarDB` |
| `postgresql.service.port` | Postgresql port | `5432` |
| `postgresql.resources.requests.memory` | Postgresql memory request | `256Mi` |
| `postgresql.resources.requests.cpu` | Postgresql cpu request | `250m` |
| `postgresql.resources.limits.memory` | Postgresql memory limit | `2Gi` |
| `postgresql.resources.limits.cpu` | Postgresql cpu limit | `2` |
| `postgresql.persistence.enabled` | Postgresql persistence en/disabled | `true` |
| `postgresql.persistence.accessMode` | Postgresql persistence accessMode | `ReadWriteOnce` |
| `postgresql.persistence.size` | Postgresql persistence size | `20Gi` |
| `postgresql.persistence.storageClass` | Postgresql persistence storageClass | `""` |
| `postgresql.securityContext.enabled` | Postgresql securityContext en/disabled | `true` |
| `postgresql.securityContext.fsGroup` | Postgresql securityContext fsGroup | `1001` |
| `postgresql.securityContext.runAsUser` | Postgresql securityContext runAsUser | `1001` |
| `postgresql.volumePermissions.enabled` | Postgres vol permissions en/disabled | `false` |
| `postgresql.volumePermissions.securityContext.runAsUser` | Postgres vol permissions secContext runAsUser | `0` |
| `postgresql.shmVolume.chmod.enabled` | Postgresql shared memory vol en/disabled | `false` |
| `postgresql.serivceAccount.enabled` | Postgresql service Account creation en/disabled | `false` |
| `postgresql.serivceAccount.name` | Postgresql service Account name | `""` |

### Tests

| Parameter                    | Description | Default |
|------------------------------| ----------- |--|
| `tests.enabled`              | Flag that allows tests to be excluded from the generated yaml | `true` |
| `tests.image`                | Change test container image | `bitnami/minideb-extras`|
| `tests.initContainers.image` | Change init test container image | `bats/bats:1.2.1` |

### ServiceAccount

| Parameter                       | Description                                                                          | Default               |
|---------------------------------|--------------------------------------------------------------------------------------|-----------------------|
| `serviceAccount.create`         | If set to true, create a serviceAccount                                              | `false`               |
| `serviceAccount.name`           | Name of the serviceAccount to create/use                                             | `sonarqube-sonarqube` |
| `serviceAccount.automountToken` | Manage `automountServiceAccountToken` field for mounting service account credentials | `false`               |
| `serviceAccount.annotations`    | Additional serviceAccount annotations                                                | `{}`                  |

### ExtraConfig

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `extraConfig.secrets` | A list of `Secret`s (which must contain key/value pairs) which may be loaded into the Scanner as environment variables | `[]` |
| `extraConfig.configmaps` | A list of `ConfigMap`s (which must contain key/value pairs) which may be loaded into the Scanner as environment variables | `[]` |

### Advanced Options

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `logging.jsonOutput` | Enable/Disable logging in JSON format | `false` |
| `account.adminPassword` | Custom new admin password | `admin` |
| `account.currentAdminPassword` | Current admin password | `admin` |
| `account.adminPasswordSecretName` | Secret containing `password` (custom password) and `currentPassword` (current password) keys for admin | `None` |
| `account.resources.requests.memory` | Memory request for Admin hook | `128Mi` |
| `account.resources.requests.cpu` | CPU request for Admin hook | `100m` |
| `account.resources.limits.memory` | Memory limit for Admin hook | `128Mi` |
| `account.resources.limits.cpu` | CPU limit for Admin hook | `100m` |
| `account.sonarWebContext` | SonarQube web context for Admin hook | `nil` |
| `curlContainerImage` | Curl container image | `curlimages/curl:latest` |
| `adminJobAnnotations` | Custom annotations for admin hook Job | `{}` |
| `terminationGracePeriodSeconds` | Configuration of `terminationGracePeriodSeconds` | `60` |


You can also configure values for the PostgreSQL database via the Postgresql [Chart](https://hub.helm.sh/charts/bitnami/postgresql)

For overriding variables see: [Customizing the chart](https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing)

### Use custom `cacerts`

In environments with air-gapped setup, especially with internal tooling (repos) and self-signed certificates it is required to provide an adequate `cacerts` which overrides the default one:

1. Create a yaml file `cacerts.yaml` with a secret that contains one or more keys to represent the certificates that you want including

   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: my-cacerts
   stringData:
     cert-1.crt: |
       xxxxxxxxxxxxxxxxxxxxxxx
   ```

2. Upload your `cacerts.yaml` to a secret in the cluster you are installing SonarQube to.

   ```shell
   kubectl apply -f cacerts.yaml
   ```

3. Set the following values of the chart:

   ```yaml
   caCerts:
     enabled: true
     secret: my-cacerts
   ```

### Elasticsearch Settings

Since SonarQube needs Elasticsearch, some [bootstrap checks](https://www.elastic.co/guide/en/elasticsearch/reference/master/bootstrap-checks.html) of the host settings are done at start.

This chart offers the option to use an initContainer in privilaged mode to automatically set certain kernel settings on the kube worker. While this can ensure proper functionality of Elasticsearch, modifying the underlying kernel settings on the Kubernetes node can impact other users. It may be best to work with your cluster administrator to either provide specific nodes with the proper kernel settings, or ensure they are set cluster wide.

To enable auto-configuration of the kube worker node, set `elasticsearch.configureNode` to `true`. This is the default behavior, so you do not need to explicitly set this.

This will run `sysctl -w vm.max_map_count=262144` on the worker where the sonarqube pod(s) get scheduled. This needs to be set to `262144` but normally defaults to `65530`. Other kernel settings are recommended by the [docker image](https://hub.docker.com/_/sonarqube/#requirements), but the defaults work fine in most cases.

To disable worker node configuration, set `elasticsearch.configureNode` to `false`. Note that if node configuration is not enabled, then you will likely need to also disable the Elasticsearch bootstrap checks. These can be explicitly disabled by setting `elasticsearch.bootstrapChecks` to `false`.

### Extra Config

For environments where another tool, such as terraform or ansible, is used to provision infrastructure or passwords then setting databases addresses and credentials via helm becomes less than ideal. Ditto for environments where this config may be visible.

In such environments, configuration may be read, via environment variables, from Secrets and ConfigMaps.

1. Create a `ConfigMap` (or `Secret`) containing key/value pairs, as expected by SonarQube.

   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: external-sonarqube-opts
   data:
     SONARQUBE_JDBC_USERNAME: foo
     SONARQUBE_JDBC_URL: jdbc:postgresql://db.example.com:5432/sonar
   ```

2. Set the following in your `values.yaml` (using the key `extraConfig.secrets` to reference `Secret`s)

   ```yaml
   extraConfig:
     configmaps:
       - external-sonarqube-opts
   ```
