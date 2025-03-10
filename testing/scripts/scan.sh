#! /bin/bash
set -e

REPO_PATH=$1
shift

SONAR_HOST=$6
SONAR_PWD=$7
# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPT_DIR/get-token.sh"

# Set default value for SONAR_LOGIN if not provided
SONAR_LOGIN=$(get_token "$SONAR_HOST" "$SONAR_PWD")
if [ $? -ne 0 ]; then
    echo "Failed to get authentication token"
    exit 1
fi

echo "Scanning $REPO_PATH"

# use temp dir to avoid data race
TEMP_DIR=$(mktemp -d)
echo "Using temp dir: $TEMP_DIR"
cp -R "$REPO_PATH" "$TEMP_DIR/repo"

cd "$TEMP_DIR/repo"
COMMAND="$@"

if [ -n "$SONAR_HOST" ]; then
    COMMAND="$COMMAND -Dsonar.host.url=$SONAR_HOST"
fi
if [ -n "$SONAR_LOGIN" ]; then
    COMMAND="$COMMAND -Dsonar.login=$SONAR_LOGIN"
fi

eval "$COMMAND"

