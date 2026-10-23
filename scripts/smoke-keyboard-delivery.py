#!/usr/bin/env python3
"""Exercise real engine keyboard delivery to a backgrounded synthetic session.

Run against one engine worker (not a load balancer) so both pages share a browser.
This proves input delivery, not screenshot publication or EPWA readiness.
"""
import json
import os
import sys
from urllib.error import HTTPError
from urllib.request import Request, urlopen

BASE = os.environ.get("UIAI_ENGINE_URL", "http://127.0.0.1:7456").rstrip("/")


def request(method, path, body=None):
    headers = {"Content-Type": "application/json"}
    if os.environ.get("UIAI_API_KEY"):
        headers["X-API-Key"] = os.environ["UIAI_API_KEY"]
    elif os.environ.get("UIAI_BEARER_TOKEN"):
        headers["Authorization"] = "Bearer " + os.environ["UIAI_BEARER_TOKEN"]
    req = Request(BASE + path, data=json.dumps(body).encode() if body is not None else None, headers=headers, method=method)
    try:
        with urlopen(req, timeout=30) as response:
            return json.load(response)
    except HTTPError as exc:
        raise RuntimeError(f"keyboard probe {method} failed with HTTP {exc.code}") from None


def run():
    sessions = []
    cleanup_errors = []
    try:
        status = request("GET", "/api/status")
        first = request("POST", "/api/session", {"url": "https://example.com", "width": 800, "height": 600})
        sid = first["session"]["id"]
        sessions.append(sid)
        request("POST", f"/api/session/{sid}/eval", {"js": "window.__uiaiKeys=[]; document.addEventListener('keydown', e=>window.__uiaiKeys.push(e.key)); document.body.tabIndex=0; document.body.focus(); return 'probe-ready';"})
        second = request("POST", "/api/session", {"url": "https://example.com", "width": 800, "height": 600})
        sessions.append(second["session"]["id"])
        for key in ("a", "Enter"):
            request("POST", f"/api/session/{sid}/press", {"key": key})
        result = request("POST", f"/api/session/{sid}/eval", {"js": "return JSON.stringify({keys:window.__uiaiKeys,focused:document.hasFocus()});"})
        observed = json.loads(result["result"])
        if observed.get("keys") != ["a", "Enter"] or observed.get("focused") is not True:
            raise RuntimeError("engine acknowledged keyboard input without focused DOM keydown delivery")
        return {"schema": "uiai.keyboard_probe.v1", "input_delivery": "verified", "keys": observed["keys"], "focused": observed["focused"], "engine_version": status.get("version"), "source_commit": status.get("source_commit"), "artifact_delivery_state": result.get("delivery_state"), "truth_notice": "Input delivery only; not proof of EPWA publication or visual correctness."}
    finally:
        for sid in reversed(sessions):
            try:
                request("DELETE", f"/api/session/{sid}")
            except Exception as exc:
                cleanup_errors.append(type(exc).__name__)
        if cleanup_errors:
            raise RuntimeError("keyboard probe session cleanup failed: " + ",".join(cleanup_errors))


if __name__ == "__main__":
    try:
        print(json.dumps(run(), sort_keys=True))
    except Exception as exc:
        print(f"keyboard probe failed ({type(exc).__name__}): {exc}", file=sys.stderr)
        sys.exit(1)
