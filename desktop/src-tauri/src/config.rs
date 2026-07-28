use std::{
    fs::{self, File, OpenOptions},
    io::Write,
    path::{Path, PathBuf},
};

use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
use fs2::FileExt;
use rand::{rngs::OsRng, RngCore};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use toml_edit::{DocumentMut, Item, Table};
use uuid::Uuid;

use crate::{
    error::{AppError, AppResult},
    keychain::CredentialStore,
};

const JOURNAL_VERSION: u32 = 1;
const PROVIDER_NAME: &str = "luoxue_desktop";

#[derive(Clone)]
pub struct ConfigManager {
    codex_dir: PathBuf,
    support_dir: PathBuf,
    credentials: CredentialStore,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum ConfigState {
    Clean,
    Managed,
    Conflict,
    Invalid,
    PermissionDenied,
}

#[derive(Debug, Serialize, Deserialize)]
struct Journal {
    version: u32,
    operation_id: Uuid,
    phase: JournalPhase,
    original_config_hash: String,
    applied_config_hash: String,
    original_auth_hash: String,
    applied_auth_hash: String,
    encrypted_payload: String,
}

#[derive(Clone, Copy, Debug, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
enum JournalPhase {
    Prepared,
    AuthWritten,
    Applied,
    Restoring,
}

#[derive(Debug, Serialize, Deserialize)]
struct BackupPayload {
    original_config: Vec<u8>,
    original_auth: Vec<u8>,
    applied_config: Vec<u8>,
    applied_auth: Vec<u8>,
    config_mode: Option<u32>,
    auth_mode: Option<u32>,
}

impl ConfigManager {
    pub fn new(
        codex_dir: PathBuf,
        support_dir: PathBuf,
        credentials: CredentialStore,
    ) -> AppResult<Self> {
        fs::create_dir_all(&support_dir)?;
        Ok(Self {
            codex_dir,
            support_dir,
            credentials,
        })
    }

    pub fn status(&self) -> (ConfigState, Option<String>) {
        if self.journal_path().exists() {
            return (ConfigState::Managed, None);
        }
        let config_path = self.config_path();
        match read_optional(&config_path) {
            Ok(bytes) => {
                let source = String::from_utf8_lossy(&bytes);
                if source.to_ascii_lowercase().contains("ccswitch") {
                    return (
                        ConfigState::Conflict,
                        Some("CCSwitch is currently managing Codex".into()),
                    );
                }
                if source.parse::<DocumentMut>().is_err() {
                    return (
                        ConfigState::Invalid,
                        Some("config.toml cannot be parsed".into()),
                    );
                }
                (ConfigState::Clean, None)
            }
            Err(error) if error.kind() == std::io::ErrorKind::PermissionDenied => (
                ConfigState::PermissionDenied,
                Some("Codex configuration is not readable".into()),
            ),
            Err(error) => (ConfigState::Invalid, Some(error.to_string())),
        }
    }

    pub fn is_managed(&self) -> bool {
        self.journal_path().exists()
    }

    pub fn apply(&self, port: u16, local_token: &str, model: &str) -> AppResult<()> {
        fs::create_dir_all(&self.codex_dir)?;
        let lock = self.acquire_lock()?;
        let config_path = self.config_path();
        let auth_path = self.auth_path();
        ensure_safe_path(&config_path)?;
        ensure_safe_path(&auth_path)?;

        let original_config = read_optional(&config_path)?;
        let original_auth = read_optional(&auth_path)?;
        let source = String::from_utf8_lossy(&original_config);
        if source.to_ascii_lowercase().contains("ccswitch") {
            return Err(AppError::ConfigConflict(
                "CCSwitch takeover detected".into(),
            ));
        }

        let applied_config = build_managed_config(&original_config, port, model)?;
        let applied_auth = build_managed_auth(&original_auth, local_token)?;
        let payload = BackupPayload {
            config_mode: file_mode(&config_path),
            auth_mode: file_mode(&auth_path),
            original_config,
            original_auth,
            applied_config,
            applied_auth,
        };
        let mut journal = Journal {
            version: JOURNAL_VERSION,
            operation_id: Uuid::new_v4(),
            phase: JournalPhase::Prepared,
            original_config_hash: digest(&payload.original_config),
            applied_config_hash: digest(&payload.applied_config),
            original_auth_hash: digest(&payload.original_auth),
            applied_auth_hash: digest(&payload.applied_auth),
            encrypted_payload: self.encrypt_payload(&payload)?,
        };
        self.write_journal(&journal)?;

        atomic_write(
            &auth_path,
            &payload.applied_auth,
            payload.auth_mode.or(Some(0o600)),
        )?;
        journal.phase = JournalPhase::AuthWritten;
        self.write_journal(&journal)?;

        if let Err(error) = atomic_write(
            &config_path,
            &payload.applied_config,
            payload.config_mode.or(Some(0o600)),
        ) {
            let _ = atomic_write(
                &auth_path,
                &payload.original_auth,
                payload.auth_mode.or(Some(0o600)),
            );
            return Err(error);
        }
        journal.phase = JournalPhase::Applied;
        self.write_journal(&journal)?;

        verify_hash(&config_path, &journal.applied_config_hash)?;
        verify_hash(&auth_path, &journal.applied_auth_hash)?;
        FileExt::unlock(&lock)?;
        Ok(())
    }

