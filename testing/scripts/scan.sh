#! /bin/bash
set -e

REPO_PATH=$1
shift

echo "Scanning $REPO_PATH"

# use temp dir to avoid data race
TEMP_DIR=$(mktemp -d)
echo "Using temp dir: $TEMP_DIR"
cp -R "$REPO_PATH" "$TEMP_DIR/repo"

cd "$TEMP_DIR/repo"
eval "$@"
