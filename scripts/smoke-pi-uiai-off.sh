#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PI_EXT="$ROOT_DIR/.pi/extensions/uiai-engine.ts"

python3 - <<'PY' "$PI_EXT"
import re
import sys
from pathlib import Path

src = Path(sys.argv[1]).read_text()
handler = re.search(r'pi\.registerCommand\("uiai", \{[\s\S]*?handler: async \(args, ctx\) => \{([\s\S]*?)\n\t\t\},\n\t\}\);', src)
if not handler:
    raise SystemExit('missing /uiai command handler')
body = handler.group(1)
required = [
    'const action = String(args || "").trim().toLowerCase();',
    '["off", "hide", "clear", "disable"].includes(action)',
    'ctx.ui.setWidget("uiai-engine", undefined);',
    'pi.appendEntry(UIAI_WIDGET_STATE_ENTRY, { visible: false });',
    'ctx.ui.notify("UIAI card hidden", "info");',
]
missing = [snippet for snippet in required if snippet not in body]
if missing:
    raise SystemExit('/uiai off handler missing expected clear behavior: ' + '; '.join(missing))

off_idx = body.index('["off", "hide", "clear", "disable"].includes(action)')
clear_idx = body.index('ctx.ui.setWidget("uiai-engine", undefined);', off_idx)
return_idx = body.index('return;', clear_idx)
fetch_idx = body.index('callEngine("/api/tools/agent-card")')
if not (off_idx < clear_idx < return_idx < fetch_idx):
    raise SystemExit('/uiai off must clear and return before fetching engine card')

menu_hide = 'if (choice === "Hide UIAI card")' in body and body.count('ctx.ui.setWidget("uiai-engine", undefined);') >= 2
if not menu_hide:
    raise SystemExit('/uiai menu missing Hide UIAI card clear path')
if 'pi.on("session_start"' not in src or 'latestWidgetVisibility(ctx) === true' not in src:
    raise SystemExit('missing session_start widget visibility restore')
if '["on", "show", "enable"].includes(action)' not in body or 'pi.appendEntry(UIAI_WIDGET_STATE_ENTRY, { visible: true });' not in body:
    raise SystemExit('/uiai on/show/enable must persist visible widget state')

print('pi /uiai off smoke ok: aliases=off,hide,clear,disable persist hidden; on/show/enable persist visible; menu hide clears widget')
PY
