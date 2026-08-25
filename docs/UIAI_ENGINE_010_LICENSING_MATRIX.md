# UIAI-ENGINE-010 Licensing Matrix — wpuiai.com Authority Closure (C-010-29)

**Authority:** wpuiai.com license authority → signed identity (`internal/auth`) with `tier` + authority-issued feature/limit claims.
**Enforcement primitives:** `license.RequireFeature` / `license.RequireFeatureMiddleware` (spec §6.5 402 envelope), tier-derived defaults in `tierFeatures`, fail-closed for unknown tiers.
**Interim posture:** loopback Evaluation Mode grants only `eval_allowed` set (§6.3). Premium pillars below are denied in evaluation and granted to known licensed tiers until authority-issued explicit feature lists replace tier derivation (tracked by the 2026-08-01 enforcement spec).
**Rule:** a Wave cannot sign off while any of its claims has an unenforced premium row.

## Treatment legend
`always_available` · `base_entitlement` (any licensed tier) · `optional_premium` (feature key required) · `read_allowance` · `recovery_exempt`

## Claim matrix

| Claim | Capability | Feature key | Treatment | Enforcement point | Bypass test |
|---|---|---|---|---|---|
| C-010-01 intent verbs | extract/act/task | base_entitlement | route mount middleware | matrix_gate_test.go |
| C-010-02 stream citizenship | event envelopes | read_allowance | SSE mount public-token rules | envelope schema test |
| C-010-03 budget governors | task caps/pause | base_entitlement | session create guard | budget_pause_test.go |
| C-010-04 deadline envelopes | never-hang | always_available | bridge wrapper | chaos suite |
| C-010-05 cost stamps | telemetry fields | always_available | envelope builder | stamp snapshot test |
| C-010-06 warm fleet | keep-alive pools | base_entitlement | pool config gate | warm bench |
| C-010-07 state continuity | checkpoints/rollback | `web_state_continuity` | session checkpoint API | continuity round-trip test |
| C-010-08 differential DOM | patch reads | base_entitlement | snapshot route param | payload bench |
| C-010-09 artifact-ref shots | ref-default | always_available | capture path default | envelope shape test |
| C-010-10 raw FPV transport | live view | base_entitlement | FPV mount | latency probe |
| C-010-11 pressure recycle | recycler | always_available | recycler tick | leak soak |
| C-010-12 queue priorities | classes | base_entitlement | enqueue API | mixed-load bench |
| C-010-13 egress breaker/fallback | circuit | always_available (fallback flag premium-neutral) | IPPool.Pick | breaker unit tests ✅ |
| C-010-14 personas | composite identity | `persona_stealth` | persona route middleware | redaction fuzz |
| C-010-15 perception anchors | semantic refs | base_entitlement | snapshotter | rebind test |
| C-010-16 auth preflight | domain probe | base_entitlement | open hook | preflight fixture |
| C-010-17 evidence chains | receipts | base_entitlement | capture layer | tamper test |
| C-010-18 consensus reads | N>1 diffs | `consensus_reads` | orchestrator route | consensus fixture |
| C-010-19 mesh workers | remote registry | `mesh_workers` | registry route | cross-host test |
| C-010-20 cockpit contract | adapters/docs | n/a (process) | PR checklist | cockpit e2e |
| C-010-21 fleet widgets | Focusa bridge | read_allowance | daemon bridge | widget fixture |
| C-010-22 browsing beads | leases | base_entitlement | silent-session spawn | lease contention test |
| C-010-23 checkpoints=workpoints | ids in packets | `web_state_continuity` | packet builder | compaction test |
| C-010-24 north-star prefetch | prerender | base_entitlement (opt-in) | launcher hook | hit-rate counter |
| C-010-25 pairing↔persona | binding | base_entitlement | pairing claim field | binding test |
| C-010-26 ops floor | caddy/metrics/logs | recovery_exempt metrics | configs | pull-scrape test |
| C-010-27 audit ledger | hash-chained log | always_available (integrity surface) | writer boundary | tamper test |
| C-010-28 time travel | reconstruction/export | read free; **`time_travel_export`** for export | context route / export route | reconstruction test |
| C-010-30 stealth hardening | fingerprint seeds | `persona_stealth` (+premium tier floor: team+) | profile launch flags | 009 defect panel |
| C-010-31 adaptive rotation | rotation IQ | `adaptive_rotation` | rotation policy engine | flag-rotate bench |
| C-010-32 solver coverage pro | extended matrix | `solver_coverage_pro` | solver dispatch | coverage matrix test |
| C-010-33 detection telemetry | corpus/redteam | internal (never customer-facing) | fixtures dir | redteam harness |

## Existing premium surfaces (pre-010, unchanged)
Critique/UIReverse/StyleEnhance/LayoutCompare/SectionDetect/Copilot/Media/Share/Reference/DesignSystem/ContentMap/BlockRecipes/Comparison/Migration → §6.6 keys as shipped.

## Bypass-resistance suite (normative)
`internal/license/matrix_test.go` (✅ evaluation denial ×7 keys, tier-grant tracking, unknown-tier fail-closed, remote-anonymous 402 envelope) — extend per wave with each new enforcement point; production probe recorded under `docs/evidence/uiai-engine-010/licensing-probe.txt`.
