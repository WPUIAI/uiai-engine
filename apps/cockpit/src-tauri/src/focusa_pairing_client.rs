use serde::{Deserialize, Serialize};
use url::Url;
const MAX_SCOPES: usize = 16;
const ALLOWED_SCOPES: [&str; 2] = ["read", "write"];
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct PairStartInput {
    pub daemon_url: String,
    pub device_name: String,
    pub platform: String,
    pub requested_scopes: Vec<String>,
    pub project_id: Option<String>,
    pub continuity_id: Option<String>,
    pub local_only: bool,
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct NativePairStartRequest {
    pub endpoint: String,
    pub client_type: &'static str,
    pub device_name: String,
    pub platform: String,
    pub requested_scopes: Vec<String>,
    pub project_id: Option<String>,
    pub continuity_id: Option<String>,
}
#[derive(Clone, Debug, Eq, PartialEq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct PairStartRoom {
    pub room_id: String,
    pub nonce: String,
    pub status: String,
    pub created_at: String,
    pub expires_at: String,
    pub pair_url: Option<String>,
    pub pair_code: Option<String>,
}
pub trait PairStartTransport {
    fn start(&self, request: NativePairStartRequest) -> Result<PairStartRoom, String>;
}
fn bounded_ref(value: &str) -> bool {
    !value.is_empty()
        && value.len() <= 256
        && value
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'~' | b':' | b'-'))
}
fn validate(input: PairStartInput) -> Result<NativePairStartRequest, String> {
    let url = Url::parse(&input.daemon_url).map_err(|_| "daemon URL invalid")?;
    let loopback = url
        .host_str()
        .is_some_and(|h| matches!(h, "127.0.0.1" | "localhost" | "::1"));
    if url.username() != ""
        || url.password().is_some()
        || url.query().is_some()
        || url.fragment().is_some()
        || !matches!(url.path(), "" | "/")
    {
        return Err("daemon URL contains forbidden components".into());
    }
    if input.local_only && !loopback {
        return Err("local-only policy blocks remote pairing".into());
    }
    if (loopback && url.scheme() != "http") || (!loopback && url.scheme() != "https") {
        return Err("daemon transport invalid".into());
    }
    if input.device_name.trim().is_empty()
        || input.device_name.len() > 120
        || input.platform.trim().is_empty()
        || input.platform.len() > 40
    {
        return Err("device metadata invalid".into());
    }
    if input.requested_scopes.is_empty()
        || input.requested_scopes.len() > MAX_SCOPES
        || input
            .requested_scopes
            .iter()
            .any(|s| !ALLOWED_SCOPES.contains(&s.as_str()))
    {
        return Err("requested scopes invalid".into());
    }
    for value in [&input.project_id, &input.continuity_id]
        .into_iter()
        .flatten()
    {
        if !bounded_ref(value) {
            return Err("scope context invalid".into());
        }
    }
    Ok(NativePairStartRequest {
        endpoint: format!(
            "{}/v1/device/pair/start",
            input.daemon_url.trim_end_matches('/')
        ),
        client_type: "cockpit",
        device_name: input.device_name,
        platform: input.platform,
        requested_scopes: input.requested_scopes,
        project_id: input.project_id,
        continuity_id: input.continuity_id,
    })
}
pub fn start_pairing<T: PairStartTransport>(
    transport: &T,
    input: PairStartInput,
) -> Result<PairStartRoom, String> {
    let request = validate(input)?;
    let room = transport.start(request)?;
    if !bounded_ref(&room.room_id)
        || !bounded_ref(&room.nonce)
        || !matches!(room.status.as_str(), "created" | "awaiting_operator")
    {
        return Err("pair-start room invalid".into());
    }
    Ok(room)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Mutex;
    struct Mock {
        seen: Mutex<Vec<NativePairStartRequest>>,
    }
    impl PairStartTransport for Mock {
        fn start(&self, r: NativePairStartRequest) -> Result<PairStartRoom, String> {
            self.seen.lock().unwrap().push(r);
            Ok(PairStartRoom {
                room_id: "room_1".into(),
                nonce: "nonce_1".into(),
                status: "created".into(),
                created_at: "2026-08-03T00:00:00Z".into(),
                expires_at: "2026-08-03T00:05:00Z".into(),
                pair_url: None,
                pair_code: Some("FOCUS-1234-5678".into()),
            })
        }
    }
    fn valid() -> PairStartInput {
        PairStartInput {
            daemon_url: "https://focusa.example".into(),
            device_name: "Operator Cockpit".into(),
            platform: "macos".into(),
            requested_scopes: vec!["read".into()],
            project_id: Some("uiai-engine".into()),
            continuity_id: Some("cad135-mission-canvas".into()),
            local_only: false,
        }
    }
    #[test]
    fn emits_cockpit_pair_start_only_after_validation() {
        let t = Mock {
            seen: Mutex::new(vec![]),
        };
        let room = start_pairing(&t, valid()).unwrap();
        assert_eq!(room.room_id, "room_1");
        let r = t.seen.lock().unwrap();
        assert_eq!(r[0].client_type, "cockpit");
        assert_eq!(r[0].endpoint, "https://focusa.example/v1/device/pair/start");
    }
    #[test]
    fn local_only_and_invalid_inputs_never_reach_transport() {
        for mutate in 0..5 {
            let t = Mock {
                seen: Mutex::new(vec![]),
            };
            let mut i = valid();
            match mutate {
                0 => i.local_only = true,
                1 => i.daemon_url = "http://focusa.example:8787".into(),
                2 => i.requested_scopes = vec!["admin".into()],
                3 => i.device_name = "".into(),
                _ => i.daemon_url = "https://user:secret@focusa.example".into(),
            };
            assert!(start_pairing(&t, i).is_err());
            assert!(t.seen.lock().unwrap().is_empty());
        }
    }
}
