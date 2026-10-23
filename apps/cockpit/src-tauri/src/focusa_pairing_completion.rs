use crate::focusa_credentials::{
    CredentialError, CredentialHandle, CredentialSecret, CredentialStore,
};
use serde::{Deserialize, Serialize};
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FocusaRoomStatus {
    pub status: String,
    pub room_id: String,
    pub device_id: String,
    pub mac_nonce: String,
    pub scopes: Vec<String>,
    pub expires_at: String,
    pub expired: bool,
    pub token: Option<String>,
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct PersistedPairing {
    pub room_id: String,
    pub device_id: String,
    pub scopes: Vec<String>,
    pub expires_at: String,
    pub token_handle: CredentialHandle,
}
pub fn validate_and_persist<S: CredentialStore>(
    store: &S,
    expected_room: &str,
    expected_nonce: &str,
    expected_device: &str,
    expected_scopes: &[String],
    token_handle: CredentialHandle,
    raw_status: &[u8],
) -> Result<PersistedPairing, String> {
    if raw_status.len() > 64 * 1024 {
        return Err("room status oversized".into());
    }
    let mut status: FocusaRoomStatus =
        serde_json::from_slice(raw_status).map_err(|_| "room status invalid")?;
    if status.status != "completed"
        || status.expired
        || status.room_id != expected_room
        || status.mac_nonce != expected_nonce
        || status.device_id != expected_device
        || status.scopes != expected_scopes
    {
        return Err("room status authority mismatch".into());
    }
    let token = status
        .token
        .take()
        .filter(|v| !v.is_empty())
        .ok_or("one-shot token unavailable")?;
    let secret = CredentialSecret::new(token).map_err(|e| e.to_string())?;
    store
        .write(&token_handle, secret)
        .map_err(|e: CredentialError| e.to_string())?;
    Ok(PersistedPairing {
        room_id: status.room_id,
        device_id: status.device_id,
        scopes: status.scopes,
        expires_at: status.expires_at,
        token_handle,
    })
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::focusa_credentials::CredentialStatus;
    use std::sync::Mutex;
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
    fn status(token: Option<&str>) -> Vec<u8> {
        serde_json::to_vec(&serde_json::json!({"status":"completed","room_id":"room_1234567890abcdef","device_id":"device_1234567890abcd","mac_nonce":"nonce_1234567890abcdef","scopes":["read"],"expires_at":"2026-08-03T00:05:00Z","expired":false,"token":token})).unwrap()
    }
    #[test]
    fn consumes_one_shot_status_directly_into_native_store() {
        let s = Store::default();
        let result = validate_and_persist(
            &s,
            "room_1234567890abcdef",
            "nonce_1234567890abcdef",
            "device_1234567890abcd",
            &["read".into()],
            CredentialHandle::parse("profile:1").unwrap(),
            &status(Some("native-token")),
        )
        .unwrap();
        assert_eq!(s.value.lock().unwrap().as_deref(), Some("native-token"));
        assert!(!serde_json::to_string(&result)
            .unwrap()
            .contains("native-token"));
    }
    struct DeniedStore;
    impl CredentialStore for DeniedStore {
        fn write(&self, _: &CredentialHandle, _: CredentialSecret) -> Result<(), CredentialError> {
            Err(CredentialError::Denied)
        }
        fn read(&self, _: &CredentialHandle) -> Result<CredentialSecret, CredentialError> {
            Err(CredentialError::Denied)
        }
        fn delete(&self, _: &CredentialHandle) -> Result<(), CredentialError> {
            Err(CredentialError::Denied)
        }
        fn status(&self, _: &CredentialHandle) -> CredentialStatus {
            CredentialStatus::Denied
        }
    }

    #[test]
    fn storage_denial_returns_no_profile_or_token() {
        assert!(validate_and_persist(
            &DeniedStore,
            "room_1234567890abcdef",
            "nonce_1234567890abcdef",
            "device_1234567890abcd",
            &["read".into()],
            CredentialHandle::parse("profile:1").unwrap(),
            &status(Some("native-token"))
        )
        .is_err());
    }

    #[test]
    fn consumed_or_mismatched_status_never_persists() {
        for raw in [status(None),status(Some("")),serde_json::to_vec(&serde_json::json!({"status":"consumed","room_id":"room_1234567890abcdef","device_id":"device_1234567890abcd","mac_nonce":"nonce_1234567890abcdef","scopes":["read"],"expires_at":"x","expired":false,"token":null})).unwrap()]{let s=Store::default();assert!(validate_and_persist(&s,"room_1234567890abcdef","nonce_1234567890abcdef","device_1234567890abcd",&["read".into()],CredentialHandle::parse("profile:1").unwrap(),&raw).is_err());assert!(s.value.lock().unwrap().is_none());}
    }
}
