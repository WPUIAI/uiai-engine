use serde::{Deserialize, Serialize};
use time::{format_description::well_known::Rfc3339, OffsetDateTime};
use url::Url;

#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct FocusaDaemonCandidate {
    pub schema: String,
    pub candidate_id: String,
    pub base_url: String,
    pub source: String,
    pub location: String,
    pub observed_at: String,
    pub health_status: String,
    pub latency_ms: u64,
    pub daemon_id: Option<String>,
    pub machine_id: Option<String>,
    pub version: Option<String>,
    pub capabilities: Option<Vec<String>>,
}
#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct FocusaPlatformHint {
    pub schema: String,
    pub daemon_url: String,
    pub daemon_id: Option<String>,
    pub device_id: Option<String>,
    pub paired: bool,
    pub client_types_seen: Vec<String>,
    pub last_verified_at: String,
}
#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct CockpitPairingRoom {
    pub schema: String,
    pub room_id: String,
    pub nonce: String,
    pub client_type: String,
    pub daemon_url: String,
    pub status: String,
    pub created_at: String,
    pub expires_at: String,
    pub bridge_owner: String,
    pub pair_url: Option<String>,
    pub pair_code: Option<String>,
    pub menubar_device_id: Option<String>,
}
#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct AuthenticatedDaemonProfile {
    pub schema: String,
    pub profile_id: String,
    pub daemon_id: String,
    pub daemon_url: String,
    pub device_id: String,
    pub client_type: String,
    pub token_handle: String,
    pub scopes: Vec<String>,
    pub source: String,
    pub paired_at: String,
    pub expires_at: String,
    pub last_verified_at: String,
    pub status: String,
}
#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct DaemonProjectCandidate {
    pub schema: String,
    pub daemon_id: String,
    pub profile_id: String,
    pub project_root: String,
    pub project_id: String,
    pub canonical_name: String,
    pub identity_status: String,
    pub observed_at: String,
    pub repo_remote: Option<String>,
    pub continuities: Option<Vec<String>>,
}
#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct ProjectScopeBinding {
    pub schema: String,
    pub binding_id: String,
    pub daemon_id: String,
    pub profile_id: String,
    pub project_id: String,
    pub project_root: String,
    pub continuity_id: String,
    pub authority_status: String,
    pub selected_at: String,
    pub workpoint_id: Option<String>,
}
#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct ScopeReconciliationDecision {
    pub schema: String,
    pub decision_id: String,
    pub left_binding_id: String,
    pub right_binding_id: String,
    pub relation: String,
    pub operator_confirmed: bool,
    pub created_at: String,
    pub evidence_refs: Vec<String>,
}
#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct PairingFixtureBundle {
    pub candidate: FocusaDaemonCandidate,
    pub hint: FocusaPlatformHint,
    pub room: CockpitPairingRoom,
    pub profile: AuthenticatedDaemonProfile,
    pub project: DaemonProjectCandidate,
    pub binding: ProjectScopeBinding,
    pub reconciliation: ScopeReconciliationDecision,
}

fn opaque(value: &str) -> bool {
    !value.is_empty()
        && value.len() <= 256
        && value
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'~' | b':' | b'-'))
}
fn timestamp(value: &str) -> bool {
    OffsetDateTime::parse(value, &Rfc3339).is_ok()
}
fn endpoint(value: &str) -> bool {
    Url::parse(value).is_ok_and(|u| {
        matches!(u.scheme(), "http" | "https")
            && u.username().is_empty()
            && u.password().is_none()
            && u.fragment().is_none()
    })
}

impl PairingFixtureBundle {
    pub fn validate(&self) -> Result<(), String> {
        if self.candidate.schema != "focusa.daemon_candidate.v1"
            || !opaque(&self.candidate.candidate_id)
            || !endpoint(&self.candidate.base_url)
            || !matches!(
                self.candidate.source.as_str(),
                "loopback" | "bonjour" | "tailscale" | "saved_hint" | "environment" | "manual"
            )
        {
            return Err("invalid candidate".into());
        }
        if self.hint.schema != "focusa.platform_hint.v1"
            || !endpoint(&self.hint.daemon_url)
            || !timestamp(&self.hint.last_verified_at)
        {
            return Err("invalid hint".into());
        }
        if self.room.schema != "focusa.cockpit_pairing_room.v1"
            || self.room.client_type != "cockpit"
            || !opaque(&self.room.room_id)
            || !opaque(&self.room.nonce)
            || !timestamp(&self.room.expires_at)
        {
            return Err("invalid room".into());
        }
        if self.profile.schema != "focusa.authenticated_daemon_profile.v1"
            || self.profile.client_type != "cockpit"
            || !opaque(&self.profile.profile_id)
            || !opaque(&self.profile.token_handle)
            || !endpoint(&self.profile.daemon_url)
        {
            return Err("invalid profile".into());
        }
        if self.project.schema != "focusa.daemon_project_candidate.v1"
            || !opaque(&self.project.daemon_id)
            || !opaque(&self.project.project_id)
        {
            return Err("invalid project".into());
        }
        if self.binding.schema != "focusa.project_scope_binding.v1"
            || !opaque(&self.binding.binding_id)
            || !opaque(&self.binding.continuity_id)
        {
            return Err("invalid binding".into());
        }
        if self.reconciliation.schema != "focusa.scope_reconciliation_decision.v1"
            || !self.reconciliation.operator_confirmed
            || self.reconciliation.left_binding_id == self.reconciliation.right_binding_id
            || !matches!(
                self.reconciliation.relation.as_str(),
                "same_project_separate_authority"
                    | "preferred_profile"
                    | "mirror_read_only"
                    | "not_same_project"
            )
        {
            return Err("invalid reconciliation".into());
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn shared_pairing_fixture_validates() {
        let value: PairingFixtureBundle = serde_json::from_str(include_str!(
            "../../tests/fixtures/focusa-pairing-valid.json"
        ))
        .unwrap();
        value.validate().unwrap();
    }
    #[test]
    fn unknown_and_unconfirmed_inputs_fail_closed() {
        let raw = include_str!("../../tests/fixtures/focusa-pairing-valid.json");
        let mut value: serde_json::Value = serde_json::from_str(raw).unwrap();
        value["hint"]["token"] = serde_json::json!("secret");
        assert!(serde_json::from_value::<PairingFixtureBundle>(value).is_err());
        let mut bundle: PairingFixtureBundle = serde_json::from_str(raw).unwrap();
        bundle.reconciliation.operator_confirmed = false;
        assert!(bundle.validate().is_err());
    }
}
