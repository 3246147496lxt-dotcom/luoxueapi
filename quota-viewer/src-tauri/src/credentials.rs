use std::{
    fs::{self, OpenOptions},
    io::Write,
    path::{Path, PathBuf},
    sync::{Arc, Mutex, MutexGuard},
};

use rand::{rngs::OsRng, RngCore};
use serde::{Deserialize, Serialize};
use zeroize::Zeroizing;

use crate::{
    error::{AppError, AppResult},
    file_replace::replace_file,
    models::PendingRefreshRotation,
};

const CREDENTIAL_FILE_VERSION: u8 = 1;
const MAX_CREDENTIAL_FILE_SIZE: u64 = 64 * 1024;

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct CredentialFile {
    version: u8,
    refresh_token: Option<Zeroizing<String>>,
    installation_id: Option<Zeroizing<String>>,
    snapshot_key: Option<Zeroizing<String>>,
    cache_id: Option<Zeroizing<String>>,
    pending_refresh_rotation: Option<PendingRefreshRotation>,
}

impl Default for CredentialFile {
    fn default() -> Self {
        Self {
            version: CREDENTIAL_FILE_VERSION,
            refresh_token: None,
            installation_id: None,
            snapshot_key: None,
            cache_id: None,
            pending_refresh_rotation: None,
        }
    }
}

impl CredentialFile {
    fn validate(&self) -> AppResult<()> {
        if self.version != CREDENTIAL_FILE_VERSION {
            return Err(AppError::Message(
                "credential file version is unsupported".into(),
            ));
        }
        if self
            .pending_refresh_rotation
            .as_ref()
            .is_some_and(|pending| !pending.is_valid())
        {
            return Err(AppError::RefreshStateInvalid);
        }
        Ok(())
    }
}

#[derive(Clone)]
pub struct CredentialStore {
    path: PathBuf,
    data: Arc<Mutex<CredentialFile>>,
}

impl CredentialStore {
    pub fn open(path: PathBuf) -> AppResult<Self> {
        let parent = path
            .parent()
            .ok_or_else(|| AppError::Message("credential file has no parent directory".into()))?;
        fs::create_dir_all(parent)?;
        restrict_directory_permissions(parent)?;

        let data = match fs::symlink_metadata(&path) {
            Ok(metadata) => {
                if !metadata.file_type().is_file() || metadata.file_type().is_symlink() {
                    return Err(AppError::Message(
                        "credential path must be a regular file".into(),
                    ));
                }
                if metadata.len() > MAX_CREDENTIAL_FILE_SIZE {
                    return Err(AppError::Message("credential file is too large".into()));
                }
                restrict_file_permissions(&path)?;
                let bytes = Zeroizing::new(fs::read(&path)?);
                let plaintext = unseal_credential_bytes(bytes.as_slice())?;
                let data = serde_json::from_slice::<CredentialFile>(plaintext.as_slice())?;
                data.validate()?;
                data
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => CredentialFile::default(),
            Err(error) => return Err(error.into()),
        };
        Ok(Self {
            path,
            data: Arc::new(Mutex::new(data)),
        })
    }

    fn data(&self) -> AppResult<MutexGuard<'_, CredentialFile>> {
        self.data
            .lock()
            .map_err(|_| AppError::Message("credential file lock is unavailable".into()))
    }

    fn update<T>(&self, mutate: impl FnOnce(&mut CredentialFile) -> AppResult<T>) -> AppResult<T> {
        let mut current = self.data()?;
        let mut next = current.clone();
        let result = mutate(&mut next)?;
        next.validate()?;
        self.persist(&next)?;
        *current = next;
        Ok(result)
    }

    fn persist(&self, data: &CredentialFile) -> AppResult<()> {
        let bytes = Zeroizing::new(serde_json::to_vec(data)?);
        if bytes.len() as u64 > MAX_CREDENTIAL_FILE_SIZE {
            return Err(AppError::Message("credential file is too large".into()));
        }
        let sealed = seal_credential_bytes(bytes.as_slice())?;
        if sealed.len() as u64 > MAX_CREDENTIAL_FILE_SIZE {
            return Err(AppError::Message("credential file is too large".into()));
        }
        write_private_atomic(&self.path, sealed.as_slice())?;
        Ok(())
    }

    pub fn refresh_token(&self) -> AppResult<Option<Zeroizing<String>>> {
        Ok(self.data()?.refresh_token.clone())
    }

    pub fn require_refresh_token(&self) -> AppResult<Zeroizing<String>> {
        self.refresh_token()?
            .ok_or_else(|| AppError::Message("quota viewer account is not connected".into()))
    }

