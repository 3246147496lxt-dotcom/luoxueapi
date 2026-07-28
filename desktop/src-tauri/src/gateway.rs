use std::{
    collections::HashMap,
    convert::Infallible,
    sync::{
        atomic::{AtomicBool, Ordering},
        Arc,
    },
    time::Instant,
};

use axum::{
    body::{Body, Bytes},
    extract::{DefaultBodyLimit, OriginalUri, State},
    http::{header, HeaderMap, Method, Response, StatusCode, Uri},
    response::{IntoResponse, Json},
    routing::{get, post},
    Router,
};
use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
use chrono::Utc;
use futures_util::StreamExt;
use hmac::{Hmac, Mac};
use serde_json::Value;
use sha2::{Digest, Sha256};
use subtle::ConstantTimeEq;
use tokio::{
    net::TcpListener,
    sync::{mpsc, oneshot, RwLock, Semaphore},
};
use uuid::Uuid;
use zeroize::Zeroizing;

use crate::{
    cloud::CloudClient,
    error::{AppError, AppResult},
    models::{RequestMetadata, RouteOption},
    storage::LocalStorage,
};

const MAX_BODY_BYTES: usize = 4 * 1024 * 1024;
const MAX_CONCURRENT_REQUESTS: usize = 32;

#[derive(Clone, Debug, Default)]
pub struct RouteSelection {
    pub group_id: Option<i64>,
    pub model: Option<String>,
}

#[derive(Clone)]
pub struct GatewayContext {
    port: u16,
    local_token: Arc<Zeroizing<String>>,
    device_secret: Arc<Zeroizing<String>>,
    cloud: CloudClient,
    storage: LocalStorage,
    selection: Arc<RwLock<RouteSelection>>,
    routes: Arc<RwLock<Vec<RouteOption>>>,
    managed_keys: Arc<RwLock<HashMap<i64, Zeroizing<String>>>>,
    activation_tx: mpsc::UnboundedSender<()>,
    activation_reported: Arc<AtomicBool>,
    concurrency: Arc<Semaphore>,
}

pub struct GatewayHandle {
    pub port: u16,
    shutdown: Option<oneshot::Sender<()>>,
}

impl GatewayHandle {
    pub fn stop(mut self) {
        if let Some(shutdown) = self.shutdown.take() {
            let _ = shutdown.send(());
        }
    }
}

pub struct GatewayDependencies {
    pub local_token: Zeroizing<String>,
    pub device_secret: Zeroizing<String>,
    pub cloud: CloudClient,
    pub storage: LocalStorage,
    pub selection: Arc<RwLock<RouteSelection>>,
    pub routes: Arc<RwLock<Vec<RouteOption>>>,
    pub managed_keys: Arc<RwLock<HashMap<i64, Zeroizing<String>>>>,
    pub activation_tx: mpsc::UnboundedSender<()>,
    pub activation_reported: Arc<AtomicBool>,
}

pub async fn start(
    preferred_port: Option<u16>,
    dependencies: GatewayDependencies,
) -> AppResult<GatewayHandle> {
    let listener = match preferred_port {
        Some(port) => match TcpListener::bind((std::net::Ipv4Addr::LOCALHOST, port)).await {
            Ok(listener) => listener,
            Err(_) => TcpListener::bind((std::net::Ipv4Addr::LOCALHOST, 0)).await?,
        },
        None => TcpListener::bind((std::net::Ipv4Addr::LOCALHOST, 0)).await?,
    };
    let port = listener.local_addr()?.port();
    let context = GatewayContext {
        port,
        local_token: Arc::new(dependencies.local_token),
        device_secret: Arc::new(dependencies.device_secret),
        cloud: dependencies.cloud,
        storage: dependencies.storage,
        selection: dependencies.selection,
        routes: dependencies.routes,
        managed_keys: dependencies.managed_keys,
        activation_tx: dependencies.activation_tx,
        activation_reported: dependencies.activation_reported,
        concurrency: Arc::new(Semaphore::new(MAX_CONCURRENT_REQUESTS)),
    };
    let app = Router::new()
        .route("/models", get(proxy))
        .route("/v1/models", get(proxy))
        .route("/responses", post(proxy))
        .route("/v1/responses", post(proxy))
        .route("/responses/compact", post(proxy))
        .route("/v1/responses/compact", post(proxy))
        .fallback(reject_path)
        .layer(DefaultBodyLimit::max(MAX_BODY_BYTES))
        .with_state(context);
    let (shutdown_tx, shutdown_rx) = oneshot::channel();
    tauri::async_runtime::spawn(async move {
        let result = axum::serve(listener, app)
            .with_graceful_shutdown(async {
                let _ = shutdown_rx.await;
            })
            .await;
        if let Err(error) = result {
            eprintln!("local gateway stopped: {error}");
        }
    });
    Ok(GatewayHandle {
        port,
        shutdown: Some(shutdown_tx),
    })
}

