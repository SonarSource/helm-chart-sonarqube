#!/bin/bash

set -euo pipefail

cd "${GITHUB_WORKSPACE:-$(pwd)}/tests/unit-test"

go test -timeout=0 -v schema_test.go
