# Euxenion

**Euxenion** is a policy-enforcing proxy for the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/). It is designed to mediate access to a single upstream MCP server using declarative policy, cryptographically verifiable authorization tickets, delegated trust, and human-in-the-loop approvals.

> **Project status:** Initial Go scaffolding. The `xenri` command supports only `help` and `version`; no MCP proxy, policy enforcement, authorization tickets, or approval workflow has been implemented. This repository cannot yet secure access to an MCP server.

## Why Euxenion?

MCP clients and servers can connect powerful tools to automated agents. Euxenion aims to place an independently operated authorization boundary between a client and an upstream server, without requiring existing MCP clients or upstream servers to be rewritten.

The proxy evaluates each policy-controlled invocation using the authenticated caller, the exact requested operation, and verified authorization evidence. Policy may allow a request, forbid it, or require additional evidence—such as approval from a person or another trusted authority—before execution.

## Design principles

- **Deny by default, fail closed.** Missing identity, invalid evidence, policy errors, and ambiguous security states must not silently permit access.
- **Evidence is not permission.** A valid ticket contributes issuer-qualified claims for policy evaluation; the policy engine still makes the authorization decision.
- **Equal footing for authorities.** Internal and external ticket issuers use the same verification and policy mechanisms. Trust is explicit, not inferred from origin.
- **Narrow approvals by default.** Human approval is bound to one precise invocation. Broader grants are possible only when policy explicitly permits them.
- **Preserve MCP compatibility.** Unmodified clients must be supported; clients that understand Euxenion-specific metadata may offer a richer approval experience.
- **One upstream boundary.** A proxy instance mediates a single upstream MCP server. Routing and multi-upstream orchestration are outside the initial scope.
- **Auditable decisions.** Authorization outcomes and ticket consumption should be explainable without leaking secrets.

## Planned architecture

```text
Existing or enhanced MCP client
            |
            v
     Euxenion MCP proxy
       |       |       |
       |       |       +-- Approval / ticket state and audit
       |       +---------- Trusted issuer registry + ticket verification
       +------------------ Policy enforcement (PEP) / evaluation (PDP)
            |
      ALLOW only
            v
    Single upstream MCP server

Xenri: human-facing CLI and approval interface
```

The planned proxy will handle MCP transport and protocol interactions while keeping identity, canonical invocation construction, ticket verification, policy evaluation, approval state, and upstream dispatch as separate responsibilities. The project uses **Go**; an embedded policy engine (currently proposed: OPA/Rego) and a durable local state store (currently proposed: SQLite) remain design choices, not implemented dependencies.

### Authorization tickets

The intended ticket format is a versioned, signed JWS/JWT profile, initially using Ed25519 (EdDSA). Tickets carry conventional issuer, audience, subject, issuance/validity, and unique-ID claims plus namespaced authorization claims.

A ticket may be bound to a canonical invocation descriptor, including the authenticated principal, proxy/upstream identity, MCP method or tool, and exact arguments. The current design calls for SHA-256 over RFC 8785 JSON Canonicalization Scheme (JCS) data to construct invocation bindings.

Trusted issuers must be explicitly configured. Expiration, audience, signature, invocation binding, replay prevention, and permitted uses are checked before evidence is considered by policy. Credentials and unverified client-provided claims must not become trusted authorization facts.

**Ticket carriage:** The preferred protocol extension is namespaced MCP request `_meta`, not an injected `authorization_tickets` tool argument that would alter upstream tool schemas. The wire contract will be specified and versioned before implementation.

### Policy decisions

The authorization contract has three outcomes:

| Outcome | Meaning |
| --- | --- |
| **Allow** | All necessary requirements are satisfied; forward the invocation once. |
| **Require** | An otherwise eligible invocation needs additional evidence; return structured, machine-readable requirements / an approval challenge. |
| **Deny** | A prohibition applies, no valid authorization path exists, or evaluation cannot safely complete. |

Policy may express prohibitions plus alternative authorization paths, including AND/OR combinations of requirements. An explicit prohibition takes precedence over satisfied grants. Policies do not execute a tool merely because a ticket verifies successfully.