    #[cfg(test)]
    pub fn set_refresh_token(&self, token: &str) -> AppResult<()> {
        self.update(|data| {
            data.refresh_token = Some(Zeroizing::new(token.to_owned()));
            Ok(())
        })
    }

    #[cfg(test)]
    pub fn clear_refresh_token(&self) -> AppResult<()> {
        self.update(|data| {
            data.refresh_token = None;
            Ok(())
        })
    }

    pub fn commit_pairing(&self, refresh_token: &str) -> AppResult<()> {
        self.update(|data| {
            data.refresh_token = Some(Zeroizing::new(refresh_token.to_owned()));
            data.pending_refresh_rotation = None;
            data.cache_id = Some(generate_cache_id());
            Ok(())
        })
    }

    pub fn commit_refresh_rotation(&self, refresh_token: &str) -> AppResult<()> {
        self.update(|data| {
            data.refresh_token = Some(Zeroizing::new(refresh_token.to_owned()));
            data.pending_refresh_rotation = None;
            Ok(())
        })
    }

    pub fn clear_authorization(&self) -> AppResult<()> {
        self.update(|data| {
            data.refresh_token = None;
            data.pending_refresh_rotation = None;
            data.cache_id = None;
            Ok(())
        })
    }

    pub fn pending_refresh_rotation(&self) -> AppResult<Option<PendingRefreshRotation>> {
        Ok(self.data()?.pending_refresh_rotation.clone())
    }

    pub fn set_pending_refresh_rotation(&self, pending: &PendingRefreshRotation) -> AppResult<()> {
        if !pending.is_valid() {
            return Err(AppError::RefreshStateInvalid);
        }
        self.update(|data| {
            data.pending_refresh_rotation = Some(pending.clone());
            Ok(())
        })
    }

    pub fn clear_pending_refresh_rotation(&self) -> AppResult<()> {
        self.update(|data| {
            data.pending_refresh_rotation = None;
            Ok(())
        })
    }

    pub fn installation_id(&self) -> AppResult<Zeroizing<String>> {
        let existing = { self.data()?.installation_id.clone() };
        if let Some(value) = existing {
            return Ok(value);
        }
        self.update(|data| {
            if let Some(value) = data.installation_id.clone() {
                return Ok(value);
            }
            let mut random = Zeroizing::new([0_u8; 32]);
            OsRng.fill_bytes(&mut *random);
            let value =
                base64::Engine::encode(&base64::engine::general_purpose::URL_SAFE_NO_PAD, *random);
            data.installation_id = Some(Zeroizing::new(value.clone()));
            Ok(Zeroizing::new(value))
        })
    }

    pub fn snapshot_key(&self) -> AppResult<Zeroizing<Vec<u8>>> {
        let existing = { self.data()?.snapshot_key.clone() };
        let encoded = match existing {
            Some(value) => value,
            None => self.update(|data| {
                if let Some(value) = data.snapshot_key.clone() {
                    return Ok(value);
                }
                let mut random = Zeroizing::new([0_u8; 32]);
                OsRng.fill_bytes(&mut *random);
                let encoded = base64::Engine::encode(
                    &base64::engine::general_purpose::URL_SAFE_NO_PAD,
                    *random,
                );
                data.snapshot_key = Some(Zeroizing::new(encoded.clone()));
                Ok(Zeroizing::new(encoded))
            })?,
        };
        let decoded = base64::Engine::decode(
            &base64::engine::general_purpose::URL_SAFE_NO_PAD,
            encoded.as_bytes(),
        )
        .map_err(|_| AppError::Message("snapshot encryption key is invalid".into()))?;
        if decoded.len() != 32 {
            return Err(AppError::Message(
                "snapshot encryption key has an invalid length".into(),
            ));
        }
        Ok(Zeroizing::new(decoded))
    }

    pub fn cache_id(&self) -> AppResult<Option<Zeroizing<String>>> {
        Ok(self.data()?.cache_id.clone())
    }

    pub fn rotate_cache_id(&self) -> AppResult<Zeroizing<String>> {
        self.update(|data| {
            let value = generate_cache_id();
            data.cache_id = Some(value.clone());
            Ok(value)
        })
    }

    #[cfg(test)]
    pub fn clear_cache_id(&self) -> AppResult<()> {
        self.update(|data| {
            data.cache_id = None;
            Ok(())
        })
    }
}

fn generate_cache_id() -> Zeroizing<String> {
    let mut random = Zeroizing::new([0_u8; 16]);
    OsRng.fill_bytes(&mut *random);
    Zeroizing::new(base64::Engine::encode(
        &base64::engine::general_purpose::URL_SAFE_NO_PAD,
        *random,
    ))
}

