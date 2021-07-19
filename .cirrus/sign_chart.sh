#!/bin/sh

set -euo pipefail

CURRENT_DIR=$(pwd)

for chart in $(find $CIRRUS_WORKING_DIR -maxdepth 1 -name "*.tgz*" -type f -exec basename "{}" ";"); do
    cd $CIRRUS_WORKING_DIR
    echo ${SONARSOURCE_SIGN_KEY_PASSPHRASE} | helm-sign --keyring ~/signing-keyring $chart
    cd $CURRENT_DIR
done