use serde::Serialize;
use std::{
    io::{Read, Write},
    net::{IpAddr, Ipv4Addr, SocketAddr, TcpListener, TcpStream},
    thread,
    time::Duration,
};

const MANIFEST_PATH: &str = "/.well-known/focusa.json";
const MAX_REQUEST_BYTES: usize = 4096;

#[derive(Clone, Debug)]
pub struct FocusaManifestEndpoint(pub String);

#[derive(Serialize)]
struct Protocols<'a> {
    focusa_deep_link: &'a str,
    cockpit_deep_link: &'a str,
    desktop_presentation: &'a str,
    fpv: &'a str,
    pairing: &'a str,
    bridge: &'a str,
    scope_context: &'a str,
}

#[derive(Serialize)]
struct AppManifest<'a> {
    schema: &'a str,
    app: &'a str,
    version: &'a str,
    channel: &'a str,
    protocols: Protocols<'a>,
    capabilities: [&'a str; 8],
}

fn manifest_json() -> Vec<u8> {
    serde_json::to_vec(&AppManifest {
        schema: "focusa.app.manifest.v2",
        app: "uiai-cockpit",
        version: env!("CARGO_PKG_VERSION"),
        channel: if cfg!(debug_assertions) {
            "dev"
        } else {
            "stable"
        },
        protocols: Protocols {
            focusa_deep_link: "1",
            cockpit_deep_link: "1",
            desktop_presentation: "1",
            fpv: "1",
            pairing: "1",
            bridge: "1",
            scope_context: "1",
        },
        capabilities: [
            "mission.open",
            "workpoint.open",
            "session.present",
            "pair.start",
            "pair.status",
            "pair.inherited",
            "scope.enumerate",
            "manifest.read",
        ],
    })
    .expect("static Focusa manifest must serialize")
}

fn write_response(stream: &mut TcpStream, status: &str, content_type: &str, body: &[u8]) {
    let header = format!(
        "HTTP/1.1 {status}\r\nContent-Type: {content_type}\r\nContent-Length: {}\r\nCache-Control: no-store\r\nX-Content-Type-Options: nosniff\r\nConnection: close\r\n\r\n",
        body.len()
    );
    let _ = stream.write_all(header.as_bytes());
    let _ = stream.write_all(body);
    let _ = stream.flush();
}

fn serve(mut stream: TcpStream) {
    let _ = stream.set_read_timeout(Some(Duration::from_secs(2)));
    let mut request = [0_u8; MAX_REQUEST_BYTES];
    let Ok(size) = stream.read(&mut request) else {
        return;
    };
    if size == 0 {
        return;
    }
    let first_line = String::from_utf8_lossy(&request[..size])
        .lines()
        .next()
        .unwrap_or_default()
        .to_owned();
    match first_line.as_str() {
        line if line == format!("GET {MANIFEST_PATH} HTTP/1.1") => write_response(
            &mut stream,
            "200 OK",
            "application/json; charset=utf-8",
            &manifest_json(),
        ),
        line if line.starts_with(&format!("GET {MANIFEST_PATH} ")) => write_response(
            &mut stream,
            "505 HTTP Version Not Supported",
            "text/plain",
            b"unsupported HTTP version",
        ),
        line if line.split_whitespace().nth(1) == Some(MANIFEST_PATH) => write_response(
            &mut stream,
            "405 Method Not Allowed",
            "text/plain",
            b"read only",
        ),
        _ => write_response(&mut stream, "404 Not Found", "text/plain", b"not found"),
    }
}

pub fn start() -> Result<FocusaManifestEndpoint, String> {
    let listener = TcpListener::bind(SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 0))
        .map_err(|error| format!("failed to bind Focusa manifest loopback server: {error}"))?;
    let address = listener.local_addr().map_err(|error| error.to_string())?;
    thread::Builder::new()
        .name("focusa-manifest".into())
        .spawn(move || {
            for stream in listener.incoming() {
                match stream {
                    Ok(stream) => serve(stream),
                    Err(_) => break,
                }
            }
        })
        .map_err(|error| format!("failed to start Focusa manifest server: {error}"))?;
    Ok(FocusaManifestEndpoint(format!(
        "http://{address}{MANIFEST_PATH}"
    )))
}

#[tauri::command]
pub fn cockpit_focusa_manifest_endpoint(state: tauri::State<'_, FocusaManifestEndpoint>) -> String {
    state.0.clone()
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::Value;

    fn request(endpoint: &str, line: &str) -> String {
        let url = url::Url::parse(endpoint).unwrap();
        let mut stream =
            TcpStream::connect((url.host_str().unwrap(), url.port().unwrap())).unwrap();
        stream
            .write_all(format!("{line}\r\nHost: localhost\r\n\r\n").as_bytes())
            .unwrap();
        stream.shutdown(std::net::Shutdown::Write).unwrap();
        let mut response = String::new();
        stream.read_to_string(&mut response).unwrap();
        response
    }

    #[test]
    fn serves_credential_free_manifest_on_loopback() {
        let endpoint = start().unwrap();
        assert!(endpoint.0.starts_with("http://127.0.0.1:"));
        let response = request(&endpoint.0, "GET /.well-known/focusa.json HTTP/1.1");
        assert!(response.starts_with("HTTP/1.1 200 OK"));
        let body = response.split("\r\n\r\n").nth(1).unwrap();
        let value: Value = serde_json::from_str(body).unwrap();
        assert_eq!(value["schema"], "focusa.app.manifest.v2");
        assert!(body.find("token").is_none());
        assert!(body.find("secret").is_none());
    }

    #[test]
    fn rejects_mutation_and_unknown_routes() {
        let endpoint = start().unwrap();
        assert!(
            request(&endpoint.0, "POST /.well-known/focusa.json HTTP/1.1")
                .starts_with("HTTP/1.1 405")
        );
        assert!(request(&endpoint.0, "GET /private HTTP/1.1").starts_with("HTTP/1.1 404"));
    }
}
