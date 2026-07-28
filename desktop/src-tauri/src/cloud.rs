use std::time::Duration;

use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
use chrono::{DateTime, Utc};
use rand::{rngs::OsRng, RngCore};
use reqwest::{redirect::Policy, StatusCode};
use serde::{de::DeserializeOwned, Deserialize, Serialize};
use sha2::{Digest, Sha256};
use url::Url;
use zeroize::Zeroizing;

use crate::{
    error::{AppError, AppResult},
    models::{DiagnosticReceipt, DiagnosticSummary, RouteOption},
};

const DEFAULT_API_BASE_URL: &str = "https://luoxueapi.cc/";
const DEFAULT_WEB_BASE_URL: &str = "https://luoxueapi.cc/";
const CONTROL_REQUEST_TIMEOUT: Duration = Duration::from_secs(30);

pub fn desktop_architecture() -> &'static str {
    match std::env::consts::ARCH {
        "aarch64" => "arm64",
        "x86_64" => "x86_64",
        other => other,
    }
}

pub fn official_web_url(path: &str) -> AppResult<Url> {
    let base = option_env!("LUOXUE_WEB_BASE_URL")
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or(DEFAULT_WEB_BASE_URL);
    let base = Url::parse(base)
        .map_err(|error| AppError::Message(format!("invalid official web URL: {error}")))?;
    base.join(path.trim_start_matches('/'))
        .map_err(|error| AppError::Message(format!("invalid official web path: {error}")))
}

#[derive(Clone)]
pub struct CloudClient {
    client: reqwest::Client,
    base_url: Url,
}

#[derive(Clone, Debug)]
pub struct PairingContext {
    pub device_code: Zeroizing<String>,
    pub verifier: Zeroizing<String>,
    pub user_code: String,
    pub verification_uri: String,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug)]
pub enum PairingExchange {
    Pending,
    Expired,
    Approved(TokenBundle),
}

#[derive(Debug, Deserialize)]
pub struct TokenBundle {
    pub access_token: Zeroizing<String>,
    pub refresh_token: Zeroizing<String>,
    pub expires_in: i64,
    #[serde(default)]
    pub account_email: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ManagedKey {
    pub group_id: i64,
    #[serde(alias = "key", alias = "token")]
    pub api_key: Zeroizing<String>,
}

#[derive(Debug, Deserialize)]
pub struct DeviceStatus {
    pub status: String,
}

#[derive(Clone, Debug, Deserialize)]
pub struct TodayUsage {
    pub requests: i64,
    pub tokens: i64,
    pub cost: f64,
    pub balance: f64,
}

#[derive(Debug, Deserialize)]
struct CreatePairingResponse {
    device_code: Zeroizing<String>,
    user_code: String,
    #[serde(alias = "verification_url")]
    verification_uri: String,
    #[serde(default)]
    verification_uri_complete: Option<String>,
    expires_in: i64,
    #[serde(default)]
    expires_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Deserialize)]
struct ApiEnvelope<T> {
    code: i64,
    #[serde(default, rename = "message")]
    _message: String,
    #[serde(default, rename = "reason")]
    _reason: String,
    data: Option<T>,
}

#[derive(Debug, Deserialize)]
struct CloudError {
    #[serde(default)]
    code: i64,
    #[serde(default, rename = "message")]
    _message: String,
    #[serde(default)]
    reason: String,
}

impl CloudClient {
    pub fn new() -> AppResult<Self> {
        let configured = option_env!("LUOXUE_API_BASE_URL").unwrap_or(DEFAULT_API_BASE_URL);
        let base_url = Url::parse(configured)
            .map_err(|error| AppError::Message(format!("invalid release API base URL: {error}")))?;
        Self::from_base_url(base_url)
    }

    pub(crate) fn from_base_url(base_url: Url) -> AppResult<Self> {
        if base_url.scheme() != "https" && !cfg!(debug_assertions) {
            return Err(AppError::Message(
                "release API base URL must use HTTPS".into(),
            ));
        }
        let client = reqwest::Client::builder()
            .redirect(Policy::none())
            .connect_timeout(Duration::from_secs(10))
            .user_agent(concat!("LuoxueAPI-Desktop/", env!("CARGO_PKG_VERSION")))
            .build()?;
        Ok(Self { client, base_url })
    }

