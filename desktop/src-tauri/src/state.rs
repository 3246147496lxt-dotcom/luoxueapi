use std::{
    collections::HashMap,
    fs::{self, OpenOptions},
    io::Write,
    path::{Path, PathBuf},
    process::Command,
    sync::{
        atomic::{AtomicBool, Ordering},
        Arc,
    },
};

use chrono::{DateTime, Duration, Utc};
use directories::{BaseDirs, ProjectDirs};
use tauri::AppHandle;
use tauri_plugin_opener::OpenerExt;
use tauri_plugin_updater::{Update, UpdaterExt};
use tokio::sync::{mpsc, Mutex, RwLock};
use zeroize::Zeroizing;

use crate::{
    cloud::{
        desktop_architecture, CloudClient, PairingContext, PairingExchange, TodayUsage, TokenBundle,
    },
    config::{ConfigManager, ConfigState},
    error::{AppError, AppResult},
    gateway::{self, GatewayDependencies, GatewayHandle, RouteSelection},
    keychain::CredentialStore,
    models::{
        CodexInstallation, DesktopSettings, DesktopSnapshot, DiagnosticCodex, DiagnosticGateway,
        DiagnosticInstallation, DiagnosticReceipt, DiagnosticRecentRequest, DiagnosticRequests,
        DiagnosticRoute, DiagnosticSummary, PairingState, PersistedState, RequestMetadata,
        RouteOption, UpdateState,
    },
    storage::LocalStorage,
};

const UPDATE_ENDPOINT_PATH: &str =
    "api/v1/desktop/releases/{{target}}/universal/{{current_version}}";

#[derive(Clone)]
pub struct AppRuntime {
    support_dir: PathBuf,
    persisted: Arc<RwLock<PersistedState>>,
    credentials: CredentialStore,
    cloud: CloudClient,
    config: ConfigManager,
    storage: LocalStorage,
    routes: Arc<RwLock<Vec<RouteOption>>>,
    selection: Arc<RwLock<RouteSelection>>,
    managed_keys: Arc<RwLock<HashMap<i64, Zeroizing<String>>>>,
    pairing: Arc<Mutex<Option<PairingContext>>>,
    access_session: Arc<RwLock<Option<AccessSession>>>,
    session_refresh: Arc<Mutex<()>>,
    usage_cache: Arc<RwLock<Option<CachedUsage>>>,
    usage_refresh: Arc<Mutex<()>>,
    update_state: Arc<RwLock<UpdateState>>,
    lifecycle: Arc<Mutex<()>>,
    gateway: Arc<Mutex<Option<GatewayHandle>>>,
    paired: Arc<AtomicBool>,
    activation_reported: Arc<AtomicBool>,
    explicit_quit: Arc<AtomicBool>,
    shutdown_started: Arc<AtomicBool>,
    shutdown_complete: Arc<AtomicBool>,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum HeartbeatOutcome {
    Idle,
    Active,
    Revoked,
}

#[derive(Clone)]
struct AccessSession {
    token: Zeroizing<String>,
    expires_at: DateTime<Utc>,
}

#[derive(Clone)]
struct CachedUsage {
    value: TodayUsage,
    fetched_at: DateTime<Utc>,
}

impl AccessSession {
    fn from_bundle(bundle: &TokenBundle) -> Self {
        Self {
            token: bundle.access_token.clone(),
            expires_at: Utc::now() + Duration::seconds(bundle.expires_in.max(1)),
        }
    }