    pub fn restore(&self) -> AppResult<()> {
        if !self.journal_path().exists() {
            return Ok(());
        }
        let lock = self.acquire_lock()?;
        let mut journal = self.read_journal()?;
        let payload = self.decrypt_payload(&journal.encrypted_payload)?;
        journal.phase = JournalPhase::Restoring;
        self.write_journal(&journal)?;

        let config_path = self.config_path();
        let auth_path = self.auth_path();
        ensure_safe_path(&config_path)?;
        ensure_safe_path(&auth_path)?;
        let current_config = read_optional(&config_path)?;
        let current_auth = read_optional(&auth_path)?;
        let restored_config = merge_managed_config(
            &payload.original_config,
            &payload.applied_config,
            &current_config,
        )?;
        let restored_auth =
            merge_managed_auth(&payload.original_auth, &payload.applied_auth, &current_auth)?;

        atomic_write(
            &config_path,
            &restored_config,
            payload.config_mode.or(Some(0o600)),
        )?;
        atomic_write(
            &auth_path,
            &restored_auth,
            payload.auth_mode.or(Some(0o600)),
        )?;
        fs::remove_file(self.journal_path())?;
        sync_parent(&self.journal_path())?;
        FileExt::unlock(&lock)?;
        Ok(())
    }

    pub fn recover_incomplete(&self) -> AppResult<()> {
        if !self.journal_path().exists() {
            return Ok(());
        }
        let journal = self.read_journal()?;
        if matches!(
            journal.phase,
            JournalPhase::Prepared | JournalPhase::AuthWritten | JournalPhase::Restoring
        ) {
            return self.restore();
        }
        Ok(())
    }

    fn acquire_lock(&self) -> AppResult<File> {
        let path = self.support_dir.join("codex-config.lock");
        let file = OpenOptions::new()
            .create(true)
            .read(true)
            .write(true)
            .open(path)?;
        file.lock_exclusive()?;
        Ok(file)
    }

    fn encrypt_payload(&self, payload: &BackupPayload) -> AppResult<String> {
        let encoded_key = self
            .credentials
            .get_or_create_random("config-backup-key", 32)?;
        let key = URL_SAFE_NO_PAD
            .decode(encoded_key.as_bytes())
            .map_err(|_| AppError::Message("invalid backup key encoding".into()))?;
        let cipher = Aes256Gcm::new_from_slice(&key)
            .map_err(|_| AppError::Message("invalid backup key".into()))?;
        let mut nonce_bytes = [0_u8; 12];
        OsRng.fill_bytes(&mut nonce_bytes);
        let plaintext = serde_json::to_vec(payload)?;
        let ciphertext = cipher
            .encrypt(Nonce::from_slice(&nonce_bytes), plaintext.as_ref())
            .map_err(|_| AppError::Message("failed to encrypt Codex backup".into()))?;
        let mut envelope = nonce_bytes.to_vec();
        envelope.extend_from_slice(&ciphertext);
        Ok(URL_SAFE_NO_PAD.encode(envelope))
    }

