#!/bin/bash
set -x

export SONAR_HOST=$1
export SONAR_USER=$2
export SONAR_PWD=$3
shift 3

bash -x scripts/scan.sh "$@"
