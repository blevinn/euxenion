# ADR 0001: v0.0.5 authorization ticket profile (draft normative v1)

Status: **proposed for human review**. This is a cryptographic-evidence profile, not an MCP authorization system.

## Security boundary

An untrusted client may present compact JWS evidence. Neither the client nor the payload authenticates the requesting principal. A caller identity must originate from a trusted transport or an independently authenticated local deployment context; the same applies to the configured upstream identity. A signed ticket from an explicitly trusted external or internal issuer can contribute **issuer-qualified evidence**. It does not grant permission to invoke a tool. Policy, human approval, durable redemption, replay defense and MCP interception are deferred to v0.1.0.

The attacker can control all message bytes, `_meta`, claimed identities, the algorithm and key IDs, ticket contents and replays; they may possess their own valid signing key. They cannot be assumed to possess trusted issuers' uncompromised private keys. Local administrators must protect the trusted issuer file, public-key files, signing key files and execution environment.

## Wire encoding: compact JWS (v1)

A ticket is exactly three nonempty **unpadded base64url** segments separated by two dots. The signing input is the original ASCII `base64url(protected) + "." + base64url(payload)`. No detached/unencoded payload, JWE, JWS JSON serialization, `crit`, `b64`, `zip`, `none`, nested JWT, or unprotected headers are accepted. Verification is over the original bytes, before interpreting claims.

Protected header is a strict JSON object containing **exactly**:
- `typ`: `"euxenion.ticket.v1"`;
- `alg`: `"Ed25519"` or `"ES256"` (deliberately **not** legacy JOSE `"EdDSA"`);
- `kid`: explicit trusted key identifier, 1–64 ASCII characters from `[A-Za-z0-9._-]`.

The payload is a strict JSON object containing exactly required `v`, `iss`, `aud`, `jti`, `iat`, `exp`, `inv`, `claims`, and optional `nbf`, `challenge_id`. Reject any unknown field and any duplicate object member *at any depth*. Required fields:
- `v`: JSON integer exactly `1`.
- `iss`: issuer identifier, 1–128 visible ASCII characters from `[A-Za-z0-9._:/-]`, matched exactly against configured trust. No origin-based or internal-issuer privilege.
- `aud`: a **single** nonempty target/proxy audience string with the same 128-character bound. An array, wildcard, glob, or implicit default is invalid.
- `jti`: random opaque ticket ID of at least 128 bits, encoded as 22 unpadded base64url characters, generated with a CSPRNG (other encodings rejected).
- `iat`, `exp`: JSON integer Unix epoch **seconds** (no floats or scientific notation). `exp > iat`; validity interval must be at most 24 hours.
- `nbf` (optional): Unix epoch seconds satisfying `iat <= nbf < exp`.
- `inv`: exact object `{"v":1,"sha256":"<43 unpadded base64url chars>"}`, the digest of the versioned invocation descriptor specified in #7. Digest must decode to exactly 32 bytes.
- `claims`: object with at most 32 issuer assertions. Claim names (1–64 chars) use `[a-z][a-z0-9_.-]*`; values are nonempty UTF-8 strings of at most 256 bytes. No nested objects, arrays or arbitrary JSON. Policy will receive the **verified issuer identifier** alongside these strings; it must not flatten issuer namespaces or regard them as authenticated caller identity.
- `challenge_id` (optional): an opaque 22-character unpadded base64url identifier (128-bit random) binding the evidence to an independently issued approval challenge. A challenge reference is **not** itself a verified approved state; v0.1.0 must validate it in durable state before acting.

The UTF-8 JSON must have no BOM or nonfinite numbers. Ticket size limit is 8 KiB, protected-header raw bytes <= 512, payload raw bytes <= 4096, and signatures must be the exact algorithm-specific size. Enforce size limits **before allocation-intensive decoding**. Header and payload JSON bytes need not be canonical: all distinct but valid encodings verify their original signing input. The invocation digest, by contrast, always uses the #7 canonicalization contract. Do not silently normalize identifiers, Unicode, claim names or audiences.

## Algorithm/key boundary

