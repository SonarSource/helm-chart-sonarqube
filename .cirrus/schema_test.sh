#!/bin/bash

set -euo pipefail

cd "$(dirname "$0")/../tests/dynamic-compatibility-test"

go test -timeout=0 -v schema_test.go
