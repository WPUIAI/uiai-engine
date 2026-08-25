use serde::{Deserialize, Serialize};
use url::Url;
const ALLOWED_SCOPES: [&str; 2] = ["read", "write"];
#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct PairStartInput {
    pub daemon_url: String,
    pub device_name: String,
    pub platform: String,
    pub requested_scopes: Vec<String>,
    pub nonce: String,
    pub callback_url: String,
    pub local_only: bool,
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct NativePairStartRequest {
    #[serde(skip)]
    pub endpoint: String,
    #[serde(skip)]
    pub client_type: &'static str,
    pub mac_name: String,
    pub mac_nonce: String,
    pub mac_callback: String,
    pub server_url: String,
    pub scopes: Vec<String>,
}
#[derive(Clone, Debug, Eq, PartialEq, Deserialize)]
pub struct FocusaFirstrunResponse {
    pub status: String,
    pub room_id: String,
    pub device_id: String,
    pub server_url: String,
    pub pair_url: String,
    pub scopes: Vec<String>,
    pub expires_at: String,
    pub expires_in_secs: u64,
    pub poll_url: String,
}
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct PairStartRoom {
    pub room_id: String,
    pub nonce: String,
    pub device_id: String,
    pub status: &'static str,
    pub daemon_url: String,
    pub pair_url: String,
    pub scopes: Vec<String>,
    pub expires_at: String,
    pub poll_url: String,
}
pub trait PairStartTransport {
    fn start(&self, request: NativePairStartRequest) -> Result<FocusaFirstrunResponse, String>;
}
fn opaque(v: &str) -> bool {
    (16..=256).contains(&v.len())
        && v.bytes()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'~' | b':' | b'-'))
}
fn validate(i: PairStartInput) -> Result<NativePairStartRequest, String> {
    let daemon = Url::parse(&i.daemon_url).map_err(|_| "daemon URL invalid")?;
    let loopback = daemon
        .host_str()
        .is_some_and(|h| matches!(h, "127.0.0.1" | "localhost" | "::1"));
    if daemon.username() != ""
        || daemon.password().is_some()
        || daemon.query().is_some()
        || daemon.fragment().is_some()
        || !matches!(daemon.path(), "" | "/")
    {
        return Err("daemon URL contains forbidden components".into());
    }
    if i.local_only && !loopback {
        return Err("local-only policy blocks remote pairing".into());
    }
    if (loopback && daemon.scheme() != "http") || (!loopback && daemon.scheme() != "https") {
        return Err("daemon transport invalid".into());
    }
    if i.device_name.trim().is_empty()
        || i.device_name.len() > 100
        || i.platform.trim().is_empty()
        || i.platform.len() > 20
        || !opaque(&i.nonce)
    {
        return Err("device offer invalid".into());
    }
    let callback = Url::parse(&i.callback_url).map_err(|_| "callback URL invalid")?;
    if callback.scheme() != "http"
        || callback
            .host_str()
            .is_none_or(|h| h.parse::<std::net::IpAddr>().is_err())
        || !callback.path().ends_with(&i.nonce)
        || callback.query().is_some()
        || callback.fragment().is_some()
    {
        return Err("callback URL invalid".into());
    }
    if i.requested_scopes.is_empty()
        || i.requested_scopes.len() > 2
        || i.requested_scopes
            .iter()
            .any(|s| !ALLOWED_SCOPES.contains(&s.as_str()))
    {
        return Err("requested scopes invalid".into());
    }
    Ok(NativePairStartRequest {
        endpoint: format!(
            "{}/v1/connect/room/firstrun",
            i.daemon_url.trim_end_matches('/')
        ),
        client_type: "cockpit",
        mac_name: format!("Cockpit · {} · {}", i.device_name.trim(), i.platform),
        mac_nonce: i.nonce,
        mac_callback: i.callback_url,
        server_url: i.daemon_url,
        scopes: i.requested_scopes,
    })
}
pub fn start_pairing<T: PairStartTransport>(
    transport: &T,
    input: PairStartInput,
) -> Result<PairStartRoom, String> {
    let nonce = input.nonce.clone();
    let daemon = input.daemon_url.clone();
    let expected_scopes = input.requested_scopes.clone();
    let response = transport.start(validate(input)?)?;
    if response.status != "waiting_for_phone"
        || !opaque(&response.room_id)
        || !opaque(&response.device_id)
        || response.scopes != expected_scopes
        || response.expires_in_secs < 5
        || response.expires_in_secs > 600
    {
        return Err("Focusa firstrun room invalid".into());
    }
    let origin = Url::parse(&daemon)
        .map_err(|_| "daemon URL invalid")?
        .origin()
        .ascii_serialization();
    for value in [&response.server_url, &response.pair_url, &response.poll_url] {
        if Url::parse(value)
            .map_err(|_| "room URL invalid")?
            .origin()
            .ascii_serialization()
            != origin
        {
            return Err("room URL origin mismatch".into());
        }
    }
    Ok(PairStartRoom {
        room_id: response.room_id,
        nonce,
        device_id: response.device_id,
        status: "awaiting_operator",
        daemon_url: daemon,
        pair_url: response.pair_url,
        scopes: response.scopes,
        expires_at: response.expires_at,
        poll_url: response.poll_url,
    })
}
#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Mutex;
    struct Mock {
        seen: Mutex<Vec<NativePairStartRequest>>,
    }
    impl PairStartTransport for Mock {
        fn start(&self, r: NativePairStartRequest) -> Result<FocusaFirstrunResponse, String> {
            self.seen.lock().unwrap().push(r);
            Ok(FocusaFirstrunResponse {
                status: "waiting_for_phone".into(),
                room_id: "room_1234567890abcdef".into(),
                device_id: "device_1234567890abcd".into(),
                server_url: "https://focusa.example".into(),
                pair_url: "https://focusa.example/connect/firstrun?mac_offer=x".into(),
                scopes: vec!["read".into()],
                expires_at: "2026-08-03T00:05:00Z".into(),
                expires_in_secs: 300,
                poll_url: "https://focusa.example/v1/connect/room/room_1234567890abcdef/status"
                    .into(),
            })
        }
    }
    fn valid() -> PairStartInput {
        PairStartInput {
            daemon_url: "https://focusa.example".into(),
            device_name: "Operator".into(),
            platform: "macos".into(),
            requested_scopes: vec!["read".into()],
            nonce: "nonce_1234567890abcdef".into(),
            callback_url: "http://192.168.1.2:9999/focusa-phone-bridge/nonce_1234567890abcdef"
                .into(),
            local_only: false,
        }
    }
    #[test]
    fn uses_exact_focusa_v2_firstrun_contract() {
        let t = Mock {
            seen: Mutex::new(vec![]),
        };
        let room = start_pairing(&t, valid()).unwrap();
        assert_eq!(room.status, "awaiting_operator");
        let r = t.seen.lock().unwrap();
        assert_eq!(
            r[0].endpoint,
            "https://focusa.example/v1/connect/room/firstrun"
        );
        assert_eq!(r[0].client_type, "cockpit");
        assert_eq!(r[0].mac_nonce, "nonce_1234567890abcdef");
    }
    #[test]
    fn invalid_policy_never_reaches_transport() {
        for n in 0..4 {
            let t = Mock {
                seen: Mutex::new(vec![]),
            };
            let mut i = valid();
            match n {
                0 => i.local_only = true,
                1 => i.requested_scopes = vec!["admin".into()],
                2 => i.nonce = "short".into(),
                _ => i.callback_url = "https://evil.example/x".into(),
            };
            assert!(start_pairing(&t, i).is_err());
            assert!(t.seen.lock().unwrap().is_empty());
        }
    }
}
