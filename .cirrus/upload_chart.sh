#!/bin/bash

set -euo pipefail

CURRENT_DIR=$(pwd)
CHARTS=$(find $CIRRUS_WORKING_DIR -maxdepth 1 -name "*.tgz*" -type f -exec basename "{}" ";")

[[ "x$CHARTS" == "x" ]] && exit 0

jfrog config add repox \
  --artifactory-url "${ARTIFACTORY_URL}" \
  --access-token "${ARTIFACTORY_ACCESS_TOKEN}"

for chart in $CHARTS; do
    cd $CIRRUS_WORKING_DIR
    jfrog rt upload --build-name "$CIRRUS_REPO_NAME" --build-number "$BUILD_NUMBER" "$chart" sonarsource-helm-builds
    cd $CURRENT_DIR
done

jfrog rt build-publish "$CIRRUS_REPO_NAME" "$BUILD_NUMBER"
