#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

# --- Configuration Variables ---
# IMPORTANT: Replace these placeholder values with your actual details.
# You might want to set these as environment variables or use a secure method
# for production environments.

# Version of the original SonarQube chart (e.g., 2025.3.0)
# This should match the version in charts/sonarqube/Chart.yaml
SONARQUBE_CHART_VERSION="2025.3.0" 

# Azure Container Registry (ACR) details
# This should match the 'registryServer' in your manifest.yaml
ACR_REGISTRY="<your_registry>.azurecr.io" 
ACR_USERNAME="${AZURE_ACR_USERNAME:-}" # Use environment variable or provide directly
ACR_PASSWORD="${AZURE_ACR_PASSWORD:-}" # Use environment variable or provide directly

# Application name from manifest.yaml (used for the CNAB bundle name)
APPLICATION_NAME="sonarqube"

# --- Script Start ---

echo "--- Starting Azure Marketplace K8s App Packaging Process ---"

cd ..
cd azure-marketplace-k8s-app

# 1. Clean up previous build artifacts
echo "1. Cleaning up old build artifacts..."
# Removes:
# - .cnab/ directory (where the bundle is built)
# - The packaged wrapper chart (if it was created previously)
# - The charts/ directory within the wrapper chart (containing the .tgz)
# - The Chart.lock file within the wrapper chart
rm -rf .cnab/ "${APPLICATION_NAME}-azure-${SONARQUBE_CHART_VERSION}" sonarqube-azure/charts/ sonarqube-azure/Chart.lock

# Ensure the wrapper chart's charts/ directory exists for unpacking
# mkdir -p sonarqube-azure/charts/

# 2. Navigate into the wrapper chart directory and update Helm dependencies
echo "2. Updating Helm dependencies for the wrapper chart (sonarqube-azure)..."
# This command will read sonarqube-azure/Chart.yaml and package the 'sonarqube'
# dependency (from ../charts/sonarqube) into sonarqube-azure/charts/sonarqube-${SONARQUBE_CHART_VERSION}.tgz
cd sonarqube-azure
helm dependency update
echo "Helm dependencies updated. Packaged subchart is now in sonarqube-azure/charts/."

# 3. Decompress the subchart for CPA validation
echo "3. Decompressing the SonarQube subchart for CPA validation..."
cd charts
tar -xzf "sonarqube-${SONARQUBE_CHART_VERSION}.tgz"
rm "sonarqube-${SONARQUBE_CHART_VERSION}.tgz"
echo "SonarQube subchart decompressed and .tgz removed."

# 4. Navigate back to the main offer directory
cd ../.. # Back to azure-marketplace-k8s-app/

# 5. Perform Docker login to your Azure Container Registry
echo "5. Logging into Azure Container Registry: ${ACR_REGISTRY}..."
if [ -z "${ACR_USERNAME}" ] || [ -z "${ACR_PASSWORD}" ]; then
  echo "WARNING: ACR_USERNAME or ACR_PASSWORD not set. Docker login might fail or prompt for credentials."
  docker login "${ACR_REGISTRY}"
else
  echo "${ACR_PASSWORD}" | docker login "${ACR_REGISTRY}" --username "${ACR_USERNAME}" --password-stdin
fi
echo "Docker login complete."

# 6. Run CPA verify within the container
echo "6. Running CPA verification (cpa verify)..."
# The -v ./:/data mounts the current directory (azure-marketplace-k8s-app) to /data inside the container.
# CPA commands will operate on files relative to /data.
docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock -v "$(pwd)":/data mcr.microsoft.com/container-package-app:latest cpa verify --directory /data
echo "CPA verification complete."

# 7. Run CPA buildbundle within the container
echo "7. Building the CPA bundle (cpa buildbundle)..."
# This creates the .cnab directory and the bundle file (e.g., sonarqube.cnab)
# in the current directory (mounted as /data in container).
DOCKER_BUILDKIT=1 docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock -v "$(pwd)":/data mcr.microsoft.com/container-package-app:latest cpa buildbundle --force --directory /data
echo "CPA bundle built successfully."
echo "CPA bundle pushed to ACR successfully!"

echo "--- Azure Marketplace K8s App Packaging Process Complete ---"