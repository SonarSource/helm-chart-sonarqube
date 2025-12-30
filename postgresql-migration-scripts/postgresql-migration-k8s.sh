#!/bin/bash

# Helper function to display usage information
show_help() {
  cat << EOF
PostgreSQL In-Cluster Migration Script for SonarQube

DESCRIPTION:
    Performs complete in-cluster PostgreSQL migration by installing a new PostgreSQL
    chart (Bitnami 10.15.0 with PostgreSQL 11.14.0) and migrating data directly between database instances within Kubernetes.

USAGE:
    $0 [OPTIONS] <source_service>

OPTIONS:
    -s source_ns    Source namespace (default: sonarqube)
    -t target_ns    Target namespace (default: sonarqube)
    -u username     PostgreSQL username (default: sonarUser)
    -p password     PostgreSQL password (default: sonarPass)
    -d database     Database name (default: sonarDB)
    -r release      New PostgreSQL release name (default: postgresql-external)
    -f values_file  Optional custom values.yaml file for PostgreSQL chart
    -h, --help      Show this help message and exit

ARGUMENTS:
    source_service  Source PostgreSQL service name (REQUIRED)

EXAMPLES:
    # Show help
    $0 --help

    # Basic migration with defaults (must specify source service)
    $0 sonarqube-postgresql

    # Custom migration with different namespaces
    $0 -s my-source-ns -t my-target-ns sonarqube-postgresql

    # Full custom migration
    $0 -s source-ns -t target-ns -u sonarUser -p sonarPass -d sonarDB -r postgres-new -f values.yaml source-svc

    # Find PostgreSQL services
    kubectl get svc -n <namespace> | grep postgresql

WORKFLOW:
    1. Installs new PostgreSQL chart (Bitnami 10.15.0 with PostgreSQL 11.14.0)
    2. Creates migration job that directly pipes data between databases
    3. Verifies migration by comparing table counts
    4. Provides JDBC configuration for SonarQube

OUTPUT:
    - New PostgreSQL instance in target namespace
    - JDBC configuration for SonarQube values.yaml
    - Migration verification results

REQUIREMENTS:
    - kubectl configured and connected to cluster
    - helm installed and configured
    - Access to both source and target namespaces
    - Source PostgreSQL instance must be running and accessible

EOF

  return 0
}

# Default values
SOURCE_NAMESPACE="sonarqube"
TARGET_NAMESPACE="sonarqube"
USERNAME="sonarUser"
PASSWORD="sonarPass"
DATABASE_NAME="sonarDB"
NEW_RELEASE_NAME="postgresql-external"
SOURCE_SERVICE=""
VALUES_FILE=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -h|--help)
      show_help
      exit 0
      ;;
    -s)
      SOURCE_NAMESPACE="$2"
      shift 2
      ;;
    -t)
      TARGET_NAMESPACE="$2"
      shift 2
      ;;
    -u)
      USERNAME="$2"
      shift 2
      ;;
    -p)
      PASSWORD="$2"
      shift 2
      ;;
    -d)
      DATABASE_NAME="$2"
      shift 2
      ;;
    -r)
      NEW_RELEASE_NAME="$2"
      shift 2
      ;;
    -f)
      VALUES_FILE="$2"
      shift 2
      ;;
    -*)
      echo "Unknown option: $1"
      echo "Use -h or --help for usage information"
      exit 1
      ;;
    *)
      # This should be the source service name
      if [[ -z "$SOURCE_SERVICE" ]]; then
        SOURCE_SERVICE="$1"
      else
        echo "Error: Multiple service names specified. Only one source service name is allowed." >&2
        exit 1
      fi
      shift
      ;;
  esac
done

# Validate required parameters
if [[ -z "$SOURCE_SERVICE" ]]; then
  echo "Error: Source PostgreSQL service name is required" >&2
  echo "Usage: $0 [OPTIONS] <source_service>"
  echo "Find PostgreSQL service with: kubectl get svc -n $SOURCE_NAMESPACE | grep postgresql"
  echo "Use -h or --help for detailed usage information"
  exit 1
fi

echo "=== PostgreSQL In-Cluster Migration Script ==="
echo "Source namespace: $SOURCE_NAMESPACE"
echo "Target namespace: $TARGET_NAMESPACE"  
echo "Source service: $SOURCE_SERVICE"
echo "Target service: $NEW_RELEASE_NAME"
echo "New release name: $NEW_RELEASE_NAME"
echo "Mode: In-cluster (no local files)"
echo ""

# Step 1: Install new PostgreSQL chart first (version 10.15.0)
echo "Step 1: Installing new PostgreSQL chart (version 10.15.0)..."

# Create namespace if it doesn't exist
kubectl create namespace $TARGET_NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Add bitnami repo
helm repo add bitnami-legacy https://raw.githubusercontent.com/bitnami/charts/archive-full-index/bitnami
helm repo update

# Install PostgreSQL with custom values
echo "Installing PostgreSQL chart..."
if [[ -n "$VALUES_FILE" ]] && [[ -f "$VALUES_FILE" ]]; then
  echo "Using custom values file: $VALUES_FILE"
  helm upgrade --install $NEW_RELEASE_NAME bitnami-legacy/postgresql \
    --version 10.15.0 \
    --namespace $TARGET_NAMESPACE \
    --values "$VALUES_FILE" \
    --wait --timeout=300s
else
  helm upgrade --install $NEW_RELEASE_NAME bitnami-legacy/postgresql \
    --version 10.15.0 \
    --namespace $TARGET_NAMESPACE \
    --set image.registry=docker.io \
    --set image.repository=bitnamilegacy/postgresql \
    --set image.tag=11.14.0 \
    --set postgresqlUsername=$USERNAME \
    --set postgresqlPassword=$PASSWORD \
    --set postgresqlDatabase=$DATABASE_NAME \
    --set persistence.enabled=true \
    --set persistence.size=8Gi \
    --set commonLabels.purpose=backup \
    --set commonLabels.source=sonarqube-migration \
    --wait --timeout=300s
fi

if [[ $? -ne 0 ]]; then
  echo "Failed to install PostgreSQL chart"
  exit 1
fi

echo "PostgreSQL chart installed successfully"

# Step 2: Create single migration job that does backup and restore
echo ""
echo "Step 2: Creating migration job..."
echo "Using source service: $SOURCE_SERVICE"
echo "Using target service: $NEW_RELEASE_NAME"

cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: postgresql-migration-job
  namespace: $TARGET_NAMESPACE
  labels:
    purpose: "migration"
    source: "sonarqube-migration"
spec:
  template:
    metadata:
      labels:
        purpose: "migration"
        source: "sonarqube-migration"
    spec:
      restartPolicy: Never
      containers:
      - name: migrate
        image: "bitnamilegacy/postgresql:11.14.0"
        env:
        - name: PGPASSWORD
          value: "$PASSWORD"
        - name: SOURCE_HOST
          value: "$SOURCE_SERVICE.$SOURCE_NAMESPACE.svc.cluster.local"
        - name: TARGET_HOST
          value: "$NEW_RELEASE_NAME"
        command:
        - /bin/bash
        - -c
        - |
          echo "=== PostgreSQL Migration ==="
          
          # Wait for source database
          echo "Waiting for source database..."
          until pg_isready -h \$SOURCE_HOST -p 5432 -U $USERNAME; do
            echo "Still waiting for source..."
            sleep 5
          done
          echo "Source database ready"
          
          # Wait for target database  
          echo "Waiting for target database..."
          until pg_isready -h \$TARGET_HOST -p 5432 -U $USERNAME; do
            echo "Still waiting for target..."
            sleep 5
          done
          echo "Target database ready"
          
          # Direct pipe: backup and restore in one command
          echo "Starting direct migration (backup → restore)..."
          pg_dump -h \$SOURCE_HOST -U $USERNAME -d $DATABASE_NAME | \
          psql -h \$TARGET_HOST -U $USERNAME -d $DATABASE_NAME
          
          if [[ \${PIPESTATUS[0]} -ne 0 ]]; then
            echo "Backup failed"
            exit 1
          fi
          
          if [[ \${PIPESTATUS[1]} -ne 0 ]]; then
            echo "Restore failed"
            exit 1
          fi
          
          echo "Migration completed successfully"
          
          # Verify migration
          echo "Verifying migration..."
          SOURCE_TABLES=\$(psql -h \$SOURCE_HOST -U $USERNAME -d $DATABASE_NAME -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';" | xargs)
          TARGET_TABLES=\$(psql -h \$TARGET_HOST -U $USERNAME -d $DATABASE_NAME -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';" | xargs)
          
          echo "Source tables: \$SOURCE_TABLES"
          echo "Target tables: \$TARGET_TABLES"
          
          if [[ "\$SOURCE_TABLES" != "\$TARGET_TABLES" ]]; then
            echo "Table count mismatch - migration verification failed"
            exit 1
          fi
          
          echo "Table count matches - migration successful"
EOF

echo "Waiting for migration job to complete..."
kubectl wait --for=condition=complete job/postgresql-migration-job -n $TARGET_NAMESPACE --timeout=600s

if [[ $? -ne 0 ]]; then
  echo "Migration job failed, checking logs..."
  kubectl logs -n $TARGET_NAMESPACE job/postgresql-migration-job
else
  echo "Migration completed successfully"
fi

# Step 3: Output connection details
echo ""
echo "=== Migration Complete ==="
echo ""
echo "JDBC Connection Details:"
echo "------------------------"
JDBC_URL="jdbc:postgresql://$NEW_RELEASE_NAME.$TARGET_NAMESPACE.svc.cluster.local:5432/$DATABASE_NAME"

echo "JDBC URL: $JDBC_URL"
echo "Username: $USERNAME"
echo "Password: $PASSWORD"
echo "Database: $DATABASE_NAME"
echo ""
echo "For SonarQube values.yaml, use:"
echo "jdbcOverwrite:"
echo "  enabled: true"
echo "  jdbcUrl: \"$JDBC_URL\""
echo "  jdbcUsername: \"$USERNAME\""
echo "  jdbcPassword: \"$PASSWORD\""
echo ""

# Clean up
echo "Cleaning up migration job..."
kubectl delete job postgresql-migration-job -n $TARGET_NAMESPACE >/dev/null 2>&1

echo "Cleanup completed"
echo ""
echo "In-cluster migration completed successfully!"
echo "Data has been migrated directly between databases."