use std::{
    future::Future,
    path::PathBuf,
    pin::Pin,
    sync::{
        atomic::{AtomicU64, Ordering},
        Arc,
    },
};

use chrono::Utc;
use serde_json::Value;
use tauri::AppHandle;
use tauri_plugin_opener::OpenerExt;
use tokio::sync::{Mutex, MutexGuard, RwLock};
use zeroize::Zeroizing;

use crate::{
    cache::{cache_is_fresh, OverviewCache},
    cloud::QuotaCloudClient,
    error::{AppError, AppResult},
    keychain::CredentialStore,
    models::{
        AccessSession, CachedOverview, PairingContext, PairingExchange, PairingState,
        PendingRefreshRotation, RefreshTokenBundle, ViewerSnapshot,
    },
};

type CloudFuture<'a, T> = Pin<Box<dyn Future<Output = AppResult<T>> + Send + 'a>>;

trait CloudPort: Send + Sync {
    fn create_pairing<'a>(
        &'a self,
        device_name: &'a str,
        installation_id: &'a str,
        os_version: &'a str,
    ) -> CloudFuture<'a, PairingContext>;

    fn exchange_pairing<'a>(
        &'a self,
        context: &'a PairingContext,
    ) -> CloudFuture<'a, PairingExchange>;

    fn refresh_session<'a>(
        &'a self,
        refresh_token: &'a str,
        pending: &'a PendingRefreshRotation,
    ) -> CloudFuture<'a, RefreshTokenBundle>;

    fn overview<'a>(&'a self, access_token: &'a str, timezone: &'a str) -> CloudFuture<'a, Value>;
}

impl CloudPort for QuotaCloudClient {
    fn create_pairing<'a>(
        &'a self,
        device_name: &'a str,
        installation_id: &'a str,
        os_version: &'a str,
    ) -> CloudFuture<'a, PairingContext> {
        Box::pin(async move {
            QuotaCloudClient::create_pairing(self, device_name, installation_id, os_version).await
        })
    }

    fn exchange_pairing<'a>(
        &'a self,
        context: &'a PairingContext,
    ) -> CloudFuture<'a, PairingExchange> {
        Box::pin(async move { QuotaCloudClient::exchange_pairing(self, context).await })
    }

    fn refresh_session<'a>(
        &'a self,
        refresh_token: &'a str,
        pending: &'a PendingRefreshRotation,
    ) -> CloudFuture<'a, RefreshTokenBundle> {
        Box::pin(
            async move { QuotaCloudClient::refresh_session(self, refresh_token, pending).await },
        )
    }

    fn overview<'a>(&'a self, access_token: &'a str, timezone: &'a str) -> CloudFuture<'a, Value> {
        Box::pin(async move { QuotaCloudClient::overview(self, access_token, timezone).await })
    }
}

trait CredentialPort: Send + Sync {
    fn refresh_token(&self) -> AppResult<Option<Zeroizing<String>>>;
    fn require_refresh_token(&self) -> AppResult<Zeroizing<String>>;
    fn set_refresh_token(&self, token: &str) -> AppResult<()>;
    fn clear_refresh_token(&self) -> AppResult<()>;
    fn pending_refresh_rotation(&self) -> AppResult<Option<PendingRefreshRotation>>;
    fn set_pending_refresh_rotation(&self, pending: &PendingRefreshRotation) -> AppResult<()>;
    fn clear_pending_refresh_rotation(&self) -> AppResult<()>;
    fn installation_id(&self) -> AppResult<Zeroizing<String>>;
    fn cache_id(&self) -> AppResult<Option<Zeroizing<String>>>;
    fn rotate_cache_id(&self) -> AppResult<Zeroizing<String>>;
    fn clear_cache_id(&self) -> AppResult<()>;
}

impl CredentialPort for CredentialStore {
    fn refresh_token(&self) -> AppResult<Option<Zeroizing<String>>> {
        CredentialStore::refresh_token(self)
    }

    fn require_refresh_token(&self) -> AppResult<Zeroizing<String>> {
        CredentialStore::require_refresh_token(self)
    }

    fn set_refresh_token(&self, token: &str) -> AppResult<()> {
        CredentialStore::set_refresh_token(self, token)
    }

    fn clear_refresh_token(&self) -> AppResult<()> {
        CredentialStore::clear_refresh_token(self)
    }

    fn pending_refresh_rotation(&self) -> AppResult<Option<PendingRefreshRotation>> {
        CredentialStore::pending_refresh_rotation(self)
    }

    fn set_pending_refresh_rotation(&self, pending: &PendingRefreshRotation) -> AppResult<()> {
        CredentialStore::set_pending_refresh_rotation(self, pending)
    }

    fn clear_pending_refresh_rotation(&self) -> AppResult<()> {
        CredentialStore::clear_pending_refresh_rotation(self)
    }

    fn installation_id(&self) -> AppResult<Zeroizing<String>> {
        CredentialStore::installation_id(self)
    }

    fn cache_id(&self) -> AppResult<Option<Zeroizing<String>>> {
        CredentialStore::cache_id(self)
    }

    fn rotate_cache_id(&self) -> AppResult<Zeroizing<String>> {
        CredentialStore::rotate_cache_id(self)
    }

    fn clear_cache_id(&self) -> AppResult<()> {
        CredentialStore::clear_cache_id(self)
    }
}

trait CachePort: Send + Sync {
    fn load(
        &self,
        expected_cache_id: &str,
        expected_display_timezone: &str,
    ) -> AppResult<Option<CachedOverview>>;
    fn save(
        &self,
        overview: Value,
        cache_id: &str,
        display_timezone: &str,
    ) -> AppResult<CachedOverview>;
    fn clear(&self) -> AppResult<()>;
}

impl CachePort for OverviewCache {
    fn load(
        &self,
        expected_cache_id: &str,
        expected_display_timezone: &str,
    ) -> AppResult<Option<CachedOverview>> {
        OverviewCache::load(self, expected_cache_id, expected_display_timezone)
    }

    fn save(
        &self,
        overview: Value,
        cache_id: &str,
        display_timezone: &str,
    ) -> AppResult<CachedOverview> {
        OverviewCache::save(self, overview, cache_id, display_timezone)
    }

    fn clear(&self) -> AppResult<()> {
        OverviewCache::clear(self)
    }
}

#[cfg(test)]
#[derive(Clone, Copy, PartialEq, Eq)]
enum TestWritePoint {
    PairingContext,
    PairingCredentials,
    RefreshedCredentials,
    OverviewCache,
}

#[cfg(test)]
struct TestWritePause {
    point: TestWritePoint,
    entered: Arc<tokio::sync::Barrier>,
    release: Arc<tokio::sync::Notify>,
}

pub struct AppRuntime {
    cloud: Arc<dyn CloudPort>,
    credentials: Arc<dyn CredentialPort>,
    cache: Arc<dyn CachePort>,
    pairing: Mutex<Option<PairingContext>>,
    access_session: RwLock<Option<AccessSession>>,
    authorization: AuthorizationGate,
    poll_singleflight: Mutex<()>,
    refresh_singleflight: Mutex<()>,
    cache_commit: Mutex<()>,
    #[cfg(test)]
    write_pause: std::sync::Mutex<Option<Arc<TestWritePause>>>,
}

struct AuthorizationGate {
    commit: Mutex<()>,
    generation: AtomicU64,
}

impl AuthorizationGate {
    fn new() -> Self {
        Self {
            commit: Mutex::new(()),
            generation: AtomicU64::new(1),
        }
    }

    fn current(&self) -> u64 {
        self.generation.load(Ordering::SeqCst)
    }

