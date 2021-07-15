#!/bin/sh

FILES="charts/*/Chart.yaml"

for f in $FILES
do
  echo "Updateing version of $f to append $BUILD_NUMBER"
  _original_version=`cat $f | yq e .version - `
  cat $f | yq e ".version = \"$_original_version+${BUILD_NUMBER}\"" - | sponge $f
done
