#!/usr/bin/env bash

set -euo pipefail

# Environment variables with default values
NAME="${NAME:-external-postgres}"
NAMESPACE="${NAMESPACE:-sonarqube}"
VERSION="${VERSION:-18.2.3}"
VALUES_FILE="${VALUES_FILE:-}"

echo "Installing PostgreSQL with the following configuration:"
echo "  Name: ${NAME}"
echo "  Namespace: ${NAMESPACE}"
echo "  Chart Version: ${VERSION}"
if [[ -n "${VALUES_FILE}" ]]; then
  echo "  Values File: ${VALUES_FILE}"
fi
echo ""

# Install PostgreSQL
echo "Installing PostgreSQL chart..."
HELM_CMD="helm upgrade -i -n ${NAMESPACE} ${NAME} oci://registry-1.docker.io/bitnamicharts/postgresql --version ${VERSION}"
if [[ -n "${VALUES_FILE}" ]]; then
  HELM_CMD="${HELM_CMD} -f ${VALUES_FILE}"
fi
eval "${HELM_CMD}"

echo ""
echo "Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql -n "${NAMESPACE}" --timeout=300s
echo "PostgreSQL installation completed."