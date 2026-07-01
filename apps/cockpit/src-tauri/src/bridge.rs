//! Bridge commands — sibling of Focusa menubar FirstRunWizard, but for the
//! UIAI Engine Cockpit desktop app. Same wire format, same bridge protocol
//! (`focusa-connect-v1`), same ScopeContext preservation (Spec 104 MBN-01).
//! Spec 53 §2.0, Spec 54 §B.5, §17.3.1 Path A replicated pairing.

use std::collections::{HashMap, HashSet};
use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream, UdpSocket};
use std::sync::{Mutex, OnceLock};
use std::time::Duration;

const BRIDGE_CALLBACK_MAX_BODY: usize = 64 * 1024;
const REQUIRED_PROTOCOL: &str = "focusa-connect-v1";
const REQUIRED_ROLE: &str = "mac_completion_payload";

#[derive(Default)]
struct BridgeState {
    completions: Mutex<HashMap<String, String>>,
    listeners: Mutex<HashSet<String>>,
}

static BRIDGE_STATE: OnceLock<BridgeState> = OnceLock::new();

fn state() -> &'static BridgeState {
    BRIDGE_STATE.get_or_init(BridgeState::default)
}

fn best_local_ip() -> String {
    UdpSocket::bind("0.0.0.0:0")
        .and_then(|socket| {
            let _ = socket.connect("8.8.8.8:80");
            socket.local_addr()
        })
        .map(|addr| addr.ip().to_string())
        .unwrap_or_else(|_| "127.0.0.1".to_string())
}

fn read_http_body(mut stream: &mut TcpStream) -> Result<String, String> {
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .map_err(|e| format!("callback read timeout setup failed: {e}"))?;
    let mut buffer = vec![0_u8; 8192];
    let mut read = 0_usize;
    loop {
        let n = stream
            .read(&mut buffer[read..])
            .map_err(|e| format!("callback read failed: {e}"))?;
        if n == 0 {
            break;
        }
        read += n;
        if read >= 4 && buffer[..read].windows(4).any(|w| w == b"\r\n\r\n") {
            break;
        }
        if read == buffer.len() {
            buffer.resize(buffer.len() * 2, 0);
            if buffer.len() > BRIDGE_CALLBACK_MAX_BODY {
                return Err("callback headers too large".to_string());
            }
        }
    }
    let header_end = buffer[..read]
        .windows(4)
        .position(|w| w == b"\r\n\r\n")
        .ok_or_else(|| "callback missing HTTP header terminator".to_string())?
        + 4;
    let body = String::from_utf8_lossy(&buffer[header_end..read]).to_string();
    Ok(body)
}

fn handle_bridge_callback(mut stream: TcpStream, nonce: String) {
    let body = match read_http_body(&mut stream) {
        Ok(b) => b,
        Err(_) => {
            let _ = stream.write_all(
                b"HTTP/1.1 422 Unprocessable Entity\r\nconnection: close\r\n\r\ninvalid body",
            );
            return;
        }
    };
    if !body_bytes_are_valid_json(&body) {
        let _ = stream.write_all(
            b"HTTP/1.1 422 Unprocessable Entity\r\nconnection: close\r\n\r\ninvalid json",
        );
        return;
    }
    let parsed: serde_json::Value = match serde_json::from_str(&body) {
        Ok(v) => v,
        Err(_) => {
            let _ = stream.write_all(
                b"HTTP/1.1 422 Unprocessable Entity\r\nconnection: close\r\n\r\ninvalid json",
            );
            return;
        }
    };
    if parsed.get("protocol").and_then(|v| v.as_str()) != Some(REQUIRED_PROTOCOL)
        || parsed.get("role").and_then(|v| v.as_str()) != Some(REQUIRED_ROLE)
    {
        let _ = stream.write_all(
            b"HTTP/1.1 422 Unprocessable Entity\r\nconnection: close\r\n\r\nprotocol/role mismatch",
        );
        return;
    }
    if let Ok(mut map) = state().completions.lock() {
        map.insert(nonce.clone(), body);
    }
    let _ = stream.write_all(
        b"HTTP/1.1 200 OK\r\ncontent-type: text/plain\r\nconnection: close\r\n\r\nFocusa Phone Bridge completion received. You can return to the Mac app.",
    );
}

fn body_bytes_are_valid_json(body: &str) -> bool {
    serde_json::from_str::<serde_json::Value>(body).is_ok()
}

#[tauri::command]
pub fn focusa_start_bridge_callback(nonce: String) -> Result<String, String> {
    if nonce.trim().is_empty() {
        return Err("nonce is required".to_string());
    }
    if let Ok(mut listeners) = state().listeners.lock() {
        if listeners.contains(&nonce) {
            return Err("callback listener already active for nonce".to_string());
        }
        listeners.insert(nonce.clone());
    }
    let listener = TcpListener::bind("0.0.0.0:0")
        .map_err(|e| format!("callback bind failed: {e}"))?;
    let port = listener
        .local_addr()
        .map_err(|e| format!("callback local addr failed: {e}"))?
        .port();
    let callback_url = format!(
        "http://{}:{}/focusa-phone-bridge/{}",
        best_local_ip(),
        port,
        nonce
    );
    std::thread::spawn({
        let nonce = nonce.clone();
        move || {
            let _ = listener.set_nonblocking(true);
            let deadline = std::time::Instant::now() + Duration::from_secs(30);
            loop {
                if std::time::Instant::now() >= deadline {
                    break;
                }
                match listener.accept() {
                    Ok((stream, _)) => {
                        handle_bridge_callback(stream, nonce.clone());
                        break;
                    }
                    Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                        std::thread::sleep(Duration::from_millis(50));
                    }
                    Err(_) => break,
                }
            }
            if let Ok(mut listeners) = state().listeners.lock() {
                listeners.remove(&nonce);
            }
        }
    });
    Ok(callback_url)
}

#[tauri::command]
pub fn focusa_take_bridge_completion(nonce: String) -> Result<Option<String>, String> {
    if let Ok(mut map) = state().completions.lock() {
        if let Some(v) = map.remove(&nonce) {
            return Ok(Some(v));
        }
    }
    Ok(None)
}

#[tauri::command]
pub fn focusa_clear_bridge(nonce: String) -> Result<(), String> {
    if let Ok(mut map) = state().completions.lock() {
        map.remove(&nonce);
    }
    Ok(())
}