    fn is_usable(&self) -> bool {
        self.expires_at > Utc::now() + Duration::seconds(30)
    }
}

impl AppRuntime {
    pub async fn initialize() -> AppResult<Self> {
        let project_dirs = ProjectDirs::from("cc", "luoxueapi", "desktop").ok_or_else(|| {
            AppError::Message("application support directory is unavailable".into())
        })?;
        let support_dir = project_dirs.data_dir().to_path_buf();
        fs::create_dir_all(&support_dir)?;
        let persisted = load_state(&support_dir.join("state.json"))?;
        let credentials = CredentialStore;
        let base_dirs = BaseDirs::new()
            .ok_or_else(|| AppError::Message("home directory is unavailable".into()))?;
        let config = ConfigManager::new(
            base_dirs.home_dir().join(".codex"),
            support_dir.clone(),
            credentials.clone(),
        )?;
        config.recover_incomplete()?;
        let storage = LocalStorage::open(support_dir.join("desktop.sqlite3"))?;
        let cloud = CloudClient::new()?;
        let paired = credentials.get("refresh-token")?.is_some();
        let routes = Arc::new(RwLock::new(persisted.routes.clone()));
        let selection = Arc::new(RwLock::new(RouteSelection {
            group_id: persisted.selected_group_id,
            model: persisted.selected_model.clone(),
        }));
        let mut cached_keys = HashMap::new();
        for group_id in &persisted.managed_group_ids {
            if let Some(key) = credentials.get(&CredentialStore::managed_key_account(*group_id))? {
                cached_keys.insert(*group_id, key);
            }
        }
        let runtime = Self {
            support_dir,
            persisted: Arc::new(RwLock::new(persisted)),
            credentials,
            cloud,
            config,
            storage,
            routes,
            selection,
            managed_keys: Arc::new(RwLock::new(cached_keys)),
            pairing: Arc::new(Mutex::new(None)),
            access_session: Arc::new(RwLock::new(None)),
            session_refresh: Arc::new(Mutex::new(())),
            usage_cache: Arc::new(RwLock::new(None)),
            usage_refresh: Arc::new(Mutex::new(())),
            update_state: Arc::new(RwLock::new(UpdateState::current())),
            lifecycle: Arc::new(Mutex::new(())),
            gateway: Arc::new(Mutex::new(None)),
            paired: Arc::new(AtomicBool::new(paired)),
            activation_reported: Arc::new(AtomicBool::new(false)),
            explicit_quit: Arc::new(AtomicBool::new(false)),
            shutdown_started: Arc::new(AtomicBool::new(false)),
            shutdown_complete: Arc::new(AtomicBool::new(false)),
        };
        if paired && runtime.config.is_managed() {
            if let Err(error) = runtime.start_gateway().await {
                eprintln!("failed to restart local gateway: {error}");
            }
        }
        Ok(runtime)
    }

    pub async fn snapshot(&self) -> AppResult<DesktopSnapshot> {
        let persisted = self.persisted.read().await.clone();
        let routes = self.routes.read().await.clone();
        let gateway = self.gateway.lock().await;
        let gateway_port = gateway.as_ref().map(|handle| handle.port);
        let gateway_status = if gateway.is_some() {
            "running"
        } else {
            "stopped"
        }
        .to_string();
        drop(gateway);
        let storage = self.storage.clone();
        let mut today = tokio::task::spawn_blocking(move || storage.today_summary())
            .await
            .map_err(|_| AppError::Message("usage database task failed".into()))??;
        if self.paired.load(Ordering::SeqCst) {
            if let Ok(cloud_usage) = self.cloud_usage().await {
                today.requests = cloud_usage.requests;
                today.tokens = cloud_usage.tokens;
                today.cost = cloud_usage.cost;
                today.balance = cloud_usage.balance;
            }
        }
        let (config_state, config_message) = self.config.status();
        Ok(DesktopSnapshot {
            paired: self.paired.load(Ordering::SeqCst),
            account_email: persisted.account_email,
            device_name: device_name(),
            gateway_status,
            gateway_port,
            takeover_enabled: self.config.is_managed(),
            config_status: config_state_name(&config_state).into(),
            config_message,
            selected_group_id: persisted.selected_group_id,
            selected_model: persisted.selected_model,
            routes,
            installations: detect_codex_installations(),
            today,
            settings: persisted.settings,
            update: self.update_state.read().await.clone(),
        })
    }

