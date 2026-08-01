#!/usr/bin/env bash
# Create a prerelease tag once, or verify a retry is referring to the same commit.
# Usage: ensure-prerelease-tag.sh <tag> <expected-commit> <pr-number>
set -euo pipefail

tag=${1:?tag is required}
expected_commit=${2:?expected commit is required}
pr_number=${3:?PR number is required}

if [[ ! $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+-pre\.[0-9]+\.[0-9a-f]+$ ]]; then
  echo "Invalid prerelease tag: $tag" >&2
  exit 1
fi

expected_commit=$(git rev-parse --verify "${expected_commit}^{commit}")

verify_tag_target() {
  local actual_commit
  actual_commit=$(git rev-list -n 1 "${tag}^{commit}")
  if [[ $actual_commit != "$expected_commit" ]]; then
    printf 'Existing tag %s targets %s, expected merged PR commit %s\n' \
      "$tag" "$actual_commit" "$expected_commit" >&2
    exit 1
  fi
}

# Check the remote rather than only local tags: Actions retries run in fresh clones.
if git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null 2>&1; then
  git fetch origin "refs/tags/$tag:refs/tags/$tag"
  verify_tag_target
  echo "Tag $tag already exists at $expected_commit"
  exit 0
fi

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git tag -a "$tag" "$expected_commit" -m "Prerelease $tag for PR #$pr_number"
if git push origin "refs/tags/$tag"; then
  echo "Created tag $tag at $expected_commit"
  exit 0
fi

# A concurrent run may have created the tag after the remote check. Remove our
# local candidate, fetch the remote ref, and only accept it if it has the same
# target commit.
git tag -d "$tag" >/dev/null
git fetch origin "refs/tags/$tag:refs/tags/$tag"
verify_tag_target
echo "Tag $tag was created concurrently at $expected_commit"
