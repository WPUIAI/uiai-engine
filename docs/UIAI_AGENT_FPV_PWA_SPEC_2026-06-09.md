# UIAI Agent FPV — PWA Variant (2026-06-09)

> **Security-corrected vision**: Every UIAI session starts private. When an
> authorized operator explicitly requests a read-only share for a non-sensitive
> origin, UIAI may issue one bounded capability URL. Session creation never
> creates or announces a share implicitly. Public control remains disabled until
> a separate governed-confirmation contract exists.

> This is the **fast-path MVP** of the full FPV Co-Pilot spec
> (`UIAI_AGENT_FPV_COPILOT_SPEC_2026-06-09.md`). The Mac app /
> Tauri / IDE-embedded versions come in Phase 2.

**Implementation status (2026-09-03)**: Security correction is a source candidate, not installed acceptance. Explicit `POST /api/fpv/share` may create a bounded read-only `/m/{token}` capability for an exact non-sensitive session origin. `browser_open` returns no share. Auth, privacy, payment, and health origins are denied. `controls=true` is denied until governed confirmation exists. Older automatic-share and public-control prose below is historical design context and does not describe authorized current behavior.

---

## 1. Why PWA, not a native app

A previous iteration of this idea (a SwiftUI Mac app) was the
"correct architecturally but wrong distribution":

- Native apps need **install friction** (download, sign in, grant
  permissions)
- Distribution channel is **App Store**, with all its gatekeeping
- Cross-platform (Linux, Windows) requires **separate code paths**
- Updates need to **wait for app store review**

A **PWA** solves all of this:

- **Zero install**: just open the URL in any browser
- **Cross-device**: works on iPhone, Android, iPad, laptop,
  Raspberry Pi browser
- **Shareable link**: Slack message, email, QR code, even SMS
- **Auto-updates**: server deploys, users always get latest
- **PWA installable**: "Add to Home Screen" for native-feel launch
- **Offline-capable**: service worker handles dropped connections
- **Push notifications**: agent-stuck alerts even when tab is closed

---

## 2. User flow

### 2.1 Session start

```
Agent: "I need to verify the WPUIAI admin settings. Starting a session."

uiai_browser_open({
  url: "https://wpuiai.com/wp-admin/admin.php?page=wpuiai-settings",
  auth_profile: "wpuiai-test-admin"
})

Response: {
  "session_id": "LgwqFy4b",
  "screenshot": "...",
  "size": {"width": 390, "height": 844}
}

Agent: "Started private session LgwqFy4b. No public share was created."
```

### 2.2 Operator opens link on phone

The phone shows:

```
┌─────────────────────────────────┐
│ ●●  Watching: LgwqFy4b          │
│ ─────────────────────────────── │
│                                 │
│ ┌─────────────────────────────┐ │
│ │                             │ │
│ │   [live browser mirror]      │ │
│ │                             │ │
│ └─────────────────────────────┘ │
│ ─────────────────────────────── │
│ Tools: navigate • fill • click  │
│ Network: 2 / 0 errors           │
│ Status: running                │
│                                 │
│ [Type a message to agent...]  │
│ [Take over] [Pause] [Annotate] │
└─────────────────────────────────┘
```

### 2.3 Future governed steering — not implemented

The following interaction sketch is historical design input only. The current public route is read-only and rejects control capability creation.

Operator types "the Unsplash field is the FIRST one on the page,
not the second". Agent's next LLM call sees this as developer
feedback, recalibrates, and continues.

Or operator clicks "Take over", manually clicks the right field,
fills the value, clicks "Release" — agent resumes with the
developer's actions baked into its context.

---

## 3. The link itself

### 3.1 URL structure

The supported public form is `https://fpv.wpuiai.com/m/<capability>`.

The capability is a 192-bit URL-safe random value stored only in the protected registry and returned by the explicit share operation. The registry separately binds policy version, session ID, exact origin, expiry, read-only mode, and maximum views. Diagnostics, audit events, revocation receipts, and notifications use only a SHA-256 share reference; they never echo the capability.

### 3.2 URL generation

**When**: Only after a separate explicit share request passes policy. Never during `uiai_browser_open`.

**Operator UX**: Treat the returned URL as a sensitive capability. Do not place it in model-visible logs, diagnostics, receipts, or notifications.

**TTL and use bounds**: Default 15 minutes, maximum 60 minutes; default 25 views, maximum 100, with one-time mode fixed to one view. Extension requires a new explicit share after policy reevaluation.

### 3.3 Operator authentication

