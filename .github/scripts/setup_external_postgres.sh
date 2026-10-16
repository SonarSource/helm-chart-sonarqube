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
# The chart's default "nano" preset caps memory at 192Mi, which the agentic scenario (SonarQube plus
# the Agent Orchestrator on the same database) gets OOMKilled at.
HELM_CMD="${HELM_CMD} --set primary.resourcesPreset=none"
HELM_CMD="${HELM_CMD} --set primary.resources.requests.cpu=100m,primary.resources.requests.memory=256Mi"
HELM_CMD="${HELM_CMD} --set primary.resources.limits.cpu=1,primary.resources.limits.memory=768Mi"
if [[ -n "${VALUES_FILE}" ]]; then
  HELM_CMD="${HELM_CMD} -f ${VALUES_FILE}"
fi
eval "${HELM_CMD}"

echo ""
echo "Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql -n "${NAMESPACE}" --timeout=300s
echo "PostgreSQL installation completed."