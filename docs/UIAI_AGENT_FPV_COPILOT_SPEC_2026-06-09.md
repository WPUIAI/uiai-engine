# UIAI Agent FPV Co-Pilot — Feature Spec (2026-06-09)

> **Vision**: A developer should be able to sit beside an agent driving
> the UIAI browser, see exactly what the agent sees in real time, and
> steer it in real time — by typing, by clicking on the live page, or
> by drawing on top of it. Like first-person-view (FPV) drone
> piloting, but for agents instead of drones.

**Origin**: This came out of a 4-hour WPUIAI E2E audit where the
operator (a human) was repeatedly saying "I can see the agent is on
the wrong tab, but I have to wait for it to ask me a question or
fail". The proposed feature collapses that feedback loop from
many-minutes to sub-second.

---

## 1. Why this matters

Today's agentic browser workflow is one-way: agent → browser → result
→ agent. The developer is *outside* the loop.

- Agent makes a bad click → developer finds out 30s later via a
  diagnostic
- Agent goes down a wrong path → developer can only intercept after
  the next `navigate`
- Agent gets stuck on a hard problem → developer can only intervene
  via a text message that the agent reads on the next LLM call

The fix: **mirror the agent's view in real time and let the
developer act on it directly**. This is the same shift as going from
"CI logs" to "live test runner" — observability is the prerequisite
for fast iteration.

---

## 2. User personas

### Solo developer ("P1")
Running 1-2 agents on a focused task. Wants:
- A single floating window showing the agent's browser
- A sidebar with the last 5-10 tool calls
- A text box to send messages
- Emergency stop button

### Engineering lead ("P2")
Supervising 3-10 agents in parallel across a team. Wants:
- Grid/canvas view of all active agents
- "Watch any agent live" drill-down
- Alert when an agent is stuck (no progress in 60s)
- Inject feedback into any agent's session

### QA engineer ("P3")
Auditing autonomous runs for correctness. Wants:
- Replay previous sessions with developer annotations
- Compare agent's run to a golden baseline
- Side-by-side: agent's run vs human's run
- Step-by-step execution mode (not continuous)

### Customer-support "copilot" user ("P4")
Watching an AI agent navigate a customer dashboard to resolve a
ticket. Wants:
- Privacy redaction (mask customer PII)
- Takeover: dev takes control of browser when stuck
- Co-pilot chat with the agent mid-flow

---

## 3. Core features

### 3.1 Live browser mirror

A real-time rendered view of the agent's current page, streamed at
15-30 fps. Lower fps for low-bandwidth clients; higher for local
loopback.

**Implementation options**:
- **WebSocket + PNG/JPEG frames** (simple, 50-200 KB/s per session)
- **CDP `Page.startScreencast`** (Chrome DevTools Protocol native)
- **WebRTC peer connection** (lowest latency, most complex)
- **H.264-encoded stream** (best for many concurrent sessions)

**Modes**:
- **Visual**: the rendered page as a human would see it
- **Accessibility tree**: text-only A11y tree (lightweight, useful
  for long sessions)
- **DOM diff**: only highlights changes since last action
- **Selector-targeted**: only show a specific element + its context

### 3.2 Data sidebar

A dockable panel with structured real-time data:

```
┌─ Session: LgwqFy4b ──────────────────┐
│ URL: wpuiai.com/admin.php?page=wpuiai  │
│ Active object: settings-tab=ai        │
│ Workpoint: 019eae3c-...               │
│ Current action: fill max_iterations=5 │
│                                       │
│ ── Recent tool calls ──                │
│ 12:34:05 fill max_iterations=5  [OK] │
│ 12:34:04 click input[name=submit] [OK]│
│ 12:34:02 navigate settings&tab=ai [OK]│
│ 12:33:58 screenshot 1280x800    [OK] │
│                                       │
│ ── Network ──                          │
│ POST /admin-ajax.php 200 (87ms)       │
│ POST /admin-ajax.php 400 (5ms) [!!]  │
│ GET  /assets/main.css 200             │
│                                       │
│ ── Console ──                          │
│ [warn] Deprecated: element.focus()    │
│                                       │
│ ── Evidence ──                         │
│ [img] settings-1280 (2s ago)          │
│ [img] settings-375 (8s ago)          │
│                                       │
│ ── Agent reasoning (last LLM call) ── │
│ "I'll update the autopilot settings..." │
└───────────────────────────────────────┘
```