fn write_private_atomic(path: &Path, bytes: &[u8]) -> std::io::Result<()> {
    let parent = path.parent().ok_or_else(|| {
        std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "credential file has no parent directory",
        )
    })?;
    fs::create_dir_all(parent)?;
    restrict_directory_permissions(parent)?;

    let temporary = path.with_extension(format!("tmp-{}", uuid::Uuid::new_v4()));
    let mut options = OpenOptions::new();
    options.create_new(true).write(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        options.mode(0o600);
    }
    let mut file = options.open(&temporary)?;
    let write_result = (|| {
        file.write_all(bytes)?;
        file.sync_all()?;
        restrict_file_permissions(&temporary)?;
        replace_file(&temporary, path)?;
        let _ = sync_directory(parent);
        Ok(())
    })();
    if write_result.is_err() {
        let _ = fs::remove_file(&temporary);
    }
    write_result
}

#[cfg(not(windows))]
fn seal_credential_bytes(bytes: &[u8]) -> std::io::Result<Zeroizing<Vec<u8>>> {
    Ok(Zeroizing::new(bytes.to_vec()))
}

#[cfg(not(windows))]
fn unseal_credential_bytes(bytes: &[u8]) -> std::io::Result<Zeroizing<Vec<u8>>> {
    Ok(Zeroizing::new(bytes.to_vec()))
}

#[cfg(windows)]
const WINDOWS_CREDENTIAL_MAGIC: &[u8] = b"LQVCRED1\0";

#[cfg(windows)]
fn seal_credential_bytes(bytes: &[u8]) -> std::io::Result<Zeroizing<Vec<u8>>> {
    use std::ptr;
    use windows_sys::Win32::Security::Cryptography::{
        CryptProtectData, CRYPTPROTECT_UI_FORBIDDEN, CRYPT_INTEGER_BLOB,
    };

    let input_length = u32::try_from(bytes.len()).map_err(|_| {
        std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "credential payload is too large",
        )
    })?;
    let input = CRYPT_INTEGER_BLOB {
        cbData: input_length,
        pbData: bytes.as_ptr().cast_mut(),
    };
    let mut output = CRYPT_INTEGER_BLOB::default();
    // SAFETY: input points at `bytes` for the duration of the call, all optional
    // pointers are null, and `output` is initialized for CryptProtectData.
    let protected = unsafe {
        CryptProtectData(
            &input,
            ptr::null(),
            ptr::null(),
            ptr::null(),
            ptr::null(),
            CRYPTPROTECT_UI_FORBIDDEN,
            &mut output,
        )
    };
    if protected == 0 {
        return Err(std::io::Error::last_os_error());
    }

    let protected = take_dpapi_output(output)?;
    let mut sealed = Zeroizing::new(Vec::with_capacity(
        WINDOWS_CREDENTIAL_MAGIC.len() + protected.len(),
    ));
    sealed.extend_from_slice(WINDOWS_CREDENTIAL_MAGIC);
    sealed.extend_from_slice(protected.as_slice());
    Ok(sealed)
}

#[cfg(windows)]
fn unseal_credential_bytes(bytes: &[u8]) -> std::io::Result<Zeroizing<Vec<u8>>> {
    use std::ptr;
    use windows_sys::Win32::Security::Cryptography::{
        CryptUnprotectData, CRYPTPROTECT_UI_FORBIDDEN, CRYPT_INTEGER_BLOB,
    };

    let protected = bytes
        .strip_prefix(WINDOWS_CREDENTIAL_MAGIC)
        .ok_or_else(|| {
            std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "credential file is not protected for the current Windows user",
            )
        })?;
    let input_length = u32::try_from(protected.len()).map_err(|_| {
        std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "credential payload is too large",
        )
    })?;
    let input = CRYPT_INTEGER_BLOB {
        cbData: input_length,
        pbData: protected.as_ptr().cast_mut(),
    };
    let mut output = CRYPT_INTEGER_BLOB::default();
    // SAFETY: input points at `protected` for the duration of the call, all
    // optional pointers are null, and `output` is initialized for the API.
    let unprotected = unsafe {
        CryptUnprotectData(
            &input,
            ptr::null_mut(),
            ptr::null(),
            ptr::null(),
            ptr::null(),
            CRYPTPROTECT_UI_FORBIDDEN,
            &mut output,
        )
    };
    if unprotected == 0 {
        return Err(std::io::Error::last_os_error());
    }
    take_dpapi_output(output)
}

