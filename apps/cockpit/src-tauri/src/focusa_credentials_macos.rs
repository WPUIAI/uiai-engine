#![cfg(target_os = "macos")]

use crate::focusa_credentials::{
    CredentialError, CredentialHandle, CredentialSecret, CredentialStatus, CredentialStore,
    CREDENTIAL_SERVICE,
};

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum MacBackendError {
    Missing,
    Locked,
    Denied,
    Unavailable,
    Failure,
}

trait MacCredentialBackend: Send + Sync {
    fn set(&self, account: &str, secret: &str) -> Result<(), MacBackendError>;
    fn get(&self, account: &str) -> Result<String, MacBackendError>;
    fn delete(&self, account: &str) -> Result<(), MacBackendError>;
}

pub(crate) struct KeyringMacBackend;

impl KeyringMacBackend {
    fn entry(account: &str) -> Result<keyring::Entry, MacBackendError> {
        keyring::Entry::new(CREDENTIAL_SERVICE, account).map_err(map_keyring_error)
    }
}

fn map_keyring_error(error: keyring::Error) -> MacBackendError {
    match error {
        keyring::Error::NoEntry => MacBackendError::Missing,
        keyring::Error::NoStorageAccess(_) => MacBackendError::Denied,
        keyring::Error::PlatformFailure(ref source)
            if source
                .to_string()
                .to_ascii_lowercase()
                .contains("interaction") =>
        {
            MacBackendError::Locked
        }
        keyring::Error::PlatformFailure(_) => MacBackendError::Failure,
        keyring::Error::BadEncoding(_)
        | keyring::Error::TooLong(_, _)
        | keyring::Error::Invalid(_, _)
        | keyring::Error::Ambiguous(_) => MacBackendError::Failure,
        _ => MacBackendError::Unavailable,
    }
}

impl MacCredentialBackend for KeyringMacBackend {
    fn set(&self, account: &str, secret: &str) -> Result<(), MacBackendError> {
        Self::entry(account)?
            .set_password(secret)
            .map_err(map_keyring_error)
    }

    fn get(&self, account: &str) -> Result<String, MacBackendError> {
        Self::entry(account)?
            .get_password()
            .map_err(map_keyring_error)
    }

    fn delete(&self, account: &str) -> Result<(), MacBackendError> {
        Self::entry(account)?
            .delete_credential()
            .map_err(map_keyring_error)
    }
}

pub struct MacOsCredentialStore<B = KeyringMacBackend> {
    backend: B,
}

impl Default for MacOsCredentialStore<KeyringMacBackend> {
    fn default() -> Self {
        Self {
            backend: KeyringMacBackend,
        }
    }
}

fn map_backend(error: MacBackendError) -> CredentialError {
    match error {
        MacBackendError::Missing => CredentialError::Missing,
        MacBackendError::Locked => CredentialError::Locked,
        MacBackendError::Denied => CredentialError::Denied,
        MacBackendError::Unavailable => CredentialError::BackendUnavailable,
        MacBackendError::Failure => CredentialError::BackendFailure,
    }
}

impl<B: MacCredentialBackend> CredentialStore for MacOsCredentialStore<B> {
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
        let value = self.backend.get(handle.as_str()).map_err(map_backend)?;
        CredentialSecret::new(value)
    }

    fn delete(&self, handle: &CredentialHandle) -> Result<(), CredentialError> {
        self.backend.delete(handle.as_str()).map_err(map_backend)
    }

    fn status(&self, handle: &CredentialHandle) -> CredentialStatus {
        match self.backend.get(handle.as_str()) {
            Ok(_) => CredentialStatus::Available,
            Err(MacBackendError::Missing) => CredentialStatus::Missing,
            Err(MacBackendError::Locked) => CredentialStatus::Locked,
            Err(MacBackendError::Denied) => CredentialStatus::Denied,
            Err(MacBackendError::Unavailable | MacBackendError::Failure) => {
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
        failure: Mutex<Option<MacBackendError>>,
    }

    impl MemoryBackend {
        fn fail_with(&self, error: MacBackendError) {
            *self.failure.lock().unwrap() = Some(error);
        }
        fn failure(&self) -> Result<(), MacBackendError> {
            self.failure.lock().unwrap().map_or(Ok(()), Err)
        }
    }

    impl MacCredentialBackend for MemoryBackend {
        fn set(&self, account: &str, secret: &str) -> Result<(), MacBackendError> {
            self.failure()?;
            self.values
                .lock()
                .unwrap()
                .insert(account.into(), secret.into());
            Ok(())
        }
        fn get(&self, account: &str) -> Result<String, MacBackendError> {
            self.failure()?;
            self.values
                .lock()
                .unwrap()
                .get(account)
                .cloned()
                .ok_or(MacBackendError::Missing)
        }
        fn delete(&self, account: &str) -> Result<(), MacBackendError> {
            self.failure()?;
            self.values
                .lock()
                .unwrap()
                .remove(account)
                .map(|_| ())
                .ok_or(MacBackendError::Missing)
        }
    }

    #[test]
    fn mac_adapter_round_trips_and_deletes_without_exposing_secret() {
        let store = MacOsCredentialStore {
            backend: MemoryBackend::default(),
        };
        let handle = CredentialHandle::parse("profile:mac-01").unwrap();
        store
            .write(
                &handle,
                CredentialSecret::new("native-secret".into()).unwrap(),
            )
            .unwrap();
        assert_eq!(store.status(&handle), CredentialStatus::Available);
        assert_eq!(
            store.read(&handle).unwrap().expose_native(),
            "native-secret"
        );
        store.delete(&handle).unwrap();
        assert_eq!(store.status(&handle), CredentialStatus::Missing);
    }

    #[test]
    fn mac_adapter_maps_locked_denied_and_unavailable_states() {
        for (failure, status, error) in [
            (
                MacBackendError::Locked,
                CredentialStatus::Locked,
                CredentialError::Locked,
            ),
            (
                MacBackendError::Denied,
                CredentialStatus::Denied,
                CredentialError::Denied,
            ),
            (
                MacBackendError::Unavailable,
                CredentialStatus::Unavailable,
                CredentialError::BackendUnavailable,
            ),
        ] {
            let backend = MemoryBackend::default();
            backend.fail_with(failure);
            let store = MacOsCredentialStore { backend };
            let handle = CredentialHandle::parse("profile:mac-02").unwrap();
            assert_eq!(store.status(&handle), status);
            assert_eq!(store.read(&handle).unwrap_err(), error);
        }
    }
}
