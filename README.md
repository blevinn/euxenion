# Euxenion

**Euxenion** is a policy-enforcing proxy for the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/). It is designed to mediate access to a single upstream MCP server using declarative policy, cryptographically verifiable authorization tickets, delegated trust, and human-in-the-loop approvals.

**Euxenion enforces authorization. Xenri facilitates it.**

> **Project status:** Early design / initial scaffolding. The proxy, CLI, ticket handling, and approval workflows described below are planned, not yet implemented. Do not rely on this repository to enforce security policy until the implementation has undergone testing and review.

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

The proxy handles MCP transport and protocol interactions while keeping identity, canonical invocation construction, ticket verification, policy evaluation, approval state, and upstream dispatch as separate responsibilities. The initial implementation direction is **Go**, with an embedded policy engine (currently proposed: OPA/Rego) and a durable local state store (initially proposed: SQLite).

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

Euxenion is the **discerning host** for MCP interactions. It connects clients to the tools they need while ensuring that each protected invocation satisfies explicit policy and independently verifiable authorization requirements. Its hospitality makes legitimate access possible—not unconditional.

**Xenri** is a companion name built around the same *xen-* (“guest” or “stranger”) root. Its shorter, friendlier form suits the human-facing interface and planned CLI: a guide for operating the proxy, inspecting authorization tickets, understanding policy decisions, and coordinating approvals. The CLI may also provide local transport bridging, but a transport-only connector is not an enforcement boundary.

**Xenri facilitates authorization. Euxenion enforces it.**

Command names, configuration schema, and package layout are not yet stable. No working CLI commands are documented here until they actually exist.

## Roadmap

1. Define the MCP interception boundary, threat model, versioned ticket and challenge schemas, and policy contract.
2. Scaffold the Go module, `xenri` command, configuration validation, and continuous integration.
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
