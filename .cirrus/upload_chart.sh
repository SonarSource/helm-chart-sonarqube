#!/bin/bash

set -euo pipefail

CURRENT_DIR=$(pwd)

for chart in $(find $CIRRUS_WORKING_DIR -maxdepth 1 -name "*.tgz*" -type f -exec basename "{}" ";"); do
    cd $CIRRUS_WORKING_DIR
    _ARTIFACT_MD5_CHECKSUM=$(md5sum $chart | awk '{ print $1 }')
    _ARTIFACT_SHA1_CHECKSUM=$(shasum -a 1 $chart | awk '{ print $1 }')
    _ARTIFACT_SHA256_CHECKSUM=$(sha256sum $chart | awk '{ print $1 }')
    echo "Uploading $chart"
    curl "-u${ARTIFACTORY_DEPLOY_USERNAME}:${ARTIFACTORY_DEPLOY_PASSWORD}" \
        -T "$chart" \
        -H "X-Checksum-MD5:${_ARTIFACT_MD5_CHECKSUM}" \
        -H "X-Checksum-Sha1:${_ARTIFACT_SHA1_CHECKSUM}" \
        -H "X-Checksum-Sha256:${_ARTIFACT_SHA256_CHECKSUM}" \
        "https://repox.jfrog.io/artifactory/sonarsource-helm-builds/$chart"
    cd $CURRENT_DIR
done