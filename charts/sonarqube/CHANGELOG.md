# SonarQube Chart Changelog
All changes to this chart will be documented in this file.

## [2025.3.1]
* Update Chart's version to 2025.3.1
* Upgrade SonarQube Server to 2025.3.1

## [2025.3.0]
* Update Chart's version to 2025.3.0
* Upgrade SonarQube Community Build to 25.5.0.107428
* Normalizes the extension for all templates
* Remove example about non-system sonar.properties
* Fix change-admin-password hook when using special characters
* Upgrade SonarQube Server to 2025.3.0

## [2025.2.0]
* Update Chart's version to 2025.2.0
* Update ingress-nginx subchart to 4.12.1
* Upgrade SonarQube Server to 2025.2.0

## [2025.1.0]
* Update Chart's version to 2025.1.0
* Upgrade SonarQube Server to 2025.1.0
* Upgrade SonarQube Community Build to 25.1.0.102122
* Update ingress-nginx subchart to 4.11.3
* Support Kubernetes v1.32
* Remove the default passcode provided with `monitoringPasscode`
* Support Openshift v4.17
* Improves editions and versions setting for sonarqube chart

## [10.8.1]
* Update Chart's version to 10.8.1
* Remove immutable labels selector `app.kubernetes.io/name` and `app.kubernetes.io/version` as it breaks upgrades
* set `image.tag` empty in default value file, `image.tag` is dynamically set according to the `edition` and `community` fields. user-defined have precedence

## [10.8.0]
* Update Chart's version to 10.8.0
* Upgrade SonarQube Server to 10.8.0
* Release SonarQube Community Build 24.12
* Support the installation of the Oracle JDBC Driver
* Support Kubernetes v1.31
* Deprecate the `community` value for the `edition` parameter
* Introduce the `community.enabled` and `community.buildNumber` parameters for SonarQube Community Build
* Deprecate the default value of `image.tag` in favor of an empty string
* Update the Chart's icon with the SonarQube Server logo
* Set `app.kubernetes.io/name` and `app.kubernetes.io/version` as selector labels
* Support Gateway on different namespace in HTTPRoute
* Change `ingress.ingressClassName` default, set it to `nginx` if `nginx.enabled` or `ingress-nginx.enabled`
* Ensure that ConfigMap resources are not created for `initFS` and `initSysctl` if not needed
* Ensure the Pod will stop at `init` stage if init_sysctl.sh failed to modify kernel parameters
* Replace the example images in initContainers, initSysctl and initFs from `busybox:1.36` to `ubuntu:24.04`, which are commented out by default
* Make the `automountServiceAccountToken` configurable with `serviceAccount.automountToken` in PodSpec
* Deprecate `sonarqubeFolder`, `jdbcOverwrite.jdbcPassword` and `terminationGracePeriodSeconds`
* Deprecate `deploymentStrategy.type`, which will be set to `Recreate`
* Deprecate `account`, `curlContainerImage`, `adminJobAnnotations`
* Deprecate the StatefulSet deployment type

## [10.7.0]
* Update Chart's version to 10.7.0
* Upgrade SonarQube to 10.7.0
* Support Kubernetes v1.30
* Upgrade ingress-nginx dependency to 4.10.1
* Deprecate `jdbcOverwrite.enable` in favor of `jdbcOverwrite.enabled`
* Fix regression on env valuesFrom in the new STS template
* Fix a typo in the new common STS template
* Enable the setup of ReadOnlyRootFilesystem in the security contexts
* Support basic chart installation on Openshift
* Include remaining Route settings
* Fix networkPolicy.additionalPolicys typo
* Support install-plugin and prometheusExporter proxy variables in secret
* Support GatewayAPI HttpRoute
* Support additional labels in the PodMonitor
* Support Openshift SCCv2 by default when Openshift.enabled=true
* Deprecate Openshift.createSCC
* Support additional CA Certificate as ConfigMap instead of Secret only
* Changed default value for caCerts.image
* Fix openshift change-admin-password-hook Job SecurityContext failure
* Support SONAR_OPENSHIFT telemetry env_var
* Update helm chart repo path in sources
* Changed SONAR_OPENSHIFT to IS_HELM_OPENSHIFT_ENABLED
* Remove socketTimeout from jdbcOverwrite.jdbcUrl's default value
* Refactor Route to be subparameter of OpenShift
* Make OpenShift.createSCC false by default
* Deprecate peristence.volumes and persistence.mounts in favor or extraVolumes and extraVolumeMounts
* Ensure kubernetes.io/version label is smaller than 63 chars

## [10.6.0]
* Update SonarQube to 10.6.0
* Update Chart's version to 10.6.0
* Fix the env-var templating when sourcing from secrets
* Fix the postgresql chart's repository link
* Add support for overriding liveness/readiness probe logic
* Use a common template for Deployment and StatefulSet

