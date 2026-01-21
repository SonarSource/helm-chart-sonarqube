#!/bin/bash
set -euo pipefail

KUBE_VERSION="${KUBE_VERSION:-1.35.0}"

echo 'Running fixtures'

for path in "sonarqube" "sonarqube-dce"; do
    STATIC_TEST_FOLDER="./tests/unit-compatibility-test/${path}"
    CHART_TEST_FOLDER="./charts/${path}"

    if ! [[ -d "$STATIC_TEST_FOLDER" ]]; then
        echo "$STATIC_TEST_FOLDER folder not found"
        echo "${path} not a valid chart path"
        exit 1
    fi

    echo "Processing fixtures for chart: $path"

    for file in "${STATIC_TEST_FOLDER}"/*; do
        TEST_CASE_NAME=$(basename "$file")
        FIXTURE_STATIC_TEST_FOLDER="./tests/unit-compatibility-test/fixtures/${path}/${TEST_CASE_NAME}"

        echo "Entering fixture test for ${TEST_CASE_NAME}"
        helm template --set monitoringPasscode='test'  --set applicationNodes.jwtSecret='some-secret' --set global.postgresql.postgresqlPostgresPassword='toto' --kube-version "$KUBE_VERSION" --dry-run --debug -f "$file" "${TEST_CASE_NAME}" ${CHART_TEST_FOLDER} > "${FIXTURE_STATIC_TEST_FOLDER}"
        echo "Ending fixture test test for ${TEST_CASE_NAME}"
    done

done