### Approvals and MCP compatibility

For an **enhanced MCP client**, Euxenion intends to expose structured challenges and, when mutually supported, a richer MCP elicitation or approval flow.

For an **unchanged MCP client**, it must retain valid MCP behavior and provide an understandable error/challenge with an out-of-band path for a person or authority to approve and for the invocation to be retried. The proxy must not assume every client understands new extensions.

Approval applies to the specific invocation (and authenticated caller) by default. Redemption must be atomic and replay-safe. Ambiguous upstream outcomes must **never** cause an automatic replay of a potentially side-effecting operation.

## The names: Euxenion and Xenri

**Euxenion** draws on the ancient Greek *eúxenos* (εὔξενος)—“hospitable” or “welcoming to strangers”—and the tradition of *proxenia*, in which trusted representatives helped visitors navigate relationships with other communities. The name captures the project's purpose: enabling useful interactions across boundaries through established trust and clear rules. See the [Liddell–Scott–Jones Greek lexicon](https://atlas.perseus.tufts.edu/dictionaries/entry/urn%3Acite2%3Ascaife-viewer%3Adictionaries.v1%3Alsj-n44575/) and [Oxford's Proxeny Networks project](https://proxenies.csad.ox.ac.uk/about).

The name suits an MCP proxy that mediates access between clients and upstream tools. Euxenion evaluates requests against policy and verified authorization evidence before forwarding permitted invocations.

**Xenri** is a companion name built around the same *xen-* (“guest” or “stranger”) root. Its shorter form suits the human-facing interface and CLI. The initial `xenri` executable offers help and version reporting; future commands will support proxy operation, ticket inspection, policy diagnostics, approvals, and potentially local transport bridging. A transport-only connector is not an enforcement boundary.

Proxy subcommands, configuration schemas, and authorization-related package layouts are not yet defined.

## Development

The initial CLI scaffold requires **Go 1.26 or newer** and has no external Go dependencies.

For a consistent development environment, use [Nix flakes](https://wiki.nixos.org/wiki/Flakes):

```bash
nix develop
go version   # Go 1.27 from the pinned nixpkgs input
gofmt -h     # available with Go
```

The dev shell provides Go (including `gofmt`), Git, and the C toolchain needed for `go test -race`. It supports Linux and macOS on x86-64 and ARM64. The committed `flake.lock` fixes the nixpkgs revision for reproducible environments; Nix must have flakes enabled. Work from the PR branch when validating a PR; otherwise the branch-specific diff checks will not match.

Once inside the shell:

```bash
go run ./cmd/xenri help
go run ./cmd/xenri version
go test ./...
go vet ./...
go build ./cmd/xenri
```

The command intentionally does not accept proxy or approval subcommands yet. Its entry point is `cmd/xenri`; `internal/cli` contains testable argument handling separate from process exit and I/O setup. Future security-sensitive components will be introduced only alongside their protocol contracts and tests.

CI runs formatting, vetting, race-enabled tests, a build, and CLI smoke tests on supported Go versions.

## Roadmap

1. Define the MCP interception boundary, threat model, versioned ticket and challenge schemas, and policy contract.
2. Scaffold the Go module, minimal `xenri` command, and CI (this PR); add configuration schema and validation before implementing proxy operations.
3. Implement transparent single-upstream MCP transport and identity propagation.
4. Add signature verification, canonical invocation binding, policy evaluation, and negative-path tests.
5. Add durable approval state, atomic redemption, audit events, and enhanced-client interoperability.
6. Perform adversarial testing and document deployment / trust assumptions before describing the proxy as production-ready.

This is a direction of travel, not a claim that these milestones are complete.

## Contributing

Contributions and security-focused design reviews are welcome. Please open an issue or a pull request rather than pushing directly to `main`. Start with [AGENTS.md](AGENTS.md) for development, agent, review, and security expectations.

This is security-sensitive infrastructure: changes to protocol behavior, identity, ticket validation, policy decisions, or approval redemption require explicit threat analysis and tests for failure paths.

## License

Licensed under the [MIT License](LICENSE).
