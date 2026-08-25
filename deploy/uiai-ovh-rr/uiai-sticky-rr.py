import hashlib
import http.server
import json
import os
import socketserver
import threading
import urllib.error
import urllib.request

TARGETS = [target.strip() for target in os.environ.get("UIAI_RR", "127.0.0.1:7456,127.0.0.1:7457").split(",") if target.strip()]
SKIP = {"host", "connection", "content-length", "transfer-encoding", "content-encoding", "accept-encoding"}
STATE_PATH = os.environ.get("UIAI_RR_STATE", "/var/lib/uiai-ovh-rr/affinity.json")
SESSION_TARGET = {}
SHARE_TARGET = {}
SESSION_LOCK = threading.Lock()
STATE_LOCK = threading.Lock()
CREATE_COUNTER = 0


def load_affinity():
    try:
        with open(STATE_PATH, encoding="utf-8") as stream:
            state = json.load(stream)
        for key, target in state.get("sessions", {}).items():
            if target in TARGETS:
                SESSION_TARGET[key] = target
        for key, target in state.get("shares", {}).items():
            if target in TARGETS:
                SHARE_TARGET[key] = target
    except (FileNotFoundError, OSError, ValueError, TypeError):
        pass


def save_affinity():
    with SESSION_LOCK:
        state = {"sessions": dict(SESSION_TARGET), "shares": dict(SHARE_TARGET)}
    directory = os.path.dirname(STATE_PATH)
    try:
        os.makedirs(directory, mode=0o700, exist_ok=True)
        temporary = f"{STATE_PATH}.{os.getpid()}.tmp"
        with STATE_LOCK:
            with open(temporary, "w", encoding="utf-8") as stream:
                json.dump(state, stream, sort_keys=True)
                stream.write("\n")
            os.chmod(temporary, 0o600)
            os.replace(temporary, STATE_PATH)
    except OSError:
        # Affinity remains functional for the current process if persistence is unavailable.
        pass


def stable_target(key):
    digest = hashlib.sha256(key.encode("utf-8")).hexdigest()
    return TARGETS[int(digest, 16) % len(TARGETS)]


def parse_session_id(body, path):
    try:
        if body:
            data = json.loads(body.decode(errors="replace"))
            sid = data.get("session_id") or (data.get("session") or {}).get("id") or data.get("id")
            if sid:
                return sid
    except (TypeError, ValueError):
        pass
    parts = path.split("?", 1)[0].strip("/").split("/")
    if "session" in parts:
        index = parts.index("session")
        if index + 1 < len(parts):
            return parts[index + 1]
    return ""


def parse_share_token(path):
    parts = path.split("?", 1)[0].strip("/").split("/")
    return parts[1] if len(parts) > 1 and parts[0] == "m" else ""


def parse_response_affinity(raw):
    try:
        data = json.loads(raw.decode(errors="replace"))
    except (TypeError, ValueError):
        return "", []
    sid = data.get("session_id") or (data.get("session") or {}).get("id") or data.get("id") or ""
    tokens = []
    for candidate in (data.get("fpv_share"), data.get("share"), data):
        if isinstance(candidate, dict) and candidate.get("token"):
            tokens.append(candidate["token"])
    return sid, tokens


def worker_for(method, path, body):
    token = parse_share_token(path)
    if token:
        with SESSION_LOCK:
            target = SHARE_TARGET.get(token)
        return target or stable_target(token)
    sid = parse_session_id(body, path)
    if sid:
        with SESSION_LOCK:
            target = SESSION_TARGET.get(sid)
        return target or stable_target(sid)
    if method == "POST" and path.rstrip("/") in {"/api/session", "/api/fpv/share"}:
        global CREATE_COUNTER
        with SESSION_LOCK:
            target = TARGETS[CREATE_COUNTER % len(TARGETS)]
            CREATE_COUNTER += 1
        return target
    return stable_target(path)


class Handler(http.server.BaseHTTPRequestHandler):
    log_message = lambda *args, **kwargs: None

    def proxy(self):
        length = int(self.headers.get("content-length") or 0)
        body = self.rfile.read(length) if length else b""
        target = worker_for(self.command, self.path, body)
        forwarded_headers = {key: value for key, value in self.headers.items() if key.lower() not in SKIP}
        if body:
            forwarded_headers["Content-Length"] = str(len(body))
        forwarded_headers["Accept-Encoding"] = "identity"
        try:
            request = urllib.request.Request(
                f"http://{target}{self.path}",
                data=body if body else None,
                method=self.command,
                headers=forwarded_headers,
            )
            with urllib.request.urlopen(request, timeout=180) as response:
                raw = response.read()
                if self.command == "POST" and response.status < 300:
                    sid, tokens = parse_response_affinity(raw)
                    changed = False
                    with SESSION_LOCK:
                        if sid:
                            changed = SESSION_TARGET.setdefault(sid, target) != target or changed
                        for token in tokens:
                            changed = SHARE_TARGET.setdefault(token, target) != target or changed
                    if changed:
                        save_affinity()
                elif self.command == "DELETE":
                    sid = parse_session_id(body, self.path)
                    if sid:
                        with SESSION_LOCK:
                            SESSION_TARGET.pop(sid, None)
                        save_affinity()
                self.send_response(response.status)
                for key, value in response.headers.items():
                    if key.lower() not in {"transfer-encoding", "content-encoding", "content-length", "connection"}:
                        self.send_header(key, value)
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)
        except urllib.error.HTTPError as error:
            raw = error.read()
            self.send_response(error.code)
            self.send_header("Content-Length", str(len(raw)))
            self.end_headers()
            self.wfile.write(raw)
        except Exception as error:
            raw = str(error).encode()
            self.send_response(502)
            self.send_header("Content-Length", str(len(raw)))
            self.end_headers()
            self.wfile.write(raw)

    do_GET = proxy
    do_POST = proxy
    do_PUT = proxy
    do_DELETE = proxy


if not TARGETS:
    raise SystemExit("UIAI_RR must contain at least one backend")
load_affinity()
socketserver.TCPServer.allow_reuse_address = True
with socketserver.ThreadingTCPServer(("127.0.0.1", 7460), Handler) as server:
    print("sticky-rr 7460 ->", TARGETS, "with persistent session/share affinity", flush=True)
    server.serve_forever()