Filterable: hide `OK` calls, only show `failed`, only show
`network`, etc.

### 3.3 Developer feedback channel

A text input that sends a message to the agent. Two delivery modes:

**Mode 1: Next-turn injection (gentle)**
Message is appended to the agent's next LLM call context. Agent
sees it as: "DEVELOPER SAYS: <message>". This is the default.

**Mode 2: Takeover (steering)**
The developer can:
- Type in any field directly (the field receives the developer's
  input as if it were a UIAI fill)
- Click any element (clicks map to UIAI selectors)
- Draw on the page to highlight an area (generates a "look at this
  region" message with the bounding box)
- Press keys (Ctrl+S, Enter, etc.) that map to UIAI press

When in takeover mode, the agent's queued actions are paused.
When the developer "releases control" (button or keyboard shortcut),
the agent resumes with the developer input as a "DEVELOPER ACTED:
<action>" entry in its context.

### 3.4 Annotation layer

The developer can draw on top of the mirrored page:
- Circles around elements
- Arrows pointing to things
- Text annotations

Annotations are sent as a message: "DEVELOPER ANNOTATION: <text>
at <selector>".

### 3.5 Pause / takeover / resume

Three explicit states for the agent:

- **RUN**: agent is making its own decisions
- **PAUSE**: agent's next action is delayed; developer can interact
- **TAKEOVER**: developer controls the browser; agent's queue is
  held

Transition: a single toolbar button + keyboard shortcut (default
`Cmd+Shift+T`).

### 3.6 Multi-agent canvas

For P2 (engineering lead): grid or canvas view of all active
agents. Click any cell to drill in. Each cell shows:
- Mini-screenshot (throttled to 1 fps to save bandwidth)
- Current URL
- Status indicator (running, paused, error, idle)
- Last tool call

### 3.7 Recording and replay

Every session is recorded. After the session, the developer can:
- Replay step-by-step
- Pause at any tool call
- "Fork from here" with developer actions inserted

The recording is stored as a sequence of: `[{tool_call, screenshot_before,
screenshot_after, network_log, console_log, duration_ms}]`.

---

## 4. Technical architecture

### 4.1 Data flow

```
┌──────────────────┐
│  UIAI Server      │
│                  │
│  CDP Browser ────┼──> WebSocket stream
│                  │    (screencast frames)
│  Tool executor ──┼──> WebSocket events
│                  │    (tool_started, tool_completed, etc.)
│  LLM broker ─────┼──> WebSocket events
│                  │    (reasoning_chunks, message_received)
│  Evidence store ─┼──> REST/GraphQL
└──────────────────┘
        │
        │ WebSocket fan-out
        ▼
┌──────────────────┐
│  FPV Client       │
│  (Mac app)        │
│                  │
│  ┌────────────┐  │
│  │ Browser    │  │◀── visual mirror
│  │ mirror     │  │
│  └────────────┘  │
│  ┌────────────┐  │
│  │ Data       │  │◀── live tool calls, network, console
│  │ sidebar    │  │
│  └────────────┘  │
│  ┌────────────┐  │
│  │ Feedback   │  │◀── text + annotation + takeover
│  │ bar        │  │
│  └────────────┘  │
│                  │
│  ┌────────────┐  │
│  │ Multi-agent│  │◀── canvas view
│  │ canvas     │  │
│  └────────────┘  │
└──────────────────┘
```

### 4.2 WebSocket protocol

Mirror channel: streams of:
```json
{ "type": "frame", "data": "<base64 jpeg>", "seq": 1234, "viewport": [1280, 800] }
```

Event channel: structured events:
```json
{ "type": "tool_started", "tool": "uiai_browser_click", "args": {...} }
{ "type": "tool_completed", "tool": "uiai_browser_click", "status": "ok", "duration_ms": 87 }
{ "type": "navigation", "url": "...", "title": "..." }
{ "type": "network", "method": "POST", "url": "/admin-ajax.php", "status": 400 }
{ "type": "console", "level": "warn", "text": "..." }
{ "type": "reasoning", "text": "I'll update..." }
{ "type": "evidence_captured", "handle": "uiai:session=X:8" }
```

Command channel (reverse direction):
```json
{ "type": "pause" }
{ "type": "resume" }
{ "type": "takeover" }
{ "type": "release" }
{ "type": "send_message", "text": "the email field is second" }
{ "type": "annotate", "selector": "input[type=email]", "text": "use this one" }
{ "type": "execute_tool", "tool": "uiai_browser_fill", "args": {...} }
```

### 4.3 Performance budget

- **Frame rate target**: 15 fps for visual mirror (50ms/frame budget)
- **Latency budget**: <100ms p50, <250ms p95 for feedback round-trip
- **Bandwidth budget**: 500 KB/s per active session for visual mirror
- **Frame skipping**: skip frames if consumer is behind; use frame
  diff for static regions
- **Quality adaptation**: drop to 5 fps or 1 fps when window is
  minimized; restore when focused

### 4.4 Security / privacy

- **Default**: dev sees what agent sees (no redaction)
- **P4 mode** (customer-support): PII redaction in the mirror; dev
  can only see redacted DOM
- **Takeover mode**: auth credentials may need to be re-entered
  depending on policy
- **Recording**: opt-in per session; default off

---

## 5. Mac app spec

### 5.1 Tech stack

- **SwiftUI** for the UI
- **AppKit** for window management (floating, always-on-top)
- **URLSessionWebSocketTask** for WebSocket client
- **WKWebView** for the local mirror rendering (receives frames
  via JS bridge or direct bitmap stream)
- **AVFoundation** for video encoding if using H.264

### 5.2 Window layout

```
┌──────────────────────────────────────────────┐
│ ●●  FPV — LgwqFy4b (Browser mirror)         [X]│
├──────────────────────────────────┬───────────┤
│                                  │ Sidebar   │
│                                  │ ── ── ──  │
│                                  │           │
│      [Browser mirror canvas]     │           │
│                                  │           │
│                                  │           │
│                                  │           │
├──────────────────────────────────┴───────────┤
│ [Annotate] [Take over] [Pause] | Type here...│
└──────────────────────────────────────────────┘
```

### 5.3 Three modes

**Pinned mode**: floats above other apps, always-on-top, dev keeps
it on the second monitor.

**Embedded mode**: webview served by the UIAI engine; dev embeds
it in their editor (VSCode, PhpStorm) as a sidebar.

**Pop-out mode**: full screen or in-app window; for focused review
of a long-running audit.

### 5.4 Mac-native features

- **Dock badge**: shows number of active agent sessions
- **Touch Bar**: pause/takeover/resume buttons
- **Menu bar item**: status indicator (running/idle/error)
- **Touch Bar scrubbing**: drag to scrub through recorded session
- **Spotlight integration**: "open FPV for last session"
- **Universal clipboard**: copy a URL from the mirror, paste in
  editor
- **System notifications**: alert when an agent is stuck
- **Voice input** (macOS 14+): speak feedback to the agent

### 5.5 Cross-platform first

Build the FPV as a **Tauri or Electron app** first, so it works on
Linux/Windows during the early period. The SwiftUI Mac app is a
v2 with deeper system integration.

### 5.6 Open URL scheme

`uiai://mirror/<session-id>` opens the FPV for a specific session.
Useful for:
- `uiai://mirror/LgwqFy4b` from the terminal
- Integration with IDE bookmarks
- Slack/Discord links: "click here to watch this agent run"

---

## 6. Concrete UX moments

### 6.1 The "agent is going down the wrong path" moment

**Without FPV**: dev sees a diagnostic 30s later saying "I clicked
Cancel on the wrong tab". Dev sends "no, go back, click the
Connections tab". Agent reads it on next turn. 60s wasted.

**With FPV**: dev sees the agent click Cancel on the wrong tab.
Dev clicks the Connections tab in the mirror. The agent's
takeover state pauses the queue. Dev types "Yes, that's the right
one. Now fill the Unsplash App ID with 851295." Agent resumes with
"DEVELOPER ACTED: click tab=connections. DEVELOPER SAYS: 'Yes
...'" in context. 5s elapsed.

### 6.2 The "I want to see what the agent sees" moment

**Without FPV**: dev asks "what's on screen?" Agent responds "I'm on
the Settings → Development tab". Dev says "describe the layout".
60s of back-and-forth.

**With FPV**: dev looks at the mirror. Done.

### 6.3 The "agent is stuck in a loop" moment

**Without FPV**: dev sees 5 failed tool calls in diagnostics. Asks
"why are you stuck?" Agent re-explains. Loop continues.

**With FPV**: dev sees the visual loop (click X, navigate, click X,
navigate...). Hits PAUSE, types "stop clicking the same thing,
take a different approach". Resume.

### 6.4 The "I want to take over for a minute" moment

**Without FPV**: dev asks agent to navigate to a specific URL.
Agent does it. Slow.

**With FPV**: dev clicks TAKEOVER. Browser is now under dev's
direct control (via CDP remote). Dev navigates, fills, clicks
whatever they need. Hits RELEASE. Agent resumes with "DEVELOPER
ACTED: navigate, fill x, click y" in context.

---

## 7. Why this is a separate feature, not a tweak

The FPV feature touches:
- A new native-app distribution channel (SwiftUI, App Store)
- New streaming protocol (WebSocket + frame encoding)
- New command channel (reverse direction)
- New permissions/security model (PII redaction)
- New privacy model (recording, takeover audit)
- New "agent pause/takeover" lifecycle state
- New Focusa integration (workpoint auto-linked to mirror session)

This is bigger than "add an option to screenshot output". It is a
new product surface.

---

## 8. Open questions for the operator

1. **Single-session or multi-session first?** Solo dev needs single
   session first; engineering lead needs multi-session. Recommend
   shipping single-session, multi-session v2.
2. **WebSocket or HTTP polling?** WebSocket is better for live
   streaming, but adds operational complexity. Recommend WS.
3. **SwiftUI native or cross-platform first?** Cross-platform
   (Tauri/Electron) gets us to more users faster. Recommend
   Tauri-v1, SwiftUI-v2.
4. **Open-source or commercial?** This is a high-leverage feature.
   Open-sourcing would let the community contribute integrations.
5. **Pricing model?** Standalone Mac app is a natural \$/month
   price tier; OSS engine + paid FPV client is a common model.
6. **Privacy-first or power-first?** The P4 (customer-support)
   persona needs PII redaction. P1 (solo dev) does not. Should
   the default be privacy-first (slower) or power-first (less
   safe)?

---

## 9. Acceptance criteria

A first-pass FPV is "done" when:
- [ ] Dev can see the agent's browser mirror in <500ms after
      `uiai_browser_open`
- [ ] Mirror updates at ≥15 fps on a normal network
- [ ] Dev can type a message and the agent receives it within
      200ms
- [ ] Dev can click an element and the agent's takeover mode is
      entered within 100ms
- [ ] Multiple sessions can be viewed in a canvas
- [ ] Session recordings are replayable
- [ ] The data sidebar is filterable
- [ ] The Mac app is available as a standalone download
- [ ] The Mac app's URL scheme `uiai://mirror/<id>` opens the
      right session

---

## 10. How this fits the agent DX arc

This is the missing piece between "agent runs autonomously" and
"agent runs as a co-pilot". The current UIAI is the first; the FPV
is the second. Both are needed:

- **Autonomous mode** (current): agent runs to completion; dev
  reviews evidence after
- **Co-pilot mode** (proposed): dev watches and steers; agent
  runs in real time

Different users want different modes. Both should be first-class.
