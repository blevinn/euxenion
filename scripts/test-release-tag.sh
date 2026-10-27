#!/usr/bin/env bash
# Exercise release-tag validation with disposable local Git history.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf -- "$tmp"' EXIT

git -C "$tmp" init --quiet -b main
git -C "$tmp" config user.name "Release Test"
git -C "$tmp" config user.email "release-test@example.invalid"
printf '0.0.1\n' > "$tmp/VERSION"
git -C "$tmp" add VERSION
git -C "$tmp" commit --quiet -m initial
main_sha="$(git -C "$tmp" rev-parse HEAD)"
git -C "$tmp" update-ref refs/remotes/origin/main "$main_sha"
git -C "$tmp" tag -a v0.0.1 -m initial

(cd "$tmp" && bash "$root/scripts/check-release-tag.sh" v0.0.1)
if (cd "$tmp" && bash "$root/scripts/check-release-tag.sh" v0.0.2 >/dev/null 2>&1); then
  echo 'Missing tags must be rejected' >&2
  exit 1
fi
printf '0.0.2\n' > "$tmp/VERSION"
if (cd "$tmp" && bash "$root/scripts/check-release-tag.sh" v0.0.1 >/dev/null 2>&1); then
  echo 'Mismatched VERSION must be rejected' >&2
  exit 1
fi
git -C "$tmp" restore VERSION

git -C "$tmp" switch --quiet -c release/v0.0
printf '0.0.2\n' > "$tmp/VERSION"
git -C "$tmp" add VERSION
git -C "$tmp" commit --quiet -m 'service v0.0.2'
service_sha="$(git -C "$tmp" rev-parse HEAD)"
git -C "$tmp" update-ref refs/remotes/origin/release/v0.0 "$service_sha"
git -C "$tmp" tag -a v0.0.2 -m 'service release'
(cd "$tmp" && bash "$root/scripts/check-release-tag.sh" v0.0.2)

git -C "$tmp" switch --quiet -c feature
printf '0.0.3\n' > "$tmp/VERSION"
git -C "$tmp" add VERSION
git -C "$tmp" commit --quiet -m 'unmerged work'
git -C "$tmp" tag -a v0.0.3 -m 'unmerged tag'
if (cd "$tmp" && bash "$root/scripts/check-release-tag.sh" v0.0.3 >/dev/null 2>&1); then
  echo 'Unmerged feature branch tags must be rejected' >&2
  exit 1
fi
printf '0.1.0\n' > "$tmp/VERSION"
git -C "$tmp" add VERSION
git -C "$tmp" commit --quiet -m 'wrong minor'
git -C "$tmp" tag -a v0.1.0 -m 'wrong minor branch'
if (cd "$tmp" && bash "$root/scripts/check-release-tag.sh" v0.1.0 >/dev/null 2>&1); then
  echo 'Wrong servicing branch must be rejected' >&2
  exit 1
fi
echo 'Release tag verification fixtures passed'
