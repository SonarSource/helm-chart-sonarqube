# SonarQube Chart Changelog
All changes to this chart will be documented in this file.

[0.1.2]
* changed external database connection logic

[0.1.1]
* added option to use external secret as jwtSecret
* removed initSysctl from application pods

[0.1.0]
* fixed pvc template for search nodes
* added flag to handle hazelcast on k8s
* adapted probes for search nodes
* made search nodes start up in parallel
* search nodes no longer wait for the database connection
* added new headless services for search and app

[0.0.1]
* separated search and app configuration
* added new search service

