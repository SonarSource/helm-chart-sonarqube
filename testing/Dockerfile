FROM docker-mirrors.alauda.cn/library/golang:1.23.6-bookworm AS builder

WORKDIR /tools
RUN mkdir -p /tools/bin && chmod -R 777 /tools/bin

ARG YQ_VERSION=4.25.2
ARG KUBECTL_VERSION=1.28.2
ARG HELM_VERSION=3.12.3
ARG JQ_VERSION=1.7

RUN set -eux; \
    if [ "$(arch)" = "arm64" ] || [ "$(arch)" = "aarch64" ]; then \
    export ARCH="arm64"; \
    export ARCH_ALIAS="arm64"; \
    else \
    export ARCH="amd64"; \
    export ARCH_ALIAS="x86_64"; \
    fi; \
    mkdir -p tmp; \
    curl -sfL https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_${ARCH} -o ./bin/yq && \
    curl -sfL https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl -o ./bin/kubectl && \
    curl -sfL https://get.helm.sh/helm-v${HELM_VERSION}-linux-${ARCH}.tar.gz | tar xzf - -C tmp 2>&1 && mv tmp/linux-${ARCH}/helm ./bin && \
    curl -sfL https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 -o ./bin/jq && \
    chmod +x ./bin/* && \
    rm -rf tmp && \
    ./bin/yq --version && \
    ./bin/kubectl version --client && \
    ./bin/helm version && \
    ./bin/jq --version


COPY testing /app
# 安装 playwright
ENV GOPROXY='https://build-nexus.alauda.cn/repository/golang/,direct'
RUN set -eux; \
    cd /app && \
    CGO_ENABLED=0 go test -c -o /tools/bin/sonarqube-e2e.test ./

RUN set -eux; \
    cd /app/sonar-service-case && \
    CGO_ENABLED=0 go test -c -o /tools/bin/sonar-service-case.test ./

FROM registry.alauda.cn:60070/ops/ubuntu:24.04

RUN mkdir -p /tools/bin
COPY --from=builder /tools/bin /tools/bin


RUN apt-get update && apt-get install -y ca-certificates tzdata bash locales make curl wget \
    && rm -rf /var/lib/apt/lists/* \
    && locale-gen en_US.UTF-8 \
    && locale-gen zh_CN.UTF-8 \
    && update-locale \
    curl -O https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-6.2.1.4610-linux-x64.zip; \
    unzip sonar-scanner-cli-6.2.1.4610-linux-x64.zip; \
    rm sonar-scanner-cli-6.2.1.4610-linux-x64.zip; \
    rm -rf /var/lib/apt/lists/*

ENV PATH="${PATH}:/app/sonar-scanner-6.2.1.4610-linux-x64/bin"

COPY . /app

WORKDIR /app/charts/sonarqube
RUN /tools/bin/helm dependency update

WORKDIR /app/testing
ENV PATH=/tools/bin:$PATH
ENV CI=true
ENV LANG=zh_CN.UTF-8
ENV LC_ALL=zh_CN.UTF-8
ENV REPORT=allure
ENV TAGS=sonarqube-e2e
ENV TEST_COMMAND="sonarqube-e2e.test"

ENTRYPOINT ["sonarqube-e2e.test"]
CMD ["--godog.concurrency=2", "--godog.format=allure", "--godog.tags=sonarqube-e2e"]