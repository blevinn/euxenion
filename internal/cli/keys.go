package cli

import (
	"fmt"
	"flag"
	"io"
	"os"

	"github.com/blevinn/euxenion/internal/keys"
	"github.com/blevinn/euxenion/internal/signatures"
)

const keysUsage = `Usage:
  xenri keys generate --alg Ed25519|ES256 --out FILE --passphrase-file FILE
  xenri keys inspect --file FILE
  xenri keys public --key FILE --passphrase-file FILE --out FILE

Keys are encrypted locally. Ticket verification and MCP enforcement are separate.
`

func runKeys(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprint(stdout, keysUsage)
		return 0
	}
	fs := flag.NewFlagSet("keys "+args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	switch args[0] {
	case "generate":
		alg := fs.String("alg", "", "Ed25519 or ES256")
		out := fs.String("out", "", "encrypted private key file")
		passfile := fs.String("passphrase-file", "", "private 0600 passphrase file")
		if fs.Parse(args[1:]) != nil || fs.NArg() != 0 || *out == "" || *passfile == "" || !signatures.Supported(signatures.Algorithm(*alg)) {
			fmt.Fprint(stderr, keysUsage)
			return 2
		}
		pass, err := keys.ReadPassphrase(*passfile)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: unsafe passphrase source")
			return 1
		}
		defer clear(pass)
		pair, err := keys.Generate(signatures.Algorithm(*alg))
		if err != nil {
			fmt.Fprintln(stderr, "xenri: key generation failed")
			return 1
		}
		encrypted, err := keys.EncryptPrivate(pair, pass)
		if err == nil {
			err = keys.WriteExclusive(*out, encrypted, true)
		}
		if err != nil {
			fmt.Fprintln(stderr, "xenri: could not securely write private key")
			return 1
		}
		fmt.Fprintf(stdout, "generated encrypted %s key kid=%s\n", pair.Algorithm, pair.ID)
		return 0
	case "inspect":
		file := fs.String("file", "", "key file")
		if fs.Parse(args[1:]) != nil || fs.NArg() != 0 || *file == "" {
			fmt.Fprint(stderr, keysUsage)
			return 2
		}
		data, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: cannot inspect key")
			return 1
		}
		alg, id, err := keys.InspectEncrypted(data)
		if err != nil {
			_, alg, id, err = keys.ParsePublic(data)
		}
		if err != nil {
			fmt.Fprintln(stderr, "xenri: unsupported or invalid key")
			return 1
		}
		fmt.Fprintf(stdout, "alg=%s kid=%s\n", alg, id)
		return 0
	case "public":
		keyfile := fs.String("key", "", "encrypted private key file")
		passfile := fs.String("passphrase-file", "", "private 0600 passphrase file")
		out := fs.String("out", "", "public key PEM output")
		if fs.Parse(args[1:]) != nil || fs.NArg() != 0 || *keyfile == "" || *passfile == "" || *out == "" {
			fmt.Fprint(stderr, keysUsage)
			return 2
		}
		password, err := keys.ReadPassphrase(*passfile)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: unsafe passphrase source")
			return 1
		}
		defer clear(password)
		data, err := keys.ReadEncrypted(*keyfile)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: cannot read encrypted key")
			return 1
		}
		pair, err := keys.DecryptPrivate(data, password)
		if err != nil {
			fmt.Fprintln(stderr, "xenri: key authentication failed")
			return 1
		}
		pem, err := keys.MarshalPublic(pair.Public)
		if err == nil {
			err = keys.WriteExclusive(*out, pem, false)
		}
		if err != nil {
			fmt.Fprintln(stderr, "xenri: cannot write public key")
			return 1
		}
		fmt.Fprintf(stdout, "exported %s public key kid=%s\n", pair.Algorithm, pair.ID)
		return 0
	default:
		fmt.Fprint(stderr, keysUsage)
		return 2
	}
}
