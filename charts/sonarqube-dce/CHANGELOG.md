# SonarQube Chart Changelog
All changes to this chart will be documented in this file.

## [3.0.1]
* Add documentation for ingress annotations

## [3.0.0]
* updated SonarQube to 9.5.0

## [2.0.4]
* Fix issue with additional network policy

## [2.0.3]
* Add automount service account token flag

## [2.0.2]
* Add documentation to setup web context via environment variable

## [2.0.1]
* Fix for issue (#215)[https://github.com/SonarSource/helm-chart-sonarqube/issues/215], adding tolerations and affinity to change password hooks

## [2.0.0]
* updated SonarQube to 9.4.0

## [1.0.7]
* Specify location of .netrc file when downloading plugins that require auth

## [1.0.6]
* Fixed properties scope for app deployment and search sts

## [1.0.5]
* Specify service account name in change admin password hook

## [1.0.4]
* secure admin password in k8s secret

## [1.0.3]
* changed description of dependency postgresql chart

## [1.0.2]
* changed links to get a better overview of sources

## [1.0.1]
* Updated all instances of the caCerts enabled check

## [1.0.0]
* updated SonarQube to 9.3.0
* officially support SonarQube DCE

## [0.6.5]
* name of elasticsearch keystore password secret is now aligned with the rest of the chart

## [0.6.4]
* properties are now correctly set

## [0.6.3]
* `livenessProbe.failureThreshold` was never rendered

## [0.6.2]
* added missing logic for `caCerts.enabled`

## [0.6.1]
* reverted liveness prove change
## [0.5.0]
* added support for additional sidecar container

## [0.4.0]
* added possibility to define the ingress pathType
* added network policies
* added possibility to define ressources for the change admin password hook
* default permissions for prometheus injector now align with pod fs permissions
* updated dependencies
* admin hook now honors web context
* added pod distribution budget

## [0.3.0]
* added support for multiple image pull secrets
  * added `searchNodes.image.pullSecrets`
  * added `ApplicationNodes.image.pullSecrets`
* deprecated support for singular image pull secret
  * deprecated `searchNodes.image.pullSecret`
  * deprecated `ApplicationNodes.image.pullSecret`
* fixed missing image pull secret in admin hook job

## [0.2.5]
* updated SonarQube to 9.2.4

## [0.2.4]
* updated SonarQube to 9.2.3

## [0.2.3]
* updated SonarQube to 9.2.2

## [0.2.2]
* replaced hardcoded port's values

## [0.2.1]
* updated SonarQube to 9.2.1

## [0.2.0]
* updated SonarQube to 9.2.0

## [0.1.7]
* added possibility to secure connection in between search nodes
* added link to community support forum

## [0.1.6]
* fixed wrong scc user reference if name was explicitly set 

## [0.1.5]
* fixed serviceaccount logic

## [0.1.4]
* fixed wrong limits reference in sonarqube-application deployment

## [0.1.3]
* Homoginized Description of the Helm Chart
* fixed wrong link in README.md

## [0.1.2]
* changed external database connection logic

## [0.1.1]
* added option to use external secret as jwtSecret
* removed initSysctl from application pods

## [0.1.0]
* fixed pvc template for search nodes
* added flag to handle hazelcast on k8s
* adapted probes for search nodes
* made search nodes start up in parallel
* search nodes no longer wait for the database connection
* added new headless services for search and app

## [0.0.1]
* separated search and app configuration
* added new search service
