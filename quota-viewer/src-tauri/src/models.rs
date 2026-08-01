use chrono::{DateTime, Utc};
use rand::{rngs::OsRng, RngCore};
use serde::{de::Error as _, Deserialize, Deserializer, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use uuid::Uuid;
use zeroize::Zeroizing;

pub const REFRESH_PROTOCOL_CANDIDATE_V1: &str = "candidate-v1";
pub const REFRESH_ROTATION_COMMITTED: &str = "committed";
pub const REFRESH_ROTATION_RECOVERED: &str = "recovered";
const PENDING_REFRESH_ROTATION_VERSION: u8 = 1;
const REFRESH_TOKEN_PREFIX: &str = "qvrt_";

#[derive(Clone)]
pub struct PairingContext {
    pub device_code: Zeroizing<String>,
    pub verifier: Zeroizing<String>,
    pub user_code: String,
    pub verification_uri: String,
    pub expires_at: DateTime<Utc>,
    pub interval: u64,
}

pub enum PairingExchange {
    Pending,
    Expired,
    Approved(TokenBundle),
}

#[derive(Deserialize)]
pub struct TokenBundle {
    pub access_token: Zeroizing<String>,
    pub refresh_token: Zeroizing<String>,
    pub expires_in: i64,
}

#[derive(Deserialize)]
pub struct RefreshTokenBundle {
    pub access_token: Zeroizing<String>,
    pub refresh_token: Zeroizing<String>,
    pub expires_in: i64,
    pub refresh_protocol: String,
    pub rotation_id: String,
    pub rotation_result: String,
}

#[derive(Clone, Deserialize, Serialize)]
pub struct PendingRefreshRotation {
    version: u8,
    pub rotation_id: String,
    pub candidate_refresh_token: Zeroizing<String>,
    pub base_token_sha256: String,
}

impl PendingRefreshRotation {
    pub fn generate(base_refresh_token: &str) -> Self {
        let candidate_refresh_token = loop {
            let mut random = Zeroizing::new([0_u8; 32]);
            OsRng.fill_bytes(&mut *random);
            let candidate = format!("{REFRESH_TOKEN_PREFIX}{}", lowercase_hex(&*random));
            if candidate != base_refresh_token {
                break candidate;
            }
        };
        Self {
            version: PENDING_REFRESH_ROTATION_VERSION,
            rotation_id: Uuid::new_v4().hyphenated().to_string(),
            candidate_refresh_token: Zeroizing::new(candidate_refresh_token),
            base_token_sha256: sha256_hex(base_refresh_token.as_bytes()),
        }
    }

    pub fn is_valid(&self) -> bool {
        self.version == PENDING_REFRESH_ROTATION_VERSION
            && Uuid::parse_str(&self.rotation_id).is_ok_and(|value| {
                value.get_version_num() == 4 && value.to_string() == self.rotation_id
            })
            && valid_candidate_refresh_token(self.candidate_refresh_token.as_str())
            && valid_lowercase_sha256(&self.base_token_sha256)
    }

    pub fn matches_base_token(&self, refresh_token: &str) -> bool {
        self.base_token_sha256 == sha256_hex(refresh_token.as_bytes())
    }

    pub fn matches_candidate_token(&self, refresh_token: &str) -> bool {
        self.candidate_refresh_token.as_str() == refresh_token
    }

    pub fn validates_response(&self, bundle: &RefreshTokenBundle) -> bool {
        bundle.refresh_protocol == REFRESH_PROTOCOL_CANDIDATE_V1
            && bundle.rotation_id == self.rotation_id
            && matches!(
                bundle.rotation_result.as_str(),
                REFRESH_ROTATION_COMMITTED | REFRESH_ROTATION_RECOVERED
            )
            && bundle.refresh_token.as_str() == self.candidate_refresh_token.as_str()
    }
}

#[derive(Clone)]
pub struct AccessSession {
    pub token: Zeroizing<String>,
    pub expires_at: DateTime<Utc>,
}

impl AccessSession {
    pub fn from_bundle(bundle: &TokenBundle) -> Self {
        Self::from_token(bundle.access_token.clone(), bundle.expires_in)
    }

    pub fn from_refresh_bundle(bundle: &RefreshTokenBundle) -> Self {
        Self::from_token(bundle.access_token.clone(), bundle.expires_in)
    }

    fn from_token(token: Zeroizing<String>, expires_in: i64) -> Self {
        let usable_seconds = expires_in.saturating_sub(30).max(1);
        Self {
            token,
            expires_at: Utc::now() + chrono::Duration::seconds(usable_seconds),
        }
    }

    pub fn is_usable(&self) -> bool {
        self.expires_at > Utc::now()
    }
}

fn valid_candidate_refresh_token(token: &str) -> bool {
    token.len() == REFRESH_TOKEN_PREFIX.len() + (32 * 2)
        && token.starts_with(REFRESH_TOKEN_PREFIX)
        && token[REFRESH_TOKEN_PREFIX.len()..]
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
}

fn valid_lowercase_sha256(value: &str) -> bool {
    value.len() == 64
        && value
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
}

fn sha256_hex(value: &[u8]) -> String {
    lowercase_hex(&Sha256::digest(value))
}

fn lowercase_hex(value: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut encoded = String::with_capacity(value.len() * 2);
    for byte in value {
        encoded.push(HEX[(byte >> 4) as usize] as char);
        encoded.push(HEX[(byte & 0x0f) as usize] as char);
    }
    encoded
}

#[derive(Clone, Deserialize)]
pub struct PairingChallenge {
    pub device_code: Zeroizing<String>,
    pub user_code: String,
    pub verification_uri: String,
    #[serde(default)]
    pub verification_uri_complete: Option<String>,
    pub expires_in: i64,
    #[serde(default)]
    pub interval: u64,
    #[serde(default)]
    pub expires_at: Option<DateTime<Utc>>,
}

#[derive(Clone, Debug, Serialize)]
pub struct PairingState {
    pub status: String,
    pub user_code: Option<String>,
    pub verification_uri: Option<String>,
    pub expires_at: Option<String>,
    pub interval: u64,
}

impl PairingState {
    pub fn waiting(context: &PairingContext) -> Self {
        Self {
            status: "waiting".into(),
            user_code: Some(context.user_code.clone()),
            verification_uri: Some(context.verification_uri.clone()),
            expires_at: Some(context.expires_at.to_rfc3339()),
            interval: context.interval,
        }
    }

    pub fn terminal(status: &str) -> Self {
        Self {
            status: status.into(),
            user_code: None,
            verification_uri: None,
            expires_at: None,
            interval: 5,
        }
    }
}

// This is the only quota-overview shape allowed to cross the native/WebView
// boundary or enter the encrypted snapshot cache. Serde deliberately ignores
// unknown response fields while deserializing; serializing this typed
// projection again removes them, including unknown fields nested in arrays.
//
// Keep this projection explicit. Do not replace it with `flatten` or an
// untyped `Value`, otherwise a future backend field could accidentally expose
// credentials or internal diagnostics to the WebView/cache.
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct SafeQuotaOverview {
    #[serde(deserialize_with = "deserialize_schema_v1")]
    schema_version: u8,
    request_id: String,
    generated_at: String,
    as_of: String,
    fresh_until: String,
    display_timezone: String,
    freshness: String,
    coverage: SafeQuotaOverviewCoverage,
    account: SafeQuotaOverviewAccount,
    wallet: SafeQuotaOverviewWallet,
    #[serde(default)]
    billing_groups: Vec<SafeQuotaOverviewBillingGroup>,
    #[serde(default)]
    subscriptions: Vec<SafeQuotaOverviewSubscription>,
    actions: SafeQuotaOverviewActions,
    #[serde(default)]
    warnings: Vec<String>,
}

impl SafeQuotaOverview {
    pub fn into_json_value(self) -> serde_json::Result<Value> {
        serde_json::to_value(self)
    }
}

pub fn sanitize_quota_overview(value: Value) -> serde_json::Result<Value> {
    serde_json::from_value::<SafeQuotaOverview>(value)?.into_json_value()
}

fn deserialize_schema_v1<'de, D>(deserializer: D) -> Result<u8, D::Error>
where
    D: Deserializer<'de>,
{
    let version = u8::deserialize(deserializer)?;
    if version != 1 {
        return Err(D::Error::custom(
            "unsupported quota overview schema version",
        ));
    }
    Ok(version)
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewCoverage {
    #[serde(default)]
    included: Vec<String>,
    #[serde(default)]
    excluded: Vec<String>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewAccount {
    display_label: String,
    data_scope: String,
    quota_state: String,
    can_make_request: Option<bool>,
    usable_group_count: i64,
    blocked_group_count: i64,
    unknown_group_count: i64,
    primary_issue: Option<SafeQuotaOverviewIssue>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewIssue {
    scope_type: String,
    scope_id: String,
    reason_code: String,
    recommended_action: String,
    recovers_at: Option<String>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewWallet {
    unit: String,
    state: String,
    available: String,
    reserved: String,
    // These remain independent account spend metrics. They must never be
    // synthesized from token/request/subscription data.
    today_spend: Option<String>,
    month_spend: Option<String>,
    balance_billed_key_count: i64,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewBillingGroup {
    id: String,
    display_name: String,
    billing_mode: String,
    state: String,
    reason_code: Option<String>,
    recommended_action: String,
    fallback_policy: String,
    resource_ref: SafeQuotaOverviewResourceRef,
    #[serde(default)]
    keys: Vec<SafeQuotaOverviewKey>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewResourceRef {
    kind: String,
    id: Option<String>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewKey {
    id: String,
    name: String,
    masked_key: String,
    state: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewSubscription {
    id: String,
    group_id: String,
    name: String,
    status: String,
    starts_at: String,
    expires_at: String,
    weekly_window: SafeQuotaOverviewWeeklyWindow,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    monthly_window: Option<SafeQuotaOverviewWeeklyWindow>,
    period_usage: Option<SafeQuotaOverviewPeriodUsage>,
    next_event: Option<SafeQuotaOverviewNextEvent>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewWeeklyWindow {
    kind: String,
    state: String,
    anchor_at: String,
    period_start: Option<String>,
    period_end: Option<String>,
    resets_at: Option<String>,
    limit: Option<String>,
    used: Option<String>,
    remaining: Option<String>,
    used_percent: Option<f64>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewPeriodUsage {
    state: String,
    observed_until: Option<String>,
    bucket_kind: String,
    total_requests: Option<i64>,
    total_tokens: Option<i64>,
    points: Option<Vec<SafeQuotaOverviewPeriodUsagePoint>>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewPeriodUsagePoint {
    index: i64,
    start_at: String,
    end_at: String,
    state: String,
    requests: Option<i64>,
    cache_hit_tokens: Option<i64>,
    cache_miss_tokens: Option<i64>,
    output_tokens: Option<i64>,
    total_tokens: Option<i64>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewNextEvent {
    kind: String,
    at: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct SafeQuotaOverviewActions {
    recharge_url: String,
    manage_keys_url: String,
    manage_subscriptions_url: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct CachedOverview {
    #[serde(default)]
    pub cache_id: String,
    #[serde(default)]
    pub display_timezone: String,
    pub overview: Value,
    pub fetched_at: DateTime<Utc>,
}

#[derive(Clone, Debug, Serialize)]
pub struct ViewerSnapshot {
    pub status: String,
    pub overview: Option<Value>,
    pub source: Option<String>,
    pub fetched_at: Option<String>,
    pub error_code: Option<String>,
    pub message: Option<String>,
}

impl ViewerSnapshot {
    pub fn disconnected() -> Self {
        Self {
            status: "disconnected".into(),
            overview: None,
            source: None,
            fetched_at: None,
            error_code: None,
            message: None,
        }
    }

    pub fn without_data(status: &str, error_code: Option<String>, message: Option<String>) -> Self {
        Self {
            status: status.into(),
            overview: None,
            source: None,
            fetched_at: None,
            error_code,
            message,
        }
    }

    pub fn from_cache(
        status: &str,
        cache: CachedOverview,
        source: &str,
        error_code: Option<String>,
        message: Option<String>,
    ) -> Self {
        Self {
            status: status.into(),
            overview: Some(cache.overview),
            source: Some(source.into()),
            fetched_at: Some(cache.fetched_at.to_rfc3339()),
            error_code,
            message,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn overview_with_unknown_secrets() -> Value {
        serde_json::json!({
            "schema_version": 1,
            "request_id": "qov-test",
            "generated_at": "2026-07-30T00:00:01Z",
            "as_of": "2026-07-30T00:00:00Z",
            "fresh_until": "2026-07-30T00:05:00Z",
            "display_timezone": "Asia/Shanghai",
            "freshness": "fresh",
            "coverage": {
                "included": ["wallet", "account_spend_today", "account_spend_month_to_date"],
                "excluded": ["routing"],
                "internal_query": "SELECT secret FROM credentials"
            },
            "account": {
                "display_label": "p***@example.com",
                "data_scope": "all_enabled_api_keys",
                "quota_state": "all_resources_available",
                "can_make_request": null,
                "usable_group_count": 1,
                "blocked_group_count": 0,
                "unknown_group_count": 0,
                "primary_issue": {
                    "scope_type": "billing_group",
                    "scope_id": "grp_1",
                    "reason_code": "none",
                    "recommended_action": "none",
                    "recovers_at": null,
                    "secret": "issue-secret"
                },
                "email": "private@example.com"
            },
            "wallet": {
                "unit": "snow_credit",
                "state": "available",
                "available": "128.6400000000",
                "reserved": "0.0000000000",
                "today_spend": "1.1600000000",
                "month_spend": "10.7400000000",
                "balance_billed_key_count": 1,
                "internal_ledger_id": "wallet-secret"
            },
            "billing_groups": [{
                "id": "grp_1",
                "display_name": "Pro 会员",
                "billing_mode": "subscription",
                "state": "usable",
                "reason_code": null,
                "recommended_action": "none",
                "fallback_policy": "none",
                "resource_ref": {
                    "kind": "subscription",
                    "id": "sub_1",
                    "database_id": 42
                },
                "keys": [{
                    "id": "key_1",
                    "name": "Codex",
                    "masked_key": "sk-••••42FD",
                    "state": "usable",
                    "raw_key": "sk-full-api-key-must-not-cross-boundary"
                }],
                "provider_secret": "billing-group-secret"
            }],
            "subscriptions": [{
                "id": "sub_1",
                "group_id": "grp_1",
                "name": "Pro 会员",
                "status": "active",
                "starts_at": "2026-07-25T09:30:00Z",
                "expires_at": "2026-08-25T09:30:00Z",
                "weekly_window": {
                    "kind": "7d_from_subscription_start",
                    "state": "active",
                    "anchor_at": "2026-07-25T09:30:00Z",
                    "period_start": "2026-07-25T09:30:00Z",
                    "period_end": "2026-08-01T09:30:00Z",
                    "resets_at": "2026-08-01T09:30:00Z",
                    "limit": "200.0000000000",
                    "used": "136.2000000000",
                    "remaining": "63.8000000000",
                    "used_percent": 68,
                    "admission_cache_key": "subscription-secret"
                },
                "monthly_window": {
                    "kind": "30d_from_subscription_start",
                    "state": "active",
                    "anchor_at": "2026-07-25T09:30:00Z",
                    "period_start": "2026-07-25T09:30:00Z",
                    "period_end": "2026-08-24T09:30:00Z",
                    "resets_at": "2026-08-24T09:30:00Z",
                    "limit": "800.0000000000",
                    "used": "208.0000000000",
                    "remaining": "592.0000000000",
                    "used_percent": 26,
                    "internal_monthly_ledger": "monthly-secret"
                },
                "period_usage": {
                    "state": "available",
                    "observed_until": "2026-07-30T00:00:00Z",
                    "bucket_kind": "anchored_24h",
                    "total_requests": 606,
                    "total_tokens": 1906000,
                    "points": [{
                        "index": 1,
                        "start_at": "2026-07-25T09:30:00Z",
                        "end_at": "2026-07-26T09:30:00Z",
                        "state": "complete",
                        "requests": 58,
                        "cache_hit_tokens": 92000,
                        "cache_miss_tokens": 41000,
                        "output_tokens": 49000,
                        "total_tokens": 182000,
                        "request_body": "point-secret"
                    }],
                    "usage_log_rows": ["period-secret"]
                },
                "next_event": {
                    "kind": "reset",
                    "at": "2026-08-01T09:30:00Z",
                    "scheduler_job": "next-event-secret"
                },
                "upstream_access_token": "subscription-secret"
            }],
            "actions": {
                "recharge_url": "https://luoxueapi.cc/purchase",
                "manage_keys_url": "https://luoxueapi.cc/keys",
                "manage_subscriptions_url": "https://luoxueapi.cc/subscriptions",
                "admin_url": "https://internal.example/admin"
            },
            "warnings": [],
            "debug": {
                "authorization": "Bearer top-level-secret"
            }
        })
    }

    #[test]
    fn quota_overview_projection_strips_unknown_and_nested_secret_fields() {
        let sanitized = sanitize_quota_overview(overview_with_unknown_secrets()).unwrap();
        let serialized = serde_json::to_string(&sanitized).unwrap();

        assert_eq!(sanitized["wallet"]["available"], "128.6400000000");
        assert_eq!(sanitized["wallet"]["today_spend"], "1.1600000000");
        assert_eq!(sanitized["wallet"]["month_spend"], "10.7400000000");
        assert_eq!(
            sanitized["billing_groups"][0]["keys"][0]["masked_key"],
            "sk-••••42FD"
        );
        assert_eq!(
            sanitized["subscriptions"][0]["period_usage"]["total_tokens"],
            1_906_000
        );
        assert_eq!(
            sanitized["subscriptions"][0]["monthly_window"]["used_percent"].as_f64(),
            Some(26.0)
        );
        assert_eq!(
            sanitized["subscriptions"][0]["monthly_window"]["kind"],
            "30d_from_subscription_start"
        );
        assert_eq!(
            sanitized["subscriptions"][0]["monthly_window"]["period_end"],
            "2026-08-24T09:30:00Z"
        );
        assert_eq!(
            sanitized["actions"]["manage_keys_url"],
            "https://luoxueapi.cc/keys"
        );

        for forbidden in [
            "sk-full-api-key-must-not-cross-boundary",
            "Bearer top-level-secret",
            "private@example.com",
            "issue-secret",
            "wallet-secret",
            "billing-group-secret",
            "subscription-secret",
            "monthly-secret",
            "internal_monthly_ledger",
            "point-secret",
            "period-secret",
            "next-event-secret",
            "internal_query",
            "internal_ledger_id",
            "raw_key",
            "provider_secret",
            "upstream_access_token",
            "admin_url",
            "debug",
        ] {
            assert!(
                !serialized.contains(forbidden),
                "unsafe value or field survived projection: {forbidden}"
            );
        }
    }

    #[test]
    fn quota_overview_projection_rejects_unknown_schema_versions() {
        let mut overview = overview_with_unknown_secrets();
        overview["schema_version"] = Value::from(2);

        assert!(sanitize_quota_overview(overview).is_err());
    }

    #[test]
    fn pending_refresh_rotation_uses_candidate_v1_shapes_and_survives_serialization() {
        let pending = PendingRefreshRotation::generate("qvrt_base");

        assert!(pending.is_valid());
        assert!(pending.matches_base_token("qvrt_base"));
        assert!(!pending.matches_base_token("qvrt_other"));
        assert_eq!(pending.candidate_refresh_token.len(), 69);
        assert!(pending.candidate_refresh_token.starts_with("qvrt_"));
        assert!(pending.candidate_refresh_token[5..]
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte)));

        let encoded = Zeroizing::new(serde_json::to_string(&pending).unwrap());
        let decoded: PendingRefreshRotation = serde_json::from_str(encoded.as_str()).unwrap();
        assert!(decoded.is_valid());
        assert_eq!(decoded.rotation_id, pending.rotation_id);
        assert_eq!(
            decoded.candidate_refresh_token.as_str(),
            pending.candidate_refresh_token.as_str()
        );
        assert_eq!(decoded.base_token_sha256, pending.base_token_sha256);
    }

    #[test]
    fn pending_refresh_rotation_requires_exact_response_echo() {
        let pending = PendingRefreshRotation::generate("qvrt_base");
        let mut response = RefreshTokenBundle {
            access_token: Zeroizing::new("access".into()),
            refresh_token: pending.candidate_refresh_token.clone(),
            expires_in: 300,
            refresh_protocol: REFRESH_PROTOCOL_CANDIDATE_V1.into(),
            rotation_id: pending.rotation_id.clone(),
            rotation_result: REFRESH_ROTATION_RECOVERED.into(),
        };
        assert!(pending.validates_response(&response));

        response.rotation_id = Uuid::new_v4().to_string();
        assert!(!pending.validates_response(&response));
        response.rotation_id = pending.rotation_id.clone();
        response.refresh_protocol = "legacy".into();
        assert!(!pending.validates_response(&response));
        response.refresh_protocol = REFRESH_PROTOCOL_CANDIDATE_V1.into();
        response.refresh_token = Zeroizing::new("qvrt_wrong".into());
        assert!(!pending.validates_response(&response));
    }
}