## [10.5.0]
* Upgrade SonarQube to 10.5.0
* Update Chart's version to 10.5.0
* Update nginx-ingress-controller dependency to version 4.9.1
* Set `automountServiceAccountToken` to false in pod's specifications
* Update default `resources` values matching better default Xmx and Xms of the SonarQube processes.
* Make `ephemeral-storage` resource's limits and requests configurable for the SonarQube container
* Set memory and cpu limits for the test container
* Deprecate nginx.enabled in favor of ingress-nginx.enabled, to match with subchart config block
* Deprecate `prometheusMonitoring.podMonitor.namespace`
* Instantiate `monitoring-web` and `monitoring-ce` endpoints when the `prometheusExporter` is enabled
* Take `sonarWebContext` into account for the `PodMonitor` path
* Fix duplicated env_var in Pods causing deployment issue (`SONAR_WEB_CONTEXT`,`SONAR_WEB_JAVAOPTS`,`SONAR_CE_JAVAOPTS`)

## [10.4.0]
* Upgrade SonarQube to 10.4.0
* Update Chart's version to 10.4.0
* Improve the description of deprecated `jvmOpts` and `jvmCeOpts` values
* Run the initSysctl init-container as root to prevent 'permission denied' issues
* Add revisionHistoryLimit configuration for SonarQube application Deployment ReplicaSets & StatefulSets
* Update the security contexts to use root as group ID
* Fix empty ingress annotations in values
* Add support for dual stack and IPv6 single stack clusters in readiness/liveness probes

## [10.3.0]
* Upgrade SonarQube to 10.3.0
* Update Chart's version to 10.3.0
* Update default images to the latest versions
* Remove the nginx-proxy-body annotation when nginx is disabled
* Enable post-upgrade in the change-admin-password hook
* Update default ContainerSecurityContext, InitContainerSecurityContext and postgresql.securityContext to match restricted podSecurityStandard
* Update initFs defaut securityContext to match baseline podSecurityStandard
* Update Elasticsearch.configureNode to false by default after 3 year deprecation
* Fix wrong condition on initSysctl feature
* Update default image of initContainers to sonarqube image, allowing for faster loading time and less external images needed
* Support Kubernetes v1.28
* Avoid duplicate SONAR_WEB_SYSTEMPASSCODE secrets
* Deprecate embedded PostgreSQL
* Update nginx-ingress-controller dependency to version 4.8.3, please carefully read the changelog of this new major version.

## [10.2.0]
* Update SonarQube to 10.2.0
* Update Chart's version to 10.2.0
* Update curl image to 8.2.0
* `readinessProbe.sonarWebContext`, `startupProbe.sonarWebContext`, `livenessProbe.sonarWebContext`, and `account.sonarWebContext` are deprecated, please use `sonarWebContext` at the value top level.
* Updates ingress-nginx dependency to 4.7.1
* Fixes broken table on README

## [10.1.0]
* Update SonarQube to 10.1.0
* Support Kubernetes v1.27 while dropping v1.23
* Changed default test process to wget, using sonarqube image as default
* Update Chart's version to 10.1.0
* Fix liveness probe to detect when a failure occurs.

## [10.0.0]
* Update SonarQube to 10.0.0
* Helm chart versioning will now follow the SonarQube product versioning

## [9.5.1]
* Make `jvmOpts` and `jvmCeOpts` not override env vars and sonar properties

## [9.5.0]
* Add helm-chart-sonarqube as chart source

## [9.4.2]
* Fixed unsupported wget parameter `--proxy off` with `--no-proxy`

## [9.4.1]
* Fix install_plugins.sh not deleting previously installed plugins

## [9.4.0]
* Added support for `extraVolumes` and `extraVolumeMounts` in sonar pod.

## [9.3.1]
* Clarify doc for custom cacert secret

## [9.3.0]
* Refactor Deployment manifest to match the Statefulset manifest

## [9.2.0]
* Add a configurable Prometheus PodMonitor resource
* Refactor Prometheus exporter's documentation and bump to version 0.17.2

## [9.1.0]
* Allow setting priorityClassName for StatefulSets

## [9.0.1]
* Adds timeoutSeconds parameter to probes

## [9.0.0]
* Update SonarQube logo
* Bootstrap chart version 9.x.x dedicated to the future SonarQube 10.0
## [8.0.0]
* Update SonarQube to 9.9.0
* Bootstrap chart version 8.x.x dedicated to SonarQube 9.9 LTS

## [7.0.2]
* Update the list of supported kubernetes versions

## [7.0.1]
* Set a new default (maximum) allowed size of the client request body on the ingress

## [7.0.0]
* Update SonarQube to 9.8.0

## [6.2.1]
* Update the postgresql chart's repository


## [6.2.0]
* Refactor Ingress to be compatible with static compatibitly test and 1.19 minimum requirement

