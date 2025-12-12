#!/bin/bash

set -eo pipefail

GCLOUD_CLI_VERSION=495.0.0
GCLOUD_CLI_CHECKSUM="5e76f6dae80e4eb07cdca607793a461162fd8d433b23ec2cc90403f686584044"
MPDEV_VERSION=0.12.4
MPDEV_CHECKSUM="fcc8aed037f9e3d79561d6658305ec38a30f29732ea7a89d128b5ab3bee490e6"
BASE_FOLDER="${BASE_FOLDER:-"/root/.gcp/cache"}"

mkdir -p ${BASE_FOLDER}

curl -LO --proto "=https" https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz
echo "${GCLOUD_CLI_CHECKSUM}  google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz" | sha256sum -c
tar -xf google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz
chmod +x ./google-cloud-sdk
mv ./google-cloud-sdk ${BASE_FOLDER}/google-cloud-sdk
rm -rf google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz

gcloud components install gke-gcloud-auth-plugin kubectl --quiet

curl -LO --proto "=https" https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/archive/refs/tags/${MPDEV_VERSION}.tar.gz
echo "${MPDEV_CHECKSUM}  ${MPDEV_VERSION}.tar.gz" | sha256sum -c
tar -xf ${MPDEV_VERSION}.tar.gz
mv marketplace-k8s-app-tools-${MPDEV_VERSION}/scripts/dev ${BASE_FOLDER}/mpdev
chmod +x ${BASE_FOLDER}/mpdev
