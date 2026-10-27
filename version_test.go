package euxenion

import (
	"regexp"
	"strings"
	"testing"
)

func TestSourceVersion(t *testing.T) {
	base := BaseVersion()
	if !regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`).MatchString(base) {
		t.Fatalf("invalid VERSION: %q", base)
	}
	if got := DevelopmentVersion(); got != "v"+base+"-dev" {
		t.Fatalf("DevelopmentVersion() = %q", got)
	}
	if strings.Contains(base, "\n") {
		t.Fatal("VERSION must be a single line")
	}
}
