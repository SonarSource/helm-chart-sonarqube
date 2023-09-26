# SonarQube

Code better in up to 27 languages. Improve Code Quality and Code Security throughout your workflow. [SonarQube](https://www.sonarqube.org/) can detect Bugs, Vulnerabilities, Security Hotspots and Code Smells and give you the guidance to fix them.

## Introduction

This chart bootstraps an instance of the latest SonarQube version with a PostgreSQL database.

The latest version of the chart installs the latest SonarQube version.

To install the version of the chart for SonarQube 9.9 LTS, please read the section [below](#installing-the-sonarqube-99-lts-chart). Deciding between LTS and Latest? [This may help](https://www.sonarsource.com/products/sonarqube/downloads/lts/)

Please note that this chart only supports SonarQube Community, Developer, and Enterprise editions.

## Compatibility

Compatible SonarQube Version: `10.2.0`

Supported Kubernetes Versions: From `1.24` to `1.27`

## Installing the chart

To install the chart:

```bash
helm repo add sonarqube https://SonarSource.github.io/helm-chart-sonarqube
helm repo update
kubectl create namespace sonarqube
helm upgrade --install -n sonarqube sonarqube sonarqube/sonarqube
```

The above command deploys SonarQube on the Kubernetes cluster in the default configuration in the sonarqube namespace. The [configuration](#configuration) section lists the parameters that can be configured during installation.

The default login is admin/admin.

## Installing the SonarQube 9.9 LTS chart

The version of the chart for the SonarQube 9.9 LTS is being distributed as the `8.x.x` version of this chart.

In order to use it, please set the version constraint `~8`, which is equivalent to `>=8.0.0 && <= 9.0.0`. That version parameter **must** be used in every helm related command including `install`, `upgrade`, `template`, and `diff` (don't treat this as an exhaustive list).

Example:
```
helm upgrade --install -n sonarqube --version ~8 sonarqube sonarqube/sonarqube
```

To upgrade from the old and unmaintained [sonarqube-lts chart](https://artifacthub.io/packages/helm/sonarqube/sonarqube-lts), please follow the steps described [in this section](#upgrade-from-the-old-sonarqube-lts-to-this-chart).

## How to use it

Take some time to read the Deploy on [SonarQube on Kubernetes](https://docs.sonarqube.org/latest/setup/sonarqube-on-kubernetes/) page.
SonarQube deployment on Kubernetes has been tested with the recommendations and constraints documented there, and deployment has some limitations.

## Uninstalling the chart

To uninstall/delete the deployment:

```bash
$ helm list
NAME        REVISION    UPDATED                     STATUS      CHART            NAMESPACE
kindly-newt 1           Mon Oct  2 15:05:44 2017    DEPLOYED    sonarqube-0.1.0  sonarqube
$ helm delete kindly-newt
```

## Prerequisites and suggested settings for production

Please read the official documentation prerequisites [here](https://docs.sonarqube.org/latest/requirements/prerequisites-and-overview/).

### Kubernetes - Pod Security Standards

The following [Pod Security levels](https://kubernetes.io/docs/concepts/security/pod-security-admission/#pod-security-levels) cannot be used in combination with SonarQube's chart:

* Baseline. The `init-sysctl` container requires `securityContext.privileged=true`.
* Restricted. In addition to the previous requirement,
  * The `sonarqube-postgresql`, `wait-for-db`, `init-sysctl`, and `sonarqube` containers require `securityContext.allowPrivilegeEscalation=true`, unrestricted capabilities, running as `root`, and a `seccompProfile` different from `RuntimeDefault` or `localhost`.

### Elasticsearch prerequisites

SonarQube runs Elasticsearch under the hood.

Elasticsearch is rolling out (strict) prerequisites that cannot be disabled when running in production context (see [this](https://www.elastic.co/blog/bootstrap_checks_annoying_instead_of_devastating) blog post regarding bootstrap checks, and the [official guide](https://www.elastic.co/guide/en/elasticsearch/reference/5.0/bootstrap-checks.html)).

Because of such constraints, even when running in Docker containers, SonarQube requires some settings at the host/kernel level.

Please carefully read the following and make sure these configurations are set up at the host level:

- [vm.max_map_count](https://www.elastic.co/guide/en/elasticsearch/reference/current/vm-max-map-count.html#vm-max-map-count)
- [seccomp filter should be available](https://github.com/SonarSource/docker-sonarqube/issues/614)

In general, please carefully read the Elasticsearch's [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/system-config.html).

### Production use case

The SonarQube helm chart is packed with multiple features enabling users to install and test SonarQube on Kubernetes easily.

Nonetheless, if you intend to run a production-grade SonarQube please follow these recommendations.

- Set `nginx.enabled` to **false**. This parameter would run the nginx chart. This is useful for testing purposes only. Ingress controllers are critical Kubernetes components, we advise users to install their own.
- Set `postgresql.enabled` to **false**. This parameter would run the postgresql pre-2022 bitnami chart. That is useful for testing purposes, however, given that the database is at the hearth of SonarQube, we advise users to be careful with it and use a well-maintained database as a service or deploy their own database on top of Kubernetes.
- Set `initSysctl.enabled` to **false**. This parameter would run **root** `sysctl` commands, while those sysctl-related values should be set by the Kubernetes administrator at the node level (see [here](#elasticsearch-prerequisites))
- Set `initFs.enabled` to **false**. This parameter would run **root** `chown` commands. The parameter exists to fix non-posix, CSI, or deprecated drivers.

## Upgrade

1. Read through the [SonarQube Upgrade Guide](https://docs.sonarqube.org/latest/setup/upgrading/) to familiarize yourself with the general upgrade process (most importantly, back up your database)
2. Change the SonarQube version on `values.yaml`
3. Redeploy SonarQube with the same helm chart (see [Install instructions](#installing-the-chart))
4. Browse to http://yourSonarQubeServerURL/setup and follow the setup instructions
5. Reanalyze your projects to get fresh data

### Upgrade from the old sonarqube-lts to this chart

Please refer to the Helm upgrade section accessible [here](https://docs.sonarqube.org/latest/setup-and-upgrade/upgrade-the-server/upgrade-guide/)

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

The prometheus exporter (`prometheusExporter.enabled=true`) converts the JMX metrics into a format that Prometheus can understand. After the metrics are exported, you can connect your Prometheus instance and scrape them.

Per default the JMX metrics for the Web Bean and the CE Bean are exposed on port 8000 and 8001. These values can be configured with `prometheusExporter.webBeanPort` and `prometheusExporter.ceBeanPort`.

### PodMonitor

If a Prometheus Operator is deployed in your cluster, you can enable a PodMonitor resource with `prometheusMonitoring.podMonitor.enabled`. It scrapes the Prometheus endpoint `/api/monitoring/metrics` exposed by the SonarQube application.

## Configuration

The following table lists the configurable parameters of the SonarQube chart and their default values.

### Global

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `deploymentType` | Deployment Type (supported values are `StatefulSet` or `Deployment`) | `StatefulSet` |
| `replicaCount`   | Number of replicas deployed (supported values are 0 and 1)  | `1` |
| `deploymentStrategy` | Deployment strategy | `{}` |
| `priorityClassName` | Schedule pods on priority (e.g. `high-priority`) | `None` |
| `schedulerName` | Kubernetes scheduler name | `None` |
| `affinity` | Node / Pod affinities | `{}` |
| `tolerations` | List of node taints to tolerate | `[]` |
| `nodeSelector` | Node labels for pod assignment | `{}` |
| `hostAliases` | Aliases for IPs in /etc/hosts | `[]` |
| `podLabels` | Map of labels to add to the pods | `{}` |
| `env` | Environment variables to attach to the pods | `{}`|
| `annotations` | SonarQube Pod annotations | `{}` |
| `edition` | SonarQube Edition to use (e.g. `community`, `developer` or `enterprise`) | `community` |
| `sonarWebContext` | SonarQube web context, also serve as default value for `ingress.path`, `account.sonarWebContext` and probes path. | `` |

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
| `OpenShift.createSCC` | If this deployment is for OpenShift, define if SCC should be created for sonarqube pod  | `true` |

### Image

| Parameter | Description | Default                        |
| --------- | ----------- |--------------------------------|
| `image.repository` | image repository | `sonarqube`                    |
| `image.tag` | `sonarqube` image tag. | `10.2.0-{{ .Values.edition }}` |
| `image.pullPolicy` | Image pull policy  | `IfNotPresent`                 |
| `image.pullSecret` | (DEPRECATED) imagePullSecret to use for private repository | `None`                         |
| `image.pullSecrets` | imagePullSecrets to use for private repository | `None`                         |

### Security

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `securityContext.fsGroup` | Group applied to mounted directories/files | `1000` |
| `containerSecurityContext.runAsUser` | User to run containers in sonarqube pod as, unless overwritten (such as for init-sysctl container) | `1000` |

### Elasticsearch

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `elasticsearch.configureNode` | [DEPRECATED] Use initSysctl.enabled instead. | `true` |
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
| `ingress.annotations` | Field to add extra annotations to the ingress | {`nginx.ingress.kubernetes.io/proxy-body-size=64m`} if `nginx.enabled=true` |

### Route

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `route.enabled` | Flag to enable OpenShift Route | `false` |
| `route.host` | Host of the route | `""` |
| `route.tls.termination` | TLS termination type. Currently supported values are `edge` and `passthrough` | `edge` |
| `route.annotations` | Optional field to add extra annotations to the route | `None` |
| `route.labels` | Route additional labels | `{}` |

### Probes

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `readinessProbe.initialDelaySeconds` | ReadinessProbe initial delay for SonarQube checking | `60` |
| `readinessProbe.periodSeconds` | ReadinessProbe period between checking SonarQube | `30` |
| `readinessProbe.failureThreshold` | ReadinessProbe threshold for marking as failed | `6` |
| `readinessProbe.timeoutSeconds`| ReadinessProbe timeout delay | `1` |
| `readinessProbe.sonarWebContext` | (DEPRECATED) SonarQube web context for readinessProbe, please use sonarWebContext at the value top level instead | `/` |
| `livenessProbe.initialDelaySeconds` | LivenessProbe initial delay for SonarQube checking | `60` |
| `livenessProbe.periodSeconds` | LivenessProbe period between checking SonarQube | `30` |
| `livenessProbe.sonarWebContext` | (DEPRECATED) SonarQube web context for LivenessProbe, please use sonarWebContext at the value top level instead | `/` |
| `livenessProbe.failureThreshold` | LivenessProbe threshold for marking as dead | `6` |
| `livenessProbe.timeoutSeconds`| LivenessProbe timeout delay | `1` |
| `startupProbe.initialDelaySeconds` | StartupProbe initial delay for SonarQube checking | `30` |
| `startupProbe.periodSeconds` | StartupProbe period between checking SonarQube | `10` |
| `startupProbe.sonarWebContext` | (DEPRECATED) SonarQube web context for StartupProbe, please use sonarWebContext at the value top level instead | `/` |
| `startupProbe.failureThreshold` | StartupProbe threshold for marking as failed | `24` |
| `startupProbe.timeoutSeconds`| StartupProbe timeout delay | `1` |

### InitContainers

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `initContainers.image` | Change init container image | `busybox:1.36` |
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
| `initSysctl.image` | Change init sysctl container image | `busybox:1.36` |
| `initSysctl.securityContext` | InitSysctl container security context | `{privileged: true}` |
| `initSysctl.resources` | InitSysctl container resource requests & limits | `{}` |
| `initFs.enabled` | Enable file permission change with init container | `true` |
| `initFs.image` | InitFS container image | `busybox:1.36` |
| `initFs.securityContext.privileged` | InitFS container needs to run privileged | `true` |

### Monitoring (Prometheus Exporter)

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `prometheusExporter.enabled` | Use the Prometheus JMX exporter | `false` |
| `prometheusExporter.version` | jmx_prometheus_javaagent version to download from Maven Central | `0.17.2` |
| `prometheusExporter.noCheckCertificate` | Flag to not check server's certificate when downloading jmx_prometheus_javaagent | `false` |
| `prometheusExporter.webBeanPort` | Port where the jmx_prometheus_javaagent exposes the metrics for the webBean | `8000` |
| `prometheusExporter.ceBeanPort` | Port where the jmx_prometheus_javaagent exposes the metrics for the ceBean | `8001` |
| `prometheusExporter.downloadURL` | Alternative full download URL for the jmx_prometheus_javaagent.jar (overrides `prometheusExporter.version`) | `""` |
| `prometheusExporter.config` | Prometheus JMX exporter config yaml for the web process, and the CE process if `prometheusExporter.ceConfig` is not set | see `values.yaml` |
| `prometheusExporter.ceConfig` | Prometheus JMX exporter config yaml for the CE process (by default, `prometheusExporter.config` is used) | `None` |
| `prometheusExporter.httpProxy` | HTTP proxy for downloading JMX agent | `""` |
| `prometheusExporter.httpsProxy` | HTTPS proxy for downloading JMX agent | `""` |
| `prometheusExporter.noProxy` | No proxy for downloading JMX agent | `""` |
| `prometheusExporter.securityContext` | Security context for downloading the jmx agent | see `values.yaml` |

### Monitoring (Prometheus PodMonitor)

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `prometheusMonitoring.podMonitor.enabled` | Enable Prometheus PodMonitor | `false` |
| `prometheusMonitoring.podMonitor.namespace` | Specify a custom namespace where the PodMonitor will be created | `default` |
| `prometheusMonitoring.podMonitor.interval` | Specify the interval how often metrics should be scraped | `30s` |
| `prometheusMonitoring.podMonitor.scrapeTimeout` | Specify the timeout after a scrape is ended | `None` |
| `prometheusMonitoring.podMonitor.jobLabel` |  Name of the label on target services that prometheus uses as job name | `None` |


### Plugins

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `plugins.install` | Link(s) to the plugin JARs to download and install | `[]` |
| `plugins.resources` | Plugin Pod resource requests & limits | `{}` |
| `plugins.httpProxy` | For use behind a corporate proxy when downloading plugins | `""` |
| `plugins.httpsProxy` | For use behind a corporate proxy when downloading plugins | `""` |
| `plugins.noProxy` | For use behind a corporate proxy when downloading plugins | `""` |
| `plugins.image` | Image for plugins container | `""`|
| `plugins.resources` | Resources for plugins container | `{}`                              |
| `plugins.netrcCreds` | Name of the secret containing .netrc file to use creds when downloading plugins | `""` |
| `plugins.noCheckCertificate` | Flag to not check server's certificate when downloading plugins | `false` |
| `plugins.securityContext` | Security context for the container to download plugins | see `values.yaml` |

### SonarQube Specific

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `jvmOpts` | (DEPRECATED) Values to add to SONARQUBE_WEB_JVM_OPTS | `""` |
| `jvmCeOpts` | (DEPRECATED) Values to add to SONAR_CE_JAVAOPTS | `""` |
| `sonarqubeFolder` | Directory name of SonarQube | `/opt/sonarqube` |
| `sonarProperties` | Custom `sonar.properties` key-value pairs (e.g., "sonarProperties.sonar.forceAuthentication=true") | `None` |
| `sonarSecretProperties` | Additional `sonar.properties` key-value pairs to load from a secret | `None` |
| `sonarSecretKey` | Name of existing secret used for settings encryption | `None` |
| `monitoringPasscode` | Value for sonar.web.systemPasscode needed for LivenessProbes (encoded to Base64 format) | `define_it` |
| `monitoringPasscodeSecretName` | Name of the secret where to load `monitoringPasscode` | `None` |
| `monitoringPasscodeSecretKey` | Key of an existing secret containing `monitoringPasscode` | `None` |
| `extraContainers` | Array of extra containers to run alongside the `sonarqube` container (aka. Sidecars) | `[]` |
| `extraVolumes` | Array of extra volumes to add to the SonarQube deployment | `[]` |
| `extraVolumeMounts` | Array of extra volume mounts to add to the SonarQube deployment | `[]` |

### Resources

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `resources.requests.memory` | SonarQube memory request | `2Gi` |
| `resources.requests.cpu` | SonarQube cpu request | `400m` |
| `resources.limits.memory` | SonarQube memory limit | `4Gi` |
| `resources.limits.cpu` | SonarQube cpu limit | `800m` |

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

### JDBC Overwrite

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `jdbcOverwrite.enable` | Enable JDBC overwrites for external Databases (disables `postgresql.enabled`) | `false` |
| `jdbcOverwrite.jdbcUrl` | The JDBC url to connect the external DB | `jdbc:postgresql://myPostgress/myDatabase?socketTimeout=1500` |
| `jdbcOverwrite.jdbcUsername` | The DB user that should be used for the JDBC connection | `sonarUser` |
| `jdbcOverwrite.jdbcPassword` | The DB password that should be used for the JDBC connection (Use this if you don't mind the DB password getting stored in plain text within the values file) | `sonarPass` |
| `jdbcOverwrite.jdbcSecretName` | Alternatively, use a pre-existing k8s secret containing the DB password | `None` |
| `jdbcOverwrite.jdbcSecretPasswordKey` | If the pre-existing k8s secret is used this allows the user to overwrite the 'key' of the password property in the secret | `None` |

### Bundled Postgres Chart

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `postgresql.enabled` | Set to `false` to use external server  | `true` |
| `postgresql.existingSecret` |  existingSecret Name of existing secret to use for PostgreSQL passwords | `nil` |
| `postgresql.postgresqlServer` | (DEPRECATED) Hostname of the external Postgresql server | `nil` |
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

| Parameter                    | Description                                                   | Default |
|------------------------------|---------------------------------------------------------------| ------- |
| `tests.enabled`              | Flag that allows tests to be excluded from the generated yaml | `true` |
| `tests.image`                | Change test container image                                   | `` |

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
| `account.adminPassword` | Custom admin password | `admin` |
| `account.currentAdminPassword` | Current admin password | `admin` |
| `account.adminPasswordSecretName` | Secret containing `password` (custom password) and `currentPassword` (current password) keys for admin | `None` |
| `account.resources.requests.memory` | Memory request for Admin hook | `128Mi` |
| `account.resources.requests.cpu` | CPU request for Admin hook | `100m` |
| `account.resources.limits.memory` | Memory limit for Admin hook | `128Mi` |
| `account.resources.limits.cpu` | CPU limit for Admin hook | `100m` |
| `account.sonarWebContext` | (DEPRECATED) SonarQube web context for Admin hook. please use sonarWebContext at the value top level instead | `nil` |
| `account.securityContext` | SecurityContext for change-password-hook | `{}` |
| `curlContainerImage` | Curl container image | `curlimages/curl:8.2.1` |
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

Since SonarQube comes bundled with an Elasticsearch instance, some [bootstrap checks](https://www.elastic.co/guide/en/elasticsearch/reference/master/bootstrap-checks.html) of the host settings are done at start.

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
