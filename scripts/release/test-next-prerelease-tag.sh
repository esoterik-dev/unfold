#!/usr/bin/env bash
# Offline tests for scripts/release/next-prerelease-tag.sh.
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
script="$repo_root/scripts/release/next-prerelease-tag.sh"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
cd "$tmpdir"
git init -q
git config user.name test
git config user.email test@example.invalid
git commit --allow-empty -qm initial
git tag v0.1.0

assert_tag() {
  local expected=$1
  shift
  local actual
  actual=$(env "$@" "$script")
  if [[ $actual != "$expected" ]]; then
    printf 'expected %s, got %s\n' "$expected" "$actual" >&2
    return 1
  fi
}

assert_tag v0.1.1-pre.42.abc1234 \
  PR_TITLE='fix: repair output' PR_LABELS='' RUN_NUMBER=42 SHORT_SHA=abc1234
assert_tag v0.2.0-pre.43.abc1234 \
  PR_TITLE='feat: add export' PR_LABELS='feature' RUN_NUMBER=43 SHORT_SHA=abc1234
assert_tag v1.0.0-pre.44.abc1234 \
  PR_TITLE='Breaking: alter CLI flags' PR_LABELS='' RUN_NUMBER=44 SHORT_SHA=abc1234
assert_tag v1.0.0-pre.45.abc1234 \
  PR_TITLE='docs: explanation' PR_LABELS='major' RUN_NUMBER=45 SHORT_SHA=abc1234

echo 'next-prerelease-tag tests passed'
