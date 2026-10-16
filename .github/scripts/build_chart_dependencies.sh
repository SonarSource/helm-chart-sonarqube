#!/bin/bash

set -euo pipefail

VERIFYING_CHART="${1}"

# Keeps every caller of this script in sync with Chart.yaml's dependencies,
# instead of registering the repo separately in each workflow.
helm repo add keda https://kedacore.github.io/charts

helm dependency build "${VERIFYING_CHART}"
