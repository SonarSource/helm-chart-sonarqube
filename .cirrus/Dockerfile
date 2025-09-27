ARG CIRRUS_AWS_ACCOUNT
FROM ${CIRRUS_AWS_ACCOUNT:-275878209202}.dkr.ecr.eu-central-1.amazonaws.com/base:j17-latest AS tools

FROM docker:27.4

USER root

##
# Note: the v in some version is a restriction for github-release-attachments datasource, where tags are expected to start with v.
##
# renovate: datasource=github-releases depName=helm packageName=helm/helm
ARG HELM_VERSION=v3.19.0
# renovate: datasource=github-releases depName=kubectl packageName=kubernetes/kubernetes
ARG KUBECTL_VERSION=v1.34.1
# renovate: datasource=pypi depName=yamllint packageName=yamllint
ARG YAMLLINT_VERSION=1.37.1
# renovate: datasource=pypi depName=yamale packageName=yamale
ARG YAMALE_VERSION=6.0.0
# renovate: datasource=github-release-attachments depName=chart-testing packageName=helm/chart-testing
ARG CHART_TESTING_VERSION=v3.13.0
ARG CHART_TESTING_CHECKSUM="fcbae93a01887730054b5b0b4536b8cfbfe6010fdffccf66b8b87f5f764287d9"
# renovate: datasource=github-release-attachments depName=yq packageName=mikefarah/yq
ARG YQ_VERSION=v4.47.2
ARG YQ_CHECKSUM="1bb99e1019e23de33c7e6afc23e93dad72aad6cf2cb03c797f068ea79814ddb0"
# renovate: datasource=github-release-attachments depName=kubeconform packageName=yannh/kubeconform
ARG KUBECONFORM_VERSION=v0.7.0
ARG KUBECONFORM_CHECKSUM="c31518ddd122663b3f3aa874cfe8178cb0988de944f29c74a0b9260920d115d3"
# renovate: datasource=github-release-attachments depName=chart-verified packageName=redhat-certification/chart-verifier
ARG CHART_VERIFIER_VERSION=1.13.13
ARG CHART_VERIFIER_CHECKSUM="5b19f0c3f7a6281e60d9d5391a5c365530f090bb217bcbf0b072e975107a425a"
#NO RENOVATE, this is not possible to fetch update automatically as there is no proper github release or json file available
ARG OPENSHIFT_CLI_VERSION=4.18.9
ARG OPENSHIFT_CLI_CHECKSUM="1e2d73c870756e3940dcb6c1112c7aa7f702a89cfdb992d11079ac852b4ea05c"

ENV HELM_BASE_URL="https://get.helm.sh"
ENV HELM_TAR_FILE="helm-${HELM_VERSION}-linux-amd64.tar.gz"
ENV HELM_HASH_SUM_FILE="helm-${HELM_VERSION}-linux-amd64.tar.gz.sha256sum"
ENV PATH=/usr/bin/google-cloud-sdk/bin:${PATH}

# renovate: datasource=pypi depName=pip packageName=pip
ENV PYTHON_PIP_VERSION=25.2

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
    pip install --break-system-packages --upgrade pip==${PYTHON_PIP_VERSION}

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
    curl -LO https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl ; \
    curl -LO https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl.sha256 ; \
    printf %s "  kubectl" >> kubectl.sha256; \
    sha256sum -c kubectl.sha256; \
    chmod +x ./kubectl ; \
    mv ./kubectl /usr/bin/kubectl ; \
    rm kubectl.sha256

RUN set -eux; \
    export CHART_TESTING_SEMVER=$(echo ${CHART_TESTING_VERSION} | sed 's/^v//'); \
    curl -LO https://github.com/helm/chart-testing/releases/download/${CHART_TESTING_VERSION}/chart-testing_${CHART_TESTING_SEMVER}_linux_amd64.tar.gz ; \
    echo "${CHART_TESTING_CHECKSUM}  chart-testing_${CHART_TESTING_SEMVER}_linux_amd64.tar.gz" | sha256sum -c ; \
    tar -xf chart-testing_${CHART_TESTING_SEMVER}_linux_amd64.tar.gz ; \
    chmod +x ./ct ; \
    mkdir -p /etc/ct ; \
    mv etc/chart_schema.yaml /etc/ct/chart_schema.yaml ; \
    mv etc/lintconf.yaml /etc/ct/lintconf.yaml ; \
    mv ./ct /usr/bin/ct ; \
    rm chart-testing_${CHART_TESTING_SEMVER}_linux_amd64.tar.gz 

RUN set -eux; \
    curl -LO https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 ; \
    echo "${YQ_CHECKSUM}  yq_linux_amd64" | sha256sum -c ; \
    chmod +x ./yq_linux_amd64 ; \
    mv ./yq_linux_amd64 /usr/bin/yq

RUN set -eux; \
    curl -LO https://github.com/yannh/kubeconform/releases/download/${KUBECONFORM_VERSION}/kubeconform-linux-amd64.tar.gz ; \
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
