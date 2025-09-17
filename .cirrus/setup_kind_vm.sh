#!/bin/bash

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update -y

# Install docker
apt-get install -y docker-ce docker-ce-cli containerd.io
systemctl start docker
systemctl enable docker

# Install chart-testing (version 1.37.1 not yet compatible)
# renovate: datasource=pypi depName=yamllint-vm packageName=yamllint
YAMLLINT_VERSION=1.35.1
# renovate: datasource=pypi depName=yamale-vm packageName=yamale
YAMALE_VERSION=6.0.0
# renovate: datasource=github-release-attachments depName=chart-testing-vm packageName=helm/chart-testing
CHART_TESTING_VERSION=v3.13.0
CHART_TESTING_CHECKSUM="fcbae93a01887730054b5b0b4536b8cfbfe6010fdffccf66b8b87f5f764287d9"
CHART_TESTING_SEMVER=$(echo ${CHART_TESTING_VERSION} | sed 's/^v//')
curl -LO https://github.com/helm/chart-testing/releases/download/${CHART_TESTING_VERSION}/chart-testing_${CHART_TESTING_SEMVER}_linux_amd64.tar.gz
echo "${CHART_TESTING_CHECKSUM}  chart-testing_${CHART_TESTING_SEMVER}_linux_amd64.tar.gz" | sha256sum -c
tar -xf chart-testing_${CHART_TESTING_SEMVER}_linux_amd64.tar.gz
chmod +x ./ct 
mkdir -p /etc/ct
mv etc/chart_schema.yaml /etc/ct/chart_schema.yaml
mv etc/lintconf.yaml /etc/ct/lintconf.yaml
mv ./ct /usr/bin/ct
rm chart-testing_${CHART_TESTING_SEMVER}_linux_amd64.tar.gz
pip install "yamllint==${YAMLLINT_VERSION}"
pip install "yamale==${YAMALE_VERSION}"

# Install kubectl
# renovate: datasource=github-releases depName=kubectl-vm packageName=kubernetes/kubernetes
KUBECTL_VERSION=v1.34.0
curl -LO https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
curl -LO https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl.sha256 
printf %s "  kubectl" >> kubectl.sha256
sha256sum -c kubectl.sha256
chmod +x ./kubectl
mv ./kubectl /usr/bin/kubectl
rm kubectl.sha256

# Install helm
# renovate: datasource=github-releases depName=helm-vm packageName=helm/helm
HELM_VERSION=v3.18.2
curl -LO https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz
tar -xf helm-${HELM_VERSION}-linux-amd64.tar.gz
chmod +x ./linux-amd64/helm
mv ./linux-amd64/helm /usr/bin/helm
rm -rf ./linux-amd64
rm helm-${HELM_VERSION}-linux-amd64.tar.gz

# Install kind
# renovate: datasource=github-release-attachments depName=kind-vm packageName=kubernetes-sigs/kind
KIND_VERSION=v0.29.0
KIND_CHECKSUM="c72eda46430f065fb45c5f70e7c957cc9209402ef309294821978677c8fb3284"
curl -LO https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64
echo "${KIND_CHECKSUM}  kind-linux-amd64" | sha256sum -c
chmod +x ./kind-linux-amd64
mv ./kind-linux-amd64 /usr/local/bin/kind

# Install artifacthub lint
# renovate: datasource=github-release-attachments depName=ah-vm packageName=artifacthub/hub
AH_VERSION=v1.21.0
AH_CHECKSUM="48d6b87b60baf4ee8fd5efbfec3bf5fb3ca783ab3f1dab625e64332b95df2a84"
AH_SEMVER=$(echo ${AH_VERSION} | sed 's/^v//'); \
curl -LO https://github.com/artifacthub/hub/releases/download/${AH_VERSION}/ah_${AH_SEMVER}_linux_amd64.tar.gz
echo "${AH_CHECKSUM}  ah_${AH_SEMVER}_linux_amd64.tar.gz" | sha256sum -c
tar -xf ah_${AH_SEMVER}_linux_amd64.tar.gz
chmod +x ./ah
mv ./ah /usr/bin/ah
rm LICENSE
rm -rf ah_${AH_SEMVER}_linux_amd64.tar.gz

docker --version
ct version
kubectl version --client
helm version
kind version