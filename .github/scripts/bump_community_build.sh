#!/bin/bash
set -euo pipefail

: "${OLD_BUILD:?OLD_BUILD is required}"
: "${NEW_BUILD:?NEW_BUILD is required}"

if [[ "${OLD_BUILD}" == "${NEW_BUILD}" ]]; then
    echo "OLD_BUILD and NEW_BUILD are identical (${NEW_BUILD}); nothing to do"
    exit 1
fi

CHART_DIR="charts/sonarqube"

assert_present() {
    local file="$1" needle="$2"
    if ! grep -qF "${needle}" "${file}"; then
        echo "ERROR: expected to find '${needle}' in ${file}" >&2
        exit 1
    fi
}

echo "Bumping SonarQube Community Build: ${OLD_BUILD} -> ${NEW_BUILD}"

# Chart.yaml — annotation strings (artifacthub.io/changes + artifacthub.io/images)
assert_present "${CHART_DIR}/Chart.yaml" "Upgrade SonarQube Community build to ${OLD_BUILD}"
assert_present "${CHART_DIR}/Chart.yaml" "sonarqube:${OLD_BUILD}-community"
sed -i \
    -e "s|Upgrade SonarQube Community build to ${OLD_BUILD}|Upgrade SonarQube Community build to ${NEW_BUILD}|" \
    -e "s|sonarqube:${OLD_BUILD}-community|sonarqube:${NEW_BUILD}-community|" \
    "${CHART_DIR}/Chart.yaml"

# CHANGELOG.md — replace only the first occurrence (top-most chart version section)
assert_present "${CHART_DIR}/CHANGELOG.md" "Upgrade SonarQube Community build to ${OLD_BUILD}"
awk -v old="${OLD_BUILD}" -v new="${NEW_BUILD}" '
    !done && index($0, "Upgrade SonarQube Community build to " old) {
        sub("Upgrade SonarQube Community build to " old, "Upgrade SonarQube Community build to " new)
        done = 1
    }
    { print }
' "${CHART_DIR}/CHANGELOG.md" > "${CHART_DIR}/CHANGELOG.md.tmp"
mv "${CHART_DIR}/CHANGELOG.md.tmp" "${CHART_DIR}/CHANGELOG.md"

# README.md — 2 occurrences (intro line + community.buildNumber parameters row)
assert_present "${CHART_DIR}/README.md" "${OLD_BUILD}"
sed -i "s|${OLD_BUILD}|${NEW_BUILD}|g" "${CHART_DIR}/README.md"

# values.yaml — community.buildNumber
assert_present "${CHART_DIR}/values.yaml" "buildNumber: \"${OLD_BUILD}\""
sed -i "s|buildNumber: \"${OLD_BUILD}\"|buildNumber: \"${NEW_BUILD}\"|" \
    "${CHART_DIR}/values.yaml"

# ci/ci-values.yaml — image.tag
assert_present "${CHART_DIR}/ci/ci-values.yaml" "tag: \"${OLD_BUILD}-master-community\""
sed -i "s|tag: \"${OLD_BUILD}-master-community\"|tag: \"${NEW_BUILD}-master-community\"|" \
    "${CHART_DIR}/ci/ci-values.yaml"

# openshift-verifier/values.yaml — image.tag
assert_present "${CHART_DIR}/openshift-verifier/values.yaml" "tag: \"${OLD_BUILD}-master-community\""
sed -i "s|tag: \"${OLD_BUILD}-master-community\"|tag: \"${NEW_BUILD}-master-community\"|" \
    "${CHART_DIR}/openshift-verifier/values.yaml"

# tests/unit-test/sonarqube_schema_test.go — expectedContainerImage constant
assert_present tests/unit-test/sonarqube_schema_test.go "expectedContainerImage string = \"sonarqube:${OLD_BUILD}\""
sed -i "s|expectedContainerImage string = \"sonarqube:${OLD_BUILD}\"|expectedContainerImage string = \"sonarqube:${NEW_BUILD}\"|" \
    tests/unit-test/sonarqube_schema_test.go

# tests/unit-test/test-cases-values/sonarqube/test-build-number.yaml
assert_present tests/unit-test/test-cases-values/sonarqube/test-build-number.yaml "buildNumber: \"${OLD_BUILD}\""
sed -i "s|buildNumber: \"${OLD_BUILD}\"|buildNumber: \"${NEW_BUILD}\"|" \
    tests/unit-test/test-cases-values/sonarqube/test-build-number.yaml

echo "Edits applied successfully."
