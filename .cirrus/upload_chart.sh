#!/bin/sh

ARTIFACTS_LOCATION="/tmp/cirrus-ci-build"
CURRENT_DIR=`pwd`

for chart in `find $ARTIFACTS_LOCATION -name "*.tgz" -type f -printf "%f\n"`; do
    cd $ARTIFACTS_LOCATION
    echo "Uploading $chart"
    curl "-u${ARTIFACTORY_DEPLOY_USERNAME}:${ARTIFACTORY_DEPLOY_PASSWORD}" "-T $chart" "https://repox.jfrog.io/artifactory/sonarsource-helm-builds/$chart"
    cd $CURRENT_DIR
done