    pub async fn tray_menu_state(
        &self,
    ) -> (
        bool,
        bool,
        Option<u16>,
        Option<i64>,
        Option<String>,
        Vec<RouteOption>,
    ) {
        let gateway = self.gateway.lock().await;
        let running = gateway.is_some();
        let port = gateway.as_ref().map(|handle| handle.port);
        drop(gateway);
        let selection = self.selection.read().await.clone();
        (
            self.paired.load(Ordering::SeqCst),
            running,
            port,
            selection.group_id,
            selection.model,
            self.routes.read().await.clone(),
        )
    }

    pub async fn list_requests(&self) -> AppResult<Vec<RequestMetadata>> {
        let retention_days = self.persisted.read().await.settings.retention_days;
        let storage = self.storage.clone();
        tokio::task::spawn_blocking(move || storage.list_requests(500, retention_days))
            .await
            .map_err(|_| AppError::Message("request database task failed".into()))?
    }

    pub async fn diagnostic_summary(&self) -> AppResult<DiagnosticSummary> {
        let persisted = self.persisted.read().await.clone();
        let routes = self.routes.read().await.clone();
        let selection = self.selection.read().await.clone();
        let gateway_port = self.gateway.lock().await.as_ref().map(|handle| handle.port);
        let gateway_status = if gateway_port.is_some() {
            "running"
        } else {
            "stopped"
        };
        let (config_state, _) = self.config.status();
        let installations = detect_codex_installations();
        let storage = self.storage.clone();
        let retention_days = persisted.settings.retention_days;
        let recent = tokio::task::spawn_blocking(move || storage.list_requests(20, retention_days))
            .await
            .map_err(|_| AppError::Message("diagnostic database task failed".into()))??;
        let success_count = recent
            .iter()
            .filter(|request| (200..300).contains(&request.status_code))
            .count();
        let average_duration_ms = average_i64(recent.iter().map(|request| request.duration_ms));
        let average_first_token_ms =
            average_i64(recent.iter().filter_map(|request| request.first_token_ms));

        Ok(DiagnosticSummary {
            app_version: env!("CARGO_PKG_VERSION").into(),
            platform: "macos".into(),
            architecture: desktop_architecture().into(),
            os_version: macos_version(),
            gateway: DiagnosticGateway {
                status: gateway_status.into(),
                port: gateway_port,
                takeover_enabled: self.config.is_managed(),
            },
            codex: DiagnosticCodex {
                config_status: config_state_name(&config_state).into(),
                installations: installations
                    .into_iter()
                    .map(|installation| DiagnosticInstallation {
                        kind: installation.kind,
                        installed: installation.installed,
                        version: installation.version,
                    })
                    .collect(),
            },
            route: DiagnosticRoute {
                group_id: selection.group_id,
                model: selection.model,
                available_route_count: routes.len(),
            },
            requests: DiagnosticRequests {
                sample_count: recent.len(),
                success_count,
                error_count: recent.len().saturating_sub(success_count),
                average_duration_ms,
                average_first_token_ms,
                recent: recent
                    .into_iter()
                    .map(|request| DiagnosticRecentRequest {
                        occurred_at: request.occurred_at,
                        model: request.model,
                        status_code: request.status_code,
                        duration_ms: request.duration_ms,
                        first_token_ms: request.first_token_ms,
                        request_id: request.request_id,
                    })
                    .collect(),
            },
        })
    }

    pub async fn upload_diagnostic(&self) -> AppResult<DiagnosticReceipt> {
        let summary = self.diagnostic_summary().await?;
        let access_token = self.access_token().await?;
        self.cloud
            .upload_diagnostic(access_token.as_str(), &summary)
            .await
    }

    pub async fn start_pairing(&self, app: &AppHandle) -> AppResult<PairingState> {
        let installation_id = self
            .credentials
            .get_or_create_random("installation-id", 32)?;
        let os_version = macos_version();
        let context = self
            .cloud
            .create_pairing(&device_name(), installation_id.as_str(), &os_version)
            .await?;
        let state = PairingState::waiting(
            context.user_code.clone(),
            context.verification_uri.clone(),
            context.expires_at.to_rfc3339(),
        );
        app.opener()
            .open_url(&context.verification_uri, None::<&str>)
            .map_err(|error| AppError::Message(format!("failed to open pairing page: {error}")))?;
        *self.pairing.lock().await = Some(context);
        Ok(state)
    }

