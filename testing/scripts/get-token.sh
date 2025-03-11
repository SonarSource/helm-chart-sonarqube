#!/bin/bash

HOST=$1
PWD=$2

#根据密码文件获取token
get_token() {
    url="$HOST/api/user_tokens/generate"
    random_name="my-token-$(date +%s%N)"
    curl -s -X POST -u "admin:$PWD" "$url" -d "name=$random_name" | jq -r '.token'
}