# Offline ticket interoperability workflow

Xenri 0.0.5 can generate signing keys, mint strict compact-JWS evidence and independently verify it **offline** without a proxy, database or network connection. The reader/verifier accepts the same public trust-file schema for internal and external issuer keys.

1. Create a strong random passphrase with a trusted secret manager, store it in a **0600 file** outside the repository, and do not put its value in command arguments or test fixtures.
2. From a scratch directory with the CLI available, copy `testdata/tickets/cli-request.json` into `request.json`.
3. Generate keys and public PEM; populate a trust JSON file with the printed key ID and the **actual public filename**.
4. Issue a private evidence file, then verify independently using **only** public trust configuration, request descriptor and ticket.

```bash
xenri keys generate --alg Ed25519 --out issuer.key --passphrase-file ./passphrase
xenri keys public --key issuer.key --passphrase-file ./passphrase --out issuer.pem
xenri keys inspect --file issuer.pem
# Populate trust.json from testdata/tickets/cli-trust-template.json, replacing the kid.
xenri tickets sign --request request.json --key issuer.key --passphrase-file ./passphrase --output evidence.jws
xenri tickets verify --ticket evidence.jws --config trust.json --request request.json
```

For ES256, use `--alg ES256` and change the trust record to `"alg":"ES256"`. Signature interoperability uses the **original** compact-JWS signing input and raw 64-byte JWS ES256 R||S values. `Ed25519` is the explicitly named profile algorithm, not JOSE `EdDSA`; third-party issuers must deliberately use this profile. External issuers do **not** need private keys on the verifying machine.

The request file is an operator-supplied *explicit assertion* of issuer, audience, principal, upstream, MCP method/tool, arguments and issuer-qualified claims. This offline utility cannot authenticate the principal named in that file. A real server must compute its own digest from authenticated identity and configured upstream, not from attacker-controlled request fields. `verify` uses the request **as the operator's expected context** and checks the expected issuer too; it never returns permission to execute. Alter arguments, principal, upstream, audience or trusted public-key configuration to observe safe verification failure.

Evidence can verify more than once until expiry; signature success does not mean a ticket is approved, unused, scoped by policy, or executable. v0.1.0 must implement policy, approval state, atomic redemption and replay defense.