## [6.1.2]
* Updated SonarQube to 9.7.1

## [6.1.1]
* Refactor templating of ConfigMap for sonar.properties
* Fix the bug where sonarSecretKey was not applied without sonar.properties set

## [6.1.0]
* Fix the installation of plugins using the standard folder `extensions/plugins` instead of `extensions/downloads` and `lib/common`
* Remove `plugins.lib` and other small edits in the documentation

## [6.0.0]
* Updated SonarQube to 9.7.0

## [5.4.1]
* Fix the right-dash curly brace issue with the additional network policy parameter

## [5.4.0]
* Allow `tests.image` to be configured and update README accordingly.
* Allow `tests.initContainers.image` to be configured and update README accordingly.

## [5.3.0]
* Use the networkPolicy.prometheusNamespace value for the network policy namespace selector
* Uncomment default value in values.yaml for backwards compatibility

## [5.2.0]
* Add support for monitoringPasscode passed as a secret and removal of livenessprobe httpheader defined in clear text

## [5.1.0]
* Bump apiVersion to v2
* Set the number of allowed replicas to 0 and 1
* Add documentation for ingress tls
* Add documentation for sonarProperties and sonarSecretProperties
* Add the possibility of using a secret for customizing the admin password

## [5.0.6]
* Updated SonarQube to 9.6.1

## [5.0.0]
* Updated SonarQube to 9.6.0

## [4.0.3]
* Add support for Openshift Route labels and annotations

## [4.0.2]
* Fix issue with Openshift route name to use use fullname instead of name

## [4.0.1]
* Add documentation for ingress annotations

## [4.0.0]
* updated SonarQube to 9.5.0

## [3.0.4]
* Fix issue with additional network policy

## [3.0.3]
* Add automount service account token flag

## [3.0.2]
* Add documentation to setup web context via environment variable

