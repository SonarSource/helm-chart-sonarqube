# SonarQube

Code better in more than 30 languages. Improve Code Quality and Code Security throughout your workflow. [SonarQube Server](https://www.sonarsource.com/products/sonarqube/) can detect Bugs, Vulnerabilities, Security Hotspots and Code Smells and give you the guidance to fix them.

## Introduction

This chart bootstraps an instance of the latest SonarQube Server version with a PostgreSQL database.

The latest version of the chart installs the latest SonarQube version.

To install the version of the chart for SonarQube 9.9 LTA, please read the section [below](#installing-the-sonarqube-99-lta-chart). Deciding between LTA and Latest? [This may help](https://www.sonarsource.com/products/sonarqube/downloads/lts/).

Please note that this chart only supports SonarQube Server Developer and Enterprise editions and SonarQube Community Build. For SonarQube Server Data Center Edition refer to this [chart](https://artifacthub.io/packages/helm/sonarqube/sonarqube-dce).

## Default Versions

SonarQube Server Version: `2025.3.1`

SonarQube Community Build: `25.5.0.107428`. If you want the use a more recent SonarQube Community Build, please set the `community.buildNumber` with the desired version.

## Kubernetes and Openshift Compatibility

Supported Kubernetes Versions: From `1.30` to `1.32`

Supported Openshift Versions: From `4.11` to `4.17`

## Installing SonarQube Server

Here is an example of how to install the SonarQube Server Developer edition:

```bash
helm repo add sonarqube https://SonarSource.github.io/helm-chart-sonarqube
helm repo update
kubectl create namespace sonarqube
export MONITORING_PASSCODE="yourPasscode"
helm upgrade --install -n sonarqube sonarqube sonarqube/sonarqube --set edition=developer,monitoringPasscode=$MONITORING_PASSCODE
```

The above command deploys SonarQube on the Kubernetes cluster in the default configuration in the sonarqube namespace.
If you are interested in deploying SonarQube on Openshift, please check the [dedicated section](#openshift).

The [configuration](#configuration) section lists the parameters that can be configured during installation.

The default login is admin/admin.

## Installing SonarQube Community Build

The SonarQube Community Edition has been replaced by the SonarQube Community Build.
If you want to install the SonarQube Community Build chart, please set `community.enabled` to `true`.

This chart by default installs the SonarQube Community Build's latest version available at the time of the Helm chart release.
If you want the use a more recent SonarQube Community Build, please set the `community.buildNumber` with the desired version.

## Upgrading to SonarQube Server 2025.1 LTA

When upgrading to SonarQube Server 2025.1 LTA from a previous versions, you should read carefully [the official documentation](https://docs.sonarsource.com/sonarqube-server/latest/server-upgrade-and-maintenance/upgrade/upgrade-the-server/determine-path/) and determine the right upgrade path based on your current SonarQube Server version.

When upgrading to the 2025.1 LTA version, you will experience a few changes.

* The `monitoringPasscode` needs to be set by the users. Set either that or `monitoringPasscodeSecretName` and `monitoringPasscodeSecretKey`.
* The `edition` parameter is now required to be explicitly set by the user to either `developer` or `enterprise`.
* Users that want to install the SonarQube Community Build, must set `community.enabled` to `true` (check the [dedicated section](#installing-the-sonarqube-community-build-chart)).

## Installing previous chart versions

### Installing the SonarQube 9.9 LTA chart

The version of the chart for the SonarQube 9.9 LTA is being distributed as the `8.x.x` version of this chart.

In order to use it, please set the version constraint `~8`, which is equivalent to `>=8.0.0 && <= 9.0.0`. That version parameter **must** be used in every helm related command including `install`, `upgrade`, `template`, and `diff` (don't treat this as an exhaustive list).

Example:

```Bash
helm upgrade --install -n sonarqube --version '~8' sonarqube sonarqube/sonarqube
```

To upgrade from the old and unmaintained [sonarqube-lts chart](https://artifacthub.io/packages/helm/sonarqube/sonarqube-lts), please follow the steps described [in this section](#upgrade-from-the-old-sonarqube-lts-to-this-chart).

## How to use it

Take some time to read the Deploy on [SonarQube on Kubernetes](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/deploy-on-kubernetes/server/introduction/) page.
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

Please read the official documentation prerequisites [here](https://docs.sonarsource.com/sonarqube/latest/requirements/prerequisites-and-overview/).

### Kubernetes - Pod Security Standards

Here is the list of containers that are compatible with the [Pod Security levels](https://kubernetes.io/docs/concepts/security/pod-security-admission/#pod-security-levels):

* privileged:
  * `init-sysctl`
* baseline:
  * `init-fs`
* restricted:
  * SQ application containers
  * SQ init containers.
  * postgresql containers.

This is achieved by setting this SecurityContext as default on **most** containers:

```yaml
allowPrivilegeEscalation: false
runAsNonRoot: true
runAsUser: 1000
runAsGroup: 0
seccompProfile:
  type: RuntimeDefault
capabilities:
  drop: ["ALL"]
readOnlyRootFilesystem: true
```

Based on that, one can run the SQ helm chart in a full restricted namespace, by deactivating the `initSysctl.enabled` and `initFs.enabled` parameters, which require root access.

Please take a look at [production-use-case](#production-use-case) for more information or directly at the values.yaml file.

### Elasticsearch prerequisites

SonarQube runs Elasticsearch under the hood.

Elasticsearch is rolling out (strict) prerequisites that cannot be disabled when running in production context (see [this](https://www.elastic.co/blog/bootstrap_checks_annoying_instead_of_devastating) blog post regarding bootstrap checks, and the [official guide](https://www.elastic.co/guide/en/elasticsearch/reference/5.0/bootstrap-checks.html)).

Because of such constraints, even when running in Docker containers, SonarQube requires some settings at the host/kernel level.

Please carefully read the following and make sure these configurations are set up at the host level:

* [vm.max_map_count](https://www.elastic.co/guide/en/elasticsearch/reference/current/vm-max-map-count.html#vm-max-map-count)
* [seccomp filter should be available](https://github.com/SonarSource/docker-sonarqube/issues/614)

In general, please carefully read the Elasticsearch's [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/system-config.html).

### Production use case

The SonarQube helm chart is packed with multiple features enabling users to install and test SonarQube on Kubernetes easily.

Nonetheless, if you intend to run a production-grade SonarQube please follow these recommendations.

* Set `ingress-nginx.enabled` to **false**. This parameter would run the nginx chart. This is useful for testing purposes only. Ingress controllers are critical Kubernetes components, we advise users to install their own.
* Set `postgresql.enabled` to **false**. This parameter would run the postgresql pre-2022 bitnami chart. That is useful for testing purposes, however, given that the database is at the hearth of SonarQube, we advise users to be careful with it and use a well-maintained database as a service or deploy their own database on top of Kubernetes.
* Set `initSysctl.enabled` to **false**. This parameter would run **root** `sysctl` commands, while those sysctl-related values should be set by the Kubernetes administrator at the node level (see [here](#elasticsearch-prerequisites))
* Set `initFs.enabled` to **false**. This parameter would run **root** `chown` commands. The parameter exists to fix non-posix, CSI, or deprecated drivers.

#### Cpu and memory settings

Monitoring cpu and memory is an important part of software reliability. The SonarQube helm chart comes with default values for cpu and memory requests and limits. Those memory values are matching the default SonarQube JVM Xmx and Xms values.

Xmx defines the maximum size of the JVM heap, this is **not** the maximum memory the JVM can allocate.

For this reason, it is recommended to set Xmx to the ~80% of the total amount of memory available on the machine (in Kubernetes, this corresponds to requests and limits).

Please find here the default SonarQube Xmx parameters to setup the memory requests and limits accordingly.

| SonarQube Offering | Sum of Xmx |
| ------------------ | ---------- |
| community build    | 1536M      |
| developer edition  | 1536M      |
| enterprise edition | 5G         |

The default request and limit for this chart are set to 2048M and 6144M, to comply with the 3 editions and the 80% rule mentioned above.

Please feel free to adjust those values to your needs. However, given that memory is a “non-compressible” resource, we advise you to set the memory requests and limits to the **same**, making memory a guaranteed resource. This is needed especially for production use cases.

To get some guidance when setting the Xmx and Xms values, please refer to this [documentation](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/configure-and-operate-a-server/environment-variables/) and set the environment variables or sonar.properties accordingly.

## Upgrade

1. Read through the [SonarQube Upgrade Guide](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/upgrade-the-server/roadmap/) to familiarize yourself with the general upgrade process (most importantly, back up your database)
2. Change the SonarQube version on `values.yaml`
3. Redeploy SonarQube with the same helm chart (see [Install instructions](#installing-the-chart))
4. Browse to <http://yourSonarQubeServerURL/setup> and follow the setup instructions
5. Reanalyze your projects to get fresh data

### Upgrade from the old sonarqube-lts to this chart

Please refer to the Helm upgrade section accessible [here](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/upgrade-the-server/upgrade/#upgrade-from-89x-lts-to-99x-lts).

## Ingress usage

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

This Helm chart offers the possibility to monitor SonarQube with Prometheus. You can find [Information on SonarQube monitoring on Kubernetes](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/deploy-on-kubernetes/set-up-monitoring/introduction/) in the SonarQube documentation.

### Export JMX metrics

The prometheus exporter (`prometheusExporter.enabled=true`) converts the JMX metrics into a format that Prometheus can understand. After the metrics are exported, you can connect your Prometheus instance and scrape them.

Per default the JMX metrics for the Web Bean and the CE Bean are exposed on port 8000 and 8001. These values can be configured with `prometheusExporter.webBeanPort` and `prometheusExporter.ceBeanPort`.

### PodMonitor

If a Prometheus Operator is deployed in your cluster, you can enable a PodMonitor resource with `prometheusMonitoring.podMonitor.enabled`. It scrapes the Prometheus endpoint `/api/monitoring/metrics` exposed by the SonarQube application.

If running on OpenShift, make sure your account has permissions to create PodMonitor resources under the monitoring.coreos.com/v1 apiVersion.

## OpenShift installation

The chart can be installed on OpenShift by setting `OpenShift.enabled=true`. Among the others, please note that this value will disable the initContainer that performs the settings required by Elasticsearch (see [here](#elasticsearch-prerequisites)). Furthermore, we strongly recommend following the [Production Use Case guidelines](#production-use-case).

Please note that `Openshift.createSCC` is deprecated and should be set to `false`. The default securityContext, together with the production configurations described [above](#production-use-case), is compatible with restricted SCCv2.

The below command will deploy SonarQube on the Openshift Kubernetes cluster. Please note this will use the embedded postgresql database and is not recommended for production.

```bash
helm repo add sonarqube https://SonarSource.github.io/helm-chart-sonarqube
helm repo update
kubectl create namespace sonarqube # If you dont have permissions to create the namespace, skip this step and replace all -n with an existing namespace name.
helm upgrade --install -n sonarqube sonarqube sonarqube/sonarqube \
  --set OpenShift.enabled=true \
  --set postgresql.securityContext.enabled=false \
  --set postgresql.containerSecurityContext.enabled=false
```

If you want to make your application publicly visible with Routes, you can set `OpenShift.route.enabled` to true. Please check the [configuration details](#openshift-1) to customize the Route base on your needs.

## License

SonarQube Community Build is released under the [GNU Lesser General Public License, Version 3.0⁠,](http://www.gnu.org/licenses/lgpl.txt) and packaged with [SSALv1](https://www.sonarsource.com/license/ssal/) analyzers. SonarQube Server Developer and Enterprise are licensed under [SonarQube Server Terms and Conditions](https://www.sonarsource.com/legal/sonarqube/terms-and-conditions/).

## Configuration

The following table lists the configurable parameters of the SonarQube chart and their default values.

### Global

| Parameter               | Description                                                                                                           | Default            |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------- | ------------------ |
| `deploymentType`        | (DEPRECATED) Deployment Type (supported values are `StatefulSet` or `Deployment`)                                     | `StatefulSet`      |
| `replicaCount`          | Number of replicas deployed (supported values are 0 and 1)                                                            | `1`                |
| `deploymentStrategy`    | Deployment strategy. Setting the strategy type is deprecated and it will be hardcoded to `Recreate`                   | `{type: Recreate}` |
| `priorityClassName`     | Schedule pods on priority (e.g. `high-priority`)                                                                      | `None`             |
| `schedulerName`         | Kubernetes scheduler name                                                                                             | `None`             |
| `affinity`              | Node / Pod affinities                                                                                                 | `{}`               |
| `tolerations`           | List of node taints to tolerate                                                                                       | `[]`               |
| `nodeSelector`          | Node labels for pod assignment                                                                                        | `{}`               |
| `hostAliases`           | Aliases for IPs in /etc/hosts                                                                                         | `[]`               |
| `podLabels`             | Map of labels to add to the pods                                                                                      | `{}`               |
| `env`                   | Environment variables to attach to the pods                                                                           | `{}`               |
| `annotations`           | SonarQube Pod annotations                                                                                             | `{}`               |
| `edition`               | SonarQube Edition to use (`developer` or `enterprise`).                                                               | `None`             |
| `community.enabled`     | Install SonarQube Community Build. When set to `true`, `edition` must not be set.                                     | `false`            |
| `community.buildNumber` | The SonarQube Community Build number to install                                                                       | `25.5.0.107428`   |
| `sonarWebContext`       | SonarQube web context, also serve as default value for `ingress.path`, `account.sonarWebContext` and probes path.     | ``                 |
| `httpProxySecret`       | Should contain `http_proxy`, `https_proxy` and `no_proxy` keys, will supersede every other proxy variables            | ``                 |
| `httpProxy`             | HTTP proxy for downloading JMX agent and install plugins, will supersede initContainer specific http proxy variables  | ``                 |
| `httpsProxy`            | HTTPS proxy for downloading JMX agent and install plugins, will supersede initContainer specific https proxy variable | ``                 |
| `noProxy`               | No proxy for downloading JMX agent and install plugins, will supersede initContainer specific no proxy variables      | ``                 |
| `ingress-nginx.enabled` | Install Nginx Ingress Helm                                                                                            | `false`            |

### NetworkPolicies

| Parameter                                 | Description                                                               | Default |
| ----------------------------------------- | ------------------------------------------------------------------------- | ------- |
| `networkPolicy.enabled`                   | Create NetworkPolicies                                                    | `false` |
| `networkPolicy.prometheusNamespace`       | Allow incoming traffic to monitoring ports from this namespace            | `nil`   |
| `networkPolicy.additionalNetworkPolicys`  | (DEPRECATED) Please use `networkPolicy.additionalNetworkPolicies` instead | `nil`   |
| `networkPolicy.additionalNetworkPolicies` | User defined NetworkPolicies (useful for external database)               | `nil`   |

### OpenShift

| Parameter                        | Description                                                                                         | Default                    |
| -------------------------------- | --------------------------------------------------------------------------------------------------- | -------------------------- |
| `OpenShift.enabled`              | Define if this deployment is for OpenShift                                                          | `false`                    |
| `OpenShift.createSCC`            | (DEPRECATED) If this deployment is for OpenShift, define if SCC should be created for sonarqube pod | `false`                    |
| `OpenShift.route.enabled`        | Flag to enable OpenShift Route                                                                      | `false`                    |
| `OpenShift.route.host`           | Host that points to the service                                                                     | `"sonarqube.your-org.com"` |
| `OpenShift.route.path`           | Path that the router watches for, to route traffic for to the service                               | `"/"`                      |
| `OpenShift.route.tls`            | TLS settings including termination type, certificates, insecure traffic, etc.                       | see `values.yaml`          |
| `OpenShift.route.wildcardPolicy` | The wildcard policy that is allowed where this route is exposed                                     | `None`                     |
| `OpenShift.route.annotations`    | Optional field to add extra annotations to the route                                                | `None`                     |
| `OpenShift.route.labels`         | Route additional labels                                                                             | `{}`                       |

### Image

| Parameter           | Description                                                | Default        |
| ------------------- | ---------------------------------------------------------- | -------------- |
| `image.repository`  | image repository                                           | `sonarqube`    |
| `image.tag`         | `sonarqube` image tag.                                     | `None`         |
| `image.pullPolicy`  | Image pull policy                                          | `IfNotPresent` |
| `image.pullSecret`  | (DEPRECATED) imagePullSecret to use for private repository | `None`         |
| `image.pullSecrets` | imagePullSecrets to use for private repository             | `None`         |

### Security

| Parameter                  | Description                                    | Default                                                                |
| -------------------------- | ---------------------------------------------- | ---------------------------------------------------------------------- |
| `securityContext`          | SecurityContext for the pod                    | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `containerSecurityContext` | SecurityContext for container in sonarqube pod | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |

### Elasticsearch

| Parameter                       | Description                                     | Default |
| ------------------------------- | ----------------------------------------------- | ------- |
| `elasticsearch.configureNode`   | [DEPRECATED] Use initSysctl.enabled instead.    | `false` |
| `elasticsearch.bootstrapChecks` | Enables/disables Elasticsearch bootstrap checks | `true`  |

### Service

| Parameter                          | Description                                        | Default     |
| ---------------------------------- | -------------------------------------------------- | ----------- |
| `service.type`                     | Kubernetes service type                            | `ClusterIP` |
| `service.externalPort`             | Kubernetes service port                            | `9000`      |
| `service.internalPort`             | Kubernetes container port                          | `9000`      |
| `service.labels`                   | Kubernetes service labels                          | `None`      |
| `service.annotations`              | Kubernetes service annotations                     | `None`      |
| `service.loadBalancerSourceRanges` | Kubernetes service LB Allowed inbound IP addresses | `None`      |
| `service.loadBalancerIP`           | Kubernetes service LB Optional fixed external IP   | `None`      |

### Ingress

| Parameter                      | Description                                                  | Default                                                                                                      |
| ------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| `nginx.enabled`                | (DEPRECATED) please use `ingress-nginx.enabled`              | `false`                                                                                                      |
| `ingress.labels`               | Ingress additional labels                                    | `{}`                                                                                                         |
| `ingress.hosts[0].name`        | Hostname to your SonarQube installation                      | `sonarqube.your-org.com`                                                                                     |
| `ingress.hosts[0].path`        | Path within the URL structure                                | `/`                                                                                                          |
| `ingress.hosts[0].serviceName` | Optional field to override the default serviceName of a path | `None`                                                                                                       |
| `ingress.hosts[0].servicePort` | Optional field to override the default servicePort of a path | `None`                                                                                                       |
| `ingress.tls`                  | Ingress secrets for TLS certificates                         | `[]`                                                                                                         |
| `ingress.ingressClassName`     | Optional field to configure ingress class name               | `None` OR `nginx` if `nginx.enabled` or `ingress-nginx.enabled`                                              |
| `ingress.annotations`          | Field to add extra annotations to the ingress                | {`nginx.ingress.kubernetes.io/proxy-body-size: "64m"`} if `ingress-nginx.enabled=true or nginx.enabled=true` |

### HttpRoute

| Parameter                    | Description                                                                                                   | Default |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------- | ------- |
| `httproute.enabled`          | Flag to enable GatewayAPI HttpRoute                                                                           | `False` |
| `httproute.gateway`          | Name of the gateway                                                                                           | `None`  |
| `httproute.gatewayNamespace` | (Optional) Name of the gateway namespace when located in a different namespace                                | `None`  |
| `httproute.hostnames`        | List of hostnames to match the HttpRoute against                                                              | `None`  |
| `httproute.labels`           | (Optional) List of extra labels to add to the HttpRoute                                                       | `None`  |
| `httproute.rules`            | (Optional) Extra Rules block of the HttpRoute. A default one is created with SonarWebContext and service port | `None`  |

### Probes

| Parameter                            | Description                                                                                                      | Default                                                        |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| `readinessProbe`                     | ReadinessProbe for SonarQube                                                                                     | `exec; curl api/system/status` see `values.yaml` for details   |
| `readinessProbe.initialDelaySeconds` | ReadinessProbe initial delay for SonarQube checking                                                              | `60`                                                           |
| `readinessProbe.periodSeconds`       | ReadinessProbe period between checking SonarQube                                                                 | `30`                                                           |
| `readinessProbe.failureThreshold`    | ReadinessProbe threshold for marking as failed                                                                   | `6`                                                            |
| `readinessProbe.timeoutSeconds`      | ReadinessProbe timeout delay                                                                                     | `1`                                                            |
| `readinessProbe.sonarWebContext`     | (DEPRECATED) SonarQube web context for readinessProbe, please use sonarWebContext at the value top level instead | `/`                                                            |
| `livenessProbe`                      | LivenessProbe for SonarQube                                                                                      | `exec: curl api/system/liveness` see `values.yaml` for details |
| `livenessProbe.initialDelaySeconds`  | LivenessProbe initial delay for SonarQube checking                                                               | `60`                                                           |
| `livenessProbe.periodSeconds`        | LivenessProbe period between checking SonarQube                                                                  | `30`                                                           |
| `livenessProbe.failureThreshold`     | LivenessProbe threshold for marking as failed                                                                    | `6`                                                            |
| `livenessProbe.timeoutSeconds`       | LivenessProbe timeout delay                                                                                      | `1`                                                            |
| `livenessProbe.sonarWebContext`      | (DEPRECATED) SonarQube web context for LivenessProbe, please use sonarWebContext at the value top level instead  | `/`                                                            |
| `startupProbe`                       | StartupProbe for SonarQube                                                                                       | `httpGet: api/system/status`                                   |
| `startupProbe.initialDelaySeconds`   | StartupProbe initial delay for SonarQube checking                                                                | `30`                                                           |
| `startupProbe.periodSeconds`         | StartupProbe period between checking SonarQube                                                                   | `10`                                                           |
| `startupProbe.failureThreshold`      | StartupProbe threshold for marking as failed                                                                     | `24`                                                           |
| `startupProbe.timeoutSeconds`        | StartupProbe timeout delay                                                                                       | `1`                                                            |
| `startupProbe.sonarWebContext`       | (DEPRECATED) SonarQube web context for StartupProbe, please use sonarWebContext at the value top level instead   | `/`                                                            |

### InitContainers

| Parameter                           | Description                                                                                                                           | Default                                                                |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `initContainers.image`              | Change init container image                                                                                                           | `"image.repository":"image.tag"`                                       |
| `initContainers.securityContext`    | SecurityContext for init containers                                                                                                   | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `initContainers.resources`          | Resources for init containers                                                                                                         | `{}`                                                                   |
| `extraInitContainers`               | Extra init containers to e.g. download required artifacts                                                                             | `{}`                                                                   |
| `caCerts.enabled`                   | Flag for enabling additional CA certificates                                                                                          | `false`                                                                |
| `caCerts.image`                     | Change init CA certificates container image                                                                                           | `"image.repository":"image.tag"`                                       |
| `caCerts.secret`                    | Name of the secret containing additional CA certificates. If defined, only secrets are going to be used.                              | `None`                                                                 |
| `caCerts.configMap.name`            | Name of the ConfigMap containing additional CA certificate. Ensure that `caCerts.secret` is not set if you want to use a `ConfigMap`. | `None`                                                                 |
| `caCerts.configMap.key`             | Name of the key containing the additional CA certificate                                                                              | `None`                                                                 |
| `caCerts.configMap.path`            | Filename that should be used for the given CA certificate                                                                             | `None`                                                                 |
| `initSysctl.enabled`                | Modify k8s worker to conform to system requirements                                                                                   | `true`                                                                 |
| `initSysctl.vmMaxMapCount`          | Set init sysctl container vm.max_map_count                                                                                            | `524288`                                                               |
| `initSysctl.fsFileMax`              | Set init sysctl container fs.file-max                                                                                                 | `131072`                                                               |
| `initSysctl.nofile`                 | Set init sysctl container open file descriptors limit                                                                                 | `131072`                                                               |
| `initSysctl.nproc`                  | Set init sysctl container open threads limit                                                                                          | `8192`                                                                 |
| `initSysctl.image`                  | Change init sysctl container image                                                                                                    | `"image.repository":"image.tag"`                                       |
| `initSysctl.securityContext`        | InitSysctl container security context                                                                                                 | `{privileged: true}`                                                   |
| `initSysctl.resources`              | InitSysctl container resource requests & limits                                                                                       | `{}`                                                                   |
| `initFs.enabled`                    | Enable file permission change with init container                                                                                     | `true`                                                                 |
| `initFs.image`                      | InitFS container image                                                                                                                | `"image.repository":"image.tag"`                                       |
| `initFs.securityContext.privileged` | InitFS container needs to run privileged                                                                                              | `true`                                                                 |

### Monitoring (Prometheus Exporter)

| Parameter                               | Description                                                                                                             | Default                                                                |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `prometheusExporter.enabled`            | Use the Prometheus JMX exporter                                                                                         | `false`                                                                |
| `prometheusExporter.version`            | jmx_prometheus_javaagent version to download from Maven Central                                                         | `0.17.2`                                                               |
| `prometheusExporter.noCheckCertificate` | Flag to not check server's certificate when downloading jmx_prometheus_javaagent                                        | `false`                                                                |
| `prometheusExporter.webBeanPort`        | Port where the jmx_prometheus_javaagent exposes the metrics for the webBean                                             | `8000`                                                                 |
| `prometheusExporter.ceBeanPort`         | Port where the jmx_prometheus_javaagent exposes the metrics for the ceBean                                              | `8001`                                                                 |
| `prometheusExporter.downloadURL`        | Alternative full download URL for the jmx_prometheus_javaagent.jar (overrides `prometheusExporter.version`)             | `""`                                                                   |
| `prometheusExporter.config`             | Prometheus JMX exporter config yaml for the web process, and the CE process if `prometheusExporter.ceConfig` is not set | see `values.yaml`                                                      |
| `prometheusExporter.ceConfig`           | Prometheus JMX exporter config yaml for the CE process (by default, `prometheusExporter.config` is used)                | `None`                                                                 |
| `prometheusExporter.httpProxy`          | HTTP proxy for downloading JMX agent                                                                                    | `""`                                                                   |
| `prometheusExporter.httpsProxy`         | HTTPS proxy for downloading JMX agent                                                                                   | `""`                                                                   |
| `prometheusExporter.noProxy`            | No proxy for downloading JMX agent                                                                                      | `""`                                                                   |
| `prometheusExporter.securityContext`    | Security context for downloading the jmx agent                                                                          | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |

### Monitoring (Prometheus PodMonitor)

| Parameter                                       | Description                                                                                                 | Default                    |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------- | -------------------------- |
| `prometheusMonitoring.podMonitor.enabled`       | Enable Prometheus PodMonitor                                                                                | `false`                    |
| `prometheusMonitoring.podMonitor.namespace`     | (DEPRECATED) This value should not be set, as the PodMonitor's namespace has to match the Release Namespace | `{{ .Release.Namespace }}` |
| `prometheusMonitoring.podMonitor.interval`      | Specify the interval how often metrics should be scraped                                                    | `30s`                      |
| `prometheusMonitoring.podMonitor.scrapeTimeout` | Specify the timeout after a scrape is ended                                                                 | `None`                     |
| `prometheusMonitoring.podMonitor.jobLabel`      | Name of the label on target services that prometheus uses as job name                                       | `None`                     |
| `prometheusMonitoring.podMonitor.labels`        | Additional labels to add to the PodMonitor                                                                  | `{}`                       |

### Plugins

| Parameter                    | Description                                                                     | Default                                                                |
| ---------------------------- | ------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `plugins.install`            | Link(s) to the plugin JARs to download and install                              | `[]`                                                                   |
| `plugins.resources`          | Plugin Pod resource requests & limits                                           | `{}`                                                                   |
| `plugins.httpProxy`          | For use behind a corporate proxy when downloading plugins                       | `""`                                                                   |
| `plugins.httpsProxy`         | For use behind a corporate proxy when downloading plugins                       | `""`                                                                   |
| `plugins.noProxy`            | For use behind a corporate proxy when downloading plugins                       | `""`                                                                   |
| `plugins.image`              | Image for plugins container                                                     | `"image.repository":"image.tag"`                                       |
| `plugins.resources`          | Resources for plugins container                                                 | `{}`                                                                   |
| `plugins.netrcCreds`         | Name of the secret containing .netrc file to use creds when downloading plugins | `""`                                                                   |
| `plugins.noCheckCertificate` | Flag to not check server's certificate when downloading plugins                 | `false`                                                                |
| `plugins.securityContext`    | Security context for the container to download plugins                          | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |

### SonarQube Specific

| Parameter                      | Description                                                                                                                              | Default          |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `jvmOpts`                      | (DEPRECATED) Values to add to `SONAR_WEB_JAVAOPTS`. Please set directly `SONAR_WEB_JAVAOPTS` or `sonar.web.javaOpts`                     | `""`             |
| `jvmCeOpts`                    | (DEPRECATED) Values to add to `SONAR_CE_JAVAOPTS`. Please set directly `SONAR_CE_JAVAOPTS` or `sonar.ce.javaOpts`                        | `""`             |
| `sonarqubeFolder`              | (DEPRECATED) Directory name of SonarQube, Due to 1-1 mapping between helm version and docker version, there is no need for configuration | `/opt/sonarqube` |
| `sonarProperties`              | Custom `sonar.properties` key-value pairs (e.g., "sonarProperties.sonar.log.level=DEBUG")                                       | `None`           |
| `sonarSecretProperties`        | Additional `sonar.properties` key-value pairs to load from a secret                                                                      | `None`           |
| `sonarSecretKey`               | Name of existing secret used for settings encryption                                                                                     | `None`           |
| `monitoringPasscode`           | Value for sonar.web.systemPasscode needed for LivenessProbes                                                                             | `None`           |
| `monitoringPasscodeSecretName` | Name of the secret where to load `monitoringPasscode`                                                                                    | `None`           |
| `monitoringPasscodeSecretKey`  | Key of an existing secret containing `monitoringPasscode`                                                                                | `None`           |
| `extraContainers`              | Array of extra containers to run alongside the `sonarqube` container (aka. Sidecars)                                                     | `[]`             |
| `extraVolumes`                 | Array of extra volumes to add to the SonarQube deployment                                                                                | `[]`             |
| `extraVolumeMounts`            | Array of extra volume mounts to add to the SonarQube deployment                                                                          | `[]`             |

### Resources

| Parameter                              | Description               | Default |
| -------------------------------------- | ------------------------- | ------- |
| `resources.requests.memory`            | SonarQube memory request  | `2048M` |
| `resources.requests.cpu`               | SonarQube cpu request     | `400m`  |
| `resources.requests.ephemeral-storage` | SonarQube storage request | `1536M` |
| `resources.limits.memory`              | SonarQube memory limit    | `6144M` |
| `resources.limits.cpu`                 | SonarQube cpu limit       | `800m`  |
| `resources.limits.ephemeral-storage`   | SonarQube storage limit   | `500Gi` |

### Persistence

| Parameter                   | Description                                       | Default         |
| --------------------------- | ------------------------------------------------- | --------------- |
| `persistence.enabled`       | Flag for enabling persistent storage              | `false`         |
| `persistence.annotations`   | Kubernetes pvc annotations                        | `{}`            |
| `persistence.existingClaim` | Do not create a new PVC but use this one          | `None`          |
| `persistence.storageClass`  | Storage class to be used                          | `""`            |
| `persistence.accessMode`    | Volumes access mode to be set                     | `ReadWriteOnce` |
| `persistence.size`          | Size of the volume                                | `5Gi`           |
| `persistence.volumes`       | (DEPRECATED) Please use extraVolumes instead      | `[]`            |
| `persistence.mounts`        | (DEPRECATED) Please use extraVolumeMounts instead | `[]`            |
| `persistence.uid`           | UID used for init-fs container                    | `1000`          |
| `persistence.guid`          | GUID used for init-fs container                   | `0`             |
| `emptyDir`                  | Configuration of resources for `emptyDir`         | `{}`            |

### JDBC Overwrite

| Parameter                                   | Description                                                                                                                                                    | Default                                    |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| `jdbcOverwrite.enable`                      | (DEPRECATED) Enable JDBC overwrites for external Databases (disables `postgresql.enabled`) ,Please use jdbcOverwrite.enabled instead                           | `false`                                    |
| `jdbcOverwrite.enabled`                     | Enable JDBC overwrites for external Databases (disables `postgresql.enabled`)                                                                                  | `false`                                    |
| `jdbcOverwrite.jdbcUrl`                     | The JDBC url to connect the external DB                                                                                                                        | `jdbc:postgresql://myPostgress/myDatabase` |
| `jdbcOverwrite.jdbcUsername`                | The DB user that should be used for the JDBC connection                                                                                                        | `sonarUser`                                |
| `jdbcOverwrite.jdbcPassword`                | (DEPRECATED) The DB password that should be used for the JDBC connection, please use `jdbcOverwrite.jdbcSecretName`  and `jdbcOverwrite.jdbcSecretPasswordKey` | `sonarPass`                                |
| `jdbcOverwrite.jdbcSecretName`              | Alternatively, use a pre-existing k8s secret containing the DB password                                                                                        | `None`                                     |
| `jdbcOverwrite.jdbcSecretPasswordKey`       | If the pre-existing k8s secret is used this allows the user to overwrite the 'key' of the password property in the secret                                      | `None`                                     |
| `jdbcOverwrite.oracleJdbcDriver.url`        | The URL of the Oracle JDBC driver to be downloaded                                                                                                             | `None`                                     |
| `jdbcOverwrite.oracleJdbcDriver.netrcCreds` | Name of the secret containing .netrc file to use creds when downloading the Oracle JDBC driver                                                                 | `None`                                     |

### Bundled PostgreSQL Chart (DEPRECATED)

The bundled PostgreSQL Chart is deprecated. Please see <https://artifacthub.io/packages/helm/sonarqube/sonarqube#production-use-case> for more information.

| Parameter                                                | Description                                                            | Default         |
| -------------------------------------------------------- | ---------------------------------------------------------------------- | --------------- |
| `postgresql.enabled`                                     | Set to `false` to use external server                                  | `true`          |
| `postgresql.existingSecret`                              | existingSecret Name of existing secret to use for PostgreSQL passwords | `nil`           |
| `postgresql.postgresqlServer`                            | (DEPRECATED) Hostname of the external PostgreSQL server                | `nil`           |
| `postgresql.postgresqlUsername`                          | PostgreSQL database user                                               | `sonarUser`     |
| `postgresql.postgresqlPassword`                          | PostgreSQL database password                                           | `sonarPass`     |
| `postgresql.postgresqlDatabase`                          | PostgreSQL database name                                               | `sonarDB`       |
| `postgresql.service.port`                                | PostgreSQL port                                                        | `5432`          |
| `postgresql.resources.requests.memory`                   | PostgreSQL memory request                                              | `256Mi`         |
| `postgresql.resources.requests.cpu`                      | PostgreSQL cpu request                                                 | `250m`          |
| `postgresql.resources.limits.memory`                     | PostgreSQL memory limit                                                | `2Gi`           |
| `postgresql.resources.limits.cpu`                        | PostgreSQL cpu limit                                                   | `2`             |
| `postgresql.persistence.enabled`                         | PostgreSQL persistence en/disabled                                     | `true`          |
| `postgresql.persistence.accessMode`                      | PostgreSQL persistence accessMode                                      | `ReadWriteOnce` |
| `postgresql.persistence.size`                            | PostgreSQL persistence size                                            | `20Gi`          |
| `postgresql.persistence.storageClass`                    | PostgreSQL persistence storageClass                                    | `""`            |
| `postgresql.securityContext.enabled`                     | PostgreSQL securityContext en/disabled                                 | `false`         |
| `postgresql.securityContext`                             | PostgreSQL securityContext                                             | `false`         |
| `postgresql.volumePermissions.enabled`                   | PostgreSQL vol permissions en/disabled                                 | `false`         |
| `postgresql.volumePermissions.securityContext.runAsUser` | PostgreSQL vol permissions secContext runAsUser                        | `0`             |
| `postgresql.shmVolume.chmod.enabled`                     | PostgreSQL shared memory vol en/disabled                               | `false`         |
| `postgresql.serivceAccount.enabled`                      | PostgreSQL service Account creation en/disabled                        | `false`         |
| `postgresql.serivceAccount.name`                         | PostgreSQL service Account name                                        | `""`            |

### Tests

| Parameter                       | Description                                                   | Default                          |
| ------------------------------- | ------------------------------------------------------------- | -------------------------------- |
| `tests.enabled`                 | Flag that allows tests to be excluded from the generated yaml | `true`                           |
| `tests.image`                   | Set the test container image                                  | `"image.repository":"image.tag"` |
| `tests.resources.limits.cpu`    | CPU limit for test container                                  | `500m`                           |
| `tests.resources.limits.memory` | Memory limit for test container                               | `200M`                           |

### ServiceAccount

| Parameter                       | Description                                                                                                                                                                                           | Default               |
| ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------- |
| `serviceAccount.create`         | If set to true, create a service account                                                                                                                                                              | `false`               |
| `serviceAccount.name`           | Name of the service account to create/use                                                                                                                                                             | `sonarqube-sonarqube` |
| `serviceAccount.automountToken` | Manage `automountServiceAccountToken` field for mounting service account credentials. Please note that this will set the default value used by SQ Pods, regardless of the service account being used. | `false`               |
| `serviceAccount.annotations`    | Additional service account annotations                                                                                                                                                                | `{}`                  |

### ExtraConfig

| Parameter                | Description                                                 | Default |
| ------------------------ | ----------------------------------------------------------- | ------- |
| `extraConfig.secrets`    | A list of `Secret`s (which must contain key/value pairs)    | `[]`    |
| `extraConfig.configmaps` | A list of `ConfigMap`s (which must contain key/value pairs) | `[]`    |

### SetAdminPassword

| Parameter                                    | Description                                                                                            | Default                                                                |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------- |
| `setAdminPassword.newPassword`               | Custom admin password                                                                                  | `AdminAdmin_12$`                                                       |
| `setAdminPassword.currentPassword`           | Current admin password                                                                                 | `admin`                                                                |
| `setAdminPassword.passwordSecretName`        | Secret containing `password` (custom password) and `currentPassword` (current password) keys for admin | `None`                                                                 |
| `setAdminPassword.resources.requests.memory` | Memory request for Admin hook                                                                          | `128Mi`                                                                |
| `setAdminPassword.resources.requests.cpu`    | CPU request for Admin hook                                                                             | `100m`                                                                 |
| `setAdminPassword.resources.limits.memory`   | Memory limit for Admin hook                                                                            | `128Mi`                                                                |
| `setAdminPassword.resources.limits.cpu`      | CPU limit for Admin hook                                                                               | `100m`                                                                 |
| `setAdminPassword.securityContext`           | SecurityContext for change-password-hook                                                               | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `setAdminPassword.image`                     | Curl container image                                                                                   | `"image.repository":"image.tag"`                                       |
| `setAdminPassword.annotations`               | Custom annotations for admin hook Job                                                                  | `{}`                                                                   |

### Advanced Options

| Parameter                           | Description                                                                                                                                                                    | Default                                                                |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------- |
| `account.adminPassword`             | (DEPRECATED) Custom admin password. Please use `setAdminPassword.newPassword` instead.                                                                                         | `AdminAdmin_12$`                                                       |
| `account.currentAdminPassword`      | (DEPRECATED) Current admin password. Please use `setAdminPassword.currentPassword` instead.                                                                                    | `admin`                                                                |
| `account.adminPasswordSecretName`   | (DEPRECATED) Secret containing `password` (custom password) and `currentPassword` (current password) keys for admin. Please use `setAdminPassword.passwordSecretName` instead. | `None`                                                                 |
| `account.resources.requests.memory` | (DEPRECATED) Memory request for Admin hook. Please use `setAdminPassword.resources.requests.memory` instead.                                                                   | `128Mi`                                                                |
| `account.resources.requests.cpu`    | (DEPRECATED) CPU request for Admin hook. Please use `setAdminPassword.resources.requests.cpu` instead.                                                                         | `100m`                                                                 |
| `account.resources.limits.memory`   | (DEPRECATED) Memory limit for Admin hook. Please use `setAdminPassword.resources.limits.memory` instead.                                                                       | `128Mi`                                                                |
| `account.resources.limits.cpu`      | (DEPRECATED) CPU limit for Admin hook. Please use `setAdminPassword.resources.limits.cpu` instead.                                                                             | `100m`                                                                 |
| `account.sonarWebContext`           | (DEPRECATED) SonarQube web context for Admin hook. Please use `sonarWebContext` at the value top level instead                                                                 | `nil`                                                                  |
| `account.securityContext`           | (DEPRECATED) SecurityContext for change-password-hook. Please use `setAdminPassword.securityContext` instead.                                                                  | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `curlContainerImage`                | (DEPRECATED) Curl container image. Please use `setAdminPassword.image` instead.                                                                                                | `"image.repository":"image.tag"`                                       |
| `adminJobAnnotations`               | (DEPRECATED) Custom annotations for admin hook Job. Please use `setAdminPassword.annotations` instead.                                                                         | `{}`                                                                   |
| `terminationGracePeriodSeconds`     | Configuration of `terminationGracePeriodSeconds`                                                                                                                               | `60`                                                                   |

You can also configure values for the PostgreSQL database via the PostgreSQL [Chart](https://hub.helm.sh/charts/bitnami/postgresql)

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

This chart offers the option to use an initContainer in privileged mode to automatically set certain kernel settings on the kube worker. While this can ensure proper functionality of Elasticsearch, modifying the underlying kernel settings on the Kubernetes node can impact other users. It may be best to work with your cluster administrator to either provide specific nodes with the proper kernel settings, or ensure they are set cluster wide.

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
