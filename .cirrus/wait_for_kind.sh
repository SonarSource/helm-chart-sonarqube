#!/bin/bash

# CirrusCI timeout is 30 minutes.
# In case of errors, we need to know what went wrong so we can debug.
# That is why the timeout is set to 25 minutes (1500 seconds)
timeout=1500

# Set the interval in seconds to check for the cluster
interval=5

echo "Waiting for Kind cluster"

# Loop until the timeout is reached or the cluster is available
for ((i=0; i<$timeout; i+=$interval)); do
  # Check if the Kind cluster is available, by checking if the cluster-info error result is empty
  if [[ -z "$(kubectl cluster-info --context kind-kind 2>&1 > /dev/null)" ]]; then
    echo "Kind cluster is available!"
    exit 0
  fi

  # Wait for the interval before checking again
  sleep "${interval}"
done

# Timeout reached, Kind cluster is not available
echo "Timeout reached, Kind cluster is not available"
kubectl cluster-info --context kind-kind 2>&1
echo "-------------------------"
echo "Detailed logs:"
kubectl cluster-info dump --context kind-kind 2>&1
exit 1