    pub fn upstream_url(&self, path: &str) -> AppResult<Url> {
        self.base_url
            .join(path.trim_start_matches('/'))
            .map_err(|error| AppError::Message(format!("invalid upstream path: {error}")))
    }

    pub fn http_client(&self) -> reqwest::Client {
        self.client.clone()
    }

    pub async fn create_pairing(
        &self,
        device_name: &str,
        installation_id: &str,
        os_version: &str,
    ) -> AppResult<PairingContext> {
        let verifier = generate_verifier();
        let challenge = URL_SAFE_NO_PAD.encode(Sha256::digest(verifier.as_bytes()));
        #[derive(Serialize)]
        struct Request<'a> {
            device_name: &'a str,
            installation_id: &'a str,
            platform: &'static str,
            architecture: &'static str,
            os_version: &'a str,
            app_version: &'static str,
            code_challenge: String,
            code_challenge_method: &'static str,
        }
        let response: CreatePairingResponse = self
            .post_json(
                "api/v1/desktop/pairings",
                &Request {
                    device_name,
                    installation_id,
                    platform: "macos",
                    architecture: desktop_architecture(),
                    os_version,
                    app_version: env!("CARGO_PKG_VERSION"),
                    code_challenge: challenge,
                    code_challenge_method: "S256",
                },
                None,
            )
            .await?;
        Ok(PairingContext {
            device_code: response.device_code,
            verifier,
            user_code: response.user_code,
            verification_uri: self.resolve_verification_uri(
                response
                    .verification_uri_complete
                    .as_deref()
                    .unwrap_or(&response.verification_uri),
            )?,
            expires_at: response
                .expires_at
                .unwrap_or_else(|| Utc::now() + chrono::Duration::seconds(response.expires_in)),
        })
    }

    pub async fn exchange_pairing(&self, context: &PairingContext) -> AppResult<PairingExchange> {
        if context.expires_at <= Utc::now() {
            return Ok(PairingExchange::Expired);
        }
        #[derive(Serialize)]
        struct Request<'a> {
            device_code: &'a str,
            code_verifier: &'a str,
        }
        let url = self.upstream_url("api/v1/desktop/pairings/token")?;
        let response = self
            .client
            .post(url)
            .json(&Request {
                device_code: context.device_code.as_str(),
                code_verifier: context.verifier.as_str(),
            })
            .timeout(CONTROL_REQUEST_TIMEOUT)
            .send()
            .await?;
        if response.status().is_success() {
            return Ok(PairingExchange::Approved(parse_success(response).await?));
        }
        let status = response.status();
        let error = parse_cloud_error(response).await;
        if error.reason == "DESKTOP_PAIRING_STATE" || status == StatusCode::PRECONDITION_REQUIRED {
            return Ok(PairingExchange::Pending);
        }
        if error.reason == "DESKTOP_PAIRING_EXPIRED" || status == StatusCode::GONE {
            return Ok(PairingExchange::Expired);
        }
        Err(AppError::Message(redacted_cloud_error(status, &error)))
    }

    pub async fn refresh_session(&self, refresh_token: &str) -> AppResult<TokenBundle> {
        #[derive(Serialize)]
        struct Request<'a> {
            refresh_token: &'a str,
        }
        self.post_json(
            "api/v1/desktop/sessions/refresh",
            &Request { refresh_token },
            None,
        )
        .await
    }

    pub async fn list_routes(&self, access_token: &str) -> AppResult<Vec<RouteOption>> {
        let url = self.upstream_url("api/v1/desktop/routes")?;
        let response = self
            .client
            .get(url)
            .bearer_auth(access_token)
            .timeout(CONTROL_REQUEST_TIMEOUT)
            .send()
            .await?;
        let status = response.status();
        if !status.is_success() {
            let error = parse_cloud_error(response).await;
            return Err(AppError::Message(redacted_cloud_error(status, &error)));
        }
        parse_success(response).await
    }

    pub async fn ensure_managed_key(
        &self,
        access_token: &str,
        group_id: i64,
    ) -> AppResult<ManagedKey> {
        let path = format!("api/v1/desktop/managed-keys/{group_id}");
        let url = self.upstream_url(&path)?;
        let response = self
            .client
            .put(url)
            .bearer_auth(access_token)
            .json(&serde_json::json!({}))
            .timeout(CONTROL_REQUEST_TIMEOUT)
            .send()
            .await?;
        let status = response.status();
        if !status.is_success() {
            let error = parse_cloud_error(response).await;
            return Err(AppError::Message(redacted_cloud_error(status, &error)));
        }
        parse_success(response).await
    }

    pub async fn heartbeat(&self, access_token: &str) -> AppResult<DeviceStatus> {
        self.post_json(
            "api/v1/desktop/me/heartbeat",
            &serde_json::json!({}),
            Some(access_token),
        )
        .await
    }

    pub async fn revoke_current_device(&self, access_token: &str) -> AppResult<DeviceStatus> {
        let response = self
            .client
            .delete(self.upstream_url("api/v1/desktop/me")?)
            .bearer_auth(access_token)
            .timeout(CONTROL_REQUEST_TIMEOUT)
            .send()
            .await?;
        let status = response.status();
        if !status.is_success() {
            let error = parse_cloud_error(response).await;
            return Err(cloud_rejection(status, &error));
        }
        parse_success(response).await
    }

    pub async fn today_usage(&self, access_token: &str) -> AppResult<TodayUsage> {
        let url = self.upstream_url("api/v1/desktop/usage/today")?;
        let response = self
            .client
            .get(url)
            .bearer_auth(access_token)
            .timeout(CONTROL_REQUEST_TIMEOUT)
            .send()
            .await?;
        let status = response.status();
        if !status.is_success() {
            let error = parse_cloud_error(response).await;
            return Err(cloud_rejection(status, &error));
        }
        parse_success(response).await
    }

    pub async fn activate_device(&self, access_token: &str) -> AppResult<DeviceStatus> {
        self.post_json(
            "api/v1/desktop/me/activate",
            &serde_json::json!({}),
            Some(access_token),
        )
        .await
    }

    pub async fn upload_diagnostic(
        &self,
        access_token: &str,
        summary: &DiagnosticSummary,
    ) -> AppResult<DiagnosticReceipt> {
        let body = serde_json::json!({
            "app_version": summary.app_version,
            "platform": summary.platform,
            "architecture": summary.architecture,
            "os_version": summary.os_version,
            "gateway": {
                "status": summary.gateway.status,
                "port": summary.gateway.port,
                "takeover_enabled": summary.gateway.takeover_enabled,
            },
            "codex": {
                "config_status": summary.codex.config_status,
                "installations": summary.codex.installations.iter().map(|installation| serde_json::json!({
                    "kind": installation.kind,
                    "installed": installation.installed,
                    "version": installation.version,
                })).collect::<Vec<_>>(),
            },
            "route": {
                "group_id": summary.route.group_id,
                "model": summary.route.model,
                "available_route_count": summary.route.available_route_count,
            },
            "requests": {
                "sample_count": summary.requests.sample_count,
                "success_count": summary.requests.success_count,
                "error_count": summary.requests.error_count,
                "average_duration_ms": summary.requests.average_duration_ms,
                "average_first_token_ms": summary.requests.average_first_token_ms,
                "recent": summary.requests.recent.iter().map(|request| serde_json::json!({
                    "occurred_at": request.occurred_at,
                    "model": request.model,
                    "status_code": request.status_code,
                    "duration_ms": request.duration_ms,
                    "first_token_ms": request.first_token_ms,
                    "request_id": request.request_id,
                })).collect::<Vec<_>>(),
            },
        });
        self.post_json("api/v1/desktop/diagnostics", &body, Some(access_token))
            .await
    }

    async fn post_json<T: DeserializeOwned>(
        &self,
        path: &str,
        body: &impl Serialize,
        token: Option<&str>,
    ) -> AppResult<T> {
        let mut request = self.client.post(self.upstream_url(path)?).json(body);
        if let Some(token) = token {
            request = request.bearer_auth(token);
        }
        let response = request.timeout(CONTROL_REQUEST_TIMEOUT).send().await?;
        let status = response.status();
        if !status.is_success() {
            let error = parse_cloud_error(response).await;
            return Err(cloud_rejection(status, &error));
        }
        parse_success(response).await
    }

    fn resolve_verification_uri(&self, value: &str) -> AppResult<String> {
        match Url::parse(value) {
            Ok(url) => Ok(url.to_string()),
            Err(url::ParseError::RelativeUrlWithoutBase) => {
                self.upstream_url(value).map(|url| url.to_string())
            }
            Err(error) => Err(AppError::Message(format!(
                "invalid verification URL: {error}"
            ))),
        }
    }
}

