//! Bonjour / mDNS discovery — sibling of menubar's focusa_discover_via_bonjour.

use serde::Serialize;

#[derive(Serialize)]
pub struct BonjourDiscovery {
    pub url: String,
    pub host: String,
    pub port: u16,
}

#[cfg(target_os = "macos")]
#[tauri::command]
pub async fn focusa_discover_via_bonjour(
    timeout_secs: Option<u64>,
) -> Result<Option<BonjourDiscovery>, String> {
    use mdns_sd::ServiceDaemon;
    let timeout_secs = timeout_secs.unwrap_or(2);
    let daemon = ServiceDaemon::new().map_err(|e| format!("mdns daemon: {e}"))?;
    let receiver = daemon
        .browse("_focusa._tcp.local")
        .map_err(|e| format!("mdns browse: {e}"))?;
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(timeout_secs);
    while std::time::Instant::now() < deadline {
        if let Ok(event) = tokio::time::timeout(
            std::time::Duration::from_millis(200),
            receiver.recv_async(),
        )
        .await
        {
            if let Ok(mdns_sd::ServiceEvent::ServiceResolved(info)) = event {
                let host = info.get_fullname().to_string();
                let port = info.get_port();
                let url = format!("http://{}:{}", host.trim_end_matches('.'), port);
                let _ = daemon.shutdown();
                return Ok(Some(BonjourDiscovery { url, host, port }));
            }
        }
    }
    let _ = daemon.shutdown();
    Ok(None)
}

#[cfg(not(target_os = "macos"))]
#[tauri::command]
pub async fn focusa_discover_via_bonjour(
    _timeout_secs: Option<u64>,
) -> Result<Option<BonjourDiscovery>, String> {
    // mDNS/Bonjour discovery is most useful on macOS LAN.
    // On other platforms, the operator can use Tailscale MagicDNS or CLI paste.
    Ok(None)
}