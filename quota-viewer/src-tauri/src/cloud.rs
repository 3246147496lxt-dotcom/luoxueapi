use std::time::Duration;

use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
use chrono::Utc;
use rand::{rngs::OsRng, RngCore};
use reqwest::{redirect::Policy, StatusCode};
use serde::{de::DeserializeOwned, Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use url::Url;
use zeroize::Zeroizing;

use crate::{
    error::{AppError, AppResult},
    models::{
        PairingChallenge, PairingContext, PairingExchange, PendingRefreshRotation,
        RefreshTokenBundle, SafeQuotaOverview,
    },
};

const DEFAULT_API_BASE_URL: &str = "https://luoxueapi.cc/";
const CLIENT_ID: &str = "luoxue-quota-viewer";
const SCOPE: &str = "quota:read";
const REQUEST_TIMEOUT: Duration = Duration::from_secs(30);

#[derive(Clone)]
pub struct QuotaCloudClient {
    client: reqwest::Client,
    base_url: Url,
}

#[derive(Deserialize)]
struct ApiEnvelope<T> {
    code: i64,
    #[serde(default, rename = "message")]
    _message: String,
    #[serde(default)]
    reason: String,
    data: Option<T>,
}

#[derive(Deserialize)]
struct CloudError {
    #[serde(default)]
    code: i64,
    #[serde(default)]
    reason: String,
}

#[derive(Serialize)]
struct RefreshSessionRequest<'a> {
    client_id: &'static str,
    refresh_token: &'a str,
    rotation_id: &'a str,
    candidate_refresh_token: &'a str,
}

impl QuotaCloudClient {
    pub fn new() -> AppResult<Self> {
        let configured = option_env!("LUOXUE_QUOTA_API_BASE_URL")
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .unwrap_or(DEFAULT_API_BASE_URL);
        Self::from_base_url(Url::parse(configured)?)
    }

    pub(crate) fn from_base_url(base_url: Url) -> AppResult<Self> {
        if base_url.scheme() != "https" && !cfg!(debug_assertions) {
            return Err(AppError::Message(
                "release quota API URL must use HTTPS".into(),
            ));
        }
        if base_url.cannot_be_a_base() || base_url.host_str().is_none() {
            return Err(AppError::Message("quota API URL must be an origin".into()));
        }
        let client = reqwest::Client::builder()
            .redirect(Policy::none())
            .connect_timeout(Duration::from_secs(10))
            .timeout(REQUEST_TIMEOUT)
            .user_agent(concat!("Luoxue-Quota-Viewer/", env!("CARGO_PKG_VERSION")))
            .build()?;
        Ok(Self { client, base_url })
    }

    fn endpoint(&self, path: &str) -> AppResult<Url> {
        Ok(self.base_url.join(path.trim_start_matches('/'))?)
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
            client_id: &'static str,
            scope: &'static str,
            code_challenge: String,
            code_challenge_method: &'static str,
            installation_id: &'a str,
            device_name: &'a str,
            platform: &'static str,
            architecture: &'static str,
            os_version: &'a str,
            app_version: &'static str,
        }
        let response: PairingChallenge = self
            .post_json(
                "api/v1/quota/pairings",
                &Request {
                    client_id: CLIENT_ID,
                    scope: SCOPE,
                    code_challenge: challenge,
                    code_challenge_method: "S256",
                    installation_id,
                    device_name,
                    platform: quota_platform(),
                    architecture: quota_architecture(),
                    os_version,
                    app_version: env!("CARGO_PKG_VERSION"),
                },
            )
            .await?;
        let verification_uri = self.resolve_verification_uri(
            response
                .verification_uri_complete
                .as_deref()
                .unwrap_or(&response.verification_uri),
        )?;
        Ok(PairingContext {
            device_code: response.device_code,
            verifier,
            user_code: response.user_code,
            verification_uri,
            expires_at: response.expires_at.unwrap_or_else(|| {
                Utc::now() + chrono::Duration::seconds(response.expires_in.max(1))
            }),
            interval: response.interval.clamp(5, 30),
        })
    }

    pub async fn exchange_pairing(&self, context: &PairingContext) -> AppResult<PairingExchange> {
        if context.expires_at <= Utc::now() {
            return Ok(PairingExchange::Expired);
        }
        #[derive(Serialize)]
        struct Request<'a> {
            client_id: &'static str,
            device_code: &'a str,
            code_verifier: &'a str,
        }
        let response = self
            .client
            .post(self.endpoint("api/v1/quota/pairings/token")?)
            .json(&Request {
                client_id: CLIENT_ID,
                device_code: context.device_code.as_str(),
                code_verifier: context.verifier.as_str(),
            })
            .send()
            .await?;
        if response.status().is_success() {
            return Ok(PairingExchange::Approved(parse_success(response).await?));
        }
        let status = response.status();
        let error = parse_cloud_error(response).await;
        if status == StatusCode::CONFLICT && error.reason == "QUOTA_AUTH_PAIRING_STATE" {
            return Ok(PairingExchange::Pending);
        }
        if error.reason == "QUOTA_AUTH_PAIRING_EXPIRED" {
            return Ok(PairingExchange::Expired);
        }
        Err(cloud_rejection(status, error))
    }

    pub async fn refresh_session(
        &self,
        refresh_token: &str,
        pending: &PendingRefreshRotation,
    ) -> AppResult<RefreshTokenBundle> {
        self.post_json(
            "api/v1/quota/sessions/refresh",
            &RefreshSessionRequest {
                client_id: CLIENT_ID,
                refresh_token,
                rotation_id: &pending.rotation_id,
                candidate_refresh_token: pending.candidate_refresh_token.as_str(),
            },
        )
        .await
    }

    pub async fn overview(&self, access_token: &str, timezone: &str) -> AppResult<Value> {
        let mut url = self.endpoint("api/v1/quota/overview")?;
        url.query_pairs_mut().append_pair("timezone", timezone);
        let response = self
            .client
            .get(url)
            .bearer_auth(access_token)
            .send()
            .await?;
        if response.status().is_success() {
            let safe_overview: SafeQuotaOverview = parse_success(response).await?;
            return Ok(safe_overview.into_json_value()?);
        }
        let status = response.status();
        let error = parse_cloud_error(response).await;
        Err(cloud_rejection(status, error))
    }

    async fn post_json<T: DeserializeOwned>(
        &self,
        path: &str,
        body: &impl Serialize,
    ) -> AppResult<T> {
        let response = self
            .client
            .post(self.endpoint(path)?)
            .json(body)
            .send()
            .await?;
        if response.status().is_success() {
            return parse_success(response).await;
        }
        let status = response.status();
        let error = parse_cloud_error(response).await;
        Err(cloud_rejection(status, error))
    }

    fn resolve_verification_uri(&self, value: &str) -> AppResult<String> {
        let url = match Url::parse(value) {
            Ok(url) => url,
            Err(url::ParseError::RelativeUrlWithoutBase) => self.endpoint(value)?,
            Err(error) => return Err(error.into()),
        };
        let same_origin = url.scheme() == self.base_url.scheme()
            && url.host_str() == self.base_url.host_str()
            && url.port_or_known_default() == self.base_url.port_or_known_default();
        if !same_origin
            || url.username() != ""
            || url.password().is_some()
            || url.fragment().is_some()
            || url.path() != "/quota-viewer/authorize"
        {
            return Err(AppError::Message(
                "quota authorization URL is not an approved official page".into(),
            ));
        }
        Ok(url.to_string())
    }
}

