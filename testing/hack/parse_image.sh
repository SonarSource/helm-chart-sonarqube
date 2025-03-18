#!/bin/bash

# Function to parse image reference
parse_image_reference() {
    local image_ref="$1"
    local full_repo tag digest repo_path

    # Extract digest if exists
    if [[ "$image_ref" == *"@"* ]]; then
        digest="${image_ref#*@}"
        image_ref="${image_ref%@*}"
    else
        digest=""
    fi

    # Extract tag if exists
    if [[ "$image_ref" == *":"* ]]; then
        tag="${image_ref#*:}"
        full_repo="${image_ref%:*}"
    else
        tag=""
        full_repo="$image_ref"
    fi

    # Extract repo path (everything after first /)
    repo_path="${full_repo#*/}"

    # Print results in a format suitable for eval
    echo "FULL_REPO=$full_repo"
    echo "REPO_PATH=$repo_path"
    echo "TAG=$tag"
    echo "DIGEST=$digest"
}
