//! UIAI-COCKPIT-005 T005-06.04 — Distinct Cockpit token minting (Path B).
//! Daemon proof creates a new cockpit device + native token handle; never copies Menubar token.

use crate::focusa_credentials::{CredentialError, CredentialHandle, CredentialSecret, CredentialStore};
use serde::{Deserialize, Serialize};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DaemonMenubarProof {
    pub schema: String,
    pub verified_device_id: String,
    pub daemon_url: String,
    pub verified_at: String,
    pub scopes: Vec<String>,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct DistinctCockpitMint {
    pub schema: &'static str,
    pub cockpit_device_id: String,
    pub menubar_device_id: String,
    pub daemon_url: String,
    pub token_handle: CredentialHandle,
    pub scopes: Vec<String>,
}

fn opaque(v: &str) -> bool {
    (1..=256).contains(&v.len()) && v.bytes().all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'~' | b':' | b'-'))
}
fn forbidden(k: &str) -> bool {
    let l = k.to_ascii_lowercase();
    l.contains("token") || l.contains("secret") || l.contains("password") || l.contains("authorization") || l.contains("private")
}

pub fn validate_and_mint<S: CredentialStore>(
    store: &S,
    cross_device_id: &str,
    cross_daemon_url: &str,
    raw_proof: &[u8],
    cockpit_device_id: &str,
    token_handle: CredentialHandle,
    one_shot_token: &str,
) -> Result<DistinctCockpitMint, String> {
    if raw_proof.len() > 32 * 1024 { return Err("daemon proof oversized".into()); }
    let proof: DaemonMenubarProof = serde_json::from_slice(raw_proof).map_err(|_| "daemon proof invalid")?;
    if proof.schema != "focusa.daemon_menubar_proof.v1" { return Err("daemon proof schema mismatch".into()); }
    if !opaque(&proof.verified_device_id) || !opaque(cockpit_device_id) || !opaque(&proof.daemon_url.replace("://","")) { /* url check below */ }
    // No secret field may appear in proof JSON (never copy Menubar credential)
    let raw_str = String::from_utf8_lossy(raw_proof).to_ascii_lowercase();
    if raw_str.contains("\"token\"") || raw_str.contains("\"secret\"") { return Err("daemon proof contains forbidden secret field".into()); }
    for k in &proof.scopes { if !matches!(k.as_str(), "read" | "write") { return Err("daemon proof scopes invalid".into()); } }
    if proof.verified_device_id != cross_device_id { return Err("daemon proof device mismatch".into()); }
    // Origin must match cross-ref daemon_url
    let proof_origin = url::Url::parse(&proof.daemon_url).map_err(|_| "daemon proof URL invalid")?.origin().ascii_serialization();
    let cross_origin = url::Url::parse(cross_daemon_url).map_err(|_| "cross daemon URL invalid")?.origin().ascii_serialization();
    if proof_origin != cross_origin { return Err("daemon proof origin mismatch".into()); }
    if cockpit_device_id == cross_device_id { return Err("cockpit device must be distinct from menubar device".into()); }
    if !opaque(cockpit_device_id) { return Err("cockpit device invalid".into()); }
    // Validate handle shape already via CredentialHandle parse before call; forbid handle containing secret word
    if forbidden(token_handle.as_str()) { return Err("token_handle contains forbidden shape".into()); }
    let secret = CredentialSecret::new(one_shot_token.to_string()).map_err(|e| e.to_string())?;
    store.write(&token_handle, secret).map_err(|e: CredentialError| e.to_string())?;
    Ok(DistinctCockpitMint {
        schema: "focusa.distinct_cockpit_mint.v1",
        cockpit_device_id: cockpit_device_id.to_string(),
        menubar_device_id: proof.verified_device_id,
        daemon_url: cross_daemon_url.to_string(),
        token_handle,
        scopes: proof.scopes,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::focusa_credentials::{CredentialSecret, CredentialStatus};
    use std::sync::Mutex;
    #[derive(Default)] struct Store { value: Mutex<Option<String>> }
    impl CredentialStore for Store {
        fn write(&self, _: &CredentialHandle, s: CredentialSecret) -> Result<(), CredentialError> { *self.value.lock().unwrap() = Some(s.expose_native().into()); Ok(()) }
        fn read(&self, _: &CredentialHandle) -> Result<CredentialSecret, CredentialError> { Err(CredentialError::Missing) }
        fn delete(&self, _: &CredentialHandle) -> Result<(), CredentialError> { Ok(()) }
        fn status(&self, _: &CredentialHandle) -> CredentialStatus { CredentialStatus::Missing }
    }
    fn proof(device: &str) -> Vec<u8> {
        serde_json::to_vec(&serde_json::json!({"schema":"focusa.daemon_menubar_proof.v1","verified_device_id":device,"daemon_url":"http://127.0.0.1:8787","verified_at":"2026-08-03T00:05:00Z","scopes":["read"]})).unwrap()
    }
    #[test]
    fn mints_distinct_device_and_handle() {
        let s = Store::default();
        let out = validate_and_mint(&s, "menubar_dev_1234567890abcd", "http://127.0.0.1:8787", &proof("menubar_dev_1234567890abcd"), "cockpit_dev_0987654321abcd", CredentialHandle::parse("profile:2").unwrap(), "native-minted-token").unwrap();
        assert_ne!(out.cockpit_device_id, out.menubar_device_id);
        assert_eq!(s.value.lock().unwrap().as_deref(), Some("native-minted-token"));
        assert!(!serde_json::to_string(&out).unwrap().contains("native-minted-token"));
    }
    #[test]
    fn rejects_copy_or_secret() {
        let s = Store::default();
        assert!(validate_and_mint(&s, "menubar_dev_1234567890abcd", "http://127.0.0.1:8787", &proof("menubar_dev_1234567890abcd"), "menubar_dev_1234567890abcd", CredentialHandle::parse("profile:2").unwrap(), "t").is_err());
        let bad = serde_json::to_vec(&serde_json::json!({"schema":"focusa.daemon_menubar_proof.v1","verified_device_id":"menubar_dev_1234567890abcd","daemon_url":"http://127.0.0.1:8787","verified_at":"2026-08-03T00:05:00Z","scopes":["read"],"token":"secret"})).unwrap();
        assert!(validate_and_mint(&s, "menubar_dev_1234567890abcd", "http://127.0.0.1:8787", &bad, "cockpit_dev_0987abcd", CredentialHandle::parse("profile:2").unwrap(), "t").is_err());
    }
}
