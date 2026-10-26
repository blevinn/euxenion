#!/usr/bin/env bash
# Validate release packaging without publishing or requiring credentials.
set -euo pipefail

goreleaser check
goreleaser release --snapshot --clean

# Exactly one archive for each of the four targets.
shopt -s nullglob
for os in linux darwin; do
  for arch in amd64 arm64; do
    archives=(dist/xenri_*_"$os"_"$arch".tar.gz)
    if (( ${#archives[@]} != 1 )); then
      printf 'Expected one %s/%s archive; found %s\n' "$os" "$arch" "${#archives[@]}" >&2
      exit 1
    fi
    tar -tzf "${archives[0]}" | grep -qx 'xenri'
  done
done

test -s dist/checksums.txt
(cd dist && shasum -a 256 -c checksums.txt)

host_os="$(go env GOOS)"
host_arch="$(go env GOARCH)"
host_archives=(dist/xenri_*_"$host_os"_"$host_arch".tar.gz)
if (( ${#host_archives[@]} != 1 )); then
  echo 'Missing host-platform archive for smoke test' >&2
  exit 1
fi
tmp="$(mktemp -d)"
trap 'rm -rf -- "$tmp"' EXIT
tar -xzf "${host_archives[0]}" -C "$tmp" xenri
version="$("$tmp/xenri" --version)"
if [[ "$version" != 'xenri v'* || "$version" == 'xenri dev' ]]; then
  printf 'Unexpected release version: %s\n' "$version" >&2
  exit 1
fi
printf 'Validated release snapshot (%s)\n' "$version"
