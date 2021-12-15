# SonarQube Chart Changelog
All changes to this chart will be documented in this file.

## [0.3.0]
* added support multiple image pull secrets
  * replaced `searchNodes.image.pullSecret` with `searchNodes.image.pullSecrets`
  * replaced `ApplicationNodes.image.pullSecret` with `ApplicationNodes.image.pullSecrets`
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

