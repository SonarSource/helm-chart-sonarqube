# SonarQube

Code better in up to 27 languages. Improve Code Quality and Code Security throughout your workflow. [SonarQube](https://www.sonarsource.com/products/sonarqube/) can detect Bugs, Vulnerabilities, Security Hotspots, and Code Smells plus gives you the guidance to fix them.

## Introduction

This helm chart bootstraps a SonarQube Data Center Edition cluster with a PostgreSQL database.

The latest version of the chart installs the latest SonarQube version.

To install the version of the chart for SonarQube 9.9 LTA, please read the section [below](#installing-the-sonarqube-99-lts-chart). Deciding between LTA and Latest? [This may help](https://www.sonarsource.com/products/sonarqube/downloads/lts/)

Please note that this chart does NOT support SonarQube Community, Developer, and Enterprise Editions.

## Compatibility

Compatible SonarQube Version: `10.7.0`

Supported Kubernetes Versions: From `1.24` to `1.31`
Supported Openshift Versions: From `4.11` to `4.16`

## Installing the chart

> **_NOTE:_**  Please refer to [the official page](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/deploy-on-kubernetes/cluster/) for further information on how to install and tune the helm chart specifications.

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

The above command deploys SonarQube on the Kubernetes cluster in the default configuration in the sonarqube namespace.
If you are interested in deploying SonarQube on Openshift, please check the [dedicated section](#openshift).

The [configuration](#configuration) section lists the parameters that can be configured during installation. 

The default login is admin/admin.

## Installing the SonarQube 9.9 LTA chart

The version of the chart for the SonarQube 9.9 LTA is being distributed as the `7.x.x` version of this chart.

In order to use it, please set the version constraint `~7`, which is equivalent to `>=7.0.0 && <= 8.0.0`. That version parameter **must** be used in every helm related command including `install`, `upgrade`, `template`, and `diff` (don't treat this as an exhaustive list).

Example:

```Bash
helm upgrade --install -n sonarqube-dce --version '~7' sonarqube sonarqube/sonarqube-dce --set ApplicationNodes.jwtSecret=$JWT_SECRET
```

## How to use it

Take some time to read the Deploy [SonarQube on Kubernetes](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/deploy-on-kubernetes/cluster/) page.
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

```Yaml
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

In general, please carefully read the Elasticsearch's [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/system-config.html) and specifically [here](https://www.elastic.co/guide/en/cloud-on-k8s/current/k8s-virtual-memory.html) for tutorial on how to change those parameters.

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

| Edition                             | Sum of Xmx |
| ----------------------------------- | ---------- |
| datacenter edition searchNodes      | 2G         |
| datacenter edition ApplicationNodes | 3G         |

To comply with the 80% rule mentioned above, we set the following default values:

* searchNodes.resources.memory.request/limit=3072M
* ApplicationNodes.resources.memory.request/limit=4096M

Please feel free to adjust those values to your needs. However, given that memory is a “non-compressible” resource, we advise you to set the memory requests and limits to the **same**, making memory a guaranteed resource. This is needed especially for production use cases.

To get some guidance when setting the Xmx and Xms values, please refer to this [documentation](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/configure-and-operate-a-server/environment-variables/) and set the environment variables or sonar.properties accordingly.

## Ingress use cases

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

This Helm chart offers the possibility to monitor SonarQube with Prometheus. You can find [Information on the SonarQube monitoring on Kubernetes](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/deploy-on-kubernetes/set-up-monitoring/introduction/) in the SonarQube documentation.

### Export JMX metrics

The prometheus exporter (`ApplicationNodes.prometheusExporter.enabled=true`) converts the JMX metrics into a format that Prometheus can understand. After the metrics are exported, you can connect your Prometheus instance and scrape them.

Per default the JMX metrics for the Web Bean and the CE Bean are exposed on port 8000 and 8001. These values can be configured with `ApplicationNodes.prometheusExporter.webBeanPort` and `ApplicationNodes.prometheusExporter.ceBeanPort`.

### PodMonitor

If a Prometheus Operator is deployed in your cluster, you can enable a PodMonitor resource with `ApplicationNodes.prometheusMonitoring.podMonitor.enabled`. It scrapes the Prometheus endpoint `/api/monitoring/metrics` exposed by the SonarQube application.

If running on OpenShift, make sure your account has permissions to create PodMonitor resources under the monitoring.coreos.com/v1 apiVersion.

## OpenShift

The chart can be installed on OpenShift by setting `OpenShift.enabled=true`. Among the others, please note that this value will disable the initContainer that performs the settings required by Elasticsearch (see [here](#elasticsearch-prerequisites)). Furthermore, we strongly recommend following the [Production Use Case guidelines](#production-use-case).

Please note that `Openshift.createSCC` is deprecated and should be set to `false`. The default securityContext, together with the production configurations described [above](#production-use-case), is compatible with restricted SCCv2.

The below command will deploy SonarQube on the Openshift Kubernetes cluster. Please note this will use the embedded postgresql database and is not recommended for production.

```bash
helm repo add sonarqube https://SonarSource.github.io/helm-chart-sonarqube
helm repo update
kubectl create namespace sonarqube-dce # If you dont have permissions to create the namespace, skip this step and replace all -n with an existing namespace name.
export JWT_SECRET=$(echo -n "your_secret" | openssl dgst -sha256 -hmac "your_key" -binary | base64) # Please take a look at the official documentation https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/deploy-on-kubernetes/cluster/
helm upgrade --install -n sonarqube-dce sonarqube sonarqube/sonarqube-dce \
  --set ApplicationNodes.jwtSecret=$JWT_SECRET \
  --set OpenShift.enabled=true \
  --set postgresql.securityContext.enabled=false \
  --set postgresql.containerSecurityContext.enabled=false
```

If you want to make your application publicly visible with Routes, you can set `OpenShift.route.enabled` to true. Please check the [configuration details](#openshift-1) to customize the Route base on your needs.

## Autoscaling

The SonarQube applications nodes can be set to automatically scale up and down based on their average CPU utilization. This is particularly useful when scanning new projects or evaluating Pull Requests with SonarQube. In order to enable the autoscaling, you can rely on the `ApplicationNodes.hpa` parameters.

Please ensure the [Metrics Server](https://github.com/kubernetes-sigs/metrics-server) is installed in your cluster to provide resource usage metrics. You can deploy it using: 

```
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

```

### Upgrading the Helm chart

When upgrading your SonarQube instance, due to high CPU usage, it is recommended to disable the autoscaling before the upgrade process, re-enabling it afterwards.

You can achieve that by either setting `ApplicationNodes.hpa.enabled` to `false` or by setting `ApplicationNodes.hpa.maxReplicas` to be the same value as `ApplicationNodes.hpa.minReplicas`.

## Secure the communication within the cluster

In order to secure the communication between Application and Search nodes, you need to set both `nodeEncryption.enabled` and `searchNodes.searchAuthentication.enabled` to `true`.

In a secured cluster, Elasticsearch nodes use certificates to identify themselves when communicating with other nodes. You need to generate a Certificate Authority (CA) together with a certificate and private key for the nodes in your cluster. Furthemore, you need to specify the Search nodes' hostnames that will be added as DNS names in the Subject Alternative Name (SAN).

As an example, let's assume that your cluster has three search nodes with the release's name set to "sq", the chart's name set to "sonarqube-dce", and the namespace set to "sonar". You will need to add the following DNS names in the SAN.

```
sq-sonarqube-dce-search-0.sq-sonarqube-dce-search.sonar.svc.cluster.local
sq-sonarqube-dce-search-1.sq-sonarqube-dce-search.sonar.svc.cluster.local
sq-sonarqube-dce-search-2.sq-sonarqube-dce-search.sonar.svc.cluster.local
sq-sonarqube-dce-search
```

Please do not forget to add the service name in the list (in this case, `sq-sonarqube-dce-search`). Also note that you can retrieve the search nodes' FQDN running `hostname -f` within one of the pods.

You can generate the required certificate, create a secret, and add it to `searchNodes.searchAuthentication.keyStoreSecret` (specifying any password using the `keyStorePassword` or `keyStorePasswordSecret` values). To do so, you might want to use the `elasticsearch-certutil` to generate the [certificate authority](https://www.elastic.co/guide/en/elasticsearch/reference/current/security-basic-setup.html#generate-certificates) and the [certificate](https://www.elastic.co/guide/en/elasticsearch/reference/current/security-basic-setup-https.html#encrypt-http-communication) to be added (when creating the last certificate, please generate only one valid for all the nodes and add the required hostnames as specified above). As a result of this process, you should get a file called `http.p12`. Please rename it to `elastic-stack-ca.p12` and create the secret whose name should be assigned to the `searchNodes.searchAuthentication.keyStoreSecret` parameter.

Finally, do not forget to set the `searchNodes.searchAuthentication.userPassword`.

## Configuration

The following table lists the configurable parameters of the SonarQube chart and their default values.

### Search Nodes Configuration

| Parameter                                                 | Description                                                                                | Default                                                                |
| --------------------------------------------------------- | ------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------- |
| `searchNodes.image.repository`                            | search image repository                                                                    | `sonarqube`                                                            |
| `searchNodes.image.tag`                                   | search image tag                                                                           | `10.7.0-datacenter-search`                                             |
| `searchNodes.image.pullPolicy`                            | search image pull policy                                                                   | `IfNotPresent`                                                         |
| `searchNodes.image.pullSecret`                            | (DEPRECATED) search imagePullSecret to use for private repository                          | `nil`                                                                  |
| `searchNodes.image.pullSecrets`                           | search imagePullSecrets to use for private repository                                      | `nil`                                                                  |
| `searchNodes.annotations`                                 | Map of annotations to add to the search pods                                               | `{}`                                                                   |
| `searchNodes.env`                                         | Environment variables to attach to the search pods                                         | `nil`                                                                  |
| `searchNodes.podLabels`                                   | Map of labels to add to the search pods                                                    | `{}`                                                                   |
| `searchNodes.sonarProperties`                             | Custom `sonar.properties` file for Search Nodes                                            | `None`                                                                 |
| `searchNodes.sonarSecretProperties`                       | Additional `sonar.properties` file for Search Nodes to load from a secret                  | `None`                                                                 |
| `searchNodes.sonarSecretKey`                              | Name of existing secret used for settings encryption                                       | `None`                                                                 |
| `searchNodes.searchAuthentication.enabled`                | Securing the Search Cluster with basic authentication and TLS in between search nodes      | `false`                                                                |
| `searchNodes.searchAuthentication.keyStoreSecret`         | Existing PKCS#12 certificate (named `elastic-stack-ca.p12`) to be used Keystore/Truststore | `""`                                                                   |
| `searchNodes.searchAuthentication.keyStorePassword`       | Password to Keystore/Truststore used in search nodes (optional)                            | `""`                                                                   |
| `searchNodes.searchAuthentication.keyStorePasswordSecret` | Existing secret for Password to Keystore/Truststore used in search nodes (optional)        | `nil`                                                                  |
| `searchNodes.searchAuthentication.userPassword`           | A User Password that will be used to authenticate against the Search Cluster               | `""`                                                                   |
| `searchNodes.replicaCount`                                | Replica count of the Search Nodes                                                          | `3`                                                                    |
| `searchNodes.podDisruptionBudget`                         | PodDisruptionBudget for the Search Nodes                                                   | `minAvailable: 2`                                                      |
| `searchNodes.podDistributionBudget`                       | (DEPRECATED typo) PodDisruptionBudget for the Search Nodes                                 | `minAvailable: 2`                                                      |
| `searchNodes.securityContext`                             | SecurityContext for the pod search nodes                                                   | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `searchNodes.containerSecurityContext`                    | SecurityContext for search container in sonarqube pod                                      | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `searchNodes.readinessProbe.initialDelaySeconds`          | ReadinessProbe initial delay for Search Node checking                                      | `0`                                                                    |
| `searchNodes.readinessProbe.periodSeconds`                | ReadinessProbe period between checking Search Node                                         | `30`                                                                   |
| `searchNodes.readinessProbe.failureThreshold`             | ReadinessProbe threshold for marking as failed                                             | `6`                                                                    |
| `searchNodes.readinessProbe.timeoutSeconds`               | ReadinessProbe timeout delay                                                               | `1`                                                                    |
| `searchNodes.livenessProbe.initialDelaySeconds`           | LivenessProbe initial delay for Search Node checking                                       | `0`                                                                    |
| `searchNodes.livenessProbe.periodSeconds`                 | LivenessProbe period between checking Search Node                                          | `30`                                                                   |
| `searchNodes.livenessProbe.failureThreshold`              | LivenessProbe threshold for marking as dead                                                | `6`                                                                    |
| `searchNodes.livenessProbe.timeoutSeconds`                | LivenessProbe timeout delay                                                                | `1`                                                                    |
| `searchNodes.startupProbe.initialDelaySeconds`            | StartupProbe initial delay for Search Node checking                                        | `20`                                                                   |
| `searchNodes.startupProbe.periodSeconds`                  | StartupProbe period between checking Search Node                                           | `10`                                                                   |
| `searchNodes.startupProbe.failureThreshold`               | StartupProbe threshold for marking as failed                                               | `24`                                                                   |
| `searchNodes.startupProbe.timeoutSeconds`                 | StartupProbe timeout delay                                                                 | `1`                                                                    |
| `searchNodes.resources.requests.memory`                   | memory request for Search Nodes                                                            | `3072M`                                                                |
| `searchNodes.resources.requests.cpu`                      | cpu request for Search Nodes                                                               | `400m`                                                                 |
| `searchNodes.resources.requests.ephemeral-storage`        | storage request for Search Nodes                                                           | `1536M`                                                                |
| `searchNodes.resources.limits.memory`                     | memory limit for Search Nodes. should not be under 3G                                      | `3072M`                                                                |
| `searchNodes.resources.limits.cpu`                        | cpu limit for Search Nodes                                                                 | `800m`                                                                 |
| `searchNodes.resources.limits.ephemeral-storage`          | storage limit for Search Nodes                                                             | `512000M`                                                              |
| `searchNodes.persistence.enabled`                         | enabled or disables the creation of VPCs for the Search Nodes                              | `true`                                                                 |
| `searchNodes.persistence.annotations`                     | PVC annotations for the Search Nodes                                                       | `{}`                                                                   |
| `searchNodes.persistence.storageClass`                    | Storage class to be used                                                                   | `""`                                                                   |
| `searchNodes.persistence.accessMode`                      | Volumes access mode to be set                                                              | `ReadWriteOnce`                                                        |
| `searchNodes.persistence.size`                            | Size of the PVC                                                                            | `5G`                                                                   |
| `searchNodes.persistence.uid`                             | UID used for init-fs container                                                             | `1000`                                                                 |
| `searchNodes.persistence.guid`                            | GUID used for init-fs container                                                            | `0`                                                                    |
| `searchNodes.extraContainers`                             | Array of extra containers to run alongside                                                 | `[]`                                                                   |
| `searchNodes.nodeSelector`                                | Node labels for search nodes' pods assignment, global nodeSelector takes precedence        | `{}`                                                                   |
| `searchNodes.affinity`                                    | Node / Pod affinities for searchNodes, global affinity takes precedence                    | `{}`                                                                   |
| `searchNodes.tolerations`                                 | List of node taints to tolerate for searchNodes, global tolerations take precedence        | `[]`                                                                   |


### App Nodes Configuration

| Parameter                                                        | Description                                                                                                                                                                                                    | Default                                                                |
| ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `ApplicationNodes.image.repository`                              | app image repository                                                                                                                                                                                           | `sonarqube`                                                            |
| `ApplicationNodes.image.tag`                                     | app image tag                                                                                                                                                                                                  | `10.7.0-datacenter-app`                                                |
| `ApplicationNodes.image.pullPolicy`                              | app image pull policy                                                                                                                                                                                          | `IfNotPresent`                                                         |
| `ApplicationNodes.image.pullSecret`                              | (DEPRECATED) app imagePullSecret to use for private repository                                                                                                                                                 | `nil`                                                                  |
| `ApplicationNodes.image.pullSecrets`                             | app imagePullSecrets to use for private repository                                                                                                                                                             | `nil`                                                                  |
| `ApplicationNodes.annotations`                                   | Map of annotations to add to the app pods                                                                                                                                                                      | `{}`                                                                   |
| `ApplicationNodes.env`                                           | Environment variables to attach to the app pods                                                                                                                                                                | `nil`                                                                  |
| `ApplicationNodes.podLabels`                                     | Map of labels to add to the app pods                                                                                                                                                                           | `{}`                                                                   |
| `ApplicationNodes.sonarProperties`                               | Custom `sonar.properties` key-value pairs for App Nodes (e.g., "ApplicationNodes.sonarProperties.sonar.forceAuthentication=true")                                                                              | `None`                                                                 |
| `ApplicationNodes.sonarSecretProperties`                         | Additional `sonar.properties` key-value pairs for App Nodes to load from a secret                                                                                                                              | `None`                                                                 |
| `ApplicationNodes.sonarSecretKey`                                | Name of existing secret used for settings encryption                                                                                                                                                           | `None`                                                                 |
| `ApplicationNodes.replicaCount`                                  | Replica count of the app Nodes                                                                                                                                                                                 | `2`                                                                    |
| `ApplicationNodes.podDisruptionBudget`                           | PodDisruptionBudget for the App Nodes                                                                                                                                                                          | `minAvailable: 1`                                                      |
| `ApplicationNodes.podDistributionBudget`                         | (DEPRECATED typo) PodDisruptionBudget for the App Nodes                                                                                                                                                        | `minAvailable: 1`                                                      |
| `ApplicationNodes.securityContext`                               | SecurityContext for the pod app nodes                                                                                                                                                                          | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `ApplicationNodes.containerSecurityContext`                      | SecurityContext for app container in sonarqube pod                                                                                                                                                             | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `ApplicationNodes.readinessProbe`                                | ReadinessProbe for the App                                                                                                                                                                                     | see `values.yaml`                                                      |
| `ApplicationNodes.readinessProbe.sonarWebContext`                | (DEPRECATED) SonarQube web context for readinessProbe, please use sonarWebContext at the value top level instead                                                                                               | `/`                                                                    |
| `ApplicationNodes.livenessProbe`                                 | LivenessProbe for the App                                                                                                                                                                                      | see `values.yaml`                                                      |
| `ApplicationNodes.livenessProbe.sonarWebContext`                 | (DEPRECATED) SonarQube web context for livenessProbe, please use sonarWebContext at the value top level instead                                                                                                | `/`                                                                    |
| `ApplicationNodes.startupProbe.initialDelaySeconds`              | StartupProbe initial delay for app Node checking                                                                                                                                                               | `45`                                                                   |
| `ApplicationNodes.startupProbe.periodSeconds`                    | StartupProbe period between checking app Node                                                                                                                                                                  | `10`                                                                   |
| `ApplicationNodes.startupProbe.failureThreshold`                 | StartupProbe threshold for marking as failed                                                                                                                                                                   | `32`                                                                   |
| `ApplicationNodes.startupProbe.timeoutSeconds`                   | StartupProbe timeout delay                                                                                                                                                                                     | `1`                                                                    |
| `ApplicationNodes.startupProbe.sonarWebContext`                  | (DEPRECATED) SonarQube web context for startupProbe, please use sonarWebContext at the value top level instead                                                                                                 | `/`                                                                    |
| `ApplicationNodes.resources.requests.memory`                     | memory request for app Nodes                                                                                                                                                                                   | `4096M`                                                                |
| `ApplicationNodes.resources.requests.cpu`                        | cpu request for app Nodes                                                                                                                                                                                      | `400m`                                                                 |
| `ApplicationNodes.resources.requests.ephemeral-storage`          | storage request for app Nodes                                                                                                                                                                                  | `1536M`                                                                |
| `ApplicationNodes.resources.limits.memory`                       | memory limit for app Nodes. should not be under 4G                                                                                                                                                             | `4096M`                                                                |
| `ApplicationNodes.resources.limits.cpu`                          | cpu limit for app Nodes                                                                                                                                                                                        | `800m`                                                                 |
| `ApplicationNodes.resources.limits.ephemeral-storage`            | storage limit for app Nodes                                                                                                                                                                                    | `512000M`                                                              |
| `ApplicationNodes.prometheusExporter.enabled`                    | Use the Prometheus JMX exporter                                                                                                                                                                                | `false`                                                                |
| `ApplicationNodes.prometheusExporter.version`                    | jmx_prometheus_javaagent version to download from Maven Central                                                                                                                                                | `0.17.2`                                                               |
| `ApplicationNodes.prometheusExporter.noCheckCertificate`         | Flag to not check server's certificate when downloading jmx_prometheus_javaagent                                                                                                                               | `false`                                                                |
| `ApplicationNodes.prometheusExporter.webBeanPort`                | Port where the jmx_prometheus_javaagent exposes the metrics for the webBean                                                                                                                                    | `8000`                                                                 |
| `ApplicationNodes.prometheusExporter.ceBeanPort`                 | Port where the jmx_prometheus_javaagent exposes the metrics for the ceBean                                                                                                                                     | `8001`                                                                 |
| `ApplicationNodes.prometheusExporter.downloadURL`                | Alternative full download URL for the jmx_prometheus_javaagent.jar (overrides `prometheusExporter.version`)                                                                                                    | `""`                                                                   |
| `ApplicationNodes.prometheusExporter.config`                     | Prometheus JMX exporter config yaml for the web process, and the CE process if `prometheusExporter.ceConfig` is not set                                                                                        | see `values.yaml`                                                      |
| `ApplicationNodes.prometheusExporter.ceConfig`                   | Prometheus JMX exporter config yaml for the CE process (by default, `prometheusExporter.config` is used                                                                                                        | `None`                                                                 |
| `ApplicationNodes.prometheusExporter.httpProxy`                  | HTTP proxy for downloading JMX agent                                                                                                                                                                           | `""`                                                                   |
| `ApplicationNodes.prometheusExporter.httpsProxy`                 | HTTPS proxy for downloading JMX agent                                                                                                                                                                          | `""`                                                                   |
| `ApplicationNodes.prometheusExporter.noProxy`                    | No proxy for downloading JMX agent                                                                                                                                                                             | `""`                                                                   |
| `ApplicationNodes.prometheusExporter.securityContext`            | Security context for downloading the jmx agent                                                                                                                                                                 | see `values.yaml`                                                      |
| `ApplicationNodes.prometheusMonitoring.podMonitor.enabled`       | Enable Prometheus PodMonitor                                                                                                                                                                                   | `false`                                                                |
| `ApplicationNodes.prometheusMonitoring.podMonitor.namespace`     | (DEPRECATED) This value should not be set, as the PodMonitor's namespace has to match the Release Namespace                                                                                                    | `{{ .Release.Namespace }}`                                             |
| `ApplicationNodes.prometheusMonitoring.podMonitor.interval`      | Specify the interval how often metrics should be scraped                                                                                                                                                       | `30s`                                                                  |
| `ApplicationNodes.prometheusMonitoring.podMonitor.scrapeTimeout` | Specify the timeout after a scrape is ended                                                                                                                                                                    | `None`                                                                 |
| `ApplicationNodes.prometheusMonitoring.podMonitor.jobLabel`      | Name of the label on target services that prometheus uses as job name                                                                                                                                          | `None`                                                                 |
| `ApplicationNodes.prometheusMonitoring.podMonitor.labels`        | Additional labels to add to the PodMonitor                                                                                                                                                                     | `{}`                                                                   |
| `ApplicationNodes.plugins.install`                               | Link(s) to the plugin JARs to download and install                                                                                                                                                             | `[]`                                                                   |
| `ApplicationNodes.plugins.resources`                             | Plugin Pod resource requests & limits                                                                                                                                                                          | `{}`                                                                   |
| `ApplicationNodes.plugins.httpProxy`                             | For use behind a corporate proxy when downloading plugins                                                                                                                                                      | `""`                                                                   |
| `ApplicationNodes.plugins.httpsProxy`                            | For use behind a corporate proxy when downloading plugins                                                                                                                                                      | `""`                                                                   |
| `ApplicationNodes.plugins.noProxy`                               | For use behind a corporate proxy when downloading plugins                                                                                                                                                      | `""`                                                                   |
| `ApplicationNodes.plugins.image`                                 | Image for plugins container                                                                                                                                                                                    | `""`                                                                   |
| `ApplicationNodes.plugins.resources`                             | Resources for plugins container                                                                                                                                                                                | `""`                                                                   |
| `ApplicationNodes.plugins.netrcCreds`                            | Name of the secret containing .netrc file to use creds when downloading plugins                                                                                                                                | `""`                                                                   |
| `ApplicationNodes.plugins.noCheckCertificate`                    | Flag to not check server's certificate when downloading plugins                                                                                                                                                | `false`                                                                |
| `ApplicationNodes.plugins.securityContext`                       | Security context for the container to download plugins                                                                                                                                                         | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `ApplicationNodes.jvmOpts`                                       | (DEPRECATED) Values to add to `SONAR_WEB_JAVAOPTS`. Please set directly `SONAR_WEB_JAVAOPTS` or `sonar.web.javaOpts`                                                                                           | `""`                                                                   |
| `ApplicationNodes.jvmCeOpts`                                     | (DEPRECATED) Values to add to `SONAR_CE_JAVAOPTS`. Please set directly `SONAR_CE_JAVAOPTS` or `sonar.ce.javaOpts`                                                                                              | `""`                                                                   |
| `ApplicationNodes.jwtSecret`                                     | A HS256 key encoded with base64 (_This value must be set before installing the chart, see [the documentation](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/deploy-on-kubernetes/cluster/)_) | `""`                                                                   |
| `ApplicationNodes.existingJwtSecret`                             | secret that contains the `jwtSecret`                                                                                                                                                                           | `nil`                                                                  |
| `ApplicationNodes.extraContainers`                               | Array of extra containers to run alongside                                                                                                                                                                     | `[]`                                                                   |
| `ApplicationNodes.extraVolumes`                                  | Array of extra volumes to add to the SonarQube deployment                                                                                                                                                      | `[]`                                                                   |
| `ApplicationNodes.extraVolumeMounts`                             | Array of extra volume mounts to add to the SonarQube deployment                                                                                                                                                | `[]`                                                                   |
| `ApplicationNodes.hpa.enabled`                                   | Enable the HorizontalPodAutoscaler (HPA) for the app deployment                                                                                                                                                | `false`                                                                |
| `ApplicationNodes.hpa.minReplicas`                               | Minimum number of replicas for the HPA                                                                                                                                                                         | `2`                                                                    |
| `ApplicationNodes.hpa.maxReplicas`                               | Maximum number of replicas for the HPA                                                                                                                                                                         | `10`                                                                   |
| `ApplicationNodes.hpa.metrics`                                   | The metrics to use for scaling                                                                                                                                                                                 | see `values.yaml`                                                      |
| `ApplicationNodes.hpa.behavior`                                  | The scaling behavior                                                                                                                                                                                           | see `values.yaml`                                                      |
| `ApplicationNodes.nodeSelector`                                  | Node labels for application nodes' pods assignment, global nodeSelector takes precedence                                                                                                                       | `{}`                                                                   |
| `ApplicationNodes.affinity`                                      | Node / Pod affinities for ApplicationNodes, global affinity takes precedence                                                                                                                                   | `{}`                                                                   |
| `ApplicationNodes.tolerations`                                   | List of node taints to tolerate for ApplicationNodes, global tolerations take precedence                                                                                                                       | `[]`                                                                   |


### Generic Configuration

| Parameter                | Description                                                                                                           | Default |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------- | ------- |
| `affinity`               | Node / Pod affinities                                                                                                 | `{}`    |
| `tolerations`            | List of node taints to tolerate                                                                                       | `[]`    |
| `priorityClassName`      | Schedule pods on priority (e.g. `high-priority`)                                                                      | `None`  |
| `nodeSelector`           | Node labels for pod assignment                                                                                        | `{}`    |
| `hostAliases`            | Aliases for IPs in /etc/hosts                                                                                         | `[]`    |
| `podLabels`              | Map of labels to add to the pods                                                                                      | `{}`    |
| `env`                    | Environment variables to attach to the pods                                                                           | `{}`    |
| `annotations`            | Map of annotations to add to the pods                                                                                 | `{}`    |
| `sonarWebContext`        | SonarQube web context, also serve as default value for `ingress.path`, `account.sonarWebContext` and probes path.     | ``      |
| `httpProxySecret`        | Should contain `http_proxy`, `https_proxy` and `no_proxy` keys, will superseed every other proxy variables            | ``      |
| `httpProxy`              | HTTP proxy for downloading JMX agent and install plugins, will superseed initContainer specific http proxy variables  | ``      |
| `httpsProxy`             | HTTPS proxy for downloading JMX agent and install plugins, will superseed initContainer specific https proxy variable | ``      |
| `noProxy`                | No proxy for downloading JMX agent and install plugins, will superseed initContainer specific no proxy variables      | ``      |
| `nodeEncryption.enabled` | Secure the communication between Application and Search nodes using TLS                                               | `false` |

### NetworkPolicies

| Parameter                                 | Description                                                               | Default |
| ----------------------------------------- | ------------------------------------------------------------------------- | ------- |
| `networkPolicy.enabled`                   | Create NetworkPolicies                                                    | `false` |
| `networkPolicy.prometheusNamespace`       | Allow incoming traffic to monitoring ports from this namespace            | `nil`   |
| `networkPolicy.additionalNetworkPolicys`  | (DEPRECATED) Please use `networkPolicy.additionalNetworkPolicies` instead | `nil`   |
| `networkPolicy.additionalNetworkPolicies` | User defined NetworkPolicies (usefull for external database)              | `nil`   |

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

### HttpRoute

| Parameter             | Description                                                                                                   | Default |
| --------------------- | ------------------------------------------------------------------------------------------------------------- | ------- |
| `httproute.enabled`   | Flag to enable GatewayAPI HttpRoute                                                                           | `False` |
| `httproute.gateway`   | Name of the gateway located in the same namespace                                                             | `None`  |
| `httproute.hostnames` | List of hostnames to match the HttpRoute against                                                              | `None`  |
| `httproute.labels`    | (Optional) List of extra labels to add to the HttpRoute                                                       | `None`  |
| `httproute.rules`     | (Optional) Extra Rules block of the HttpRoute. A default one is created with SonarWebContext and service port | `None`  |

### Elasticsearch

| Parameter                       | Description                                     | Default |
| ------------------------------- | ----------------------------------------------- | ------- |
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
| `ingress-nginx.enabled`        | Install Nginx Ingress Helm                                   | `false`                                                                                                      |
| `nginx.enabled`                | (DEPRECATED) please use `ingress-nginx.enabled`              | `false`                                                                                                      |
| `ingress.enabled`              | Flag to enable Ingress                                       | `false`                                                                                                      |
| `ingress.labels`               | Ingress additional labels                                    | `{}`                                                                                                         |
| `ingress.hosts[0].name`        | Hostname to your SonarQube installation                      | `sonarqube.your-org.com`                                                                                     |
| `ingress.hosts[0].path`        | Path within the URL structure                                | `/`                                                                                                          |
| `ingress.hosts[0].serviceName` | Optional field to override the default serviceName of a path | `None`                                                                                                       |
| `ingress.hosts[0].servicePort` | Optional field to override the default servicePort of a path | `None`                                                                                                       |
| `ingress.tls`                  | Ingress secrets for TLS certificates                         | `[]`                                                                                                         |
| `ingress.ingressClassName`     | Optional field to configure ingress class name               | `None`                                                                                                       |
| `ingress.annotations`          | Field to add extra annotations to the ingress                | {`nginx.ingress.kubernetes.io/proxy-body-size: "64m"`} if `ingress-nginx.enabled=true or nginx.enabled=true` |

### InitContainers

| Parameter                           | Description                                                                                                                           | Default                                                                |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `initContainers.image`              | Change init container image                                                                                                           | `ApplicationNodes.image`                                               |
| `initContainers.securityContext`    | SecurityContext for init containers                                                                                                   | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `initContainers.resources`          | Resources for init containers                                                                                                         | `{}`                                                                   |
| `extraInitContainers`               | Extra init containers to e.g. download required artifacts                                                                             | `{}`                                                                   |
| `caCerts.enabled`                   | Flag for enabling additional CA certificates                                                                                          | `false`                                                                |
| `caCerts.image`                     | Change init CA certificates container image                                                                                           | `ApplicationNodes.image`                                               |
| `caCerts.secret`                    | Name of the secret containing additional CA certificates. If defined, only secrets are going to be used.                              | `None`                                                                 |
| `caCerts.configMap.name`            | Name of the ConfigMap containing additional CA certificate. Ensure that `caCerts.secret` is not set if you want to use a `ConfigMap`. | `None`                                                                 |
| `caCerts.configMap.key`             | Name of the key containing the additional CA certificate                                                                              | `None`                                                                 |
| `caCerts.configMap.path`            | Filename that should be used for the given CA certificate                                                                             | `None`                                                                 |
| `initSysctl.enabled`                | Modify k8s worker to conform to system requirements                                                                                   | `true`                                                                 |
| `initSysctl.vmMaxMapCount`          | Set init sysctl container vm.max_map_count                                                                                            | `524288`                                                               |
| `initSysctl.fsFileMax`              | Set init sysctl container fs.file-max                                                                                                 | `131072`                                                               |
| `initSysctl.nofile`                 | Set init sysctl container open file descriptors limit                                                                                 | `131072`                                                               |
| `initSysctl.nproc`                  | Set init sysctl container open threads limit                                                                                          | `8192`                                                                 |
| `initSysctl.image`                  | Change init sysctl container image                                                                                                    | `ApplicationNodes.image`                                               |
| `initSysctl.securityContext`        | InitSysctl container security context                                                                                                 | `{privileged: true}`                                                   |
| `initSysctl.resources`              | InitSysctl container resource requests & limits                                                                                       | `{}`                                                                   |
| `initFs.enabled`                    | Enable file permission change with init container                                                                                     | `true`                                                                 |
| `initFs.image`                      | InitFS container image                                                                                                                | `ApplicationNodes.image`                                               |
| `initFs.securityContext.privileged` | InitFS container needs to run privileged                                                                                              | `true`                                                                 |

### SonarQube Specific

| Parameter                      | Description                                                                             | Default          |
| ------------------------------ | --------------------------------------------------------------------------------------- | ---------------- |
| `sonarqubeFolder`              | Directory name of SonarQube                                                             | `/opt/sonarqube` |
| `monitoringPasscode`           | Value for sonar.web.systemPasscode needed for LivenessProbes (encoded to Base64 format) | `define_it`      |
| `monitoringPasscodeSecretName` | Name of the secret where to load `monitoringPasscode`                                   | `None`           |
| `monitoringPasscodeSecretKey`  | Key of an existing secret containing `monitoringPasscode`                               | `None`           |
| `extraContainers`              | Array of extra containers to run alongside the `sonarqube` container (aka. Sidecars)    | `[]`             |

### JDBC Overwrite

| Parameter                                   | Description                                                                                                                                                  | Default                                    |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------ |
| `jdbcOverwrite.enable`                      | (DEPRECATED) Enable JDBC overwrites for external Databases (disables `postgresql.enabled`) ,Please use jdbcOverwrite.enabled instead                         | `false`                                    |
| `jdbcOverwrite.enabled`                     | Enable JDBC overwrites for external Databases (disable `postgresql.enabled`)                                                                                 | `false`                                    |
| `jdbcOverwrite.jdbcUrl`                     | The JDBC url to connect the external DB                                                                                                                      | `jdbc:postgresql://myPostgress/myDatabase` |
| `jdbcOverwrite.jdbcUsername`                | The DB user that should be used for the JDBC connection                                                                                                      | `sonarUser`                                |
| `jdbcOverwrite.jdbcPassword`                | The DB password that should be used for the JDBC connection (Use this if you don't mind the DB password getting stored in plain text within the values file) | `sonarPass`                                |
| `jdbcOverwrite.jdbcSecretName`              | Alternatively, use a pre-existing k8s secret containing the DB password                                                                                      | `None`                                     |
| `jdbcOverwrite.jdbcSecretPasswordKey`       | If the pre-existing k8s secret is used this allows the user to overwrite the 'key' of the password property in the secret                                    | `None`                                     |
| `jdbcOverwrite.oracleJdbcDriver.url`        | The URL of the Oracle JDBC driver to be downloaded                                                                                                           | `None`                                     |
| `jdbcOverwrite.oracleJdbcDriver.netrcCreds` | Name of the secret containing .netrc file to use creds when downloading the Oracle JDBC driver                                                               | `None`                                     |

### Bundled PostgreSQL Chart (DEPRECATED)

The bundled PostgreSQL Chart is deprecated. Please see <https://artifacthub.io/packages/helm/sonarqube/sonarqube#production-use-case> for more information.

| Parameter                                                | Description                                                            | Default         |
| -------------------------------------------------------- | ---------------------------------------------------------------------- | --------------- |
| `postgresql.enabled`                                     | Set to `false` to use external database server                         | `true`          |
| `postgresql.existingSecret`                              | existingSecret Name of existing secret to use for PostgreSQL passwords | `nil`           |
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
| `postgresql.securityContext.fsGroup`                     | PostgreSQL securityContext fsGroup                                     | `1001`          |
| `postgresql.securityContext.runAsUser`                   | PostgreSQL securityContext runAsUser                                   | `1001`          |
| `postgresql.volumePermissions.enabled`                   | PostgreSQL vol permissions en/disabled                                 | `false`         |
| `postgresql.volumePermissions.securityContext.runAsUser` | PostgreSQL vol permissions secContext runAsUser                        | `0`             |
| `postgresql.shmVolume.chmod.enabled`                     | PostgreSQL shared memory vol en/disabled                               | `false`         |
| `postgresql.serivceAccount.enabled`                      | PostgreSQL service Account creation en/disabled                        | `false`         |
| `postgresql.serivceAccount.name`                         | PostgreSQL service Account name                                        | `""`            |

### Tests

| Parameter                       | Description                                                   | Default                                                            |
| ------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------ |
| `tests.enabled`                 | Flag that allows tests to be excluded from the generated yaml | `true`                                                             |
| `tests.image`                   | Set the test container image                                  | `"ApplicationNodes.image.repository":"ApplicationNodes.image.tag"` |
| `tests.resources.limits.cpu`    | CPU limit for test container                                  | `500m`                                                             |
| `tests.resources.limits.memory` | Memory limit for test container                               | `200M`                                                             |

### ServiceAccount

| Parameter                       | Description                                                                          | Default               |
| ------------------------------- | ------------------------------------------------------------------------------------ | --------------------- |
| `serviceAccount.create`         | If set to true, create a serviceAccount                                              | `false`               |
| `serviceAccount.name`           | Name of the serviceAccount to create/use                                             | `sonarqube-sonarqube` |
| `serviceAccount.automountToken` | Manage `automountServiceAccountToken` field for mounting service account credentials | `false`               |
| `serviceAccount.annotations`    | Additional serviceAccount annotations                                                | `{}`                  |

### ExtraConfig

| Parameter                | Description                                                                                                               | Default |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------- | ------- |
| `extraConfig.secrets`    | A list of `Secret`s (which must contain key/value pairs) which may be loaded into the Scanner as environment variables    | `[]`    |
| `extraConfig.configmaps` | A list of `ConfigMap`s (which must contain key/value pairs) which may be loaded into the Scanner as environment variables | `[]`    |

### Advanced Options

| Parameter                           | Description                                                                                                  | Default                  |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------ | ------------------------ |
| `logging.jsonOutput`                | Enable/Disable logging in JSON format                                                                        | `false`                  |
| `account.adminPassword`             | Custom new admin password                                                                                    | `admin`                  |
| `account.currentAdminPassword`      | Current admin password                                                                                       | `admin`                  |
| `account.adminPasswordSecretName`   | Secret containing `password` (custom password) and `currentPassword` (current password) keys for admin       | `None`                   |
| `account.resources.requests.memory` | Memory request for Admin hook                                                                                | `128Mi`                  |
| `account.resources.requests.cpu`    | CPU request for Admin hook                                                                                   | `100m`                   |
| `account.resources.limits.memory`   | Memory limit for Admin hook                                                                                  | `128Mi`                  |
| `account.resources.limits.cpu`      | CPU limit for Admin hook                                                                                     | `100m`                   |
| `curlContainerImage`                | Curl container image                                                                                         | `ApplicationNodes.image` |
| `account.sonarWebContext`           | (DEPRECATED) SonarQube web context for Admin hook. please use sonarWebContext at the value top level instead | `nil`                    |
| `account.securityContext`           | Security context for downloading the prometheus exporter                                                     | see `values.yaml`        |
| `adminJobAnnotations`               | Custom annotations for admin hook Job                                                                        | `{}`                     |
| `terminationGracePeriodSeconds`     | Configuration of `terminationGracePeriodSeconds`                                                             | `60`                     |
| `gcp_marketplace`                   | boolean to indicate the chart run as a google cloud marketplace app, trigger specific resources creation     | ``                       |

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
