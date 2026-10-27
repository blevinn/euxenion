# Signature primitives

The `internal/signatures` registry supports exactly two version-1 algorithms: `Ed25519` (RFC 8032 pure Ed25519) and `ES256` (P-256 ECDSA/SHA-256). Ed25519 is the Euxenion protected-JWS identifier, **not** the registered generic JOSE `EdDSA` name. This choice requires explicit interoperability agreement with third-party ticket issuers. Signatures cover the caller-provided original compact-JWS signing input bytes. ES256 signatures on the JWS wire are 64-byte unsigned big-endian `R || S`, not DER.

Sign/Verify reject unsupported/disabled algorithms, wrong key type or curve, malformed signatures and range-invalid ES256 components. These primitives do **not** parse tickets, consult the issuer trust registry, validate expiry/binding, authenticate principals, make policy decisions or prevent replay. Protected-header/trusted-key algorithm binding belongs to #11/#12.
