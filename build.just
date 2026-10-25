# Build all currently supported targets.
default: cli

# Compile Xenri into dist/xenri.
cli:
    mkdir -p dist
    go build -o dist/xenri ./cmd/xenri
