#!/bin/bash
set -euo pipefail

KUBE_VERSION="${KUBE_VERSION:-1.25.0}"
STATIC_TEST_FOLDER='../../tests/unit-compatibility-test/'$(basename $PWD)


if ! [ -d "$STATIC_TEST_FOLDER" ]; then
    echo "$STATIC_TEST_FOLDER folder not found"
    echo "you are probably not in the root of a chart"
    exit 1
fi

echo 'Running unit compatibility tests for Kubernetes version' $KUBE_VERSION

for file in "$STATIC_TEST_FOLDER"/*; do
    TEST_CASE_NAME=$(basename "$file")
    echo 'Entering test for' $TEST_CASE_NAME
    helm template --kube-version $KUBE_VERSION --dry-run --debug -f "$file" $TEST_CASE_NAME . | kubeconform --kubernetes-version $KUBE_VERSION --summary --strict
    echo 'Ending test for' $TEST_CASE_NAME
done