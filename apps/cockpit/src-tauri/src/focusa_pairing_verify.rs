use crate::focusa_credentials::{CredentialHandle, CredentialStore};
use serde::Serialize;
#[derive(Clone, Debug)]
pub struct VerifiedIdentity {
    pub daemon_id: String,
    pub device_id: String,
    pub scopes: Vec<String>,
}
pub trait VerificationTransport {
    fn authenticated_identity(
        &self,
        daemon_url: &str,
        token: &str,
    ) -> Result<VerifiedIdentity, String>;
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct VerifiedProfile {
    pub profile_id: String,
    pub daemon_id: String,
    pub daemon_url: String,
    pub device_id: String,
    pub token_handle: CredentialHandle,
    pub scopes: Vec<String>,
    pub status: &'static str,
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct PairingReceipt {
    pub schema: &'static str,
    pub outcome: &'static str,
    pub profile_id: String,
    pub daemon_id: String,
    pub device_id: String,
    pub evidence_refs: Vec<String>,
}
fn opaque(v: &str) -> bool {
    !v.is_empty()
        && v.len() <= 256
        && v.bytes()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'~' | b':' | b'-'))
}
pub fn verify_profile<T: VerificationTransport, S: CredentialStore>(
    transport: &T,
    store: &S,
    daemon_url: &str,
    expected_daemon: &str,
    expected_device: &str,
    expected_scopes: &[String],
    profile_id: &str,
    handle: CredentialHandle,
) -> Result<(VerifiedProfile, PairingReceipt), String> {
    if ![expected_daemon, expected_device, profile_id]
        .iter()
        .all(|v| opaque(v))
    {
        return Err("profile references invalid".into());
    }
    let secret = store.read(&handle).map_err(|e| e.to_string())?;
    let identity = transport.authenticated_identity(daemon_url, secret.expose_native())?;
    if identity.daemon_id != expected_daemon
        || identity.device_id != expected_device
        || identity.scopes != expected_scopes
    {
        return Err("authenticated profile identity mismatch".into());
    }
    let profile = VerifiedProfile {
        profile_id: profile_id.into(),
        daemon_id: identity.daemon_id.clone(),
        daemon_url: daemon_url.into(),
        device_id: identity.device_id.clone(),
        token_handle: handle,
        scopes: identity.scopes,
        status: "active",
    };
    let receipt = PairingReceipt {
        schema: "focusa.pairing_receipt.v1",
        outcome: "verified",
        profile_id: profile.profile_id.clone(),
        daemon_id: profile.daemon_id.clone(),
        device_id: profile.device_id.clone(),
        evidence_refs: vec![
            format!("daemon:{}", profile.daemon_id),
            format!("device:{}", profile.device_id),
        ],
    };
    Ok((profile, receipt))
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::focusa_credentials::{CredentialError, CredentialSecret, CredentialStatus};
    struct Store;
    impl CredentialStore for Store {
        fn write(&self, _: &CredentialHandle, _: CredentialSecret) -> Result<(), CredentialError> {
            Ok(())
        }
        fn read(&self, _: &CredentialHandle) -> Result<CredentialSecret, CredentialError> {
            CredentialSecret::new("native-token".into())
        }
        fn delete(&self, _: &CredentialHandle) -> Result<(), CredentialError> {
            Ok(())
        }
        fn status(&self, _: &CredentialHandle) -> CredentialStatus {
            CredentialStatus::Available
        }
    }
    struct Transport {
        mismatch: bool,
    }
    impl VerificationTransport for Transport {
        fn authenticated_identity(&self, _: &str, token: &str) -> Result<VerifiedIdentity, String> {
            assert_eq!(token, "native-token");
            Ok(VerifiedIdentity {
                daemon_id: if self.mismatch { "other" } else { "daemon_1" }.into(),
                device_id: "device_1".into(),
                scopes: vec!["read".into()],
            })
        }
    }
    #[test]
    fn activates_only_after_authenticated_identity_proof() {
        let (profile, receipt) = verify_profile(
            &Transport { mismatch: false },
            &Store,
            "https://focusa.example",
            "daemon_1",
            "device_1",
            &["read".into()],
            "profile_1",
            CredentialHandle::parse("profile:1").unwrap(),
        )
        .unwrap();
        assert_eq!(profile.status, "active");
        assert_eq!(receipt.outcome, "verified");
        let serialized = serde_json::to_string(&(profile, receipt)).unwrap();
        assert!(!serialized.contains("native-token"));
    }
    #[test]
    fn identity_mismatch_never_activates() {
        assert!(verify_profile(
            &Transport { mismatch: true },
            &Store,
            "https://focusa.example",
            "daemon_1",
            "device_1",
            &["read".into()],
            "profile_1",
            CredentialHandle::parse("profile:1").unwrap()
        )
        .is_err());
    }
}
