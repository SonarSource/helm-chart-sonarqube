#!/bin/bash

set -xeo pipefail

VERSION_SEPERATOR="-"
[[ ${CIRRUS_RELEASE:-} != "" ]] && VERSION_SEPERATOR="+"

PREVIOUS_RELEASE=$(gh api "/repos/{owner}/{repo}/releases" --jq "[.[] | select(.target_commitish==\"${CIRRUS_BRANCH}\")][1].tag_name")

# There MIGHT be a some edge case where PREVIOUS_RELEASE shouldn't be HEAD,
# for example, releasing a patch for non-LTA. To be investigated.
[[ -z "${PREVIOUS_RELEASE}" ]] && PREVIOUS_RELEASE="HEAD" || echo "${PREVIOUS_RELEASE}"

echo $(ct list-changed --since "${PREVIOUS_RELEASE}" --target-branch "${CIRRUS_BRANCH}")

for chart in $(ct list-changed --since "${PREVIOUS_RELEASE}" --target-branch "${CIRRUS_BRANCH}"); do
    _original_version=$(cat $chart/Chart.yaml | yq '.version' -)
    _new_version="${_original_version}${VERSION_SEPERATOR}${BUILD_NUMBER}"
    helm dependency build "${chart}"
    helm package --version "${_new_version}" "${chart}"
done
