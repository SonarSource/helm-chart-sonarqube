#!/bin/sh

CURRENT_DIR=${pwd}

for chart in ${find $CIRRUS_WORKING_DIR -name "*.tgz" -type f -maxdepth 1 -exec basename "{}" ";"}; do
    cd $CIRRUS_WORKING_DIR
    echo "Uploading $chart"
    curl "-u${ARTIFACTORY_DEPLOY_USERNAME}:${ARTIFACTORY_DEPLOY_PASSWORD}" -T "$chart" "https://repox.jfrog.io/artifactory/sonarsource-helm-builds/$chart"
    cd $CURRENT_DIR
done