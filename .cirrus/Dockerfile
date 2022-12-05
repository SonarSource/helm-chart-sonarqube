ARG CIRRUS_AWS_ACCOUNT
FROM ${CIRRUS_AWS_ACCOUNT}.dkr.ecr.eu-central-1.amazonaws.com/base:j11-latest as tools

FROM docker:20.10

USER root

ARG HELM_VERSION=3.6.3
ARG KUBECTL_VERSION=1.21.0
ARG CHART_TESTING_VERSION=3.4.0
ARG CHART_TESTING_HASHSUM="7a8275b001e41901a2b4d24463e951981082d3496447f59ff1d53e837ae31cb7"
ARG KIND_VERSION=0.11.1
ARG KIND_HASHSUM="949f81b3c30ca03a3d4effdecda04f100fa3edc07a28b19400f72ede7c5f0491"
ARG YAMLLINT_VERSION=1.26.1
ARG YAMALE_VERSION=3.0.6
ARG AH_VERSION=1.2.0
ARG AH_CHECKSUM="30d3b080c5e65530d6089813233f10a881820361e2b9b32fb7b744466ff74d27"
ARG YQ_VERSION=3.4.1
ARG YQ_CHECKSUM="adbc6dd027607718ac74ceac15f74115ac1f3caef68babfb73246929d4ffb23c"

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
    curl -LO https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 ; \
    echo "${YQ_CHECKSUM}  yq_linux_amd64" | sha256sum -c ; \
    chmod +x ./yq_linux_amd64 ; \
    mv ./yq_linux_amd64 /usr/bin/yq

COPY --from=tools /usr/local/bin/cirrus-env /usr/local/bin/cirrus-env
COPY --from=tools /usr/bin/gh /usr/local/bin/gh
COPY --from=tools /home/sonarsource/.m2/settings-public.xml /root/.m2/settings-public.xml