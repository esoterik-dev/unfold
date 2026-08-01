#!/usr/bin/env bash
# Offline retry/idempotency tests for ensure-prerelease-tag.sh.
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
script="$repo_root/scripts/release/ensure-prerelease-tag.sh"
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

remote="$tmpdir/remote.git"
git init --bare -q "$remote"

git -C "$tmpdir" clone -q "$remote" first
git -C "$tmpdir/first" config user.name test
git -C "$tmpdir/first" config user.email test@example.invalid
git -C "$tmpdir/first" commit --allow-empty -qm initial
merged_commit=$(git -C "$tmpdir/first" rev-parse HEAD)
git -C "$tmpdir/first" push -q origin HEAD:main

tag=v0.1.1-pre.42.abc1234
(
  cd "$tmpdir/first"
  "$script" "$tag" "$merged_commit" 123
)

# A workflow retry gets a fresh checkout. It must validate the tag and succeed
# without trying to create it again.
git -C "$tmpdir" clone -q "$remote" retry
git -C "$tmpdir/retry" checkout -q "$merged_commit"
retry_output=$(
  cd "$tmpdir/retry"
  "$script" "$tag" "$merged_commit" 123
)
[[ $retry_output == "Tag $tag already exists at $merged_commit" ]]

# Do not silently accept a tag of the same name that targets another commit.
git -C "$tmpdir" clone -q "$remote" mismatch
git -C "$tmpdir/mismatch" config user.name test
git -C "$tmpdir/mismatch" config user.email test@example.invalid
git -C "$tmpdir/mismatch" commit --allow-empty -qm other
other_commit=$(git -C "$tmpdir/mismatch" rev-parse HEAD)
git -C "$tmpdir/mismatch" tag -a v0.1.2-pre.43.def5678 -m wrong "$other_commit"
git -C "$tmpdir/mismatch" push -q origin v0.1.2-pre.43.def5678

if (
  cd "$tmpdir/retry"
  "$script" v0.1.2-pre.43.def5678 "$merged_commit" 123
); then
  echo 'expected mismatched tag target to fail' >&2
  exit 1
fi

echo 'ensure-prerelease-tag retry tests passed'
