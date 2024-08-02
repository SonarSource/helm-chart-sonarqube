#!/bin/bash

set -euo pipefail

report=$(<"${1:-/dev/stdin}")

violations=$(echo "${report}" | docker run --rm -i mikefarah/yq e '.results | filter(.type == "Mandatory") | filter(.outcome == "FAIL")')

if [[ -z "${violations}" ]]; then
  echo "No violations found"
  exit 0
fi

echo "${violations}"
exit 1
