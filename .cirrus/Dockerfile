# ARG CIRRUS_AWS_ACCOUNT
# FROM ${CIRRUS_AWS_ACCOUNT}.dkr.ecr.eu-central-1.amazonaws.com/base:j11-latest as tools

FROM docker:20.10

USER root

ARG HELM_VERSION=3.13.2
ARG KUBECTL_VERSION=1.28.3
ARG CHART_TESTING_VERSION=3.10.1
ARG CHART_TESTING_HASHSUM="6f48140d92d4c4be52b31bf84a8b5d50fb815df721920d6751d4cbccdd3bac2b"
ARG KIND_VERSION=0.20.0
ARG KIND_HASHSUM="513a7213d6d3332dd9ef27c24dab35e5ef10a04fa27274fe1c14d8a246493ded"
ARG YAMLLINT_VERSION=1.33.0
ARG YAMALE_VERSION=4.0.4
ARG AH_VERSION=1.16.0
ARG AH_CHECKSUM="d5955b409361922f609c261bf4b124c4d8ecd294cf85e5fee11c2ba774d45366"
ARG YQ_VERSION=4.35.2
ARG YQ_CHECKSUM="8afd786b3b8ba8053409c5e7d154403e2d4ed4cf3e93c237462dc9ef75f38c8d"
ARG KUBECONFORM_VERSION=0.6.4
ARG KUBECONFORM_CHECKSUM="2b4ebeaa4d5ac4843cf8f7b7e66a8874252b6b71bc7cbfc4ef1cbf85acec7c07"

ENV HELM_BASE_URL="https://get.helm.sh"
ENV HELM_TAR_FILE="helm-v${HELM_VERSION}-linux-amd64.tar.gz"
ENV HELM_HASH_SUM_FILE="helm-v${HELM_VERSION}-linux-amd64.tar.gz.sha256sum"

RUN apk add --update --no-cache \
    ca-certificates \
    bash \
    jq \
    moreutils \
    curl \
    git \
    gnupg \
    libc6-compat \
    openssh-client \
    py3-pip \
    py3-wheel \
    python3 && \
    pip install --upgrade pip==21.0.1


RUN pip install "yamllint==${YAMLLINT_VERSION}" && \
    pip install "yamale==${YAMALE_VERSION}"

RUN set -eux; \
    curl -sL ${HELM_BASE_URL}/${HELM_TAR_FILE} -o ${HELM_TAR_FILE} ; \
    curl -sL ${HELM_BASE_URL}/${HELM_HASH_SUM_FILE} -o ${HELM_HASH_SUM_FILE} ;\
    sha256sum -c ${HELM_HASH_SUM_FILE} ; \
    tar -xf ${HELM_TAR_FILE} ; \
    mv linux-amd64/helm /usr/bin/helm ; \
    chmod +x /usr/bin/helm ; \
    rm -rf linux-amd64 ; \
    rm ${HELM_TAR_FILE} ; \
    rm ${HELM_HASH_SUM_FILE} 

RUN set -eux; \
    curl -LO https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl ; \
    curl -LO https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl.sha256 ; \
    printf %s "  kubectl" >> kubectl.sha256; \
    sha256sum -c kubectl.sha256; \
    chmod +x ./kubectl ; \
    mv ./kubectl /usr/bin/kubectl ; \
    rm kubectl.sha256

RUN set -eux; \
    curl -LO https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64 ; \
    echo "${KIND_HASHSUM}  kind-linux-amd64" | sha256sum -c ; \
    chmod +x ./kind-linux-amd64 ; \
    mv ./kind-linux-amd64 /usr/bin/kind

RUN set -eux; \
    curl -LO https://github.com/helm/chart-testing/releases/download/v${CHART_TESTING_VERSION}/chart-testing_${CHART_TESTING_VERSION}_linux_amd64.tar.gz ; \
    echo "${CHART_TESTING_HASHSUM}  chart-testing_${CHART_TESTING_VERSION}_linux_amd64.tar.gz" | sha256sum -c ; \
    tar -xf chart-testing_${CHART_TESTING_VERSION}_linux_amd64.tar.gz ; \
    chmod +x ./ct ; \
    mkdir -p /etc/ct ; \
    mv etc/chart_schema.yaml /etc/ct/chart_schema.yaml ; \
    mv etc/lintconf.yaml /etc/ct/lintconf.yaml ; \
    mv ./ct /usr/bin/ct ; \
    rm chart-testing_${CHART_TESTING_VERSION}_linux_amd64.tar.gz 

RUN set -eux; \
    curl -LO https://github.com/artifacthub/hub/releases/download/v${AH_VERSION}/ah_${AH_VERSION}_linux_amd64.tar.gz ; \
    echo "${AH_CHECKSUM}  ah_${AH_VERSION}_linux_amd64.tar.gz" | sha256sum -c ; \
    tar -xf ah_${AH_VERSION}_linux_amd64.tar.gz ; \
    chmod +x ./ah ; \
    mv ./ah /usr/bin/ah ; \
    rm LICENSE

RUN set -eux; \
    curl -LO https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_amd64 ; \
    echo "${YQ_CHECKSUM}  yq_linux_amd64" | sha256sum -c ; \
    chmod +x ./yq_linux_amd64 ; \
    mv ./yq_linux_amd64 /usr/bin/yq

RUN set -eux; \
    curl -LO https://github.com/yannh/kubeconform/releases/download/v${KUBECONFORM_VERSION}/kubeconform-linux-amd64.tar.gz ; \
    echo "${KUBECONFORM_CHECKSUM}  kubeconform-linux-amd64.tar.gz" | sha256sum -c ; \
    tar -xf kubeconform-linux-amd64.tar.gz ; \
    chmod +x ./kubeconform ; \
    mv ./kubeconform /usr/bin/kubeconform ; \
    rm kubeconform-linux-amd64.tar.gz

# COPY --from=tools /usr/local/bin/cirrus-env /usr/local/bin/cirrus-env
# COPY --from=tools /usr/local/bin/gh /usr/local/bin/gh
# COPY --from=tools /home/sonarsource/.m2/settings-public.xml /root/.m2/settings-public.xml