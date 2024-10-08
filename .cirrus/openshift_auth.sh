#!/bin/bash

set -xeuo pipefail

ROSA_OPENSHIFT_URL=${ROSA_OPENSHIFT_URL:?}
ROSA_OPENSHIFT_USER=${ROSA_OPENSHIFT_USER:?}
ROSA_OPENSHIFT_PASSWORD=${ROSA_OPENSHIFT_PASSWORD:?}

oc login "${ROSA_OPENSHIFT_URL}" --username "${ROSA_OPENSHIFT_USER}" --password "${ROSA_OPENSHIFT_PASSWORD}"
