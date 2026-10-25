# AGENTS.md — Euxenion contributor and coding-agent guidance

This file is the standing instruction set for contributors and coding agents working in this repository. Read [README.md](README.md) first. Treat design details marked *planned* or *proposed* as provisional; do not claim unimplemented functionality exists.

## Project identity and scope

- **Euxenion** is the policy-enforcing MCP proxy and underlying architecture.
- **Xenri** is the planned human-facing interface/persona and CLI. Its convenient transport/client features do **not** substitute for server-side enforcement.
- The initial boundary is **one downstream client connection to one configured upstream MCP server** per proxy instance; avoid silently expanding the trust boundary or introducing implicit multi-upstream routing.
- The implementation direction is **Go**, with explicit interfaces between MCP transport, authenticated identity, ticket verification, policy evaluation, approval/redemption state, and auditing. OPA/Rego, Ed25519 JWS/JWT, and SQLite are current design choices requiring a concrete ADR/spec before being treated as stable APIs.
- This repository currently contains design/scaffolding documents, not a working authorization proxy. Do not invent commands, green tests, or operational guarantees.

## Non-negotiable workflow

1. **PRs only.** Work on a topic branch. Never push directly to `main`, merge a PR, enable auto-merge, or change repository administration settings without explicit human direction.
2. Make each PR narrowly scoped and describe its intent, security implications, compatibility impact, verification, and any unresolved design questions. **Agent-authored PR descriptions must include the fenced validation script and extended commit message specified below.**
3. Do not change user-approved architectural decisions incidentally. If a change alters the trust boundary, policy semantics, or MCP interoperability, propose it clearly for review before implementation.
4. Prefer a test or a small focused change over broad speculative scaffolding. Do not add unrelated changes to a documentation-only PR.
5. Before declaring work complete, inspect the final diff; report what was actually tested versus what could not be tested.

## Agent-authored pull request requirements

**Every pull request created or updated by an agent, including documentation-only PRs, MUST include both artifacts in the PR description itself:**

1. **Validation script** — a complete, copy-paste runnable shell script in a fenced `bash` code block. It must be safe to run from the repository root; use `set -euo pipefail` for Bash; fail nonzero when checks fail; and actually validate the scope of the proposed changes. Prefer existing project tooling rather than inventing commands or infrastructure. For documentation-only changes, meaningful file, formatting, and content checks are acceptable. State any prerequisites, and never use validation scripts that push, merge, modify remote state, or require secrets.
2. **Extended commit message** — a proposed **squash-merge commit message**, in a fenced `text` code block, with a concise imperative/conventional subject, a substantive body explaining motivation and key changes, and security/compatibility and validation notes where relevant. It must describe the **whole PR**, not just the last commit, and must be ready for human review or use at merge time.

Use explicit PR headings `## Validation script` and `## Extended commit message` immediately before their respective fenced blocks. Keep the **Verification** section separate: report which commands were actually run, their observed results, and any unrun checks or limitations. **Including a script does not mean it was executed.** Do not claim tests passed without having run them.

When an agent changes a PR after opening it, update both blocks to match the final diff and re-evaluate verification. Missing or stale blocks make the PR incomplete even if all status checks pass.

## Security invariants

- **Fail closed:** malformed/missing identity, untrusted issuers, invalid signatures, bad audience/subject, expired or not-yet-valid tickets, policy errors, store failures, and malformed authorization state must never result in an implicit allow.
- **Policy is authoritative:** verified tickets are evidence, not capabilities that bypass the policy decision. Explicit forbids outrank allowed paths. A `require` result is not permission to invoke the upstream server.
- **Authenticate provenance:** keep authenticated caller identity separate from client-controlled JSON, `_meta`, headers, ticket payloads, and upstream-supplied claims. Never treat a self-asserted principal as authenticated.
- **Bind exact requests:** authorization for a one-shot invocation must cover the intended principal, destination, MCP operation, and exact arguments via an unambiguous, versioned canonicalization/digest procedure. Detect differences that could alter semantics.
- **Prevent replay and races:** enforce ticket ID, expiry, use limits, atomic approval redemption, and concurrent-consumer safety in durable state. Never treat a failed or ambiguous database transaction as redeemed permission.
- **Do not retry side effects:** if upstream execution may have occurred, never automatically replay a tool invocation solely because the transport/result was ambiguous.
- **Honor protocol boundaries:** namespaced MCP `_meta` is the intended location for optional authorization evidence. Do not inject synthetic authorization fields into upstream tool `arguments` or leak evidence upstream unless explicitly required and safe.
- **Avoid information leaks:** never log full authorization tokens, signing keys, credentials, sensitive arguments, or unnecessarily detailed denials. Audit records should be privacy-conscious and tamper-evident where feasible.
- **No shortcuts for dev mode:** test-only bypasses must not be enabled by default or reachable in production paths.

## MCP compatibility

- Existing MCP clients must continue to communicate without understanding Euxenion-specific extensions.
- Enhanced clients can receive structured requirements and richer approval interactions only when supported by the negotiated protocol/capabilities.
- Preserve upstream tool names, input schemas, response semantics, error categories, and cancellation behavior except for the explicitly documented authorization boundary.
- Design tests for both legacy and enhanced flows. In particular, denial and `require` outcomes must never accidentally dispatch the protected upstream operation.

## Implementation and testing expectations

- Prefer standard Go tooling: `gofmt`, `go test`, static analysis, race tests, and minimal justified dependencies **once a Go module exists**. Do not report these as run before code is present.
- Keep transport adapters separate from the authorization core; inject time, key lookup, persistence, and upstream dispatch so security paths can be deterministically tested.
- Model policy outcomes and error classes with explicit types. Avoid boolean shortcuts that conflate `deny`, `require`, and evaluation failure.
- Validate configuration strictly, including trust anchors, issuer/audience relationships, and unsafe defaults. Favor versioned schema changes and migrations over silent reinterpretation.
- Write negative and adversarial tests alongside happy-path tests: forged/wrong-issuer tickets, mismatched digest, expired evidence, replay, concurrent redemption, store outages, policy conflicts, confused-deputy cases, unsupported MCP capabilities, and ambiguous upstream execution.
- Include compatibility tests for stdio and HTTP transports if/when each is implemented. Do not assert protocol support on the basis of design alone.
- Record a brief architecture decision when introducing a persistent security-sensitive dependency or protocol representation.

## Documentation and PR expectations

- Update the README for user-visible or architectural changes; update AGENTS.md when contributor constraints evolve.
- State clearly what exists today, what is planned, and what remains unsafe or unsupported.
- Do not commit keys, tokens, approval data, personal information, local databases, compiled binaries, or development secrets. Use sanitized fixtures.
- PR descriptions should include **Summary**, **Security / compatibility considerations**, **Verification**, and **Open questions**.
- Human review is required before merging, even if an automated check passes.

