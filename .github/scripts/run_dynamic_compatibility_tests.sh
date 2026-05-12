#!/usr/bin/env bash
#
# Run the SonarQube dynamic compatibility tests against the cluster pointed at
# by the current kubectl context. Assumes the cluster (e.g. kind) and helm/go
# are already available.
#
# Postgres is installed per-test from inside the Go test code, so this script
# only needs to handle chart-dependency wiring.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo ">>> Building chart dependencies"
"${REPO_ROOT}/.github/scripts/build_chart_dependencies.sh" "${REPO_ROOT}/charts/sonarqube"
"${REPO_ROOT}/.github/scripts/build_chart_dependencies.sh" "${REPO_ROOT}/charts/sonarqube-dce"

echo ">>> Running dynamic compatibility tests"
cd "${REPO_ROOT}/tests/dynamic-compatibility-test"


# Execute 4 tests in a package(--parallel) and 4 packages(-p) in parallel
# This helps in reducing the time taken to execute the dynamic tests
go test -v -timeout=90m -parallel 2 -p 2 ./...
