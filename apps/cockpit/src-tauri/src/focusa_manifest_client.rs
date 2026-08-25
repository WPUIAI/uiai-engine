use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    io::{Read, Write},
    net::{IpAddr, SocketAddr, TcpStream, ToSocketAddrs},
    time::Duration,
};
use url::Url;

const MANIFEST_PATH: &str = "/.well-known/focusa.json";
const MAX_RESPONSE_BYTES: usize = 32 * 1024;
const TIMEOUT: Duration = Duration::from_secs(2);

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct FocusaSiblingManifest {
    pub schema: String,
    pub app: String,
    pub version: String,
    pub channel: String,
    pub protocols: BTreeMap<String, String>,
    pub capabilities: Vec<String>,
}

fn resolve_loopback(url: &Url) -> Result<SocketAddr, String> {
    if url.scheme() != "http" || url.username() != "" || url.password().is_some() {
        return Err("sibling manifest endpoint must be credential-free loopback HTTP".into());
    }
    if url.path() != MANIFEST_PATH || url.query().is_some() || url.fragment().is_some() {
        return Err("sibling manifest endpoint path is invalid".into());
    }
    let port = url
        .port()
        .ok_or("sibling manifest endpoint requires an explicit port")?;
    let host = url
        .host_str()
        .ok_or("sibling manifest endpoint requires a host")?;
    let addresses: Vec<_> = (host, port)
        .to_socket_addrs()
        .map_err(|_| "sibling manifest host did not resolve")?
        .collect();
    if addresses.is_empty() || addresses.iter().any(|address| !address.ip().is_loopback()) {
        return Err("sibling manifest endpoint must resolve only to loopback".into());
    }
    addresses
        .into_iter()
        .find(|address| matches!(address.ip(), IpAddr::V4(_)))
        .ok_or_else(|| "sibling manifest endpoint has no supported loopback address".into())
}

pub fn fetch(endpoint: &str) -> Result<FocusaSiblingManifest, String> {
    if endpoint.len() > 2048 {
        return Err("sibling manifest endpoint is oversized".into());
    }
    let url = Url::parse(endpoint).map_err(|_| "sibling manifest endpoint is invalid")?;
    let address = resolve_loopback(&url)?;
    let mut stream = TcpStream::connect_timeout(&address, TIMEOUT)
        .map_err(|_| "sibling manifest endpoint is unavailable")?;
    stream
        .set_read_timeout(Some(TIMEOUT))
        .map_err(|error| error.to_string())?;
    stream
        .set_write_timeout(Some(TIMEOUT))
        .map_err(|error| error.to_string())?;
    stream
        .write_all(format!("GET {MANIFEST_PATH} HTTP/1.1\r\nHost: localhost\r\nAccept: application/json\r\nConnection: close\r\n\r\n").as_bytes())
        .map_err(|_| "sibling manifest request failed")?;

    let mut response = Vec::new();
    stream
        .take((MAX_RESPONSE_BYTES + 1) as u64)
        .read_to_end(&mut response)
        .map_err(|_| "sibling manifest response failed")?;
    if response.len() > MAX_RESPONSE_BYTES {
        return Err("sibling manifest response is oversized".into());
    }
    let separator = response
        .windows(4)
        .position(|window| window == b"\r\n\r\n")
        .ok_or("sibling manifest response is malformed")?;
    let headers = std::str::from_utf8(&response[..separator])
        .map_err(|_| "sibling manifest headers are invalid")?;
    if !headers
        .lines()
        .next()
        .is_some_and(|line| line == "HTTP/1.1 200 OK")
    {
        return Err("sibling manifest response was not successful".into());
    }
    let manifest: FocusaSiblingManifest = serde_json::from_slice(&response[separator + 4..])
        .map_err(|_| "sibling manifest body is invalid")?;
    if manifest.schema != "focusa.app.manifest.v2"
        || manifest.app.is_empty()
        || manifest.version.is_empty()
        || manifest.channel.is_empty()
        || manifest.protocols.is_empty()
        || manifest.capabilities.is_empty()
        || manifest
            .protocols
            .iter()
            .any(|(key, value)| key.len() > 64 || value.len() > 32)
        || manifest
            .capabilities
            .iter()
            .any(|value| value.is_empty() || value.len() > 128)
    {
        return Err("sibling manifest contract is invalid".into());
    }
    Ok(manifest)
}

#[tauri::command]
pub async fn cockpit_fetch_focusa_manifest(
    endpoint: String,
) -> Result<FocusaSiblingManifest, String> {
    tauri::async_runtime::spawn_blocking(move || fetch(&endpoint))
        .await
        .map_err(|_| "sibling manifest client stopped unexpectedly".to_string())?
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn reads_the_local_cockpit_manifest() {
        let endpoint = crate::focusa_manifest_server::start().unwrap();
        let manifest = fetch(&endpoint.0).unwrap();
        assert_eq!(manifest.schema, "focusa.app.manifest.v2");
        assert_eq!(
            manifest.protocols.get("pairing").map(String::as_str),
            Some("1")
        );
    }

    #[test]
    fn rejects_remote_credentialed_and_wrong_path_endpoints() {
        for endpoint in [
            "https://example.com/.well-known/focusa.json",
            "http://user:password@127.0.0.1:8787/.well-known/focusa.json",
            "http://127.0.0.1:8787/private",
            "http://127.0.0.1:8787/.well-known/focusa.json?token=no",
        ] {
            assert!(fetch(endpoint).is_err(), "accepted {endpoint}");
        }
    }
}
