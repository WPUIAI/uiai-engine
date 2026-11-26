# 106z — EPWA → Focusa → Wirebot Go: lightweight Workstream evidence lens

**Status:** proposed bounded consumer amendment; no change to producer capture, publication, default access, owner auth, or completion authority.

## Existing contract

- [106r EPWA delivery](106r-uiai-evidence-pwa-epwa-delivery-agent-first-contract.md): **EPWA is Evidence Package Web Archive**. `uiai.epwa_delivery.v1` records durable package identity, artifact hash, scope, state, recovery and access; the Evidence PWA is the independently viewable human artifact.
- [106o autonomous work state](106o-uiai-evidence-pwa-autonomous-work-state-substrate-invariant.md): UIAI owns capture/render and immutable artifact snapshots. Focusa owns Workstream/Workpoint and review/settlement. An artifact/EPWA package is not proof that a task was accepted.
- [106d Focusa Action Deck](106d-uiai-evidence-pwa-focusa-action-deck-amendment.md): exact source scope, declared actions, audience/access class and paired/online posture. Go may **inspect** an approved projection; do not fork Action Deck or introduce another execution router.
- `internal/epwadelivery/types.go`, `contracts/schemas/uiai.epwa_delivery.v1.schema.json`: canonical producer fields already include `scope.project_ref`, `scope.workstream_ref`, `scope.workpoint_ref`, `scope.continuity_ref`, artifact hashes, delivery/access postures.

## Required Go consumption seam

UIAI continues delivering the original complete EPWA. The consumer **does not embed or copy an EPWA PWA** inside Wirebot Go. Instead a trusted existing source owner creates a small read-only summary appropriate to the exact authenticated Workstream:

```text
artifact_ref, artifact_revision, manifest_sha256,
delivery_id, delivery_state, package_ref, package_sha256,
project_ref, workstream_ref, continuity_ref, workpoint_ref,
producer, observed_at, access_posture, freshness,
safe_inspect_handoff? (trusted source-generated, audience-scoped)
```

All refs must be source-derived from an actual `uiai.epwa_delivery.v1` record. The consumer also reads Focusa's **separate** review/completion receipts. Do not copy a `verified`, `accepted`, `settled` or `completed` flag from `delivery_state=ready`; there is no such inference.

## Publication and security boundaries

- Preserve EPWA content address and portable/offline behavior. Never republish artifacts, mutate immutable manifests, or duplicate the EPWA service worker to create a Go tile.
- Do not send public/signed/private token-bearing `record_url`, `portable_url` or filesystem paths directly into unauthorized Go card payloads. Use a current server-verified, allowlisted scoped inspect handoff if and only if permitted; otherwise show reference, posture and recovery, without clickable media.
- Public-safe, private, unlisted, redacted, blocked and revoked are distinct. `public_safe` must not be assumed for customer private sessions.
- The existing source EPWA is the **full evidence view**. Go shows only provenance, compact thumbnail **when a genuine policy-safe derivative is authorized**, integrity/delivery state, independent review and a route to inspect source.
- Offline Go uses no cached private images or evidence metadata. The independently exported EPWA may be inspected offline under its separate access policy; this does not make Go an offline evidence store.

## Lifecycle and acceptance

1. Workstream A and B each publish an EPWA in the same owner project; Go only sees A when viewing authorized A.
2. `ready`, `pending_reconcile`, `corrupt`, `stale`, `blocked`, `redacted` map exactly; no silent `success` approximation.
3. Delivered + integrity valid + Focusa review pending renders **Evidence captured / awaiting review**.
4. Published artifact revoked/unavailable/expired no longer shows an actionable viewer; the source ref and permitted recovery remain.
5. First-party Wirebot Core mediates the owner-bound consumer; Go never contacts raw loopback UIAI, proxies arbitrary URLs or receives UIAI credentials.
6. Source owner validates same Workstream/Continuity across UIAI ScopeBinding and Focusa receipt before combining for display.

**Implementation note:** this is a consumer integration proposal and **does not create a new EPWA endpoint**. Reconcile route/permission ownership against current Issue #106 code and the Focusa read projection before implementing a producer change.