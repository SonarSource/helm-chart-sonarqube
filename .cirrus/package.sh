#!/bin/bash

set -xeuo pipefail

: "${CIRRUS_BRANCH:?}"
: "${BUILD_NUMBER:?}"
: "${CIRRUS_BASE_BRANCH:=}"

[[ -n "${CIRRUS_BASE_BRANCH}" ]] && TARGET_BRANCH="${CIRRUS_BASE_BRANCH}" || TARGET_BRANCH="${CIRRUS_BRANCH}"

PREVIOUS_RELEASE=$(gh api "/repos/{owner}/{repo}/releases" --jq "[.[] | select(.target_commitish==\"${TARGET_BRANCH}\")][1].tag_name")

[[ -z "${PREVIOUS_RELEASE}" ]] && CHARTS=("charts/sonarqube-dce" "charts/sonarqube") || CHARTS=$(ct list-changed --since "${PREVIOUS_RELEASE}" --target-branch "${TARGET_BRANCH}")

BUILD_METADATA="-${BUILD_NUMBER}"
[[ ${CIRRUS_RELEASE:-} != "" ]] && BUILD_METADATA=""

echo "${CHARTS[@]}"

for chart in ${CHARTS[@]}; do
    _original_version=$(yq '.version' "${chart}"/Chart.yaml)
    _new_version="${_original_version}${BUILD_METADATA}"
    helm dependency build "${chart}"
    helm package --version "${_new_version}" "${chart}"
done
