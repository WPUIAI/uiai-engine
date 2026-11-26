# Evidence Progressive Web App — customer-first viewing standard

**Owner directive:** 2026-11-14 · Proposed implementation in this PR. 

**EPWA means Evidence Progressive Web App.** A portable archive is an output format, not the product's name. Preserve the existing wire protocol, offline packages and share URLs.

## Customer experience

An ordinary visitor comes to understand **what was captured** and **what it demonstrates**, not to decipher Workpoints, hashes, custody fields or execution topology. Make the evidence itself the first meaningful action; use plain, confident language without false completion claims. Progressive disclosure makes the underlying machinery available for expert examination.

The record should orient the user immediately, provide a simple link to the captured evidence, and keep provenance/authority/validity layers, reviewer controls, case facts and developer machine records discoverable without crowding the first screen. Review actions that are actually eligible and require a decision must automatically reveal their panel.

## Non-negotiable truth and control

- **Evidence is not automatically proof of task completion.** Captured bytes, integrity, provenance, verification, completion, settlement and legal admissibility remain distinct.
- Do not invent customer success, say an action occurred when it did not, or hide material limitations, access restrictions, pending/revoked states or proof gaps.
- The captured artifact and custody records remain immutable. UIAI owns evidence capture and packaging; Focusa remains the execution and completion authority.
- Retain read-only defaults, reviewer authorization, record IDs, share paths, security boundaries, redaction and signed/hashed package contracts.
- Keep full semantic accessibility, low-memory operation, read-only/offline usability and en/es/ar localization including RTL.

## Fidelity acceptance

1. Reader opens a package or registry and can see what was captured without engineering jargon.
2. The captured evidence is one prominent interaction away; the technical details and validities are keyboard-accessible via expandable sections.
3. Authorized review remains visible and accessible when action is actually required; it never bypasses existing policy.
4. Export/share/review routes and immutable data remain unchanged; no changes to integrity calculations or project-scoped records.
5. Regression tests cover user-visible labels, localization, technical-detail preservation and exact acronym. Inspect rendered mobile/desktop, RTL and real browser behavior before claiming live acceptance.

Implementation references: `internal/evidenceshare/assets/`, `web/evidence/`, EPWA issue #106, and the delivery contract `docs/106r-uiai-evidence-pwa-epwa-delivery-agent-first-contract.md`.
