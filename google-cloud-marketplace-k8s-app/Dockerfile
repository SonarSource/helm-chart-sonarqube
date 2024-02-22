ARG FROM=gcr.io/cloud-marketplace-tools/k8s/deployer_helm:latest
FROM $FROM

COPY charts/sonarqube-dce /tmp/chart.tmp/sonarqube-dce
RUN cd /tmp \
        && mv chart.tmp/* chart \
        && tar -czvf /tmp/chart.tar.gz chart \
        && mkdir -p /data/chart \
        && mv chart.tar.gz /data/chart/ \
        && rm -Rf chart chart.tmp

COPY google-cloud-marketplace-k8s-app/schema.yaml /data/schema.yaml
# Provide registry prefix and tag for default values for images.
ARG REGISTRY
ARG TAG
RUN cat /data/schema.yaml \
        | env -i "REGISTRY=$REGISTRY" "TAG=$TAG" envsubst \
        > /data/schema.yaml.new \
        && mv /data/schema.yaml.new /data/schema.yaml