async fn reject_path() -> GatewayFailure {
    GatewayFailure::new(
        StatusCode::NOT_FOUND,
        "path_not_allowed",
        "This local gateway path is not available",
    )
}

async fn proxy(
    State(context): State<GatewayContext>,
    OriginalUri(original_uri): OriginalUri,
    method: Method,
    headers: HeaderMap,
    body: Bytes,
) -> Result<Response<Body>, GatewayFailure> {
    validate_request(&context, &original_uri, &method, &headers)?;
    let permit = context
        .concurrency
        .clone()
        .try_acquire_owned()
        .map_err(|_| {
            GatewayFailure::new(
                StatusCode::TOO_MANY_REQUESTS,
                "gateway_busy",
                "The local gateway is busy",
            )
        })?;

    let parsed = if method == Method::POST {
        Some(serde_json::from_slice::<Value>(&body).map_err(|_| {
            GatewayFailure::new(
                StatusCode::BAD_REQUEST,
                "invalid_json",
                "The request body is not valid JSON",
            )
        })?)
    } else {
        None
    };
    let selection = context.selection.read().await.clone();
    let current_group = selection.group_id.ok_or_else(|| {
        GatewayFailure::new(
            StatusCode::SERVICE_UNAVAILABLE,
            "route_not_selected",
            "Select an OpenAI route in LuoxueAPI Desktop",
        )
    })?;
    let session_id = parsed.as_ref().and_then(extract_session_id);
    let group_id = match session_id {
        Some(session_id) => {
            let session_hash = hash_session(&context.device_secret, session_id);
            let storage = context.storage.clone();
            tokio::task::spawn_blocking(move || {
                storage.resolve_session_group(&session_hash, current_group)
            })
            .await
            .map_err(|_| GatewayFailure::internal("session_store_unavailable"))?
            .map_err(|_| GatewayFailure::internal("session_store_unavailable"))?
        }
        None => current_group,
    };
    let model = parsed
        .as_ref()
        .and_then(|value| value.get("model"))
        .and_then(Value::as_str)
        .map(ToOwned::to_owned)
        .or(selection.model);
    validate_model(&context, group_id, model.as_deref()).await?;

    let managed_keys = context.managed_keys.read().await;
    let managed_key = managed_keys.get(&group_id).ok_or_else(|| {
        GatewayFailure::new(
            StatusCode::SERVICE_UNAVAILABLE,
            "managed_key_unavailable",
            "The selected route needs to be reconnected",
        )
    })?;
    let upstream_url = context
        .cloud
        .upstream_url(original_uri.path())
        .map_err(|_| GatewayFailure::internal("invalid_upstream_path"))?;
    let mut request = context
        .cloud
        .http_client()
        .request(method.clone(), upstream_url)
        .bearer_auth(managed_key.as_str());
    for name in [header::ACCEPT, header::CONTENT_TYPE, header::USER_AGENT] {
        if let Some(value) = headers.get(&name) {
            request = request.header(name, value);
        }
    }
    for name in ["openai-beta", "x-request-id", "x-stainless-helper-method"] {
        if let Some(value) = headers.get(name) {
            request = request.header(name, value);
        }
    }
    if method == Method::POST {
        request = request.body(body);
    }
    let started = Instant::now();
    let upstream = request.send().await.map_err(|_| {
        GatewayFailure::new(
            StatusCode::BAD_GATEWAY,
            "cloud_unreachable",
            "LuoxueAPI is temporarily unreachable",
        )
    })?;
    if upstream.status().is_redirection() {
        return Err(GatewayFailure::new(
            StatusCode::BAD_GATEWAY,
            "upstream_redirect_rejected",
            "Unexpected cloud redirect",
        ));
    }

    let status = upstream.status();
    let response_headers = upstream.headers().clone();
    let request_id = extract_request_id(&response_headers)
        .unwrap_or_else(|| format!("local_{}", Uuid::new_v4().simple()));
    let first_header_ms = started.elapsed().as_millis() as i64;
    let response_model = model.unwrap_or_default();
    let record_request = method == Method::POST;
    let activation_candidate = status.is_success()
        && method == Method::POST
        && matches!(original_uri.path(), "/responses" | "/v1/responses");
    let storage = context.storage.clone();
    let activation_tx = context.activation_tx.clone();
    let activation_reported = context.activation_reported.clone();
    let mut upstream_stream = upstream.bytes_stream();
    let response_request_id = request_id.clone();
    let stream = async_stream::stream! {
        let _permit = permit;
        let mut usage = UsageAccumulator::default();
        let mut first_token_ms = None;
        let mut stream_complete = true;
        while let Some(item) = upstream_stream.next().await {
            match item {
                Ok(chunk) => {
                    if first_token_ms.is_none() {
                        first_token_ms = Some(started.elapsed().as_millis() as i64);
                    }
                    usage.consume(&chunk);
                    yield Ok::<Bytes, Infallible>(chunk);
                }
                Err(_) => {
                    stream_complete = false;
                    break;
                },
            }
        }
        if record_request {
            let item = RequestMetadata {
                id: Uuid::new_v4().to_string(),
                occurred_at: Utc::now().to_rfc3339(),
                model: response_model,
                status_code: status.as_u16(),
                input_tokens: usage.input_tokens,
                output_tokens: usage.output_tokens,
                duration_ms: started.elapsed().as_millis() as i64,
                first_token_ms: first_token_ms.or(Some(first_header_ms)),
                request_id: response_request_id,
            };
            let _ = tokio::task::spawn_blocking(move || storage.insert_request(&item)).await;
        }
        if activation_candidate
            && stream_complete
            && activation_reported
                .compare_exchange(false, true, Ordering::SeqCst, Ordering::SeqCst)
                .is_ok()
            && activation_tx.send(()).is_err()
        {
            activation_reported.store(false, Ordering::SeqCst);
        }
    };

    let mut builder = Response::builder().status(status);
    for name in [header::CONTENT_TYPE, header::CACHE_CONTROL] {
        if let Some(value) = response_headers.get(&name) {
            builder = builder.header(name, value);
        }
    }
    builder = builder.header("x-request-id", request_id);
    builder
        .body(Body::from_stream(stream))
        .map_err(|_| GatewayFailure::internal("response_build_failed"))
}

