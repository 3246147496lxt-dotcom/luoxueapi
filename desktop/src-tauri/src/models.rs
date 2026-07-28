use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RouteOption {
    #[serde(alias = "group_id")]
    pub group_id: i64,
    pub name: String,
    #[serde(default = "default_rate_multiplier", alias = "rate_multiplier")]
    pub rate_multiplier: f64,
    pub models: Vec<String>,
}

fn default_rate_multiplier() -> f64 {
    1.0
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CodexInstallation {
    pub kind: String,
    pub installed: bool,
    pub version: Option<String>,
    pub path: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TodaySummary {
    pub requests: i64,
    pub tokens: i64,
    pub cost: f64,
    pub balance: f64,
    pub average_first_token_ms: Option<i64>,
}

impl Default for TodaySummary {
    fn default() -> Self {
        Self {
            requests: 0,
            tokens: 0,
            cost: 0.0,
            balance: 0.0,
            average_first_token_ms: None,
        }
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DesktopSettings {
    pub launch_at_login: bool,
    pub notifications: bool,
    pub retention_days: u32,
    pub theme: String,
    pub locale: String,
}

impl Default for DesktopSettings {
    fn default() -> Self {
        Self {
            launch_at_login: true,
            notifications: true,
            retention_days: 7,
            theme: "system".into(),
            locale: "zh-CN".into(),
        }
    }
}

#[derive(Clone, Debug, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct PersistedState {
    #[serde(flatten)]
    pub settings: DesktopSettings,
    pub gateway_port: Option<u16>,
    pub selected_group_id: Option<i64>,
    pub selected_model: Option<String>,
    #[serde(default)]
    pub managed_group_ids: Vec<i64>,
    #[serde(default)]
    pub account_email: Option<String>,
    #[serde(default)]
    pub routes: Vec<RouteOption>,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DesktopSnapshot {
    pub paired: bool,
    pub account_email: Option<String>,
    pub device_name: String,
    pub gateway_status: String,
    pub gateway_port: Option<u16>,
    pub takeover_enabled: bool,
    pub config_status: String,
    pub config_message: Option<String>,
    pub selected_group_id: Option<i64>,
    pub selected_model: Option<String>,
    pub routes: Vec<RouteOption>,
    pub installations: Vec<CodexInstallation>,
    pub today: TodaySummary,
    pub settings: DesktopSettings,
    pub update: UpdateState,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateState {
    pub state: String,
    pub current_version: String,
    pub version: Option<String>,
}

impl UpdateState {
    pub fn current() -> Self {
        Self {
            state: "current".into(),
            current_version: env!("CARGO_PKG_VERSION").into(),
            version: None,
        }
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RequestMetadata {
    pub id: String,
    pub occurred_at: String,
    pub model: String,
    pub status_code: u16,
    pub input_tokens: i64,
    pub output_tokens: i64,
    pub duration_ms: i64,
    pub first_token_ms: Option<i64>,
    pub request_id: String,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PairingState {
    pub status: String,
    pub user_code: Option<String>,
    pub verification_uri: Option<String>,
    pub expires_at: Option<String>,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticGateway {
    pub status: String,
    pub port: Option<u16>,
    pub takeover_enabled: bool,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticInstallation {
    pub kind: String,
    pub installed: bool,
    pub version: Option<String>,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticCodex {
    pub config_status: String,
    pub installations: Vec<DiagnosticInstallation>,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticRoute {
    pub group_id: Option<i64>,
    pub model: Option<String>,
    pub available_route_count: usize,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticRecentRequest {
    pub occurred_at: String,
    pub model: String,
    pub status_code: u16,
    pub duration_ms: i64,
    pub first_token_ms: Option<i64>,
    pub request_id: String,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticRequests {
    pub sample_count: usize,
    pub success_count: usize,
    pub error_count: usize,
    pub average_duration_ms: Option<i64>,
    pub average_first_token_ms: Option<i64>,
    pub recent: Vec<DiagnosticRecentRequest>,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticSummary {
    pub app_version: String,
    pub platform: String,
    pub architecture: String,
    pub os_version: String,
    pub gateway: DiagnosticGateway,
    pub codex: DiagnosticCodex,
    pub route: DiagnosticRoute,
    pub requests: DiagnosticRequests,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticReceipt {
    pub id: String,
    #[serde(alias = "created_at")]
    pub created_at: String,
    #[serde(alias = "expires_at")]
    pub expires_at: String,
}

impl PairingState {
    pub fn waiting(user_code: String, verification_uri: String, expires_at: String) -> Self {
        Self {
            status: "waiting".into(),
            user_code: Some(user_code),
            verification_uri: Some(verification_uri),
            expires_at: Some(expires_at),
        }
    }
}
