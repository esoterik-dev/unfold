#!/usr/bin/env bash
# Print the next valid Go-semver prerelease tag for a merged PR.
# Required environment: PR_TITLE, PR_LABELS, RUN_NUMBER.
# Optional environment: SHORT_SHA (defaults to the current commit's short SHA).
set -euo pipefail

pr_title=${PR_TITLE:-}
pr_labels=${PR_LABELS:-}
run_number=${RUN_NUMBER:?RUN_NUMBER must be set}
short_sha=${SHORT_SHA:-$(git rev-parse --short=7 HEAD)}

if [[ ! $run_number =~ ^[0-9]+$ ]]; then
  echo "RUN_NUMBER must contain only digits" >&2
  exit 1
fi
if [[ ! $short_sha =~ ^[0-9a-fA-F]+$ ]]; then
  echo "SHORT_SHA must be a hexadecimal git SHA prefix" >&2
  exit 1
fi

latest_tag=$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-version:refname | \
  grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n 1 || true)

if [[ -z $latest_tag ]]; then
  latest_tag=v0.0.0
fi

IFS=. read -r major minor patch <<<"${latest_tag#v}"
classification="${pr_title,,} ${pr_labels,,}"

if [[ $classification =~ (^|[^[:alnum:]])(major|breaking)([^[:alnum:]]|$) ]]; then
  major=$((major + 1))
  minor=0
  patch=0
elif [[ $classification =~ (^|[^[:alnum:]])(minor|feature)([^[:alnum:]]|$) ]]; then
  minor=$((minor + 1))
  patch=0
else
  patch=$((patch + 1))
fi

# pre.<run>.<sha> is semver-valid: identifiers are dot-separated and only use
# alphanumerics/hyphens. Go accepts this form as a module version prerelease.
printf 'v%s.%s.%s-pre.%s.%s\n' "$major" "$minor" "$patch" "$run_number" "${short_sha,,}"