fn validate_request(
    context: &GatewayContext,
    uri: &Uri,
    method: &Method,
    headers: &HeaderMap,
) -> Result<(), GatewayFailure> {
    let allowed = matches!(
        (method.as_str(), uri.path()),
        ("GET", "/models")
            | ("GET", "/v1/models")
            | ("POST", "/responses")
            | ("POST", "/v1/responses")
            | ("POST", "/responses/compact")
            | ("POST", "/v1/responses/compact")
    );
    if !allowed || uri.path().contains('%') || uri.query().is_some() {
        return Err(GatewayFailure::new(
            StatusCode::NOT_FOUND,
            "path_not_allowed",
            "This local gateway path is not available",
        ));
    }
    if headers.contains_key(header::ORIGIN) {
        return Err(GatewayFailure::new(
            StatusCode::FORBIDDEN,
            "browser_origin_rejected",
            "Browser-origin requests are not accepted",
        ));
    }
    let expected_host = format!("127.0.0.1:{}", context.port);
    if headers
        .get(header::HOST)
        .and_then(|value| value.to_str().ok())
        != Some(expected_host.as_str())
    {
        return Err(GatewayFailure::new(
            StatusCode::FORBIDDEN,
            "invalid_host",
            "The Host header is not valid for this gateway",
        ));
    }
    let token = headers
        .get(header::AUTHORIZATION)
        .and_then(|value| value.to_str().ok())
        .and_then(|value| value.strip_prefix("Bearer "))
        .ok_or_else(|| {
            GatewayFailure::new(
                StatusCode::UNAUTHORIZED,
                "local_auth_required",
                "Local gateway authentication is required",
            )
        })?;
    if !constant_time_token_eq(token, context.local_token.as_str()) {
        return Err(GatewayFailure::new(
            StatusCode::UNAUTHORIZED,
            "local_auth_invalid",
            "Local gateway authentication failed",
        ));
    }
    Ok(())
}

