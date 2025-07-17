#!/bin/bash

set -euo pipefail

PROJECT_ROOT=$(readlink -f "$(dirname "$0")"/..)
CHART_NAME="${1:-$(basename "${PWD}")}"

# NOTE: CHART_PATH is set to the first argument if it is provided,
# otherwise it is set to "." (current dir)
CHART_PATH="${1:+charts/$1}"
CHART_PATH="${CHART_PATH:-.}"

KUBE_VERSION="${KUBE_VERSION:-1.33.1}"
STATIC_TEST_FOLDER="${PROJECT_ROOT}/tests/unit-compatibility-test/${CHART_NAME}"

if ! [[ -d "${STATIC_TEST_FOLDER}" ]]; then
    echo "${STATIC_TEST_FOLDER} folder not found"
    echo "you are probably not in the root of a chart"
    exit 1
fi

echo "Running unit compatibility tests for Kubernetes version ${KUBE_VERSION}"

for file in "${STATIC_TEST_FOLDER}"/*; do
    TEST_CASE_NAME=$(basename "${file}")
    echo "Entering test for ${TEST_CASE_NAME}"
    helm template \
        --kube-version "${KUBE_VERSION}" \
        --dry-run \
        --debug \
        --set monitoringPasscode='test' \
        --set applicationNodes.jwtSecret='some-secret' \
        -f "${file}" "${TEST_CASE_NAME}" "${CHART_PATH}" \
    | kubeconform \
        --kubernetes-version "${KUBE_VERSION}" \
        --summary \
        --strict -schema-location default \
        -schema-location "${PROJECT_ROOT}/tests/test-crds/{{.ResourceKind}}.json"
    echo "Ending test for ${TEST_CASE_NAME}"
done
