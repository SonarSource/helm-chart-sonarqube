# SonarQube Chart Changelog
All changes to this chart will be documented in this file.

## [9.1.1]
* Update SonarQube to 8.5.1.
* **Fix:** Purge plugins directory before download.

## [9.0.0]
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
