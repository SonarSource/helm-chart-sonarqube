#! /bin/bash

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPT_DIR/get-token.sh"

HOST=$1
PWD=$2
COMPONENT=$3
BRANCH=${4:-main}
MAX_RETRIES=${5:-20}



TOKEN=$(get_token "$HOST" "$PWD")
echo "获取到的Token是: $TOKEN"

# 获取 SonarQube 分析结果
get_response() {
    # 使用 curl 获取分析活动的 JSON 响应
    curl -s -u "$TOKEN": "$HOST/api/ce/activity?component=$COMPONENT&type=REPORT&branch=$BRANCH"
}

# Function to check if all tasks have status SUCCESS
check_all_success() {
    # Return failure if response is empty
    if [ -z "$response" ]; then
        return 1
    fi
    # Get the number of tasks
    length=$(echo "$response" | jq -e '.tasks | length') || return 1
    if [ "$length" -eq 0 ]; then
        return 1
    fi
    # Check if all task statuses are "SUCCESS"
    echo "$response" | jq -e '[.tasks[].status] | all(. == "SUCCESS")' >/dev/null
}

# 最大重试次数和间隔时间
SLEEP_INTERVAL=1

# 初始化重试计数器
retry_count=0

while true; do
    response=$(get_response)

    if check_all_success; then
        echo "所有任务的状态都是 SUCCESS。"
        exit 0
    else
        retry_count=$((retry_count + 1))
        if [ "$retry_count" -ge "$MAX_RETRIES" ]; then
            echo "存在任务的状态不是 SUCCESS，且已达到最大重试次数。"
            exit 1
        fi
        echo "任务尚未完成，等待 ${SLEEP_INTERVAL} 秒后重试（第 ${retry_count} 次）。"
        sleep "$SLEEP_INTERVAL"
    fi
done
