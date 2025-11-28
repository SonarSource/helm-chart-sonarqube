#!/bin/bash

set -xeuo pipefail

# GitHub Actions environment variables mapping
: "${GITHUB_REF_NAME:?}"        # Current branch name
: "${BUILD_NUMBER:?}"            # Build number from get-build-number action
: "${GITHUB_BASE_REF:=}"         # Base branch for PRs

if [[ -n "${GITHUB_BASE_REF}" ]]; then
    TARGET_BRANCH="${GITHUB_BASE_REF}"
else
    TARGET_BRANCH="${GITHUB_REF_NAME}"
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
# For GitHub Actions, check if this is a tag-based release
[[ ${GITHUB_REF_TYPE:-} == "tag" ]] && BUILD_METADATA=""

echo "${CHARTS[@]}"

# shellcheck disable=SC2068  # Because ct list-changed will return a string, we want the potential split here
for chart in ${CHARTS[@]}; do
    _original_version=$(yq '.version' "${chart}"/Chart.yaml)
    _new_version="${_original_version}${BUILD_METADATA}"
    helm dependency build "${chart}"
    helm package --version "${_new_version}" "${chart}"
done
