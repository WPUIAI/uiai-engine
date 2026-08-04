//! Bridge commands — sibling of Focusa menubar FirstRunWizard, but for the
//! UIAI Engine Cockpit desktop app. Same wire format, same bridge protocol
//! (`focusa-connect-v1`), same ScopeContext preservation (Spec 104 MBN-01).
//! Spec 53 §2.0, Spec 54 §B.5, §17.3.1 Path A replicated pairing.

use serde::Serialize;
use std::collections::HashMap;
use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream, UdpSocket};
use std::sync::{
    atomic::{AtomicBool, Ordering},
    Arc, Mutex, OnceLock,
};
use std::time::Duration;

const BRIDGE_CALLBACK_MAX_BODY: usize = 64 * 1024;
const REQUIRED_PROTOCOL: &str = "focusa-connect-v1";
const REQUIRED_ROLE: &str = "mac_completion_payload";

#[derive(Default)]
struct BridgeLease {
    room_id: String,
    cancel: Arc<AtomicBool>,
}

#[derive(Serialize)]
pub struct PairingBridgeDescriptor {
    pub room_id: String,
    pub callback_url: String,
    pub ttl_secs: u64,
    pub bridge_owner: &'static str,
}

#[derive(Default)]
struct BridgeState {
    completions: Mutex<HashMap<String, String>>,
    listeners: Mutex<HashMap<String, BridgeLease>>,
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

fn read_http_body(stream: &mut TcpStream) -> Result<String, String> {
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

fn opaque(value: &str) -> bool {
    (16..=256).contains(&value.len())
        && value.bytes().all(|byte| {
            byte.is_ascii_alphanumeric() || matches!(byte, b'.' | b'_' | b'~' | b':' | b'-')
        })
}

fn start_bridge(
    room_id: String,
    nonce: String,
    ttl_secs: u64,
) -> Result<PairingBridgeDescriptor, String> {
    if !opaque(&room_id) || !opaque(&nonce) {
        return Err("room and nonce must be opaque bounded references".to_string());
    }
    let ttl_secs = ttl_secs.clamp(5, 300);
    let cancel = Arc::new(AtomicBool::new(false));
    if let Ok(mut listeners) = state().listeners.lock() {
        if listeners.contains_key(&nonce) {
            return Err("callback listener already active for nonce".to_string());
        }
        listeners.insert(
            nonce.clone(),
            BridgeLease {
                room_id: room_id.clone(),
                cancel: Arc::clone(&cancel),
            },
        );
    }
    let listener =
        TcpListener::bind("0.0.0.0:0").map_err(|e| format!("callback bind failed: {e}"))?;
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
            let deadline = std::time::Instant::now() + Duration::from_secs(ttl_secs);
            loop {
                if cancel.load(Ordering::Relaxed) || std::time::Instant::now() >= deadline {
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
    Ok(PairingBridgeDescriptor {
        room_id,
        callback_url,
        ttl_secs,
        bridge_owner: "cockpit",
    })
}

#[tauri::command]
pub fn focusa_start_pairing_bridge(
    room_id: String,
    nonce: String,
    ttl_secs: u64,
) -> Result<PairingBridgeDescriptor, String> {
    start_bridge(room_id, nonce, ttl_secs)
}

#[tauri::command]
pub fn focusa_start_bridge_callback(nonce: String) -> Result<String, String> {
    let legacy_room = format!("legacy-room-{nonce}");
    Ok(start_bridge(legacy_room, nonce, 30)?.callback_url)
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
    if let Ok(mut listeners) = state().listeners.lock() {
        if let Some(lease) = listeners.remove(&nonce) {
            let _ = lease.room_id;
            lease.cancel.store(true, Ordering::Relaxed);
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pairing_bridge_owns_a_bounded_room_nonce_lease() {
        let nonce = "nonce_1234567890abcdef".to_string();
        let room = "room_1234567890abcdef".to_string();
        let descriptor = start_bridge(room.clone(), nonce.clone(), 999).unwrap();
        assert_eq!(descriptor.room_id, room);
        assert_eq!(descriptor.bridge_owner, "cockpit");
        assert_eq!(descriptor.ttl_secs, 300);
        assert!(descriptor.callback_url.ends_with(&nonce));
        assert!(start_bridge("room_abcdefghijklmnop".into(), nonce.clone(), 30).is_err());
        focusa_clear_bridge(nonce.clone()).unwrap();
        assert!(start_bridge("room_abcdefghijklmnop".into(), nonce.clone(), 5).is_ok());
        focusa_clear_bridge(nonce).unwrap();
    }

    #[test]
    fn pairing_bridge_rejects_unbounded_or_nonopaque_authority() {
        for value in ["short", "contains space!!!!!", "contains/path!!!!!"] {
            assert!(start_bridge(value.into(), "nonce_1234567890abcdef".into(), 30).is_err());
        }
    }
}
