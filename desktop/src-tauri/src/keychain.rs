use rand::{rngs::OsRng, RngCore};
use zeroize::Zeroizing;

use crate::error::{AppError, AppResult};

const SERVICE: &str = "cc.luoxueapi.desktop";

#[derive(Clone, Default)]
pub struct CredentialStore;

impl CredentialStore {
    fn entry(account: &str) -> AppResult<keyring::Entry> {
        Ok(keyring::Entry::new(SERVICE, account)?)
    }

    pub fn get(&self, account: &str) -> AppResult<Option<Zeroizing<String>>> {
        match Self::entry(account)?.get_password() {
            Ok(value) => Ok(Some(Zeroizing::new(value))),
            Err(keyring::Error::NoEntry) => Ok(None),
            Err(error) => Err(error.into()),
        }
    }

    pub fn set(&self, account: &str, value: &str) -> AppResult<()> {
        Self::entry(account)?.set_password(value)?;
        Ok(())
    }

    pub fn delete(&self, account: &str) -> AppResult<()> {
        match Self::entry(account)?.delete_credential() {
            Ok(()) | Err(keyring::Error::NoEntry) => Ok(()),
            Err(error) => Err(error.into()),
        }
    }

    pub fn get_or_create_random(
        &self,
        account: &str,
        bytes: usize,
    ) -> AppResult<Zeroizing<String>> {
        if let Some(value) = self.get(account)? {
            return Ok(value);
        }
        let mut random = vec![0_u8; bytes];
        OsRng.fill_bytes(&mut random);
        let value =
            base64::Engine::encode(&base64::engine::general_purpose::URL_SAFE_NO_PAD, random);
        self.set(account, &value)?;
        Ok(Zeroizing::new(value))
    }

    pub fn managed_key_account(group_id: i64) -> String {
        format!("managed-key:{group_id}")
    }

    pub fn require(&self, account: &str) -> AppResult<Zeroizing<String>> {
        self.get(account)?
            .ok_or_else(|| AppError::Message(format!("missing credential: {account}")))
    }
}
