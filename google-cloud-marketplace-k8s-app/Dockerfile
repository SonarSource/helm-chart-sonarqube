ARG FROM=gcr.io/cloud-marketplace-tools/k8s/deployer_helm:13.0.10
FROM $FROM

ENV WAIT_FOR_READY_TIMEOUT 1200
ENV TESTER_TIMEOUT 1200

# The base image's bundled kubectl/helm are built with go1.26.5, which is flagged for
# CVE-2026-39821 (fixed in go1.26.6) by GCP Marketplace's package vulnerability scan.
# Upstream marketplace-k8s-app-tools hasn't cut a new tag with fixed binaries yet, so
# override them here. The old kubectl lives in a version-named directory
# (/opt/kubectl/<major.minor>), so it must be explicitly removed - just re-pointing the
# /opt/kubectl/default symlink would leave the vulnerable binary on disk at its own path.
# renovate: datasource=github-releases depName=kubectl-marketplace-deployer packageName=kubernetes/kubernetes extractVersion=^v(?<version>.*)$
ARG KUBECTL_VERSION=v1.37.0
ARG KUBECTL_CHECKSUM_NO_RENOVATE="6129359f4e1f3848a5572ccb0b26cf28b8ca08cef38c95a765b2f64a2c961a2f"
RUN OLD_KUBECTL_DIR="$(readlink -f /opt/kubectl/default)" \
        && curl --proto "=https" --fail -fsSL -o /tmp/kubectl "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" \
        && echo "${KUBECTL_CHECKSUM_NO_RENOVATE}  /tmp/kubectl" | sha256sum -c \
        && chmod 755 /tmp/kubectl \
        && rm -rf "$OLD_KUBECTL_DIR" \
        && mkdir -p "/opt/kubectl/${KUBECTL_VERSION#v}" \
        && mv /tmp/kubectl "/opt/kubectl/${KUBECTL_VERSION#v}/kubectl" \
        && ln -sfn "/opt/kubectl/${KUBECTL_VERSION#v}" /opt/kubectl/default \
        && /opt/kubectl/default/kubectl version --client

# Helm's latest stable release (v3.21.4/v4.2.4) is still built with go1.26.5. Only the
# v3.22.0 pre-release line has picked up a fixed Go toolchain so far. Pinning a Helm 3
# pre-release (rather than the "latest incl. pre-releases" Helm 4 the base image would
# otherwise install) avoids repeating the unvalidated Helm 3->4 jump from SONAR-30893.
# Revisit once a stable Helm 3.22.x (or later) release ships with go1.26.6+.
# renovate: datasource=github-releases depName=helm-marketplace-deployer packageName=helm/helm
ARG HELM_VERSION=v3.22.0-rc.1
ARG HELM_CHECKSUM_NO_RENOVATE="c64ef32b65487bb238f1a56990ca6ec3c68d0536f21d71679d14d239ae4caed0"
RUN curl --proto "=https" --fail -fsSL -o /tmp/helm.tar.gz "https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz" \
        && echo "${HELM_CHECKSUM_NO_RENOVATE}  /tmp/helm.tar.gz" | sha256sum -c \
        && tar -xzf /tmp/helm.tar.gz -C /tmp \
        && install -m 755 /tmp/linux-amd64/helm /usr/local/bin/helm \
        && rm -rf /tmp/helm.tar.gz /tmp/linux-amd64 \
        && helm version

COPY charts/sonarqube-dce /tmp/chart.tmp/sonarqube-dce
WORKDIR /tmp
RUN mv chart.tmp/* chart \
        && tar -czvf /tmp/chart.tar.gz chart \
        && mkdir -p /data/chart \
        && mv chart.tar.gz /data/chart/ \
        && rm -Rf chart chart.tmp

COPY google-cloud-marketplace-k8s-app/schema.yaml /data/schema.yaml
# Provide registry prefix and tag for default values for images.
# Might not be needed on V2 schema
ARG REGISTRY
ARG TAG
RUN cat /data/schema.yaml \
        | env -i "REGISTRY=$REGISTRY" "TAG=$TAG" envsubst \
        > /data/schema.yaml.new \
        && mv /data/schema.yaml.new /data/schema.yaml

# Test-only chart overlay (ephemeral PostgreSQL), merged into the chart only
# during the functionality test (create_manifests.sh --mode=test).
COPY google-cloud-marketplace-k8s-app/data-test/chart /tmp/test-chart.tmp/chart
RUN tar -czvf /tmp/test-chart.tar.gz -C /tmp/test-chart.tmp chart \
        && mkdir -p /data-test/chart \
        && mv /tmp/test-chart.tar.gz /data-test/chart/chart.tar.gz \
        && rm -Rf /tmp/test-chart.tmp

COPY google-cloud-marketplace-k8s-app/data-test/schema.yaml /data-test/schema.yaml
# Provide registry prefix and tag for default values for images.
# Might not be needed on V2 schema
ARG REGISTRY
ARG TAG
RUN cat /data-test/schema.yaml \
        | env -i "REGISTRY=$REGISTRY" "TAG=$TAG" envsubst \
        > /data-test/schema.yaml.new \
        && mv /data-test/schema.yaml.new /data-test/schema.yaml
