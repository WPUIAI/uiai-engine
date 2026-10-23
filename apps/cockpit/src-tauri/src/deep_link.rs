use serde::Serialize;
use std::sync::Mutex;
use tauri::{App, AppHandle, Emitter, Manager, State};
use tauri_plugin_deep_link::DeepLinkExt;
use url::Url;

const DEEP_LINK_EVENT: &str = "cockpit-deep-link";
const DEEP_LINK_REJECTED_EVENT: &str = "cockpit-deep-link-rejected";
const MAX_OPAQUE_REF_LEN: usize = 256;

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum CockpitDeepLinkRoute {
    LiveSession,
    Focusa,
    Evidence,
    SettingsPairing,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
pub struct CockpitDeepLinkIntent {
    pub schema: &'static str,
    pub route: CockpitDeepLinkRoute,
    pub target_ref: Option<String>,
    pub handoff_ref: Option<String>,
}

#[derive(Default)]
pub struct PendingDeepLink(Mutex<Option<CockpitDeepLinkIntent>>);

fn opaque_ref(value: &str, field: &str) -> Result<String, String> {
    if value.is_empty() || value.len() > MAX_OPAQUE_REF_LEN {
        return Err(format!(
            "{field} must be 1..={MAX_OPAQUE_REF_LEN} characters"
        ));
    }
    if !value.bytes().all(|byte| {
        byte.is_ascii_alphanumeric() || matches!(byte, b'.' | b'_' | b'~' | b':' | b'-')
    }) {
        return Err(format!("{field} contains unsupported characters"));
    }
    Ok(value.to_string())
}

fn path_segments(url: &Url) -> Vec<&str> {
    url.path_segments()
        .map(|segments| segments.filter(|segment| !segment.is_empty()).collect())
        .unwrap_or_default()
}

pub fn parse_cockpit_deep_link(raw: &str) -> Result<CockpitDeepLinkIntent, String> {
    let url = Url::parse(raw).map_err(|_| "malformed cockpit deep link".to_string())?;
    if url.scheme() != "cockpit" {
        return Err("unsupported deep-link scheme".into());
    }
    if !url.username().is_empty()
        || url.password().is_some()
        || url.port().is_some()
        || url.fragment().is_some()
    {
        return Err("credentials, ports, and fragments are not allowed".into());
    }
    let host = url
        .host_str()
        .ok_or_else(|| "deep-link route is missing".to_string())?;
    let segments = path_segments(&url);
    let query: Vec<(String, String)> = url
        .query_pairs()
        .map(|(key, value)| (key.into_owned(), value.into_owned()))
        .collect();

    match (host, segments.as_slice()) {
        ("live", ["session", session_id]) => {
            let session_id = opaque_ref(session_id, "session_id")?;
            if query.len() > 1 || query.first().is_some_and(|(key, _)| key != "handoff") {
                return Err("live session accepts only one handoff query field".into());
            }
            let handoff_ref = query
                .first()
                .map(|(_, value)| opaque_ref(value, "handoff_ref"))
                .transpose()?;
            Ok(CockpitDeepLinkIntent {
                schema: "uiai.cockpit.deep_link.v1",
                route: CockpitDeepLinkRoute::LiveSession,
                target_ref: Some(session_id),
                handoff_ref,
            })
        }
        ("focusa", [target_ref]) if query.is_empty() => Ok(CockpitDeepLinkIntent {
            schema: "uiai.cockpit.deep_link.v1",
            route: CockpitDeepLinkRoute::Focusa,
            target_ref: Some(opaque_ref(target_ref, "focusa_ref")?),
            handoff_ref: None,
        }),
        ("evidence", [target_ref]) if query.is_empty() => Ok(CockpitDeepLinkIntent {
            schema: "uiai.cockpit.deep_link.v1",
            route: CockpitDeepLinkRoute::Evidence,
            target_ref: Some(opaque_ref(target_ref, "evidence_ref")?),
            handoff_ref: None,
        }),
        ("settings", ["pairing"]) if query.is_empty() => Ok(CockpitDeepLinkIntent {
            schema: "uiai.cockpit.deep_link.v1",
            route: CockpitDeepLinkRoute::SettingsPairing,
            target_ref: None,
            handoff_ref: None,
        }),
        _ => Err("unknown or non-canonical cockpit deep-link route".into()),
    }
}

pub fn focus_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
    }
}

fn accept_url(app: &AppHandle, raw: &str) {
    match parse_cockpit_deep_link(raw) {
        Ok(intent) => {
            if let Ok(mut pending) = app.state::<PendingDeepLink>().0.lock() {
                *pending = Some(intent.clone());
            }
            let _ = app.emit(DEEP_LINK_EVENT, intent);
            focus_main_window(app);
        }
        Err(reason) => {
            let _ = app.emit(DEEP_LINK_REJECTED_EVENT, reason);
        }
    }
}

pub fn install(app: &mut App) {
    let handle = app.handle().clone();
    app.deep_link().on_open_url(move |event| {
        for url in event.urls() {
            accept_url(&handle, url.as_str());
        }
    });

    if let Ok(Some(urls)) = app.deep_link().get_current() {
        for url in urls {
            accept_url(app.handle(), url.as_str());
        }
    }
}

#[tauri::command]
pub fn cockpit_take_deep_link(state: State<'_, PendingDeepLink>) -> Option<CockpitDeepLinkIntent> {
    state.0.lock().ok().and_then(|mut pending| pending.take())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accepts_the_canonical_live_session_route() {
        let intent =
            parse_cockpit_deep_link("cockpit://live/session/session_01?handoff=handoff:abc-123")
                .unwrap();
        assert_eq!(intent.route, CockpitDeepLinkRoute::LiveSession);
        assert_eq!(intent.target_ref.as_deref(), Some("session_01"));
        assert_eq!(intent.handoff_ref.as_deref(), Some("handoff:abc-123"));
    }

    #[test]
    fn accepts_focusa_evidence_and_pairing_routes() {
        assert_eq!(
            parse_cockpit_deep_link("cockpit://focusa/workpoint:abc")
                .unwrap()
                .route,
            CockpitDeepLinkRoute::Focusa
        );
        assert_eq!(
            parse_cockpit_deep_link("cockpit://evidence/evidence_123")
                .unwrap()
                .route,
            CockpitDeepLinkRoute::Evidence
        );
        assert_eq!(
            parse_cockpit_deep_link("cockpit://settings/pairing")
                .unwrap()
                .route,
            CockpitDeepLinkRoute::SettingsPairing
        );
    }

    #[test]
    fn rejects_unknown_fields_routes_and_encoded_refs() {
        for raw in [
            "https://live/session/session_01",
            "cockpit://live/session/session_01?unknown=value",
            "cockpit://live/session/session_01?handoff=one&handoff=two",
            "cockpit://live/session/%2Fadmin",
            "cockpit://settings/other",
            "cockpit://live/session/session_01#fragment",
        ] {
            assert!(parse_cockpit_deep_link(raw).is_err(), "accepted {raw}");
        }
    }
}
