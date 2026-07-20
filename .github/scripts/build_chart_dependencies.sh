#!/bin/bash

set -euo pipefail

VERIFYING_CHART="${1}"

helm dependency build "${VERIFYING_CHART}"
