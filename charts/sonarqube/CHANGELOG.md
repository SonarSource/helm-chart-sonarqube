# SonarQube Chart Changelog
All changes to this chart will be documented in this file.

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
