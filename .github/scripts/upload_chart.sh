#!/bin/bash

set -xeuo pipefail

: "${ARTIFACTORY_URL:?}"
: "${ARTIFACTORY_ACCESS_TOKEN:?}"
: "${GITHUB_WORKSPACE:?}"           # GitHub Actions working directory
: "${GITHUB_REPOSITORY:?}"          # GitHub repository name
: "${BUILD_NUMBER:?}"

CHART_TO_UPLOAD=${1:-}

# Extract repository name from GITHUB_REPOSITORY (format: owner/repo)
REPO_NAME=$(basename "${GITHUB_REPOSITORY}")

# If there is a $1 argument, treat it as the chart to sign by looking for $1*.tgz* files
# Otherwise, look for all *.tgz* files in the working directory
NAME_GLOB="*.tgz*"
if [[ -n "${CHART_TO_UPLOAD}" ]]; then
    NAME_GLOB="${CHART_TO_UPLOAD}-[0-9]*.tgz*"
fi

find_charts=$(find "${GITHUB_WORKSPACE}" -maxdepth 1 -name "${NAME_GLOB}" -type f -exec basename "{}" ";" || exit 1)

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

jf config add repox \
    --artifactory-url "${ARTIFACTORY_URL}" \
    --access-token "${ARTIFACTORY_ACCESS_TOKEN}"

for chart in "${CHART_TO_UPLOAD[@]}"; do
    echo "Uploading ${chart}"
    jf rt upload --build-name "${REPO_NAME}" --build-number "${BUILD_NUMBER}" "${chart}" sonarsource-helm-builds
done

jf rt build-publish "${REPO_NAME}" "${BUILD_NUMBER}"
