# Nexus IQ

***This chart is deprecated. Nexus IQ chart is now managed by sonatype: https://artifacthub.io/packages/helm/sonatype/nexus-iq-server***

## What is Nexus IQ

Shares component intelligence with your teams early, often and throughout the software supply chain so they make better decisions and build better software.

Offers a fully-customizeable policy engine, so you can define which components are acceptable, and which are not.

Integrates with popular development tools including, but not limited to: Maven, Eclipse, IntelliJ, Visual Studio, GitHub, Bamboo, Jenkins, Xebia Labs, and SonarQube.

Provides a full suite of supported REST APIs that provide access to core features for custom implementations.

## Introduction

This chart bootstraps a Nexus IQ deployment on a cluster using Helm.

## Installing the Chart

### Installing with Helm 3.x

```bash
helm repo add oteemocharts https://oteemo.github.io/charts
helm install nexusiq oteemocharts/nexusiq
```

### Templating with Helm 3.x

To template this with helm 3.x:

 1. Complete the values file with your values.
 2. Execute the ```helm template``` command to generate your manifest files
 3. Execute the ```kubectl apply``` command to create the deployment within your kubernetes cluster.

## Uninstalling the Chart

### Uninstalling with Helm 3.x

```bash
$ helm list
NAME       REVISION  UPDATED                    STATUS   CHART      NAMESPACE
nexusiq      1       Fri Sep  1 13:19:50 2017   DEPLOYED nexusiq    default
$ helm delete nexusiq
```

### Uninstalling without Helm 3.x

In a tiller-less helm 2.x environment you must individually delete the objects created by the helm chart: deployment, persistent volumes, and persistent volume claims.

## Configuration

The following table lists the configurable parameters of the NexusIQ chart and their default values.

| Parameter                                   | Description                         | Default                                 |
| ------------------------------------------  | ----------------------------------  | ----------------------------------------|
| `nexusIQ.repository`                       | NexusIQ image repo | `sonatype/nexus-iq-server` |
| `nexusIQ.tag`                              | NexusIQ image version  | `1.63.0`                                     |
| `nexusIQ.pullPolicy`                        | NexusIQ image pull policy    |  `IfNotPresent` |
| `nexusIQ.metricsPort`                        | NexusIQ port to expose prometheus metrics over    |  `8071` |
| `nexusIQ.applicationPort`                        | NexusIQ application port    |  `8070` |
| `nexusIQ.portName`                        | blank    |  `nexus-iq-server` |
| `nexusIQ.livenessProbe.initialDelaySeconds`                        | LivenessProbe initial delay    |  `30` |
| `nexusIQ.livenessProbe.periodSeconds`                        | LivenessProbe period seconds    |  `30` |
| `nexusIQ.livenessProbe.failureThreshold`                        | LivenessProbe failure threshold    |  `6` |
| `nexusIQ.livenessProbe.path`                        | LivenessProbe path    |  `/` |
| `nexusIQ.readinessProbe.initialDelaySeconds`                        | ReadinessProbe initial delay    |  `30` |
| `nexusIQ.readinessProbe.periodSeconds`                        | ReadinessProbe period seconds    |  `30` |
| `nexusIQ.readinessProbe.failureThreshold`                        | ReadinessProbe failure threshold    |  `6` |
| `nexusIQ.readinessProbe.path`                        | ReadinessProbe path    |  `/` |
| `service.enabled`                       | Service Enabled Flag | `false` |
| `service.name`                       | Name for Service | `nexus-iq-server` |
| `service.type`                       | Service Type | `ClusterIP` |
| `service.port`                       | Service Port | `80` |
| `ingress.enabled`                       | Ingress Enabled Flag | `false` |
| `ingress.annotations`                       | Ingress annotations | blank |
| `ingress.hostName`                       | Ingress host name | blank |
| `ingress.hosts`                       | Ingress hosts | blank |
| `ingress.tls`                       | Ingress TLS configuration | blank |
| `persistence.enabled`                       | Enable persistent storage | `false` |
| `persistence.accessMode`                       | Set Storage Access Mode| `ReadWriteOnce` |
| `persistence.storageSize`                       | Set Storage Size | `25Gi` |
| `persistence.storageClass`                       | Set Storage Type | `gp2` |
| `persistence.labels`                       | Set Storage Labels | blank |
| `persistence.annotations`                       | Set storage annotations | blank |

## After Installing the Chart

After installing the chart a couple of actions still need to be done in order to use NexusIQ. Please follow the instructions below.

### NexusIQ Configuration

The following steps need to be executed in order to use NexusIQ:

 1. Install the license. Without a valid license you will not be able to navigate past the license page and use NexusIQ in any way.
 2. Configure basic permissions. By default NexusIQ creates a default `admin` user with a password of `admin123` that is not configurable at boostrap. You MUST change this immediately upon logging in.
 3. (Optional) Configure LDAP.

### Nexus IQ Server System Requirements

The following table lists the system requirements of the Nexus IQ Server

| Resource                                | Description                         |
| ------------------------------------------  | ---------------------------------- |
| `CPU & RAM`                       | Recommend a processor with at least 8 CPU cores and 8GB of RAM for initial setup. A minimum of 6GB of process space should be available to the IQ Server. Additional RAM can improve the performance due to decreased disk caching. |
| `Disk`                            | Storage requirements range with the number of applications projected to use the IQ Server. 500 GB to 1 TB of free disk space should provide more than adequate resources. |
| `Account` | It is recommended that an unprivileged service account be created if running the IQ Server as a daemon. |
| `Operating System` | Generally, any machine that can run a supported Sun/Oracle Java version should work. Refer to the Oracle documentation for specifics: Oracle JDK 8 and JRE 8 Certified System Configurations. The most widely used operating system for the IQ Server is Linux and therefore customers should consider it the best tested platform. |
| `Ports` | The IQ Server requires the following network access. Inbound: 8070 TCP: Used by all IQ Server clients for HTTP access. This port is configurable. 8071 TCP: Used by the local host or other IT monitoring tools for monitoring and operating functions. This port is optional and configurable. If not specified, port 8081 will be used. Outbound: 443 TCP to <https://clm.sonatype.com> : Used by the IQ Server to securely access Sonatype Data Services. This hostname and port are not configurable. Sonatype Data Services must be reachable by IQ Server on the following URL: <https://clm.sonatype.com/> . |
| `Java` | OpenJDK 8 (since December 2018, IQ Server release 55). Prior to IQ Server release 63, the IQ Server used to check if the used JVM is supported. This check does not work for certain OpenJDK versions/flavors. You can disable this check by adding -Dclm.disableJreCheck=true to the command used to start the IQ Server. |

### Important Links

1. Nexus IQ Server Web Page - <https://www.sonatype.com/nexus-iq-server>
2. Nexus IQ Server Documentation & Help Page - <https://help.sonatype.com/iqserver>
3. Nexus IQ Server Getting Started Guide - <https://help.sonatype.com/iqserver/getting-started>
4. Nexus IQ Docker Repo & Docker Documentation - <https://hub.docker.com/r/sonatype/nexus-iq-server>