- `Ed25519`: RFC 8032 Ed25519 signature, 64 bytes, RFC 8032 public key 32 bytes; Ed25519 (not Ed25519ph or Ed25519ctx). In JOSE terms the explicit `alg` is this profile's `Ed25519`, **not** the generic `EdDSA`. Both sides must opt into this nonstandard identifier.
- `ES256`: P-256 ECDSA over SHA-256, using a **64-byte JWS R||S** signature, not ASN.1 DER. Require `1 <= R,S < curve order`. Keys must be P-256, not other curves.
- Algorithms are selected only from an explicit registry. The configured trusted issuer + `kid` entry contains an exact expected algorithm and public key. Verify header `alg` equals that trusted record. Never dynamically resolve a URL, load a key from header content, accept HMAC or fall back to a different algorithm.
- Public/private encoding and encrypted private-key handling are specified in #9/#10. Trust configuration is public-key-only (#11). Rotation uses *new `kid`*, overlapping trusted keys explicitly; removal of a key revokes it for **new verifications**. Already redeemed state and revocation are separate responsibilities.

## Verification API contract

The verifier is given: raw compact JWS, explicit trusted audience, trusted canonical invocation digest, trusted issuer registry, and an injected clock. It must:
1. Check limits and strict envelope/JSON schemas; reject duplicate keys including nested claims.
2. Choose the trust record by the **signed** `iss` and protected `kid` (an untrusted hint until signature verification), matching exact `alg` and key type; reject absent/disabled records.
3. Verify the cryptographic signature over original compact signing input.
4. Re-check audience and invocation binding using constant-time digest comparison; validate time with an injected UTC clock.
5. Return immutable *verified evidence* containing issuer ID, key ID, claims, audience, invocation digest, expiration, ticket ID and optional challenge ID. **Never** return `allow`, dispatch an MCP method, redeem `jti`, or interpret a ticket as policy permission.

Temporal rule: `iat` may not be more than **30 seconds in the future**; `nbf` (if present) may not be more than 30 seconds in the future; **expiration has no positive grace** (`now >= exp` fails). Reject zero/negative timestamps, overflow, and negative-duration intervals. Any internal I/O, parsing, trust lookup or cryptographic error fails closed. Error classes are `malformed`, `unsupported`, `untrusted`, `invalid_signature`, `invalid_binding`, `invalid_time`, and `internal`; external CLI output must not expose raw tokens or sensitive args.

## Canonical invocation dependency (#7)

Descriptor v1 must be constructed from **authenticated** principal and locally configured upstream, the MCP method and tool name, and the exact JSON tool arguments. No attacker-provided `principal`, `upstream`, ticket `_meta`, transient session timestamps, arrival order, request IDs or transport framing enter the digest. Canonical JSON follows RFC 8785 JCS or an explicitly reviewed equivalent profile that rejects ambiguous input and duplicate keys. The descriptor and digest version are never inferred from the JWS header.

## Anti-replay and policy limitations

Successful verification does not establish that a request is permitted or unused. A captured valid ticket can verify repeatedly until expiry; v0.1.0 must provide persistent one-time ticket redemption, exact challenge state checks, explicit deny precedence, issuer-qualified policy evaluation, and no automatic retries on ambiguous upstream outcomes. Broader grants require a **different explicitly versioned profile and positive policy**; v1 `inv` is always exact, and an arbitrary claim cannot waive digest checks.

## Example unsigned payload (illustrative; not a functional signed ticket)

```json
{
  "v": 1,
  "iss": "issuer.example",
  "aud": "proxy.production",
  "jti": "AQIDBAUGBwgJCgsMDQ4PEA",
  "iat": 1793000000,
  "exp": 1793000300,
  "inv": {"v": 1, "sha256": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
  "claims": {"approved_by": "operator"},
  "challenge_id": "ERITFRYXGBkaGxwdHh8gIQ"
}
```

Security review prerequisites for implementation: confirm canonicalization edge cases (#7), algorithm interoperability (#8), key trust and rotation (#11), and whether the selected nonstandard `Ed25519` JWS alg identifier meets intended external issuer needs. Any change to these normative fields requires an explicit profile version.
