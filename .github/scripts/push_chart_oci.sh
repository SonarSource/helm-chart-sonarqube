#!/bin/bash

set -euo pipefail

# Pushes the chart packages (*.tgz) attached to a published GitHub release to an OCI registry.
# The packages are pushed as they are, so the OCI layer is byte-identical to the release asset.
# Log in to the registry before running this script (helm registry login).
#
# Usage: push_chart_oci.sh <release-tag> <oci-repository> [helm push flags...]
#   e.g. push_chart_oci.sh sonarqube-2026.5.1005-sonarqube-dce-2026.5.1005 oci://ghcr.io/sonarsource/helm-chart-sonarqube

: "${GITHUB_REPOSITORY:?}"          # owner/repo that holds the release

RELEASE_TAG="${1:?Missing release tag}"
OCI_REPOSITORY="${2:?Missing OCI repository, e.g. oci://ghcr.io/owner/repo}"
shift 2

charts_dir=$(mktemp -d)
trap 'rm -rf "${charts_dir}"' EXIT

release=$(gh release view "${RELEASE_TAG}" --repo "${GITHUB_REPOSITORY}" --json isDraft,publishedAt)
if [[ $(jq -r '.isDraft' <<< "${release}") == "true" ]]; then
    echo "Release ${RELEASE_TAG} is a draft, refusing to push its charts." >&2
    exit 1
fi

gh release download "${RELEASE_TAG}" --repo "${GITHUB_REPOSITORY}" --dir "${charts_dir}" --pattern '*.tgz'

# helm push sets the manifest's org.opencontainers.image.created annotation from the archive's mtime.
# Use the release's publication time, so that re-runs and backfills push the same manifest digest.
touch -d "$(jq -r '.publishedAt' <<< "${release}")" "${charts_dir}"/*.tgz

for chart in "${charts_dir}"/*.tgz; do
    echo "Pushing $(basename "${chart}") to ${OCI_REPOSITORY}"
    helm push "${chart}" "${OCI_REPOSITORY}" "$@"
done
