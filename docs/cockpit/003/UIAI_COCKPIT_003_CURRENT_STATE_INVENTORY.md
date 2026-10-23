# UIAI-COCKPIT-003 Current-State Inventory

**Scope:** T003-01 baseline before sidebar migration
**Source:** `UIAI_COCKPIT_003_SIDEBAR_NAVIGATION_IA_DND_IMPLEMENTATION_SPEC_2026-08-01_v1.0.md`

## Current implementation map

| Surface | Current file/ref | State before T003 migration | Destination |
|---|---|---|---|
| Shell/navigation | `apps/cockpit/src/routes/+layout.svelte` | Slice 0 shell; route list and footer were local markup | `CockpitShell` + manifest-backed sidebar |
| Overview | `apps/cockpit/src/routes/+page.svelte` | Phase 0 card composition | Overview/Mission Deck |
| Cards | `apps/cockpit/src/lib/cards/phase0-card-manifest.ts` | Contract-backed 14-card manifest | Mapped workspace destinations; manifest retained |
| Browser Live | `apps/cockpit/src/routes/runs/+page.svelte` | Real session/screenshot/navigation/diagnostics surface | Live workspace adapter and contextual collection |
| Documents/Research | `apps/cockpit/src/routes/documents/+page.svelte` | Real search and Source-to-Markdown capture | Documents and Research placements |
| Studio | `apps/cockpit/src/routes/studio/+page.svelte` | Real screenshot capture surface | Studio workspace |
| Evidence | `apps/cockpit/src/routes/evidence/+page.svelte` | Truthful empty state pending evidence | Evidence saved views and reports |
| Test Lab | `apps/cockpit/src/routes/test-lab/+page.svelte` | Real health/session checks | Test Lab workspace |
| Settings | `apps/cockpit/src/routes/settings/+page.svelte` | Local engine and scope settings | Fixed footer destination |
| Browser client | `apps/cockpit/src/lib/engine-client.ts` | Engine API boundary | Controllers/adapters; no direct shell HTTP |
| OTA | `apps/cockpit/src/lib/updater.ts` and `src-tauri/tauri.conf.json` | Signed update client; publication was missing | Cockpit-specific stable manifest |

## Baseline fixture coverage

The T003 fixture registry covers: Expanded, Compact, Hidden, Overlay, empty context, missing scope, and blocked entitlement. Fixtures are test-only and contain no product objects, user identities, counts, Workpoints, or backend state.

## Migration boundary

The sidebar owns presentation taxonomy, local ordering, visibility, density, and width. Browser sessions, leases, scope, authority, Evidence, entitlements, and backend objects remain owned by their existing engine/Focusa adapters and guards.
