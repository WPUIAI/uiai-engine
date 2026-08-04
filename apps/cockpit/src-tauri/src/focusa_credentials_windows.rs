#[cfg(target_os = "windows")]
use crate::focusa_credentials::CREDENTIAL_SERVICE;
use crate::focusa_credentials::{
    CredentialError, CredentialHandle, CredentialSecret, CredentialStatus, CredentialStore,
};

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum WindowsBackendError {
    Missing,
    Denied,
    Unavailable,
    Failure,
}
trait WindowsCredentialBackend: Send + Sync {
    fn set(&self, a: &str, s: &str) -> Result<(), WindowsBackendError>;
    fn get(&self, a: &str) -> Result<String, WindowsBackendError>;
    fn delete(&self, a: &str) -> Result<(), WindowsBackendError>;
}

#[cfg(target_os = "windows")]
pub(crate) struct WindowsNativeBackend;
#[cfg(target_os = "windows")]
fn map_keyring(error: keyring::Error) -> WindowsBackendError {
    match error {
        keyring::Error::NoEntry => WindowsBackendError::Missing,
        keyring::Error::NoStorageAccess(_) => WindowsBackendError::Denied,
        keyring::Error::PlatformFailure(_) => WindowsBackendError::Failure,
        _ => WindowsBackendError::Unavailable,
    }
}
#[cfg(target_os = "windows")]
impl WindowsNativeBackend {
    fn entry(account: &str) -> Result<keyring::Entry, WindowsBackendError> {
        keyring::Entry::new(CREDENTIAL_SERVICE, account).map_err(map_keyring)
    }
}
#[cfg(target_os = "windows")]
impl WindowsCredentialBackend for WindowsNativeBackend {
    fn set(&self, a: &str, s: &str) -> Result<(), WindowsBackendError> {
        Self::entry(a)?.set_password(s).map_err(map_keyring)
    }
    fn get(&self, a: &str) -> Result<String, WindowsBackendError> {
        Self::entry(a)?.get_password().map_err(map_keyring)
    }
    fn delete(&self, a: &str) -> Result<(), WindowsBackendError> {
        Self::entry(a)?.delete_credential().map_err(map_keyring)
    }
}

pub struct WindowsCredentialStore<B> {
    backend: B,
}
#[cfg(target_os = "windows")]
impl Default for WindowsCredentialStore<WindowsNativeBackend> {
    fn default() -> Self {
        Self {
            backend: WindowsNativeBackend,
        }
    }
}
fn map_backend(e: WindowsBackendError) -> CredentialError {
    match e {
        WindowsBackendError::Missing => CredentialError::Missing,
        WindowsBackendError::Denied => CredentialError::Denied,
        WindowsBackendError::Unavailable => CredentialError::BackendUnavailable,
        WindowsBackendError::Failure => CredentialError::BackendFailure,
    }
}
impl<B: WindowsCredentialBackend> CredentialStore for WindowsCredentialStore<B> {
    fn write(&self, h: &CredentialHandle, s: CredentialSecret) -> Result<(), CredentialError> {
        self.backend
            .set(h.as_str(), s.expose_native())
            .map_err(map_backend)
    }
    fn read(&self, h: &CredentialHandle) -> Result<CredentialSecret, CredentialError> {
        CredentialSecret::new(self.backend.get(h.as_str()).map_err(map_backend)?)
    }
    fn delete(&self, h: &CredentialHandle) -> Result<(), CredentialError> {
        self.backend.delete(h.as_str()).map_err(map_backend)
    }
    fn status(&self, h: &CredentialHandle) -> CredentialStatus {
        match self.backend.get(h.as_str()) {
            Ok(_) => CredentialStatus::Available,
            Err(WindowsBackendError::Missing) => CredentialStatus::Missing,
            Err(WindowsBackendError::Denied) => CredentialStatus::Denied,
            Err(WindowsBackendError::Unavailable | WindowsBackendError::Failure) => {
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
        failure: Mutex<Option<WindowsBackendError>>,
    }
    impl MemoryBackend {
        fn fail(&self, e: WindowsBackendError) {
            *self.failure.lock().unwrap() = Some(e)
        }
        fn check(&self) -> Result<(), WindowsBackendError> {
            self.failure.lock().unwrap().map_or(Ok(()), Err)
        }
    }
    impl WindowsCredentialBackend for MemoryBackend {
        fn set(&self, a: &str, s: &str) -> Result<(), WindowsBackendError> {
            self.check()?;
            self.values.lock().unwrap().insert(a.into(), s.into());
            Ok(())
        }
        fn get(&self, a: &str) -> Result<String, WindowsBackendError> {
            self.check()?;
            self.values
                .lock()
                .unwrap()
                .get(a)
                .cloned()
                .ok_or(WindowsBackendError::Missing)
        }
        fn delete(&self, a: &str) -> Result<(), WindowsBackendError> {
            self.check()?;
            self.values
                .lock()
                .unwrap()
                .remove(a)
                .map(|_| ())
                .ok_or(WindowsBackendError::Missing)
        }
    }
    #[test]
    fn credential_manager_contract_round_trips_and_deletes() {
        let store = WindowsCredentialStore {
            backend: MemoryBackend::default(),
        };
        let h = CredentialHandle::parse("profile:windows-01").unwrap();
        store
            .write(
                &h,
                CredentialSecret::new("native-windows-secret".into()).unwrap(),
            )
            .unwrap();
        assert_eq!(store.status(&h), CredentialStatus::Available);
        assert_eq!(
            store.read(&h).unwrap().expose_native(),
            "native-windows-secret"
        );
        store.delete(&h).unwrap();
        assert_eq!(store.status(&h), CredentialStatus::Missing);
    }
    #[test]
    fn denied_and_unavailable_fail_safely() {
        for (failure, status, error) in [
            (
                WindowsBackendError::Denied,
                CredentialStatus::Denied,
                CredentialError::Denied,
            ),
            (
                WindowsBackendError::Unavailable,
                CredentialStatus::Unavailable,
                CredentialError::BackendUnavailable,
            ),
        ] {
            let backend = MemoryBackend::default();
            backend.fail(failure);
            let store = WindowsCredentialStore { backend };
            let h = CredentialHandle::parse("profile:windows-02").unwrap();
            assert_eq!(store.status(&h), status);
            assert_eq!(store.read(&h).unwrap_err(), error);
        }
    }
}
