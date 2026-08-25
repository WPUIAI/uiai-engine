//! Versioned desktop-presentation and app-handoff contracts shared with the
//! native Go Engine and the Focusa Menubar. This module carries data and
//! validation only; it owns no browser, window, daemon, or persistence state.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

pub const SCHEMA_BROWSER_RUNTIME_MANIFEST_V1: &str = "uiai.browser_runtime_manifest.v1";
pub const SCHEMA_DESKTOP_PRESENTATION_REQUEST_V1: &str = "uiai.desktop_presentation_request.v1";
pub const SCHEMA_DESKTOP_PRESENTATION_RECEIPT_V1: &str = "uiai.desktop_presentation_receipt.v1";
pub const SCHEMA_DESKTOP_PRESENTATION_STATUS_V1: &str = "uiai.desktop_presentation_status.v1";
pub const SCHEMA_APP_HANDOFF_INTENT_V1: &str = "uiai.app_handoff_intent.v1";
pub const SCHEMA_APP_HANDOFF_RECEIPT_V1: &str = "uiai.app_handoff_receipt.v1";
pub const SCHEMA_FOCUSA_APP_MANIFEST_V2: &str = "focusa.app.manifest.v2";

pub const SCHEMA_IDS: [&str; 7] = [
    SCHEMA_BROWSER_RUNTIME_MANIFEST_V1,
    SCHEMA_DESKTOP_PRESENTATION_REQUEST_V1,
    SCHEMA_DESKTOP_PRESENTATION_RECEIPT_V1,
    SCHEMA_DESKTOP_PRESENTATION_STATUS_V1,
    SCHEMA_APP_HANDOFF_INTENT_V1,
    SCHEMA_APP_HANDOFF_RECEIPT_V1,
    SCHEMA_FOCUSA_APP_MANIFEST_V2,
];

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ScopeRef {
    pub project_root_key: Option<String>,
    pub workstream_key: Option<String>,
    pub continuity_id: Option<String>,
    pub thread_id: Option<String>,
    pub session_id: Option<String>,
    pub authority_state: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ClientRef {
    pub client_type: String,
    pub client_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct BrowserRuntimeManifest {
    pub schema: String,
    pub runtime_id: String,
    pub engine: String,
    pub version: String,
    pub cdp_protocol: String,
    pub platform: String,
    pub arch: String,
    pub executable_relpath: String,
    pub sha256: String,
    pub signed: bool,
    pub source: String,
    pub built_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DesktopPresentationRequest {
    pub schema: String,
    pub mode: String,
    pub reason: String,
    pub scope_ref: Option<ScopeRef>,
    pub requested_by: ClientRef,
    pub focus: bool,
    pub expires_in_ms: i64,
    pub idempotency_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DesktopPresentationReceipt {
    pub schema: String,
    pub presentation_id: String,
    pub session_id: String,
    pub status: String,
    pub cockpit_instance_id: Option<String>,
    pub handoff_ref: Option<String>,
    pub reason_code: Option<String>,
    pub created_at: String,
    pub expires_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DesktopPresentationStatus {
    pub schema: String,
    pub presentation_id: String,
    pub session_id: String,
    pub status: String,
    pub reason_code: Option<String>,
    pub observed_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct AppHandoffIntent {
    pub schema: String,
    pub intent_id: String,
    pub scheme: String,
    pub route: String,
    pub target_ref: String,
    pub requested_by: ClientRef,
    pub protocol_version: String,
    pub created_at: String,
    pub expires_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct AppHandoffReceipt {
    pub schema: String,
    pub intent_id: String,
    pub status: String,
    pub target_app: String,
    pub resolved_ref: Option<String>,
    pub reason_code: Option<String>,
    pub observed_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct AppManifest {
    pub schema: String,
    pub app: String,
    pub version: String,
    pub channel: String,
    pub protocols: HashMap<String, String>,
    pub capabilities: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FixtureBundle {
    pub runtime_manifest: BrowserRuntimeManifest,
    pub presentation_request: DesktopPresentationRequest,
    pub presentation_receipt: DesktopPresentationReceipt,
    pub presentation_status: DesktopPresentationStatus,
    pub handoff_intent: AppHandoffIntent,
    pub handoff_receipt: AppHandoffReceipt,
    pub app_manifest: AppManifest,
}

pub fn validate_opaque_ref(value: &str) -> Result<(), String> {
    if value.is_empty() || value.len() > 160 {
        return Err("opaque ref length must be 1-160".into());
    }
    if !value
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '.' | '_' | ':' | '-'))
    {
        return Err(
            "opaque ref contains URL, path, query, fragment, authority, whitespace, or unsafe data"
                .into(),
        );
    }
    if !value
        .chars()
        .next()
        .is_some_and(|ch| ch.is_ascii_alphanumeric())
    {
        return Err("opaque ref must start with an alphanumeric character".into());
    }
    Ok(())
}

fn validate_timestamp(value: &str, name: &str) -> Result<(), String> {
    if value.len() < 20 || !value.ends_with('Z') || !value.contains('T') {
        return Err(format!("{name} must be RFC3339 UTC"));
    }
    Ok(())
}

fn validate_client(value: &ClientRef) -> Result<(), String> {
    if !matches!(
        value.client_type.as_str(),
        "pi" | "cockpit" | "menubar" | "api" | "mcp" | "cli"
    ) {
        return Err(format!("unsupported client_type {}", value.client_type));
    }
    validate_opaque_ref(&value.client_id)
}

fn validate_status(value: &str) -> Result<(), String> {
    if matches!(
        value,
        "requested"
            | "resolving_session"
            | "resolving_cockpit"
            | "launching"
            | "attaching"
            | "visible"
            | "focused"
            | "already_visible"
            | "blocked"
            | "unavailable"
            | "failed"
            | "blocked_scope"
            | "session_missing"
            | "cockpit_missing"
            | "incompatible"
            | "attach_failed"
            | "desktop_unavailable"
            | "expired"
            | "cancelled"
    ) {
        Ok(())
    } else {
        Err(format!("unsupported presentation status {value}"))
    }
}

pub fn validate_runtime_manifest(value: &BrowserRuntimeManifest) -> Result<(), String> {
    if value.schema != SCHEMA_BROWSER_RUNTIME_MANIFEST_V1 {
        return Err("runtime schema mismatch".into());
    }
    validate_opaque_ref(&value.runtime_id)?;
    if value.engine != "chromium"
        || value.source != "uiai-release"
        || value.version.is_empty()
        || value.cdp_protocol.is_empty()
        || value.platform.is_empty()
        || value.arch.is_empty()
    {
        return Err("runtime identity fields are incomplete".into());
    }
    if value.executable_relpath.is_empty()
        || value.executable_relpath.starts_with('/')
        || value.executable_relpath.contains("..")
    {
        return Err("runtime executable path must remain package-relative".into());
    }
    if value.sha256.len() != 64
        || !value
            .sha256
            .chars()
            .all(|ch| ch.is_ascii_digit() || ('a'..='f').contains(&ch))
    {
        return Err("runtime sha256 must be lowercase hexadecimal".into());
    }
    validate_timestamp(&value.built_at, "built_at")
}

pub fn validate_presentation_request(value: &DesktopPresentationRequest) -> Result<(), String> {
    if value.schema != SCHEMA_DESKTOP_PRESENTATION_REQUEST_V1 {
        return Err("presentation request schema mismatch".into());
    }
    if !matches!(value.mode.as_str(), "full" | "pip" | "focus_existing") {
        return Err("unsupported presentation mode".into());
    }
    if !matches!(
        value.reason.as_str(),
        "operator_request"
            | "takeover_required"
            | "policy_confirmation"
            | "failure_recovery"
            | "workflow"
    ) {
        return Err("unsupported presentation reason".into());
    }
    validate_client(&value.requested_by)?;
    validate_opaque_ref(&value.idempotency_key)?;
    if !(1000..=300000).contains(&value.expires_in_ms) {
        return Err("presentation expiry is out of range".into());
    }
    if let Some(scope) = &value.scope_ref {
        if !matches!(
            scope.authority_state.as_str(),
            "verified" | "missing" | "stale" | "conflict" | "read_only"
        ) {
            return Err("unsupported scope authority_state".into());
        }
        for current in [
            &scope.project_root_key,
            &scope.workstream_key,
            &scope.continuity_id,
            &scope.thread_id,
            &scope.session_id,
        ]
        .into_iter()
        .flatten()
        {
            validate_opaque_ref(current)?;
        }
    }
    Ok(())
}

pub fn validate_presentation_receipt(value: &DesktopPresentationReceipt) -> Result<(), String> {
    if value.schema != SCHEMA_DESKTOP_PRESENTATION_RECEIPT_V1 {
        return Err("presentation receipt schema mismatch".into());
    }
    validate_opaque_ref(&value.presentation_id)?;
    validate_opaque_ref(&value.session_id)?;
    if let Some(current) = &value.cockpit_instance_id {
        validate_opaque_ref(current)?;
    }
    if let Some(current) = &value.handoff_ref {
        validate_opaque_ref(current)?;
    }
    validate_status(&value.status)?;
    validate_timestamp(&value.created_at, "created_at")?;
    validate_timestamp(&value.expires_at, "expires_at")
}

pub fn validate_presentation_status(value: &DesktopPresentationStatus) -> Result<(), String> {
    if value.schema != SCHEMA_DESKTOP_PRESENTATION_STATUS_V1 {
        return Err("presentation status schema mismatch".into());
    }
    validate_opaque_ref(&value.presentation_id)?;
    validate_opaque_ref(&value.session_id)?;
    validate_status(&value.status)?;
    validate_timestamp(&value.observed_at, "observed_at")
}

pub fn validate_handoff_intent(value: &AppHandoffIntent) -> Result<(), String> {
    if value.schema != SCHEMA_APP_HANDOFF_INTENT_V1 {
        return Err("handoff intent schema mismatch".into());
    }
    validate_opaque_ref(&value.intent_id)?;
    let route_ok = match value.scheme.as_str() {
        "focusa" => matches!(
            value.route.as_str(),
            "mission" | "card" | "workpoint" | "connect"
        ),
        "cockpit" => matches!(
            value.route.as_str(),
            "live/session" | "focusa" | "evidence" | "settings/pairing"
        ),
        _ => false,
    };
    if !route_ok {
        return Err("unsupported handoff scheme or route".into());
    }
    validate_opaque_ref(&value.target_ref)?;
    validate_client(&value.requested_by)?;
    if value.protocol_version != "1" {
        return Err("unsupported handoff protocol version".into());
    }
    validate_timestamp(&value.created_at, "created_at")?;
    validate_timestamp(&value.expires_at, "expires_at")
}

pub fn validate_handoff_receipt(value: &AppHandoffReceipt) -> Result<(), String> {
    if value.schema != SCHEMA_APP_HANDOFF_RECEIPT_V1 {
        return Err("handoff receipt schema mismatch".into());
    }
    validate_opaque_ref(&value.intent_id)?;
    if let Some(current) = &value.resolved_ref {
        validate_opaque_ref(current)?;
    }
    if !matches!(
        value.status.as_str(),
        "opened" | "focused" | "blocked" | "unavailable" | "failed"
    ) {
        return Err("unsupported handoff receipt status".into());
    }
    if !matches!(
        value.target_app.as_str(),
        "focusa-menubar" | "uaiengine-cockpit"
    ) {
        return Err("unsupported handoff target app".into());
    }
    validate_timestamp(&value.observed_at, "observed_at")
}

pub fn validate_app_manifest(value: &AppManifest) -> Result<(), String> {
    if value.schema != SCHEMA_FOCUSA_APP_MANIFEST_V2 {
        return Err("app manifest schema mismatch".into());
    }
    if !matches!(value.app.as_str(), "focusa-menubar" | "uaiengine-cockpit")
        || value.version.is_empty()
        || !matches!(value.channel.as_str(), "stable" | "preview" | "dev")
    {
        return Err("unsupported app manifest identity".into());
    }
    for protocol in [
        "focusa_deep_link",
        "cockpit_deep_link",
        "desktop_presentation",
        "fpv",
    ] {
        if value.protocols.get(protocol).is_none_or(String::is_empty) {
            return Err(format!("missing protocol {protocol}"));
        }
    }
    if value.capabilities.is_empty() || value.capabilities.iter().any(String::is_empty) {
        return Err("app capabilities are required".into());
    }
    Ok(())
}

pub fn validate_fixture_bundle(value: &FixtureBundle) -> Result<(), String> {
    validate_runtime_manifest(&value.runtime_manifest)?;
    validate_presentation_request(&value.presentation_request)?;
    validate_presentation_receipt(&value.presentation_receipt)?;
    validate_presentation_status(&value.presentation_status)?;
    validate_handoff_intent(&value.handoff_intent)?;
    validate_handoff_receipt(&value.handoff_receipt)?;
    validate_app_manifest(&value.app_manifest)
}

#[cfg(test)]
mod tests {
    use super::*;

    const VALID: &str =
        include_str!("../../../../tests/fixtures/desktop-presentation/valid-contracts.json");
    const MENUBAR_MANIFEST: &str = include_str!(
        "../../../../tests/fixtures/desktop-presentation/focusa-app-manifest.valid.json"
    );
    const SECRET: &str =
        include_str!("../../../../tests/fixtures/desktop-presentation/handoff-secret.invalid.json");
    const RAW_PATH: &str = include_str!(
        "../../../../tests/fixtures/desktop-presentation/handoff-raw-path.invalid.json"
    );
    const PRIVATE_URL: &str = include_str!(
        "../../../../tests/fixtures/desktop-presentation/handoff-private-url.invalid.json"
    );
    const UNKNOWN_ROUTE: &str = include_str!(
        "../../../../tests/fixtures/desktop-presentation/handoff-unknown-route.invalid.json"
    );

    #[test]
    fn shared_fixture_bundle_validates() {
        let bundle: FixtureBundle = serde_json::from_str(VALID).expect("fixture parses");
        validate_fixture_bundle(&bundle).expect("fixture validates");
        assert_eq!(SCHEMA_IDS.len(), 7);

        let menubar: AppManifest =
            serde_json::from_str(MENUBAR_MANIFEST).expect("Menubar manifest parses");
        validate_app_manifest(&menubar).expect("Menubar manifest validates");
        assert_eq!(menubar.app, "focusa-menubar");
    }

    #[test]
    fn invalid_handoff_fixtures_fail_closed() {
        assert!(serde_json::from_str::<AppHandoffIntent>(SECRET).is_err());
        for source in [RAW_PATH, PRIVATE_URL, UNKNOWN_ROUTE] {
            let intent: AppHandoffIntent =
                serde_json::from_str(source).expect("fixture shape parses");
            assert!(validate_handoff_intent(&intent).is_err());
        }
    }

    #[test]
    fn opaque_refs_reject_urls_paths_queries_and_fragments() {
        for value in [
            "/tmp/private",
            "https://example.com",
            "session?token=secret",
            "session#fragment",
            r"C:\Users\operator",
            "contains space",
        ] {
            assert!(validate_opaque_ref(value).is_err(), "{value} should fail");
        }
    }
}
