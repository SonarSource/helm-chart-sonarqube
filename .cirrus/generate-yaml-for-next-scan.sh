#!/bin/bash
set -euo pipefail

KUBE_VERSION="${KUBE_VERSION:-1.28.0}"
STATIC_TEST_FOLDER='../../tests/unit-compatibility-test/'$(basename $PWD)
YAML_SCAN_FOLDER='../../helm-chart-yaml-files/'$(basename $PWD)

rm -R $YAML_SCAN_FOLDER 2>/dev/null || true
mkdir -p $YAML_SCAN_FOLDER


if ! [ -d "$STATIC_TEST_FOLDER" ]; then
    echo "$STATIC_TEST_FOLDER folder not found"
    echo "you are probably not in the root of a chart"
    exit 1
fi

echo 'Running templating of our helm chart for scanning them on next'

for file in "$STATIC_TEST_FOLDER"/*; do
    TEST_CASE_NAME=$(basename "$file")
    echo 'Entering templating for' $TEST_CASE_NAME
    helm template --kube-version $KUBE_VERSION --dry-run --debug --set postgresql.enabled=false -f "$file" $TEST_CASE_NAME . > $YAML_SCAN_FOLDER/$TEST_CASE_NAME
    echo 'Ending templating for' $TEST_CASE_NAME
done