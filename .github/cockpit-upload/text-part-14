## C.1 UIAI Engine repository sources

- `README.md`
- `docs/HLT_LEDGER.md`
- `docs/UIAI_OPERATOR_BROWSER_DESKTOP_SPEC_2026-06-19.md`
- `docs/UIAI_UX_DX_FPV_CONSOLIDATION_2026-06-12.md`
- `docs/UIAI_AGENT_FPV_COPILOT_SPEC_2026-06-09.md`
- `docs/UIAI_AGENT_FPV_PWA_SPEC_2026-06-09.md`
- `docs/UIAI_BROWSER_UX_DX_FEEDBACK_2026-06-09.md`
- `docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md`
- `docs/AGENT_EXPERIENCE_ROADMAP_IMPLEMENTATION_SUMMARY.md`
- `docs/PUBLIC_API_PARITY_MATRIX.md`
- `docs/AGENT_NON_BROWSER_API_EXPOSURE_INVENTORY.md`
- `docs/SOURCE_TO_MARKDOWN_AGENT_SPEC.md`
- `docs/SESSION_API.md`
- `internal/server/server.go`
- `internal/routes/tools.go`
- `internal/routes/fpv.go`
- `internal/routes/markdown.go`
- `internal/media/jobs.go`
- `apps/cockpit/src/lib/contracts/*`
- `apps/cockpit/src/lib/cards/phase0-card-manifest.ts`
- `apps/cockpit/src/routes/+layout.svelte`
- `apps/cockpit/smoke/smoke-runner.ts`
- UIAI Engine issue: browser session broker with leases and safe parking.

## C.2 Focusa authority sources

Preserve the existing desktop spec’s referenced Focusa sources, especially:

- multi-device sync and CRDT;
- device pairing and pairing-room/wizard/revoke/repair;
- ontology-backed tool contracts;
- project-root reconciliation;
- typed scoped runtime and singleton elimination;
- merged-scope DX/UX;
- context bootstrap and delivery;
- installer/license/update authority;
- benchmark/proof/receipt plans;
- cloud control plane and tool gateway;
- current UIAI diagnostics integration.

## C.3 Agentic-browser architecture reference

- `Visual and Interactive Feedback Reports from Agentic Workers`, supplied in the product discussion and conformed in v0.5 as Review Reports/Report Canvas under the existing mission, authority, evidence, and verification model.

- `Agentic Browser Best Practices Specification` (`ABPS-1.0`), supplied for this evaluation. Its normative requirements for mission architecture, authority, action routing, receipts, verification, settlement, security, reliability, memory, privacy, economics, interoperability, worker identity, UX, observability, testing, maturity, and Focusa/UIAI responsibility mapping are integrated primarily in Sections 4, 7–9, 13, 17–21, Annex A.9, and Annex E.

## C.4 External design and implementation references

Use current official guidance when implementing:

