#!/bin/bash

# Helper function to display usage information
show_help() {
  cat << EOF
PostgreSQL Backup Script for SonarQube Migration

DESCRIPTION:
    Creates a backup of PostgreSQL data from a Kubernetes pod for migration
    to an external PostgreSQL database when upgrading SonarQube Helm charts.

USAGE:
    $0 [OPTIONS] <postgres_service>

OPTIONS:
    -n namespace    Kubernetes namespace where the Helm chart is deployed (default: sonarqube)
    -u username     PostgreSQL username (default: sonarUser)  
    -p password     PostgreSQL password (default: sonarPass)
    -d database     PostgreSQL database name (default: sonarDB)
    -h, --help      Show this help message and exit

ARGUMENTS:
    postgres_service PostgreSQL service name (REQUIRED)

EXAMPLES:
    # Show help
    $0 --help

    # Basic usage with defaults
    $0 my-postgresql-service

    # With custom options
    $0 -n sonarqube -u sonarUser -p sonarPass -d sonarDB sonarqube-postgresql

    # Find PostgreSQL service first
    kubectl get svc -n sonarqube | grep postgresql
    $0 sonarqube-postgresql

OUTPUT:
    Creates sonarqube_backup_YYYYMMDD_HHMMSS.sql file in current directory

REQUIREMENTS:
    - kubectl configured and connected to cluster
    - Access to the specified Kubernetes namespace
    - PostgreSQL instance must be running and accessible

EOF

  return 0
}

# Default values
NAMESPACE="sonarqube"
USERNAME="sonarUser"
PASSWORD="sonarPass"
DATABASE_NAME="sonarDB"
POSTGRES_SERVICE=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -h|--help)
      show_help
      exit 0
      ;;
    -n)
      NAMESPACE="$2"
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
    -*)
      echo "Unknown option: $1"
      echo "Use -h or --help for usage information"
      exit 1
      ;;
    *)
      # This should be the postgres service name
      if [[ -z "$POSTGRES_SERVICE" ]]; then
        POSTGRES_SERVICE="$1"
      else
        echo "Error: Multiple service names specified. Only one service name is allowed." >&2
        exit 1
      fi
      shift
      ;;
  esac
done

echo "=== PostgreSQL Backup Script ==="
echo "Namespace: $NAMESPACE"
echo "Username: $USERNAME"
echo "Database: $DATABASE_NAME"
echo "PostgreSQL service: ${POSTGRES_SERVICE:-'(required)'}"
echo ""

# Step 1: Backup from source PostgreSQL
echo "Step 1: Backing up from source PostgreSQL..."

# Check if PostgreSQL service is provided
if [[ -z "$POSTGRES_SERVICE" ]]; then
  echo "Error: PostgreSQL service name is required" >&2
  echo "Usage: $0 [OPTIONS] <postgres_service>"
  echo "Find PostgreSQL service with: kubectl get svc -n $NAMESPACE | grep postgresql"
  echo "Use -h or --help for detailed usage information"
  exit 1
fi

echo "Using PostgreSQL service: $POSTGRES_SERVICE"

# Generate timestamped backup filename
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILENAME="sonarqube_backup_${TIMESTAMP}.sql"

echo "Starting backup using container..."

# Create backup job using PostgreSQL container
echo "Creating temporary backup pod..."
kubectl run postgresql-backup-pod --rm -i --restart=Never \
  --image=bitnamilegacy/postgresql:11.14.0 \
  --namespace=$NAMESPACE \
  --env="PGPASSWORD=$PASSWORD" \
  -- sh -c "pg_dump -h $POSTGRES_SERVICE -U $USERNAME $DATABASE_NAME" > $BACKUP_FILENAME

# Capture the exit code
BACKUP_EXIT_CODE=$?

echo "Backup pod completed"

# Validate backup file
if [[ $BACKUP_EXIT_CODE -ne 0 ]] || [[ ! -s $BACKUP_FILENAME ]]; then
  echo "Backup failed or file is empty"
  echo "Check service name and credentials, then try again"
  rm -f $BACKUP_FILENAME
  exit 1
fi

# Show backup info
BACKUP_SIZE=$(wc -c < $BACKUP_FILENAME)
BACKUP_LINES=$(wc -l < $BACKUP_FILENAME)
echo "Backup size: $(($BACKUP_SIZE / 1024))KB ($BACKUP_LINES lines)"

echo "Backup completed: $BACKUP_FILENAME"
echo ""

echo "=== Backup Complete ==="
echo ""
echo "Backup file created: $BACKUP_FILENAME"
echo "Backup size: $(($BACKUP_SIZE / 1024))KB ($BACKUP_LINES lines)"
echo ""
echo "=== External Database Setup Example ==="
echo "# 1. Create PostgreSQL instance (version 11.x+ recommended)"
echo "# 2. Configure network access/firewall rules"
echo "# 3. Create database and user:"
echo "#    CREATE DATABASE $DATABASE_NAME;"
echo "#    CREATE USER $USERNAME WITH PASSWORD '$PASSWORD';"
echo "#    GRANT ALL PRIVILEGES ON DATABASE $DATABASE_NAME TO $USERNAME;"
echo "# 4. Restore: PGPASSWORD=$PASSWORD psql -h <endpoint> -U $USERNAME -d $DATABASE_NAME < $BACKUP_FILENAME"
echo ""
echo "=== SonarQube Configuration for External Database ==="
echo "For external PostgreSQL connection, use in values.yaml:"
echo "jdbcOverwrite:"
echo "  enabled: true"
echo "  jdbcUrl: \"jdbc:postgresql://<endpoint>:5432/$DATABASE_NAME\""
echo "  jdbcUsername: \"$USERNAME\""
echo "  jdbcPassword: \"$PASSWORD\""



