#!/bin/bash

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update -y

# Install docker
apt-get install -y docker-ce docker-ce-cli containerd.io
systemctl start docker
systemctl enable docker

# Install chart-testing
YAMLLINT_VERSION=1.35.1
YAMALE_VERSION=5.2.1
CHART_TESTING_VERSION=3.11.0
CHART_TESTING_HASHSUM="38914f285ce309f86a876522640a27b57840a435d3480195f0878e104f1e8044"
curl -LO https://github.com/helm/chart-testing/releases/download/v${CHART_TESTING_VERSION}/chart-testing_${CHART_TESTING_VERSION}_linux_amd64.tar.gz
echo "${CHART_TESTING_HASHSUM}  chart-testing_${CHART_TESTING_VERSION}_linux_amd64.tar.gz" | sha256sum -c
tar -xf chart-testing_${CHART_TESTING_VERSION}_linux_amd64.tar.gz
chmod +x ./ct 
mkdir -p /etc/ct
mv etc/chart_schema.yaml /etc/ct/chart_schema.yaml
mv etc/lintconf.yaml /etc/ct/lintconf.yaml
mv ./ct /usr/bin/ct
rm chart-testing_${CHART_TESTING_VERSION}_linux_amd64.tar.gz
pip install "yamllint==${YAMLLINT_VERSION}"
pip install "yamale==${YAMALE_VERSION}"

# Install kubectl
KUBECTL_VERSION=1.31.0
curl -LO https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl
curl -LO https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl.sha256 
printf %s "  kubectl" >> kubectl.sha256
sha256sum -c kubectl.sha256
chmod +x ./kubectl
mv ./kubectl /usr/bin/kubectl
rm kubectl.sha256

# Install helm
HELM_VERSION=3.16.1
HELM_CHECKSUM="e57e826410269d72be3113333dbfaac0d8dfdd1b0cc4e9cb08bdf97722731ca9"
curl -LO https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz
echo "${HELM_CHECKSUM}  helm-v${HELM_VERSION}-linux-amd64.tar.gz" | sha256sum -c
tar -xf helm-v${HELM_VERSION}-linux-amd64.tar.gz
chmod +x ./linux-amd64/helm
mv ./linux-amd64/helm /usr/bin/helm
rm -rf ./linux-amd64
rm helm-v${HELM_VERSION}-linux-amd64.tar.gz

# Install kind
KIND_VERSION=0.24.0
KIND_CHECKSUM="b89aada5a39d620da3fcd16435b7f28d858927dd53f92cbac77686b0588b600d"
curl -LO https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64
echo "${KIND_CHECKSUM}  kind-linux-amd64" | sha256sum -c
chmod +x ./kind-linux-amd64
mv ./kind-linux-amd64 /usr/local/bin/kind

# Install artifacthub lint
AH_VERSION=1.19.0
AH_CHECKSUM="0e430493521ce387ca04d79b26646a86f92886dbcceb44985bb71082a9530ca5"
curl -LO https://github.com/artifacthub/hub/releases/download/v${AH_VERSION}/ah_${AH_VERSION}_linux_amd64.tar.gz
echo "${AH_CHECKSUM}  ah_${AH_VERSION}_linux_amd64.tar.gz" | sha256sum -c
tar -xf ah_${AH_VERSION}_linux_amd64.tar.gz
chmod +x ./ah
mv ./ah /usr/bin/ah
rm LICENSE
rm -rf ah_${AH_VERSION}_linux_amd64.tar.gz

docker --version
ct version
kubectl version --client
helm version
kind version