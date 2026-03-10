#!/bin/bash

set -euo pipefail

report=$(<"${1:-/dev/stdin}")

violations=$(echo "${report}" | yq e '.results | filter(.type == "Mandatory") | filter(.outcome == "FAIL")')

if [[ "${violations}" = "[]" ]]; then
  echo "No violations found"
  exit 0
fi

echo "${violations}"
exit 1
