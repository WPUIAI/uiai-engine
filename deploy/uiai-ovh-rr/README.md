# UIAI OVH FPV affinity proxy

`uiai-sticky-rr.py` is the small loopback proxy behind KnownHost's authenticated SSH forward on `127.0.0.1:7456` and the OVH `cloudflared-wpuiai` tunnel.

It preserves session and FPV-share affinity across the two UIAI workers (`7456` and `7457`). `/m/{token}` requests do not carry a session ID, so the proxy persists the token-to-worker mapping; hashing alone is insufficient because the worker that creates a share is authoritative for that share's in-memory state.

Install the script as `/usr/local/sbin/uiai-sticky-rr.py`, ensure `/var/lib/uiai-ovh-rr` is root-owned and mode `0700`, then restart `uiai-ovh-rr.service`. Verify a fresh `/api/session` response with its returned `fpv_share.public_url`, then check the public page, `/status`, and `/screenshot.jpg` before touching the Cloudflare tunnel configuration.
