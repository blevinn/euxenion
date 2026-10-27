package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blevinn/euxenion/internal/keys"
	"github.com/blevinn/euxenion/internal/signatures"
	"github.com/blevinn/euxenion/internal/trust"
)

func TestOfflineTicketWorkflow(t *testing.T) {
	for _, alg := range []signatures.Algorithm{signatures.Ed25519, signatures.ES256} {
		t.Run(string(alg), func(t *testing.T) {
			dir := t.TempDir()
			password := filepath.Join(dir, "password")
			keyFile := filepath.Join(dir, "issuer.key")
			pub := filepath.Join(dir, "issuer.pem")
			trustFile := filepath.Join(dir, "trust.json")
			requestFile := filepath.Join(dir, "request.json")
			ticket := filepath.Join(dir, "ticket.jws")
			if err := os.WriteFile(password, []byte("long temporary passphrase for tests\n"), 0600); err != nil {
				t.Fatal(err)
			}
			run := func(args ...string) (int, string, string) {
				t.Helper()
				var out, errs bytes.Buffer
				status := Run(args, &out, &errs)
				return status, out.String(), errs.String()
			}
			status, _, errs := run("keys", "generate", "--alg", string(alg), "--out", keyFile, "--passphrase-file", password)
			if status != 0 {
				t.Fatal("generate", errs)
			}
			status, _, errs = run("keys", "public", "--key", keyFile, "--passphrase-file", password, "--out", pub)
			if status != 0 {
				t.Fatal("export", errs)
			}
			pem, err := os.ReadFile(pub)
			if err != nil {
				t.Fatal(err)
			}
			_, _, kid, err := keys.ParsePublic(pem)
			if err != nil {
				t.Fatal(err)
			}
			cfg, _ := json.Marshal(trust.Config{Version: 1, Issuers: []trust.IssuerConfig{
				{ID: "issuer.external", Keys: []trust.KeyConfig{{ID: kid, Algorithm: alg, PublicFile: "issuer.pem"}}},
			}})
			if err := os.WriteFile(trustFile, cfg, 0644); err != nil {
				t.Fatal(err)
			}
			req := ticketRequest{Version: 1, Issuer: "issuer.external", Audience: "proxy.demo",
				Principal: "operator:one", Upstream: "upstream:one", Method: "tools/call",
				Tool: "write", Arguments: []byte(`{"message":"demo"}`),
				Claims: map[string]string{"approved_by": "operator"}, ValidForSeconds: 300}
			raw, _ := json.Marshal(req)
			if err := os.WriteFile(requestFile, raw, 0600); err != nil {
				t.Fatal(err)
			}
			status, out, errs := run("tickets", "sign", "--request", requestFile, "--key", keyFile, "--passphrase-file", password, "--output", ticket)
			if status != 0 || !strings.Contains(out, "not an MCP authorization") {
				t.Fatalf("sign: %d %s %s", status, out, errs)
			}
			status, out, errs = run("tickets", "verify", "--ticket", ticket, "--config", trustFile, "--request", requestFile)
			if status != 0 || errs != "" || !strings.Contains(out, "not authorized") {
				t.Fatalf("verify: %d %s %s", status, out, errs)
			}
			req.Arguments = []byte(`{"message":"changed"}`)
			raw, _ = json.Marshal(req)
			if err := os.WriteFile(requestFile, raw, 0600); err != nil {
				t.Fatal(err)
			}
			status, _, errs = run("tickets", "verify", "--ticket", ticket, "--config", trustFile, "--request", requestFile)
			if status == 0 || strings.Contains(errs, "message") {
				t.Fatalf("wrong binding verified or leaked args: %d %s", status, errs)
			}
		})
	}
}
