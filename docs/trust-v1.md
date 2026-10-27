# Issuer trust registry v1

Trust configuration is strict, local JSON (not YAML), with exactly `v:1` and `issuers`. Each issuer has a unique `id` and a nonempty `keys` list containing `kid`, `alg` and `public_key` (a **relative** filename below the configuration directory). Example with **placeholder** ID; replace with the key ID printed by `xenri keys inspect`:

```json
{
  "v": 1,
  "issuers": [
    {
      "id": "issuer.example",
      "keys": [
        {"kid": "REPLACE_WITH_ACTUAL_22_CHAR_KID", "alg": "Ed25519", "public_key": "issuer.pem"}
      ]
    }
  ]
}
```

Only `Ed25519` and `ES256` are accepted. Each configured key ID must equal base64url(first 16 bytes of SHA-256(SPKI DER)) for the referenced, parseable Ed25519/P-256 public key; it is **not** an issuer identity by itself. Duplicate issuer names and duplicate IDs **within** one issuer fail loading. The same public `kid` is permitted for two distinct configured issuers, but lookups always require **both** the issuer name and key ID, plus an exact algorithm match.

Public keys are standard SPKI PEM; the trust registry has no signing keys. Invalid or unknown configuration keys, duplicate JSON names, unsupported algorithms/curves, unknown issuer/key/alg combinations, absent files, path traversal, symlink files and group/world-writable config or key files fail closed. Keep the directory and its parents administrator-controlled. No remote JWK/URL lookup, self-asserted issuer, implicit local-issuer privilege, or global-key-ID matching is available.

Rotate by adding a separate new key record and migrating issuers to sign with the new `kid`; revoke for **new** verifications by removing the old entry and reloading the registry. Existing approvals/redemptions, delayed revocation and policy are v0.1.0 concerns. Loading/verifying a trusted public key does not authorize execution.