Three modes, decided by Focusa policy:

- **Bounded read-only capability**: current source candidate; exact session/origin, expiry, and view bounds apply.
- **Focusa-authenticated**: future; not implemented by this candidate.
- **Governed control**: future; fail closed until a stronger confirmation/approval contract is implemented.

For P4 (customer-support) persona, the link can be **scoped to
a specific customer session** with **PII redaction** in the
mirror.

---

## 4. PWA architecture

### 4.1 Tech stack

- **Plain HTML/CSS/JS** (no build step required)
- **Service Worker** for offline support
- **WebSocket** for live frame + event streaming
- **WebRTC** (optional) for sub-100ms low-latency mode
- **Service Worker Push API** for background alerts

### 4.2 PWA shell

The PWA is a small (~50KB) app served by the UIAI engine at
`https://uiai.example.com/m/<id>`. It contains:

- **Frame canvas** (auto-scales to viewport, pinch-zoom)
- **Sidebar tabs** (Tools, Network, Console, Evidence, Reasoning)
- **Feedback bar** (text input + send)
- **Takeover toolbar** (Take over / Pause / Release / Annotate)

### 4.3 Mobile-first UX

- **Vertical layout**: mirror on top, sidebar scrollable, feedback at bottom
- **Pinch-zoom + pan** on the mirror
- **Tap-and-hold** on an element → annotation
- **Swipe left/right** on sidebar to switch tabs
- **Pull to refresh** triggers a forced re-snapshot

### 4.4 Connection modes

- **WebSocket**: default, low-latency, ~15 fps
- **Server-Sent Events (SSE)**: fallback when WS blocked, ~5 fps
- **HTTP polling**: last-resort for corporate firewalls, ~1 fps
- **WebRTC**: opt-in for power users with sub-100ms latency

The PWA auto-detects the best available mode and falls back
gracefully.

---

## 5. Mobile-specific affordances

### 5.1 Touch interactions

- **Pinch-zoom on mirror** → zoom up to 5x for fine detail
- **Tap-and-hold on element** → opens annotation popover
- **Two-finger tap** → quick screenshot of current frame
- **Swipe up from bottom** → expand feedback bar to full screen
- **Long-press on screenshot** → save to phone

### 5.2 Mobile-specific visualizations

- **Tap to focus an element** → the agent receives a "look at this
  element" message with the selector
- **Voice input** (mobile keyboard mic button) → dictation sent
  to agent
- **Shake phone** → pause agent (useful when walking and need
  attention)
- **AirDrop** a URL to the PWA → load that URL in the agent's
  browser

### 5.3 Push notifications

The PWA registers for push notifications with these triggers:

- Agent stuck (no progress in 60s)
- Agent error (any tool failure)
- Agent asks a clarifying question
- Agent completes a major milestone (e.g. finished a settings
  update)

This means the operator can lock the phone and still get
notifications.

---

## 6. Connection states & resilience

### 6.1 Disconnect handling

If the PWA loses connection:
- Show a "Reconnecting…" banner
- Try exponential backoff (1s, 2s, 4s, 8s, 30s, 60s)
- Once reconnected, replay missed events from `last_event_seq`
- If reconnect fails after 5 minutes, show a "session may be
  paused" message

### 6.2 Multi-tab handling

Multiple tabs may consume separate bounded views of the same read-only capability. Concurrent view counting is atomic. Takeover mode is not implemented; a future governed-control design must add non-replayable operator binding and stronger confirmation before enabling it.

### 6.3 Background behavior

When the phone is locked or PWA is backgrounded:
- PWA continues receiving events for the next 30s
- Beyond 30s, PWA pauses the frame stream
- Push notifications still fire
- Re-opening the PWA re-syncs from `last_event_seq`

---

## 7. Concrete mobile screens

### 7.1 Live mirror (home screen)

```
┌─────────────────────────────────┐
│ ●●  Watching: LgwqFy4b          │
│ ─────────────────────────────── │
│                                 │
│   [WPUIAI Dashboard rendered]    │
│                                 │
│                                 │
│ ─────────────────────────────── │
│ ● Running • Tools 12 • 0 errors│
└─────────────────────────────────┘
 [💬] [▶] [⏸] [🖍]     [Send]  
```

### 7.2 Sidebar (tools tab)

