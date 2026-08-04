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
}
