# W2 evidence — 2026-08-25 (partial: intent/budget LIVE; screenshot stall tracked #98)
- intent extract live x2: {"heading":"Example Domain"}, confidence=1.0, receipts rcpt_*
- budget create/status live; unit suite covers charge/exceed/pause/resume/redaction
- C-010-09 artifact-ref default verified in code+tests; live blocked by #98 pooled-page stall (intermittent)
- keepalive prelaunch reverted (launch race) — re-enable after single-flight fix per #98

## #99 ROOT CAUSE + FIX (2026-08-25 14:05 UTC)
- MultiPool(8) round-robin × lazy per-pool launch × 5m idle shutdown ⇒ chronic cold launches; Chrome-151 prewarm (~10s) collided across pools → >25s stalls alternating with fast hits.
- FIX: vision.max_pool=1 (+browsers=1) per worker; UIAI_VISION_KEEPALIVE=1 re-enabled safely on single-pool.
- RESULT (6 nocache screenshots via caddy): 218ms / 235ms / 182ms — warm fleet live (C-010-06 ✅), stalls eliminated.
