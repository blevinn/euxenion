// Package euxenion provides the source-tree version for development builds.
package euxenion

import (
	_ "embed"
	"strings"
)

// sourceVersion is the single source of truth for the version on this branch.
//go:embed VERSION
var sourceVersion string

// BaseVersion returns the version declared in the source tree, without a v prefix.
func BaseVersion() string {
	return strings.TrimSpace(sourceVersion)
}

// DevelopmentVersion identifies non-release builds made directly from the source.
func DevelopmentVersion() string {
	return "v" + BaseVersion() + "-dev"
}
