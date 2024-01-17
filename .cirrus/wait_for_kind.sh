#!/bin/bash

# CirrusCI timeout is 30 minutes.
# In case of errors, we need to know what went wrong so we can debug.
# That is why the timeout is set to 25 minutes (1500 seconds)
timeout=1500

# Set the interval in seconds to wait for the cluster to be up
interval=200

# Set the default Kubernetes version to use
KUBE_VERSION="${KUBE_VERSION:-1.25.0}"

echo "Waiting for Kind cluster"

# Loop until the timeout is reached or the cluster is available
for ((i=0; i<$timeout; i+=$interval)); do
  # Check if the Kind cluster is available, by checking the return code.
  if kind create cluster --image "kindest/node:v${KUBE_VERSION}" --wait "${interval}"s -v 8; then
    echo "Kind cluster is available!"
    exit 0
  else
    echo "kind cluster creation failed, retrying"
    kind delete cluster > /dev/null 2>&1
  fi
done

# Timeout reached, Kind cluster is not available
echo "Timeout reached, Kind cluster is not available, exiting. Please read the logs on the above trace"
exit 1