    pub async fn open_pairing_page(&self, app: &AppHandle) -> AppResult<()> {
        let verification_uri = self
            .pairing
            .lock()
            .await
            .as_ref()
            .map(|context| context.verification_uri.clone())
            .ok_or_else(|| AppError::Message("start device pairing first".into()))?;
        app.opener()
            .open_url(&verification_uri, None::<&str>)
            .map_err(|error| AppError::Message(format!("failed to open pairing page: {error}")))
    }

    pub async fn poll_pairing(&self) -> AppResult<PairingState> {
        let context = self
            .pairing
            .lock()
            .await
            .clone()
            .ok_or_else(|| AppError::Message("start device pairing first".into()))?;
        match self.cloud.exchange_pairing(&context).await? {
            PairingExchange::Pending => Ok(PairingState::waiting(
                context.user_code,
                context.verification_uri,
                context.expires_at.to_rfc3339(),
            )),
            PairingExchange::Expired => Ok(PairingState {
                status: "expired".into(),
                user_code: None,
                verification_uri: None,
                expires_at: None,
            }),
            PairingExchange::Approved(tokens) => {
                self.credentials
                    .set("refresh-token", tokens.refresh_token.as_str())?;
                *self.access_session.write().await = Some(AccessSession::from_bundle(&tokens));
                self.paired.store(true, Ordering::SeqCst);
                *self.pairing.lock().await = None;

                let routes = self.cloud.list_routes(tokens.access_token.as_str()).await?;
                *self.routes.write().await = routes.clone();
                {
                    let mut persisted = self.persisted.write().await;
                    persisted.selected_group_id = None;
                    persisted.selected_model = None;
                    persisted.managed_group_ids.clear();
                    persisted.routes = routes;
                    persisted.account_email = tokens.account_email;
                    self.persist_locked(&persisted)?;
                }
                Ok(PairingState {
                    status: "approved".into(),
                    user_code: None,
                    verification_uri: None,
                    expires_at: None,
                })
            }
        }
    }

    pub async fn restart_gateway(&self) -> AppResult<()> {
        self.stop_gateway().await;
        self.start_gateway().await
    }

    pub async fn enable_takeover(&self) -> AppResult<()> {
        self.start_gateway().await?;
        let port = self
            .gateway
            .lock()
            .await
            .as_ref()
            .map(|handle| handle.port)
            .ok_or_else(|| AppError::Message("local gateway did not start".into()))?;
        let model = self
            .selection
            .read()
            .await
            .model
            .clone()
            .ok_or_else(|| AppError::Message("select a default model first".into()))?;
        let local_token = self.credentials.require("local-token")?;
        let config = self.config.clone();
        tokio::task::spawn_blocking(move || config.apply(port, local_token.as_str(), &model))
            .await
            .map_err(|_| AppError::Message("configuration task failed".into()))??;
        Ok(())
    }

    pub async fn disable_takeover(&self) -> AppResult<()> {
        let config = self.config.clone();
        tokio::task::spawn_blocking(move || config.restore())
            .await
            .map_err(|_| AppError::Message("configuration restore task failed".into()))??;
        self.stop_gateway().await;
        Ok(())
    }