    fn require_current(&self, expected: u64) -> AppResult<()> {
        if self.current() != expected {
            return Err(AppError::Message(
                "quota authorization state changed".into(),
            ));
        }
        Ok(())
    }

    fn begin_transition(&self) -> u64 {
        self.generation.fetch_add(1, Ordering::SeqCst) + 1
    }

    fn begin_transition_if_current(&self, expected: u64) -> Option<u64> {
        self.generation
            .compare_exchange(expected, expected + 1, Ordering::SeqCst, Ordering::SeqCst)
            .ok()
            .map(|_| expected + 1)
    }

    async fn lock_current(&self, expected: u64) -> AppResult<MutexGuard<'_, ()>> {
        let guard = self.commit.lock().await;
        self.require_current(expected)?;
        Ok(guard)
    }

    async fn lock_commit(&self) -> MutexGuard<'_, ()> {
        self.commit.lock().await
    }
}

impl AppRuntime {
    pub fn new(cache_path: PathBuf) -> AppResult<Self> {
        let credentials = CredentialStore;
        let snapshot_key = credentials.snapshot_key()?;
        Ok(Self {
            cloud: Arc::new(QuotaCloudClient::new()?),
            credentials: Arc::new(credentials),
            cache: Arc::new(OverviewCache::new(cache_path, snapshot_key)?),
            pairing: Mutex::new(None),
            access_session: RwLock::new(None),
            authorization: AuthorizationGate::new(),
            poll_singleflight: Mutex::new(()),
            refresh_singleflight: Mutex::new(()),
            cache_commit: Mutex::new(()),
            #[cfg(test)]
            write_pause: std::sync::Mutex::new(None),
        })
    }

    #[cfg(test)]
    fn with_ports(
        cloud: Arc<dyn CloudPort>,
        credentials: Arc<dyn CredentialPort>,
        cache: Arc<dyn CachePort>,
    ) -> Self {
        Self {
            cloud,
            credentials,
            cache,
            pairing: Mutex::new(None),
            access_session: RwLock::new(None),
            authorization: AuthorizationGate::new(),
            poll_singleflight: Mutex::new(()),
            refresh_singleflight: Mutex::new(()),
            cache_commit: Mutex::new(()),
            write_pause: std::sync::Mutex::new(None),
        }
    }

    #[cfg(test)]
    async fn pause_at_write(&self, point: TestWritePoint) {
        let pause = self
            .write_pause
            .lock()
            .unwrap()
            .as_ref()
            .filter(|pause| pause.point == point)
            .cloned();
        if let Some(pause) = pause {
            pause.entered.wait().await;
            pause.release.notified().await;
        }
    }

    pub async fn snapshot(&self, display_timezone: &str) -> AppResult<ViewerSnapshot> {
        let generation = self.authorization.current();
        let connected = match self.reconcile_stored_refresh_state(generation).await {
            Ok(connected) => connected,
            Err(error) if error.is_authorization_invalid() => {
                let _ = self.invalidate_authorization(generation).await;
                return Ok(ViewerSnapshot::without_data(
                    "auth-invalid",
                    Some(error.code()),
                    Some(error.to_string()),
                ));
            }
            Err(error) => return Err(error),
        };
        if !connected {
            return Ok(ViewerSnapshot::disconnected());
        }
        let Some(cache_id) = self.credentials.cache_id()? else {
            return Ok(ViewerSnapshot::without_data("loading", None, None));
        };
        match self.cache.load(cache_id.as_str(), display_timezone)? {
            Some(cache) => {
                let fresh = cache_is_fresh(&cache, Utc::now())
                    && cache.overview.get("freshness").and_then(Value::as_str) == Some("fresh");
                Ok(ViewerSnapshot::from_cache(
                    if fresh { "ready" } else { "stale" },
                    cache,
                    "cache",
                    None,
                    None,
                ))
            }
            None => Ok(ViewerSnapshot::without_data("loading", None, None)),
        }
    }

    async fn reconcile_stored_refresh_state(&self, generation: u64) -> AppResult<bool> {
        let _guard = self.authorization.lock_current(generation).await?;
        let refresh_token = self.credentials.refresh_token()?;
        let pending = self.credentials.pending_refresh_rotation()?;
        match (refresh_token, pending) {
            (None, None) => Ok(false),
            (Some(_), None) => Ok(true),
            (Some(refresh_token), Some(pending))
                if pending.matches_base_token(refresh_token.as_str()) =>
            {
                Ok(true)
            }
            (Some(refresh_token), Some(pending))
                if pending.matches_candidate_token(refresh_token.as_str()) =>
            {
                self.credentials.clear_pending_refresh_rotation()?;
                Ok(true)
            }
            _ => Err(refresh_protocol_invalid()),
        }
    }

    pub async fn start_pairing(&self, app: &AppHandle) -> AppResult<PairingState> {
        self.start_pairing_with_open(|url| {
            app.opener()
                .open_url(url, None::<&str>)
                .map_err(|_| AppError::Message("failed to open the authorization page".into()))
        })
        .await
    }

    async fn start_pairing_with_open<F>(&self, open_page: F) -> AppResult<PairingState>
    where
        F: FnOnce(&str) -> AppResult<()>,
    {
        // Invalidate older pairing work before waiting for any lock or network request.
        let generation = self.authorization.begin_transition();
        {
            let _guard = self.authorization.lock_current(generation).await?;
            *self.pairing.lock().await = None;
        }
        let installation_id = self.credentials.installation_id()?;
        let device_name = hostname::get()
            .ok()
            .and_then(|value| value.into_string().ok())
            .filter(|value| !value.trim().is_empty())
            .unwrap_or_else(|| "落雪额度查看器".into());
        let context = self
            .cloud
            .create_pairing(&device_name, installation_id.as_str(), "")
            .await?;
        let state = PairingState::waiting(&context);
        let _guard = self.authorization.lock_current(generation).await?;
        #[cfg(test)]
        self.pause_at_write(TestWritePoint::PairingContext).await;
        *self.pairing.lock().await = Some(context.clone());
        if let Err(error) = self.authorization.require_current(generation) {
            *self.pairing.lock().await = None;
            return Err(error);
        }
        if let Err(error) = open_page(context.verification_uri.as_str()) {
            *self.pairing.lock().await = None;
            return Err(error);
        }
        Ok(state)
    }

    pub async fn open_pairing_page(&self, app: &AppHandle) -> AppResult<()> {
        let generation = self.authorization.current();
        let _guard = self.authorization.lock_current(generation).await?;
        let url = self
            .pairing
            .lock()
            .await
            .as_ref()
            .map(|context| context.verification_uri.clone())
            .ok_or_else(|| AppError::Message("start quota authorization first".into()))?;
        app.opener()
            .open_url(&url, None::<&str>)
            .map_err(|_| AppError::Message("failed to open the authorization page".into()))
    }

    pub async fn cancel_pairing(&self) -> AppResult<()> {
        // This is deliberately synchronous: an in-flight network request cannot delay
        // the user's cancellation intent.
        self.authorization.begin_transition();
        self.clear_authorization_state().await
    }

    pub async fn poll_pairing(&self) -> AppResult<PairingState> {
        let generation = self.authorization.current();
        let _poll_guard = self.poll_singleflight.lock().await;
        self.authorization.require_current(generation)?;
        let context = self
            .pairing
            .lock()
            .await
            .clone()
            .ok_or_else(|| AppError::Message("start quota authorization first".into()))?;
        match self.cloud.exchange_pairing(&context).await? {
            PairingExchange::Pending => {
                self.authorization.require_current(generation)?;
                Ok(PairingState::waiting(&context))
            }
            PairingExchange::Expired => {
                let _guard = self.authorization.lock_current(generation).await?;
                *self.pairing.lock().await = None;
                Ok(PairingState::terminal("expired"))
            }
            PairingExchange::Approved(tokens) => {
                self.commit_pairing_approval(generation, &tokens).await?;
                Ok(PairingState::terminal("approved"))
            }
        }
    }

