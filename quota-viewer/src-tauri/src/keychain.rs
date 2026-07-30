use rand::{rngs::OsRng, RngCore};
use zeroize::Zeroizing;

use crate::{
    error::{AppError, AppResult},
    models::PendingRefreshRotation,
};

const SERVICE: &str = "cc.luoxueapi.quota-viewer";
const REFRESH_TOKEN_ACCOUNT: &str = "refresh-token";
const INSTALLATION_ID_ACCOUNT: &str = "installation-id";
const SNAPSHOT_KEY_ACCOUNT: &str = "snapshot-key";
const CACHE_ID_ACCOUNT: &str = "cache-id";
const PENDING_REFRESH_ROTATION_ACCOUNT: &str = "pending-refresh-rotation-v1";

#[derive(Clone, Default)]
pub struct CredentialStore;

impl CredentialStore {
    fn entry(account: &str) -> AppResult<keyring::Entry> {
        Ok(keyring::Entry::new(SERVICE, account)?)
    }

    fn get(&self, account: &str) -> AppResult<Option<Zeroizing<String>>> {
        match Self::entry(account)?.get_password() {
            Ok(value) => Ok(Some(Zeroizing::new(value))),
            Err(keyring::Error::NoEntry) => Ok(None),
            Err(error) => Err(error.into()),
        }
    }

    fn set(&self, account: &str, value: &str) -> AppResult<()> {
        Self::entry(account)?.set_password(value)?;
        Ok(())
    }

    fn delete(&self, account: &str) -> AppResult<()> {
        match Self::entry(account)?.delete_credential() {
            Ok(()) | Err(keyring::Error::NoEntry) => Ok(()),
            Err(error) => Err(error.into()),
        }
    }

    pub fn refresh_token(&self) -> AppResult<Option<Zeroizing<String>>> {
        self.get(REFRESH_TOKEN_ACCOUNT)
    }

    pub fn require_refresh_token(&self) -> AppResult<Zeroizing<String>> {
        self.refresh_token()?
            .ok_or_else(|| AppError::Message("quota viewer account is not connected".into()))
    }

    pub fn set_refresh_token(&self, token: &str) -> AppResult<()> {
        self.set(REFRESH_TOKEN_ACCOUNT, token)
    }

    pub fn clear_refresh_token(&self) -> AppResult<()> {
        self.delete(REFRESH_TOKEN_ACCOUNT)
    }

    pub fn pending_refresh_rotation(&self) -> AppResult<Option<PendingRefreshRotation>> {
        let Some(encoded) = self.get(PENDING_REFRESH_ROTATION_ACCOUNT)? else {
            return Ok(None);
        };
        let pending = serde_json::from_str::<PendingRefreshRotation>(encoded.as_str())
            .map_err(|_| AppError::RefreshStateInvalid)?;
        if !pending.is_valid() {
            return Err(AppError::RefreshStateInvalid);
        }
        Ok(Some(pending))
    }

    pub fn set_pending_refresh_rotation(&self, pending: &PendingRefreshRotation) -> AppResult<()> {
        if !pending.is_valid() {
            return Err(AppError::RefreshStateInvalid);
        }
        let encoded = Zeroizing::new(serde_json::to_string(pending)?);
        self.set(PENDING_REFRESH_ROTATION_ACCOUNT, encoded.as_str())
    }

    pub fn clear_pending_refresh_rotation(&self) -> AppResult<()> {
        self.delete(PENDING_REFRESH_ROTATION_ACCOUNT)
    }

    pub fn installation_id(&self) -> AppResult<Zeroizing<String>> {
        if let Some(value) = self.get(INSTALLATION_ID_ACCOUNT)? {
            return Ok(value);
        }
        let mut random = [0_u8; 32];
        OsRng.fill_bytes(&mut random);
        let value =
            base64::Engine::encode(&base64::engine::general_purpose::URL_SAFE_NO_PAD, random);
        self.set(INSTALLATION_ID_ACCOUNT, &value)?;
        Ok(Zeroizing::new(value))
    }

    pub fn snapshot_key(&self) -> AppResult<Zeroizing<Vec<u8>>> {
        let encoded = match self.get(SNAPSHOT_KEY_ACCOUNT)? {
            Some(value) => value,
            None => {
                let mut random = [0_u8; 32];
                OsRng.fill_bytes(&mut random);
                let encoded = base64::Engine::encode(
                    &base64::engine::general_purpose::URL_SAFE_NO_PAD,
                    random,
                );
                self.set(SNAPSHOT_KEY_ACCOUNT, &encoded)?;
                Zeroizing::new(encoded)
            }
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
        self.get(CACHE_ID_ACCOUNT)
    }

    pub fn rotate_cache_id(&self) -> AppResult<Zeroizing<String>> {
        let mut random = [0_u8; 16];
        OsRng.fill_bytes(&mut random);
        let value =
            base64::Engine::encode(&base64::engine::general_purpose::URL_SAFE_NO_PAD, random);
        self.set(CACHE_ID_ACCOUNT, &value)?;
        Ok(Zeroizing::new(value))
    }

    pub fn clear_cache_id(&self) -> AppResult<()> {
        self.delete(CACHE_ID_ACCOUNT)
    }
}
