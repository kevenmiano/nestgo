#!/usr/bin/env bash
set -euo pipefail

tag="${1:?usage: verify-release-tag.sh vMAJOR.MINOR.PATCH}"
if [[ ! "$tag" =~ ^v(0|1)\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "::error::NestGo uses the root Go module path and accepts v0.x.y/v1.x.y tags; v2+ needs a /v2 module path."
  exit 1
fi

git fetch --no-tags origin +refs/heads/main:refs/remotes/origin/main
git fetch --no-tags origin "refs/tags/$tag:refs/tags/$tag"
commit="$(git rev-parse --verify "refs/tags/$tag^{commit}")"
if ! git merge-base --is-ancestor "$commit" refs/remotes/origin/main; then
  echo "::error::Tag $tag is not reachable from main"
  exit 1
fi

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  printf 'tag=%s\ncommit=%s\n' "$tag" "$commit" >> "$GITHUB_OUTPUT"
fi
echo "Validated release $tag at $commit (reachable from main)"
