# Run non-mutating formatting, vet, and test checks.
default:
    @unformatted="$(gofmt -l version.go version_test.go cmd/xenri internal/cli)"; if [ -n "$unformatted" ]; then printf 'Go files need gofmt:\n%s\n' "$unformatted" >&2; exit 1; fi
    go vet ./...
    go test ./...
    bash scripts/test-release-tag.sh

# Run the complete CI-oriented CLI validation suite.
ci: default
    #!/usr/bin/env bash
    set -euo pipefail

    go test -race ./...

    tmp="$(mktemp -d)"
    trap 'rm -rf -- "$tmp"' EXIT
    go build -o "$tmp/xenri" ./cmd/xenri
    test "$("$tmp/xenri" version)" = "xenri v$(cat VERSION)-dev"
    "$tmp/xenri" --help | grep -Fq 'MCP proxy operations are not implemented.'

    status=0
    "$tmp/xenri" proxy >"$tmp/proxy.stdout" 2>"$tmp/proxy.stderr" || status=$?
    test "$status" -eq 2
    test ! -s "$tmp/proxy.stdout"
    grep -Fq 'unknown command "proxy"' "$tmp/proxy.stderr"

# Validate release packaging, checksums and local version without publishing.
# Requires GoReleaser v2.18.3 on PATH; writes ignored dist/ output.
release:
    bash scripts/check-release.sh
