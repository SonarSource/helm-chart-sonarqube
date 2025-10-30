# SonarQube Server - Azure Marketplace Bundle

This folder contains templates and definitions required to build the SonarQube Server Marketplace app.

# Contents

## sonarqube-azure

We define a wrapper chart of `charts/sonarqube`. This includes those settings that are required from the Azure marketplace (e.g., `global.azure.images`), which also activate specific features in the wrapped chart.

# How to test the metering service

## Generate usage events

In order to test if the SonarQube Server instance can send an usage event to the metering service, you can call the following endpoint and verify that it returns `{"success":true,"message":null}`.

```
curl -X POST "https://<SQS_URL>/api/v2/marketplace/azure/billing" \
  -u "<SQS_TOKEN>:" \
  -H "Content-Type: application/json"
```

## Verify that the metering service receives usage events

You need to perform the following actions from an Azure VM (e.g., from a pod in a k8s cluster on Azure).

- *Step 1* Get the CLIENT_ID by echoing the corresponding environment variable
- *Step 2* Retrieve the token using `curl -H "Metadata: true" "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&client_id=${CLIENT_ID}&resource=20e940b3-4c77-4b0b-9a53-9e16a1b010a7"`
- *Step 3* Verify that requests are submitted using `curl -H "Authorization: Bearer <Token from the previous request>" "https://marketplaceapi.microsoft.com/api/usageEvents?api-version=2018-08-31&usageStartDate=2025-10-09T06:00"`