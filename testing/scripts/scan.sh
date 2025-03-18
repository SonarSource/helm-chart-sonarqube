#! /bin/bash
set -x

REPO_PATH=$1
shift

echo "Scanning $REPO_PATH"

# use temp dir to avoid data race
TEMP_DIR=$(mktemp -d)
echo "Using temp dir: $TEMP_DIR"
cp -R "$REPO_PATH" "$TEMP_DIR/repo"

# Check if -Dsonar.login or -Dsonar.token is present in the array
SONAR_LOGIN_PRESENT=false
SONAR_TOKEN_PRESENT=false

for arg in "$@"; do
    if [[ "$arg" == -Dsonar.login* ]]; then
        SONAR_LOGIN_PRESENT=true
    elif [[ "$arg" == -Dsonar.token* ]]; then
        SONAR_TOKEN_PRESENT=true
    fi
done

# If neither is present, generate a token
if ! $SONAR_LOGIN_PRESENT && ! $SONAR_TOKEN_PRESENT; then
    echo "Generating token using get_token script"
    url="${SONAR_HOST}/api/user_tokens/generate?name=my-token-$(date +%s)"
    TOKEN=$(curl -s -k -X POST -u "${SONAR_USER}:${SONAR_PWD}" "$url" | jq -r '.token')
    if [ -z "$TOKEN" ]; then
        echo "Failed to generate token"
        exit 1
    fi
    set -- "$@" "-Dsonar.login=$TOKEN"
fi

cd "$TEMP_DIR/repo"
echo "Running command: $@"
eval "$@"
