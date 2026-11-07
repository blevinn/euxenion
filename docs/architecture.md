# Engineering design: protocol-aware capability gateway

**Status:** Proposed architecture. The repository currently contains only Xenri CLI scaffolding; none of these gateway frontends or security controls are implemented. This document refines the direction without changing the approved 0.0.5/0.1.0 deliverables.

See [strategy.md](strategy.md) for product boundaries and [../README.md](../README.md) for current implementation status.

## Architecture and responsibility boundaries

```text
MCP clients        OpenAI/provider SDKs       Git clients       CLI/browser/other apps
    |                      |                       |                       |
[MCP adapter]       [Inference adapters]     [Git smart HTTP]     [HTTP/SOCKS egress]
    |                      |                       |                       |
    +----------------------+-----------------------+-----------------------+
                                  |
                       Authenticated workload identity
                                  |
                   Policy decision + verified ticket evidence
                                  |
                 Admission / budgets / reservations / audit
                                  |
                Protocol-specific routing, credentials and dispatch
                                  |
             MCP servers / inference providers / Git hosts / network
```

Frontends perform protocol parsing and construct a **versioned capability descriptor**. The shared core never blindly interprets client claims as verified facts; each adapter records provenance and what can be enforced. A frontend may enforce a coarse predicate but cannot claim stronger guarantees than its observation and execution path support.

Suggested internal domains: `identity`, `capability`, `policy`, `tickets`, `admission`, `audit`, `secrets`, and frontend/backend packages for `mcp`, `inference`, `git`, `egress`. This is a separation of responsibilities, not a commitment to package names, public APIs, or a plugin framework. Retain the Go single-binary/CLI direction unless a concrete isolation need emerges.

## Capability and decision contract

Illustrative normalized policy input:

```json
{
  "version": 1,
  "subject": {"id": "agent:coder-17", "authn": "verified-workload"},
  "action": "git.push",
  "resource": {"type": "git.repository", "id": "github.com/org/repo"},
  "attributes": {"ref_updates": [{"ref": "refs/heads/topic", "old_oid": "...", "new_oid": "..."}]},
  "verified": ["subject", "resource", "action", "attributes.ref_updates"],
  "enforcement": {"frontend": "git-smart-http", "capabilities": ["repository", "ref-update"]},
  "evidence": {"ticket_ids": []}
}
```

The `verified` list is a **conceptual** provenance mechanism, not a suggested user-supplied trust assertion. A typed internal representation must distinguish verified, derived, and untrusted fields; adapters and policy evaluate only authenticated/verified claims for security decisions.

Policy outcomes remain **allow**, **require** (additional evidence or approval), and **deny**, with explicit failure-to-deny behavior, reason codes, required evidence, and allowed constraints. Explicit prohibitions outrank positive paths. A valid signature never bypasses policy. A `require` outcome never dispatches upstream.

A request pipeline should be:

1. Authenticate the caller and resolve a configured resource/route.
2. Parse and validate the transport request, derive verified attributes, and identify enforcement capabilities.
3. Verify issuer, audience, subject, expiry, scope, invocation binding, use limits and replay state for tickets.
4. Evaluate policy using verified facts plus authorized evidence, failing closed on uncertainty.
5. Perform admission control (request quotas, concurrency, token/cost/byte reservations as applicable).
6. Dispatch once through a protocol-specific upstream adapter, with cancellation/streaming semantics.
7. Settle observed usage and record policy, transport, execution, and unknown-outcome states without disclosing secrets.

Authorization, admission, execution, and settlement are **distinct stages**. Budget tickets grant permission to consume within constraints; preventing overspend requires atomic reservations and reconciliation. Do not automatically replay ambiguous side effects.

## Ticket binding and approvals

Preserve equal internal/external issuer support and exact-invocation approvals by default. Bind an exact ticket to a versioned, canonical descriptor including authenticated principal, destination, operation, relevant arguments/parameters, and the intended effects identifiable prior to dispatch. Existing proposed JWS/JWT Ed25519 profile and JCS invocation digest remain *proposals* pending protocol/ADR review; don't silently redefine their semantics.

MCP can carry namespaced `_meta` for enhanced clients. Unmodified clients receive compatible structured failures plus an out-of-band approval/retry path. Other transports need protocol-native error/challenge behavior or a helper; opaque TCP streams cannot safely be held open indefinitely for a human approval. Redemption must be durable, atomic and replay-safe. Inference requests may bind model, parameter bounds and request digest; broader grants must have explicit spend/time/use constraints. Git push approval should bind repository, authenticated identity, full old/new object IDs and ref transitions, with upstream concurrency validation.

