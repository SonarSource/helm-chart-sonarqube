ARG CIRRUS_AWS_ACCOUNT
FROM ${CIRRUS_AWS_ACCOUNT:-275878209202}.dkr.ecr.eu-central-1.amazonaws.com/base:j11-latest AS tools

FROM docker:27.4

USER root

ARG HELM_VERSION=3.16.3
ARG KUBECTL_VERSION=1.32.0
ARG CHART_TESTING_VERSION=3.11.0
ARG CHART_TESTING_HASHSUM="38914f285ce309f86a876522640a27b57840a435d3480195f0878e104f1e8044"
ARG YAMLLINT_VERSION=1.35.1
ARG YAMALE_VERSION=5.2.1
ARG YQ_VERSION=4.44.5
ARG YQ_CHECKSUM="638c4b251c49201fc94b598834b715f8f1c6e9b1854d2820772d2c79f0289002"
ARG KUBECONFORM_VERSION=0.6.7
ARG KUBECONFORM_CHECKSUM="95f14e87aa28c09d5941f11bd024c1d02fdc0303ccaa23f61cef67bc92619d73"
ARG CHART_VERIFIER_VERSION=1.13.8
ARG CHART_VERIFIER_CHECKSUM="1ed190a3aa4f961269e59f9596408415761ab368d761a08eae2960a84d957352"
ARG OPENSHIFT_CLI_VERSION=4.17.8
ARG OPENSHIFT_CLI_CHECKSUM="f2dc55e02e53bb6140d6c79ce74b66f685f74f48dd3f368b2cde3674a4710185"

ENV HELM_BASE_URL="https://get.helm.sh"
ENV HELM_TAR_FILE="helm-v${HELM_VERSION}-linux-amd64.tar.gz"
ENV HELM_HASH_SUM_FILE="helm-v${HELM_VERSION}-linux-amd64.tar.gz.sha256sum"
ENV PATH=/usr/bin/google-cloud-sdk/bin:${PATH}

RUN apk add --update --no-cache \
    aws-cli \
    bash \
    ca-certificates \
    curl \
    gcompat \
    git \
    gnupg \
    go \
    jq \
    libc6-compat \
    libstdc++ \
    moreutils \
    openssh-client \
    py3-pip \
    py3-wheel \
    python3 && \
    pip install --break-system-packages --upgrade pip==24.2

RUN pip install --break-system-packages "yamllint==${YAMLLINT_VERSION}" && \
    pip install --break-system-packages "yamale==${YAMALE_VERSION}"

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

RUN set -eux; \
    curl -LO https://github.com/redhat-certification/chart-verifier/releases/download/${CHART_VERIFIER_VERSION}/chart-verifier-${CHART_VERIFIER_VERSION}.tgz ; \
    echo "${CHART_VERIFIER_CHECKSUM}  chart-verifier-${CHART_VERIFIER_VERSION}.tgz" | sha256sum -c ; \
    tar -xf chart-verifier-${CHART_VERIFIER_VERSION}.tgz ; \
    chmod +x ./chart-verifier ; \
    mv ./chart-verifier /usr/bin/chart-verifier ; \
    rm chart-verifier-${CHART_VERIFIER_VERSION}.tgz

RUN set -eux; \
    curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${OPENSHIFT_CLI_VERSION}/openshift-client-linux.tar.gz ; \
    echo "${OPENSHIFT_CLI_CHECKSUM}  openshift-client-linux.tar.gz" | sha256sum -c ; \
    tar -xf openshift-client-linux.tar.gz ; \
    chmod +x ./oc ; \
    mv ./oc /usr/bin/oc ; \
    rm openshift-client-linux.tar.gz kubectl README.md

COPY --from=tools /usr/local/bin/cirrus-env /usr/local/bin/cirrus-env
COPY --from=tools /usr/local/bin/gh /usr/local/bin/gh
COPY --from=tools /home/sonarsource/.m2/settings-public-auth.xml /root/.m2/settings-public.xml
