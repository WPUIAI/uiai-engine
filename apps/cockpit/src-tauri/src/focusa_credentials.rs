use serde::Serialize;
use std::fmt;
use zeroize::Zeroizing;

pub const CREDENTIAL_SERVICE: &str = "UIAI Engine Cockpit Token";

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct CredentialHandle(String);

impl CredentialHandle {
    pub fn parse(value: impl Into<String>) -> Result<Self, CredentialError> {
        let value = value.into();
        if value.is_empty()
            || value.len() > 256
            || !value.bytes().all(|byte| {
                byte.is_ascii_alphanumeric() || matches!(byte, b'.' | b'_' | b'~' | b':' | b'-')
            })
        {
            return Err(CredentialError::InvalidHandle);
        }
        Ok(Self(value))
    }

    pub fn as_str(&self) -> &str {
        &self.0
    }
}

/// Native-only secret material. It cannot be serialized or cloned, is always
/// redacted in debug/display output, and is zeroized when dropped.
pub struct CredentialSecret(Zeroizing<String>);

impl CredentialSecret {
    pub fn new(value: String) -> Result<Self, CredentialError> {
        if value.is_empty() || value.len() > 16 * 1024 {
            return Err(CredentialError::InvalidSecret);
        }
        Ok(Self(Zeroizing::new(value)))
    }

    pub(crate) fn expose_native(&self) -> &str {
        self.0.as_str()
    }
}

impl fmt::Debug for CredentialSecret {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str("CredentialSecret([REDACTED])")
    }
}

impl fmt::Display for CredentialSecret {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str("[REDACTED]")
    }
}

#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum CredentialStatus {
    Available,
    Missing,
    Locked,
    Denied,
    Unavailable,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct CredentialDescriptor {
    pub handle: CredentialHandle,
    pub service: &'static str,
    pub status: CredentialStatus,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum CredentialError {
    InvalidHandle,
    InvalidSecret,
    Missing,
    Locked,
    Denied,
    BackendUnavailable,
    BackendFailure,
}

impl fmt::Display for CredentialError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        let message = match self {
            Self::InvalidHandle => "credential handle is invalid",
            Self::InvalidSecret => "credential secret is invalid",
            Self::Missing => "credential is missing",
            Self::Locked => "credential store is locked",
            Self::Denied => "credential access was denied",
            Self::BackendUnavailable => "credential store is unavailable",
            Self::BackendFailure => "credential operation failed",
        };
        formatter.write_str(message)
    }
}