    async fn commit_pairing_approval(
        &self,
        generation: u64,
        tokens: &crate::models::TokenBundle,
    ) -> AppResult<()> {
        {
            let _guard = self.authorization.lock_current(generation).await?;
            #[cfg(test)]
            self.pause_at_write(TestWritePoint::PairingCredentials)
                .await;
            self.credentials.clear_pending_refresh_rotation()?;
            self.credentials.rotate_cache_id()?;
            self.credentials
                .set_refresh_token(tokens.refresh_token.as_str())?;
            *self.access_session.write().await = Some(AccessSession::from_bundle(tokens));
            *self.pairing.lock().await = None;
        }
        let _cache_guard = self.cache_commit.lock().await;
        self.authorization.require_current(generation)?;
        self.cache.clear()
    }

    pub async fn refresh_overview(&self, timezone: &str) -> ViewerSnapshot {
        let generation = self.authorization.current();
        match self.credentials.refresh_token() {
            Ok(Some(_)) => {}
            Ok(None) => return ViewerSnapshot::disconnected(),
            Err(error) => return self.stale_or_unavailable(error, timezone),
        }

        match self
            .fetch_overview_with_recovery(timezone, generation)
            .await
        {
            Ok(overview) => {
                if self.authorization.require_current(generation).is_err() {
                    return ViewerSnapshot::disconnected();
                }
                if let Err(error) = validate_overview(&overview, timezone) {
                    return self.stale_or_unavailable(error, timezone);
                }
                let server_fresh =
                    overview.get("freshness").and_then(Value::as_str) == Some("fresh");
                let live_cache = CachedOverview {
                    cache_id: String::new(),
                    display_timezone: timezone.to_string(),
                    overview: overview.clone(),
                    fetched_at: Utc::now(),
                };
                let cache_id = match self.ensure_cache_id(generation).await {
                    Ok(value) => value,
                    Err(_) if self.authorization.require_current(generation).is_err() => {
                        return ViewerSnapshot::disconnected()
                    }
                    Err(error) => {
                        return ViewerSnapshot::from_cache(
                            if server_fresh { "ready" } else { "stale" },
                            live_cache,
                            "network",
                            Some(error.code()),
                            Some(error.to_string()),
                        )
                    }
                };
                match self
                    .commit_overview_cache(generation, overview, cache_id.as_str(), timezone)
                    .await
                {
                    Ok(cache) => ViewerSnapshot::from_cache(
                        if server_fresh { "ready" } else { "stale" },
                        cache,
                        "network",
                        None,
                        None,
                    ),
                    Err(_) if self.authorization.require_current(generation).is_err() => {
                        ViewerSnapshot::disconnected()
                    }
                    Err(error) => ViewerSnapshot::from_cache(
                        if server_fresh { "ready" } else { "stale" },
                        live_cache,
                        "network",
                        Some(error.code()),
                        Some(error.to_string()),
                    ),
                }
            }
            Err(error) if error.is_authorization_invalid() => {
                if !self.invalidate_authorization(generation).await {
                    return ViewerSnapshot::disconnected();
                }
                ViewerSnapshot::without_data(
                    "auth-invalid",
                    Some(error.code()),
                    Some(error.to_string()),
                )
            }
            Err(error) => {
                if self.authorization.require_current(generation).is_err() {
                    ViewerSnapshot::disconnected()
                } else {
                    self.stale_or_unavailable(error, timezone)
                }
            }
        }
    }

    async fn ensure_cache_id(&self, generation: u64) -> AppResult<Zeroizing<String>> {
        let _guard = self.authorization.lock_current(generation).await?;
        match self.credentials.cache_id()? {
            Some(value) => Ok(value),
            None => self.credentials.rotate_cache_id(),
        }
    }

    async fn commit_overview_cache(
        &self,
        generation: u64,
        overview: Value,
        cache_id: &str,
        timezone: &str,
    ) -> AppResult<CachedOverview> {
        let _guard = self.cache_commit.lock().await;
        self.authorization.require_current(generation)?;
        #[cfg(test)]
        self.pause_at_write(TestWritePoint::OverviewCache).await;
        self.cache.save(overview, cache_id, timezone)
    }

    pub async fn disconnect(&self) -> AppResult<()> {
        // Signal invalidation before any contended lock so network work becomes stale
        // immediately, then wait only for short local commit sections to finish.
        self.authorization.begin_transition();
        self.clear_authorization_state().await
    }

    async fn fetch_overview_with_recovery(
        &self,
        timezone: &str,
        generation: u64,
    ) -> AppResult<Value> {
        let token = self.access_token(generation).await?;
        match self.cloud.overview(token.as_str(), timezone).await {
            Err(error) if error.is_access_token_recoverable() => {
                {
                    let _guard = self.authorization.lock_current(generation).await?;
                    let mut current = self.access_session.write().await;
                    if current
                        .as_ref()
                        .is_some_and(|session| session.token.as_str() == token.as_str())
                    {
                        *current = None;
                    }
                }
                let refreshed = self.access_token(generation).await?;
                self.cloud.overview(refreshed.as_str(), timezone).await
            }
            result => result,
        }
    }

    async fn access_token(&self, generation: u64) -> AppResult<Zeroizing<String>> {
        self.authorization.require_current(generation)?;
        if let Some(session) = self
            .access_session
            .read()
            .await
            .as_ref()
            .filter(|session| session.is_usable())
        {
            return Ok(session.token.clone());
        }

        // Only one caller may rotate a refresh token at a time. This lock is
        // intentionally independent from the short authorization commit lock.
        let _refresh_guard = self.refresh_singleflight.lock().await;
        self.authorization.require_current(generation)?;
        if let Some(session) = self
            .access_session
            .read()
            .await
            .as_ref()
            .filter(|session| session.is_usable())
        {
            return Ok(session.token.clone());
        }

        let (refresh_token, pending) = self.prepare_refresh_rotation(generation).await?;
        let bundle = self
            .cloud
            .refresh_session(refresh_token.as_str(), &pending)
            .await?;
        if !pending.validates_response(&bundle) {
            return Err(refresh_protocol_invalid());
        }
        let token = {
            let _guard = self.authorization.lock_current(generation).await?;
            #[cfg(test)]
            self.pause_at_write(TestWritePoint::RefreshedCredentials)
                .await;
            self.credentials
                .set_refresh_token(pending.candidate_refresh_token.as_str())?;
            self.credentials.clear_pending_refresh_rotation()?;
            let token = bundle.access_token.clone();
            *self.access_session.write().await = Some(AccessSession::from_refresh_bundle(&bundle));
            token
        };
        Ok(token)
    }

    async fn prepare_refresh_rotation(
        &self,
        generation: u64,
    ) -> AppResult<(Zeroizing<String>, PendingRefreshRotation)> {
        let _guard = self.authorization.lock_current(generation).await?;
        let refresh_token = self.credentials.require_refresh_token()?;
        let pending = match self.credentials.pending_refresh_rotation()? {
            Some(pending) if pending.matches_base_token(refresh_token.as_str()) => pending,
            Some(pending) if pending.matches_candidate_token(refresh_token.as_str()) => {
                // The process previously promoted the candidate but exited before
                // deleting the journal. Reconcile that committed local state before
                // beginning a new, independently recoverable rotation.
                self.credentials.clear_pending_refresh_rotation()?;
                let next = PendingRefreshRotation::generate(refresh_token.as_str());
                self.credentials.set_pending_refresh_rotation(&next)?;
                next
            }
            Some(_) => return Err(refresh_protocol_invalid()),
            None => {
                let pending = PendingRefreshRotation::generate(refresh_token.as_str());
                self.credentials.set_pending_refresh_rotation(&pending)?;
                pending
            }
        };
        Ok((refresh_token, pending))
    }

