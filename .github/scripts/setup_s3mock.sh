#!/usr/bin/env bash

set -euo pipefail

# Deploys the S3 stand-in used by the agentic CI scenario and waits for it to be ready.
NAMESPACE="${NAMESPACE:-sonarqube}"

echo "Installing s3mock in namespace ${NAMESPACE}..."
kubectl apply -n "${NAMESPACE}" -f "$(dirname "$0")/../ci/s3mock.yaml"
kubectl rollout status deployment/s3 -n "${NAMESPACE}" --timeout=300s
echo "s3mock installation completed."
