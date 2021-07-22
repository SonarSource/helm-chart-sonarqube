#!/bin/bash

set -euo pipefail

CURRENT_DIR=$(pwd)

jfrog config add repox --artifactory-url ${ARTIFACTORY_URL} --user ${ARTIFACTORY_DEPLOY_USERNAME} --password ${ARTIFACTORY_DEPLOY_PASSWORD} --basic-auth-only

for chart in $(find $CIRRUS_WORKING_DIR -maxdepth 1 -name "*.tgz*" -type f -exec basename "{}" ";"); do
    cd $CIRRUS_WORKING_DIR
    jfrog rt upload --build-name "$CIRRUS_REPO_NAME" --build-number "$BUILD_NUMBER" "$chart" sonarsource-helm-builds
    cd $CURRENT_DIR
done

jfrog rt build-publish "$CIRRUS_REPO_NAME" "$BUILD_NUMBER"
