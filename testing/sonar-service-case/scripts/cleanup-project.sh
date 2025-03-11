#! /bin/bash
set -e

HOST=$1
TOKEN=$2
COMPONENT=$3

curl -X POST "$HOST/api/projects/delete?project=$COMPONENT" -u "$TOKEN:"
