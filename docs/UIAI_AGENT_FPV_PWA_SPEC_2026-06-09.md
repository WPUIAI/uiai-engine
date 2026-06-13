# UIAI Agent FPV — PWA Variant (2026-06-09)

> **Vision**: At the start of every UIAI session, the agent announces a
> **share link**. The operator opens the link in any mobile browser,
> gets a live view of what the agent sees, and can steer the agent
> back to the right path. No app install, no app store, no native
> distribution. Just a URL.

> This is the **fast-path MVP** of the full FPV Co-Pilot spec
> (`UIAI_AGENT_FPV_COPILOT_SPEC_2026-06-09.md`). The Mac app /
> Tauri / IDE-embedded versions come in Phase 2.

**Implementation status (2026-06-12)**: Read-only MVP implemented. Agents create a share with `POST /api/fpv/share` / Pi `uiai_fpv_share`; operators open `/m/{token}` for a public tokenized PWA that polls session status, diagnostics summary, and screenshot frames. Steering remains intentionally out of scope until the co-pilot controls slice.

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
  "mirror_url": "https://uiai.example.com/m/LgwqFy4b?tk=signed_token_xyz",
  "mirror_url_expires_at": "2026-06-09T23:00:00Z"
}

Agent: "Started session LgwqFy4b. Watch live: https://uiai.example.com/m/LgwqFy4b?tk=signed_token_xyz"
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

### 2.3 Operator steers

Operator types "the Unsplash field is the FIRST one on the page,
not the second". Agent's next LLM call sees this as developer
feedback, recalibrates, and continues.

Or operator clicks "Take over", manually clicks the right field,
fills the value, clicks "Release" — agent resumes with the
developer's actions baked into its context.

---

## 3. The link itself

### 3.1 URL structure

Three forms, all equivalent:

- `https://uiai.example.com/m/<session-id>?tk=<signed-token>`
- `https://uiai.example.com/mirror/?s=<session-id>&t=<token>`
- `https://mirror.uiai.example.com/v1/sessions/<id>?token=<token>`

The token is a JWT-style signed value with:
- `session_id` claim
- `operator_role` claim (read-only / steer / full-takeover)
- `exp` claim (default 1 hour)
- `iss` claim (UIAI server identity)

### 3.2 URL generation

**When**: At `uiai_browser_open` time. The agent's response includes
`mirror_url` automatically.

**Operator UX**: The agent can read it out:
> "Started session. Watch live at https://uiai.example.com/m/LgwqFy4b"

Or the Focusa workpoint can capture it as evidence:
> `evidence_handle: "uiai:session=LgwqFy4b:mirror_url"`

**TTL**: Default 1 hour; operator can request extension via the
PWA itself.

### 3.3 Operator authentication

Three modes, decided by Focusa policy:

- **Anonymous read-only**: anyone with the link can view
- **Focusa-authenticated**: link requires Focusa auth
- **Token-authenticated**: link contains a signed token (default)

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

If the operator has the PWA open in two tabs, both can view
(broadcast). Only one can be in takeover mode at a time.
A tab in takeover is marked with a "Steering" badge; the other
tab shows a "Another device is steering" notice.

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

### 7.3 Takeover mode

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

### 8.1 Mirror URL in workpoint evidence

When the agent reports its mirror URL, Focusa can:
- Auto-link the URL to the active workpoint
- Show a "Live" badge in the workpoint UI
- One-click open in the operator's default mobile browser
- Auto-evidence: every minute, capture a screenshot via the
  PWA-shared connection

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
    "mirror_url": "https://uiai.example.com/m/LgwqFy4b",
    "operator_steering": "allowed" | "read_only" | "denied"
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
