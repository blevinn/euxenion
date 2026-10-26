package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/blevinn/euxenion"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantStatus    int
		wantOut       string
		wantErrSubstr string
	}{
		{name: "no arguments", wantOut: "Usage:\n"},
		{name: "help", args: []string{"help"}, wantOut: "Commands:\n"},
		{name: "help flag", args: []string{"--help"}, wantOut: "MCP proxy operations are not implemented."},
		{name: "short help flag", args: []string{"-h"}, wantOut: "Usage:\n"},
		{name: "version", args: []string{"version"}, wantOut: "xenri " + euxenion.DevelopmentVersion() + "\n"},
		{name: "version flag", args: []string{"--version"}, wantOut: "xenri dev\n"},
		{name: "unimplemented proxy", args: []string{"proxy"}, wantStatus: 2, wantErrSubstr: `unknown command "proxy"`},
		{name: "unknown command", args: []string{"bogus"}, wantStatus: 2, wantErrSubstr: `unknown command "bogus"`},
		{name: "extra help argument", args: []string{"help", "unknown"}, wantStatus: 2, wantErrSubstr: "help does not accept arguments"},
		{name: "extra version argument", args: []string{"version", "extra"}, wantStatus: 2, wantErrSubstr: "version does not accept arguments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := Run(tt.args, &stdout, &stderr)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
			if !strings.Contains(stdout.String(), tt.wantOut) {
				t.Errorf("stdout = %q, missing %q", stdout.String(), tt.wantOut)
			}
			if tt.wantErrSubstr == "" {
				if stderr.Len() != 0 {
					t.Errorf("stderr = %q, want empty", stderr.String())
				}
			} else if !strings.Contains(stderr.String(), tt.wantErrSubstr) {
				t.Errorf("stderr = %q, missing %q", stderr.String(), tt.wantErrSubstr)
			}
			if tt.wantStatus != 0 && stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty on error", stdout.String())
			}
		})
	}
}
