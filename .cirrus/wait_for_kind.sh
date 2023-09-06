#!/bin/bash

# Set the timeout in seconds
timeout=3600

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
  sleep $interval
done

# Timeout reached, Kind cluster is not available
echo "Timeout reached, Kind cluster is not available"
echo "Error logs:"
kubectl cluster-info --context kind-kind 2>&1
exit 1