async fn validate_model(
    context: &GatewayContext,
    group_id: i64,
    model: Option<&str>,
) -> Result<(), GatewayFailure> {
    let Some(model) = model else {
        return Ok(());
    };
    let routes = context.routes.read().await;
    let route = routes
        .iter()
        .find(|route| route.group_id == group_id)
        .ok_or_else(|| {
            GatewayFailure::new(
                StatusCode::SERVICE_UNAVAILABLE,
                "pinned_route_unavailable",
                "The session route is no longer available",
            )
        })?;
    if !route.models.iter().any(|candidate| candidate == model) {
        return Err(GatewayFailure::new(
            StatusCode::BAD_REQUEST,
            "model_not_in_route",
            "The selected model is not available on this route",
        ));
    }
    Ok(())
}

fn extract_session_id(value: &Value) -> Option<&str> {
    for key in ["session_id", "conversation_id", "prompt_cache_key"] {
        if let Some(value) = value
            .get(key)
            .and_then(Value::as_str)
            .filter(|value| !value.is_empty())
        {
            return Some(value);
        }
        if let Some(value) = value
            .get("metadata")
            .and_then(|metadata| metadata.get(key))
            .and_then(Value::as_str)
            .filter(|value| !value.is_empty())
        {
            return Some(value);
        }
    }
    None
}

fn hash_session(secret: &str, session_id: &str) -> String {
    let mut hmac =
        Hmac::<Sha256>::new_from_slice(secret.as_bytes()).expect("HMAC accepts any key size");
    hmac.update(session_id.as_bytes());
    URL_SAFE_NO_PAD.encode(hmac.finalize().into_bytes())
}

fn constant_time_token_eq(candidate: &str, expected: &str) -> bool {
    let candidate = Sha256::digest(candidate.as_bytes());
    let expected = Sha256::digest(expected.as_bytes());
    bool::from(candidate.ct_eq(&expected))
}

fn extract_request_id(headers: &HeaderMap) -> Option<String> {
    ["x-request-id", "request-id", "openai-request-id"]
        .iter()
        .find_map(|name| {
            headers
                .get(*name)
                .and_then(|value| value.to_str().ok())
                .map(ToOwned::to_owned)
        })
}

#[derive(Default)]
struct UsageAccumulator {
    buffer: Vec<u8>,
    input_tokens: i64,
    output_tokens: i64,
}

impl UsageAccumulator {
    fn consume(&mut self, chunk: &[u8]) {
        if self.buffer.len() + chunk.len() > 256 * 1024 {
            self.buffer.clear();
        }
        self.buffer.extend_from_slice(chunk);
        while let Some(position) = self.buffer.iter().position(|byte| *byte == b'\n') {
            let line = self.buffer.drain(..=position).collect::<Vec<_>>();
            let line = String::from_utf8_lossy(&line);
            let data = line
                .trim()
                .strip_prefix("data:")
                .map(str::trim)
                .unwrap_or(line.trim());
            if data.starts_with('{') {
                if let Ok(value) = serde_json::from_str::<Value>(data) {
                    if let Some(usage) = find_usage(&value) {
                        self.input_tokens = usage.0;
                        self.output_tokens = usage.1;
                    }
                }
            }
        }
    }
}

fn find_usage(value: &Value) -> Option<(i64, i64)> {
    if let Some(usage) = value.get("usage") {
        let input = usage
            .get("input_tokens")
            .and_then(Value::as_i64)
            .unwrap_or(0);
        let output = usage
            .get("output_tokens")
            .and_then(Value::as_i64)
            .unwrap_or(0);
        if input > 0 || output > 0 {
            return Some((input, output));
        }
    }
    value.get("response").and_then(find_usage)
}