/// Implementations remain native. In particular, `read` is never exposed as a
/// Tauri command; callers use it only inside native authenticated requests.
pub trait CredentialStore: Send + Sync {
    fn write(
        &self,
        handle: &CredentialHandle,
        secret: CredentialSecret,
    ) -> Result<(), CredentialError>;
    fn read(&self, handle: &CredentialHandle) -> Result<CredentialSecret, CredentialError>;
    fn delete(&self, handle: &CredentialHandle) -> Result<(), CredentialError>;
    fn status(&self, handle: &CredentialHandle) -> CredentialStatus;
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct NativeRequest {
    pub method: String,
    pub url: String,
    pub body: Option<Vec<u8>>,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct NativeResponse {
    pub status: u16,
    pub body: Vec<u8>,
}

pub trait NativeRequestExecutor: Send + Sync {
    fn execute(
        &self,
        request: NativeRequest,
        authorization: &str,
    ) -> Result<NativeResponse, CredentialError>;
}

/// Resolves credentials only while executing native requests. Neither the raw
/// credential nor the Authorization value can be returned through this API.
pub struct NativeCredentialResolver<S, E> {
    store: S,
    executor: E,
}

impl<S: CredentialStore, E: NativeRequestExecutor> NativeCredentialResolver<S, E> {
    pub fn new(store: S, executor: E) -> Self {
        Self { store, executor }
    }

    pub fn execute(
        &self,
        handle: &CredentialHandle,
        request: NativeRequest,
    ) -> Result<NativeResponse, CredentialError> {
        let secret = self.store.read(handle)?;
        let authorization = Zeroizing::new(format!("Bearer {}", secret.expose_native()));
        let response = self.executor.execute(request, authorization.as_str())?;
        if response
            .body
            .windows(secret.expose_native().len())
            .any(|window| window == secret.expose_native().as_bytes())
        {
            return Err(CredentialError::BackendFailure);
        }
        Ok(response)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn secret_is_redacted_and_descriptor_contains_only_handle_and_status() {
        let raw = "super-sensitive-daemon-token";
        let secret = CredentialSecret::new(raw.to_owned()).unwrap();
        assert_eq!(secret.expose_native(), raw);
        assert!(!format!("{secret:?}").contains(raw));
        assert!(!format!("{secret}").contains(raw));

        let descriptor = CredentialDescriptor {
            handle: CredentialHandle::parse("profile:daemon-01").unwrap(),
            service: CREDENTIAL_SERVICE,
            status: CredentialStatus::Available,
        };
        let json = serde_json::to_string(&descriptor).unwrap();
        assert!(json.contains("profile:daemon-01"));
        assert!(!json.contains(raw));
    }

    #[test]
    fn handles_and_secret_sizes_are_bounded() {
        for invalid in ["", "has space", "has/slash", "has?query"] {
            assert_eq!(
                CredentialHandle::parse(invalid),
                Err(CredentialError::InvalidHandle)
            );
        }
        assert!(CredentialHandle::parse("profile_01:daemon-01").is_ok());
        assert!(CredentialSecret::new(String::new()).is_err());
        assert!(CredentialSecret::new("x".repeat(16 * 1024 + 1)).is_err());
    }

    #[test]
    fn errors_are_stable_and_secret_free() {
        for error in [
            CredentialError::Missing,
            CredentialError::Locked,
            CredentialError::Denied,
            CredentialError::BackendUnavailable,
            CredentialError::BackendFailure,
        ] {
            let rendered = error.to_string();
            assert!(!rendered.contains("token"));
            assert!(!rendered.contains("secret"));
        }
    }

    struct ReadOnlyStore;
    impl CredentialStore for ReadOnlyStore {
        fn write(&self, _: &CredentialHandle, _: CredentialSecret) -> Result<(), CredentialError> {
            unreachable!()
        }
        fn read(&self, _: &CredentialHandle) -> Result<CredentialSecret, CredentialError> {
            CredentialSecret::new("native-token-value".into())
        }
        fn delete(&self, _: &CredentialHandle) -> Result<(), CredentialError> {
            unreachable!()
        }
        fn status(&self, _: &CredentialHandle) -> CredentialStatus {
            CredentialStatus::Available
        }
    }
    struct Executor {
        echo_secret: bool,
    }
    impl NativeRequestExecutor for Executor {
        fn execute(
            &self,
            _: NativeRequest,
            authorization: &str,
        ) -> Result<NativeResponse, CredentialError> {
            assert_eq!(authorization, "Bearer native-token-value");
            Ok(NativeResponse {
                status: 200,
                body: if self.echo_secret {
                    b"native-token-value".to_vec()
                } else {
                    b"{\"ok\":true}".to_vec()
                },
            })
        }
    }

    #[test]
    fn native_resolver_injects_authority_without_returning_it() {
        let resolver =
            NativeCredentialResolver::new(ReadOnlyStore, Executor { echo_secret: false });
        let response = resolver
            .execute(
                &CredentialHandle::parse("profile:native-01").unwrap(),
                NativeRequest {
                    method: "GET".into(),
                    url: "https://focusa.example/v1/projects".into(),
                    body: None,
                },
            )
            .unwrap();
        assert_eq!(response.status, 200);
        assert!(!response
            .body
            .windows(18)
            .any(|window| window == b"native-token-value"));
    }

    #[test]
    fn native_resolver_blocks_reflected_credentials() {
        let resolver = NativeCredentialResolver::new(ReadOnlyStore, Executor { echo_secret: true });
        assert_eq!(
            resolver.execute(
                &CredentialHandle::parse("profile:native-01").unwrap(),
                NativeRequest {
                    method: "GET".into(),
                    url: "https://focusa.example/v1/projects".into(),
                    body: None
                }
            ),
            Err(CredentialError::BackendFailure)
        );
    }
}
