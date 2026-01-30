#!/bin/sh

set -euo pipefail

echo "Running as user: $(whoami) (UID: $(id -u), GID: $(id -g))"

GH_PAGES_FOLDER="$(pwd)/../gh-pages"

# Debug: Check .git directory permissions
echo "Current directory: $(pwd)"
echo ".git directory owner: $(ls -ld .git | awk '{print $3":"$4}')"
echo ".git/refs/heads owner: $(ls -ld .git/refs/heads | awk '{print $3":"$4}')"
echo "Can we write to .git/refs/heads? $(test -w .git/refs/heads && echo 'YES' || echo 'NO')"

# git worktree add "$GH_PAGES_FOLDER" gh-pages
# cp -rT "$1/" "$GH_PAGES_FOLDER/"
# (cd "$GH_PAGES_FOLDER" && {
#     helm repo index --url "https://github.com/$GITHUB_REPOSITORY/releases/download/$2/" --merge index.yaml .
#     git add index.yaml
#     git commit --message "Update index"
#     git push
# })