#[cfg(windows)]
fn take_dpapi_output(
    output: windows_sys::Win32::Security::Cryptography::CRYPT_INTEGER_BLOB,
) -> std::io::Result<Zeroizing<Vec<u8>>> {
    use windows_sys::Win32::Foundation::LocalFree;

    if output.pbData.is_null() || output.cbData == 0 {
        if !output.pbData.is_null() {
            // SAFETY: DPAPI owns this zero-length LocalAlloc buffer, if any.
            unsafe {
                let _ = LocalFree(output.pbData.cast());
            }
        }
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidData,
            "Windows returned an invalid protected credential buffer",
        ));
    }
    // SAFETY: successful DPAPI calls return a LocalAlloc-owned buffer of
    // exactly cbData bytes. Copy it into zeroizing Rust memory, wipe the OS
    // buffer, then release it with LocalFree as required by DPAPI.
    let bytes = unsafe {
        let slice = std::slice::from_raw_parts(output.pbData, output.cbData as usize);
        let copied = Zeroizing::new(slice.to_vec());
        if output.cbData > 0 {
            std::ptr::write_bytes(output.pbData, 0, output.cbData as usize);
        }
        let _ = LocalFree(output.pbData.cast());
        copied
    };
    Ok(bytes)
}

#[cfg(unix)]
fn sync_directory(path: &Path) -> std::io::Result<()> {
    fs::File::open(path)?.sync_all()
}

#[cfg(not(unix))]
fn sync_directory(_path: &Path) -> std::io::Result<()> {
    Ok(())
}

#[cfg(unix)]
fn restrict_directory_permissions(path: &Path) -> std::io::Result<()> {
    use std::os::unix::fs::PermissionsExt;

    fs::set_permissions(path, fs::Permissions::from_mode(0o700))
}

#[cfg(not(unix))]
fn restrict_directory_permissions(_path: &Path) -> std::io::Result<()> {
    Ok(())
}

#[cfg(unix)]
fn restrict_file_permissions(path: &Path) -> std::io::Result<()> {
    use std::os::unix::fs::PermissionsExt;

    fs::set_permissions(path, fs::Permissions::from_mode(0o600))
}

