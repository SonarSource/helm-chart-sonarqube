#!/bin/bash

set -euo pipefail

VERIFYING_CHART="${1}"

helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm dependency build "${VERIFYING_CHART}"