## Protocol-specific designs

### MCP (first deliverable)

Maintain a **single configured upstream MCP server** in the 0.1.0 MVP. Preserve tool schemas, cancellation, error semantics and compatibility. Do not add synthetic tool arguments or leak ticket evidence upstream. Multi-upstream support is future work. Existing 0.0.5 cryptographic work remains the prerequisite.

### Inference (future)

Start with a focused OpenAI-compatible frontend, model aliases, SSE streaming, and one upstream provider. Native provider pass-through is an option for features that don't normalize cleanly; it still requires authenticated routing, endpoint allowlists, credential injection, cancellation, usage extraction and bounded accounting. Keep provider translations, tool calls, structured outputs, reasoning tokens and multimodal semantics adapter-specific. Never claim full LiteLLM parity. Restrict retry behavior where a request may already have generated/billed work. Prompt and response logging are opt-in and privacy-sensitive.

### Git (future)

Use **Git smart HTTP** for ordinary clone/fetch and eventually push; preserve packfile/negotiation behavior and native clients. Initial controls are per authenticated identity/repository and operation (`git-upload-pack` versus `git-receive-pack`). Do not infer authorization from user-controlled repository URLs; use configured mappings and upstream credentials.

Ref-level push policy requires robust wire-protocol parsing, full object IDs, ref-state checks and (for fast-forward checks) verified object ancestry. A request inspected before forwarding can race with remote state; rely on hosting-side rules/hooks for high-assurance protections. Git read authorization defaults to **repository scope**: partial ref advertisement is not a file-level confidentiality boundary. The previously discussed Git-over-MCP bridge is an optional compatibility adapter for MCP-only egress, not the canonical Git path. Approval of a push is a challenge-and-retry operation, not an indefinitely stalled HTTP connection.

### Generic egress (deferred)

Begin, if justified later, with HTTP CONNECT and SOCKS5 **CONNECT only**. A tunnel exposes destination identity/port and connection metrics, **not the encrypted HTTP method, path, Git ref change, or model request within it**. Cleartext HTTP proxying can expose additional attributes. UDP, SOCKS BIND, transparent interception, arbitrary raw traffic, and TLS MITM are out of initial scope.

For Incus and other sandboxes, bind authenticated workload identity to the egress path; proxy environment variables alone are not isolation. The sandbox must deny direct external egress and cover DNS, IPv6, UDP/QUIC and local routes as appropriate. Reject SSRF-prone destinations, validate DNS results and connect to the vetted address to resist rebinding, constrain ports and upstream routing, and prevent generic egress from bypassing protocol-aware rules. Domain/IP aliases and CDNs limit perfect service identification; explicitly document residual risk.

## Threats, safeguards and operational constraints

- **Confused deputy:** identity must come from authenticated transport, not MCP metadata, headers, hostnames, or ticket subject assertions.
- **Policy downgrades:** absent verified attributes must produce deny/require, not fallback to a permissive lower-visibility decision.
- **Credential leakage:** keep upstream tokens in the gateway; constrain routes and never forward internal ticket evidence to arbitrary upstreams.
- **Replay/races:** atomic approval redemption, ticket audience/subject binding, and transaction failure handling.
- **TOCTOU:** Git ref-state changes and DNS rebinding must be handled close to use; retain host-side protections.
- **Privacy and accounting:** log minimized identifiers and outcomes; accurately distinguish reserved, observed, and estimated usage.
- **Availability:** bound connections, streams, request bodies, concurrency and timeouts; propagate cancellation and support safe shutdown.
- **Protocol compatibility:** test standards-compliant clients, streaming, errors, malformed inputs and abrupt disconnects for each frontend.

## Evolution and open design decisions

Capability schema/versioning, identity credential formats, CEL versus OPA/Rego or other policy engines, persistence and distributed quota settlement, audit event format, ticket carriage for non-MCP clients, Git push inspection guarantees, and sandbox-to-gateway networking require ADRs before implementation.

The design **does not** mandate a new inference, Git, or SOCKS milestone. Any future frontend needs its own protocol threat model, compatibility tests, failure-path tests, explicit guarantees, and review. The 0.0.5 ticket foundations and 0.1.0 single-upstream MCP MVP remain the immediate engineering objectives.
