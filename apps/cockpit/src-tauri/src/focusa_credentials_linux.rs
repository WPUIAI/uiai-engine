#[cfg(target_os = "linux")]
use crate::focusa_credentials::CREDENTIAL_SERVICE;
use crate::focusa_credentials::{
    CredentialError, CredentialHandle, CredentialSecret, CredentialStatus, CredentialStore,
};

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum LinuxBackendError {
    Missing,
    Locked,
    Denied,
    Unavailable,
    Failure,
}

trait LinuxCredentialBackend: Send + Sync {
    fn set(&self, account: &str, secret: &str) -> Result<(), LinuxBackendError>;
    fn get(&self, account: &str) -> Result<String, LinuxBackendError>;
    fn delete(&self, account: &str) -> Result<(), LinuxBackendError>;
}

#[cfg(target_os = "linux")]
pub(crate) struct SecretServiceBackend;

#[cfg(target_os = "linux")]
fn map_keyring(error: keyring::Error) -> LinuxBackendError {
    match error {
        keyring::Error::NoEntry => LinuxBackendError::Missing,
        keyring::Error::NoStorageAccess(ref source) => {
            let message = source.to_string().to_ascii_lowercase();
            if message.contains("locked") {
                LinuxBackendError::Locked
            } else if message.contains("denied") || message.contains("permission") {
                LinuxBackendError::Denied
            } else {
                LinuxBackendError::Unavailable
            }
        }
        keyring::Error::PlatformFailure(_) => LinuxBackendError::Failure,
        _ => LinuxBackendError::Failure,
    }
}

#[cfg(target_os = "linux")]
impl SecretServiceBackend {
    fn entry(account: &str) -> Result<keyring::Entry, LinuxBackendError> {
        keyring::Entry::new(CREDENTIAL_SERVICE, account).map_err(map_keyring)
    }
}

#[cfg(target_os = "linux")]
impl LinuxCredentialBackend for SecretServiceBackend {
    fn set(&self, account: &str, secret: &str) -> Result<(), LinuxBackendError> {
        Self::entry(account)?
            .set_password(secret)
            .map_err(map_keyring)
    }
    fn get(&self, account: &str) -> Result<String, LinuxBackendError> {
        Self::entry(account)?.get_password().map_err(map_keyring)
    }
    fn delete(&self, account: &str) -> Result<(), LinuxBackendError> {
        Self::entry(account)?
            .delete_credential()
            .map_err(map_keyring)
    }
}

pub struct LinuxCredentialStore<B> {
    backend: B,
}

#[cfg(target_os = "linux")]
impl Default for LinuxCredentialStore<SecretServiceBackend> {
    fn default() -> Self {
        Self {
            backend: SecretServiceBackend,
        }
    }
}

fn map_backend(error: LinuxBackendError) -> CredentialError {
    match error {
        LinuxBackendError::Missing => CredentialError::Missing,
        LinuxBackendError::Locked => CredentialError::Locked,
        LinuxBackendError::Denied => CredentialError::Denied,
        LinuxBackendError::Unavailable => CredentialError::BackendUnavailable,
        LinuxBackendError::Failure => CredentialError::BackendFailure,
    }
}

impl<B: LinuxCredentialBackend> CredentialStore for LinuxCredentialStore<B> {
    fn write(
        &self,
        handle: &CredentialHandle,
        secret: CredentialSecret,
    ) -> Result<(), CredentialError> {
        self.backend
            .set(handle.as_str(), secret.expose_native())
            .map_err(map_backend)
    }
    fn read(&self, handle: &CredentialHandle) -> Result<CredentialSecret, CredentialError> {
        CredentialSecret::new(self.backend.get(handle.as_str()).map_err(map_backend)?)
    }
    fn delete(&self, handle: &CredentialHandle) -> Result<(), CredentialError> {
        self.backend.delete(handle.as_str()).map_err(map_backend)
    }
    fn status(&self, handle: &CredentialHandle) -> CredentialStatus {
        match self.backend.get(handle.as_str()) {
            Ok(_) => CredentialStatus::Available,
            Err(LinuxBackendError::Missing) => CredentialStatus::Missing,
            Err(LinuxBackendError::Locked) => CredentialStatus::Locked,
            Err(LinuxBackendError::Denied) => CredentialStatus::Denied,
            Err(LinuxBackendError::Unavailable | LinuxBackendError::Failure) => {
                CredentialStatus::Unavailable
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{collections::BTreeMap, sync::Mutex};

    #[derive(Default)]
    struct MemoryBackend {
        values: Mutex<BTreeMap<String, String>>,
        failure: Mutex<Option<LinuxBackendError>>,
    }
    impl MemoryBackend {
        fn fail(&self, error: LinuxBackendError) {
            *self.failure.lock().unwrap() = Some(error)
        }
        fn check(&self) -> Result<(), LinuxBackendError> {
            self.failure.lock().unwrap().map_or(Ok(()), Err)
        }
    }
    impl LinuxCredentialBackend for MemoryBackend {
        fn set(&self, a: &str, s: &str) -> Result<(), LinuxBackendError> {
            self.check()?;
            self.values.lock().unwrap().insert(a.into(), s.into());
            Ok(())
        }
        fn get(&self, a: &str) -> Result<String, LinuxBackendError> {
            self.check()?;
            self.values
                .lock()
                .unwrap()
                .get(a)
                .cloned()
                .ok_or(LinuxBackendError::Missing)
        }
        fn delete(&self, a: &str) -> Result<(), LinuxBackendError> {
            self.check()?;
            self.values
                .lock()
                .unwrap()
                .remove(a)
                .map(|_| ())
                .ok_or(LinuxBackendError::Missing)
        }
    }

    #[test]
    fn secret_service_contract_round_trips_and_deletes() {
        let store = LinuxCredentialStore {
            backend: MemoryBackend::default(),
        };
        let h = CredentialHandle::parse("profile:linux-01").unwrap();
        store
            .write(
                &h,
                CredentialSecret::new("native-linux-secret".into()).unwrap(),
            )
            .unwrap();
        assert_eq!(store.status(&h), CredentialStatus::Available);
        assert_eq!(
            store.read(&h).unwrap().expose_native(),
            "native-linux-secret"
        );
        store.delete(&h).unwrap();
        assert_eq!(store.status(&h), CredentialStatus::Missing);
    }
    #[test]
    fn unavailable_locked_and_denied_fail_safely() {
        for (failure, status, error) in [
            (
                LinuxBackendError::Unavailable,
                CredentialStatus::Unavailable,
                CredentialError::BackendUnavailable,
            ),
            (
                LinuxBackendError::Locked,
                CredentialStatus::Locked,
                CredentialError::Locked,
            ),
            (
                LinuxBackendError::Denied,
                CredentialStatus::Denied,
                CredentialError::Denied,
            ),
        ] {
            let backend = MemoryBackend::default();
            backend.fail(failure);
            let store = LinuxCredentialStore { backend };
            let h = CredentialHandle::parse("profile:linux-02").unwrap();
            assert_eq!(store.status(&h), status);
            assert_eq!(store.read(&h).unwrap_err(), error);
        }
    }
}
