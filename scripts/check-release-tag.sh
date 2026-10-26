#!/usr/bin/env bash
# Enforce source/tag equality and restrict tag ancestry to main or its
# version-matched servicing branch. Run from the checked-out tag's root.
set -euo pipefail

tag="${1:?usage: bash scripts/check-release-tag.sh vMAJOR.MINOR.PATCH}"
if [[ ! "$tag" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  printf 'Invalid release tag: %s\n' "$tag" >&2
  exit 1
fi
minor_branch="release/v${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"
source_version="$(cat VERSION)"
if [[ "$source_version" != "${tag#v}" ]]; then
  printf 'VERSION (%s) must match tag (%s)\n' "$source_version" "$tag" >&2
  exit 1
fi

tag_commit="$(git rev-parse --verify -q "refs/tags/$tag^{commit}")" || {
  printf 'Release tag %s does not exist locally\n' "$tag" >&2
  exit 1
}
head_commit="$(git rev-parse HEAD)"
if [[ "$tag_commit" != "$head_commit" ]]; then
  echo 'The checkout must be at the exact release tag commit' >&2
  exit 1
fi

if git merge-base --is-ancestor "$head_commit" refs/remotes/origin/main; then
  echo "Release $tag comes from main"
elif git show-ref --verify --quiet "refs/remotes/origin/$minor_branch" &&
     git merge-base --is-ancestor "$head_commit" "refs/remotes/origin/$minor_branch"; then
  echo "Release $tag comes from $minor_branch"
else
  printf 'Release %s must be on main or %s\n' "$tag" "$minor_branch" >&2
  exit 1
fi
