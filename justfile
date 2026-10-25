# Euxenion developer commands. Run `just --list` for recipes.

# Build artifacts and binaries.
mod build

# Run validation and CI checks.
mod check

# Display available recipes.
default:
    @just --list

# Format Go sources in place.
fmt:
    gofmt -w cmd/xenri internal/cli
