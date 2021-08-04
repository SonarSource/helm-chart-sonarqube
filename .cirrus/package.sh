#!/bin/bash

set -eo pipefail

VERSION_SEPERATOR="-"
[[ ${CIRRUS_RELEASE:-} != "" ]] && VERSION_SEPERATOR="+"

LAST_TAG=$(gh api "/repos/{owner}/{repo}/releases?per_page=2" --jq ".[1].tag_name")

[ -z "$LAST_TAG" ] && LAST_TAG="HEAD" || echo $LAST_TAG

echo $(ct list-changed --since $LAST_TAG)

for chart in $(ct list-changed --since $LAST_TAG); do
    _original_version=$(cat $chart/Chart.yaml | yq r - version)
    _new_version="$_original_version$VERSION_SEPERATOR$BUILD_NUMBER"
    helm dependency build $chart
    helm package --version $_new_version $chart
done