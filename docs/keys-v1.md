# Key format and lifecycle v1

The offline Xenri key commands support `Ed25519` (RFC 8032) and `ES256` (P-256). Public keys use a **single** standard SubjectPublicKeyInfo `-----BEGIN PUBLIC KEY-----` PEM block, suitable for third-party issuer trust. The deterministic key ID is unpadded base64url of the **first 16 bytes of SHA-256 of the canonical DER SubjectPublicKeyInfo**, regardless of issuer. It is a lookup identifier, not a trust assertion; different issuers may independently use the same `kid`.

Private key files have a v1 Euxenion JSON envelope (`v`, `alg`, `kid`, `kdf`, `iterations`, `salt`, `cipher`, `nonce`, `ciphertext`). PKCS#8 DER is protected by PBKDF2-HMAC-SHA256 (600,000 iterations, random 16-byte salt) and AES-256-GCM (random 12-byte nonce, 128-bit authentication tag). Fields are unpadded base64url. AEAD additional authenticated data is the UTF-8 concatenation `euxenion.encrypted-pkcs8.v1:` + algorithm + `:` + key ID. Readers bound iteration counts and ciphertext sizes before running KDF. The design is interoperable by following the above explicit representation; it is **not standard encrypted PKCS#8 PEM/PBES2** and must not be misidentified as such. A separate #10 ADR addresses storage threat limits and operational hardening.

```bash
# Create a private, non-world-readable passphrase file using your own
# secure local method. Never put a secret directly into a shell argument.
chmod 600 ./local-passphrase
xenri keys generate --alg Ed25519 --out ./issuer.key --passphrase-file ./local-passphrase
xenri keys inspect --file ./issuer.key
xenri keys public --key ./issuer.key --passphrase-file ./local-passphrase --out ./issuer.pem
xenri keys inspect --file ./issuer.pem
```

`generate` refuses overwrites, writes new private files at 0600, and prints only the algorithm and public key ID. `public` never prints private material, only writes a public PEM. `inspect` displays non-secret metadata. File passphrases must be 12–1024 bytes and restricted to owner access. The passphrase file is a deliberate non-interactive convenience; keep it out of the repository and logs.

Private keys do not belong in MCP proxy trust configuration or SQLite. Imported/unencrypted private keys are **not loaded as signing keys** by these commands. Local malware, privileged processes, key substitution and exposure from a compromised terminal/workspace are outside the protection of file permissions and at-rest encryption.
