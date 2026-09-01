#!/bin/bash

set -eo pipefail

# renovate: datasource=github-tags depName=gcloud-cli-vm packageName=twistedpair/google-cloud-sdk
GCLOUD_CLI_VERSION=525.0.0
GCLOUD_CLI_CHECKSUM_NO_RENOVATE="75941a1017e233bf42f7d7240488ed29b42dd3f347a4e453ee3d505932d2c475"

# renovate: datasource=github-releases depName=mpdev-vm packageName=GoogleCloudPlatform/marketplace-k8s-app-tools
MPDEV_VERSION=13.0.10
MPDEV_CHECKSUM_NO_RENOVATE="c7cf77c990e7ed6cc7d99abfbd3b26528b146a03fab449fba9f284dda8d4fcc6"
BASE_FOLDER="${BASE_FOLDER:-"/root/.gcp/cache"}"

mkdir -p ${BASE_FOLDER}

curl --proto "=https" --fail -LO  https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz
echo "${GCLOUD_CLI_CHECKSUM_NO_RENOVATE}  google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz" | sha256sum -c
tar -xf google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz
chmod +x ./google-cloud-sdk
mv ./google-cloud-sdk ${BASE_FOLDER}/google-cloud-sdk
rm -rf google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz

gcloud components install gke-gcloud-auth-plugin kubectl

curl --proto "=https" --fail -LO https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/archive/refs/tags/${MPDEV_VERSION}.tar.gz
echo "${MPDEV_CHECKSUM_NO_RENOVATE}  ${MPDEV_VERSION}.tar.gz" | sha256sum -c
tar -xf ${MPDEV_VERSION}.tar.gz
mv marketplace-k8s-app-tools-${MPDEV_VERSION}/scripts/dev ${BASE_FOLDER}/mpdev
chmod +x ${BASE_FOLDER}/mpdev
ls -la ${BASE_FOLDER}/mpdev