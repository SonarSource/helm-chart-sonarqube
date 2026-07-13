#!/usr/bin/env bash

set -euo pipefail

# Environment variables with default values
NAME="${NAME:-external-minio}"
NAMESPACE="${NAMESPACE:-sonarqube}"
# Fixed credentials and bucket, matching the docker-compose harness's own convention
# (agentic-workflows/private/integration-harness) so a test-values file can reference known,
# stable values instead of a freshly generated secret each install.
ROOT_USER="${ROOT_USER:-minioadmin}"
ROOT_PASSWORD="${ROOT_PASSWORD:-minioadmin}"
DEFAULT_BUCKET="${DEFAULT_BUCKET:-agentic-jobs}"

# NOTE: unlike setup_external_postgres.sh, this does not install a Helm chart. Bitnami's `minio`
# chart (the natural equivalent of their `postgresql` chart) has had all of its free-tier image
# tags — including `:latest` — pulled from Docker Hub since their August 2025 policy change, so
# `bitnami/minio` is no longer pullable without a paid subscription. This deploys the official,
# freely available `minio/minio` image directly instead (the same image the docker-compose harness
# itself already builds from — agentic-workflows/private/integration-harness/local-infra/minio).

echo "Installing MinIO with the following configuration:"
echo "  Name: ${NAME}"
echo "  Namespace: ${NAMESPACE}"
echo "  Default Bucket: ${DEFAULT_BUCKET}"
echo ""

echo "Applying MinIO manifests..."
kubectl apply -n "${NAMESPACE}" -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: ${NAME}
type: Opaque
stringData:
  root-user: "${ROOT_USER}"
  root-password: "${ROOT_PASSWORD}"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${NAME}
  labels:
    app: ${NAME}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${NAME}
  template:
    metadata:
      labels:
        app: ${NAME}
    spec:
      containers:
        - name: minio
          image: minio/minio:latest
          args: ["server", "/data"]
          env:
            - name: MINIO_ROOT_USER
              valueFrom: {secretKeyRef: {name: ${NAME}, key: root-user}}
            - name: MINIO_ROOT_PASSWORD
              valueFrom: {secretKeyRef: {name: ${NAME}, key: root-password}}
          ports:
            - containerPort: 9000
          readinessProbe:
            httpGet: {path: /minio/health/ready, port: 9000}
            initialDelaySeconds: 5
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: ${NAME}
spec:
  selector:
    app: ${NAME}
  ports:
    - port: 9000
      targetPort: 9000
EOF

echo ""
echo "Waiting for MinIO to be ready..."
kubectl wait --for=condition=ready pod -l app="${NAME}" -n "${NAMESPACE}" --timeout=300s

echo ""
echo "Creating bucket '${DEFAULT_BUCKET}' (idempotent)..."
kubectl delete job "${NAME}-init" -n "${NAMESPACE}" --ignore-not-found
kubectl apply -n "${NAMESPACE}" -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${NAME}-init
spec:
  backoffLimit: 3
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: mc
          image: minio/mc:latest
          command:
            - sh
            - -c
            - "mc alias set local http://${NAME}:9000 ${ROOT_USER} ${ROOT_PASSWORD} && mc mb -p local/${DEFAULT_BUCKET}"
EOF
kubectl wait --for=condition=complete job/"${NAME}-init" -n "${NAMESPACE}" --timeout=120s
echo "MinIO installation completed."
