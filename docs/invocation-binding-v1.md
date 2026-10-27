# Invocation binding v1

The `invocation` package defines an **independent, strict deterministic JSON profile**, not complete RFC 8785. It serializes the versioned object `{"v":1,"principal":...,"upstream":...,"method":...,"tool":...,"arguments":...}` with UTF-8, lexicographically sorted object keys, JSON-escaped strings and no insignificant whitespace. Numbers must already be in canonical ECMAScript-compatible decimal form and round-trip exactly to IEEE-754 double precision; duplicate object keys, invalid Unicode and surrogate escapes, alternative number spellings, excess depth and overlarge arguments are rejected. No key order, whitespace or permitted Unicode escape difference can affect the digest. The profile is deliberately narrower than full JCS: it does **not** promise to accept every otherwise valid JSON request.

Digest is `base64url(SHA-256(canonical-bytes))`, without padding; `inv.v` is 1. The digest binds an authenticated principal, a locally configured upstream, the MCP method/tool, and the *entire* tool arguments value. A tool argument named `_meta` is a real argument and is bound. Request **transport** `_meta`, request IDs, client-provided principal/upstream fields and volatile session data are not included.

The package does not authenticate its input. Callers must get principal and upstream from trusted deployment configuration / authenticated transport, **not** client-supplied JSON, and must refuse invocation if that provenance is absent. No policy evaluation, authorization or anti-replay is performed.

This profile is frozen for v1 ticket binding; changing accepted number/string rules requires an explicitly different version and independent security review.