    pub async fn select_route(&self, group_id: i64, model: String) -> AppResult<()> {
        let route = self
            .routes
            .read()
            .await
            .iter()
            .find(|route| route.group_id == group_id)
            .cloned()
            .ok_or_else(|| AppError::Message("selected route is not available".into()))?;
        if !route.models.iter().any(|candidate| candidate == &model) {
            return Err(AppError::Message(
                "selected model is not available on this route".into(),
            ));
        }
        if !self.managed_keys.read().await.contains_key(&group_id) {
            let access_token = self.access_token().await?;
            let key = self
                .cloud
                .ensure_managed_key(access_token.as_str(), group_id)
                .await?;
            if key.group_id != group_id {
                return Err(AppError::Message("managed key route mismatch".into()));
            }
            self.credentials.set(
                &CredentialStore::managed_key_account(group_id),
                key.api_key.as_str(),
            )?;
            self.managed_keys
                .write()
                .await
                .insert(group_id, key.api_key);
        }
        *self.selection.write().await = RouteSelection {
            group_id: Some(group_id),
            model: Some(model.clone()),
        };
        {
            let mut persisted = self.persisted.write().await;
            persisted.selected_group_id = Some(group_id);
            persisted.selected_model = Some(model);
            if !persisted.managed_group_ids.contains(&group_id) {
                persisted.managed_group_ids.push(group_id);
            }
            self.persist_locked(&persisted)?;
        }
        if self.config.is_managed() {
            let config = self.config.clone();
            tokio::task::spawn_blocking(move || config.restore())
                .await
                .map_err(|_| AppError::Message("configuration update task failed".into()))??;
            self.enable_takeover().await?;
        }
        Ok(())
    }

    pub async fn update_settings(&self, settings: DesktopSettings) -> AppResult<()> {
        validate_settings(&settings)?;
        let mut persisted = self.persisted.write().await;
        persisted.settings = settings;
        self.persist_locked(&persisted)
    }

    pub async fn notifications_enabled(&self) -> bool {
        self.persisted.read().await.settings.notifications
    }

    pub async fn check_for_updates(&self, app: &AppHandle) -> AppResult<DesktopSnapshot> {
        *self.update_state.write().await = UpdateState {
            state: "checking".into(),
            current_version: env!("CARGO_PKG_VERSION").into(),
            version: None,
        };
        match self.find_update(app).await {
            Ok(Some(update)) => {
                *self.update_state.write().await = UpdateState {
                    state: "available".into(),
                    current_version: update.current_version,
                    version: Some(update.version),
                };
            }
            Ok(None) => *self.update_state.write().await = UpdateState::current(),
            Err(error) => {
                *self.update_state.write().await = UpdateState {
                    state: "error".into(),
                    current_version: env!("CARGO_PKG_VERSION").into(),
                    version: None,
                };
                return Err(error);
            }
        }
        self.snapshot().await
    }

    pub async fn install_update(&self, app: &AppHandle) -> AppResult<()> {
        let update = self
            .find_update(app)
            .await?
            .ok_or_else(|| AppError::Message("no update is currently available".into()))?;
        *self.update_state.write().await = UpdateState {
            state: "downloading".into(),
            current_version: update.current_version.clone(),
            version: Some(update.version.clone()),
        };
        if let Err(error) = update.download_and_install(|_, _| {}, || {}).await {
            *self.update_state.write().await = UpdateState {
                state: "available".into(),
                current_version: update.current_version,
                version: Some(update.version),
            };
            return Err(AppError::Message(format!(
                "update download or signature verification failed: {error}"
            )));
        }
        *self.update_state.write().await = UpdateState {
            state: "restart_required".into(),
            current_version: update.current_version,
            version: Some(update.version),
        };
        Ok(())
    }

    async fn find_update(&self, app: &AppHandle) -> AppResult<Option<Update>> {
        let public_key = option_env!("LUOXUE_UPDATE_PUBKEY")
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .ok_or_else(|| {
                AppError::Message("release update public key is not configured".into())
            })?;
        let access_token = self.access_token().await?;
        let endpoint = self.cloud.upstream_url(UPDATE_ENDPOINT_PATH)?;
        let updater = app
            .updater_builder()
            .endpoints(vec![endpoint])
            .map_err(|error| AppError::Message(format!("invalid update endpoint: {error}")))?
            .header("Authorization", format!("Bearer {}", access_token.as_str()))
            .map_err(|error| AppError::Message(format!("invalid update authorization: {error}")))?
            .pubkey(public_key)
            .build()
            .map_err(|error| AppError::Message(format!("failed to initialize updater: {error}")))?;
        updater
            .check()
            .await
            .map_err(|error| AppError::Message(format!("update check failed: {error}")))
    }

