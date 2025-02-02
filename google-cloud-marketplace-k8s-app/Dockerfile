ARG FROM=gcr.io/cloud-marketplace-tools/k8s/deployer_helm:0.12.2
FROM $FROM

ENV WAIT_FOR_READY_TIMEOUT 1200
ENV TESTER_TIMEOUT 1200

LABEL com.googleapis.cloudmarketplace.product.service.name=services/sonarqube-dce

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

COPY google-cloud-marketplace-k8s-app/data-test/schema.yaml /data-test/schema.yaml
# Provide registry prefix and tag for default values for images.
# Might not be needed on V2 schema
ARG REGISTRY
ARG TAG
RUN cat /data-test/schema.yaml \
        | env -i "REGISTRY=$REGISTRY" "TAG=$TAG" envsubst \
        > /data-test/schema.yaml.new \
        && mv /data-test/schema.yaml.new /data-test/schema.yaml
