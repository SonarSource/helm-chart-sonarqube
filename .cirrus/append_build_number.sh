#!/bin/sh

FILES="charts/*/Chart.yaml"

for f in $FILES
do
  echo "Updateing version of $f to append $BUILD_NUMBER"
  _original_version=`cat $f | yq r - version `
   yq w -i $f version "$_original_version+${BUILD_NUMBER}"
done