#[cfg(not(unix))]
fn restrict_file_permissions(_path: &Path) -> std::io::Result<()> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn credentials_persist_across_restarts_in_a_private_file() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("credentials-v1.json");
        let store = CredentialStore::open(path.clone()).unwrap();
        let installation_id = store.installation_id().unwrap();
        let snapshot_key = store.snapshot_key().unwrap();
        store.set_refresh_token("refresh-secret").unwrap();
        let pending = PendingRefreshRotation::generate("refresh-secret");
        store.set_pending_refresh_rotation(&pending).unwrap();
        let cache_id = store.rotate_cache_id().unwrap();

        assert_eq!(fs::read_dir(directory.path()).unwrap().count(), 1);
        let disk = fs::read(&path).unwrap();
        #[cfg(windows)]
        assert!(!disk
            .windows("refresh-secret".len())
            .any(|window| window == b"refresh-secret"));
        #[cfg(not(windows))]
        assert!(String::from_utf8(disk).unwrap().contains("refresh-secret"));

        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;

            assert_eq!(
                fs::metadata(&path).unwrap().permissions().mode() & 0o777,
                0o600
            );
            assert_eq!(
                fs::metadata(directory.path()).unwrap().permissions().mode() & 0o777,
                0o700
            );
        }

        drop(store);
        let reopened = CredentialStore::open(path).unwrap();
        assert_eq!(
            reopened.refresh_token().unwrap().unwrap().as_str(),
            "refresh-secret"
        );
        assert_eq!(
            reopened.installation_id().unwrap().as_str(),
            installation_id.as_str()
        );
        assert_eq!(
            reopened.snapshot_key().unwrap().as_slice(),
            snapshot_key.as_slice()
        );
        assert_eq!(
            reopened.cache_id().unwrap().unwrap().as_str(),
            cache_id.as_str()
        );
        assert!(reopened.pending_refresh_rotation().unwrap().is_some());
    }

    #[test]
    fn cleared_credentials_stay_cleared_after_reopening() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("credentials-v1.json");
        let store = CredentialStore::open(path.clone()).unwrap();
        store.snapshot_key().unwrap();
        store.set_refresh_token("refresh-secret").unwrap();
        store
            .set_pending_refresh_rotation(&PendingRefreshRotation::generate("refresh-secret"))
            .unwrap();
        store.rotate_cache_id().unwrap();

        store.clear_refresh_token().unwrap();
        store.clear_pending_refresh_rotation().unwrap();
        store.clear_cache_id().unwrap();
        drop(store);

        let reopened = CredentialStore::open(path).unwrap();
        assert!(reopened.refresh_token().unwrap().is_none());
        assert!(reopened.pending_refresh_rotation().unwrap().is_none());
        assert!(reopened.cache_id().unwrap().is_none());
    }

    #[test]
    fn unsupported_credential_file_versions_fail_closed() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("credentials-v1.json");
        let unsupported = seal_credential_bytes(
            br#"{"version":2,"refresh_token":null,"installation_id":null,"snapshot_key":null,"cache_id":null,"pending_refresh_rotation":null}"#,
        )
        .unwrap();
        fs::write(&path, unsupported.as_slice()).unwrap();

        let error = match CredentialStore::open(path) {
            Ok(_) => panic!("unsupported credential file should fail"),
            Err(error) => error,
        };
        assert_eq!(error.to_string(), "credential file version is unsupported");
    }

    #[test]
    fn authorization_transitions_are_committed_together() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("credentials-v1.json");
        let store = CredentialStore::open(path.clone()).unwrap();
        store.snapshot_key().unwrap();
        store
            .set_pending_refresh_rotation(&PendingRefreshRotation::generate("old-refresh"))
            .unwrap();

        store.commit_pairing("paired-refresh").unwrap();
        assert_eq!(
            store.refresh_token().unwrap().unwrap().as_str(),
            "paired-refresh"
        );
        assert!(store.pending_refresh_rotation().unwrap().is_none());
        assert!(store.cache_id().unwrap().is_some());

        let next = PendingRefreshRotation::generate("paired-refresh");
        let candidate = next.candidate_refresh_token.clone();
        store.set_pending_refresh_rotation(&next).unwrap();
        store.commit_refresh_rotation(candidate.as_str()).unwrap();
        assert_eq!(
            store.refresh_token().unwrap().unwrap().as_str(),
            candidate.as_str()
        );
        assert!(store.pending_refresh_rotation().unwrap().is_none());

        store.clear_authorization().unwrap();
        drop(store);
        let reopened = CredentialStore::open(path).unwrap();
        assert!(reopened.refresh_token().unwrap().is_none());
        assert!(reopened.pending_refresh_rotation().unwrap().is_none());
        assert!(reopened.cache_id().unwrap().is_none());
    }

    #[cfg(unix)]
    #[test]
    fn opening_existing_credentials_repairs_private_permissions() {
        use std::os::unix::fs::PermissionsExt;

        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("credentials-v1.json");
        fs::write(
            &path,
            serde_json::to_vec(&CredentialFile::default()).unwrap(),
        )
        .unwrap();
        fs::set_permissions(directory.path(), fs::Permissions::from_mode(0o755)).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o644)).unwrap();

        CredentialStore::open(path.clone()).unwrap();

        assert_eq!(
            fs::metadata(directory.path()).unwrap().permissions().mode() & 0o777,
            0o700
        );
        assert_eq!(
            fs::metadata(path).unwrap().permissions().mode() & 0o777,
            0o600
        );
    }

    #[cfg(unix)]
    #[test]
    fn symbolic_link_credential_paths_are_rejected() {
        use std::os::unix::fs::symlink;

        let directory = tempfile::tempdir().unwrap();
        let target = directory.path().join("target.json");
        fs::write(
            &target,
            serde_json::to_vec(&CredentialFile::default()).unwrap(),
        )
        .unwrap();
        let link = directory.path().join("credentials-v1.json");
        symlink(&target, &link).unwrap();

        let error = match CredentialStore::open(link) {
            Ok(_) => panic!("symbolic links must not be followed"),
            Err(error) => error,
        };
        assert_eq!(error.to_string(), "credential path must be a regular file");
    }

    #[test]
    fn oversized_credential_files_are_rejected() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("credentials-v1.json");
        fs::write(&path, vec![b'x'; MAX_CREDENTIAL_FILE_SIZE as usize + 1]).unwrap();

        let error = match CredentialStore::open(path) {
            Ok(_) => panic!("oversized credential files must not be read"),
            Err(error) => error,
        };
        assert_eq!(error.to_string(), "credential file is too large");
    }

    #[cfg(windows)]
    #[test]
    fn plaintext_windows_credentials_are_rejected() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("credentials-v1.json");
        fs::write(
            &path,
            serde_json::to_vec(&CredentialFile::default()).unwrap(),
        )
        .unwrap();

        let error = match CredentialStore::open(path) {
            Ok(_) => panic!("plaintext Windows credentials must fail closed"),
            Err(error) => error,
        };
        assert!(error
            .to_string()
            .contains("not protected for the current Windows user"));
    }
}