fn quota_platform() -> &'static str {
    #[cfg(target_os = "macos")]
    {
        return "macos";
    }
    #[cfg(target_os = "windows")]
    {
        return "windows";
    }
    #[allow(unreachable_code)]
    "macos"
}

fn quota_architecture() -> &'static str {
    match std::env::consts::ARCH {
        "aarch64" => "arm64",
        "x86_64" => "x86_64",
        _ => "x86_64",
    }
}

fn generate_verifier() -> Zeroizing<String> {
    let mut random = [0_u8; 32];
    OsRng.fill_bytes(&mut random);
    Zeroizing::new(URL_SAFE_NO_PAD.encode(random))
}

async fn parse_success<T: DeserializeOwned>(response: reqwest::Response) -> AppResult<T> {
    let envelope = response.json::<ApiEnvelope<T>>().await?;
    if envelope.code != 0 {
        return Err(AppError::CloudRejected {
            status: 200,
            reason: envelope.reason,
        });
    }
    envelope
        .data
        .ok_or_else(|| AppError::Message("quota API response did not include data".into()))
}

async fn parse_cloud_error(response: reqwest::Response) -> CloudError {
    response.json::<CloudError>().await.unwrap_or(CloudError {
        code: 0,
        reason: String::new(),
    })
}

fn cloud_rejection(status: StatusCode, error: CloudError) -> AppError {
    let reason = if !error.reason.is_empty() {
        error.reason
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
    fn pkce_verifier_has_backend_compatible_shape() {
        let verifier = generate_verifier();
        assert_eq!(verifier.len(), 43);
        assert!(verifier
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || byte == b'-' || byte == b'_'));
        let challenge = URL_SAFE_NO_PAD.encode(Sha256::digest(verifier.as_bytes()));
        assert_eq!(challenge.len(), 43);
    }

    #[test]
    fn authorization_page_must_stay_on_the_configured_origin_and_path() {
        let client =
            QuotaCloudClient::from_base_url(Url::parse("https://luoxueapi.cc/").unwrap()).unwrap();
        assert_eq!(
            client
                .resolve_verification_uri("/quota-viewer/authorize?user_code=ABCD-EFGH")
                .unwrap(),
            "https://luoxueapi.cc/quota-viewer/authorize?user_code=ABCD-EFGH"
        );
        assert!(client
            .resolve_verification_uri("https://evil.example/quota-viewer/authorize")
            .is_err());
        assert!(client
            .resolve_verification_uri("https://luoxueapi.cc/desktop/authorize")
            .is_err());
        assert!(client
            .resolve_verification_uri("https://user@luoxueapi.cc/quota-viewer/authorize")
            .is_err());
    }

    #[test]
    fn refresh_request_always_uses_the_candidate_v1_tuple() {
        let pending = PendingRefreshRotation::generate("qvrt_base");
        let value = serde_json::to_value(RefreshSessionRequest {
            client_id: CLIENT_ID,
            refresh_token: "qvrt_base",
            rotation_id: &pending.rotation_id,
            candidate_refresh_token: pending.candidate_refresh_token.as_str(),
        })
        .unwrap();

        assert_eq!(
            value,
            serde_json::json!({
                "client_id": "luoxue-quota-viewer",
                "refresh_token": "qvrt_base",
                "rotation_id": pending.rotation_id.clone(),
                "candidate_refresh_token": pending.candidate_refresh_token.as_str(),
            })
        );
    }
}
