package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeysLifecycle(t *testing.T) {
	for _, alg := range []string{"Ed25519", "ES256"} {
		t.Run(alg, func(t *testing.T) {
			dir := t.TempDir()
			pass := filepath.Join(dir, "passphrase")
			key := filepath.Join(dir, "signing.key")
			pub := filepath.Join(dir, "signing.pem")
			if err := os.WriteFile(pass, []byte("local test passphrase only\n"), 0600); err != nil {
				t.Fatal(err)
			}
			run := func(args ...string) (int, string, string) {
				t.Helper()
				var out, errs bytes.Buffer
				status := Run(args, &out, &errs)
				return status, out.String(), errs.String()
			}
			status, out, errs := run("keys", "generate", "--alg", alg, "--out", key, "--passphrase-file", pass)
			if status != 0 || errs != "" || !strings.Contains(out, "encrypted") {
				t.Fatalf("generate: %d %q %q", status, out, errs)
			}
			status, out, errs = run("keys", "inspect", "--file", key)
			if status != 0 || !strings.Contains(out, "alg="+alg) || errs != "" {
				t.Fatalf("inspect: %d %q %q", status, out, errs)
			}
			status, _, errs = run("keys", "public", "--key", key, "--passphrase-file", pass, "--out", pub)
			if status != 0 || errs != "" {
				t.Fatalf("public: %d %q", status, errs)
			}
			pem, err := os.ReadFile(pub)
			if err != nil || !bytes.Contains(pem, []byte("BEGIN PUBLIC KEY")) {
				t.Fatal("bad exported PEM", err)
			}
			status, _, _ = run("keys", "generate", "--alg", alg, "--out", key, "--passphrase-file", pass)
			if status == 0 {
				t.Fatal("overwrote existing key")
			}
		})
	}
}