    pub async fn safe_logout(&self) -> AppResult<()> {
        let _lifecycle_guard = self.lifecycle.lock().await;
        self.disable_takeover().await?;
        match self.access_token().await {
            Ok(access_token) => {
                if let Err(error) = self
                    .cloud
                    .revoke_current_device(access_token.as_str())
                    .await
                {
                    if !error.is_desktop_session_revoked() {
                        return Err(error);
                    }
                }
            }
            Err(error) if error.is_desktop_session_revoked() => {}
            Err(error) => return Err(error),
        }
        self.clear_local_state().await
    }

    pub async fn heartbeat_once(&self) -> AppResult<HeartbeatOutcome> {
        if !self.paired.load(Ordering::SeqCst) {
            return Ok(HeartbeatOutcome::Idle);
        }

        let _lifecycle_guard = self.lifecycle.lock().await;
        if !self.paired.load(Ordering::SeqCst) {
            return Ok(HeartbeatOutcome::Idle);
        }

        let access_token = match self.access_token().await {
            Ok(token) => token,
            Err(error) if error.is_desktop_session_revoked() => {
                self.recover_remote_revocation().await?;
                return Ok(HeartbeatOutcome::Revoked);
            }
            Err(error) => return Err(error),
        };
        match self.cloud.heartbeat(access_token.as_str()).await {
            Ok(device) if device.status == "active" => Ok(HeartbeatOutcome::Active),
            Ok(_) => {
                self.recover_remote_revocation().await?;
                Ok(HeartbeatOutcome::Revoked)
            }
            Err(error) if error.is_desktop_session_revoked() => {
                self.recover_remote_revocation().await?;
                Ok(HeartbeatOutcome::Revoked)
            }
            Err(error) => Err(error),
        }
    }

    async fn recover_remote_revocation(&self) -> AppResult<()> {
        self.disable_takeover()
            .await
            .map_err(|error| AppError::RemoteRevocation(error.to_string()))?;
        self.clear_local_state()
            .await
            .map_err(|error| AppError::RemoteRevocation(error.to_string()))
    }

    async fn clear_local_state(&self) -> AppResult<()> {
        let group_ids = self.persisted.read().await.managed_group_ids.clone();
        for group_id in group_ids {
            self.credentials
                .delete(&CredentialStore::managed_key_account(group_id))?;
        }
        for account in [
            "refresh-token",
            "local-token",
            "device-secret",
            "config-backup-key",
        ] {
            self.credentials.delete(account)?;
        }
        self.managed_keys.write().await.clear();
        self.routes.write().await.clear();
        *self.selection.write().await = RouteSelection::default();
        *self.access_session.write().await = None;
        *self.usage_cache.write().await = None;
        self.paired.store(false, Ordering::SeqCst);
        self.activation_reported.store(false, Ordering::SeqCst);
        let mut persisted = self.persisted.write().await;
        *persisted = PersistedState::default();
        self.persist_locked(&persisted)?;
        let storage = self.storage.clone();
        tokio::task::spawn_blocking(move || storage.clear_private_state())
            .await
            .map_err(|_| AppError::Message("request database cleanup failed".into()))??;
        Ok(())
    }