pub struct GatewayFailure {
    status: StatusCode,
    code: &'static str,
    message: &'static str,
}

impl GatewayFailure {
    fn new(status: StatusCode, code: &'static str, message: &'static str) -> Self {
        Self {
            status,
            code,
            message,
        }
    }

    fn internal(code: &'static str) -> Self {
        Self::new(
            StatusCode::INTERNAL_SERVER_ERROR,
            code,
            "The local gateway could not complete this request",
        )
    }
}

impl IntoResponse for GatewayFailure {
    fn into_response(self) -> axum::response::Response {
        (
            self.status,
            Json(serde_json::json!({
                "error": { "code": self.code, "message": self.message }
            })),
        )
            .into_response()
    }
}

impl From<AppError> for GatewayFailure {
    fn from(_: AppError) -> Self {
        Self::internal("gateway_internal_error")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::sync::Mutex as TokioMutex;

    #[derive(Clone, Default)]
    struct CapturedRequest {
        authorization: Option<String>,
        body: Vec<u8>,
        count: usize,
    }

    async fn mock_responses(
        State(captured): State<Arc<TokioMutex<CapturedRequest>>>,
        headers: HeaderMap,
        body: Bytes,
    ) -> Response<Body> {
        let mut captured = captured.lock().await;
        captured.authorization = headers
            .get(header::AUTHORIZATION)
            .and_then(|value| value.to_str().ok())
            .map(ToOwned::to_owned);
        captured.body = body.to_vec();
        captured.count += 1;
        let chunks = [
            Ok::<_, Infallible>(Bytes::from_static(b"data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n")),
            Ok::<_, Infallible>(Bytes::from_static(b"data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":2,\"output_tokens\":1}}}\n\n")),
        ];
        Response::builder()
            .status(StatusCode::OK)
            .header(header::CONTENT_TYPE, "text/event-stream")
            .header("x-request-id", "upstream-request")
            .body(Body::from_stream(futures_util::stream::iter(chunks)))
            .unwrap()
    }

    async fn test_gateway(
        preferred_port: Option<u16>,
    ) -> (
        GatewayHandle,
        Arc<TokioMutex<CapturedRequest>>,
        tempfile::TempDir,
        tokio::task::JoinHandle<()>,
        mpsc::UnboundedReceiver<()>,
    ) {
        let captured = Arc::new(TokioMutex::new(CapturedRequest::default()));
        let upstream = TcpListener::bind((std::net::Ipv4Addr::LOCALHOST, 0))
            .await
            .unwrap();
        let upstream_port = upstream.local_addr().unwrap().port();
        let upstream_app = Router::new()
            .route("/v1/responses", post(mock_responses))
            .with_state(captured.clone());
        let upstream_task = tokio::spawn(async move {
            axum::serve(upstream, upstream_app).await.unwrap();
        });

        let directory = tempfile::tempdir().unwrap();
        let storage = LocalStorage::open(directory.path().join("gateway.db")).unwrap();
        let mut keys = HashMap::new();
        keys.insert(17, Zeroizing::new("managed-secret".into()));
        let (activation_tx, activation_rx) = mpsc::unbounded_channel();
        let handle = super::start(
            preferred_port,
            GatewayDependencies {
                local_token: Zeroizing::new("local-secret".into()),
                device_secret: Zeroizing::new("device-secret".into()),
                cloud: CloudClient::from_base_url(
                    url::Url::parse(&format!("http://127.0.0.1:{upstream_port}/")).unwrap(),
                )
                .unwrap(),
                storage,
                selection: Arc::new(RwLock::new(RouteSelection {
                    group_id: Some(17),
                    model: Some("gpt-5.4".into()),
                })),
                routes: Arc::new(RwLock::new(vec![RouteOption {
                    group_id: 17,
                    name: "OpenAI".into(),
                    rate_multiplier: 1.0,
                    models: vec!["gpt-5.4".into()],
                }])),
                managed_keys: Arc::new(RwLock::new(keys)),
                activation_tx,
                activation_reported: Arc::new(AtomicBool::new(false)),
            },
        )
        .await
        .unwrap();
        (handle, captured, directory, upstream_task, activation_rx)
    }

    #[test]
    fn session_signal_priority_is_stable() {
        let value = serde_json::json!({
            "session_id": "session",
            "conversation_id": "conversation",
            "prompt_cache_key": "cache"
        });
        assert_eq!(extract_session_id(&value), Some("session"));
    }

    #[test]
    fn usage_accumulator_extracts_only_usage_numbers() {
        let mut usage = UsageAccumulator::default();
        usage.consume(b"data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":12,\"output_tokens\":5}}}\n\n");
        assert_eq!((usage.input_tokens, usage.output_tokens), (12, 5));
        assert!(usage.buffer.is_empty());
    }

    #[test]
    fn session_hash_changes_with_secret() {
        assert_ne!(hash_session("a", "same"), hash_session("b", "same"));
    }

    #[tokio::test]
    async fn gateway_replaces_auth_and_preserves_sse_bytes() {
        let (handle, captured, _directory, upstream_task, mut activation_rx) =
            test_gateway(None).await;
        let expected = b"data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":2,\"output_tokens\":1}}}\n\n";
        let response = reqwest::Client::new()
            .post(format!("http://127.0.0.1:{}/v1/responses", handle.port))
            .bearer_auth("local-secret")
            .json(&serde_json::json!({"model":"gpt-5.4","input":"not persisted"}))
            .send()
            .await
            .unwrap();
        assert_eq!(response.status(), StatusCode::OK);
        assert_eq!(
            response.headers().get("x-request-id").unwrap(),
            "upstream-request"
        );
        assert_eq!(response.bytes().await.unwrap().as_ref(), expected);
        tokio::time::timeout(std::time::Duration::from_secs(1), activation_rx.recv())
            .await
            .unwrap()
            .expect("first successful Responses request should activate the device");

        let captured = captured.lock().await.clone();
        assert_eq!(
            captured.authorization.as_deref(),
            Some("Bearer managed-secret")
        );
        assert!(String::from_utf8(captured.body)
            .unwrap()
            .contains("not persisted"));
        handle.stop();
        upstream_task.abort();
    }

    #[tokio::test]
    async fn gateway_rejects_unauthorized_browser_and_model_requests() {
        let (handle, captured, _directory, upstream_task, _activation_rx) =
            test_gateway(None).await;
        let client = reqwest::Client::new();
        let base = format!("http://127.0.0.1:{}", handle.port);

        let wrong_token = client
            .post(format!("{base}/v1/responses"))
            .bearer_auth("wrong")
            .json(&serde_json::json!({"model":"gpt-5.4"}))
            .send()
            .await
            .unwrap();
        assert_eq!(wrong_token.status(), StatusCode::UNAUTHORIZED);

        let browser = client
            .post(format!("{base}/v1/responses"))
            .bearer_auth("local-secret")
            .header(header::ORIGIN, "https://example.com")
            .json(&serde_json::json!({"model":"gpt-5.4"}))
            .send()
            .await
            .unwrap();
        assert_eq!(browser.status(), StatusCode::FORBIDDEN);

        let model = client
            .post(format!("{base}/v1/responses"))
            .bearer_auth("local-secret")
            .json(&serde_json::json!({"model":"not-allowed"}))
            .send()
            .await
            .unwrap();
        assert_eq!(model.status(), StatusCode::BAD_REQUEST);

        let path = client
            .post(format!("{base}/v1/chat/completions"))
            .bearer_auth("local-secret")
            .json(&serde_json::json!({}))
            .send()
            .await
            .unwrap();
        assert_eq!(path.status(), StatusCode::NOT_FOUND);
        assert_eq!(captured.lock().await.count, 0);
        handle.stop();
        upstream_task.abort();
    }

    #[tokio::test]
    async fn occupied_preferred_port_falls_back_without_releasing_the_socket() {
        let occupied = TcpListener::bind((std::net::Ipv4Addr::LOCALHOST, 0))
            .await
            .unwrap();
        let occupied_port = occupied.local_addr().unwrap().port();
        let (handle, _captured, _directory, upstream_task, _activation_rx) =
            test_gateway(Some(occupied_port)).await;
        assert_ne!(handle.port, occupied_port);
        handle.stop();
        upstream_task.abort();
    }
}
