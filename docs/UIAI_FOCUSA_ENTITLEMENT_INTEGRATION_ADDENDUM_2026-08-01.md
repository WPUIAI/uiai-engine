# UIAI × Focusa Entitlement Integration Addendum

This addendum is normative wherever `UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md` discusses scope, tokens, tools, or browser execution.

- Focusa project/Workpoint scope is cognitive/action context, not UIAI product permission.
- UIAI health, loopback, local API, extension, Pi/MCP, Cockpit, provider, or Focusa pairing tokens do not grant UIAI.
- A bundle lease must contain an explicit `uiai-engine` product grant.
- Focusa may broker only a short-lived child token no broader or longer-lived than the parent grant.
- UIAI independently verifies audience, parent lease id/sequence/digest, node/client, feature, time, limits, nonce/token id, and replay state.
- Browser/session/search/Markdown/diagnostics/media/control execution is gated before allocation or mutation.
- ResearchDiagnosticsPacket and evidence handles describe results; they do not retroactively authorize execution.
- Missing/invalid/expired/revoked UIAI entitlement is recovery-only even when Focusa itself is licensed.
- Protected UIAI workers/capsules require their own signed component/operation/key-envelope chain.

The hand-in-glove choreography becomes:

```text
Focusa project/Workpoint authority
+ UIAI caller authentication
+ explicit UIAI product/feature/limit authority
→ browser/research/diagnostics action
→ bounded evidence proposal
→ Focusa evidence/continuity decision
```

The mandatory entitlement spec, protected-worker addendum, endpoint matrix, and implementation work breakdown control conflicts.