async fn parse_success<T: DeserializeOwned>(response: reqwest::Response) -> AppResult<T> {
    let envelope = response.json::<ApiEnvelope<T>>().await?;
    if envelope.code != 0 {
        return Err(AppError::Message(format!(
            "cloud response rejected ({})",
            envelope.code
        )));
    }
    envelope
        .data
        .ok_or_else(|| AppError::Message("cloud response did not include data".into()))
}

async fn parse_cloud_error(response: reqwest::Response) -> CloudError {
    response.json::<CloudError>().await.unwrap_or(CloudError {
        code: 0,
        _message: String::new(),
        reason: String::new(),
    })
}

fn generate_verifier() -> Zeroizing<String> {
    let mut random = [0_u8; 32];
    OsRng.fill_bytes(&mut random);
    Zeroizing::new(URL_SAFE_NO_PAD.encode(random))
}

fn redacted_cloud_error(status: StatusCode, error: &CloudError) -> String {
    cloud_rejection(status, error).to_string()
}

fn cloud_rejection(status: StatusCode, error: &CloudError) -> AppError {
    let reason = if !error.reason.is_empty() {
        error.reason.clone()
    } else if error.code != 0 {
        format!("CODE_{}", error.code)
    } else {
        format!("HTTP_{}", status.as_u16())
    };
    AppError::CloudRejected {
        status: status.as_u16(),
        reason,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_api_origin_uses_production_main_site() {
        assert_eq!(DEFAULT_API_BASE_URL, "https://luoxueapi.cc/");

        let client = CloudClient::from_base_url(Url::parse(DEFAULT_API_BASE_URL).unwrap()).unwrap();
        let pairing_url = client.upstream_url("api/v1/desktop/pairings").unwrap();

        assert_eq!(
            pairing_url.as_str(),
            "https://luoxueapi.cc/api/v1/desktop/pairings"
        );
    }

    #[test]
    fn success_envelope_matches_backend_contract() {
        let envelope: ApiEnvelope<Vec<RouteOption>> = serde_json::from_value(serde_json::json!({
            "code": 0,
            "message": "success",
            "data": [{"group_id": 7, "name": "OpenAI", "models": ["gpt-5.4"]}]
        }))
        .unwrap();
        let routes = envelope.data.unwrap();
        assert_eq!(routes[0].group_id, 7);
        assert_eq!(routes[0].rate_multiplier, 1.0);
    }

    #[test]
    fn backend_reason_is_preserved_for_pairing_state() {
        let error: CloudError = serde_json::from_value(serde_json::json!({
            "code": 409,
            "message": "desktop pairing is not in the required state",
            "reason": "DESKTOP_PAIRING_STATE"
        }))
        .unwrap();
        assert_eq!(error.reason, "DESKTOP_PAIRING_STATE");
        assert_eq!(error.code, 409);
    }

    #[test]
    fn revoked_device_errors_remain_machine_classifiable() {
        let error = cloud_rejection(
            StatusCode::FORBIDDEN,
            &CloudError {
                code: 403,
                _message: "desktop device is not authorized".into(),
                reason: "DESKTOP_DEVICE_FORBIDDEN".into(),
            },
        );
        assert!(error.is_desktop_session_revoked());
        assert!(!error
            .to_string()
            .contains("desktop device is not authorized"));
    }

    #[test]
    fn today_usage_matches_desktop_billing_contract() {
        let envelope: ApiEnvelope<TodayUsage> = serde_json::from_value(serde_json::json!({
            "code": 0,
            "message": "success",
            "data": {
                "requests": 12,
                "tokens": 3456,
                "cost": 1.25,
                "balance": 42.5
            }
        }))
        .unwrap();
        let usage = envelope.data.unwrap();
        assert_eq!(usage.requests, 12);
        assert_eq!(usage.tokens, 3456);
        assert_eq!(usage.cost, 1.25);
        assert_eq!(usage.balance, 42.5);
    }

    #[test]
    fn apple_silicon_uses_backend_architecture_name() {
        let expected = if std::env::consts::ARCH == "aarch64" {
            "arm64"
        } else {
            std::env::consts::ARCH
        };
        assert_eq!(desktop_architecture(), expected);
    }
}