    fn stale_or_unavailable(&self, error: AppError, display_timezone: &str) -> ViewerSnapshot {
        let cache_id = match self.credentials.cache_id() {
            Ok(Some(value)) => value,
            Ok(None) => {
                return ViewerSnapshot::without_data(
                    "unavailable",
                    Some(error.code()),
                    Some(error.to_string()),
                )
            }
            Err(cache_error) => {
                return ViewerSnapshot::without_data(
                    "unavailable",
                    Some(cache_error.code()),
                    Some(cache_error.to_string()),
                )
            }
        };
        match self.cache.load(cache_id.as_str(), display_timezone) {
            Ok(Some(cache)) => ViewerSnapshot::from_cache(
                "stale",
                cache,
                "cache",
                Some(error.code()),
                Some(error.to_string()),
            ),
            Ok(None) => ViewerSnapshot::without_data(
                "unavailable",
                Some(error.code()),
                Some(error.to_string()),
            ),
            Err(cache_error) => ViewerSnapshot::without_data(
                "unavailable",
                Some(cache_error.code()),
                Some(cache_error.to_string()),
            ),
        }
    }

    async fn invalidate_authorization(&self, generation: u64) -> bool {
        if self
            .authorization
            .begin_transition_if_current(generation)
            .is_none()
        {
            return false;
        }
        let _ = self.clear_authorization_state().await;
        true
    }

    async fn clear_authorization_state(&self) -> AppResult<()> {
        let mut first_error = None;
        {
            let _guard = self.authorization.lock_commit().await;
            if let Err(error) = self.credentials.clear_refresh_token() {
                first_error = Some(error);
            }
            if let Err(error) = self.credentials.clear_pending_refresh_rotation() {
                if first_error.is_none() {
                    first_error = Some(error);
                }
            }
            if let Err(error) = self.credentials.clear_cache_id() {
                if first_error.is_none() {
                    first_error = Some(error);
                }
            }
            *self.access_session.write().await = None;
            *self.pairing.lock().await = None;
        }
        {
            let _guard = self.cache_commit.lock().await;
            if let Err(error) = self.cache.clear() {
                if first_error.is_none() {
                    first_error = Some(error);
                }
            }
        }
        match first_error {
            Some(error) => Err(error),
            None => Ok(()),
        }
    }
}

fn refresh_protocol_invalid() -> AppError {
    AppError::RefreshStateInvalid
}

