# SonarQube

Code better in more than 30 languages. Improve Code Quality and Code Security throughout your workflow. [SonarQube](https://www.sonarsource.com/products/sonarqube/) can detect Bugs, Vulnerabilities, Security Hotspots, and Code Smells plus gives you the guidance to fix them.

## Introduction

This helm chart bootstraps a SonarQube Data Center Edition cluster. It requires an external database (see [Installing the chart](#installing-the-chart)).

The latest version of the chart installs the latest SonarQube version.

To install SonarQube Server Long-Term Active (LTA), please read the section [below](#upgrading-to-sonarqube-server-lta). Deciding between LTA and Latest? [This may help](https://www.sonarsource.com/products/sonarqube/downloads/lts/).

Please note that this chart does NOT support SonarQube Community, Developer, and Enterprise Editions.

## Default Versions

| Component | Image | Default tag |
| --------- | ----- | ----------- |
| SonarQube Server application nodes | `sonarqube` | `2026.5.0-datacenter-app` |
| SonarQube Server search nodes | `sonarqube` | `2026.5.0-datacenter-search` |
| MCP Server | `sonarsource/sonarqube-mcp` | `2026.5.0` |
| Agent Orchestrator | `sonarsource/sonarqube-agent-orchestrator` | `2026.5.0` |
| Hunter Agent | `sonarsource/sonarqube-hunter-agent` | `2026.5.0` |
| Remediation Agent | `sonarsource/sonarqube-remediation-agent` | `2026.5.0` |
| Vortex | `sonarsource/sonar-vortex` | `2026.5.0` |

## Kubernetes and Openshift Compatibility

Supported Kubernetes Versions: From `1.32` to `1.35`

Supported Openshift Versions: From `4.17` to `4.20`

**Note:** The Kubernetes version range above applies to non-OpenShift Kubernetes clusters. For OpenShift, the supported range is defined by the OpenShift versions listed here and is validated as a platform, including its embedded Kubernetes version.

## Installing the chart

> **_NOTE:_**  Please refer to [the official page](https://docs.sonarsource.com/sonarqube-server/server-installation/data-center-edition/introduction) for further information on how to install and tune the helm chart specifications.

Prior to installing the chart, please ensure that the `monitoringPasscode` and `applicationNodes.jwtSecret` are properly set. The `applicationNodes.jwtSecret` value needs to be set with a HS256 key encoded with base64. In the following, an example on how to generate this key on a Unix system:

```bash
echo -n "your_secret" | openssl dgst -sha256 -hmac "your_key" -binary | base64
```

Please also note that the chart requires an external database. If you want to perform a quick testing, you might want to follow the steps outlined [here](#setting-up-an-external-database-for-quick-testing). You will be required to set the following values accordingly: `jdbcOverwrite.jdbcUrl`, `jdbcOverwrite.jdbcUsername`, `jdbcOverwrite.jdbcSecretName`, and `jdbcOverwrite.jdbcSecretPasswordKey`.

To install the chart:

```bash
helm repo add sonarqube https://SonarSource.github.io/helm-chart-sonarqube
helm repo update
kubectl create namespace sonarqube-dce
export JWT_SECRET=$(echo -n "your_secret" | openssl dgst -sha256 -hmac "your_key" -binary | base64)
export MONITORING_PASSCODE="yourPasscode"
export JDBC_URL="jdbc:postgresql://<your-db-host>:5432/<your-database>" # must be replaced: the chart rejects the placeholder URL
export JDBC_USERNAME="sonar"
export JDBC_PASSWORD_SECRET_NAME="jdbc-secret"
export JDBC_PASSWORD_SECRET_KEY="jdbc-password"
helm upgrade --install -n sonarqube-dce sonarqube sonarqube/sonarqube-dce --set applicationNodes.jwtSecret=$JWT_SECRET,monitoringPasscode=$MONITORING_PASSCODE,jdbcOverwrite.jdbcUrl=$JDBC_URL,jdbcOverwrite.jdbcUsername=$JDBC_USERNAME,jdbcOverwrite.jdbcSecretName=$JDBC_PASSWORD_SECRET_NAME,jdbcOverwrite.jdbcSecretPasswordKey=$JDBC_PASSWORD_SECRET_KEY
```

The above command deploys SonarQube on the Kubernetes cluster in the default configuration in the `sonarqube-dce` namespace.
If you are interested in deploying SonarQube on Openshift, please check the [dedicated section](#openshift).

The [configuration](#configuration) section lists the parameters that can be configured during installation.

The default login is admin/admin.

## Upgrading to SonarQube Server LTA

When upgrading your SonarQube Server to a new Long-Term Active (LTA) release, you should carefully read the official upgrade documentation to determine the correct update path based on your current server version.

<!-- TODO: the 2026.5 LTA-to-LTA notes are not published yet (the URL redirects to "2026.5 LTA is on its way"); check the link once they are live -->
* For SonarQube Server 2026.5 LTA, refer to the [LTA-to-LTA Upgrade Notes (2026.5)](https://docs.sonarsource.com/sonarqube-server/2026.5/server-update-and-maintenance/lta-to-lta-release-notes).
* For SonarQube Server 2026.1 LTA, refer to the [LTA-to-LTA Upgrade Notes (2026.1)](https://docs.sonarsource.com/sonarqube-server/2026.1/server-update-and-maintenance/lta-to-lta-release-notes).
* For SonarQube Server 2025.4 LTA, refer to the [LTA-to-LTA Upgrade Notes (2025.4)](https://docs.sonarsource.com/sonarqube-server/2025.4/server-update-and-maintenance/lta-to-lta-release-notes).
* For SonarQube Server 2025.1 LTA, refer to the [LTA-to-LTA Upgrade Notes (2025.1)](https://docs.sonarsource.com/sonarqube-server/2025.1/server-update-and-maintenance/release-notes-and-notices/lta-to-lta-release-upgrade-notes).

When upgrading to the 2026.5 LTA chart (`2026.5.1000`) from the 2026.1 LTA, the search nodes move from Elasticsearch 8 to 9 and need a specific procedure (see [below](#elasticsearch-8-to-9-upgrading-from-20263-or-earlier-to-20264-or-later)). Read also [Upgrade to 2026.5.1000](#upgrade-to-202651000).

When upgrading from a chart prior to `2026.1.0` (the 2026.1 LTA), you will experience a few changes.

* The deprecated PostgreSQL dependency has been removed. You must connect your SonarQube Server instance to an external database (`jdbcOverwrite.enabled` is set to true by default). You must set the following parameters: `jdbcOverwrite.jdbcUrl`, `jdbcOverwrite.jdbcUsername`, `jdbcOverwrite.jdbcSecretName`, and `jdbcOverwrite.jdbcSecretPasswordKey`.

### Upgrade process

1. Read through the [SonarQube Upgrade Guide](https://docs.sonarsource.com/sonarqube-server/server-update-and-maintenance/update/roadmap) to familiarize yourself with the general upgrade process (most importantly, back up your database)
2. Read the chart-specific notes below for every chart version you cross
3. Upgrade to the chart version that ships the target SonarQube version (`helm repo update`, then `helm upgrade` with `--version`), rather than only changing `applicationNodes.image.tag` and `searchNodes.image.tag` on your current chart. When crossing an Elasticsearch major, follow the [Elasticsearch 8 to 9](#elasticsearch-8-to-9-upgrading-from-20263-or-earlier-to-20264-or-later) procedure instead
4. Browse to <http://yourSonarQubeServerURL/setup> and follow the setup instructions
5. Reanalyze your projects to get fresh data

### Elasticsearch 8 to 9 (upgrading from 2026.3 or earlier to 2026.4 or later)

SonarQube Server 2026.4 and later run Elasticsearch 9; 2025.x to 2026.3 (including the 2026.1 LTA) run Elasticsearch 8. A rolling update of the search StatefulSet cannot cross an Elasticsearch major version. Plan a maintenance window and back up the database first.

1. Scale search to 0 with your **current** chart, without changing the image tag, and wait until the search pods are gone:

```bash
helm upgrade -n sonarqube-dce <yourReleaseName> <yourCurrentChart> --reuse-values --set searchNodes.replicaCount=0
```

2. Upgrade to the new chart and restore your previous search replica count (default `3`). Use `--reset-then-reuse-values` (Helm `>= 3.14`) or pass your values file with `-f`: `--reuse-values` would keep the previous chart's image tags and skip the new defaults.

```bash
helm upgrade -n sonarqube-dce <yourReleaseName> <yourNewChart> --reset-then-reuse-values --set searchNodes.replicaCount=3
```

3. After search is Ready, browse to `/setup` and follow the instructions. Indexes are rebuilt into `es9`; rolling back in place is not supported.

On `helm upgrade`, the chart fails while search pods are still running on the previous Elasticsearch major. The check relies on `lookup`, so it does not run under `helm template` or GitOps tools that render manifests (Argo CD, Flux), nor for custom image tags: in those cases follow the procedure above yourself. Set `searchNodes.skipEsMajorUpgradeCheck: true` only if you need to bypass the check.

### Upgrade to 2026.5.1000

Chart `2026.5.1000` (SonarQube Server 2026.5 LTA) contains the following breaking or behavior changes:

* **Elasticsearch 9**: see [above](#elasticsearch-8-to-9-upgrading-from-20263-or-earlier-to-20264-or-later) when upgrading from SonarQube Server 2026.3 or earlier.
* **Probes**: the chart now manages the application node liveness/readiness probe handlers. Custom `exec`/`httpGet`/`tcpSocket`/`grpc` handlers are ignored; use `overrideCommand` instead. The default `timeoutSeconds` is now `5` on search and application nodes, and `applicationNodes.livenessProbe.failureThreshold` is now `8`.
* **Prometheus exporter**: the default scrape path is now `/metrics` instead of `/` (set `applicationNodes.prometheusExporter.metricsPath: /` to keep the old one), and built-in JVM metrics use OpenMetrics names (see [Export JMX metrics](#export-jmx-metrics)).
* **Memory defaults**: the `applicationNodes.resources` memory request and limit are now `8192M`, to fit the higher Web/CE heap defaults of SonarQube Server 2026.5. Make sure your nodes can schedule them, or set your own values.
* **ingress-nginx**: the bundled ingress-nginx controller subchart has been removed (see [below](#upgrade-from-versions-prior-to-202651000-ingress-nginx-controller-subchart-removed)). If you use `ingress.enabled`, set `ingress.ingressClassName` to your controller's class unless your cluster has a default `IngressClass`.
* **Agent runtimes**: `hunterAgent.serviceAccount.create` and `remediationAgent.serviceAccount.create` now default to `true`, so the runtimes no longer run under the top-level `serviceAccount`.

### Upgrade from versions prior to 2026.1.0

> **Note**: If you are not using the PostgreSQL dependency (`postgresql.enabled=false`), you can skip this section.

> **⚠️ Important**: Users upgrading to this chart from versions before 2026.1.0 and relying on the deprecated PostgreSQL dependency **must** follow the below instructions to avoid data loss.

Starting from `2026.1.0`, we removed the deprecated PostgreSQL dependency: this chart always requires an external database.

In order to upgrade to the newest chart from one version prior to this, you need to 

1. backup your database
2. import it to a new database
3. set the JDBC URL in the SonarQube chart

We identify the following migrations strategies and provide two example migration scripts to help you with this process. **These scripts are provided for reference and should be reviewed and adapted to your specific environment before use.** Both scripts are available in the `postgresql-migration-scripts/` directory of this chart's GitHub repository.

#### Option 1: Backup and Restore to an external database (Recommended)

You can perform a backup of the existing database and restore it on an external and fully managed database.

Please check `./postgresql-backup.sh` as a reference to create your own script that makes a backup file for external PostgreSQL migration:

```bash
./postgresql-backup.sh [OPTIONS] <postgres_service>

# Options:
# -n namespace    Kubernetes namespace (default: sonarqube)
# -u username     PostgreSQL username (default: sonarUser)
# -p password     PostgreSQL password (default: sonarPass)  
# -d database     Database name (default: sonarDB)
# -h, --help      Show help

# Examples:
./postgresql-backup.sh sonarqube-postgresql
./postgresql-backup.sh -n sonarqube -u myuser -p mypass -d mydb sonarqube-postgresql
```

Creates `sonarqube_backup.sql` for restoration to any external PostgreSQL service (AWS RDS, Azure Database, Google Cloud SQL, etc.).

**Example restoration to AWS RDS:**

```bash
PGPASSWORD=mypassword psql -h my-rds-endpoint.amazonaws.com -U myuser -d mydb < sonarqube_backup.sql
```

#### Option 2: In-Cluster Migration to an external Postgresql chart

If you wish to continue using a PostgreSQL chart to store SonarQube data, you can backup the database and restore it in a new (external) PostgreSQL chart having the same version (10.15.0).

Please check `postgresql-migration-k8s.sh` as a reference to build your own script that performs an in-cluster migration to a new PostgreSQL chart:

```bash
./postgresql-migration-k8s.sh [OPTIONS] <source_service>

# Options:
# -s source_ns    Source namespace (default: sonarqube)
# -t target_ns    Target namespace (default: sonarqube)
# -u username     PostgreSQL username (default: sonarUser)
# -p password     PostgreSQL password (default: sonarPass)
# -d database     Database name (default: sonarDB)
# -r release      New PostgreSQL release name (default: postgresql-external)
# -f values_file  Optional custom values.yaml file for PostgreSQL chart

# Examples:
./postgresql-migration-k8s.sh sonarqube-postgresql
./postgresql-migration-k8s.sh -s my-source-ns -t my-target-ns sonarqube-postgresql
```

This script:
* Installs a new PostgreSQL chart in the target namespace
* Migrates data directly between PostgreSQL instances within Kubernetes
* Provides the JDBC configuration for your SonarQube values.yaml

After migration, update your SonarQube configuration:

```yaml
jdbcOverwrite:
  jdbcUrl: "jdbc:postgresql://<your-endpoint>:5432/<database>"
  jdbcUsername: "<username>"
  jdbcSecretName: "<secret-with-the-password>"
  jdbcSecretPasswordKey: "<password-key>"
```

### Upgrade from versions prior to 2026.5.1000 (ingress-nginx controller subchart removed)

> **Note**: If you are not using the `ingress-nginx.enabled`/`nginx.enabled` bundled ingress-nginx controller subchart, you can skip this section. `ingress.enabled` (the plain `Ingress` resource, for use with your own controller) remains supported and needs no migration.

> **⚠️ Important**: Starting from `2026.5.1000`, this chart no longer bundles the deprecated `ingress-nginx.enabled`/`nginx.enabled` ingress-nginx controller subchart, following the retirement of the ingress-nginx controller. `httproute.enabled` (Gateway API) has been available since before this removal, so you can adopt it on your current chart version, side-by-side with your existing ingress, before upgrading to `2026.5.1000`. Alternatively, you can switch to `ingress.enabled` with a self-managed ingress controller.

We provide a migration script to help with this: `nginx-to-istio-migration.sh`, available in the `gateway-api-migration-scripts/` directory of this chart's GitHub repository. **This script is provided for reference and should be reviewed and adapted to your specific environment before use.**

By default (`--mode generate`), the script never touches your live release. It reads your current Helm values (from the live release, or a local file with `--values-file`), detects your existing ingress/ingress-nginx configuration (hostnames, TLS, nginx annotations, and any LoadBalancer Service annotations already set under `ingress-nginx.controller.service.annotations`), and writes into an output directory (default: `gateway-api-migration-<release-name>/`, override with `--output-dir`):
* a Gateway API `Gateway` manifest reusing your existing LoadBalancer Service annotations as-is — `--cloud aws|gcp|onprem` only selects which `--aws-*`/`--gcp-*`/`--metallb-pool` flags apply on top to add or override specific keys; nothing is invented on your behalf
* if the ingress-nginx controller is still enabled, a `values-coexist-<release-name>.yaml` that keeps `ingress`/`ingress-nginx` exactly as-is and only adds `httproute`, so both controllers can route traffic in parallel while you verify Istio
* a complete replacement `values.yaml` with `ingress`/`ingress-nginx` removed and `httproute` added (including an explicit `backendRefs` rule), everything else preserved as-is from your current release values

```bash
./nginx-to-istio-migration.sh --cloud aws --release-name sonarqube --namespace sonarqube-dce \
  --chart-flavor sonarqube-dce --aws-subnets subnet-abc,subnet-def

# Options:
# --cloud aws|gcp|onprem   Target environment (REQUIRED)
# --mode generate|apply    generate: render files only (default)
#                          apply: also install Gateway API CRDs + Istio and
#                          apply the Gateway
# --namespace ns           Kubernetes namespace of the release (default: sonarqube)
# --release-name name      Helm release name (default: sonarqube)
# --chart-flavor flavor    sonarqube|sonarqube-dce (default: sonarqube)
# --values-file path       Read values from a local file instead of the live release
# --hostnames h1,h2        Override auto-detected hostnames
# --output-dir dir         Directory to write generated files into (default: gateway-api-migration-<release-name>)
# --aws-subnets ids        Comma-separated subnet IDs. Only required if not
#                          already set via ingress-nginx.controller.service.annotations
# --coexist-chart-version  Chart version for the coexistence helm upgrade below,
#                          when ingress-nginx is still enabled (default: 2026.4.0,
#                          the last chart version that still bundles it)
# -h, --help               Show all options
```

With `--mode apply`, the script additionally installs the Gateway API CRDs and Istio (unless `--skip-istio`) and applies the generated `Gateway` manifest, so the Gateway exists before you upgrade. In either mode, the script only ever prints the next-step command(s) — it does not run them for you.

**Keep the generated `Gateway` manifest** (e.g. commit it to version control) — it isn't managed by `helm upgrade`, so it's your only record for re-applying it later (this or another cluster, disaster recovery, etc.).

If the ingress-nginx controller is still enabled, two `helm upgrade` commands are printed instead of one, since newer chart versions drop the `ingress-nginx` subchart entirely regardless of values:

```bash
# 1. Coexistence step — pinned to --coexist-chart-version, the last chart version
#    that still bundles ingress-nginx, so it keeps running alongside Istio
helm upgrade sonarqube sonarqube/sonarqube-dce --version 2026.4.0 -f gateway-api-migration-sonarqube/values-coexist-sonarqube.yaml -n sonarqube-dce

# 2. Verify Istio is handling traffic correctly, e.g. port-forward the Istio
#    ingress gateway Service (kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80)
#    and curl it with the Host header set to your hostname — or use whatever else
#    fits your setup (its LoadBalancer URL, a temporary DNS record, etc.)

# 3. Cutover step — once verified, moves to the target chart version and removes
#    the ingress-nginx controller
helm upgrade sonarqube sonarqube/sonarqube-dce -f gateway-api-migration-sonarqube/values-gateway-api-sonarqube.yaml -n sonarqube-dce
```

Otherwise, a single `helm upgrade` is printed:

```bash
helm upgrade sonarqube sonarqube/sonarqube-dce -f gateway-api-migration-sonarqube/values-gateway-api-sonarqube.yaml -n sonarqube-dce
```

Run these commands yourself once you've reviewed the generated files and are ready to switch your release over to Gateway API.

### ApplicationNodes renamed to applicationNodes

Prior to SonarQube Server Datacenter 10.8, we used different naming conventions for `searchNodes` and `ApplicationNodes`: camel case in the former and not in the latter.

Starting from 10.8, `ApplicationNodes` is deprecated in favor of `applicationNodes`. `ApplicationNodes` is still accepted, but it will be removed in a future release, so we advise you to rename it (if you are interested in the technical implementation, please take a look at this [PR](https://github.com/SonarSource/helm-chart-sonarqube/pull/586)).

Please report any encountered bugs to <https://community.sonarsource.com/>.

### Upgrade from the old sonarqube-lts to this chart

The sonarqube-lts chart was never a Data Center Edition chart. To move an 8.9 LTA installation to this chart, please refer to the [SonarQube 9.9 upgrade guide](https://docs.sonarsource.com/sonarqube-server/9.9/setup-and-upgrade/upgrade-the-server/upgrade-guide/); SonarQube 8.9 LTA is end-of-life.

## Installing previous chart versions

### Installing the SonarQube 9.9 LTA chart

The version of the chart for the SonarQube 9.9 LTA, which is end-of-life, is being distributed as the `7.x.x` version of this chart.

In order to use it, please set the version constraint `~7`, which is equivalent to `>=7.0.0 <8.0.0`. That version parameter **must** be used in every helm related command including `install`, `upgrade`, `template`, and `diff` (don't treat this as an exhaustive list).

Example:

```Bash
helm upgrade --install -n sonarqube-dce --version '~7' sonarqube sonarqube/sonarqube-dce --set ApplicationNodes.jwtSecret=$JWT_SECRET
```

## How to use it

Take some time to read the Deploy [SonarQube on Kubernetes](https://docs.sonarsource.com/sonarqube-server/server-installation/data-center-edition/introduction) page.
SonarQube deployment on Kubernetes has been tested with the recommendations and constraints documented there, and deployment has some limitations.

## Uninstalling the chart

To uninstall/delete the deployment:

```bash
helm list -n sonarqube-dce
helm uninstall -n sonarqube-dce <yourReleaseName>
```

## Setting up an external database for quick testing

In order to perform a quick testing of the chart, you can install a [postgresql chart](https://artifacthub.io/packages/helm/bitnami/postgresql) on your cluster. You can look at [this setup example](../../.github/scripts/setup_external_postgres.sh) to get install the chart. For more information and settings, please refer to the chart documentation.

After the database is available, please set the values, as in the following example.

```
jdbcOverwrite:
  jdbcUrl: "jdbc:postgresql://<release-name>-postgresql.<namespace>.svc.cluster.local:5432/<database-name>"
  jdbcUsername: "<username>"
  jdbcSecretName: "<release-name>-postgresql"
  jdbcSecretPasswordKey: "postgres-password"
```

## Prerequisites and suggested settings for production

Please read the official documentation prerequisites [here](https://docs.sonarsource.com/sonarqube-server/server-installation/server-host-requirements).

### Kubernetes - Pod Security Standards

Here is the list of containers that are compatible with the [Pod Security levels](https://kubernetes.io/docs/concepts/security/pod-security-admission/#pod-security-levels):

* privileged:
  * `init-sysctl`
* baseline:
  * `init-fs`
* restricted:
  * SQ application containers
  * SQ init containers.

When the [agentic features](#agentic-features) are enabled with `gvisor.enabled` and `gvisor.installer.enabled` (the default, outside OpenShift), the chart also deploys the `gvisor-installer` DaemonSet in the release namespace, which requires the **privileged** level.

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
```

The init containers additionally set `readOnlyRootFilesystem: true`; the application and search containers do not by default.

Based on that, one can run the SQ helm chart in a full restricted namespace, by deactivating the `initSysctl.enabled` and `initFs.enabled` parameters, which require root access, and, with the agentic features, `gvisor.installer.enabled` (see [Production use case](#production-use-case)).

Please take a look at [production-use-case](#production-use-case) for more information or directly at the values.yaml file.

### Elasticsearch prerequisites

SonarQube runs Elasticsearch under the hood.

Elasticsearch is rolling out (strict) prerequisites that cannot be disabled when running in production context (see [this](https://www.elastic.co/blog/bootstrap_checks_annoying_instead_of_devastating) blog post regarding bootstrap checks, and the [official guide](https://www.elastic.co/guide/en/elasticsearch/reference/current/bootstrap-checks.html)).

Because of such constraints, even when running in Docker containers, SonarQube requires some settings at the host/kernel level.

Please carefully read the following and make sure these configurations are set up at the host level:

* [vm.max_map_count](https://www.elastic.co/guide/en/elasticsearch/reference/current/vm-max-map-count.html#vm-max-map-count)
* [seccomp filter should be available](https://www.elastic.co/docs/deploy-manage/deploy/self-managed/bootstrap-checks)

In general, please carefully read the Elasticsearch's [documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/system-config.html) and specifically [here](https://www.elastic.co/guide/en/cloud-on-k8s/current/k8s-virtual-memory.html) for tutorial on how to change those parameters.

### Production use case

The SonarQube helm chart is packed with multiple features enabling users to install and test SonarQube on Kubernetes easily.

Nonetheless, if you intend to run a production-grade SonarQube please follow these recommendations.

* Set `initSysctl.enabled` to **false**. This parameter would run **root** `sysctl` commands, while those sysctl-related values should be set by the Kubernetes administrator at the node level (see [here](#elasticsearch-prerequisites))
* Set `initFs.enabled` to **false**. This parameter would run **root** `chown` commands. The parameter exists to fix non-posix, CSI, or deprecated drivers.
* If you enable the [agentic features](#agentic-features), install the sandbox runtime yourself: provision gVisor (`runsc`) on the nodes, or use your provider's sandbox, and set `gvisor.installer.enabled` to **false**. The bundled installer runs a **privileged** DaemonSet that modifies the node's containerd configuration. Keep `gvisor.enabled` set to **true**, or bring your own runtime (see [Sandboxing](#sandboxing)). On OpenShift, install the OpenShift sandboxed containers operator (Kata).
* If you want autoscaling for the agentic features, install [KEDA](https://keda.sh/) cluster-wide yourself and leave `keda.enabled` set to **false**. KEDA is a cluster-wide operator whose lifecycle should not be tied to a SonarQube release. Vortex autoscaling requires KEDA `>= 2.20`.
* If your cluster spans multiple failure domains, configure `searchNodes.topologySpreadConstraints` to spread search pods across zones and nodes. This is especially important for search pods because they are stateful and depend on persistent volumes.

#### Spreading pods across topology domains

If your Kubernetes cluster spans multiple failure domains such as availability zones, we recommend configuring `topologySpreadConstraints` for SonarQube pods.

For `applicationNodes`, spreading across zones or hosts improves resilience, but those pods are stateless and can usually be recreated on another eligible node when capacity is available.

For `searchNodes`, spreading is more important in production. Search pods are stateful and use persistent volumes. In many Kubernetes environments, those volumes are provisioned in a single topology domain such as an availability zone. Once a PVC is bound, the search pod usually has to restart on a node in the same zone. If multiple search pods are initially placed in the same zone and that zone becomes unavailable, the search cluster may be left with fewer than two running nodes.

To reduce that risk, make sure the cluster has eligible worker nodes in multiple zones, that nodes expose the labels used by your constraints, and that `searchNodes.topologySpreadConstraints` is set before installation or before scaling the search cluster. Using `whenUnsatisfiable: DoNotSchedule` for search pods is often preferable in production because it prevents silently placing multiple replicas in the same failure domain, but it also means new pods will remain pending until enough eligible nodes are available.

The example below spreads search pods across availability zones by using the standard `topology.kubernetes.io/zone` label:

```yaml
searchNodes:
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: topology.kubernetes.io/zone
      whenUnsatisfiable: DoNotSchedule
      labelSelector:
        matchLabels:
          app: sonarqube-dce-search
```

For application pods, a similar configuration can be used when you want to spread replicas across zones while keeping scheduling more flexible:

```yaml
applicationNodes:
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: topology.kubernetes.io/zone
      whenUnsatisfiable: ScheduleAnyway
      labelSelector:
        matchLabels:
          app: sonarqube-dce
```

#### CPU and memory settings

Monitoring CPU and memory is an important part of software reliability. The SonarQube helm chart comes with default values for CPU and memory requests and limits.

Xmx defines the maximum size of the JVM heap, this is **not** the maximum memory the JVM can allocate. For this reason, it is recommended to set the sum of the Xmx values to ~80% of the memory available to the container (in Kubernetes, this corresponds to requests and limits).

Please find here the default SonarQube Xmx parameters to setup the memory requests and limits accordingly.

| Nodes            | Web | Compute Engine | Search | Sum of Xmx |
| ---------------- | --- | -------------- | ------ | ---------- |
| applicationNodes | 2G  | 4G             | -      | 6G         |
| searchNodes      | -   | -              | 2G     | 2G         |

The chart defaults are:

* `searchNodes.resources` memory request/limit: `3072M`, which fits the search nodes' heap.
* `applicationNodes.resources` memory request/limit: `8192M`, which fits the application nodes' heap.

The default CPU limit (`800m`) is low for production workloads; raise it according to your analysis load.

Given that memory is a “non-compressible” resource, we advise you to set the memory requests and limits to the **same**, making memory a guaranteed resource. This is needed especially for production use cases.

When the [agentic features](#agentic-features) are enabled, size your nodes for these additional default requests and limits (CPU / memory / ephemeral storage):

| Component | Requests | Limits |
| --------- | -------- | ------ |
| Vortex | `1` / `6Gi` / `2Gi` | `2` / `8Gi` / `2Gi` |
| Agent Orchestrator | `250m` / `512Mi` / `512Mi` | `1` / `1Gi` / `2Gi` |
| Hunter Agent (per replica) | `1` / `8Gi` / `15Gi` | `2` / `8Gi` / `15Gi` |
| Remediation Agent (per replica) | `1` / `2Gi` / `10Gi` | `4` / `8Gi` / `50Gi` |
| Agent Egress Proxy (2 replicas) | `50m` / `64Mi` / `64Mi` | `250m` / `128Mi` / `128Mi` |
| MCP Server | not set | not set |

To get some guidance when setting the Xmx and Xms values, please refer to this [documentation](https://docs.sonarsource.com/sonarqube-server/server-installation/system-properties/configuration-methods) and set the environment variables or sonar.properties accordingly.

## Ingress usage

> **Note**: The bundled `ingress-nginx.enabled`/`nginx.enabled` ingress-nginx controller subchart has been removed, following the retirement of the ingress-nginx controller (announced in November 2025, effective March 2026). `ingress.enabled` (the plain `Ingress` resource) remains supported, for use with a self-managed ingress controller.
We recommend migrating to the [Gateway API](https://gateway-api.sigs.k8s.io/guides/) via `httproute.enabled` (see the `httproute.*` values below). If you continue using `ingress.enabled`, please refer to the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) for a list of controllers to install yourself.

### Path

Some clouds may need the path to be `/*` instead of `/`. Try this first if you are having issues getting traffic through the ingress.

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

This Helm chart offers the possibility to monitor SonarQube with Prometheus. You can find [Information on SonarQube monitoring on Kubernetes](https://docs.sonarsource.com/sonarqube-server/server-installation/on-kubernetes-or-openshift/set-up-monitoring) in the SonarQube documentation.

### Export JMX metrics

The prometheus exporter (`applicationNodes.prometheusExporter.enabled=true`) converts the JMX metrics into a format that Prometheus can understand. After the metrics are exported, you can connect your Prometheus instance and scrape them.

Per default the JMX metrics for the Web Bean and the CE Bean are exposed on port 8000 and 8001. These values can be configured with `applicationNodes.prometheusExporter.webBeanPort` and `applicationNodes.prometheusExporter.ceBeanPort`.

The exporter uses version `1.6.0` by default. Versions `1.1.0` and later are downloaded from GitHub Releases, while earlier versions are downloaded from Maven Central. GitHub downloads redirect to `release-assets.githubusercontent.com`; restricted environments must allowlist both `github.com` and `release-assets.githubusercontent.com`, or set `applicationNodes.prometheusExporter.downloadURL` to an accessible JAR URL. Exporter metrics are served on `/metrics` by default; set `applicationNodes.prometheusExporter.metricsPath` to `/` when using an exporter or configuration that serves metrics at the root path. Set `applicationNodes.prometheusExporter.sha256` to optionally verify the downloaded JAR; the checksum must match the selected version or custom URL.

In version 1.6.0, built-in JVM metric names use OpenMetrics naming (for example, `jvm_memory_bytes_used` is now `jvm_memory_used_bytes`), so update related dashboards and alerts. Metrics generated from `config.rules` are unaffected.

### PodMonitor

If a Prometheus Operator is deployed in your cluster, you can enable a PodMonitor resource with `applicationNodes.prometheusMonitoring.podMonitor.enabled`. It scrapes the Prometheus endpoint `/api/monitoring/metrics` exposed by the SonarQube application nodes and, when `applicationNodes.prometheusExporter.enabled` is `true`, the exporter ports on `applicationNodes.prometheusExporter.metricsPath`.

If running on OpenShift, make sure your account has permissions to create PodMonitor resources under the monitoring.coreos.com/v1 apiVersion.

## OpenShift

The chart can be installed on OpenShift by setting `OpenShift.enabled=true`. Among the others, please note that this value will disable the initContainer that performs the settings required by Elasticsearch (see [here](#elasticsearch-prerequisites)). Furthermore, we strongly recommend following the [Production Use Case guidelines](#production-use-case).

Please note that `OpenShift.createSCC` is deprecated and should be set to `false`. The default securityContext, together with the production configurations described [above](#production-use-case), is compatible with restricted SCCv2.

The below command will deploy SonarQube on the Openshift Kubernetes cluster.

```bash
helm repo add sonarqube https://SonarSource.github.io/helm-chart-sonarqube
helm repo update
kubectl create namespace sonarqube-dce # If you dont have permissions to create the namespace, skip this step and replace all -n with an existing namespace name.
# Please take a look at the official documentation https://docs.sonarsource.com/sonarqube-server/server-installation/data-center-edition/introduction
export JWT_SECRET=$(echo -n "your_secret" | openssl dgst -sha256 -hmac "your_key" -binary | base64) 
export MONITORING_PASSCODE="yourPasscode"
export JDBC_URL="jdbc:postgresql://<your-db-host>:5432/<your-database>" # must be replaced: the chart rejects the placeholder URL
export JDBC_USERNAME="sonar"
export JDBC_PASSWORD_SECRET_NAME="jdbc-secret"
export JDBC_PASSWORD_SECRET_KEY="jdbc-password"
helm upgrade --install -n sonarqube-dce sonarqube sonarqube/sonarqube-dce \
  --set applicationNodes.jwtSecret=$JWT_SECRET \
  --set OpenShift.enabled=true \
  --set monitoringPasscode=$MONITORING_PASSCODE \
  --set jdbcOverwrite.jdbcUrl=$JDBC_URL \
  --set jdbcOverwrite.jdbcUsername=$JDBC_USERNAME \
  --set jdbcOverwrite.jdbcSecretName=$JDBC_PASSWORD_SECRET_NAME \
  --set jdbcOverwrite.jdbcSecretPasswordKey=$JDBC_PASSWORD_SECRET_KEY
```

If you want to make your application publicly visible with Routes, you can set `OpenShift.route.enabled` to true. Please check the [configuration details](#openshift-1) to customize the Route base on your needs.

### Defunct (zombie) processes from probes

The default `readinessProbe` and `livenessProbe` are `exec` probes that fork short-lived processes inside the container on every invocation (the `readinessProbe` runs `curl` piped into `grep`, the `livenessProbe` runs `curl`). On some OpenShift / kubelet versions, when a probe exceeds its `timeoutSeconds` (default `5`) the kubelet kills the probe's parent shell before its child processes finish. Those children are then reparented to PID 1 (the SonarQube JVM, which does not reap them) and remain as defunct (`<defunct>` / zombie) processes. Because the probe runs throughout the pod's lifecycle, these can slowly accumulate and, in extreme cases, approach the pod's thread/process limit.

If you observe a growing number of defunct processes on the application pods, increase the probe timeout further to give the command enough time to complete before the kubelet kills it, for example:

```yaml
applicationNodes:
  readinessProbe:
    timeoutSeconds: 10
  livenessProbe:
    timeoutSeconds: 10
```

A value comfortably above the default prevents the probe command from being killed mid-execution and stops the accumulation of defunct processes.

### Setting up an external database for testing

In order to perform a quick testing of the chart, you can install a [postgresql chart](https://artifacthub.io/packages/helm/bitnami/postgresql) on your cluster, then set the appropriate values in the chart. To have the postgresql chart running in Openshift, you can look at [this example values file](./openshift-verifier/postgres-values.yaml). For more information and settings, please refer to the chart documentation.

## Autoscaling

The SonarQube applications nodes can be set to automatically scale up and down based on their average CPU utilization. This is particularly useful when scanning new projects or evaluating Pull Requests with SonarQube. In order to enable the autoscaling, you can rely on the `applicationNodes.hpa` parameters.

Please ensure the [Metrics Server](https://github.com/kubernetes-sigs/metrics-server) is installed in your cluster to provide resource usage metrics. You can deploy it using:

```
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

```

### Upgrading the Helm chart

When upgrading your SonarQube instance, due to high CPU usage, it is recommended to disable the autoscaling before the upgrade process, re-enabling it afterwards.

You can achieve that by either setting `applicationNodes.hpa.enabled` to `false` or by setting `applicationNodes.hpa.maxReplicas` to be the same value as `applicationNodes.hpa.minReplicas`.

## Additional features

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

Since SonarQube needs Elasticsearch, some [bootstrap checks](https://www.elastic.co/guide/en/elasticsearch/reference/current/bootstrap-checks.html) of the host settings are done at start on the search nodes.

This chart offers the option to use an initContainer in privileged mode to automatically set certain kernel settings on the kube worker. While this can ensure proper functionality of Elasticsearch, modifying the underlying kernel settings on the Kubernetes node can impact other users. It may be best to work with your cluster administrator to either provide specific nodes with the proper kernel settings, or ensure they are set cluster wide.

Auto-configuration of the kube worker node is controlled by `initSysctl.enabled`, which is `true` by default (and disabled on OpenShift). The initContainer runs on the search nodes only.

This will run `sysctl -w vm.max_map_count=524288` (`initSysctl.vmMaxMapCount`) on the workers where the search pods get scheduled, together with `fs.file-max`, `nofile` and `nproc` (`initSysctl.fsFileMax`, `initSysctl.nofile`, `initSysctl.nproc`). The kernel default for `vm.max_map_count` is usually `65530`, which is too low.

To disable worker node configuration, set `initSysctl.enabled` to `false`. The nodes running the search pods must then already meet these settings.

### MCP (Model Context Protocol) Server

When `mcp.enabled` is set to `true`, the chart deploys a separate MCP server pod alongside the SonarQube application nodes and automatically wires the two together.

**What gets deployed:**
- A `Deployment` running the MCP container
- A `ClusterIP` Service exposing port `8080` within the cluster
- A `PersistentVolumeClaim` for MCP's `/data` directory (when `mcp.persistence.enabled=true`)

**How the integration works:**

The MCP pod waits for SonarQube to report `"status":"UP"` before starting (via an init container). Once running, the application nodes are configured to call MCP at `http://<release>-sonarqube-dce-mcp:8080` via the `SONAR_MCP_SERVERURL` and `SONAR_MCP_ENABLED` environment variables, which the chart injects automatically.

**Minimal configuration example:**

```yaml
mcp:
  enabled: true
  image:
    repository: sonarsource/sonarqube-mcp
    tag: "2026.5.0"
```

**Accessing the MCP server locally:**

```bash
kubectl port-forward svc/<release>-sonarqube-dce-mcp 8080:8080 -n <namespace>
```

**Persistence:**

MCP uses `/data` to store its files. `mcp.persistence.enabled` defaults to `true`. For local testing only (e.g. Kind/minikube), you can disable it — but data will be lost on pod restarts:

```yaml
mcp:
  enabled: true
  persistence:
    enabled: false
```

**Storage permissions:**

The MCP server runs as a non-root user (UID 1000, GID 0) and must write to `/data`. The chart sets `mcp.podSecurityContext.fsGroup: 0` by default so the mounted volume is group-writable by the MCP process.

**Your storage provider must honor filesystem group ownership changes (`fsGroup`).** Drivers such as NFS, hostPath, CSI drivers configured with `fsGroupPolicy: None`, and many pre-provisioned or `existingClaim` volumes ignore `fsGroup`. In those cases the volume stays root-owned and MCP fails to start with errors like `Cannot create directory` under `/data`. To handle them, either pre-provision a `/data` volume writable by group `0` (or UID `1000`), or add an ownership-fixing init container:

```yaml
mcp:
  initContainers:
    - name: chown-data
      image: busybox:1.36
      command: ["sh", "-c", "chown -R 1000:0 /data && chmod -R g+rwX /data"]
      securityContext:
        runAsUser: 0
      volumeMounts:
        - name: mcp-data
          mountPath: /data
```

On OpenShift, do not set `fsGroup`/`runAsUser`/`runAsGroup` — the platform assigns the UID from the namespace's SCC range, and the chart removes these keys automatically when `OpenShift.enabled=true`. Note that the `chown` init container above runs as root and is therefore rejected by the `restricted` Pod Security Standard and by OpenShift's default SCC; in those environments, pre-provision a `/data` volume writable by group `0` instead — the MCP image's process runs under group `0` regardless of which UID the SCC assigns, so a UID-specific owner will not work.

**TLS (encrypted communication):**

When `mcp.tls.enabled` is set to `true`, the MCP server starts in HTTPS mode using the keystore from `mcp.tls.keystoreSecretName`. Application nodes connect to it over `https://`. Alternatively, `istio.enabled` encrypts this hop together with every other one (see [Securing communication with TLS](#securing-communication-with-tls)).

If the keystore uses a self-signed certificate, SonarQube's JVM will reject the connection unless the CA certificate is trusted. Use the `caCerts` feature to import it into SonarQube's JVM truststore:

1. Create a Secret containing the CA certificate in PEM format:

   ```bash
   kubectl create secret generic mcp-ca-cert \
     --from-file=mcp-ca.crt=/path/to/ca.pem \
     -n <namespace>
   ```

2. Reference it in your values:

   ```yaml
   mcp:
     tls:
       enabled: true
       keystoreSecretName: mcp-keystore-secret
       keystoreSecretKey: keystore.p12
       passwordSecretName: mcp-keystore-password
       passwordSecretKey: password
       keystoreType: PKCS12

   caCerts:
     enabled: true
     secret: mcp-ca-cert
   ```

**Scheduling:**

`mcp.nodeSelector`, `mcp.affinity` and `mcp.tolerations` set scheduling for the MCP pod; each wins over the chart's global `.Values.nodeSelector`/`.affinity`/`.tolerations` when set, and falls back to it otherwise — same convention as `vortexAnalysis`/`agentOrchestrator`/`hunterAgent`/`remediationAgent`. Note that the search and application nodes use the opposite precedence: the chart-wide `nodeSelector` wins over `searchNodes.nodeSelector`/`applicationNodes.nodeSelector`. `mcp.topologySpreadConstraints` is MCP-specific with no chart-wide equivalent to fall back to. The chart-wide `priorityClassName` value is applied to the MCP pod automatically; there is no separate `mcp.priorityClassName`.

### Agentic features

The chart can deploy the SonarQube agentic components next to the SonarQube application nodes:

* **Vortex** (`vortexAnalysis.enabled`): analysis service that SonarQube sends analysis requests to.
* **Agent Orchestrator** (`agentOrchestrator.enabled`): dispatches jobs to the agent runtimes. Required by both agents.
* **Hunter Agent** (`hunterAgent.enabled`): detection agent runtime.
* **Remediation Agent** (`remediationAgent.enabled`): remediation agent runtime. Enabling it also enables Vortex.
* **Agent Egress Proxy** (`agentEgressProxy`): a forward proxy that is the only way out of the cluster for the agent runtimes. It has no toggle: it is deployed automatically whenever an agent is enabled.

**Prerequisites:**

* **Database.** The Agent Orchestrator shares SonarQube's database and supports **PostgreSQL** only. SonarQube itself also supports Microsoft SQL Server and Oracle (see the [installation requirements](https://docs.sonarsource.com/sonarqube-server/server-installation/server-host-requirements)), but the agentic features need PostgreSQL. `agentOrchestrator.coreDb.*` defaults to the `jdbcOverwrite` values and can override them individually.
* **Storage.** The agentic features share an object store: the Agent Orchestrator writes the job artifacts to it (`agentOrchestrator.storage`), SonarQube reads the agent job logs from it (`sonar.agentic.storage.*` in `applicationNodes.sonarProperties`), and Vortex restores the analysis context from it (`vortexAnalysis.storage`). The simplest setup points all three at the same bucket. Supported backends:
  * `S3` (default, recommended for production): AWS S3, or any S3-compatible endpoint such as MinIO via `endpoint` and path-style addressing. Credentials come from inline keys, an `existingSecret`, or, when both are blank, the pod's IAM identity (node instance role or IRSA through the component's `serviceAccount.annotations`).
  * `FILESYSTEM` / `NFS`: a shared `ReadWriteMany` volume mounted through `extraVolumes`/`extraVolumeMounts`, with `storage.filesystem.baseDir` set. Use a single `securityContext.fsGroup` across the orchestrator and the agents.
  * Vortex additionally supports `AZURE` and `GCS`.

  When the pods authenticate with their IAM identity (e.g. IRSA), SonarQube (`serviceAccount`), the Agent Orchestrator and Vortex (`<component>.serviceAccount`) each need their own ServiceAccount (`create: true`) carrying the role annotation. The agent runtimes don't access the store directly (the orchestrator hands them presigned URLs) and get their own ServiceAccount by default, so they never inherit SonarQube's cloud role; keep `<hunterAgent|remediationAgent>.serviceAccount.create` set to `true`.
* **Signing secret.** The agentic components sign the messages they exchange with keys derived from one instance secret you create:

  ```bash
  kubectl create secret generic agentic-instance-secret -n <namespace> \
    --from-literal=instance-secret="$(openssl rand -base64 48)"
  ```

* **A sandboxed container runtime** on the nodes running the agents (see [Sandboxing](#sandboxing)).
* **KEDA**, only if you want to autoscale the agents or Vortex.

No LLM provider key is needed at install time: the LLM provider is configured in the SonarQube UI once the features are running. Only its hostname has to be allowed through the egress proxy. For the Remediation Agent to open pull requests, bind the project to a GitHub App DevOps Platform integration in SonarQube.

**Minimal configuration example:**

```yaml
monitoringPasscode: "<your-passcode>"   # or monitoringPasscodeSecretName/Key
applicationNodes:
  jwtSecret: "<your-jwt-secret>"      # or existingJwtSecret
  sonarProperties:
    sonar.agentic.storage.type: S3
    sonar.agentic.storage.bucket: my-agentic-artifacts
    sonar.agentic.storage.region: eu-west-1
jdbcOverwrite:
  jdbcUrl: "jdbc:postgresql://postgres.example.com:5432/sonarqube"
  jdbcUsername: "sonarqube"
  jdbcSecretName: "sonarqube-db"
  jdbcSecretPasswordKey: "password"
agenticSigningSecret:
  existingSecret: agentic-instance-secret
agentOrchestrator:
  enabled: true
  storage:
    type: S3
    region: eu-west-1
    bucket: my-agentic-artifacts
    pathStyle: false
vortexAnalysis:
  storage:
    type: S3
    region: eu-west-1
    bucket: my-agentic-artifacts
hunterAgent:
  enabled: true
remediationAgent:
  enabled: true
agentEgressProxy:
  allowedDomains:
    - api.anthropic.com                               # your LLM provider
    - my-agentic-artifacts.s3.eu-west-1.amazonaws.com # agentOrchestrator.storage
```

The Vortex pod can take several minutes to become ready on a first start, while it loads its analyzers.

#### Sandboxing

The agent runtimes execute LLM-driven jobs, so they run under a sandboxed container runtime:

* **gVisor (default).** `gvisor.enabled=true` creates a `gvisor` RuntimeClass (handler `runsc`) and schedules the agents on nodes labeled `gvisor.enabled: "true"`. The bundled installer (`gvisor.installer.enabled`) is convenient for testing, but it is a privileged DaemonSet that only works on self-managed containerd nodes. In production, provision `runsc` yourself (node image, GKE Sandbox, ...), label the nodes, and set `gvisor.installer.enabled=false`. `RuntimeClass` is cluster-scoped: give each release its own `gvisor.runtimeClassName`.
* **Bring your own secure runtime.** Set `gvisor.enabled=false`, `agentRuntimeSandbox.enabled=true` and `agentRuntimeSandbox.runtimeClassName` to a RuntimeClass your platform provides, e.g. Kata Containers (`kata-mshv-vm-isolation` on AKS Pod Sandboxing).
* **OpenShift.** gVisor is not available; the agents run under Kata Containers through `OpenShift.agentRuntimeClassName` (default `kata`, or `kata-remote` for peer pods). Install the OpenShift sandboxed containers operator first: the chart does not create this RuntimeClass and fails the install if it is missing.

Running the agents without a sandbox (`gvisor.enabled=false` with no `agentRuntimeSandbox`) is possible but not recommended.

#### Network egress

The agent runtimes reach the internet only through the Agent Egress Proxy, which allows the domains in `agentEgressProxy.allowedDomains` and nothing else. Add:

* your LLM provider's API hostname;
* the hostname of `agentOrchestrator.storage` (agents read and write job artifacts directly through presigned URLs), otherwise jobs fail at the first artifact download;
* any other endpoint your agents must reach.

Prefer exact hostnames: an entry with a leading dot (`.example.com`) also allows every subdomain. Overlapping entries (for example `.example.com` together with `api.example.com`) are rejected.

The proxy allows ports 80 and 443. For another port, add it both to `agentEgressProxy.extraSquidConf` (`Safe_ports`/`SSL_ports`) and to `agentEgressProxy.networkPolicy.egressPorts`.

#### Autoscaling

Autoscaling is off by default:

* `agentOrchestrator.autoscaling`: a standard `HorizontalPodAutoscaler` on CPU/memory.
* `hunterAgent.autoscaling` / `remediationAgent.autoscaling`: a KEDA `ScaledObject` driven by the orchestrator's job queue.
* `vortexAnalysis.autoscaling`: a KEDA `ScaledObject` driven by Vortex's concurrent requests. Requires KEDA `>= 2.20`.

`minReplicas` must be at least `2`, except for Vortex with `vortexAnalysis.autoscaling.aggregateAcrossReplicas: false`, where it must be exactly `1`. The KEDA CRDs are detected at install time; under `helm template` set `agentKeda.assumeInstalled: true`. With GitOps tools that apply rendered manifests (Argo CD, Flux), set `autoscaling.manageReplicas: false` so each sync does not reset the replica count.

To encrypt the traffic between the agentic components, see [Securing communication with TLS](#securing-communication-with-tls).

### Securing communication with TLS

Each hop of a deployment can be encrypted:

1. **Traffic from users to SonarQube.** Terminate TLS at the entry point: `ingress.tls` for an Ingress, a TLS listener on the Gateway referenced by `httproute`, or `OpenShift.route.tls` on OpenShift.
2. **Traffic between the chart's workloads.** Set `istio.enabled=true` (requires Istio installed in sidecar mode; SonarQube Server is tested with it). Every chart-owned workload (the application and search nodes, MCP, Vortex, the Agent Orchestrator and the Agent Egress Proxy) gets an Istio sidecar and a `STRICT` `PeerAuthentication`, so they only accept mutual TLS. Istio must know every port up front, so also pin the Hazelcast ports that the application nodes otherwise allocate dynamically; the chart fails the install if either is missing:

   ```yaml
   istio:
     enabled: true
   applicationNodes:
     webPort: 4023   # Web process cluster communication
     cePort: 4024    # Compute Engine process cluster communication
   ```

   Use `istio.revision` for a revisioned control plane. `istio.istiodClusterIP` (default `auto`) looks up istiod's address at install time; under `helm template`, set it to istiod's ClusterIP.
3. **Traffic from the sandboxed agent runtimes.** Standard sidecar injection does not work under gVisor or Kata, so the agents stay outside the mesh and the egress proxy accepts them through a `PERMISSIVE` exception. Set `istio.meshSidecar.enabled=true` to give them a mesh identity and remove that exception. Requires Kubernetes `>= 1.29` and a CNI that enforces NetworkPolicies.
4. **Application nodes to search nodes.** Set `nodeEncryption.enabled=true` and `searchNodes.searchAuthentication.enabled=true`. Elasticsearch nodes then identify themselves with certificates:
   1. Generate a certificate authority and a single certificate valid for all search nodes with [`elasticsearch-certutil`](https://www.elastic.co/guide/en/elasticsearch/reference/current/security-basic-setup-https.html#encrypt-http-communication). Its Subject Alternative Names must include every search pod's FQDN and the search Service name. For a release `sq` in namespace `sonar` with three search nodes:

      ```
      sq-sonarqube-dce-search-0.sq-sonarqube-dce-search.sonar.svc.cluster.local
      sq-sonarqube-dce-search-1.sq-sonarqube-dce-search.sonar.svc.cluster.local
      sq-sonarqube-dce-search-2.sq-sonarqube-dce-search.sonar.svc.cluster.local
      sq-sonarqube-dce-search
      ```

      `hostname -f` inside a search pod prints its FQDN.
   2. Rename the resulting `http.p12` to `elastic-stack-ca.p12`, store it in a Secret and set `searchNodes.searchAuthentication.keyStoreSecret` to its name, with the password in `keyStorePassword` or `keyStorePasswordSecret`.
   3. Set `searchNodes.searchAuthentication.userPassword`.
5. **Application nodes to MCP without Istio.** Set `mcp.tls.*` and add the CA to SonarQube's truststore with `caCerts` (see [MCP (Model Context Protocol) Server](#mcp-model-context-protocol-server)).
6. **Traffic from SonarQube to the database.** Add the TLS parameters to the JDBC URL, e.g. `jdbc:postgresql://host:5432/sonarqube?sslmode=verify-full`, and add its CA with `caCerts` if it is not publicly trusted. The Agent Orchestrator only takes the host and database name from that URL, not its parameters.
7. **Traffic to object storage and the LLM provider.** Use an `https://` storage endpoint. Agent traffic through the egress proxy is tunneled with HTTPS `CONNECT`; the proxy does not intercept TLS.

| Hop | Mechanism | Values |
| --- | --------- | ------ |
| Users → SonarQube | Ingress / Gateway / Route TLS | `ingress.tls`, `httproute`, `OpenShift.route.tls` |
| Between chart workloads | Istio mutual TLS | `istio.enabled`, `applicationNodes.webPort`, `applicationNodes.cePort` |
| Agent runtimes → egress proxy | Istio mutual TLS | `istio.meshSidecar.enabled` |
| Application nodes → search nodes | Elasticsearch TLS and authentication | `nodeEncryption.enabled`, `searchNodes.searchAuthentication.*` |
| Application nodes → MCP | HTTPS | `mcp.tls.*`, `caCerts` |
| SonarQube → database | JDBC TLS | `jdbcOverwrite.jdbcUrl`, `caCerts` |
| Agents → storage and LLM | HTTPS through the egress proxy | `agentOrchestrator.storage.endpoint`, `agentEgressProxy.allowedDomains` |

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
     SONAR_LOG_LEVEL: INFO
     SONAR_TELEMETRY_ENABLE: "false"
   ```

   Do not set the `SONAR_JDBC_*` variables this way: the chart always sets them from `jdbcOverwrite`. Keep the database password in a Secret referenced by `jdbcOverwrite.jdbcSecretName`/`jdbcOverwrite.jdbcSecretPasswordKey`.

2. Set the following in your `values.yaml` (using the key `extraConfig.secrets` to reference `Secret`s)

   ```yaml
   extraConfig:
     configmaps:
       - external-sonarqube-opts
   ```

## Configuration

The following table lists the configurable parameters of the SonarQube chart and their default values.

> **DEPRECATION NOTICE: ApplicationNodes values should be renamed to applicationNodes.** We deprecated `ApplicationNodes` (with capital **A**); it is still accepted, but it will be removed in a future release. We advise everyone to rename `ApplicationNodes` to `applicationNodes`. More information can be found [in the section above](#applicationnodes-renamed-to-applicationnodes).

### Search Nodes Configuration

| Parameter                                                 | Description                                                                                | Default                                                                |
| --------------------------------------------------------- | ------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------- |
| `searchNodes.image.repository`                            | search image repository                                                                    | `sonarqube`                                                            |
| `searchNodes.image.tag`                                   | search image tag                                                                           | `2026.5.0-datacenter-search`                                             |
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
| `searchNodes.readinessProbe.timeoutSeconds`               | ReadinessProbe timeout delay                                                               | `5`                                                                    |
| `searchNodes.livenessProbe.initialDelaySeconds`           | LivenessProbe initial delay for Search Node checking                                       | `0`                                                                    |
| `searchNodes.livenessProbe.periodSeconds`                 | LivenessProbe period between checking Search Node                                          | `30`                                                                   |
| `searchNodes.livenessProbe.failureThreshold`              | LivenessProbe threshold for marking as dead                                                | `6`                                                                    |
| `searchNodes.livenessProbe.timeoutSeconds`                | LivenessProbe timeout delay                                                                | `5`                                                                    |
| `searchNodes.startupProbe.initialDelaySeconds`            | StartupProbe initial delay for Search Node checking                                        | `20`                                                                   |
| `searchNodes.startupProbe.periodSeconds`                  | StartupProbe period between checking Search Node                                           | `10`                                                                   |
| `searchNodes.startupProbe.failureThreshold`               | StartupProbe threshold for marking as failed                                               | `24`                                                                   |
| `searchNodes.startupProbe.timeoutSeconds`                 | StartupProbe timeout delay                                                                 | `5`                                                                    |
| `searchNodes.resources.requests.memory`                   | memory request for Search Nodes                                                            | `3072M`                                                                |
| `searchNodes.resources.requests.cpu`                      | CPU request for Search Nodes                                                               | `400m`                                                                 |
| `searchNodes.resources.requests.ephemeral-storage`        | storage request for Search Nodes                                                           | `1536M`                                                                |
| `searchNodes.resources.limits.memory`                     | memory limit for Search Nodes. should not be under 3G                                      | `3072M`                                                                |
| `searchNodes.resources.limits.cpu`                        | CPU limit for Search Nodes                                                                 | `800m`                                                                 |
| `searchNodes.resources.limits.ephemeral-storage`          | storage limit for Search Nodes                                                             | `512000M`                                                              |
| `searchNodes.persistence.enabled`                         | enabled or disables the creation of VPCs for the Search Nodes                              | `true`                                                                 |
| `searchNodes.persistence.annotations`                     | PVC annotations for the Search Nodes                                                       | `{}`                                                                   |
| `searchNodes.persistence.storageClass`                    | Storage class to be used                                                                   | `""`                                                                   |
| `searchNodes.persistence.accessMode`                      | Volumes access mode to be set                                                              | `ReadWriteOnce`                                                        |
| `searchNodes.persistence.size`                            | Size of the PVC                                                                            | `5Gi`                                                                  |
| `searchNodes.persistence.uid`                             | UID used for init-fs container                                                             | `1000`                                                                 |
| `searchNodes.persistence.volumes`                         | Set existing volumes                                                                       | `[]`                                                                   |
| `searchNodes.persistence.guid`                            | GUID used for init-fs container                                                            | `0`                                                                    |
| `searchNodes.extraContainers`                             | Array of extra containers to run alongside                                                 | `[]`                                                                   |
| `searchNodes.extraInitContainers`                         | Array of extra init containers to run before the search container                          | `[]`                                                                   |
| `searchNodes.nodeSelector`                                | Node labels for search nodes' pods assignment, global nodeSelector takes precedence        | `{}`                                                                   |
| `searchNodes.affinity`                                    | Node / Pod affinities for searchNodes, global affinity takes precedence                    | `{}`                                                                   |
| `searchNodes.tolerations`                                 | List of node taints to tolerate for searchNodes, global tolerations take precedence        | `[]`                                                                   |
| `searchNodes.topologySpreadConstraints`                   | Topology spread constraints to apply to the search pods                                    | `[]`                                                                   |

### App Nodes Configuration

| Parameter                                                        | Description                                                                                                                                                                                                    | Default                                                                |
| ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `applicationNodes.image.repository`                              | app image repository                                                                                                                                                                                           | `sonarqube`                                                            |
| `applicationNodes.image.tag`                                     | app image tag                                                                                                                                                                                                  | `2026.5.0-datacenter-app`                                                |
| `applicationNodes.image.pullPolicy`                              | app image pull policy                                                                                                                                                                                          | `IfNotPresent`                                                         |
| `applicationNodes.image.pullSecret`                              | (DEPRECATED) app imagePullSecret to use for private repository                                                                                                                                                 | `nil`                                                                  |
| `applicationNodes.image.pullSecrets`                             | app imagePullSecrets to use for private repository                                                                                                                                                             | `nil`                                                                  |
| `applicationNodes.annotations`                                   | Map of annotations to add to the app pods                                                                                                                                                                      | `{}`                                                                   |
| `applicationNodes.env`                                           | Environment variables to attach to the app pods                                                                                                                                                                | `nil`                                                                  |
| `applicationNodes.podLabels`                                     | Map of labels to add to the app pods                                                                                                                                                                           | `{}`                                                                   |
| `applicationNodes.sonarProperties`                               | Custom `sonar.properties` key-value pairs for App Nodes (e.g., "applicationNodes.sonarProperties.sonar.log.level=DEBUG")                                                                              | `None`                                                                 |
| `applicationNodes.sonarSecretProperties`                         | Additional `sonar.properties` key-value pairs for App Nodes to load from a secret                                                                                                                              | `None`                                                                 |
| `applicationNodes.sonarSecretKey`                                | Name of existing secret used for settings encryption                                                                                                                                                           | `None`                                                                 |
| `applicationNodes.replicaCount`                                  | Replica count of the app Nodes                                                                                                                                                                                 | `2`                                                                    |
| `applicationNodes.podDisruptionBudget`                           | PodDisruptionBudget for the App Nodes                                                                                                                                                                          | `minAvailable: 1`                                                      |
| `applicationNodes.podDistributionBudget`                         | (DEPRECATED typo) PodDisruptionBudget for the App Nodes                                                                                                                                                        | `minAvailable: 1`                                                      |
| `applicationNodes.securityContext`                               | SecurityContext for the pod app nodes                                                                                                                                                                          | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `applicationNodes.containerSecurityContext`                      | SecurityContext for app container in sonarqube pod                                                                                                                                                             | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `applicationNodes.readinessProbe`                                | ReadinessProbe for the App. Handler is chart-managed; legacy `exec`, `httpGet`, `tcpSocket`, and `grpc` values are ignored. Use `applicationNodes.readinessProbe.overrideCommand` to customize it | `exec; curl api/system/status`                                                  |
| `applicationNodes.readinessProbe.overrideCommand`                | Optional command to use instead of the chart-managed readiness probe command (rendered with `tpl`, so Helm template expressions are supported)       | `None`                                                                          |
| `applicationNodes.readinessProbe.initialDelaySeconds`            | ReadinessProbe initial delay for app Node checking                                                                                                                                                             | `0`                                                                    |
| `applicationNodes.readinessProbe.periodSeconds`                  | ReadinessProbe period between checking app Node                                                                                                                                                                | `30`                                                                   |
| `applicationNodes.readinessProbe.failureThreshold`               | ReadinessProbe threshold for marking as failed                                                                                                                                                                 | `8`                                                                    |
| `applicationNodes.readinessProbe.timeoutSeconds`                 | ReadinessProbe timeout delay                                                                                                                                                                                   | `5`                                                                    |
| `applicationNodes.readinessProbe.sonarWebContext`                | (DEPRECATED) SonarQube web context for readinessProbe, please use sonarWebContext at the value top level instead                                                                                               | `/`                                                                    |
| `applicationNodes.livenessProbe`                                 | LivenessProbe for the App. Handler is chart-managed; legacy `exec`, `httpGet`, `tcpSocket`, and `grpc` values are ignored. Use `applicationNodes.livenessProbe.overrideCommand` to customize it | `exec: curl api/system/liveness`                                               |
| `applicationNodes.livenessProbe.overrideCommand`                 | Optional command to use instead of the chart-managed liveness probe command (rendered with `tpl`, so Helm template expressions are supported)        | `None`                                                                          |
| `applicationNodes.livenessProbe.initialDelaySeconds`             | LivenessProbe initial delay for app Node checking                                                                                                                                                              | `0`                                                                    |
| `applicationNodes.livenessProbe.periodSeconds`                   | LivenessProbe period between checking app Node                                                                                                                                                                 | `30`                                                                   |
| `applicationNodes.livenessProbe.failureThreshold`                | LivenessProbe threshold for marking as failed                                                                                                                                                                  | `8`                                                                    |
| `applicationNodes.livenessProbe.timeoutSeconds`                  | LivenessProbe timeout delay                                                                                                                                                                                    | `5`                                                                    |
| `applicationNodes.livenessProbe.sonarWebContext`                 | (DEPRECATED) SonarQube web context for livenessProbe, please use sonarWebContext at the value top level instead                                                                                                | `/`                                                                    |
| `applicationNodes.startupProbe`                                  | StartupProbe for the App                                                                                                                                                                                       | `httpGet: api/system/status`                                           |
| `applicationNodes.startupProbe.initialDelaySeconds`              | StartupProbe initial delay for app Node checking                                                                                                                                                               | `45`                                                                   |
| `applicationNodes.startupProbe.periodSeconds`                    | StartupProbe period between checking app Node                                                                                                                                                                  | `10`                                                                   |
| `applicationNodes.startupProbe.failureThreshold`                 | StartupProbe threshold for marking as failed                                                                                                                                                                   | `32`                                                                   |
| `applicationNodes.startupProbe.timeoutSeconds`                   | StartupProbe timeout delay                                                                                                                                                                                     | `1`                                                                    |
| `applicationNodes.startupProbe.sonarWebContext`                  | (DEPRECATED) SonarQube web context for startupProbe, please use sonarWebContext at the value top level instead                                                                                                 | `/`                                                                    |
| `applicationNodes.resources.requests.memory`                     | memory request for app Nodes                                                                                                                                                                                   | `8192M`                                                                |
| `applicationNodes.resources.requests.cpu`                        | CPU request for app Nodes                                                                                                                                                                                      | `400m`                                                                 |
| `applicationNodes.resources.requests.ephemeral-storage`          | storage request for app Nodes                                                                                                                                                                                  | `1536M`                                                                |
| `applicationNodes.resources.limits.memory`                       | memory limit for app Nodes. should not be under 6G                                                                                                                                                             | `8192M`                                                                |
| `applicationNodes.resources.limits.cpu`                          | CPU limit for app Nodes                                                                                                                                                                                        | `800m`                                                                 |
| `applicationNodes.resources.limits.ephemeral-storage`            | storage limit for app Nodes                                                                                                                                                                                    | `512000M`                                                              |
| `applicationNodes.prometheusExporter.enabled`                    | Use the Prometheus JMX exporter                                                                                                                                                                                | `false`                                                                |
| `applicationNodes.prometheusExporter.version`                    | jmx_prometheus_javaagent version; versions 1.1.0 and later download from GitHub Releases, earlier versions from Maven Central                                                                                  | `1.6.0`                                                                |
| `applicationNodes.prometheusExporter.metricsPath`                | HTTP path served by the jmx_prometheus_javaagent (`/` can be used for exporters or configurations serving metrics at the root)                                                                                 | `/metrics`                                                             |
| `applicationNodes.prometheusExporter.noCheckCertificate`         | Flag to not check server's certificate when downloading jmx_prometheus_javaagent                                                                                                                               | `false`                                                                |
| `applicationNodes.prometheusExporter.webBeanPort`                | Port where the jmx_prometheus_javaagent exposes the metrics for the webBean                                                                                                                                    | `8000`                                                                 |
| `applicationNodes.prometheusExporter.ceBeanPort`                 | Port where the jmx_prometheus_javaagent exposes the metrics for the ceBean                                                                                                                                     | `8001`                                                                 |
| `applicationNodes.prometheusExporter.downloadURL`                | Custom full download URL for the jmx_prometheus_javaagent.jar (overrides `prometheusExporter.version`)                                                                                                         | `""`                                                                   |
| `applicationNodes.prometheusExporter.sha256`                     | Optional SHA-256 checksum for the downloaded jmx_prometheus_javaagent.jar                                                                                                                                       | `None`                                                                 |
| `applicationNodes.prometheusExporter.config`                     | Prometheus JMX exporter config yaml for the web process, and the CE process if `prometheusExporter.ceConfig` is not set                                                                                        | see `values.yaml`                                                      |
| `applicationNodes.prometheusExporter.ceConfig`                   | Prometheus JMX exporter config yaml for the CE process (by default, `prometheusExporter.config` is used                                                                                                        | `None`                                                                 |
| `applicationNodes.prometheusExporter.httpProxy`                  | HTTP proxy for downloading JMX agent                                                                                                                                                                           | `""`                                                                   |
| `applicationNodes.prometheusExporter.httpsProxy`                 | HTTPS proxy for downloading JMX agent                                                                                                                                                                          | `""`                                                                   |
| `applicationNodes.prometheusExporter.noProxy`                    | No proxy for downloading JMX agent                                                                                                                                                                             | `""`                                                                   |
| `applicationNodes.prometheusExporter.securityContext`            | Security context for downloading the jmx agent                                                                                                                                                                 | see `values.yaml`                                                      |
| `applicationNodes.prometheusMonitoring.podMonitor.enabled`       | Enable Prometheus PodMonitor                                                                                                                                                                                   | `false`                                                                |
| `applicationNodes.prometheusMonitoring.podMonitor.namespace`     | (DEPRECATED) This value should not be set, as the PodMonitor's namespace has to match the Release Namespace                                                                                                    | `{{ .Release.Namespace }}`                                             |
| `applicationNodes.prometheusMonitoring.podMonitor.interval`      | Specify the interval how often metrics should be scraped                                                                                                                                                       | `30s`                                                                  |
| `applicationNodes.prometheusMonitoring.podMonitor.scrapeTimeout` | Specify the timeout after a scrape is ended                                                                                                                                                                    | `None`                                                                 |
| `applicationNodes.prometheusMonitoring.podMonitor.jobLabel`      | Name of the label on target services that prometheus uses as job name                                                                                                                                          | `None`                                                                 |
| `applicationNodes.prometheusMonitoring.podMonitor.labels`        | Additional labels to add to the PodMonitor                                                                                                                                                                     | `{}`                                                                   |
| `applicationNodes.plugins.install`                               | Link(s) to the plugin JARs to download and install                                                                                                                                                             | `[]`                                                                   |
| `applicationNodes.plugins.resources`                             | Plugin Pod resource requests & limits                                                                                                                                                                          | `{}`                                                                   |
| `applicationNodes.plugins.httpProxy`                             | For use behind a corporate proxy when downloading plugins                                                                                                                                                      | `""`                                                                   |
| `applicationNodes.plugins.httpsProxy`                            | For use behind a corporate proxy when downloading plugins                                                                                                                                                      | `""`                                                                   |
| `applicationNodes.plugins.noProxy`                               | For use behind a corporate proxy when downloading plugins                                                                                                                                                      | `""`                                                                   |
| `applicationNodes.plugins.image`                                 | Image for plugins container                                                                                                                                                                                    | `"image.repository":"image.tag"`                                       |
| `applicationNodes.plugins.resources`                             | Resources for plugins container                                                                                                                                                                                | `""`                                                                   |
| `applicationNodes.plugins.netrcCreds`                            | Name of the secret containing .netrc file to use creds when downloading plugins                                                                                                                                | `""`                                                                   |
| `applicationNodes.plugins.noCheckCertificate`                    | Flag to not check server's certificate when downloading plugins                                                                                                                                                | `false`                                                                |
| `applicationNodes.plugins.securityContext`                       | Security context for the container to download plugins                                                                                                                                                         | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `applicationNodes.jvmOpts`                                       | (DEPRECATED) Values to add to `SONAR_WEB_JAVAOPTS`. Please set directly `SONAR_WEB_JAVAOPTS` or `sonar.web.javaOpts`                                                                                           | `""`                                                                   |
| `applicationNodes.jvmCeOpts`                                     | (DEPRECATED) Values to add to `SONAR_CE_JAVAOPTS`. Please set directly `SONAR_CE_JAVAOPTS` or `sonar.ce.javaOpts`                                                                                              | `""`                                                                   |
| `applicationNodes.jwtSecret`                                     | A HS256 key encoded with base64 (_This value must be set before installing the chart, see [the documentation](#installing-the-chart)<!-- TODO: link the official docs page on the DCE JWT secret once one exists -->_) | `""`                                                                   |
| `applicationNodes.existingJwtSecret`                             | secret that contains the `jwtSecret`                                                                                                                                                                           | `nil`                                                                  |
| `applicationNodes.extraContainers`                               | Array of extra containers to run alongside                                                                                                                                                                     | `[]`                                                                   |
| `applicationNodes.extraInitContainers`                           | Array of extra init containers to run before the application container                                                                                                                                         | `[]`                                                                   |
| `applicationNodes.extraVolumes`                                  | Array of extra volumes to add to the SonarQube deployment                                                                                                                                                      | `[]`                                                                   |
| `applicationNodes.extraVolumeMounts`                             | Array of extra volume mounts to add to the SonarQube deployment                                                                                                                                                | `[]`                                                                   |
| `applicationNodes.hpa.enabled`                                   | Enable the HorizontalPodAutoscaler (HPA) for the app deployment                                                                                                                                                | `false`                                                                |
| `applicationNodes.hpa.minReplicas`                               | Minimum number of replicas for the HPA                                                                                                                                                                         | `2`                                                                    |
| `applicationNodes.hpa.maxReplicas`                               | Maximum number of replicas for the HPA                                                                                                                                                                         | `10`                                                                   |
| `applicationNodes.hpa.metrics`                                   | The metrics to use for scaling                                                                                                                                                                                 | see `values.yaml`                                                      |
| `applicationNodes.hpa.behavior`                                  | The scaling behavior                                                                                                                                                                                           | see `values.yaml`                                                      |
| `applicationNodes.nodeSelector`                                  | Node labels for application nodes' pods assignment, global nodeSelector takes precedence                                                                                                                       | `{}`                                                                   |
| `applicationNodes.affinity`                                      | Node / Pod affinities for applicationNodes, global affinity takes precedence                                                                                                                                   | `{}`                                                                   |
| `applicationNodes.tolerations`                                   | List of node taints to tolerate for applicationNodes, global tolerations take precedence                                                                                                                       | `[]`                                                                   |
| `applicationNodes.topologySpreadConstraints`                     | Topology spread constraints to apply to the application pods                                                                                                                                                    | `[]`                                                                   |
| `applicationNodes.port`                                   | The Hazelcast port for communication with each application member of the cluster.                                                                                                                       | `9003`                                                                   |
| `applicationNodes.webPort`                                   | The Hazelcast port for communication with the WebServer process. If not specified, a dynamic port will be chosen. **Required when `istio.enabled=true`** - see [Securing communication with TLS](#securing-communication-with-tls)                                                  | ``                                                                   |
| `applicationNodes.cePort`                                   | The Hazelcast port for communication with the ComputeEngine process. If not specified, a dynamic port will be chosen. **Required when `istio.enabled=true`** - see [Securing communication with TLS](#securing-communication-with-tls)                                               | ``                                                                   |

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
| `sonarWebContext`        | SonarQube web context, also serve as default value for `ingress.path`, `httproute` path, `account.sonarWebContext` and probes path. | ``      |
| `httpProxySecret`        | Should contain `http_proxy`, `https_proxy` and `no_proxy` keys, will superseed every other proxy variables            | ``      |
| `httpProxy`              | HTTP proxy for downloading JMX agent and install plugins, will superseed initContainer specific http proxy variables  | ``      |
| `httpsProxy`             | HTTPS proxy for downloading JMX agent and install plugins, will superseed initContainer specific https proxy variable | ``      |
| `noProxy`                | No proxy for downloading JMX agent and install plugins, will superseed initContainer specific no proxy variables      | ``      |
| `nodeEncryption.enabled` | Secure the communication between Application and Search nodes using TLS                                               | `false` |
| `sonarSecretKey`         | Name of existing secret used for settings encryption. When `agentOrchestrator.enabled`, this secret is also mounted into the orchestrator, with its path exposed via the `AGENTIC_SECRET_KEY_PATH` env var | `None`  |

### NetworkPolicies

| Parameter                                 | Description                                                               | Default |
| ----------------------------------------- | ------------------------------------------------------------------------- | ------- |
| `networkPolicy.enabled`                   | Create NetworkPolicies                                                    | `false` |
| `networkPolicy.prometheusNamespace`       | Allow incoming traffic to monitoring ports from this namespace            | `"monitoring"` |
| `networkPolicy.additionalNetworkPolicys`  | (DEPRECATED) Please use `networkPolicy.additionalNetworkPolicies` instead | `nil`   |
| `networkPolicy.additionalNetworkPolicies` | User defined NetworkPolicies (usefull for external database)              | `nil`   |

### OpenShift

| Parameter                        | Description                                                                                         | Default                    |
| -------------------------------- | --------------------------------------------------------------------------------------------------- | -------------------------- |
| `OpenShift.enabled`              | Define if this deployment is for OpenShift                                                          | `false`                    |
| `OpenShift.createSCC`            | (DEPRECATED) If this deployment is for OpenShift, define if SCC should be created for sonarqube pod | `false`                    |
| `OpenShift.agentRuntimeClassName` | `RuntimeClass` sandboxing the agent runtimes on OpenShift (from the sandboxed containers operator, never created by the chart). `kata-remote` for peer pods; `""` for no sandbox | `kata`                     |
| `OpenShift.skipAgentRuntimeClassCheck` | Skip the install-time check that `OpenShift.agentRuntimeClassName` exists in the cluster | `false`                    |
| `OpenShift.route.enabled`        | Flag to enable OpenShift Route                                                                      | `false`                    |
| `OpenShift.route.host`           | Host that points to the service                                                                     | `"sonarqube.your-org.com"` |
| `OpenShift.route.path`           | Path that the router watches for, to route traffic for to the service                               | `"/"`                      |
| `OpenShift.route.tls`            | TLS settings including termination type, certificates, insecure traffic, etc.                       | see `values.yaml`          |
| `OpenShift.route.wildcardPolicy` | The wildcard policy that is allowed where this route is exposed                                     | `None`                     |
| `OpenShift.route.annotations`    | Optional field to add extra annotations to the route                                                | `None`                     |
| `OpenShift.route.labels`         | Route additional labels                                                                             | `{}`                       |

### HttpRoute

| Parameter                    | Description                                                                                                   | Default |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------- | ------- |
| `httproute.enabled`          | Flag to enable GatewayAPI HttpRoute                                                                           | `False` |
| `httproute.gateway`          | Name of the gateway                                                                                           | `None`  |
| `httproute.gatewayNamespace` | (Optional) Name of the gateway namespace when located in a different namespace                                | `None`  |
| `httproute.hostnames`        | List of hostnames to match the HttpRoute against                                                              | `None`  |
| `httproute.labels`           | (Optional) List of extra labels to add to the HttpRoute                                                       | `None`  |
| `httproute.rules`            | (Optional) Extra Rules block of the HttpRoute. A default one is created with SonarWebContext and service port | `None`  |

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

| Parameter                      | Description                                                  | Default        |
| ------------------------------ | ------------------------------------------------------------ | -------------- |
| `ingress.enabled`              | Enable the built-in `Ingress` resource                        | `false`        |
| `ingress.labels`               | Ingress additional labels                                    | `{}`           |
| `ingress.hosts[0].name`        | Hostname to your SonarQube installation                      | `sonarqube.your-org.com` |
| `ingress.hosts[0].path`        | Path within the URL structure                                | `/`            |
| `ingress.hosts[0].serviceName` | Optional field to override the default serviceName of a path | `None`         |
| `ingress.hosts[0].servicePort` | Optional field to override the default servicePort of a path | `None`         |
| `ingress.tls`                  | Ingress secrets for TLS certificates                          | `[]`           |
| `ingress.ingressClassName`     | Ingress class name. This chart no longer bundles a controller, so set this to your own controller's class | `None` |
| `ingress.annotations`          | Field to add extra annotations to the ingress                | `{}`           |

### InitContainers

| Parameter                           | Description                                                                                                                           | Default                                                                |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `initContainers.image`              | Change init container image                                                                                                           | `applicationNodes.image`                                               |
| `initContainers.securityContext`    | SecurityContext for init containers                                                                                                   | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `initContainers.resources`          | Resources for init containers                                                                                                         | `{}`                                                                   |
| `extraInitContainers`               | **(DEPRECATED)** Use `searchNodes.extraInitContainers` or `applicationNodes.extraInitContainers` instead. | `{}`                                                                   |
| `caCerts.enabled`                   | Flag for enabling additional CA certificates                                                                                          | `false`                                                                |
| `caCerts.image`                     | Change init CA certificates container image                                                                                           | `applicationNodes.image`                                               |
| `caCerts.secret`                    | Name of the secret containing additional CA certificates. If defined, only secrets are going to be used.                              | `None`                                                                 |
| `caCerts.configMap.name`            | Name of the ConfigMap containing additional CA certificate(s). Ensure that `caCerts.secret` is not set if you want to use a `ConfigMap`. If `key`/`path` are omitted, every key in the ConfigMap is mounted and imported. | `None`                                                                 |
| `caCerts.configMap.key`             | Name of the single key to mount from the ConfigMap. Omit to mount all keys instead                                                    | `None`                                                                 |
| `caCerts.configMap.path`            | Filename that should be used for the given CA certificate when `key` is set                                                           | `None`                                                                 |
| `initSysctl.enabled`                | Modify k8s worker to conform to system requirements                                                                                   | `true`                                                                 |
| `initSysctl.vmMaxMapCount`          | Set init sysctl container vm.max_map_count                                                                                            | `524288`                                                               |
| `initSysctl.fsFileMax`              | Set init sysctl container fs.file-max                                                                                                 | `131072`                                                               |
| `initSysctl.nofile`                 | Set init sysctl container open file descriptors limit                                                                                 | `131072`                                                               |
| `initSysctl.nproc`                  | Set init sysctl container open threads limit                                                                                          | `8192`                                                                 |
| `initSysctl.image`                  | Change init sysctl container image                                                                                                    | `applicationNodes.image`                                               |
| `initSysctl.securityContext`        | InitSysctl container security context                                                                                                 | `{privileged: true}`                                                   |
| `initSysctl.resources`              | InitSysctl container resource requests & limits                                                                                       | `{}`                                                                   |
| `initFs.enabled`                    | Enable file permission change with init container                                                                                     | `true`                                                                 |
| `initFs.image`                      | InitFS container image                                                                                                                | `applicationNodes.image`                                               |
| `initFs.securityContext.privileged` | InitFS container needs to run privileged                                                                                              | `false`                                                                |

### SonarQube Specific

| Parameter                      | Description                                                                                                                              | Default          |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `sonarqubeFolder`              | (DEPRECATED) Directory name of SonarQube, Due to 1-1 mapping between helm version and docker version, there is no need for configuration | `/opt/sonarqube` |
| `monitoringPasscode`           | Value for sonar.web.systemPasscode needed for LivenessProbes                                                                             | `None`           |
| `monitoringPasscodeSecretName` | Name of the secret where to load `monitoringPasscode`                                                                                    | `None`           |
| `monitoringPasscodeSecretKey`  | Key of an existing secret containing `monitoringPasscode`                                                                                | `None`           |
| `extraContainers`              | Array of extra containers to run alongside the `sonarqube` container (aka. Sidecars)                                                     | `[]`             |

### JDBC Overwrite

| Parameter                                   | Description                                                                                                                                                   | Default                                    |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| `jdbcOverwrite.enable`                      | (DEPRECATED) Enable JDBC overwrites for external Databases. (It must be set to true)                          | `true`                                    |
| `jdbcOverwrite.enabled`                     | (DEPRECATED) Enable JDBC overwrites for external Databases. (It must be set to true)                                                                                  | `true`                                    |
| `jdbcOverwrite.jdbcUrl`                     | The JDBC url to connect the external DB; the placeholder default is rejected and must be replaced                                                                                                             | `jdbc:postgresql://myPostgres/myDatabase` |
| `jdbcOverwrite.jdbcUsername`                | The DB user that should be used for the JDBC connection                                                                                                       | `None`                                |
| `jdbcOverwrite.jdbcPassword`                | (DEPRECATED) The DB password that should be used for the JDBC connection, please use `jdbcOverwrite.jdbcSecretName` and `jdbcOverwrite.jdbcSecretPasswordKey` | `None`                                |
| `jdbcOverwrite.jdbcSecretName`              | Alternatively, use a pre-existing k8s secret containing the DB password                                                                                       | `None`                                     |
| `jdbcOverwrite.jdbcSecretPasswordKey`       | If the pre-existing k8s secret is used this allows the user to overwrite the 'key' of the password property in the secret                                     | `None`                                     |
| `jdbcOverwrite.oracleJdbcDriver.url`        | The URL of the Oracle JDBC driver to be downloaded                                                                                                            | `None`                                     |
| `jdbcOverwrite.oracleJdbcDriver.netrcCreds` | Name of the secret containing .netrc file to use creds when downloading the Oracle JDBC driver                                                                | `None`                                     |

### Tests

| Parameter                       | Description                                                   | Default                                                            |
| ------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------ |
| `tests.enabled`                 | Flag that allows tests to be excluded from the generated yaml | `true`                                                             |
| `tests.image`                   | Set the test container image                                  | `"applicationNodes.image.repository":"applicationNodes.image.tag"` |
| `tests.resources.limits.cpu`    | CPU limit for test container                                  | `500m`                                                             |
| `tests.resources.limits.memory` | Memory limit for test container                               | `200M`                                                             |

### ServiceAccount

| Parameter                       | Description                                                                                                                                                                                           | Default               |
| ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------- |
| `serviceAccount.create`         | If set to true, create a service account                                                                                                                                                              | `false`               |
| `serviceAccount.name`           | Name of the service account to create/use                                                                                                                                                             | `sonarqube-sonarqube` |
| `serviceAccount.automountToken` | Manage `automountServiceAccountToken` field for mounting service account credentials. Please note that this will set the default value used by SQ Pods, regardless of the service account being used. | `false`               |
| `serviceAccount.annotations`    | Additional service account annotations                                                                                                                                                                | `{}`                  |

### MCP (Model Context Protocol)

| Parameter                               | Description                                                                                              | Default                                                                |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `mcp.enabled`                           | Deploy the MCP server alongside SonarQube DCE                                                            | `false`                                                                |
| `mcp.image.repository`                  | MCP server image repository                                                                              | `sonarsource/sonarqube-mcp`                                            |
| `mcp.image.tag`                         | MCP server image tag                                                                                     | `"2026.5.0"` |
| `mcp.image.pullPolicy`                  | Image pull policy for the MCP container                                                                  | `IfNotPresent`                                                         |
| `mcp.port`                              | Port the MCP server listens on                                                                           | `8080`                                                                 |
| `mcp.healthCheckInterval`               | How often SonarQube checks MCP health (seconds). Sets `SONAR_MCP_HEALTHCHECKINTERVAL` on SonarQube.     | `""`                                                                   |
| `mcp.persistence.enabled`               | Enable persistent storage for MCP data (`/data`)                                                         | `true`                                                                 |
| `mcp.persistence.annotations`           | Annotations for the MCP PVC                                                                              | `{}`                                                                   |
| `mcp.persistence.existingClaim`         | Use an existing PVC instead of creating one                                                              | `""`                                                                   |
| `mcp.persistence.storageClass`          | Storage class for the MCP PVC                                                                            | `""`                                                                   |
| `mcp.persistence.accessMode`            | Access mode for the MCP PVC                                                                              | `ReadWriteOnce`                                                        |
| `mcp.persistence.size`                  | Size of the MCP PVC                                                                                      | `1Gi`                                                                  |
| `mcp.livenessProbe.initialDelaySeconds` | Initial delay before the liveness probe starts                                                           | `60`                                                                   |
| `mcp.livenessProbe.periodSeconds`       | How often the liveness probe runs                                                                        | `30`                                                                   |
| `mcp.livenessProbe.failureThreshold`    | Number of failures before the pod is restarted                                                           | `6`                                                                    |
| `mcp.livenessProbe.timeoutSeconds`      | Timeout for the liveness probe                                                                           | `1`                                                                    |
| `mcp.tls.enabled`                       | Enable TLS (HTTPS) between SonarQube and the MCP server                                                  | `false`                                                                |
| `mcp.tls.keystoreSecretName`            | Name of the Secret containing the keystore file                                                          | `""`                                                                   |
| `mcp.tls.keystoreSecretKey`             | Key inside the Secret that holds the keystore file                                                       | `keystore.p12`                                                         |
| `mcp.tls.passwordSecretName`            | Name of the Secret containing the keystore password                                                      | `""`                                                                   |
| `mcp.tls.passwordSecretKey`             | Key inside the password Secret                                                                           | `password`                                                             |
| `mcp.tls.keystoreType`                  | Keystore format (`PKCS12` or `JKS`)                                                                      | `PKCS12`                                                               |
| `mcp.podSecurityContext`                | Pod-level security context for the MCP pod. `fsGroup` makes `/data` group-writable (omitted on OpenShift) | `{fsGroup: 0}`                                                         |
| `mcp.containerSecurityContext`          | Security context for the MCP container                                                                   | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `mcp.initContainers`                    | Additional init containers for the MCP pod (e.g. to chown `/data` when the storage driver ignores `fsGroup`) | `[]`                                                                   |
| `mcp.env`                               | Additional environment variables for the MCP container                                                   | `[]`                                                                   |
| `mcp.resources`                         | CPU/memory resource requests and limits for the MCP container                                            | `{}`                                                                   |
| `mcp.annotations`                       | Annotations for the MCP pod                                                                              | `{}`                                                                   |
| `mcp.nodeSelector`                      | Node selector for the MCP pod                                                                            | `{}`                                                                   |
| `mcp.affinity`                          | Affinity rules for the MCP pod                                                                            | `{}`                                                                   |
| `mcp.tolerations`                       | Tolerations for the MCP pod                                                                              | `[]`                                                                   |
| `mcp.topologySpreadConstraints`         | Topology spread constraints for the MCP pod                                                              | `[]`                                                                   |

### Agents

See [Agentic features](#agentic-features) for how to enable and configure these components.

| Parameter                                       | Description                                                                                                     | Default                                                                |
| ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `vortexAnalysis.enabled`                        | Deploy Vortex and set `sonar.vortex.analysis.url` on the app nodes. When unset, defaults to `true` if `remediationAgent.enabled` is `true` | `null`                                                                 |
| `vortexAnalysis.image.repository`               | Vortex image repository (required when enabled)                                                         | `"sonarsource/sonar-vortex"`                                           |
| `vortexAnalysis.image.tag`                      | Vortex image tag | `"2026.5.0"` |
| `vortexAnalysis.image.pullPolicy`               | Vortex image pull policy                                                                                | `IfNotPresent`                                                         |
| `vortexAnalysis.image.pullSecret`               | imagePullSecret for the Vortex image                                                                    | `nil`                                                                  |
| `vortexAnalysis.image.pullSecrets`              | imagePullSecrets for the Vortex image                                                                   | `nil`                                                                  |
| `vortexAnalysis.port`                           | Port the container / Service serves HTTP on                                                                      | `8080`                                                                 |
| `vortexAnalysis.replicaCount`                   | Vortex replica count                                                                                    | `1`                                                                    |
| `vortexAnalysis.strategy`                       | Deployment update strategy. Must be `RollingUpdate` once `vortexAnalysis.autoscaling.enabled` is `true` | `{type: RollingUpdate}`                                       |
| `vortexAnalysis.autoscaling.enabled`             | Enable a KEDA `ScaledObject` reading Vortex's own `GET /metrics/max-concurrent-requests`. Requires the KEDA CRDs — see `agentKeda.assumeInstalled` below | `false`                                                    |
| `vortexAnalysis.autoscaling.minReplicas`         | Minimum replicas; validated `>= 2` (exactly `1` when `aggregateAcrossReplicas` is `false` - no scale-to-zero for this fleet) | `3`                                                     |
| `vortexAnalysis.autoscaling.maxReplicas`         | Maximum replicas                                                                                          | `20`                                                                   |
| `vortexAnalysis.autoscaling.pollingInterval`     | KEDA `pollingInterval` (seconds) for the `metrics-api` trigger; validated `<= windowSeconds`              | `15`                                                                   |
| `vortexAnalysis.autoscaling.scaleDownStabilizationSeconds` | KEDA-managed HPA `behavior.scaleDown.stabilizationWindowSeconds`; deliberately long to absorb a few failed polls | `900`                                                        |
| `vortexAnalysis.autoscaling.targetConcurrentRequests` | Max concurrent requests one replica should carry before the fleet scales out (per-replica, not a fleet total) | `2`                                                        |
| `vortexAnalysis.autoscaling.aggregateAcrossReplicas` | `true` sums the metric across every replica behind the Service (requires KEDA `>= 2.20.0`); `false` falls back to KEDA's single-random-replica probe | `true`                                        |
| `vortexAnalysis.autoscaling.windowSeconds`       | Sliding window Vortex reports its peak concurrency over (`METRICS_CONCURRENT_REQUESTS_WINDOW_SECONDS`); blank leaves the image's own default | `30`                                            |
| `vortexAnalysis.autoscaling.metricPath`          | Path of the metric endpoint on the Vortex container                                                      | `/metrics/max-concurrent-requests`                                     |
| `vortexAnalysis.autoscaling.manageReplicas`      | Set `false` to always omit `replicas` once autoscaling is enabled, even on the first apply — needed for GitOps tooling (Argo CD, Flux) | `true`                                                |
| `vortexAnalysis.terminationGracePeriodSeconds`   | Pod grace period; ships unconditionally, independent of autoscaling                                      | `180`                                                                  |
| `vortexAnalysis.storage.type`                   | Object storage backend for Vortex analysis context restoration (`SONAR_AGENTIC_STORAGE_TYPE`); `S3`, `FILESYSTEM`, `AZURE`, `GCS` or `NFS`; required | `S3`                                                       |
| `vortexAnalysis.storage.bucket`                 | Bucket holding Vortex analysis context items (`SONAR_AGENTIC_STORAGE_BUCKET`); required for an object-store type, ignored for `FILESYSTEM`/`NFS` | `""`                                                        |
| `vortexAnalysis.storage.region`                 | AWS region of the bucket (`SONAR_AGENTIC_STORAGE_REGION`); required for an object-store type, ignored for `FILESYSTEM`/`NFS` | `""`                                                        |
| `vortexAnalysis.storage.pathStyleAccess`        | Path-style S3 access (`SONAR_AGENTIC_STORAGE_PATH_STYLE_ACCESS`); `false` for AWS S3, `true` for most S3-compatible endpoints | `false`                                                     |
| `vortexAnalysis.storage.endpoint`               | Custom S3-compatible endpoint (`SONAR_AGENTIC_STORAGE_ENDPOINT`, e.g. MinIO); blank for real AWS S3   | `""`                                                                   |
| `vortexAnalysis.storage.filesystem.baseDir`     | Vortex's own directory (`SONAR_AGENTIC_STORAGE_FILESYSTEM_BASE_DIR`) when `type` is `FILESYSTEM`/`NFS` - a separate store from `agentOrchestrator.storage.filesystem.baseDir`, not required to match it; needs a matching `extraVolumeMounts` entry on this pod | `""`                     |
| `vortexAnalysis.storage.accessKey`              | Inline access key for `endpoint` above; ignored when `existingSecret` is set                                      | `""`                                                                   |
| `vortexAnalysis.storage.secretKey`              | Inline secret key for `endpoint` above; ignored when `existingSecret` is set                                      | `""`                                                                   |
| `vortexAnalysis.storage.existingSecret`         | Existing secret providing `SONAR_AGENTIC_STORAGE_ACCESS_KEY` / `SONAR_AGENTIC_STORAGE_SECRET_KEY`                 | `""`                                                                   |
| `vortexAnalysis.serviceAccount.create`          | Create a dedicated ServiceAccount for the Vortex pod, independent of the top-level `serviceAccount`      | `false`                                                                |
| `vortexAnalysis.serviceAccount.name`            | Name of that ServiceAccount; defaults to the Vortex fullname when `create` is true                       | `""`                                                                   |
| `vortexAnalysis.serviceAccount.automountToken`  | Automount the ServiceAccount token into the pod; needed for IRSA                                                  | `false`                                                                |
| `vortexAnalysis.serviceAccount.annotations`     | Annotations for that ServiceAccount (e.g. `eks.amazonaws.com/role-arn` for IRSA)                                  | `{}`                                                                   |
| `vortexAnalysis.containerSecurityContext`       | Security context for the Vortex container                                                               | [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `vortexAnalysis.env`                            | Additional environment variables for the Vortex container                                               | `[]`                                                                   |
| `vortexAnalysis.resources`                      | CPU/memory/ephemeral-storage resource requests and limits for the Vortex container        | requests `1` CPU / `6Gi` / `2Gi`, limits `2` CPU / `8Gi` / `2Gi`       |
| `vortexAnalysis.extraVolumes`                   | Extra volumes for the Vortex pod - e.g. the mount for `storage.filesystem.baseDir` above          | `[]`                                                                   |
| `vortexAnalysis.extraVolumeMounts`              | Extra volume mounts for the Vortex container - mount `readOnly: true`: Vortex only reads context, never writes it | `[]`                                                                   |
| `vortexAnalysis.nodeSelector`                   | Node labels for the Vortex pod                                                                           | `{}`                                                                   |
| `vortexAnalysis.affinity`                       | Affinity for the Vortex pod                                                                              | `{}`                                                                   |
| `vortexAnalysis.tolerations`                    | Tolerations for the Vortex pod                                                                           | `[]`                                                                   |
| `vortexAnalysis.topologySpreadConstraints`      | Topology spread constraints for the Vortex pod                                                          | `[]`                                                                   |
| `vortexAnalysis.annotations`                    | Annotations for the Vortex pod                                                                          | `{}`                                                                   |
| `agentOrchestrator.enabled`                                           | Deploy the Agent Orchestrator                                                                                                            | `false`                                                                        |
| `hunterAgent.enabled`                                            | Deploy the Hunter Agent (requires `agentOrchestrator.enabled=true`)                                                                           | `false`                                                                        |
| `remediationAgent.enabled`                                       | Deploy the Remediation Agent (requires `agentOrchestrator.enabled=true`; auto-enables `vortexAnalysis.enabled` unless it is explicitly `false`)                                    | `false`                                                                        |
| `hunterAgent.networkPolicy.enabled`                              | Render a NetworkPolicy for the Hunter Agent, independently of the top-level `networkPolicy.enabled`                                      | `true`                                                                        |
| `remediationAgent.networkPolicy.enabled`                         | Render a NetworkPolicy for the Remediation Agent, independently of the top-level `networkPolicy.enabled`                                 | `true`                                                                        |
| `gvisor.enabled`                                                 | Create the gVisor `RuntimeClass` and run the agent runtimes under it. Ignored when `OpenShift.enabled` is `true` (see `OpenShift.agentRuntimeClassName`) | `true`                                                                         |
| `gvisor.runtimeClassName`                                        | Name of the cluster-scoped `RuntimeClass`                                                                                               | `gvisor`                                                                       |
| `gvisor.handler`                                                 | containerd runtime handler the `RuntimeClass` targets                                                                                   | `runsc`                                                                        |
| `gvisor.nodeSelector`                                            | `RuntimeClass` scheduling selector, and the label the installer applies when ready (set `null` to disable pinning)                      | `{gvisor.enabled: "true"}`                                                     |
| `gvisor.installer.enabled`                                       | Deploy the privileged installer DaemonSet (requires `gvisor.enabled=true`)                                                              | `true`                                                                        |
| `gvisor.installer.image.repository`                              | Installer image repository                                                                                                              | `debian`                                                                       |
| `gvisor.installer.image.tag`                                     | Installer image tag (human-readable; ignored once `digest` is set)                                                                      | `stable-slim`                                                                  |
| `gvisor.installer.image.digest`                                  | Pinned installer image digest, for a reproducible pull; blank falls back to `repository:tag`                                            | pinned `debian:stable-slim` digest                                             |
| `gvisor.installer.image.pullPolicy`                              | Installer image pull policy                                                                                                              | `IfNotPresent`                                                                 |
| `gvisor.installer.runscVersion`                                  | Pinned gVisor release to install                                                                                                        | `"20260706"`                                                                   |
| `gvisor.installer.containerdConfigPath`                          | Path to the node's containerd config                                                                                                    | `/etc/containerd/config.toml`                                                  |
| `gvisor.installer.runscConfig.network`                           | `runsc` network mode written to `runsc.toml`                                                                                            | `host`                                                                         |
| `gvisor.installer.runscConfig.overlay2`                          | `runsc` overlay2 setting written to `runsc.toml`                                                                                        | `root:self,size=50g`                                                           |
| `gvisor.installer.resources`                                     | Installer container resource requests/limits                                                                                            | requests `100m` / `128Mi` / `50Mi`, limits `500m` / `256Mi` / `200Mi`          |
| `gvisor.installer.nodeSelector`                                  | Which nodes to install on (empty = all)                                                                                                 | `{}`                                                                           |
| `gvisor.installer.tolerations`                                   | Installer DaemonSet tolerations                                                                                                         | `[{operator: Exists}]`                                                         |
| `gvisor.installer.annotations`                                   | Installer pod annotations                                                                                                               | `{}`                                                                           |
| `agentRuntimeSandbox.enabled`                                    | Generic alternative to `gvisor.enabled` - treat the agent runtimes as sandboxed without needing `gvisor.*` (e.g. Kata Containers)       | `false`                                                                        |
| `agentRuntimeSandbox.runtimeClassName`                           | RuntimeClass to schedule the agent runtime pods onto; ignored when `gvisor.enabled=true`                                               | `""`                                                                           |
| `istio.enabled`                                                  | Put chart-owned workloads under STRICT mTLS, and let their sidecars reach istiod's control plane through the agent NetworkPolicies      | `false`                                                                        |
| `istio.namespace`                                                | Namespace istiod runs in                                                                                                                | `istio-system`                                                                |
| `istio.revision`                                                 | Revision of a revisioned (canary) control plane, e.g. `1-24-1`; makes the chart pin `istiod-<revision>` instead of `istiod` for agent runtime pods. Leave `""` for a non-revisioned install | `""`                                                       |
| `istio.istiodClusterIP`                                          | ClusterIP of the `istiod` Service, pinned into agent runtime pods via `hostAliases` so kube-dns egress can be dropped from their `NetworkPolicy`. `"auto"` reads the Service, an address is used as given, `""` opts out | `"auto"`                        |
| `istio.meshSidecar.enabled`                                      | Give the sandboxed agent runtimes a real mTLS identity via a hand-authored `istio-proxy` sidecar + `Sidecar` resource (requires `istio.enabled=true` and the runtime being sandboxed) | `false`                                             |
| `istio.meshSidecar.meshPort`                                     | Port the runtime pod's own Envoy binds for inbound mesh traffic; must differ from `hunterAgent.port`/`remediationAgent.port` and stay outside `15000`-`15100` | `18080`                                                       |
| `istio.meshSidecar.proxyImage.repository`                        | `istio-proxy` image repository                                                                                                          | `registry.istio.io/release/proxyv2`                                           |
| `istio.meshSidecar.proxyImage.tag`                                | `istio-proxy` image tag - the Istio version this pattern is validated against                                                          | `"1.30.4"`                                                                     |
| `istio.meshSidecar.resources`                                    | `istio-proxy` container resource requests/limits                                                                                        | requests `100m` / `128Mi`, limits `2` / `1Gi`                                  |
| `agentOrchestrator.serviceAccount.create`                             | Create a dedicated ServiceAccount for the orchestrator pod, independent of the top-level `serviceAccount`                                | `false`                                                                        |
| `agentOrchestrator.serviceAccount.name`                               | Name of that ServiceAccount; defaults to `<fullname>-agent-orchestrator` when `create` is true                                         | `""`                                                                           |
| `agentOrchestrator.serviceAccount.automountToken`                     | Automount the ServiceAccount token into the pod; needed for IRSA                                                                          | `false`                                                                        |
| `agentOrchestrator.serviceAccount.annotations`                        | Annotations for that ServiceAccount (e.g. an IRSA role ARN)                                                                              | `{}`                                                                           |
| `<hunterAgent\|remediationAgent>.serviceAccount.create`          | Create a dedicated ServiceAccount for this agent's pod. With `false`, the pod runs under the top-level `serviceAccount` and inherits its token and cloud role                                    | `true`                                                                       |
| `<hunterAgent\|remediationAgent>.serviceAccount.name`            | Name of that ServiceAccount; defaults to `<fullname>-agent-runtime-<family>` when `create` is true                                     | `""`                                                                           |
| `<hunterAgent\|remediationAgent>.serviceAccount.automountToken`  | Automount the ServiceAccount token into the pod; needed for IRSA                                                                          | `false`                                                                        |
| `<hunterAgent\|remediationAgent>.serviceAccount.annotations`     | Annotations for that ServiceAccount (e.g. an IRSA role ARN)                                                                              | `{}`                                                                           |
| `agentOrchestrator.image.repository`                                  | Agent Orchestrator image repository (required when enabled)                                                                              | `"sonarsource/sonarqube-agent-orchestrator"`                                   |
| `agentOrchestrator.image.tag`                                         | Agent Orchestrator image tag                                                                                                             | `"2026.5.0"` |
| `agentOrchestrator.image.pullPolicy`                                  | Agent Orchestrator image pull policy                                                                                                     | `IfNotPresent`                                                                 |
| `agentOrchestrator.image.pullSecrets`                                 | imagePullSecrets for the orchestrator image                                                                                              | `nil`                                                                          |
| `agentOrchestrator.port`                                              | Port the orchestrator listens on (also set as `SERVER_PORT`)                                                                             | `8080`                                                                         |
| `agentOrchestrator.replicaCount`                                      | Orchestrator replica count                                                                                                               | `1`                                                                            |
| `agentOrchestrator.revisionHistoryLimit`                              | How many old ReplicaSets to retain; blank uses the Kubernetes default (`10`)                                                              | `""`                                                                           |
| `agentOrchestrator.strategy`                                          | Deployment update strategy                                                           | `{type: RollingUpdate}`                                                                           |
| `agentOrchestrator.podLabels`                                         | Extra labels for the orchestrator pod                                                                                                     | `{}`                                                                           |
| `agentOrchestrator.annotations`                                       | Extra annotations for the orchestrator pod                                                                                                | `{}`                                                                           |
| `agentOrchestrator.nodeSelector`                                      | Orchestrator nodeSelector                                                                                                                 | `{}`                                                                           |
| `agentOrchestrator.tolerations`                                       | Orchestrator tolerations                                                                                                                  | `[]`                                                                           |
| `agentOrchestrator.affinity`                                          | Orchestrator affinity                                                                                                                     | `{}`                                                                           |
| `agentOrchestrator.topologySpreadConstraints`                         | Orchestrator topology spread constraints                                                                                                  | `[]`                                                                           |
| `agentOrchestrator.resources`                                         | Orchestrator container resources (cpu / memory / ephemeral-storage)                                                                       | requests `250m` / `512Mi` / `512Mi`, limits `1` / `1Gi` / `2Gi`                |
| `agentOrchestrator.securityContext`                                   | Orchestrator pod security context                                                                                                        | `{}`                                                                           |
| `agentOrchestrator.containerSecurityContext`                          | Orchestrator container security context, incl. `readOnlyRootFilesystem`                                                                  | [Restricted podSecurityStandard](#kubernetes---pod-security-standards)        |
| `agentOrchestrator.scheduler.enabled`                                 | Enable the orchestrator's internal scheduler; injects `SONAR_AGENTIC_ORCHESTRATOR_SCHEDULER_ENABLED=true`                                | `false`                                                                       |
| `agentOrchestrator.probes.readiness`/`.liveness`                      | Orchestrator readiness/liveness probes (`enabled`, `path`, `initialDelaySeconds`, `periodSeconds`, `timeoutSeconds`, `successThreshold`, `failureThreshold`) | `enabled: true`, `/readyz`/`/livez`, see `values.yaml`          |
| `agentOrchestrator.terminationGracePeriodSeconds`                     | Orchestrator pod grace period; ships unconditionally. Must comfortably exceed both shutdown phases of the orchestrator image's `spring.lifecycle.timeout-per-shutdown-phase` (`1260`s each)                          | `2580`                                                                         |
| `agentOrchestrator.coreDb.endpoint`                                   | CORE DB `host:port` (`CORE_DB_READ_WRITE_ENDPOINT`); blank = derived from `jdbcOverwrite.jdbcUrl`                                        | `""`                                                                           |
| `agentOrchestrator.coreDb.name`                                       | CORE DB name (`CORE_DB_NAME`); blank = derived from `jdbcOverwrite.jdbcUrl`                                                              | `""`                                                                           |
| `agentOrchestrator.coreDb.username`                                   | CORE DB user (`CORE_DB_USERNAME`); blank = `jdbcOverwrite.jdbcUsername`                                                                  | `""`                                                                           |
| `agentOrchestrator.coreDb.passwordSecretName`                         | Secret holding the CORE DB password (`CORE_DB_PASSWORD`); blank = the `jdbcOverwrite`-derived secret. Point at a raw-password secret if SonarQube's is encrypted/unavailable | `""`                                                                           |
| `agentOrchestrator.coreDb.passwordSecretKey`                          | Key within `coreDb.passwordSecretName`; blank = the `jdbcOverwrite`-derived key                                                          | `""`                                                                           |
| `agentOrchestrator.storage.type`                                      | Storage backend (`SONAR_AGENTIC_ORCHESTRATOR_STORAGE_TYPE`): `S3`, `FILESYSTEM` or `NFS`                                                 | `S3`                                                                           |
| `agentOrchestrator.storage.endpoint`                                  | Object-storage endpoint; blank = real AWS S3 regional endpoint                                                                           | `""`                                                                           |
| `agentOrchestrator.storage.region`                                    | Object-storage region                                                                                                                    | `us-east-1`                                                                    |
| `agentOrchestrator.storage.bucket`                                    | Object-storage bucket for the shared job artifacts (required for an object-store type, ignored for `FILESYSTEM`/`NFS`)                    | `agent-jobs`                                                                 |
| `agentOrchestrator.storage.pathStyle`                                 | Path-style addressing (`true` for MinIO, `false` for AWS S3)                                                                             | `true`                                                                         |
| `agentOrchestrator.storage.accessKey`                                 | Inline S3 access key; leave blank for credential-less access (pod IAM role / IRSA)                                                       | `""`                                                                           |
| `agentOrchestrator.storage.secretKey`                                 | Inline S3 secret key; leave blank for credential-less access                                                                             | `""`                                                                           |
| `agentOrchestrator.storage.existingSecret`                            | Existing secret providing `AGENTIC_STORAGE_ACCESS_KEY` / `AGENTIC_STORAGE_SECRET_KEY`                                                     | `""`                                                                           |
| `agentOrchestrator.storage.filesystem.baseDir`                        | Shared agent-jobs directory (`SONAR_AGENTIC_ORCHESTRATOR_STORAGE_FILESYSTEM_BASE_DIR`) when `type` is `FILESYSTEM`/`NFS`, mounted at its root here; each of `hunterAgent`/`remediationAgent` mounts only its own `subPath` at the matching `extraVolumeMounts` entry on this pod. Not related to `vortexAnalysis.storage.filesystem.baseDir`, which is Vortex's own, separate store | `""`             |
| `agentOrchestrator.github.apiBaseUrl`                                 | `AGENTIC_GITHUB_API_BASE_URL`; blank = the orchestrator's default (real api.github.com)                                                  | `""`                                                                           |
| `agentOrchestrator.env`                                               | Extra env vars for the orchestrator container, appended after the chart-set env                                                          | `[]`                                                                           |
| `agentOrchestrator.extraVolumes`                                      | Extra volumes for the orchestrator pod - e.g. the shared agent-jobs volume (mounted at its root here; `hunterAgent`/`remediationAgent` each mount only their own `subPath` - see `storage.filesystem.baseDir` above) | `[]`                    |
| `agentOrchestrator.extraVolumeMounts`                                 | Extra volume mounts for the orchestrator container                                                                                        | `[]`                                                                           |
| `agentOrchestrator.autoscaling.enabled`                               | Enable a plain CPU/Memory `HorizontalPodAutoscaler` for the orchestrator                                                                  | `false`                                                                        |
| `agentOrchestrator.autoscaling.minReplicas`                           | Minimum replicas; validated `>= 2`                                                                                                        | `2`                                                                            |
| `agentOrchestrator.autoscaling.maxReplicas`                           | Maximum replicas                                                                                                                          | `5`                                                                            |
| `agentOrchestrator.autoscaling.metrics`                               | Raw `autoscaling/v2` HPA `metrics:` pass-through                                                                                          | CPU `averageUtilization: 75`                                                   |
| `agentOrchestrator.autoscaling.behavior`                              | Raw `autoscaling/v2` HPA `behavior:` pass-through                                                                                         | `scaleDown.stabilizationWindowSeconds: 300`                                    |
| `agentOrchestrator.autoscaling.manageReplicas`                        | Set `false` to always omit `replicas` once autoscaling is enabled, even on the first apply — needed for GitOps tooling (Argo CD, Flux)         | `true`                                                                         |
| `<hunterAgent\|remediationAgent>.image.repository`               | Agent image repository (required when the agent is enabled)                                                                              | `"sonarsource/sonarqube-hunter-agent"` / `"sonarsource/sonarqube-remediation-agent"` |
| `<hunterAgent\|remediationAgent>.image.tag`                      | Agent image tag                                                                                                                          | `"2026.5.0"` |
| `<hunterAgent\|remediationAgent>.image.pullPolicy`               | Agent image pull policy                                                                                                                  | `IfNotPresent`                                                                 |
| `<hunterAgent\|remediationAgent>.port`                           | Agent container / Service port                                                                                                           | `8090`                                                                         |
| `<hunterAgent\|remediationAgent>.replicaCount`                   | Agent replica count (`>1` enables L4 429 re-routing across replicas)                                                                     | `1`                                                                            |
| `<hunterAgent\|remediationAgent>.revisionHistoryLimit`           | How many old ReplicaSets to retain; blank uses the Kubernetes default (`10`)                                                              | `""`                                                                           |
| `<hunterAgent\|remediationAgent>.strategy`                       | Deployment update strategy                                                           | `{type: RollingUpdate}`                                                                           |
| `<hunterAgent\|remediationAgent>.podLabels`                      | Extra labels for this runtime's pod                                                                                                       | `{}`                                                                           |
| `<hunterAgent\|remediationAgent>.annotations`                    | Extra annotations for this runtime's pod                                                                                                  | `{}`                                                                           |
| `<hunterAgent\|remediationAgent>.nodeSelector`                   | This runtime's nodeSelector                                                                                                                | `{}`                                                                           |
| `<hunterAgent\|remediationAgent>.tolerations`                    | This runtime's tolerations                                                                                                                 | `[]`                                                                           |
| `<hunterAgent\|remediationAgent>.affinity`                       | This runtime's affinity                                                                                                                    | `{}`                                                                           |
| `<hunterAgent\|remediationAgent>.topologySpreadConstraints`      | This runtime's topology spread constraints                                                                                                | `[]`                                                                           |
| `<hunterAgent\|remediationAgent>.resources`                      | Agent container resources (both runtimes ship sized defaults)                                                                 | see [CPU and memory](#cpu-and-memory-settings)                                                                      |
| `<hunterAgent\|remediationAgent>.securityContext`                | Agent pod security context                                                                                                               | `{}`                                                                           |
| `<hunterAgent\|remediationAgent>.containerSecurityContext`       | Agent container security context. No `readOnlyRootFilesystem`: the agent images need their home directory writable | [Restricted podSecurityStandard](#kubernetes---pod-security-standards)        |
| `<hunterAgent\|remediationAgent>.probes.readiness`/`.liveness`   | This runtime's readiness/liveness probes (`enabled`, `path`, `initialDelaySeconds`, `periodSeconds`, `timeoutSeconds`, `successThreshold`, `failureThreshold`) | `enabled: true`, `/readyz`/`/livez`, see `values.yaml` |
| `<hunterAgent\|remediationAgent>.storage.type`                   | This runtime's own storage target type (`S3`, `FILESYSTEM` or `NFS`); blank defers to `agentOrchestrator.storage`. Not yet read by any container - see `storage.filesystem.baseDir` below | `""`                            |
| `<hunterAgent\|remediationAgent>.storage.bucket`                 | This runtime's own bucket, if `type` is an object store and different from `agentOrchestrator.storage.bucket`                            | `""`                                                                           |
| `<hunterAgent\|remediationAgent>.storage.region`                 | This runtime's own region, paired with `bucket` above                                                                                     | `""`                                                                           |
| `<hunterAgent\|remediationAgent>.storage.endpoint`               | This runtime's own S3-compatible endpoint, paired with `bucket` above                                                                     | `""`                                                                           |
| `<hunterAgent\|remediationAgent>.storage.pathStyle`               | Path-style addressing for `endpoint` above; omitted, not defaulted, so unset stays distinguishable from an explicit override  | unset (defers to `agentOrchestrator.storage.pathStyle`)                        |
| `<hunterAgent\|remediationAgent>.storage.filesystem.baseDir`     | This runtime's own absolute mount path for a `FILESYSTEM`/`NFS` backend, e.g. `/agentic-storage/hunter`. When set, `validation.yaml` requires a matching `extraVolumeMounts` entry (see below) | `""`     |
| `<hunterAgent\|remediationAgent>.storage.prefix`                 | Key prefix within the target above (e.g. `hunter/`)                                                                                       | `""`                                                                           |
| `<hunterAgent\|remediationAgent>.storage.accessKey`              | Inline access key for `endpoint` above; ignored when `existingSecret` is set                                                              | `""`                                                                           |
| `<hunterAgent\|remediationAgent>.storage.secretKey`              | Inline secret key for `endpoint` above; ignored when `existingSecret` is set                                                              | `""`                                                                           |
| `<hunterAgent\|remediationAgent>.storage.existingSecret`         | Existing secret providing the access/secret key pair above                                                                                | `""`                                                                           |
| `<hunterAgent\|remediationAgent>.env`                            | Extra env vars for the agent container                                                                                                   | `[]`                                                                           |
| `<hunterAgent\|remediationAgent>.extraVolumes`                   | Extra volumes for the agent pod - e.g. this runtime's `subPath` mount of a shared FILESYSTEM/NFS agent-jobs volume | `[]`     |
| `<hunterAgent\|remediationAgent>.extraVolumeMounts`              | Extra volume mounts for the agent container - same use case as `extraVolumes` above                                                     | `[]`                                                                           |
| `<hunterAgent\|remediationAgent>.terminationGracePeriodSeconds`  | Pod grace period; ships unconditionally. Must comfortably exceed that runtime image's fixed `uvicorn --timeout-graceful-shutdown` (`44400`s hunter / `3600`s remediation) | `44430` / `3630`                                                              |
| `<hunterAgent\|remediationAgent>.autoscaling.enabled`            | Enable a KEDA `ScaledObject` for this runtime family. Requires the KEDA CRDs — see `agentKeda.assumeInstalled` below                       | `false`                                                                        |
| `<hunterAgent\|remediationAgent>.autoscaling.minReplicas`        | Minimum replicas; validated `>= 2` (no scale-to-zero for this fleet)                                                                       | `2`                                                                            |
| `<hunterAgent\|remediationAgent>.autoscaling.maxReplicas`        | Maximum replicas                                                                                                                          | `25`                                                                           |
| `<hunterAgent\|remediationAgent>.autoscaling.pollingInterval`    | KEDA `pollingInterval` (seconds) for the `metrics-api` trigger against the orchestrator's `GET /metrics/queue`                             | `15`                                                                           |
| `<hunterAgent\|remediationAgent>.autoscaling.scaleDownStabilizationSeconds` | KEDA-managed HPA `behavior.scaleDown.stabilizationWindowSeconds`                                                                | `600`                                                                          |
| `<hunterAgent\|remediationAgent>.autoscaling.manageReplicas`     | Set `false` to always omit `replicas` once autoscaling is enabled, even on the first apply — needed for GitOps tooling (Argo CD, Flux)         | `true`                                                                         |
| `agentKeda.assumeInstalled`                                      | Override the `.Capabilities.APIVersions`-based KEDA CRD detection (`true`/`false`); needed under `helm template`/`--dry-run=client`, where `Capabilities` is always empty | `nil` (auto-detect)                                    |
| `keda.enabled`                                                   | Install the [KEDA](https://keda.sh) operator as a dependency of this release. Any other key under `keda` is passed through to the upstream `keda` chart's own values | `false`                                                |
| `hunterAgent.scriptPath`                                         | Detection-agent entrypoint exposed as `SCRIPT_PATH`, which switches the runtime into hunter/detection mode; `""` omits the env var                        | `/home/agent/app/.venv/bin/detection-agent`                                    |
| `hunterAgent.playbookKey`                                        | Detection playbook to load, exposed as `PLAYBOOK_KEY`; `""` omits the env var                                                                              | `appsec`                                                                       |
| `hunterAgent.playbookVersion`                                    | Version of the above playbook to load, exposed as `PLAYBOOK_VERSION`; `""` omits the env var                                                              | `stable`                                                                       |
| `hunterAgent.resources`                                          | Resources for the SonarQube Hunter agent runtime, sized for repo checkout + scan; very large repos may need `4` CPU / `20Gi`                                | requests `1` CPU / `8Gi` / `15Gi`, limits `2` CPU / `8Gi` / `15Gi`            |
| `remediationAgent.remediationScriptPath`                         | Remediation-agent entrypoint exposed as `REMEDIATION_SCRIPT_PATH` (`REMEDIATION_RULE_INFO_ENDPOINT`/`REMEDIATION_ANALYSIS_ENDPOINT` are derived from the SonarQube Server URL); `""` omits the env var | `/home/agent/app/.venv/lib/python3.13/site-packages/remediation_agent/main.py` |
| `remediationAgent.resources`                                     | Resources for the SonarQube Remediation Agent (SQRA) runtime, sized for repo clone + agent/LLM loop                                      | requests `1` CPU / `2Gi` / `10Gi`, limits `4` CPU / `8Gi` / `50Gi`             |
| `agentEgressProxy.image.repository`                              | Squid image repository                                                                                                                   | `ubuntu/squid`                                                                 |
| `agentEgressProxy.image.tag`                                     | Squid image tag                                                                                                                          | `6.6-24.04_edge`                                                               |
| `agentEgressProxy.image.digest`                                  | Pinned Squid image digest; wins over `tag`, blank falls back to `repository:tag`. Pinned by default — `_edge` is a floating channel that is rebuilt in place, so the tag alone is not reproducible | see `values.yaml`                        |
| `agentEgressProxy.image.pullPolicy`                              | Squid image pull policy                                                                                                                  | `IfNotPresent`                                                                 |
| `agentEgressProxy.image.pullSecret`/`.pullSecrets`                | imagePullSecret(s) for the Squid image                                                                                                   | `""` / `[]`                                                                    |
| `agentEgressProxy.port`                                          | Port Squid listens on for every family except Remediation, baked into each such runtime's `HTTP_PROXY`/`HTTPS_PROXY`                     | `3128`                                                                         |
| `agentEgressProxy.remediationPort`                               | Separate port Squid listens on for the Remediation runtime, the only listener whose ACL admits SonarQube's own agentic endpoints         | `3129`                                                                         |
| `agentEgressProxy.replicaCount`                                  | Proxy replica count                                                                                                                       | `2`                                                                            |
| `agentEgressProxy.revisionHistoryLimit`                          | How many old ReplicaSets to retain; blank uses the Kubernetes default (`10`)                                                              | `""`                                                                           |
| `agentEgressProxy.strategy`                                      | Deployment update strategy; blank uses the Kubernetes default (`RollingUpdate`)                                                           | `{}`                                                                           |
| `agentEgressProxy.podLabels`/`.annotations`                      | Extra labels/annotations for the proxy pod                                                                                               | `{}`                                                                           |
| `agentEgressProxy.nodeSelector`/`.tolerations`/`.affinity`       | Proxy pod scheduling. With `affinity` empty the chart applies a soft (preferred) `podAntiAffinity` on `kubernetes.io/hostname` so replicas spread across nodes; setting it replaces that default outright | `{}` / `[]` / `{}`      |
| `agentEgressProxy.resources`                                     | Proxy container resources                                                                                                                 | see `values.yaml`                                                              |
| `agentEgressProxy.maxFiledescriptors`                            | Squid `max_filedescriptors`. Squid sizes its per-descriptor tables from `RLIMIT_NOFILE` at startup, so an unpinned value costs ~170 MB on the 1048576-descriptor limit container runtimes hand out and the pod is OOMKilled before serving a request. Two descriptors per proxied request; raise this and `resources.limits.memory` together | `4096`  |
| `agentEgressProxy.securityContext`/`.containerSecurityContext`   | Proxy pod/container security context                                                                                                     | container: [Restricted podSecurityStandard](#kubernetes---pod-security-standards) |
| `agentEgressProxy.probes.readiness`/`.liveness`                  | Proxy `tcpSocket` readiness/liveness probes (`periodSeconds`, `timeoutSeconds`) — Squid has no HTTP health endpoint                       | see `values.yaml`                                                              |
| `agentEgressProxy.serviceAccount.create`/`.name`/`.automountToken`/`.annotations` | Dedicated ServiceAccount for the proxy pod, independent of the top-level `serviceAccount`                               | `false` / `""` / `false` / `{}`                                                |
| `agentEgressProxy.allowedDomains`                                | Domains runtimes may reach through the proxy (Squid `dstdomain` ACL); a leading dot allows the domain and all subdomains, so don't also list a domain it covers (the render fails). Empty means runtimes can reach *only* SonarQube's own agentic endpoints — no LLM provider is callable | `[]`                    |
| `agentEgressProxy.extraSquidConf`                                | Raw `squid.conf` lines spliced in after the generated allow rules but before the final `http_access deny all`, so an `http_access allow …` added here takes effect. Also the way to add `Safe_ports`/`SSL_ports` entries. Unvalidated — a syntax error stops Squid and takes out egress for every runtime | `""`     |
| `agentEgressProxy.networkPolicy.enabled`                         | Render a NetworkPolicy for the proxy, independently of the top-level `networkPolicy.enabled`                                             | `true`                                                                        |
| `agentEgressProxy.networkPolicy.egressPorts`                     | Ports the proxy's own NetworkPolicy allows egress to on `0.0.0.0/0` (`NetworkPolicy` can't match FQDNs — the real restriction is Squid's `allowedDomains`) | `[80, 443]`                                                    |
| `agentEgressProxy.networkPolicy.egressExcludeCidrs`              | CIDRs subtracted (`ipBlock.except`) from that `0.0.0.0/0` egress rule. Defaults to the link-local range carrying the cloud metadata service, so a mistaken `allowedDomains` entry can't turn the proxy into an SSRF pivot | `[169.254.0.0/16]`   |
| `agentEgressProxy.podDisruptionBudget.minAvailable`              | `PodDisruptionBudget` `minAvailable` for the proxy                                                                                        | `1`                                                                            |
| `agentEgressProxy.env`                                           | Extra env vars for the proxy container                                                                                                   | `[]`                                                                           |
| `agentEgressProxy.extraVolumes`/`.extraVolumeMounts`             | Extra volumes/mounts for the proxy pod/container                                                                                         | `[]`                                                                           |
| `agenticSigningSecret.existingSecret`                            | Name of an existing Secret holding the operator-provided agentic instance secret every signing key is derived from. Not chart-managed — create it out of band. **Required** whenever `hunterAgent`/`remediationAgent`/`vortexAnalysis` is enabled | unset                        |
| `agenticSigningSecret.key`                                       | Key within that Secret holding the instance secret                                                                                       | `instance-secret`                                                              |
| `agentKeyDerivation.enabled`                                     | Run the `pre-install`/`pre-upgrade` hook Job that derives the per-consumer key Secrets. Set `false` only if you provide the `<fullname>-agentic-keys-*` Secrets yourself | `true`                                       |
| `agentKeyDerivation.image.repository`                            | Image carrying `derive-keys.sh`. Blank makes the whole `agentKeyDerivation.image` block fall back to `agentOrchestrator.image`; setting it opts out of that fallback entirely | `""`                       |
| `agentKeyDerivation.image.tag`                                   | Tag for that image (only consulted once `repository` is set)                                                                             | `""`                                                                           |
| `agentKeyDerivation.image.pullPolicy`                            | Pull policy for the hook Job's image                                                                                                     | `IfNotPresent`                                                                 |
| `agentKeyDerivation.image.pullSecrets`                           | imagePullSecrets for the hook Job's image                                                                                                | `nil`                                                                          |
| `agentKeyDerivation.serviceAccount.create`                       | Create a dedicated ServiceAccount for the hook Job; `false` reuses the top-level `serviceAccount`. The Role granting `create`/`update` on `secrets` is bound either way | `true`                        |
| `agentKeyDerivation.serviceAccount.name`                         | Name of the ServiceAccount the hook Job runs as; defaults to `<fullname>-agent-key-derivation` when `create` is true, and to the top-level `serviceAccount` when it is false                               | `""`                                                                           |
| `agentKeyDerivation.serviceAccount.annotations`                  | Annotations for that ServiceAccount                                                                                                      | `{}`                                                                           |
| `agentKeyDerivation.nodeSelector`                                | Hook Job pod nodeSelector                                                                                                                 | `{}`                                                                           |
| `agentKeyDerivation.tolerations`                                 | Hook Job pod tolerations                                                                                                                  | `[]`                                                                           |
| `agentKeyDerivation.affinity`                                    | Hook Job pod affinity                                                                                                                     | `{}`                                                                           |
| `agentKeyDerivation.topologySpreadConstraints`                   | Hook Job pod topology spread constraints                                                                                                  | `[]`                                                                           |
| `agentKeyDerivation.resources`                                   | Hook Job container resources                                                                                                             | requests `50m`/`64Mi`, limits `250m`/`128Mi`                                   |
| `agentKeyDerivation.securityContext`                             | Hook Job pod security context                                                                                                            | `{}`                                                                           |
| `agentKeyDerivation.containerSecurityContext`                    | Hook Job container security context                                                                                                      | non-root `900:900`, no privilege escalation, all capabilities dropped, read-only root filesystem |
| `agentKeyDerivation.activeDeadlineSeconds`                       | Fail the hook Job (and the release) if key derivation and the Secret writes don't finish within this window, instead of hanging the install | `300`                                                                        |

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
| `logging.jsonOutput`                | (DEPRECATED) Enable/Disable logging in JSON format. Deprecated in favor of the ENV var SONAR_LOG_JSONOUTPUT or the `sonar.properties`'s `sonar.log.jsonOutput`                 | `false`                                                                |
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
| `terminationGracePeriodSeconds`     | (DEPRECATED) this field is not used in the templates                                                                                                                           | `60`                                                                   |

You can also configure values for the PostgreSQL database via the PostgreSQL [Chart](https://hub.helm.sh/charts/bitnami/postgresql)

For overriding variables see: [Customizing the chart](https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing)

## License

SonarQube Server Data Center Edition is licensed under [SonarQube Server Terms and Conditions](https://www.sonarsource.com/legal/sonarqube/terms-and-conditions/).
