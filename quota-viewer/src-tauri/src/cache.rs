use std::{
    fs,
    path::{Path, PathBuf},
};

use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
use chrono::{DateTime, Utc};
use rand::{rngs::OsRng, RngCore};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use zeroize::Zeroizing;

use crate::{
    error::{AppError, AppResult},
    file_replace::replace_file,
    models::{sanitize_quota_overview, CachedOverview},
};

#[derive(Clone)]
pub struct OverviewCache {
    path: PathBuf,
    encryption_key: Zeroizing<Vec<u8>>,
}

#[derive(Deserialize, Serialize)]
struct EncryptedCacheEnvelope {
    version: u8,
    nonce: String,
    ciphertext: String,
}

impl OverviewCache {
    pub fn new(path: PathBuf, encryption_key: Zeroizing<Vec<u8>>) -> AppResult<Self> {
        if encryption_key.len() != 32 {
            return Err(AppError::Message(
                "snapshot encryption key has an invalid length".into(),
            ));
        }
        Ok(Self {
            path,
            encryption_key,
        })
    }

    pub fn load(
        &self,
        expected_cache_id: &str,
        expected_display_timezone: &str,
    ) -> AppResult<Option<CachedOverview>> {
        match fs::read(&self.path) {
            Ok(bytes) => {
                let envelope: EncryptedCacheEnvelope = serde_json::from_slice(&bytes)?;
                if envelope.version != 1 {
                    return Err(AppError::Message(
                        "snapshot cache version is unsupported".into(),
                    ));
                }
                let nonce = URL_SAFE_NO_PAD
                    .decode(envelope.nonce)
                    .map_err(|_| AppError::Message("snapshot cache nonce is invalid".into()))?;
                if nonce.len() != 12 {
                    return Err(AppError::Message(
                        "snapshot cache nonce has an invalid length".into(),
                    ));
                }
                let ciphertext = URL_SAFE_NO_PAD
                    .decode(envelope.ciphertext)
                    .map_err(|_| AppError::Message("snapshot cache payload is invalid".into()))?;
                let cipher = Aes256Gcm::new_from_slice(self.encryption_key.as_slice())
                    .map_err(|_| AppError::Message("snapshot encryption key is invalid".into()))?;
                let plaintext = cipher
                    .decrypt(Nonce::from_slice(&nonce), ciphertext.as_ref())
                    .map_err(|_| {
                        AppError::Message("snapshot cache could not be decrypted".into())
                    })?;
                let mut cache: CachedOverview = serde_json::from_slice(&plaintext)?;
                if cache.cache_id != expected_cache_id
                    || cache.display_timezone != expected_display_timezone
                {
                    return Ok(None);
                }
                // Re-project decrypted snapshots as well. This protects a
                // downgraded client from exposing fields written by a newer
                // version and cleans any legacy untyped cache before it can
                // cross into the WebView.
                cache.overview = sanitize_quota_overview(cache.overview)?;
                Ok(Some(cache))
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(None),
            Err(error) => Err(error.into()),
        }
    }

    pub fn save(
        &self,
        overview: Value,
        cache_id: &str,
        display_timezone: &str,
    ) -> AppResult<CachedOverview> {
        let overview = sanitize_quota_overview(overview)?;
        let cache = CachedOverview {
            cache_id: cache_id.to_string(),
            display_timezone: display_timezone.to_string(),
            overview,
            fetched_at: Utc::now(),
        };
        if let Some(parent) = self.path.parent() {
            fs::create_dir_all(parent)?;
        }
        let plaintext = serde_json::to_vec(&cache)?;
        let mut nonce = [0_u8; 12];
        OsRng.fill_bytes(&mut nonce);
        let cipher = Aes256Gcm::new_from_slice(self.encryption_key.as_slice())
            .map_err(|_| AppError::Message("snapshot encryption key is invalid".into()))?;
        let ciphertext = cipher
            .encrypt(Nonce::from_slice(&nonce), plaintext.as_ref())
            .map_err(|_| AppError::Message("snapshot cache could not be encrypted".into()))?;
        let envelope = EncryptedCacheEnvelope {
            version: 1,
            nonce: URL_SAFE_NO_PAD.encode(nonce),
            ciphertext: URL_SAFE_NO_PAD.encode(ciphertext),
        };
        let temporary = self.path.with_extension("json.tmp");
        let bytes = serde_json::to_vec(&envelope)?;
        fs::write(&temporary, bytes)?;
        restrict_file_permissions(&temporary)?;
        replace_file(&temporary, &self.path)?;
        Ok(cache)
    }

    pub fn clear(&self) -> AppResult<()> {
        match fs::remove_file(&self.path) {
            Ok(()) => Ok(()),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Err(error) => Err(error.into()),
        }
    }
}

pub fn cache_is_fresh(cache: &CachedOverview, now: DateTime<Utc>) -> bool {
    cache
        .overview
        .get("fresh_until")
        .and_then(Value::as_str)
        .and_then(|value| DateTime::parse_from_rfc3339(value).ok())
        .is_some_and(|value| value.with_timezone(&Utc) > now)
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

    fn quota_overview(fresh_until: &str) -> Value {
        serde_json::json!({
            "schema_version": 1,
            "request_id": "qov-cache-test",
            "generated_at": "2026-07-30T00:00:01Z",
            "as_of": "2026-07-30T00:00:00Z",
            "fresh_until": fresh_until,
            "display_timezone": "Asia/Shanghai",
            "freshness": "fresh",
            "coverage": {
                "included": ["wallet", "account_spend_today", "account_spend_month_to_date"],
                "excluded": []
            },
            "account": {
                "display_label": "p***@example.com",
                "data_scope": "all_enabled_api_keys",
                "quota_state": "all_resources_available",
                "can_make_request": null,
                "usable_group_count": 1,
                "blocked_group_count": 0,
                "unknown_group_count": 0,
                "primary_issue": null
            },
            "wallet": {
                "unit": "snow_credit",
                "state": "available",
                "available": "128.6400000000",
                "reserved": "0.0000000000",
                "today_spend": "1.1600000000",
                "month_spend": "10.7400000000",
                "balance_billed_key_count": 1,
                "ledger_secret": "must-not-enter-cache"
            },
            "billing_groups": [{
                "id": "grp_1",
                "display_name": "标准计费",
                "billing_mode": "balance",
                "state": "usable",
                "reason_code": null,
                "recommended_action": "none",
                "fallback_policy": "none",
                "resource_ref": {"kind": "wallet", "id": null},
                "keys": [{
                    "id": "key_1",
                    "name": "Codex",
                    "masked_key": "sk-••••42FD",
                    "state": "usable",
                    "full_api_key": "sk-cache-full-api-key"
                }]
            }],
            "subscriptions": [],
            "actions": {
                "recharge_url": "https://luoxueapi.cc/purchase",
                "manage_keys_url": "https://luoxueapi.cc/keys",
                "manage_subscriptions_url": "https://luoxueapi.cc/subscriptions"
            },
            "warnings": [],
            "native_debug_secret": "must-not-enter-cache"
        })
    }

    #[test]
    fn cache_round_trip_preserves_decimal_strings_and_freshness() {
        let directory = tempfile::tempdir().unwrap();
        let store = OverviewCache::new(
            directory.path().join("overview.json"),
            Zeroizing::new(vec![7_u8; 32]),
        )
        .unwrap();
        let fresh_until = (Utc::now() + chrono::Duration::minutes(5)).to_rfc3339();
        let cached = store
            .save(quota_overview(&fresh_until), "session-a", "Asia/Shanghai")
            .unwrap();

        let loaded = store.load("session-a", "Asia/Shanghai").unwrap().unwrap();
        assert_eq!(loaded.overview["wallet"]["available"], "128.6400000000");
        assert_eq!(loaded.overview["wallet"]["today_spend"], "1.1600000000");
        assert_eq!(loaded.overview["wallet"]["month_spend"], "10.7400000000");
        let serialized = serde_json::to_string(&loaded.overview).unwrap();
        assert!(!serialized.contains("sk-cache-full-api-key"));
        assert!(!serialized.contains("must-not-enter-cache"));
        assert!(!serialized.contains("full_api_key"));
        assert!(!serialized.contains("ledger_secret"));
        assert!(!serialized.contains("native_debug_secret"));
        assert!(store.load("session-b", "Asia/Shanghai").unwrap().is_none());
        assert!(store.load("session-a", "UTC").unwrap().is_none());
        assert!(cache_is_fresh(&cached, Utc::now()));
        let disk = fs::read_to_string(directory.path().join("overview.json")).unwrap();
        assert!(!disk.contains("128.6400000000"));

        store.clear().unwrap();
        assert!(store.load("session-a", "Asia/Shanghai").unwrap().is_none());
    }

    #[test]
    fn repeated_saves_replace_the_existing_cache_file() {
        let directory = tempfile::tempdir().unwrap();
        let store = OverviewCache::new(
            directory.path().join("overview.json"),
            Zeroizing::new(vec![7_u8; 32]),
        )
        .unwrap();
        let fresh_until = (Utc::now() + chrono::Duration::minutes(5)).to_rfc3339();
        store
            .save(quota_overview(&fresh_until), "session-a", "Asia/Shanghai")
            .unwrap();

        let mut replacement = quota_overview(&fresh_until);
        replacement["wallet"]["available"] = serde_json::json!("64.3200000000");
        store
            .save(replacement, "session-a", "Asia/Shanghai")
            .unwrap();

        let loaded = store.load("session-a", "Asia/Shanghai").unwrap().unwrap();
        assert_eq!(loaded.overview["wallet"]["available"], "64.3200000000");
    }

    #[test]
    fn legacy_cache_identity_fields_default_to_empty_and_never_match() {
        let directory = tempfile::tempdir().unwrap();
        let store = OverviewCache::new(
            directory.path().join("overview.json"),
            Zeroizing::new(vec![9_u8; 32]),
        )
        .unwrap();
        let legacy = serde_json::json!({
            "overview": {"freshness": "fresh"},
            "fetched_at": Utc::now()
        });
        let nonce = [3_u8; 12];
        let cipher = Aes256Gcm::new_from_slice(store.encryption_key.as_slice()).unwrap();
        let ciphertext = cipher
            .encrypt(
                Nonce::from_slice(&nonce),
                serde_json::to_vec(&legacy).unwrap().as_ref(),
            )
            .unwrap();
        let envelope = EncryptedCacheEnvelope {
            version: 1,
            nonce: URL_SAFE_NO_PAD.encode(nonce),
            ciphertext: URL_SAFE_NO_PAD.encode(ciphertext),
        };
        fs::write(&store.path, serde_json::to_vec(&envelope).unwrap()).unwrap();

        assert!(store.load("session-a", "Asia/Shanghai").unwrap().is_none());
    }
}