    fn decrypt_payload(&self, encrypted: &str) -> AppResult<BackupPayload> {
        let encoded_key = self.credentials.require("config-backup-key")?;
        let key = URL_SAFE_NO_PAD
            .decode(encoded_key.as_bytes())
            .map_err(|_| AppError::Message("invalid backup key encoding".into()))?;
        let envelope = URL_SAFE_NO_PAD
            .decode(encrypted)
            .map_err(|_| AppError::Message("invalid encrypted backup".into()))?;
        if envelope.len() < 13 {
            return Err(AppError::Message("encrypted backup is truncated".into()));
        }
        let cipher = Aes256Gcm::new_from_slice(&key)
            .map_err(|_| AppError::Message("invalid backup key".into()))?;
        let plaintext = cipher
            .decrypt(Nonce::from_slice(&envelope[..12]), &envelope[12..])
            .map_err(|_| AppError::Message("Codex backup authentication failed".into()))?;
        Ok(serde_json::from_slice(&plaintext)?)
    }

    fn read_journal(&self) -> AppResult<Journal> {
        Ok(serde_json::from_slice(&fs::read(self.journal_path())?)?)
    }

    fn write_journal(&self, journal: &Journal) -> AppResult<()> {
        atomic_write(
            &self.journal_path(),
            &serde_json::to_vec_pretty(journal)?,
            Some(0o600),
        )
    }

