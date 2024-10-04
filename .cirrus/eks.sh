#!/bin/bash

set -xeuo pipefail

CIRRUS_OIDC_TOKEN=${CIRRUS_OIDC_TOKEN:?}
CIRRUS_CLUSTER_NAME=${CIRRUS_CLUSTER_NAME:?}
CIRRUS_BRANCH=${CIRRUS_BRANCH:?}
AWS_IAM_ROLE=HelmChartSonarQubeCICDRole

# This is the dev account
ROLE_AWS_ACCOUNT="460386131003"
CLUSTER_NAME="CirrusCI-8-dev"

echo "${CIRRUS_OIDC_TOKEN}" > /tmp/web_identity_token_file

mkdir -p ~/.aws
mkdir -p ~/.kube

CREDENTIALS=$(readlink -f ~/.aws/credentials)

touch "${CREDENTIALS}"
cat <<EOF > "${CREDENTIALS}"
[default]
region=eu-central-1
role_arn=arn:aws:iam::${ROLE_AWS_ACCOUNT}:role/${AWS_IAM_ROLE}
web_identity_token_file=/tmp/web_identity_token_file
EOF

aws eks update-kubeconfig --name "${CLUSTER_NAME}"