## [3.0.1]
* Fix for issue (#215)[https://github.com/SonarSource/helm-chart-sonarqube/issues/215], adding tolerations and affinity to change password hooks

## [3.0.0]
* updated SonarQube to 9.4.0

## [2.0.7]
* Specify location of .netrc file when downloading plugins that require auth

## [2.0.6]
* Specify service account name in change admin password hook

## [2.0.5]
* secure admin password in k8s secret

## [2.0.4]
* no longer automount service account token

## [2.0.3]
* changed description of dependency postgresql chart

## [2.0.2]
* changed links to get a better overview of sources

## [2.0.1]
* Updated all instances of the caCerts enabled check

## [2.0.0]
* updated SonarQube to 9.3.0

## [1.6.5]
* add securitycontext to wait-for-db and change-password hook

## [1.6.4]
* properties are now correctly set

## [1.6.3]
* `livenessProbe.failureThreshold` was never rendered

## [1.6.2]
* added missing logic for `caCerts.enabled`

## [1.6.1]
* fix missing `SONAR_WEB_SYSTEMPASSCODE` environment variable causing failed liveness checks

## [1.5.1]
* added possibility to define host of a route

## [1.5.0]
* detached sonarqube edition from version

## [1.4.0]
* added possibility to define the ingress pathType
* added network policies
* added possibility to define ressources for the change admin password hook
* default permissions for prometheus injector now align with pod fs permissions
* updated dependencies
* admin hook now honors web context

## [1.3.0]
* added support for multiple image pull secrets
  * added `image.pullSecrets`
* deprecated support for singular image pull secret
  * deprecated `image.pullSecret`
* fixed missing image pull secret in admin hook job

## [1.2.5]
* updated SonarQube to 9.2.4

## [1.2.4]
* updated SonarQube to 9.2.3

## [1.2.3]
* updated SonarQube to 9.2.2

## [1.2.2]
* fix hardcoded reference to port 9000

## [1.2.1]
* updated SonarQube to 9.2.1

## [1.2.0]
* updated SonarQube to 9.2.0

## [1.1.11]
* fixed missing POD level security context for statefulset deployment

## [1.1.10]
* added link to community support forum
* Use liveness endpoint instead of helth endpoint for liveness probe

## [1.1.9]
* fixed wrong scc user reference if name was explicitly set

## [1.1.8]
* fixed serviceaccount logic

## [1.1.7]
* fixed wrong artifact hub images annotation

## [1.1.6]
* updated sonarqube to 9.1.0

## [1.1.5]
* added resources to ui-test pod template

## [1.1.4]
* fixed artifacthub annotations

## [1.1.3]
* fixed `invalid: metadata.labels: Invalid value` error on the `chart` label of the pvc

## [1.1.2]
* fixed condition check to add new certificates

## [1.1.1]
* updated default application version to 9.0.1
* release to helm repository

## [1.1.0]
* update jdbc overwrite values
  * replace `jdbcUrlOverride` with `jdbcOverwrite.jdbcUrl`
  * remove useless `jdbcDatabaseType` (was always postgres)
* deprecate `postgresql.postgresqlServer`, `postgresql.existingSecret` and `postgresql.existingSecretPasswordKey` in favor of new `jdbcOverwrite` values
* update dependency Charts
  * `bitnami/postgresql` from 8.6.4 to 10.4.8
  * `ingress-nginx/ingress-nginx` from 3.29.0 to 3.31.0

## [1.0.19]
* Add optional ingress parameter `ingressClassName`

## [1.0.18]
* added route support for OpenShift deployments

## [1.0.17]
* Add an additional configuration parameter `extraContainers` to allow an array of containers to run alongside the sonarqube container

## [1.0.16]
* fixed usage of `sonarSecretProperties`

## [1.0.15]
* bump jmx_exporter to 0.16.0

## [1.0.14]
* added hostAliases to deploymentType statefulset

## [1.0.13]
* made prometheus exporter port configurable and support prometheus PodMonitor

## [1.0.12]
* make sure SQ is restarted when the JMX Prometheus exporter agents configuration changes

## [1.0.11]
* JMX Prometheus exporter agent is now also enabled on the CE process
* `prometheusExporter.ceConfig` allows specific config of the JMX Prometheus exporter agent for the CE process

## [1.0.10]
* added prometheusExporter.noCheckCertificate option

## [1.0.9]
* add missing imagePullSecrets in sts install type

## [1.0.8]
* fix typo in initfs
* fix plugin installation init container permissions
* fix duplicated mount point for conf when sonar.properties are defined

## [1.0.7]
* fix invalid yaml render in `secret.yaml` when using external postgresql

## [1.0.6]
* added `prometheusExporter.downloadURL` (custom download URL for the agent jar)

## [1.0.5]
* replace `rjkernick/alpine-wget` with `curlimages/curl`
* update `install-plugins` script
* fix possible issue with prometheus init container and `env` set in the `values.yaml`

## [1.0.4]
* fix for missing `serviceAccountName` in STS deployment kind

## [1.0.3]
* fixed prometheus config volume mount if disabled
* switched from wget to curl image per default for downloading agent
* added support for proxy envs

## [1.0.2]
* added option to configure CE java opts separately

## [1.0.1]
* fixed missing conditional that was introduced in 0.9.2.2 to sonarqube-sts.yaml
* updated default application version to 8.9

## [1.0.0]
* changed default deployment from replica set to statefull set
* added default support for prometheus jmx exporter
* added init filesystem container
* added nginx-ingress as optional dependency
* updated application version to 8.8-community
* improved readiness/startup and liveness probes
* improved documentation

## [0.9.6.2]
* Change order of env variables to better support 7.9-lts

## [0.9.6.1]
* Add support for setting custom annotations in admin hook job.

## [0.9.6.0]
* Add the possibility of definining the secret key name of the postgres password.

## [0.9.5.0]
* Add Ingress default backend for GCE class

## [0.9.2.3]
* Added namespace to port-foward command in notes.

## [0.9.2.2]
* Added a condition to deployment.yaml so that `wait-for-db` initContainer is only created if `postgresql.enabled=true`

## [0.9.2.1]
* Updated the configuration table to include the additional keys added in release 9.2.0.

## [0.9.2.0]
* Added functionality for deployments to OpenShift clusters.
    * .Values.OpenShift flag to signify if deploying to OpenShift.
	* Ability to have chart generate an SCC allowing the init-sysctl container to run as privileged.
	* Setting of a seperate securityContext section for the main SonarQube container to avoid running as root.
	* Exposing additional `postreSQL` keys in values.yaml to support configuring postgres to run under standard "restricted" or "anyuid"/"nonroot" SCCs on OpenShift.
* Added initContainer `wait-for-db` to await postgreSQL successful startup before starting SonarQube, to avoid race conditions.

## [0.9.1.1]
* Update SonarQube to 8.5.1.
* **Fix:** Purge plugins directory before download.

## [0.9.0.0]
* Update SonarQube to 8.5.
* **Breaking change:** Rework init containers.
    * Move global defaults from `plugins` section to `initContainers`.
    * Update container images.
* **Deprecation:** `elasticsearch.configureNode` in favor of `initSysctl.enabled`.
* Rework sysctl with support for custom values.
* Rework plugins installation via `opt/sonarqube/extensions/downloads` folder that is handled by SonarQube itself.
    * **Breaking change:** remove `plugins.deleteDefaultPlugins` as SonarQube stores bundled plugins out of `opt/sonarqube/extensions`.
* Rename deprecated `SONARQUBE_` environment variables to `SONAR_` ones.
* **Breaking change:** Rename `enabledTests` to `tests.enabled`.
* Add `terminationGracePeriodSeconds`.
