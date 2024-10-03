#!/bin/bash

set -eo pipefail

GCLOUD_CLI_VERSION=466.0.0
GCLOUD_CLI_CHECKSUM="ab7e256cb7e05f8ad2f4410cf33f2f9dcc3dbe0c3ed7b745f85c7d9793043e4d"
BASE_FOLDER="${BASE_FOLDER:-"/root/.gcp/cache"}"

export PATH=${BASE_FOLDER}/bin:${BASE_FOLDER}/google-cloud-sdk/bin:${PATH}

mkdir -p ${BASE_FOLDER}

curl -LO https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz
echo "${GCLOUD_CLI_CHECKSUM}  google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz" | sha256sum -c
tar -xf google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz
chmod +x ./google-cloud-sdk
mv ./google-cloud-sdk ${BASE_FOLDER}/google-cloud-sdk
rm -rf google-cloud-cli-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz

curl -o mpdev https://raw.githubusercontent.com/GoogleCloudPlatform/marketplace-k8s-app-tools/master/scripts/dev
chmod +x mpdev
mkdir -p ${BASE_FOLDER}/bin
mv mpdev ${BASE_FOLDER}/bin

gcloud components install gke-gcloud-auth-plugin
gcloud components install kubectl
gcloud --version