    fn config_path(&self) -> PathBuf {
        self.codex_dir.join("config.toml")
    }
    fn auth_path(&self) -> PathBuf {
        self.codex_dir.join("auth.json")
    }
    fn journal_path(&self) -> PathBuf {
        self.support_dir.join("codex-config-journal.json")
    }
}

fn build_managed_config(original: &[u8], port: u16, model: &str) -> AppResult<Vec<u8>> {
    let mut document = parse_toml(original)?;
    document["model_provider"] = toml_edit::value(PROVIDER_NAME);
    document["model"] = toml_edit::value(model);
    if !document.as_table().contains_key("model_providers") {
        document["model_providers"] = Item::Table(Table::new());
    }
    let providers = document["model_providers"]
        .as_table_mut()
        .ok_or_else(|| AppError::UnsafeConfig("model_providers is not a TOML table".into()))?;
    let mut provider = Table::new();
    provider["name"] = toml_edit::value("Luoxue API Desktop");
    provider["base_url"] = toml_edit::value(format!("http://127.0.0.1:{port}/v1"));
    provider["wire_api"] = toml_edit::value("responses");
    let mut auth = Table::new();
    auth["command"] = toml_edit::value("/usr/bin/security");
    let mut args = toml_edit::Array::new();
    for argument in [
        "find-generic-password",
        "-w",
        "-s",
        "cc.luoxueapi.desktop",
        "-a",
        "local-token",
    ] {
        args.push(argument);
    }
    auth["args"] = toml_edit::value(args);
    auth["timeout_ms"] = toml_edit::value(5_000);
    auth["refresh_interval_ms"] = toml_edit::value(0);
    provider["auth"] = Item::Table(auth);
    providers[PROVIDER_NAME] = Item::Table(provider);
    Ok(document.to_string().into_bytes())
}

fn build_managed_auth(original: &[u8], _local_token: &str) -> AppResult<Vec<u8>> {
    parse_json_object(original)?;
    Ok(original.to_vec())
}

fn merge_managed_config(original: &[u8], applied: &[u8], current: &[u8]) -> AppResult<Vec<u8>> {
    let original = parse_toml(original)?;
    let applied = parse_toml(applied)?;
    let mut current = parse_toml(current)?;
    if managed_config_matches(&current, &original) {
        return Ok(current.to_string().into_bytes());
    }
    merge_top_level_item(&mut current, &original, &applied, "model_provider")?;
    merge_top_level_item(&mut current, &original, &applied, "model")?;
    merge_provider_item(&mut current, &original, &applied)?;
    Ok(current.to_string().into_bytes())
}

fn merge_managed_auth(original: &[u8], applied: &[u8], current: &[u8]) -> AppResult<Vec<u8>> {
    if original == applied {
        parse_json_object(current)?;
        return Ok(current.to_vec());
    }
    let current_bytes = current;
    let original = parse_json_object(original)?;
    let applied = parse_json_object(applied)?;
    let mut current = parse_json_object(current)?;
    let current_value = current.get("OPENAI_API_KEY");
    let original_value = original.get("OPENAI_API_KEY");
    let applied_value = applied.get("OPENAI_API_KEY");
    if current_value == original_value {
        return Ok(current_bytes.to_vec());
    }
    if current_value != applied_value {
        return Err(AppError::ConfigConflict(
            "OPENAI_API_KEY changed outside LuoxueAPI Desktop".into(),
        ));
    }
    match original_value {
        Some(value) => {
            current.insert("OPENAI_API_KEY".into(), value.clone());
        }
        None => {
            current.remove("OPENAI_API_KEY");
        }
    }
    Ok(serde_json::to_vec_pretty(&current)?)
}

fn merge_top_level_item(
    current: &mut DocumentMut,
    original: &DocumentMut,
    applied: &DocumentMut,
    key: &str,
) -> AppResult<()> {
    let current_value = item_string(current.as_table().get(key));
    let original_value = item_string(original.as_table().get(key));
    let applied_value = item_string(applied.as_table().get(key));
    if current_value == original_value {
        return Ok(());
    }
    if current_value != applied_value {
        return Err(AppError::ConfigConflict(format!(
            "{key} changed outside LuoxueAPI Desktop"
        )));
    }
    match original.as_table().get(key) {
        Some(value) => {
            current.as_table_mut().insert(key, value.clone());
        }
        None => {
            current.as_table_mut().remove(key);
        }
    }
    Ok(())
}

fn merge_provider_item(
    current: &mut DocumentMut,
    original: &DocumentMut,
    applied: &DocumentMut,
) -> AppResult<()> {
    let current_item = provider_item(current);
    let applied_item = provider_item(applied);
    let original_item = provider_item(original);
    if item_string(current_item) == item_string(original_item) {
        return Ok(());
    }
    if item_string(current_item) != item_string(applied_item) {
        return Err(AppError::ConfigConflict(
            "luoxue_desktop provider changed outside LuoxueAPI Desktop".into(),
        ));
    }
    let original_item = original_item.cloned();
    let original_had_provider_table = original.as_table().contains_key("model_providers");
    let remove_empty_provider_table = {
        let providers = current
            .as_table_mut()
            .entry("model_providers")
            .or_insert(Item::Table(Table::new()));
        let table = providers
            .as_table_mut()
            .ok_or_else(|| AppError::UnsafeConfig("model_providers is not a TOML table".into()))?;
        match original_item {
            Some(value) => {
                table.insert(PROVIDER_NAME, value);
            }
            None => {
                table.remove(PROVIDER_NAME);
            }
        }
        !original_had_provider_table && table.is_empty()
    };
    if remove_empty_provider_table {
        current.as_table_mut().remove("model_providers");
    }
    Ok(())
}

fn managed_config_matches(current: &DocumentMut, expected: &DocumentMut) -> bool {
    ["model_provider", "model"].into_iter().all(|key| {
        item_string(current.as_table().get(key)) == item_string(expected.as_table().get(key))
    }) && item_string(provider_item(current)) == item_string(provider_item(expected))
}

fn provider_item(document: &DocumentMut) -> Option<&Item> {
    document
        .as_table()
        .get("model_providers")?
        .as_table()?
        .get(PROVIDER_NAME)
}

fn item_string(item: Option<&Item>) -> Option<String> {
    item.map(ToString::to_string)
}

fn parse_toml(bytes: &[u8]) -> AppResult<DocumentMut> {
    let source = std::str::from_utf8(bytes)
        .map_err(|_| AppError::UnsafeConfig("config.toml is not UTF-8".into()))?;
    Ok(source.parse::<DocumentMut>()?)
}

fn parse_json_object(bytes: &[u8]) -> AppResult<serde_json::Map<String, serde_json::Value>> {
    if bytes.is_empty() {
        return Ok(serde_json::Map::new());
    }
    serde_json::from_slice::<serde_json::Value>(bytes)?
        .as_object()
        .cloned()
        .ok_or_else(|| AppError::UnsafeConfig("auth.json root is not an object".into()))
}

fn read_optional(path: &Path) -> std::io::Result<Vec<u8>> {
    match fs::read(path) {
        Ok(bytes) => Ok(bytes),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(Vec::new()),
        Err(error) => Err(error),
    }
}

fn ensure_safe_path(path: &Path) -> AppResult<()> {
    match fs::symlink_metadata(path) {
        Ok(metadata) if metadata.file_type().is_symlink() => Err(AppError::UnsafeConfig(format!(
            "{} is a symbolic link",
            path.display()
        ))),
        Ok(metadata) if !metadata.is_file() => Err(AppError::UnsafeConfig(format!(
            "{} is not a regular file",
            path.display()
        ))),
        Ok(_) => Ok(()),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Err(error) => Err(error.into()),
    }
}

fn digest(bytes: &[u8]) -> String {
    URL_SAFE_NO_PAD.encode(Sha256::digest(bytes))
}

fn verify_hash(path: &Path, expected: &str) -> AppResult<()> {
    if digest(&fs::read(path)?) != expected {
        return Err(AppError::Message(format!(
            "write verification failed for {}",
            path.display()
        )));
    }
    Ok(())
}

fn atomic_write(path: &Path, bytes: &[u8], mode: Option<u32>) -> AppResult<()> {
    let parent = path
        .parent()
        .ok_or_else(|| AppError::Message("configuration path has no parent".into()))?;
    fs::create_dir_all(parent)?;
    let temp = parent.join(format!(
        ".{}.luoxue.{}.tmp",
        path.file_name()
            .and_then(|name| name.to_str())
            .unwrap_or("config"),
        Uuid::new_v4()
    ));
    let result = (|| -> AppResult<()> {
        let mut options = OpenOptions::new();
        options.create_new(true).write(true);
        let mut file = options.open(&temp)?;
        #[cfg(unix)]
        if let Some(mode) = mode {
            use std::os::unix::fs::PermissionsExt;
            file.set_permissions(fs::Permissions::from_mode(mode))?;
        }
        file.write_all(bytes)?;
        file.sync_all()?;
        fs::rename(&temp, path)?;
        sync_parent(path)?;
        Ok(())
    })();
    if result.is_err() {
        let _ = fs::remove_file(&temp);
    }
    result
}

fn sync_parent(path: &Path) -> AppResult<()> {
    if let Some(parent) = path.parent() {
        File::open(parent)?.sync_all()?;
    }
    Ok(())
}

#[cfg(unix)]
fn file_mode(path: &Path) -> Option<u32> {
    use std::os::unix::fs::MetadataExt;
    fs::metadata(path)
        .ok()
        .map(|metadata| metadata.mode() & 0o777)
}

#[cfg(not(unix))]
fn file_mode(_path: &Path) -> Option<u32> {
    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn restore_preserves_unrelated_toml_changes() {
        let original = b"model = \"old\"\nuser_setting = true\n";
        let applied = build_managed_config(original, 11430, "gpt-5.4").unwrap();
        let mut current = parse_toml(&applied).unwrap();
        current["new_setting"] = toml_edit::value(7);
        let restored = String::from_utf8(
            merge_managed_config(original, &applied, current.to_string().as_bytes()).unwrap(),
        )
        .unwrap();
        assert!(restored.contains("model = \"old\""));
        assert!(restored.contains("user_setting = true"));
        assert!(restored.contains("new_setting = 7"));
        assert!(!restored.contains("luoxue_desktop"));
        assert!(!restored.contains("[model_providers]"));
    }

    #[test]
    fn restore_rejects_owned_field_conflict() {
        let original = b"model = \"old\"\n";
        let applied = build_managed_config(original, 11430, "gpt-5.4").unwrap();
        let current = String::from_utf8(applied.clone())
            .unwrap()
            .replace("gpt-5.4", "gpt-5.3-codex");
        assert!(matches!(
            merge_managed_config(original, &applied, current.as_bytes()),
            Err(AppError::ConfigConflict(_))
        ));
    }

    #[test]
    fn managed_config_uses_keychain_command_without_openai_auth() {
        let applied = build_managed_config(b"", 11430, "gpt-5.6").unwrap();
        let document = parse_toml(&applied).unwrap();
        let provider = document["model_providers"][PROVIDER_NAME]
            .as_table()
            .unwrap();
        assert!(provider.get("requires_openai_auth").is_none());
        assert!(provider.get("env_key").is_none());
        let auth = provider["auth"].as_table().unwrap();
        assert_eq!(auth["command"].as_str(), Some("/usr/bin/security"));
        let args = auth["args"].as_array().unwrap();
        assert_eq!(
            args.iter()
                .filter_map(|value| value.as_str())
                .collect::<Vec<_>>(),
            vec![
                "find-generic-password",
                "-w",
                "-s",
                "cc.luoxueapi.desktop",
                "-a",
                "local-token"
            ]
        );
        assert_eq!(auth["timeout_ms"].as_integer(), Some(5_000));
        assert_eq!(auth["refresh_interval_ms"].as_integer(), Some(0));
    }

    #[test]
    fn managed_auth_does_not_overwrite_openai_api_key() {
        let original = br#"{"OPENAI_API_KEY":"old","theme":"dark"}"#;
        let applied = build_managed_auth(original, "local").unwrap();
        assert_eq!(applied, original);
        let current = br#"{"OPENAI_API_KEY":"changed","theme":"light","new":true}"#;
        assert_eq!(
            merge_managed_auth(original, &applied, current).unwrap(),
            current
        );
    }

    #[test]
    fn auth_restore_preserves_new_fields_for_legacy_managed_payload() {
        let original = br#"{"OPENAI_API_KEY":"old","theme":"dark"}"#;
        let applied = br#"{"OPENAI_API_KEY":"local","theme":"dark"}"#;
        let current = br#"{"OPENAI_API_KEY":"local","theme":"light","new":true}"#;
        let restored: serde_json::Value =
            serde_json::from_slice(&merge_managed_auth(original, applied, current).unwrap())
                .unwrap();
        assert_eq!(restored["OPENAI_API_KEY"], "old");
        assert_eq!(restored["theme"], "light");
        assert_eq!(restored["new"], true);
    }

    #[test]
    fn prepared_recovery_accepts_files_that_are_still_original() {
        let original_config = b"model = \"old\"\nuser_setting = true\n";
        let applied_config = build_managed_config(original_config, 11430, "gpt-5.4").unwrap();
        let original_auth = br#"{"OPENAI_API_KEY":"old","theme":"dark"}"#;
        let applied_auth = build_managed_auth(original_auth, "local").unwrap();

        let restored_config =
            merge_managed_config(original_config, &applied_config, original_config).unwrap();
        let restored_auth =
            merge_managed_auth(original_auth, &applied_auth, original_auth).unwrap();
        assert_eq!(restored_config, original_config);
        assert_eq!(restored_auth, original_auth);
        assert_eq!(
            serde_json::from_slice::<serde_json::Value>(&restored_auth).unwrap()["OPENAI_API_KEY"],
            "old"
        );
    }

    #[test]
    fn auth_written_recovery_handles_mixed_original_and_applied_files() {
        let original_config = b"model = \"old\"\n";
        let applied_config = build_managed_config(original_config, 11430, "gpt-5.4").unwrap();
        let original_auth = br#"{"OPENAI_API_KEY":"old"}"#;
        let applied_auth = build_managed_auth(original_auth, "local").unwrap();

        let restored_config =
            merge_managed_config(original_config, &applied_config, original_config).unwrap();
        let restored_auth =
            merge_managed_auth(original_auth, &applied_auth, &applied_auth).unwrap();
        assert_eq!(restored_config, original_config);
        assert_eq!(
            serde_json::from_slice::<serde_json::Value>(&restored_auth).unwrap()["OPENAI_API_KEY"],
            "old"
        );
    }

    #[test]
    fn restoring_recovery_is_idempotent_after_config_was_already_restored() {
        let original_config = b"model = \"old\"\n";
        let applied_config = build_managed_config(original_config, 11430, "gpt-5.4").unwrap();
        let original_auth = br#"{"OPENAI_API_KEY":"old"}"#;
        let applied_auth = build_managed_auth(original_auth, "local").unwrap();

        let restored_config =
            merge_managed_config(original_config, &applied_config, original_config).unwrap();
        let restored_auth =
            merge_managed_auth(original_auth, &applied_auth, &applied_auth).unwrap();
        assert_eq!(restored_config, original_config);
        assert_eq!(
            serde_json::from_slice::<serde_json::Value>(&restored_auth).unwrap()["OPENAI_API_KEY"],
            "old"
        );
    }
}
