use crate::desktop_contract::{validate_handoff_intent, AppHandoffIntent, AppHandoffReceipt};
use std::process::Command;
use time::{format_description::well_known::Rfc3339, OffsetDateTime};
use url::Url;

fn receipt(intent: &AppHandoffIntent, status: &str, reason: Option<&str>) -> AppHandoffReceipt {
    AppHandoffReceipt {
        schema: "uiai.app_handoff_receipt.v1".into(),
        intent_id: intent.intent_id.clone(),
        status: status.into(),
        target_app: "focusa-menubar".into(),
        resolved_ref: (status == "opened").then(|| intent.target_ref.clone()),
        reason_code: reason.map(str::to_string),
        observed_at: OffsetDateTime::now_utc()
            .format(&Rfc3339)
            .unwrap_or_default(),
    }
}

fn focusa_url(intent: &AppHandoffIntent) -> Result<Url, String> {
    validate_handoff_intent(intent)?;
    if intent.scheme != "focusa" {
        return Err("cockpit can launch only focusa handoffs".into());
    }
    let expires = OffsetDateTime::parse(&intent.expires_at, &Rfc3339)
        .map_err(|_| "invalid handoff expiry".to_string())?;
    if expires <= OffsetDateTime::now_utc() {
        return Err("handoff intent expired".into());
    }
    if intent.route == "connect" {
        let mut url = Url::parse("focusa://connect").map_err(|_| "invalid focusa route")?;
        url.query_pairs_mut()
            .append_pair("payload", &intent.target_ref);
        Ok(url)
    } else {
        Url::parse(&format!("focusa://{}/{}", intent.route, intent.target_ref))
            .map_err(|_| "invalid focusa route".into())
    }
}

fn launch(url: &Url) -> std::io::Result<bool> {
    #[cfg(target_os = "macos")]
    let status = Command::new("open").arg(url.as_str()).status()?;
    #[cfg(target_os = "windows")]
    let status = Command::new("cmd")
        .args(["/C", "start", "", url.as_str()])
        .status()?;
    #[cfg(all(unix, not(target_os = "macos")))]
    let status = Command::new("xdg-open").arg(url.as_str()).status()?;
    Ok(status.success())
}

#[tauri::command]
pub fn cockpit_open_focusa_handoff(intent: AppHandoffIntent) -> AppHandoffReceipt {
    let url = match focusa_url(&intent) {
        Ok(url) => url,
        Err(reason) => return receipt(&intent, "blocked", Some(&reason)),
    };
    match launch(&url) {
        Ok(true) => receipt(&intent, "opened", None),
        Ok(false) => receipt(&intent, "unavailable", Some("focusa_handler_unavailable")),
        Err(_) => receipt(&intent, "unavailable", Some("focusa_launch_failed")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::desktop_contract::ClientRef;

    fn intent(route: &str, target: &str, expires_at: &str) -> AppHandoffIntent {
        AppHandoffIntent {
            schema: "uiai.app_handoff_intent.v1".into(),
            intent_id: "handoff_01".into(),
            scheme: "focusa".into(),
            route: route.into(),
            target_ref: target.into(),
            requested_by: ClientRef {
                client_type: "cockpit".into(),
                client_id: "uaiengine-cockpit:0.3.0:dev".into(),
            },
            protocol_version: "1".into(),
            created_at: "2026-08-03T00:00:00Z".into(),
            expires_at: expires_at.into(),
        }
    }

    #[test]
    fn builds_only_allowlisted_opaque_focusa_routes() {
        let value = intent("workpoint", "wp_01", "2099-08-03T00:00:00Z");
        assert_eq!(
            focusa_url(&value).unwrap().as_str(),
            "focusa://workpoint/wp_01"
        );
        let connect = intent("connect", "pair_payload", "2099-08-03T00:00:00Z");
        assert_eq!(
            focusa_url(&connect).unwrap().as_str(),
            "focusa://connect?payload=pair_payload"
        );
    }

    #[test]
    fn blocks_expired_or_cross_scheme_handoffs() {
        assert!(focusa_url(&intent("mission", "mission_01", "2020-01-01T00:00:00Z")).is_err());
        let mut wrong = intent("workpoint", "wp_01", "2099-08-03T00:00:00Z");
        wrong.scheme = "cockpit".into();
        wrong.route = "focusa".into();
        assert!(focusa_url(&wrong).is_err());
    }
}
