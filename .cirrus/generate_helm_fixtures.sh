#!/bin/bash
set -euo pipefail

KUBE_VERSION="${KUBE_VERSION:-1.25.0}"
STATIC_TEST_FOLDER='../../tests/unit-compatibility-test/'$(basename $PWD)

if ! [ -d "$STATIC_TEST_FOLDER" ]; then
    echo "$STATIC_TEST_FOLDER folder not found"
    echo "you are probably not in the root of a chart"
    exit 1
fi

echo 'Generating all unit test files to folder'

for file in "$STATIC_TEST_FOLDER"/*; do

    TEST_CASE_NAME=$(basename "$file")
    FIXTURE_STATIC_TEST_FOLDER="../../tests/unit-compatibility-test/fixtures/$(basename $PWD)/${TEST_CASE_NAME}"

    echo 'Entering test for' $TEST_CASE_NAME
    helm template --kube-version $KUBE_VERSION --dry-run --debug -f "$file" $TEST_CASE_NAME . > ${FIXTURE_STATIC_TEST_FOLDER}
    echo 'Ending test for' $TEST_CASE_NAME
done