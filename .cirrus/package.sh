#!/bin/bash

set -xeuo pipefail

: "${CIRRUS_BRANCH:?}"
: "${BUILD_NUMBER:?}"
: "${CIRRUS_BASE_BRANCH:=}"

if [[ -n "${CIRRUS_BASE_BRANCH}" ]]; then
    TARGET_BRANCH="${CIRRUS_BASE_BRANCH}"
else
    TARGET_BRANCH="${CIRRUS_BRANCH}"
fi

PREVIOUS_RELEASE=$(gh api "/repos/{owner}/{repo}/releases" --jq "[.[] | select(.target_commitish==\"${TARGET_BRANCH}\")][1].tag_name")

if [[ -z "${PREVIOUS_RELEASE}" ]]; then
    CHARTS=("charts/sonarqube-dce" "charts/sonarqube")
else
    # shellcheck disable=SC2178  # This will output a string, we will use it only in the for-loop, which will split it
    CHARTS=$(ct list-changed --since "${PREVIOUS_RELEASE}" --target-branch "${TARGET_BRANCH}")
fi

# If there is a $1 argument, and it is contained in the CHARTS array, then we will only package that chart
ARG_CHART_NAME=${1:+charts/$1}
if [[ -n "${ARG_CHART_NAME}" ]] && [[ "${CHARTS[*]}" =~ ${ARG_CHART_NAME} ]]; then
    CHARTS=("${ARG_CHART_NAME}")
fi

BUILD_METADATA="-${BUILD_NUMBER}"
[[ ${CIRRUS_RELEASE:-} != "" ]] && BUILD_METADATA=""

echo "${CHARTS[@]}"

# shellcheck disable=SC2068  # Because ct list-changed will return a string, we want the potential split here
for chart in ${CHARTS[@]}; do
    _original_version=$(yq '.version' "${chart}"/Chart.yaml)
    _new_version="${_original_version}${BUILD_METADATA}"
    helm dependency build "${chart}"
    helm package --version "${_new_version}" "${chart}"
done
