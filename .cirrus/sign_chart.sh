#!/bin/sh

set -euo pipefail

echo $SONARSOURCE_SIGN_KEY_PASSPHRASE | gpg --batch --yes --passphrase-fd 0 --import /tmp/key

CURRENT_DIR=$(pwd)

for chart in $(find $CIRRUS_WORKING_DIR -maxdepth 1 -name "*.tgz*" -type f -exec basename "{}" ";"); do
    cd $CIRRUS_WORKING_DIR
    echo $SONARSOURCE_SIGN_KEY_PASSPHRASE | gpg --batch --yes --pinentry-mode loopback --passphrase-fd 0 --output $chart.asc --detach-sig $chart
    cd $CURRENT_DIR
done