    pub async fn start_gateway(&self) -> AppResult<()> {
        let mut gateway = self.gateway.lock().await;
        if gateway.is_some() {
            return Ok(());
        }
        let preferred_port = self.persisted.read().await.gateway_port;
        let (activation_tx, mut activation_rx) = mpsc::unbounded_channel();
        let handle = gateway::start(
            preferred_port,
            GatewayDependencies {
                local_token: self.credentials.get_or_create_random("local-token", 32)?,
                device_secret: self.credentials.get_or_create_random("device-secret", 32)?,
                cloud: self.cloud.clone(),
                storage: self.storage.clone(),
                selection: self.selection.clone(),
                routes: self.routes.clone(),
                managed_keys: self.managed_keys.clone(),
                activation_tx,
                activation_reported: self.activation_reported.clone(),
            },
        )
        .await?;
        let port = handle.port;
        *gateway = Some(handle);
        let mut persisted = self.persisted.write().await;
        persisted.gateway_port = Some(port);
        self.persist_locked(&persisted)?;
        let runtime = self.clone();
        tauri::async_runtime::spawn(async move {
            while activation_rx.recv().await.is_some() {
                let result = async {
                    let access_token = runtime.access_token().await?;
                    runtime.cloud.activate_device(access_token.as_str()).await?;
                    Ok::<_, AppError>(())
                }
                .await;
                if result.is_ok() {
                    break;
                }
                runtime.activation_reported.store(false, Ordering::SeqCst);
            }
        });
        Ok(())
    }

    async fn access_token(&self) -> AppResult<Zeroizing<String>> {
        if let Some(session) = self
            .access_session
            .read()
            .await
            .as_ref()
            .filter(|session| session.is_usable())
        {
            return Ok(session.token.clone());
        }

        let _refresh_guard = self.session_refresh.lock().await;
        if let Some(session) = self
            .access_session
            .read()
            .await
            .as_ref()
            .filter(|session| session.is_usable())
        {
            return Ok(session.token.clone());
        }

        let refresh_token = self.credentials.require("refresh-token")?;
        let bundle = self.cloud.refresh_session(refresh_token.as_str()).await?;
        self.credentials
            .set("refresh-token", bundle.refresh_token.as_str())?;
        let access_token = bundle.access_token.clone();
        *self.access_session.write().await = Some(AccessSession::from_bundle(&bundle));
        self.paired.store(true, Ordering::SeqCst);
        Ok(access_token)
    }

    async fn cloud_usage(&self) -> AppResult<TodayUsage> {
        const CACHE_TTL: Duration = Duration::seconds(60);
        if let Some(cached) = self
            .usage_cache
            .read()
            .await
            .as_ref()
            .filter(|cached| cached.fetched_at + CACHE_TTL > Utc::now())
        {
            return Ok(cached.value.clone());
        }

        let _refresh_guard = self.usage_refresh.lock().await;
        let stale = self.usage_cache.read().await.clone();
        if let Some(cached) = stale
            .as_ref()
            .filter(|cached| cached.fetched_at + CACHE_TTL > Utc::now())
        {
            return Ok(cached.value.clone());
        }

        let result = async {
            let access_token = self.access_token().await?;
            self.cloud.today_usage(access_token.as_str()).await
        }
        .await;
        match result {
            Ok(value) => {
                *self.usage_cache.write().await = Some(CachedUsage {
                    value: value.clone(),
                    fetched_at: Utc::now(),
                });
                Ok(value)
            }
            Err(_) if stale.is_some() => Ok(stale.expect("stale cache checked above").value),
            Err(error) => Err(error),
        }
    }

    pub async fn stop_gateway(&self) {
        if let Some(handle) = self.gateway.lock().await.take() {
            handle.stop();
        }
    }

    pub fn request_explicit_quit(&self) {
        self.explicit_quit.store(true, Ordering::SeqCst);
    }

    pub fn should_restore_on_exit(&self) -> bool {
        self.explicit_quit.load(Ordering::SeqCst) && !self.shutdown_complete.load(Ordering::SeqCst)
    }

    pub fn begin_shutdown(&self) -> bool {
        !self.shutdown_started.swap(true, Ordering::SeqCst)
    }

    pub async fn finish_explicit_shutdown(&self) -> AppResult<()> {
        self.disable_takeover().await?;
        self.shutdown_complete.store(true, Ordering::SeqCst);
        Ok(())
    }