```
┌─────────────────────────────────┐
│ Tools (last 12)              [X]│
│ ─────────────────────────────── │
│ 12:34  fill max_iterations=5 ✓ │
│ 12:34  click submit ✓           │
│ 12:33  navigate ai ✓            │
│ 12:33  screenshot 1280x800 ✓   │
│ 12:32  resize 1280x800 ✓       │
│ ─────────────────────────────── │
│ 12:31  fill bogus_field=99 ✗   │
│        ↳ Element not found     │
│ 12:30  ...                      │
│                                 │
│ [Filter: All / Failed only]     │
└─────────────────────────────────┘
```

### 7.3 Future takeover mode — not implemented

```
┌─────────────────────────────────┐
│ ◀ BACK TO PASSIVE VIEW         │
│ ─────────────────────────────── │
│                                 │
│ [Browser under your control]    │
│ Tap any element to interact     │
│                                 │
│ ─────────────────────────────── │
│ Steer the agent:                │
│ [___________________] [Send]   │
│ [RELEASE CONTROL]               │
└─────────────────────────────────┘
```

---

## 8. Integration with Focusa

### 8.1 Share reference in Workpoint evidence

Focusa evidence may store only the token-redacted SHA-256 `share_ref`, policy decision, session/origin binding, expiry, and revocation outcome. It must not store or narrate the public capability URL. Capability transport to the explicitly authorized operator is a separate sensitive channel.

### 8.2 Operator-driven workpoint advancement

If the operator is steering the agent through the PWA, the
workpoint's `next_action` should reflect what the operator just
did, not what the agent was about to do. `focusa_to_uiai_bridge`
needs a `source: "operator_pwa"` parameter to disambiguate.

### 8.3 Workpoint ↔ PWA session binding

A Focusa workpoint can declare:

```json
{
  "workpoint": "019eae3c-...",
  "binding": {
    "uiai_session_id": "LgwqFy4b",
    "share_ref": "fpv-share:sha256:<digest>",
    "operator_steering": "read_only" | "denied"
  }
}
```

The PWA reads this on connect, enforces `operator_steering`,
and emits events tagged with `workpoint_id`.

---

## 9. Why this is the right Phase 1

| Phase 1 (PWA) | Phase 2 (Mac app / Tauri) |
|---|---|
| Zero install friction | Native window management |
| Works on any phone | Floating always-on-top |
| Auto-updates | Touch Bar / menu bar |
| Shareable link | Spotlight integration |
| PWA installable | Voice input (macOS 14+) |
| Push notifications | Native clipboard |
| 50KB PWA | 50MB native app |
| Built in a week | Built in a quarter |

The PWA is the **5% effort, 95% value** play. Ship it, learn
from usage, then decide if a Mac app is worth it.

---

## 10. Acceptance criteria

A first-pass PWA FPV is "done" when:
- [ ] `uiai_browser_open` returns a `mirror_url` in the response
- [ ] The link loads in any modern mobile browser
- [ ] The PWA shows the live mirror at ≥10 fps on WiFi
- [ ] Operator can type a message and the agent receives it
- [ ] Operator can enter takeover mode and control the browser
- [ ] Push notifications fire for "agent stuck" / "agent error"
- [ ] Operator can install the PWA to home screen
- [ ] Recording is captured for replay
- [ ] The link is signed with a 1-hour TTL
- [ ] Reconnect after disconnect works within 5 retries
- [ ] Multi-tab broadcast works (one mirror, multiple observers)

---

## 11. Open questions for the operator

1. **Generate link on every session or only when requested?** Default
   always for observability; opt-in for P4 (customer support) due
   to privacy.
2. **Mobile browser support floor?** Recommend iOS Safari 16+ and
   Android Chrome 100+ (covers ~95% of modern phones).
3. **Voice transcription** (mic input) — should this go to the agent
   or just to the operator? Recommend operator (the agent already
   reads the typed text).
4. **PII redaction** — for P4 (support) persona, which fields get
   masked by default? Recommend opt-in, not default, with a clear
   redaction list.
5. **Operator identity** — anonymous, Focusa account, or
   email-based? Recommend Focusa account for auditability.

---

## 12. How this relates to the full FPV spec

This PWA variant is the **MVP / Phase 1** of the full FPV Co-Pilot
vision. The Mac app, Tauri app, and IDE-embedded versions are
**Phase 2+**. They share the WebSocket protocol and command channel
described here, so the PWA is the foundation; the others are
alternative front-ends.

See `UIAI_AGENT_FPV_COPILOT_SPEC_2026-06-09.md` for the full
vision, including:
- Multi-agent canvas view
- Recording and replay
- PII redaction for support persona
- Takeover semantics
- Mac-native features (Touch Bar, Spotlight, voice)
