package cli

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/blevinn/euxenion/internal/invocation"
	"github.com/blevinn/euxenion/internal/keys"
	"github.com/blevinn/euxenion/internal/strictjson"
	"github.com/blevinn/euxenion/internal/tickets"
	"github.com/blevinn/euxenion/internal/trust"
)

const ticketsUsage = `Usage:
  xenri tickets sign --request FILE --key FILE --passphrase-file FILE --output FILE
  xenri tickets verify --ticket FILE --config FILE --request FILE

Verification only checks cryptographic evidence. It NEVER authorizes MCP execution.
`

type ticketRequest struct {
	Version int `json:"v"`
	Issuer string `json:"iss"`
	Audience string `json:"aud"`
	Principal string `json:"principal"`
	Upstream string `json:"upstream"`
	Method string `json:"method"`
	Tool string `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
	Claims map[string]string `json:"claims"`
	ChallengeID string `json:"challenge_id,omitempty"`
	ValidForSeconds int64 `json:"valid_for_seconds"`
}

func readTicketRequest(path string) (ticketRequest, string, error) {
	var req ticketRequest
	data, err := os.ReadFile(path)
	if err != nil || len(data) > invocation.MaxArgumentsBytes {
		return req, "", tickets.ErrMalformed
	}
	if err = strictjson.Decode(data, &req, invocation.MaxArgumentsBytes); err != nil {
		return req, "", tickets.ErrMalformed
	}
	if req.Version != 1 || req.ValidForSeconds < 1 || req.ValidForSeconds > 86400 {
		return req, "", tickets.ErrMalformed
	}
	digest, err := invocation.Digest(invocation.Descriptor{
		Principal: req.Principal, Upstream: req.Upstream,
		Method: req.Method, Tool: req.Tool, Arguments: req.Arguments,
	})
	if err != nil {
		return req, "", tickets.ErrMalformed
	}
	return req, digest, nil
}

func runTickets(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprint(stdout, ticketsUsage)
		return 0
	}
	fs := flag.NewFlagSet("tickets "+args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	switch args[0] {
	case "sign":
		request := fs.String("request", "", "explicit operator request")
		keyFile := fs.String("key", "", "encrypted private key")
		passFile := fs.String("passphrase-file", "", "owner-only password file")
		output := fs.String("output", "", "owner-only ticket output")
		if fs.Parse(args[1:]) != nil || fs.NArg() != 0 || *request == "" || *keyFile == "" || *passFile == "" || *output == "" {
			fmt.Fprint(stderr, ticketsUsage)
			return 2
		}
		req, digest, err := readTicketRequest(*request)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: invalid ticket request")
			return 1
		}
		pass, err := keys.ReadPassphrase(*passFile)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: unsafe passphrase source")
			return 1
		}
		defer clear(pass)
		encoded, err := keys.ReadEncrypted(*keyFile)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: cannot read encrypted signing key")
			return 1
		}
		pair, err := keys.DecryptPrivate(encoded, pass)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: key authentication failed")
			return 1
		}
		jti := make([]byte, 16)
		if _, err = rand.Read(jti); err != nil {
			fmt.Fprintln(stderr, "xenri: ticket randomness unavailable")
			return 1
		}
		now := time.Now().UTC().Unix()
		payload := tickets.Payload{
			Version: 1, Issuer: req.Issuer, Audience: req.Audience,
			ID: base64.RawURLEncoding.EncodeToString(jti),
			Issued: now, Expires: now + req.ValidForSeconds,
			Invocation: tickets.Invocation{Version: 1, SHA256: digest},
			Claims: req.Claims, ChallengeID: req.ChallengeID,
		}
		token, err := tickets.Issue(payload, pair)
		if err == nil {
			err = keys.WriteExclusive(*output, []byte(token+"\n"), true)
		}
		if err != nil {
			fmt.Fprintln(stderr, "xenri: ticket issuance failed")
			return 1
		}
		fmt.Fprintln(stdout, "ticket evidence written; not an MCP authorization or replay-safe grant")
		return 0
	case "verify":
		ticketFile := fs.String("ticket", "", "owner-only ticket file")
		config := fs.String("config", "", "trusted public issuer configuration")
		request := fs.String("request", "", "trusted expected invocation and audience")
		if fs.Parse(args[1:]) != nil || fs.NArg() != 0 || *ticketFile == "" || *config == "" || *request == "" {
			fmt.Fprint(stderr, ticketsUsage)
			return 2
		}
		req, digest, err := readTicketRequest(*request)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: invalid expected invocation")
			return 1
		}
		reg, err := trust.LoadFile(*config)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: invalid issuer trust configuration")
			return 1
		}
		data, err := keys.ReadEncrypted(*ticketFile)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: cannot safely read ticket evidence")
			return 1
		}
		evidence, err := tickets.Verify(strings.TrimSuffix(string(data), "\n"), req.Audience, digest, reg, time.Now().UTC())
		if err != nil {
			fmt.Fprintln(stderr, "xenri: ticket evidence verification failed")
			return 1
		}
		if evidence.Issuer != req.Issuer {
			fmt.Fprintln(stderr, "xenri: unexpected ticket issuer")
			return 1
		}
		fmt.Fprintf(stdout, "cryptographically verified issuer=%s kid=%s; not authorized for MCP dispatch\n", evidence.Issuer, evidence.KeyID)
		return 0
	default:
		fmt.Fprint(stderr, ticketsUsage)
		return 2
	}
}
