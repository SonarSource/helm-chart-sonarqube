#!/bin/bash

set -xeuo pipefail

: "${ARTIFACTORY_URL:?}"
: "${ARTIFACTORY_ACCESS_TOKEN:?}"
: "${CIRRUS_WORKING_DIR:?}"
: "${CIRRUS_REPO_NAME:?}"
: "${BUILD_NUMBER:?}"

CHART_TO_UPLOAD=${1:-}

# If there is a $1 argument, treat it as the chart to sign by looking for $1*.tgz* files
# Otherwise, look for all *.tgz* files in the working directory
NAME_GLOB="*.tgz*"
if [[ -n "${CHART_TO_UPLOAD}" ]]; then
    NAME_GLOB="${CHART_TO_UPLOAD}-[0-9]*.tgz*"
fi

find_charts=$(find "${CIRRUS_WORKING_DIR}" -maxdepth 1 -name "${NAME_GLOB}" -type f -exec basename "{}" ";" || exit 1)

CHART_TO_UPLOAD=()
if [[ -n "${find_charts}" ]]; then
    while IFS= read -r chart; do
        CHART_TO_UPLOAD+=("${chart}")
    done <<< "${find_charts}"
fi

if [[ ${#CHART_TO_UPLOAD[@]} -eq 0 ]]; then
    echo "No charts found to upload."
    exit 1
fi

jfrog config add repox \
    --artifactory-url "${ARTIFACTORY_URL}" \
    --access-token "${ARTIFACTORY_ACCESS_TOKEN}"

for chart in "${CHART_TO_UPLOAD[@]}"; do
    echo "Uploading ${chart}"
    jfrog rt upload --build-name "${CIRRUS_REPO_NAME}" --build-number "${BUILD_NUMBER}" "${chart}" sonarsource-helm-builds
done

jfrog rt build-publish "${CIRRUS_REPO_NAME}" "${BUILD_NUMBER}"
