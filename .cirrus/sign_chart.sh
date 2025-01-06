#!/bin/bash

set -euo pipefail

: "${SONARSOURCE_SIGN_KEY_PASSPHRASE:?}"
: "${CIRRUS_WORKING_DIR:?}"

# If there is a $1 argument, treat it as the chart to sign by looking for $1*.tgz* files
# Otherwise, look for all *.tgz* files in the working directory
CHART_TO_SIGN=${1:-}
NAME_GLOB="*.tgz*"
if [[ -n "${CHART_TO_SIGN}" ]]; then
    NAME_GLOB="${CHART_TO_SIGN}-[0-9]*.tgz*"
fi

find_charts=$(find "${CIRRUS_WORKING_DIR}" -maxdepth 1 -name "${NAME_GLOB}" -type f -exec basename "{}" ";" || exit 1)

CHART_TO_SIGN=()
while IFS= read -r chart; do
    CHART_TO_SIGN+=("${chart}")
done <<< "${find_charts}"

if [[ ${#CHART_TO_SIGN[@]} -eq 0 ]]; then
    echo "No charts found to sign."
    exit 1
fi

# Debugging: Print the charts to be signed
echo "Charts to sign: ${CHART_TO_SIGN[*]}"

echo "${SONARSOURCE_SIGN_KEY_PASSPHRASE}" | gpg --batch --yes --passphrase-fd 0 --import /tmp/key

for chart in "${CHART_TO_SIGN[@]}"; do
    echo "Signing ${chart}"
    echo "${SONARSOURCE_SIGN_KEY_PASSPHRASE}" | gpg --batch --yes --pinentry-mode loopback --passphrase-fd 0 --output "${chart}.asc" --detach-sig "${chart}"
done