fn validate_overview(value: &Value, expected_display_timezone: &str) -> AppResult<()> {
    let valid = value.get("schema_version").and_then(Value::as_i64) == Some(1)
        && value.get("generated_at").and_then(Value::as_str).is_some()
        && value.get("fresh_until").and_then(Value::as_str).is_some()
        && value.get("display_timezone").and_then(Value::as_str) == Some(expected_display_timezone)
        && value.get("account").and_then(Value::as_object).is_some()
        && value.get("wallet").and_then(Value::as_object).is_some()
        && value
            .get("subscriptions")
            .and_then(Value::as_array)
            .is_some();
    if !valid {
        return Err(AppError::Message(
            "quota API returned an unsupported snapshot".into(),
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use std::{
        collections::VecDeque,
        sync::{
            atomic::{AtomicUsize, Ordering as AtomicOrdering},
            Arc, Mutex as StdMutex,
        },
        time::Duration,
    };

    use chrono::Duration as ChronoDuration;
    use tokio::sync::{Barrier, Notify};

    use super::*;

    #[derive(Clone)]
    struct Pause {
        entered: Arc<Barrier>,
        release: Arc<Notify>,
    }

    impl Pause {
        fn new() -> Self {
            Self {
                entered: Arc::new(Barrier::new(2)),
                release: Arc::new(Notify::new()),
            }
        }

        async fn block(&self) {
            self.entered.wait().await;
            self.release.notified().await;
        }

        async fn wait_until_entered(&self) {
            self.entered.wait().await;
        }

        fn resume(&self) {
            self.release.notify_one();
        }
    }

    #[derive(Clone, Debug, PartialEq, Eq)]
    struct RefreshRequestRecord {
        refresh_token: String,
        rotation_id: String,
        candidate_refresh_token: String,
        base_token_sha256: String,
    }

    enum TestRefreshOutcome {
        Success,
        ServiceUnavailable,
        Superseded,
        RecoveryExpired,
        AuthInvalid,
        MismatchedEcho,
    }

    #[derive(Default)]
    struct TestCloud {
        create_pause: Option<Pause>,
        exchange_pause: Option<Pause>,
        refresh_pause: Option<Pause>,
        overview_pause: Option<Pause>,
        exchange_calls: AtomicUsize,
        refresh_calls: AtomicUsize,
        overview_calls: AtomicUsize,
        refresh_requests: StdMutex<Vec<RefreshRequestRecord>>,
        refresh_outcomes: StdMutex<VecDeque<TestRefreshOutcome>>,
    }

    impl CloudPort for TestCloud {
        fn create_pairing<'a>(
            &'a self,
            _device_name: &'a str,
            _installation_id: &'a str,
            _os_version: &'a str,
        ) -> CloudFuture<'a, PairingContext> {
            Box::pin(async move {
                if let Some(pause) = &self.create_pause {
                    pause.block().await;
                }
                Ok(pairing_context())
            })
        }

        fn exchange_pairing<'a>(
            &'a self,
            _context: &'a PairingContext,
        ) -> CloudFuture<'a, PairingExchange> {
            Box::pin(async move {
                self.exchange_calls.fetch_add(1, AtomicOrdering::SeqCst);
                if let Some(pause) = &self.exchange_pause {
                    pause.block().await;
                }
                Ok(PairingExchange::Approved(token_bundle(
                    "paired-access",
                    "paired-refresh",
                )))
            })
        }

        fn refresh_session<'a>(
            &'a self,
            refresh_token: &'a str,
            pending: &'a PendingRefreshRotation,
        ) -> CloudFuture<'a, RefreshTokenBundle> {
            Box::pin(async move {
                self.refresh_calls.fetch_add(1, AtomicOrdering::SeqCst);
                self.refresh_requests
                    .lock()
                    .unwrap()
                    .push(RefreshRequestRecord {
                        refresh_token: refresh_token.into(),
                        rotation_id: pending.rotation_id.clone(),
                        candidate_refresh_token: pending
                            .candidate_refresh_token
                            .as_str()
                            .to_owned(),
                        base_token_sha256: pending.base_token_sha256.clone(),
                    });
                if let Some(pause) = &self.refresh_pause {
                    pause.block().await;
                }
                match self
                    .refresh_outcomes
                    .lock()
                    .unwrap()
                    .pop_front()
                    .unwrap_or(TestRefreshOutcome::Success)
                {
                    TestRefreshOutcome::Success => Ok(refresh_token_bundle(
                        pending,
                        "refreshed-access",
                        crate::models::REFRESH_ROTATION_COMMITTED,
                    )),
                    TestRefreshOutcome::ServiceUnavailable => Err(AppError::CloudRejected {
                        status: 503,
                        reason: "QUOTA_REFRESH_OUTCOME_UNKNOWN".into(),
                    }),
                    TestRefreshOutcome::Superseded => Err(AppError::CloudRejected {
                        status: 401,
                        reason: "QUOTA_REFRESH_ROTATION_SUPERSEDED".into(),
                    }),
                    TestRefreshOutcome::RecoveryExpired => Err(AppError::CloudRejected {
                        status: 401,
                        reason: "QUOTA_REFRESH_RECOVERY_EXPIRED".into(),
                    }),
                    TestRefreshOutcome::AuthInvalid => Err(AppError::CloudRejected {
                        status: 401,
                        reason: "QUOTA_REFRESH_TOKEN_INVALID".into(),
                    }),
                    TestRefreshOutcome::MismatchedEcho => {
                        let mut response = refresh_token_bundle(
                            pending,
                            "refreshed-access",
                            crate::models::REFRESH_ROTATION_COMMITTED,
                        );
                        response.rotation_id = uuid::Uuid::new_v4().to_string();
                        Ok(response)
                    }
                }
            })
        }

        fn overview<'a>(
            &'a self,
            _access_token: &'a str,
            timezone: &'a str,
        ) -> CloudFuture<'a, Value> {
            Box::pin(async move {
                self.overview_calls.fetch_add(1, AtomicOrdering::SeqCst);
                if let Some(pause) = &self.overview_pause {
                    pause.block().await;
                }
                Ok(valid_overview(timezone))
            })
        }
    }

    #[derive(Default)]
    struct TestCredentialState {
        refresh_token: Option<String>,
        pending_refresh_rotation: Option<PendingRefreshRotation>,
        cache_id: Option<String>,
    }

    #[derive(Default)]
    struct TestCredentials {
        state: StdMutex<TestCredentialState>,
        rotations: AtomicUsize,
    }

    impl TestCredentials {
        fn connected() -> Self {
            Self {
                state: StdMutex::new(TestCredentialState {
                    refresh_token: Some("old-refresh".into()),
                    pending_refresh_rotation: None,
                    cache_id: Some("old-cache".into()),
                }),
                rotations: AtomicUsize::new(0),
            }
        }

        fn token(&self) -> Option<String> {
            self.state.lock().unwrap().refresh_token.clone()
        }

        fn current_cache_id(&self) -> Option<String> {
            self.state.lock().unwrap().cache_id.clone()
        }

        fn pending(&self) -> Option<PendingRefreshRotation> {
            self.state.lock().unwrap().pending_refresh_rotation.clone()
        }
    }

    impl CredentialPort for TestCredentials {
        fn refresh_token(&self) -> AppResult<Option<Zeroizing<String>>> {
            Ok(self
                .state
                .lock()
                .unwrap()
                .refresh_token
                .clone()
                .map(Zeroizing::new))
        }

        fn require_refresh_token(&self) -> AppResult<Zeroizing<String>> {
            self.refresh_token()?
                .ok_or_else(|| AppError::Message("not connected".into()))
        }

        fn set_refresh_token(&self, token: &str) -> AppResult<()> {
            self.state.lock().unwrap().refresh_token = Some(token.into());
            Ok(())
        }

        fn clear_refresh_token(&self) -> AppResult<()> {
            self.state.lock().unwrap().refresh_token = None;
            Ok(())
        }

        fn pending_refresh_rotation(&self) -> AppResult<Option<PendingRefreshRotation>> {
            Ok(self.state.lock().unwrap().pending_refresh_rotation.clone())
        }

        fn set_pending_refresh_rotation(&self, pending: &PendingRefreshRotation) -> AppResult<()> {
            self.state.lock().unwrap().pending_refresh_rotation = Some(pending.clone());
            Ok(())
        }

        fn clear_pending_refresh_rotation(&self) -> AppResult<()> {
            self.state.lock().unwrap().pending_refresh_rotation = None;
            Ok(())
        }

        fn installation_id(&self) -> AppResult<Zeroizing<String>> {
            Ok(Zeroizing::new("test-installation".into()))
        }

        fn cache_id(&self) -> AppResult<Option<Zeroizing<String>>> {
            Ok(self
                .state
                .lock()
                .unwrap()
                .cache_id
                .clone()
                .map(Zeroizing::new))
        }

        fn rotate_cache_id(&self) -> AppResult<Zeroizing<String>> {
            let id = format!(
                "test-cache-{}",
                self.rotations.fetch_add(1, AtomicOrdering::SeqCst) + 1
            );
            self.state.lock().unwrap().cache_id = Some(id.clone());
            Ok(Zeroizing::new(id))
        }

        fn clear_cache_id(&self) -> AppResult<()> {
            self.state.lock().unwrap().cache_id = None;
            Ok(())
        }
    }

    #[derive(Default)]
    struct TestCache {
        value: StdMutex<Option<CachedOverview>>,
        saves: AtomicUsize,
    }

    impl TestCache {
        fn populated() -> Self {
            Self {
                value: StdMutex::new(Some(CachedOverview {
                    cache_id: "old-cache".into(),
                    display_timezone: "Asia/Shanghai".into(),
                    overview: valid_overview("Asia/Shanghai"),
                    fetched_at: Utc::now(),
                })),
                saves: AtomicUsize::new(0),
            }
        }

        fn is_empty(&self) -> bool {
            self.value.lock().unwrap().is_none()
        }
    }

    impl CachePort for TestCache {
        fn load(
            &self,
            expected_cache_id: &str,
            expected_display_timezone: &str,
        ) -> AppResult<Option<CachedOverview>> {
            Ok(self.value.lock().unwrap().clone().filter(|cache| {
                cache.cache_id == expected_cache_id
                    && cache.display_timezone == expected_display_timezone
            }))
        }

        fn save(
            &self,
            overview: Value,
            cache_id: &str,
            display_timezone: &str,
        ) -> AppResult<CachedOverview> {
            self.saves.fetch_add(1, AtomicOrdering::SeqCst);
            let cache = CachedOverview {
                cache_id: cache_id.into(),
                display_timezone: display_timezone.into(),
                overview,
                fetched_at: Utc::now(),
            };
            *self.value.lock().unwrap() = Some(cache.clone());
            Ok(cache)
        }

        fn clear(&self) -> AppResult<()> {
            *self.value.lock().unwrap() = None;
            Ok(())
        }
    }

    fn pairing_context() -> PairingContext {
        PairingContext {
            device_code: Zeroizing::new("device-code".into()),
            verifier: Zeroizing::new("verifier".into()),
            user_code: "ABCD-EFGH".into(),
            verification_uri: "https://luoxueapi.cc/quota-viewer/authorize".into(),
            expires_at: Utc::now() + ChronoDuration::minutes(5),
            interval: 5,
        }
    }

    fn token_bundle(access: &str, refresh: &str) -> crate::models::TokenBundle {
        crate::models::TokenBundle {
            access_token: Zeroizing::new(access.into()),
            refresh_token: Zeroizing::new(refresh.into()),
            expires_in: 300,
        }
    }

    fn refresh_token_bundle(
        pending: &PendingRefreshRotation,
        access: &str,
        result: &str,
    ) -> RefreshTokenBundle {
        RefreshTokenBundle {
            access_token: Zeroizing::new(access.into()),
            refresh_token: pending.candidate_refresh_token.clone(),
            expires_in: 300,
            refresh_protocol: crate::models::REFRESH_PROTOCOL_CANDIDATE_V1.into(),
            rotation_id: pending.rotation_id.clone(),
            rotation_result: result.into(),
        }
    }

    fn valid_overview(timezone: &str) -> Value {
        serde_json::json!({
            "schema_version": 1,
            "generated_at": "2026-07-30T00:00:00Z",
            "fresh_until": "2026-07-30T00:05:00Z",
            "display_timezone": timezone,
            "freshness": "fresh",
            "account": {},
            "wallet": {},
            "subscriptions": []
        })
    }

    fn runtime_with(
        cloud: Arc<TestCloud>,
        credentials: Arc<TestCredentials>,
        cache: Arc<TestCache>,
    ) -> Arc<AppRuntime> {
        Arc::new(AppRuntime::with_ports(cloud, credentials, cache))
    }

    fn pause_write(runtime: &AppRuntime, point: TestWritePoint) -> Pause {
        let pause = Pause::new();
        *runtime.write_pause.lock().unwrap() = Some(Arc::new(TestWritePause {
            point,
            entered: Arc::clone(&pause.entered),
            release: Arc::clone(&pause.release),
        }));
        pause
    }

    async fn wait_for_generation_change(runtime: &AppRuntime, previous: u64) {
        tokio::time::timeout(Duration::from_millis(100), async {
            while runtime.authorization.current() == previous {
                tokio::task::yield_now().await;
            }
        })
        .await
        .expect("transition must publish its generation before waiting for commit");
    }

    #[test]
    fn overview_validation_accepts_v1_and_rejects_partial_payloads() {
        let overview = serde_json::json!({
            "schema_version": 1,
            "generated_at": "2026-07-30T00:00:00Z",
            "fresh_until": "2026-07-30T00:05:00Z",
            "display_timezone": "Asia/Shanghai",
            "account": {},
            "wallet": {},
            "subscriptions": []
        });
        assert!(validate_overview(&overview, "Asia/Shanghai").is_ok());
        assert!(validate_overview(&overview, "UTC").is_err());
        assert!(validate_overview(
            &serde_json::json!({
                "schema_version": 1,
                "wallet": {}
            }),
            "Asia/Shanghai"
        )
        .is_err());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn generation_transition_is_visible_before_commit_lock_is_available() {
        let authorization = Arc::new(AuthorizationGate::new());
        let generation = authorization.current();
        let commit = authorization.lock_current(generation).await.unwrap();

        let next = authorization.begin_transition();
        assert_eq!(next, generation + 1);
        assert_eq!(authorization.current(), generation + 1);
        drop(commit);
    }

    #[tokio::test(flavor = "current_thread")]
    async fn cancel_prevents_a_stale_start_from_opening_or_storing_pairing() {
        let pause = Pause::new();
        let cloud = Arc::new(TestCloud {
            create_pause: Some(pause.clone()),
            ..Default::default()
        });
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(cloud, Arc::clone(&credentials), Arc::clone(&cache));
        let opened = Arc::new(AtomicUsize::new(0));

        let start_runtime = Arc::clone(&runtime);
        let start_opened = Arc::clone(&opened);
        let start = tokio::spawn(async move {
            start_runtime
                .start_pairing_with_open(move |_| {
                    start_opened.fetch_add(1, AtomicOrdering::SeqCst);
                    Ok(())
                })
                .await
        });
        pause.wait_until_entered().await;

        tokio::time::timeout(Duration::from_millis(100), runtime.cancel_pairing())
            .await
            .expect("cancel must not wait for create-pairing network I/O")
            .unwrap();
        pause.resume();

        assert!(start.await.unwrap().is_err());
        assert_eq!(opened.load(AtomicOrdering::SeqCst), 0);
        assert!(runtime.pairing.lock().await.is_none());
        assert!(credentials.token().is_none());
        assert!(credentials.current_cache_id().is_none());
        assert!(cache.is_empty());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn cancel_during_pairing_context_commit_prevents_the_page_from_opening() {
        let cloud = Arc::new(TestCloud::default());
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(cloud, Arc::clone(&credentials), Arc::clone(&cache));
        let write_pause = pause_write(&runtime, TestWritePoint::PairingContext);
        let opened = Arc::new(AtomicUsize::new(0));
        let generation = runtime.authorization.current();

        let start_runtime = Arc::clone(&runtime);
        let start_opened = Arc::clone(&opened);
        let start = tokio::spawn(async move {
            start_runtime
                .start_pairing_with_open(move |_| {
                    start_opened.fetch_add(1, AtomicOrdering::SeqCst);
                    Ok(())
                })
                .await
        });
        write_pause.wait_until_entered().await;

        let cancel_runtime = Arc::clone(&runtime);
        let cancel = tokio::spawn(async move { cancel_runtime.cancel_pairing().await });
        wait_for_generation_change(&runtime, generation + 1).await;
        write_pause.resume();

        assert!(start.await.unwrap().is_err());
        cancel.await.unwrap().unwrap();
        assert_eq!(opened.load(AtomicOrdering::SeqCst), 0);
        assert!(runtime.pairing.lock().await.is_none());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn cancel_cleans_credentials_before_a_stale_poll_can_commit() {
        let pause = Pause::new();
        let cloud = Arc::new(TestCloud {
            exchange_pause: Some(pause.clone()),
            ..Default::default()
        });
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(cloud, Arc::clone(&credentials), Arc::clone(&cache));
        *runtime.pairing.lock().await = Some(pairing_context());

        let poll_runtime = Arc::clone(&runtime);
        let poll = tokio::spawn(async move { poll_runtime.poll_pairing().await });
        pause.wait_until_entered().await;

        tokio::time::timeout(Duration::from_millis(100), runtime.cancel_pairing())
            .await
            .expect("cancel must not wait for pairing exchange network I/O")
            .unwrap();
        assert!(credentials.token().is_none());
        assert!(credentials.current_cache_id().is_none());
        assert!(cache.is_empty());
        pause.resume();

        assert!(poll.await.unwrap().is_err());
        assert!(credentials.token().is_none());
        assert!(runtime.access_session.read().await.is_none());
        assert!(runtime.pairing.lock().await.is_none());
        assert!(cache.is_empty());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn cancel_cleans_pairing_credentials_written_after_invalidation() {
        let cloud = Arc::new(TestCloud::default());
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(cloud, Arc::clone(&credentials), Arc::clone(&cache));
        *runtime.pairing.lock().await = Some(pairing_context());
        let write_pause = pause_write(&runtime, TestWritePoint::PairingCredentials);
        let generation = runtime.authorization.current();

        let poll_runtime = Arc::clone(&runtime);
        let poll = tokio::spawn(async move { poll_runtime.poll_pairing().await });
        write_pause.wait_until_entered().await;
        let cancel_runtime = Arc::clone(&runtime);
        let cancel = tokio::spawn(async move { cancel_runtime.cancel_pairing().await });
        wait_for_generation_change(&runtime, generation).await;
        write_pause.resume();

        let _ = poll.await.unwrap();
        cancel.await.unwrap().unwrap();
        assert!(credentials.token().is_none());
        assert!(credentials.current_cache_id().is_none());
        assert!(runtime.access_session.read().await.is_none());
        assert!(runtime.pairing.lock().await.is_none());
        assert!(cache.is_empty());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn pairing_poll_is_singleflight_for_concurrent_callers() {
        let pause = Pause::new();
        let cloud = Arc::new(TestCloud {
            exchange_pause: Some(pause.clone()),
            ..Default::default()
        });
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::default());
        let runtime = runtime_with(
            Arc::clone(&cloud),
            Arc::clone(&credentials),
            Arc::clone(&cache),
        );
        *runtime.pairing.lock().await = Some(pairing_context());

        let first_runtime = Arc::clone(&runtime);
        let first = tokio::spawn(async move { first_runtime.poll_pairing().await });
        pause.wait_until_entered().await;
        let second_runtime = Arc::clone(&runtime);
        let second = tokio::spawn(async move { second_runtime.poll_pairing().await });
        tokio::task::yield_now().await;
        assert_eq!(cloud.exchange_calls.load(AtomicOrdering::SeqCst), 1);

        pause.resume();
        assert_eq!(first.await.unwrap().unwrap().status, "approved");
        assert!(second.await.unwrap().is_err());
        assert_eq!(cloud.exchange_calls.load(AtomicOrdering::SeqCst), 1);
    }

    #[tokio::test(flavor = "current_thread")]
    async fn disconnect_returns_while_refresh_network_is_blocked_and_old_tokens_never_reappear() {
        let pause = Pause::new();
        let cloud = Arc::new(TestCloud {
            refresh_pause: Some(pause.clone()),
            ..Default::default()
        });
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(cloud, Arc::clone(&credentials), Arc::clone(&cache));

        let refresh_runtime = Arc::clone(&runtime);
        let refresh =
            tokio::spawn(async move { refresh_runtime.refresh_overview("Asia/Shanghai").await });
        pause.wait_until_entered().await;

        assert!(
            credentials.pending().is_some(),
            "the recovery tuple must be durable before the HTTP request"
        );
        tokio::time::timeout(Duration::from_millis(100), runtime.disconnect())
            .await
            .expect("disconnect must not wait for refresh-session network I/O")
            .unwrap();
        assert!(credentials.token().is_none());
        assert!(credentials.pending().is_none());
        assert!(credentials.current_cache_id().is_none());
        assert!(cache.is_empty());
        pause.resume();

        let snapshot = refresh.await.unwrap();
        assert_eq!(snapshot.status, "disconnected");
        assert!(credentials.token().is_none());
        assert!(credentials.pending().is_none());
        assert!(runtime.access_session.read().await.is_none());
        assert!(cache.is_empty());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn disconnect_cleans_refreshed_credentials_written_after_invalidation() {
        let cloud = Arc::new(TestCloud::default());
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(cloud, Arc::clone(&credentials), Arc::clone(&cache));
        let write_pause = pause_write(&runtime, TestWritePoint::RefreshedCredentials);
        let generation = runtime.authorization.current();

        let refresh_runtime = Arc::clone(&runtime);
        let refresh =
            tokio::spawn(async move { refresh_runtime.refresh_overview("Asia/Shanghai").await });
        write_pause.wait_until_entered().await;
        let disconnect_runtime = Arc::clone(&runtime);
        let disconnect = tokio::spawn(async move { disconnect_runtime.disconnect().await });
        wait_for_generation_change(&runtime, generation).await;
        write_pause.resume();

        let _ = refresh.await.unwrap();
        disconnect.await.unwrap().unwrap();
        assert!(credentials.token().is_none());
        assert!(credentials.current_cache_id().is_none());
        assert!(runtime.access_session.read().await.is_none());
        assert!(cache.is_empty());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn disconnect_prevents_an_inflight_overview_from_recreating_cache() {
        let pause = Pause::new();
        let cloud = Arc::new(TestCloud {
            overview_pause: Some(pause.clone()),
            ..Default::default()
        });
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(cloud, Arc::clone(&credentials), Arc::clone(&cache));
        *runtime.access_session.write().await = Some(AccessSession::from_bundle(&token_bundle(
            "usable", "unused",
        )));

        let refresh_runtime = Arc::clone(&runtime);
        let refresh =
            tokio::spawn(async move { refresh_runtime.refresh_overview("Asia/Shanghai").await });
        pause.wait_until_entered().await;
        runtime.disconnect().await.unwrap();
        pause.resume();

        assert_eq!(refresh.await.unwrap().status, "disconnected");
        assert!(cache.is_empty());
        assert_eq!(cache.saves.load(AtomicOrdering::SeqCst), 0);
        assert!(credentials.token().is_none());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn disconnect_clears_a_cache_written_after_invalidation() {
        let cloud = Arc::new(TestCloud::default());
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(cloud, Arc::clone(&credentials), Arc::clone(&cache));
        *runtime.access_session.write().await = Some(AccessSession::from_bundle(&token_bundle(
            "usable", "unused",
        )));
        let write_pause = pause_write(&runtime, TestWritePoint::OverviewCache);
        let generation = runtime.authorization.current();

        let refresh_runtime = Arc::clone(&runtime);
        let refresh =
            tokio::spawn(async move { refresh_runtime.refresh_overview("Asia/Shanghai").await });
        write_pause.wait_until_entered().await;
        let disconnect_runtime = Arc::clone(&runtime);
        let disconnect = tokio::spawn(async move { disconnect_runtime.disconnect().await });
        wait_for_generation_change(&runtime, generation).await;
        write_pause.resume();

        let _ = refresh.await.unwrap();
        disconnect.await.unwrap().unwrap();
        assert_eq!(cache.saves.load(AtomicOrdering::SeqCst), 1);
        assert!(cache.is_empty());
        assert!(credentials.token().is_none());
        assert!(credentials.current_cache_id().is_none());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn refresh_session_is_singleflight_for_concurrent_callers() {
        let pause = Pause::new();
        let cloud = Arc::new(TestCloud {
            refresh_pause: Some(pause.clone()),
            ..Default::default()
        });
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::default());
        let runtime = runtime_with(
            Arc::clone(&cloud),
            Arc::clone(&credentials),
            Arc::clone(&cache),
        );
        let generation = runtime.authorization.current();

        let first_runtime = Arc::clone(&runtime);
        let first =
            tokio::spawn(async move { first_runtime.access_token(generation).await.unwrap() });
        pause.wait_until_entered().await;
        let second_runtime = Arc::clone(&runtime);
        let second =
            tokio::spawn(async move { second_runtime.access_token(generation).await.unwrap() });
        tokio::task::yield_now().await;
        assert_eq!(cloud.refresh_calls.load(AtomicOrdering::SeqCst), 1);

        pause.resume();
        assert_eq!(first.await.unwrap().as_str(), "refreshed-access");
        assert_eq!(second.await.unwrap().as_str(), "refreshed-access");
        assert_eq!(cloud.refresh_calls.load(AtomicOrdering::SeqCst), 1);
        let requests = cloud.refresh_requests.lock().unwrap();
        assert_eq!(requests.len(), 1);
        assert_eq!(
            credentials.token().as_deref(),
            Some(requests[0].candidate_refresh_token.as_str())
        );
        assert!(credentials.pending().is_none());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn retry_after_unknown_outcome_reuses_the_exact_persisted_rotation_tuple() {
        let cloud = Arc::new(TestCloud::default());
        cloud.refresh_outcomes.lock().unwrap().extend([
            TestRefreshOutcome::ServiceUnavailable,
            TestRefreshOutcome::Success,
        ]);
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::default());
        let runtime = runtime_with(
            Arc::clone(&cloud),
            Arc::clone(&credentials),
            Arc::clone(&cache),
        );
        let generation = runtime.authorization.current();

        let first = runtime.access_token(generation).await;
        assert!(first.is_err());
        let pending_after_failure = credentials
            .pending()
            .expect("unknown outcome must retain its recovery tuple");
        assert_eq!(credentials.token().as_deref(), Some("old-refresh"));

        let access = runtime.access_token(generation).await.unwrap();
        assert_eq!(access.as_str(), "refreshed-access");
        let requests = cloud.refresh_requests.lock().unwrap();
        assert_eq!(requests.len(), 2);
        assert_eq!(requests[0], requests[1]);
        assert_eq!(requests[0].rotation_id, pending_after_failure.rotation_id);
        assert_eq!(
            requests[0].candidate_refresh_token,
            pending_after_failure.candidate_refresh_token.as_str()
        );
        assert_eq!(
            credentials.token().as_deref(),
            Some(requests[0].candidate_refresh_token.as_str())
        );
        assert!(credentials.pending().is_none());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn restart_reconciles_candidate_promoted_before_pending_journal_was_cleared() {
        let cloud = Arc::new(TestCloud::default());
        let credentials = Arc::new(TestCredentials::connected());
        let stale_pending = PendingRefreshRotation::generate("old-refresh");
        let promoted_candidate = stale_pending.candidate_refresh_token.as_str().to_owned();
        {
            let mut state = credentials.state.lock().unwrap();
            state.refresh_token = Some(promoted_candidate.clone());
            state.pending_refresh_rotation = Some(stale_pending.clone());
        }
        let runtime = runtime_with(
            Arc::clone(&cloud),
            Arc::clone(&credentials),
            Arc::new(TestCache::default()),
        );

        runtime
            .access_token(runtime.authorization.current())
            .await
            .unwrap();

        let requests = cloud.refresh_requests.lock().unwrap();
        assert_eq!(requests.len(), 1);
        assert_eq!(requests[0].refresh_token, promoted_candidate);
        assert_ne!(requests[0].rotation_id, stale_pending.rotation_id);
        assert_ne!(
            requests[0].candidate_refresh_token,
            stale_pending.candidate_refresh_token.as_str()
        );
        assert_eq!(
            credentials.token().as_deref(),
            Some(requests[0].candidate_refresh_token.as_str())
        );
        assert!(credentials.pending().is_none());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn startup_snapshot_clears_a_journal_after_the_candidate_was_promoted() {
        let credentials = Arc::new(TestCredentials::connected());
        let stale_pending = PendingRefreshRotation::generate("old-refresh");
        {
            let mut state = credentials.state.lock().unwrap();
            state.refresh_token = Some(stale_pending.candidate_refresh_token.as_str().to_owned());
            state.pending_refresh_rotation = Some(stale_pending);
        }
        let runtime = runtime_with(
            Arc::new(TestCloud::default()),
            Arc::clone(&credentials),
            Arc::new(TestCache::populated()),
        );

        let snapshot = runtime.snapshot("Asia/Shanghai").await.unwrap();

        assert!(matches!(snapshot.status.as_str(), "ready" | "stale"));
        assert!(snapshot.overview.is_some());
        assert!(credentials.pending().is_none());
        assert!(credentials.token().is_some());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn startup_snapshot_invalidates_an_unreconcilable_refresh_journal() {
        let credentials = Arc::new(TestCredentials::connected());
        {
            let mut state = credentials.state.lock().unwrap();
            state.refresh_token = Some("unrelated-refresh".into());
            state.pending_refresh_rotation = Some(PendingRefreshRotation::generate("old-refresh"));
        }
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(
            Arc::new(TestCloud::default()),
            Arc::clone(&credentials),
            Arc::clone(&cache),
        );

        let snapshot = runtime.snapshot("Asia/Shanghai").await.unwrap();

        assert_eq!(snapshot.status, "auth-invalid");
        assert_eq!(
            snapshot.error_code.as_deref(),
            Some("QUOTA_REFRESH_PROTOCOL_INVALID")
        );
        assert!(credentials.token().is_none());
        assert!(credentials.pending().is_none());
        assert!(cache.is_empty());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn inconsistent_active_and_pending_tokens_fail_closed_without_network_use() {
        let cloud = Arc::new(TestCloud::default());
        let credentials = Arc::new(TestCredentials::connected());
        {
            let mut state = credentials.state.lock().unwrap();
            state.refresh_token = Some("unrelated-refresh".into());
            state.pending_refresh_rotation = Some(PendingRefreshRotation::generate("old-refresh"));
        }
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(
            Arc::clone(&cloud),
            Arc::clone(&credentials),
            Arc::clone(&cache),
        );

        let snapshot = runtime.refresh_overview("Asia/Shanghai").await;

        assert_eq!(snapshot.status, "auth-invalid");
        assert_eq!(
            snapshot.error_code.as_deref(),
            Some("QUOTA_REFRESH_PROTOCOL_INVALID")
        );
        assert_eq!(cloud.refresh_calls.load(AtomicOrdering::SeqCst), 0);
        assert!(credentials.token().is_none());
        assert!(credentials.pending().is_none());
        assert!(cache.is_empty());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn mismatched_server_echo_never_promotes_the_candidate() {
        let cloud = Arc::new(TestCloud::default());
        cloud
            .refresh_outcomes
            .lock()
            .unwrap()
            .push_back(TestRefreshOutcome::MismatchedEcho);
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(
            Arc::clone(&cloud),
            Arc::clone(&credentials),
            Arc::clone(&cache),
        );

        let snapshot = runtime.refresh_overview("Asia/Shanghai").await;

        assert_eq!(snapshot.status, "auth-invalid");
        assert_eq!(
            snapshot.error_code.as_deref(),
            Some("QUOTA_REFRESH_PROTOCOL_INVALID")
        );
        assert_eq!(cloud.refresh_calls.load(AtomicOrdering::SeqCst), 1);
        assert!(credentials.token().is_none());
        assert!(credentials.pending().is_none());
        assert!(cache.is_empty());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn explicit_superseded_rotation_requires_reconnection_and_clears_the_journal() {
        let cloud = Arc::new(TestCloud::default());
        cloud
            .refresh_outcomes
            .lock()
            .unwrap()
            .push_back(TestRefreshOutcome::Superseded);
        let credentials = Arc::new(TestCredentials::connected());
        let cache = Arc::new(TestCache::populated());
        let runtime = runtime_with(
            Arc::clone(&cloud),
            Arc::clone(&credentials),
            Arc::clone(&cache),
        );

        let snapshot = runtime.refresh_overview("Asia/Shanghai").await;

        assert_eq!(snapshot.status, "auth-invalid");
        assert_eq!(
            snapshot.error_code.as_deref(),
            Some("QUOTA_REFRESH_ROTATION_SUPERSEDED")
        );
        assert!(credentials.token().is_none());
        assert!(credentials.pending().is_none());
        assert!(cache.is_empty());
    }

    #[tokio::test(flavor = "current_thread")]
    async fn recovery_expiry_and_invalid_refresh_tokens_also_require_reconnection() {
        for (outcome, reason) in [
            (
                TestRefreshOutcome::RecoveryExpired,
                "QUOTA_REFRESH_RECOVERY_EXPIRED",
            ),
            (
                TestRefreshOutcome::AuthInvalid,
                "QUOTA_REFRESH_TOKEN_INVALID",
            ),
        ] {
            let cloud = Arc::new(TestCloud::default());
            cloud.refresh_outcomes.lock().unwrap().push_back(outcome);
            let credentials = Arc::new(TestCredentials::connected());
            let cache = Arc::new(TestCache::populated());
            let runtime = runtime_with(
                Arc::clone(&cloud),
                Arc::clone(&credentials),
                Arc::clone(&cache),
            );

            let snapshot = runtime.refresh_overview("Asia/Shanghai").await;

            assert_eq!(snapshot.status, "auth-invalid");
            assert_eq!(snapshot.error_code.as_deref(), Some(reason));
            assert!(credentials.token().is_none());
            assert!(credentials.pending().is_none());
            assert!(cache.is_empty());
        }
    }

    #[tokio::test(flavor = "current_thread")]
    async fn approved_pairing_replaces_the_account_and_clears_an_old_pending_rotation() {
        let cloud = Arc::new(TestCloud::default());
        let credentials = Arc::new(TestCredentials::connected());
        credentials
            .set_pending_refresh_rotation(&PendingRefreshRotation::generate("old-refresh"))
            .unwrap();
        let runtime = runtime_with(
            cloud,
            Arc::clone(&credentials),
            Arc::new(TestCache::default()),
        );
        *runtime.pairing.lock().await = Some(pairing_context());

        let state = runtime.poll_pairing().await.unwrap();

        assert_eq!(state.status, "approved");
        assert_eq!(credentials.token().as_deref(), Some("paired-refresh"));
        assert!(credentials.pending().is_none());
    }
}