- Apple Human Interface Guidelines: [design principles](https://developer.apple.com/design/human-interface-guidelines/design-principles), [designing for macOS](https://developer.apple.com/design/human-interface-guidelines/designing-for-macos/), [typography](https://developer.apple.com/design/human-interface-guidelines/typography), [icons](https://developer.apple.com/design/human-interface-guidelines/icons), [SF Symbols](https://developer.apple.com/design/human-interface-guidelines/sf-symbols), [sidebars](https://developer.apple.com/design/human-interface-guidelines/sidebars), [popovers](https://developer.apple.com/design/human-interface-guidelines/popovers), [sheets](https://developer.apple.com/design/human-interface-guidelines/sheets), [alerts](https://developer.apple.com/design/human-interface-guidelines/alerts), [loading](https://developer.apple.com/design/human-interface-guidelines/loading), [motion](https://developer.apple.com/design/human-interface-guidelines/motion), and [accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility);
- [WCAG 2.2](https://www.w3.org/TR/WCAG22/): focus visibility and appearance, target size, dragging alternatives, contrast, keyboard behavior, and accessible authentication;
- [WAI-ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/patterns/): disclosure, dialog, tabs, toolbar, tooltip, tree, and splitter behavior;
- Tauri v2 official documentation and WebdriverIO Tauri testing guidance;
- Maestro official documentation for supported targets, output artifacts, reports, and recording;
- PDF.js for a customizable PDF viewer layer;
- Docling or an equivalent local semantic document parser for multi-format extraction;
- isolated PDF structural/OCR/conversion/signing tools selected only after license and security review.

---

# Annex D — Decisions still requiring explicit acceptance

1. Rename the document and in-app product from **UIAI Operator Browser** to **UIAI Cockpit** everywhere, while retaining historical references in changelog/migration notes.
2. Adopt the task-oriented sidebar defined in Section 6.2.
3. Migrate `browser_execution` to `uiai_execution + execution_domain` additively.
4. Make Activity the shared home for Jobs, approvals, notifications, history, and audit.
5. Make Capabilities the complete discoverability surface for all UIAI route families.
6. Add WorkObject, WorkspaceManifest, CapabilityManifest, ArtifactRef, Job, and RunnerAdapter contracts.
7. Treat Documents and Test Lab as first-party product tracks requiring UIAI Engine backend work, not UI-only features.
8. Keep FPV PWA as a first-class mobile/share projection of Live rather than deprecating it.
9. Establish the first-party document implementation stack after dependency/license/security review.
10. Choose exact timing for dynamic capability registry generation after static first-party contracts are proven.
11. Decide whether the existing desktop specification is replaced wholesale or split into this product/UX master plus a separate implementation/release appendix.
12. Convert each accepted new section into beads under the existing Cockpit epic and update acceptance criteria rather than creating an unrelated roadmap.
13. Adopt Section 10 as a merge- and release-blocking design-system contract, including token lint, shared component fixtures, visual regression, accessibility, motion, overlay, and human review gates.
14. Accept Cockpit as the Mission Experience layer, Focusa as Mission Kernel, and UIAI as execution/evidence plane; do not create a Cockpit-local canonical mission store.
15. Adopt additive MissionContractRef, CompletionPredicateRef, WorkerRef, TaskLease, CapabilityGrantRef, ActionProposalRef, ActionReceiptRef, VerificationResult, SettlementState, BudgetSnapshot, ActuatorRef, and DataClassification contracts.
16. Adopt R0–R5 risk classes and scoped capability-lease approvals for consequential operations.
17. Require false-done prevention, predicate-linked evidence sufficiency, independent verification for high-risk outcomes, and explicit provisional-versus-settled status.
18. Require actuator-neutral routing and prohibit fallback from expanding authority or disclosure.
19. Make idempotency, ambiguous-result reconciliation, classified retries, dead-letter handling, worker leases, and Session Broker behavior release requirements for shared/consequential execution.
20. Add ABPS conformance and maturity reporting to release/benchmark evidence without overstating unimplemented capabilities.
21. Adopt Review Reports as derived Evidence/work-object projections, not a new canonical task or memory system.
22. Adopt the Report Canvas, report families, lifecycle, audience profiles, declarative blocks, interaction manifests, and version-bound sharing model.
23. Require that report interactions route through standard guards/orchestration and that report approval remain distinct from mission settlement or follow-up authorization.
24. Add a UIAI report-composer/artifact track while preserving Focusa ownership of durable mission decisions and Workpoint creation.

---

# Annex E — ABPS-1.0 conformance evaluation

This annex records the evaluation of the v0.3 Cockpit specification against the supplied Agentic Browser Best Practices Specification and the integration outcome in v0.4.

Status meanings:

- **Strong:** already explicit and materially aligned before this revision.
- **Partial:** present in fragments but lacked a complete contract or cross-workspace model.
- **Missing:** not explicit enough to support a production conformance claim.
- **External owner:** required by the system, but Cockpit must surface rather than canonically own it.
- **Integrated v0.4:** normative additions now define the required product/contract posture; implementation may remain future work.

| ABPS domain | v0.3 evaluation | v0.4 integration |
|---|---|---|
| Browser is replaceable actuator, not canonical brain | Strong | Preserved in Sections 2, 4.1, and 13. |
| Focusa as canonical mission/continuity authority | Strong | Elevated to explicit Mission Kernel mapping. |
| Mission Experience not chat-only | Strong/Partial | Overview formally becomes Mission Deck. |
| Versioned Mission Contract | Missing | Added in Section 4.3; Focusa-owned additive contract. |
| Completion Contract and predicates | Missing | Added in Section 4.4 and workspace verification rules. |
| Explicit mission/task/action states | Partial | Added canonical lifecycle model in Section 4.5 and Activity. |
| Deterministic Authority Kernel | Strong/Partial | Existing guards retained; capability-grant and action-boundary rules added. |
| Capability grants, expiry, use/value/delegation | Missing | Added Section 4.6 and approval-lease UX. |
| Risk-sensitive approvals | Partial | Unified approval existed; R0–R5 and consequence fields added. |
| Action Proposal | Partial | Command preview broadened to machine-readable proposal contract. |
| Action Router across connector/API/MCP/browser/human | Missing | Added Section 4.8 and Capabilities routing UX. |
| Consequential Action Receipt | Partial | Artifact/evidence refs existed; full receipt contract added. |
| Independent verification | Partial | Test/evidence proof existed; executor/verifier separation added. |
| False-done prevention | Missing | Added predicate/evidence and truthful status requirements. |
| Provisional completion and settlement | Missing | Added mission/action lifecycle and Evidence/Mission Deck surfaces. |
| Contradiction preservation/reconciliation | Partial | Existing conflict UX broadened to evidence contradictions. |
| Preconditions/postconditions | Partial | Flow assertions existed; now required for consequential tasks. |
| Idempotency and ambiguous retry reconciliation | Missing | Added Section 4.11 and Automations/Testing gates. |
| Classified bounded retries and dead-letter | Partial | Job retries existed; semantic classes and dead-letter added. |
| Compensation versus undo | Strong/Partial | Existing distinction preserved and tied to orchestration contracts. |
| Multi-worker locks, leases, ownership, backpressure | Partial | Existing Scope/CRDT/session-broker work broadened to all task ownership. |
| Hostile content / instruction-data separation | Partial | Document/browser safety existed; foundational untrusted-content rule added. |
| Data classification, provenance, taint, egress | Missing | Added Sections 4.12 and 17.5. |
| Credential broker / opaque handles | Partial | Keychain existed; origin-bound short-lived broker requirements added. |
| Structured browser tools are untrusted | Missing | Added origin/manifest/schema/risk/re-evaluation contract. |
| Upload/download hostile-file policy | Partial | Document safety existed; cross-workspace file policy added. |
| Memory store separation and quarantine promotion | Partial | No parallel memory existed; controlled promotion model added. |
| Model/data routing and minimum necessary context | Partial | AI consent/cost existed; data-class and economic routing added. |
| Budgets and cost per verified outcome | Partial | Cost guards existed; mission budget and verified-outcome metrics added. |
| Worker identity and delegation chains | Partial | Node roles existed; WorkerRef/lease/delegation requirements added. |
| Meaningful progress, uncertainty, intervention | Strong/Partial | Progressive disclosure/FPV strong; mission progress and uncertainty rules added. |
| Operational telemetry and incident linkage | Partial | Local telemetry existed; governed-outcome metrics and incident tests added. |
| Security red-team categories | Partial | General security tests existed; ABPS attack matrix added. |
| Maturity/conformance levels | Missing | Added Section 4.16 and truthful capability labeling. |
| UI typography/iconography/motion/overlays/composability | Strong | v0.3 Section 10 retained as strict release contract. |
| Visual and interactive agent-work reports | Partial/Missing | Integrated in v0.5 as derived Review Reports, Report Canvas, governed interaction manifests, evidence provenance, versioning, sharing, and follow-up contracts. |
| PDF/Office and Test Lab as central workspaces | Cockpit extension beyond ABPS | Preserved; now governed by the same mission/action/evidence contracts. |

## E.1 Highest-risk gaps found in v0.3

The v0.3 spec was strong as an information architecture, desktop UX, scope-aware adapter shell, FPV surface, artifact workspace, and design-system contract. Its primary weakness was that it could make a complex execution platform look coherent without formally defining the governed path from intent to externally verified and settled outcome.

The most consequential gaps were:

1. no explicit Mission Contract or Completion Contract;
2. insufficient distinction between successful execution and verified/settled outcome;
3. no general Capability Grant or Action Proposal/Receipt contract;
4. browser-centric capability placement without an explicit actuator-neutral Action Router;
5. fragmented idempotency, retry, reconciliation, and dead-letter requirements;
6. incomplete hostile-content, data-classification, egress, credential-broker, and structured-tool policies;
7. no controlled procedural-memory promotion model;
8. incomplete worker identity/delegation and shared-budget model;
9. no maturity language preventing a polished UI from overstating autonomy or verification.

## E.2 Best-practice integrations deliberately not assigned to Cockpit

The following are system requirements but remain external-owner responsibilities:

