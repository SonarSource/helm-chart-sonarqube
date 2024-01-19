#!/bin/bash

set -eoUu pipefail

if [[ -n "${DOCKER_USERNAME}" ]] && [[ -n "${DOCKER_PASSWORD}" ]]; then
    helm repo add clustersecret https://charts.clustersecret.io/
    helm install clustersecret clustersecret/cluster-secret -n clustersecret --create-namespace
    DOCKER_CONFIG=$(kubectl create secret docker-registry unused --docker-username=${DOCKER_USERNAME} --docker-password=${DOCKER_PASSWORD} --dry-run -o json | jq '.data.".dockerconfigjson"')
    sed -i "s|DOCKER_CONFIG_JSON|${DOCKER_CONFIG}|g" .cirrus/docker_hub_test_pull_secret.yaml
    kubectl apply -f .cirrus/docker_hub_test_pull_secret.yaml
fi

ct lint --config test.yaml --all
ct install --config test.yaml --all
