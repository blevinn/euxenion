# Euxenion strategy: capability gateway

**Status:** Proposed long-term direction; not an implemented product contract.  
**Scope:** Strategy and product boundaries, distinct from the current MCP MVP.

## Thesis

Euxenion is a **policy-enforcing gateway for autonomous workloads**. It mediates access to external capabilities using protocol-aware frontends, a common authorization and accounting core, and narrowly scoped, independently verifiable delegated authority.

MCP tools, model inference, Git, and ordinary network access are different protocols, not different security principals. Agents should be governed consistently across those capabilities without forcing all interactions through MCP or a universal wire format.

The differentiator is **security and delegation across heterogeneous agent capabilities**, not replacing every feature of LiteLLM, Git hosting, or an enterprise network security platform. Go remains the preferred implementation language; Xenri remains the CLI/human interaction surface.

## Product principles

1. **Policy is authoritative; tickets are evidence.** Internal and external issuers are first-class peers, subject to explicit trust configuration. Exact-invocation grants are the default; bounded broader grants require policy.
2. **Enforcement guarantees are explicit.** Only authorize predicates a frontend can verify and enforce. The capability core must not confuse permission to open a TCP connection with permission to perform a particular Git push or MCP tool call inside it.
3. **Existing clients work.** Preserve native Git smart HTTP, MCP interoperability, provider-native inference where practical, and standard SOCKS/HTTP clients. Enhanced clients may provide richer approval and retry flows.
4. **Deny by default and fail closed.** If identity, state, policy, or required evidence cannot be established, do not dispatch. Never silently downgrade a failed semantic policy to network-level allow.
5. **Small trusted core, protocol-specific edges.** Reuse identity, tickets, policy, admission control, accounting, audit, and secrets management; retain specialized parsing, streaming, dispatch, and error semantics in each frontend.
6. **Deliberately bounded scope.** Build a mediation gateway, not an agent framework, model host, Git server, observability suite, generic VPN, or transparent TLS-inspection appliance.

## Protocol portfolio

| Family | Intent | Initial useful scope | Deferred complexity |
| --- | --- | --- | --- |
| MCP | Tool/resource authorization, approval | One upstream, existing clients, exact invocation tickets | Multi-upstream orchestration and richer client UX |
| Inference | Model/provider access and cost controls | OpenAI-compatible interface, streaming, provider routing; optional native pass-through | Broad LiteLLM parity, every multimodal/provider feature |
| Git | Credential-brokered repository operations | Smart HTTP clone/fetch, repository-level policy | Push/ref-aware policy, approval, server-side integration, MCP-only transport bridge |
| Network egress | Unmodified applications and sandbox traffic | HTTP CONNECT and SOCKS5 CONNECT with destination policy | UDP, transparent interception, sophisticated network inspection |

MCP remains the **first implementation milestone**. The other frontends are architectural direction, not committed release features or claims of availability.

## Authorization and trust boundaries

The unit of policy is an authenticated **subject performing an action against a resource** with verified security-relevant attributes, optionally supported by tickets. The policy decision is separate from transport admission, upstream execution, and usage settlement. Audit records should state what was observed and what was merely claimed.

Authorization quality depends on protocol visibility:

| Level | Observable request | Appropriate guarantees |
| --- | --- | --- |
| Connection | Destination/protocol/port/identity | Destination and connection policy |
| HTTP | Host/method/path when visible | HTTP request policy, not encrypted tunneled contents |
| Application | MCP method, inference operation, Git service | Application operation policy |
| Semantic | MCP arguments, model parameters, verified Git ref changes | Fine-grained policy only after parsing/verification |

These are **capability descriptions, not automatically ordered security levels**. Frontends can expose different combinations of visibility. Allowlists for generic egress must not provide a bypass around more restrictive protocol-aware access; sandbox network restrictions and upstream controls provide defense in depth.

## Strategic sequencing

- **0.0.5:** Continue cryptographic foundation work without broadening the committed release.
- **0.1.0:** Deliver the existing single-upstream MCP MVP. Keep the policy/evidence contract capability-neutral where inexpensive and rigorously tested; do not add Git, inference, or SOCKS implementation requirements.
- **Post-0.1.0 (proposed):** Evaluate a read-only Git smart HTTP frontend and a focused inference proxy as separate increments; choose ordering by demand and verified security value.
- **Later:** Push authorization, admission/budget settlement, enhanced approval adapters, HTTP/SOCKS egress, and Incus-oriented workload identity and network enforcement.

Exact releases, sequencing, and work items require separate roadmap decisions; no milestone is amended by this strategy document.

## Non-goals and success criteria

Do not build a universal content/command policy abstraction that pretends to understand opaque tunnels, arbitrary prompts, or Git packfiles. Do not promise file-level confidentiality from Git ref filtering, global budgets from signed tickets alone, or protected-branch enforcement without trustworthy ref-state validation or upstream hosting policy.

Success means an operator can give an isolated agent **one authenticated identity and coherent policy** across several protocols, allow precise low-risk operations, require review for high-risk actions, account for consumption, and obtain an audit trail reflecting the real enforcement boundary—all while preserving normal client behavior.
