use crate::focusa_credentials::{
    CredentialError, CredentialHandle, CredentialSecret, CredentialStore,
};
use serde::{Deserialize, Serialize};
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct BridgeEnvelope {
    protocol: String,
    role: String,
    mac_completion_payload: String,
}
#[derive(Clone, Debug)]
pub struct PairingGrant {
    pub room_id: String,
    pub nonce: String,
    pub daemon_id: String,
    pub client_type: String,
    pub device_id: String,
    pub scopes: Vec<String>,
    pub expires_unix: i64,
    pub token: String,
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct PersistedPairing {
    pub daemon_id: String,
    pub device_id: String,
    pub scopes: Vec<String>,
    pub expires_unix: i64,
    pub token_handle: CredentialHandle,
}
pub trait CompletionTransport {
    fn exchange(&self, daemon_url: &str, payload: &str) -> Result<PairingGrant, String>;
}
fn opaque(v: &str) -> bool {
    !v.is_empty()
        && v.len() <= 256
        && v.bytes()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'~' | b':' | b'-'))
}
pub fn validate_and_persist<T: CompletionTransport, S: CredentialStore>(
    transport: &T,
    store: &S,
    daemon_url: &str,
    expected_room: &str,
    expected_nonce: &str,
    expected_scopes: &[String],
    token_handle: CredentialHandle,
    now_unix: i64,
    raw_bridge: &str,
) -> Result<PersistedPairing, String> {
    if raw_bridge.len() > 64 * 1024 {
        return Err("bridge completion oversized".into());
    }
    let envelope: BridgeEnvelope =
        serde_json::from_str(raw_bridge).map_err(|_| "bridge completion invalid")?;
    if envelope.protocol != "focusa-connect-v1"
        || envelope.role != "mac_completion_payload"
        || envelope.mac_completion_payload.is_empty()
        || envelope.mac_completion_payload.len() > 48 * 1024
    {
        return Err("bridge completion protocol invalid".into());
    }
    let mut grant = transport.exchange(daemon_url, &envelope.mac_completion_payload)?;
    if grant.room_id != expected_room
        || grant.nonce != expected_nonce
        || grant.client_type != "cockpit"
        || !opaque(&grant.daemon_id)
        || !opaque(&grant.device_id)
        || grant.expires_unix <= now_unix
        || grant.scopes != expected_scopes
        || grant.token.is_empty()
    {
        return Err("pairing grant mismatch".into());
    }
    let secret =
        CredentialSecret::new(std::mem::take(&mut grant.token)).map_err(|e| e.to_string())?;
    store
        .write(&token_handle, secret)
        .map_err(|e: CredentialError| e.to_string())?;
    Ok(PersistedPairing {
        daemon_id: grant.daemon_id,
        device_id: grant.device_id,
        scopes: grant.scopes,
        expires_unix: grant.expires_unix,
        token_handle,
    })
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::focusa_credentials::CredentialStatus;
    use std::sync::Mutex;
    struct Transport {
        grant: Mutex<Option<PairingGrant>>,
    }
    impl CompletionTransport for Transport {
        fn exchange(&self, _: &str, _: &str) -> Result<PairingGrant, String> {
            self.grant.lock().unwrap().take().ok_or("consumed".into())
        }
    }
    #[derive(Default)]
    struct Store {
        value: Mutex<Option<String>>,
    }
    impl CredentialStore for Store {
        fn write(&self, _: &CredentialHandle, s: CredentialSecret) -> Result<(), CredentialError> {
            *self.value.lock().unwrap() = Some(s.expose_native().into());
            Ok(())
        }
        fn read(&self, _: &CredentialHandle) -> Result<CredentialSecret, CredentialError> {
            Err(CredentialError::Missing)
        }
        fn delete(&self, _: &CredentialHandle) -> Result<(), CredentialError> {
            Ok(())
        }
        fn status(&self, _: &CredentialHandle) -> CredentialStatus {
            CredentialStatus::Missing
        }
    }
    fn grant() -> PairingGrant {
        PairingGrant {
            room_id: "room_1234567890".into(),
            nonce: "nonce_1234567890".into(),
            daemon_id: "daemon_1".into(),
            client_type: "cockpit".into(),
            device_id: "device_1".into(),
            scopes: vec!["read".into()],
            expires_unix: 200,
            token: "native-token".into(),
        }
    }
    fn envelope() -> String {
        "{\"protocol\":\"focusa-connect-v1\",\"role\":\"mac_completion_payload\",\"mac_completion_payload\":\"opaque-exchange\"}".into()
    }
    #[test]
    fn validates_then_persists_without_returning_token() {
        let t = Transport {
            grant: Mutex::new(Some(grant())),
        };
        let s = Store::default();
        let result = validate_and_persist(
            &t,
            &s,
            "https://focusa.example",
            "room_1234567890",
            "nonce_1234567890",
            &["read".into()],
            CredentialHandle::parse("profile:1").unwrap(),
            100,
            &envelope(),
        )
        .unwrap();
        assert_eq!(result.device_id, "device_1");
        assert_eq!(s.value.lock().unwrap().as_deref(), Some("native-token"));
        assert!(!serde_json::to_string(&result)
            .unwrap()
            .contains("native-token"));
    }
    #[test]
    fn mismatch_expiry_and_client_type_never_persist() {
        for mutation in 0..3 {
            let mut g = grant();
            match mutation {
                0 => g.nonce = "wrong".into(),
                1 => g.expires_unix = 50,
                _ => g.client_type = "menubar".into(),
            };
            let t = Transport {
                grant: Mutex::new(Some(g)),
            };
            let s = Store::default();
            assert!(validate_and_persist(
                &t,
                &s,
                "https://focusa.example",
                "room_1234567890",
                "nonce_1234567890",
                &["read".into()],
                CredentialHandle::parse("profile:1").unwrap(),
                100,
                &envelope()
            )
            .is_err());
            assert!(s.value.lock().unwrap().is_none());
        }
    }
}
