#!/bin/sh

set -euo pipefail

GH_PAGES_FOLDER="$(pwd)/../gh-pages"
git worktree add "$GH_PAGES_FOLDER" gh-pages
cp -rT "$1/" "$GH_PAGES_FOLDER/"
(cd "$GH_PAGES_FOLDER" && {
    helm repo index --url "https://github.com/$GITHUB_REPOSITORY/releases/download/$2/" --merge index.yaml .
    git add index.yaml
    git commit --message "Update index"
    git push
})