    pub fn cancel_explicit_shutdown(&self) {
        self.explicit_quit.store(false, Ordering::SeqCst);
        self.shutdown_started.store(false, Ordering::SeqCst);
    }

    fn persist_locked(&self, state: &PersistedState) -> AppResult<()> {
        save_state(&self.support_dir.join("state.json"), state)
    }
}

fn average_i64(values: impl Iterator<Item = i64>) -> Option<i64> {
    let (sum, count) = values.fold((0_i128, 0_i128), |(sum, count), value| {
        (sum + i128::from(value), count + 1)
    });
    (count > 0).then(|| (sum / count) as i64)
}

fn config_state_name(state: &ConfigState) -> &'static str {
    match state {
        ConfigState::Clean => "clean",
        ConfigState::Managed => "managed",
        ConfigState::Conflict => "conflict",
        ConfigState::Invalid => "invalid",
        ConfigState::PermissionDenied => "permission_denied",
    }
}

fn validate_settings(settings: &DesktopSettings) -> AppResult<()> {
    if !matches!(settings.retention_days, 1 | 7 | 14 | 30) {
        return Err(AppError::Message(
            "unsupported request retention period".into(),
        ));
    }
    if !matches!(settings.theme.as_str(), "system" | "light" | "dark") {
        return Err(AppError::Message("unsupported theme".into()));
    }
    if !matches!(settings.locale.as_str(), "zh-CN" | "en") {
        return Err(AppError::Message("unsupported locale".into()));
    }
    Ok(())
}

fn device_name() -> String {
    hostname::get()
        .ok()
        .and_then(|name| name.into_string().ok())
        .filter(|name| !name.is_empty())
        .unwrap_or_else(|| "This Mac".into())
}

fn macos_version() -> String {
    Command::new("/usr/bin/sw_vers")
        .arg("-productVersion")
        .output()
        .ok()
        .filter(|output| output.status.success())
        .and_then(|output| String::from_utf8(output.stdout).ok())
        .map(|version| version.trim().to_string())
        .filter(|version| !version.is_empty())
        .unwrap_or_default()
}

fn detect_codex_installations() -> Vec<CodexInstallation> {
    let cli_path = std::env::var_os("PATH").and_then(|paths| {
        std::env::split_paths(&paths)
            .map(|path| path.join("codex"))
            .find(|path| path.is_file())
    });
    let desktop_path = PathBuf::from("/Applications/Codex.app");
    vec![
        CodexInstallation {
            kind: "cli".into(),
            installed: cli_path.is_some(),
            version: None,
            path: cli_path.map(|path| path.display().to_string()),
        },
        CodexInstallation {
            kind: "desktop".into(),
            installed: desktop_path.exists(),
            version: None,
            path: desktop_path
                .exists()
                .then(|| desktop_path.display().to_string()),
        },
    ]
}

fn load_state(path: &Path) -> AppResult<PersistedState> {
    match fs::read(path) {
        Ok(bytes) => Ok(serde_json::from_slice(&bytes)?),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => Ok(PersistedState::default()),
        Err(error) => Err(error.into()),
    }
}

fn save_state(path: &Path, state: &PersistedState) -> AppResult<()> {
    let parent = path
        .parent()
        .ok_or_else(|| AppError::Message("state path has no parent".into()))?;
    fs::create_dir_all(parent)?;
    let temp = parent.join(".state.json.tmp");
    let mut file = OpenOptions::new()
        .create(true)
        .truncate(true)
        .write(true)
        .open(&temp)?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        file.set_permissions(fs::Permissions::from_mode(0o600))?;
    }
    file.write_all(&serde_json::to_vec_pretty(state)?)?;
    file.sync_all()?;
    fs::rename(temp, path)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::UPDATE_ENDPOINT_PATH;

    #[test]
    fn updater_requests_the_shared_universal_macos_artifact() {
        assert!(UPDATE_ENDPOINT_PATH.contains("/{{target}}/universal/"));
        assert!(!UPDATE_ENDPOINT_PATH.contains("{{arch}}"));
    }
}
