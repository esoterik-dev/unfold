#!/usr/bin/env bash
# One-time bootstrap for the first real Go module tag.
# Run this from an up-to-date checkout of main, inspect the tag, then push it:
#   ./scripts/release/create-initial-tag.sh v0.1.0
#   git show v0.1.0
#   git push origin v0.1.0
set -euo pipefail

version=${1:-v0.1.0}

if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Version must be a stable semver tag such as v0.1.0" >&2
  exit 1
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Refusing to tag with uncommitted changes" >&2
  exit 1
fi

if git rev-parse -q --verify "refs/tags/$version" >/dev/null; then
  echo "Tag $version already exists locally; refusing to overwrite it" >&2
  exit 1
fi

if git ls-remote --exit-code --tags origin "refs/tags/$version" >/dev/null 2>&1; then
  echo "Tag $version already exists on origin; refusing to overwrite it" >&2
  exit 1
fi

git tag -a "$version" -m "Initial stable module release $version" HEAD
echo "Created $version at $(git rev-parse --short HEAD). Review it, then run: git push origin $version"
