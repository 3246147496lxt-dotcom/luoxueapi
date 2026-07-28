mod cloud;
mod config;
mod error;
mod gateway;
mod keychain;
mod models;
mod state;
mod storage;

use std::process::Command;

use models::{
    DesktopSettings, DesktopSnapshot, DiagnosticReceipt, DiagnosticSummary, PairingState,
    RequestMetadata,
};
use state::{AppRuntime, HeartbeatOutcome};
use tauri::{
    menu::{CheckMenuItem, IsMenuItem, Menu, MenuItem, PredefinedMenuItem, Submenu},
    tray::TrayIconBuilder,
    AppHandle, Emitter, Manager, RunEvent, State, WindowEvent, Wry,
};
use tauri_plugin_autostart::{MacosLauncher, ManagerExt as AutostartExt};
use tauri_plugin_notification::NotificationExt;
use tauri_plugin_opener::OpenerExt;

type CommandResult<T> = Result<T, String>;

#[tauri::command]
async fn get_snapshot(state: State<'_, AppRuntime>) -> CommandResult<DesktopSnapshot> {
    state.snapshot().await.map_err(|error| error.to_string())
}

#[tauri::command]
async fn list_requests(state: State<'_, AppRuntime>) -> CommandResult<Vec<RequestMetadata>> {
    state
        .list_requests()
        .await
        .map_err(|error| error.to_string())
}

#[tauri::command]
async fn start_pairing(
    app: AppHandle,
    state: State<'_, AppRuntime>,
) -> CommandResult<PairingState> {
    state
        .start_pairing(&app)
        .await
        .map_err(|error| error.to_string())
}

#[tauri::command]
async fn open_pairing_page(app: AppHandle, state: State<'_, AppRuntime>) -> CommandResult<()> {
    state
        .open_pairing_page(&app)
        .await
        .map_err(|error| error.to_string())
}

#[tauri::command]
async fn poll_pairing(app: AppHandle, state: State<'_, AppRuntime>) -> CommandResult<PairingState> {
    let result = state
        .poll_pairing()
        .await
        .map_err(|error| error.to_string())?;
    if result.status == "approved" {
        refresh_tray_menu(&app).await;
    }
    Ok(result)
}

#[tauri::command]
async fn restart_gateway(
    app: AppHandle,
    state: State<'_, AppRuntime>,
) -> CommandResult<DesktopSnapshot> {
    state
        .restart_gateway()
        .await
        .map_err(|error| error.to_string())?;
    refresh_tray_menu(&app).await;
    state.snapshot().await.map_err(|error| error.to_string())
}

#[tauri::command]
async fn set_takeover(
    app: AppHandle,
    state: State<'_, AppRuntime>,
    enabled: bool,
) -> CommandResult<DesktopSnapshot> {
    if enabled {
        state
            .enable_takeover()
            .await
            .map_err(|error| error.to_string())?;
        let mut settings = state
            .snapshot()
            .await
            .map_err(|error| error.to_string())?
            .settings;
        settings.launch_at_login = true;
        state
            .update_settings(settings)
            .await
            .map_err(|error| error.to_string())?;
        app.autolaunch()
            .enable()
            .map_err(|error| error.to_string())?;
    } else {
        state
            .disable_takeover()
            .await
            .map_err(|error| error.to_string())?;
    }
    refresh_tray_menu(&app).await;
    state.snapshot().await.map_err(|error| error.to_string())
}

#[tauri::command]
async fn select_route(
    app: AppHandle,
    state: State<'_, AppRuntime>,
    group_id: i64,
    model: String,
) -> CommandResult<DesktopSnapshot> {
    state
        .select_route(group_id, model)
        .await
        .map_err(|error| error.to_string())?;
    refresh_tray_menu(&app).await;
    state.snapshot().await.map_err(|error| error.to_string())
}

#[tauri::command]
async fn update_settings(
    app: AppHandle,
    state: State<'_, AppRuntime>,
    settings: DesktopSettings,
) -> CommandResult<DesktopSnapshot> {
    if settings.launch_at_login {
        app.autolaunch()
            .enable()
            .map_err(|error| error.to_string())?;
    } else {
        state
            .disable_takeover()
            .await
            .map_err(|error| error.to_string())?;
        app.autolaunch()
            .disable()
            .map_err(|error| error.to_string())?;
    }
    state
        .update_settings(settings)
        .await
        .map_err(|error| error.to_string())?;
    state.snapshot().await.map_err(|error| error.to_string())
}

#[tauri::command]
async fn open_codex() -> CommandResult<()> {
    Command::new("open")
        .arg("-a")
        .arg("Codex")
        .spawn()
        .map(|_| ())
        .map_err(|error| format!("failed to open Codex: {error}"))
}

#[tauri::command]
async fn open_account_page(app: AppHandle, destination: String) -> CommandResult<()> {
    let path = match destination.as_str() {
        "routes" => "keys",
        "balance" => "purchase",
        _ => return Err("unsupported account destination".into()),
    };
    let url = cloud::official_web_url(path).map_err(|error| error.to_string())?;
    app.opener()
        .open_url(url.as_str(), None::<&str>)
        .map_err(|error| format!("failed to open account page: {error}"))
}

#[tauri::command]
fn cli_launch_command() -> String {
    "codex".into()
}

#[tauri::command]
async fn check_for_updates(
    app: AppHandle,
    state: State<'_, AppRuntime>,
) -> CommandResult<DesktopSnapshot> {
    state
        .check_for_updates(&app)
        .await
        .map_err(|error| error.to_string())
}

#[tauri::command]
async fn install_update(app: AppHandle, state: State<'_, AppRuntime>) -> CommandResult<()> {
    state
        .install_update(&app)
        .await
        .map_err(|error| error.to_string())?;
    app.restart()
}

#[tauri::command]
async fn diagnostic_summary(state: State<'_, AppRuntime>) -> CommandResult<DiagnosticSummary> {
    state
        .diagnostic_summary()
        .await
        .map_err(|error| error.to_string())
}

#[tauri::command]
async fn upload_diagnostic(state: State<'_, AppRuntime>) -> CommandResult<DiagnosticReceipt> {
    state
        .upload_diagnostic()
        .await
        .map_err(|error| error.to_string())
}

#[tauri::command]
async fn prepare_uninstall(app: AppHandle, state: State<'_, AppRuntime>) -> CommandResult<()> {
    state
        .safe_logout()
        .await
        .map_err(|error| error.to_string())?;
    app.autolaunch()
        .disable()
        .map_err(|error| error.to_string())?;
    refresh_tray_menu(&app).await;
    Ok(())
}

#[tauri::command]
async fn logout(app: AppHandle, state: State<'_, AppRuntime>) -> CommandResult<()> {
    state
        .safe_logout()
        .await
        .map_err(|error| error.to_string())?;
    app.autolaunch()
        .disable()
        .map_err(|error| error.to_string())?;
    refresh_tray_menu(&app).await;
    Ok(())
}

fn show_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
    }
}

async fn notify_if_enabled(
    app: &AppHandle,
    runtime: &AppRuntime,
    title: &str,
    body: impl Into<String>,
) {
    if !runtime.notifications_enabled().await {
        return;
    }
    let _ = app
        .notification()
        .builder()
        .title(title)
        .body(body.into())
        .show();
}

fn request_safe_quit(app: &AppHandle) {
    app.state::<AppRuntime>().request_explicit_quit();
    app.exit(0);
}

fn build_app_menu(app: &AppHandle) -> tauri::Result<Menu<tauri::Wry>> {
    let quit = MenuItem::with_id(
        app,
        "quit_app",
        "安全退出落雪API Desktop",
        true,
        Some("CmdOrCtrl+Q"),
    )?;
    let app_menu = Submenu::with_items(
        app,
        "落雪API Desktop",
        true,
        &[
            &PredefinedMenuItem::about(app, None, None)?,
            &PredefinedMenuItem::separator(app)?,
            &PredefinedMenuItem::hide(app, None)?,
            &PredefinedMenuItem::hide_others(app, None)?,
            &PredefinedMenuItem::separator(app)?,
            &quit,
        ],
    )?;
    let edit_menu = Submenu::with_items(
        app,
        "编辑",
        true,
        &[
            &PredefinedMenuItem::undo(app, None)?,
            &PredefinedMenuItem::redo(app, None)?,
            &PredefinedMenuItem::separator(app)?,
            &PredefinedMenuItem::cut(app, None)?,
            &PredefinedMenuItem::copy(app, None)?,
            &PredefinedMenuItem::paste(app, None)?,
            &PredefinedMenuItem::select_all(app, None)?,
        ],
    )?;
    let window_menu = Submenu::with_items(
        app,
        "窗口",
        true,
        &[
            &PredefinedMenuItem::minimize(app, None)?,
            &PredefinedMenuItem::close_window(app, None)?,
        ],
    )?;
    Menu::with_items(app, &[&app_menu, &edit_menu, &window_menu])
}

fn initial_tray_menu(app: &AppHandle) -> tauri::Result<Menu<Wry>> {
    let status = MenuItem::with_id(app, "status", "落雪API Desktop", false, None::<&str>)?;
    let open = MenuItem::with_id(app, "open", "打开主窗口", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "安全退出", true, None::<&str>)?;
    Menu::with_items(app, &[&status, &open, &quit])
}

async fn build_tray_menu(app: &AppHandle) -> tauri::Result<Menu<Wry>> {
    let runtime = app.state::<AppRuntime>().inner().clone();
    let (paired, running, port, selected_group, selected_model, routes) =
        runtime.tray_menu_state().await;
    let status_text = match (paired, running, port) {
        (false, _, _) => "状态：等待连接账号".to_string(),
        (true, true, Some(port)) => format!("网关：运行中 · 127.0.0.1:{port}"),
        (true, true, None) => "网关：运行中".to_string(),
        (true, false, _) => "网关：已停止".to_string(),
    };
    let status = MenuItem::with_id(app, "status", status_text, false, None::<&str>)?;

    let mut model_items: Vec<CheckMenuItem<Wry>> = Vec::new();
    for route in &routes {
        for model in &route.models {
            let id = format!("tray_model:{}:{}", route.group_id, model);
            let label = format!("{} · {}", model, route.name).replace('&', "&&");
            model_items.push(CheckMenuItem::with_id(
                app,
                id,
                label,
                paired,
                selected_group == Some(route.group_id)
                    && selected_model.as_deref() == Some(model.as_str()),
                None::<&str>,
            )?);
        }
    }
    if model_items.is_empty() {
        model_items.push(CheckMenuItem::with_id(
            app,
            "tray_model_unavailable",
            "暂无可用模型",
            false,
            false,
            None::<&str>,
        )?);
    }
    let model_refs: Vec<&dyn IsMenuItem<Wry>> = model_items
        .iter()
        .map(|item| item as &dyn IsMenuItem<Wry>)
        .collect();
    let model_title = selected_model
        .as_deref()
        .map(|model| format!("默认模型：{model}"))
        .unwrap_or_else(|| "默认模型：未选择".into());
    let models = Submenu::with_items(app, model_title, paired, &model_refs)?;
    let open = MenuItem::with_id(app, "open", "打开主窗口", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "安全退出", true, None::<&str>)?;
    Menu::with_items(
        app,
        &[
            &status,
            &models,
            &PredefinedMenuItem::separator(app)?,
            &open,
            &quit,
        ],
    )
}

async fn refresh_tray_menu(app: &AppHandle) {
    let result = async {
        let menu = build_tray_menu(app).await?;
        let tray = app
            .tray_by_id("main")
            .ok_or_else(|| tauri::Error::AssetNotFound("main tray icon is unavailable".into()))?;
        tray.set_menu(Some(menu))
    }
    .await;
    if let Err(error) = result {
        eprintln!("failed to refresh tray menu: {error}");
    }
}

fn handle_tray_model_selection(app: &AppHandle, event_id: &str) {
    let Some(value) = event_id.strip_prefix("tray_model:") else {
        return;
    };
    let Some((group_id, model)) = value.split_once(':') else {
        return;
    };
    let Ok(group_id) = group_id.parse::<i64>() else {
        return;
    };
    let model = model.to_string();
    let app = app.clone();
    tauri::async_runtime::spawn(async move {
        let runtime = app.state::<AppRuntime>().inner().clone();
        let (_, _, _, selected_group, _, _) = runtime.tray_menu_state().await;
        let first_setup = selected_group.is_none();
        let result = async {
            if first_setup
                && runtime
                    .snapshot()
                    .await
                    .map_err(|error| error.to_string())?
                    .today
                    .balance
                    <= 0.0
            {
                return Err("余额不足，请先在网页充值".into());
            }
            runtime
                .select_route(group_id, model)
                .await
                .map_err(|error| error.to_string())?;
            if first_setup {
                runtime
                    .enable_takeover()
                    .await
                    .map_err(|error| error.to_string())?;
                app.autolaunch()
                    .enable()
                    .map_err(|error| error.to_string())?;
            }
            Ok::<_, String>(())
        }
        .await;
        match result {
            Ok(()) => {
                refresh_tray_menu(&app).await;
                let _ = app.emit("desktop-state-changed", ());
            }
            Err(error) => {
                show_main_window(&app);
                notify_if_enabled(&app, &runtime, "无法切换默认模型", error).await;
            }
        }
    });
}

fn install_tray(app: &AppHandle) -> tauri::Result<()> {
    let menu = initial_tray_menu(app)?;
    let mut builder = TrayIconBuilder::with_id("main")
        .menu(&menu)
        .show_menu_on_left_click(true);
    if let Some(icon) = app.default_window_icon() {
        builder = builder.icon(icon.clone());
    }
    builder
        .on_menu_event(|app, event| match event.id.as_ref() {
            "open" => show_main_window(app),
            "quit" => {
                request_safe_quit(app);
            }
            id => handle_tray_model_selection(app, id),
        })
        .build(app)?;
    Ok(())
}

fn spawn_device_heartbeat(app: &AppHandle) {
    let app = app.clone();
    tauri::async_runtime::spawn(async move {
        let mut interval = tokio::time::interval(std::time::Duration::from_secs(60));
        interval.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);
        loop {
            interval.tick().await;
            let runtime = app.state::<AppRuntime>().inner().clone();
            match runtime.heartbeat_once().await {
                Ok(HeartbeatOutcome::Revoked) => {
                    refresh_tray_menu(&app).await;
                    let _ = app.emit("desktop-state-changed", ());
                    notify_if_enabled(
                        &app,
                        &runtime,
                        "设备授权已撤销",
                        "Codex 配置已恢复，请重新配对后继续使用。",
                    )
                    .await;
                }
                Err(error) if error.is_remote_revocation() => {
                    show_main_window(&app);
                    notify_if_enabled(
                        &app,
                        &runtime,
                        "设备授权已撤销",
                        format!("Codex 配置恢复失败，请在应用内处理：{error}"),
                    )
                    .await;
                }
                Err(error) => eprintln!("desktop heartbeat failed: {error}"),
                Ok(HeartbeatOutcome::Idle | HeartbeatOutcome::Active) => {}
            }
        }
    });
}

pub fn run() {
    let runtime = tauri::async_runtime::block_on(AppRuntime::initialize())
        .expect("failed to initialize LuoxueAPI Desktop runtime");
    let app = tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, _, _| {
            show_main_window(app)
        }))
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(
            tauri_plugin_autostart::Builder::new()
                .macos_launcher(MacosLauncher::LaunchAgent)
                .app_name("落雪API Desktop")
                .build(),
        )
        .menu(build_app_menu)
        .on_menu_event(|app, event| {
            if event.id.as_ref() == "quit_app" {
                request_safe_quit(app);
            }
        })
        .manage(runtime)
        .invoke_handler(tauri::generate_handler![
            get_snapshot,
            list_requests,
            start_pairing,
            open_pairing_page,
            poll_pairing,
            restart_gateway,
            set_takeover,
            select_route,
            update_settings,
            open_codex,
            open_account_page,
            cli_launch_command,
            check_for_updates,
            install_update,
            diagnostic_summary,
            upload_diagnostic,
            prepare_uninstall,
            logout,
        ])
        .setup(|app| {
            install_tray(app.handle())?;
            let app_handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                refresh_tray_menu(&app_handle).await;
            });
            spawn_device_heartbeat(app.handle());
            Ok(())
        })
        .on_window_event(|window, event| {
            if let WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = window.hide();
            }
        })
        .build(tauri::generate_context!())
        .expect("failed to build LuoxueAPI Desktop");

    app.run(|app_handle, event| {
        if let RunEvent::ExitRequested { api, .. } = event {
            let runtime = app_handle.state::<AppRuntime>().inner().clone();
            if runtime.should_restore_on_exit() {
                api.prevent_exit();
                if runtime.begin_shutdown() {
                    let app_handle = app_handle.clone();
                    tauri::async_runtime::spawn(async move {
                        match runtime.finish_explicit_shutdown().await {
                            Ok(()) => app_handle.exit(0),
                            Err(error) => {
                                runtime.cancel_explicit_shutdown();
                                show_main_window(&app_handle);
                                notify_if_enabled(
                                    &app_handle,
                                    &runtime,
                                    "无法安全退出",
                                    format!("Codex 配置尚未恢复：{error}"),
                                )
                                .await;
                            }
                        }
                    });
                }
            }
        }
    });
}

#[cfg(test)]
mod tests {
    #[test]
    fn updater_plugin_config_contains_pubkey_string() {
        let config: serde_json::Value = serde_json::from_str(include_str!("../tauri.conf.json"))
            .expect("tauri.conf.json must contain valid JSON");

        assert!(
            config
                .pointer("/plugins/updater/pubkey")
                .is_some_and(serde_json::Value::is_string),
            "plugins.updater.pubkey must be configured so the updater plugin can initialize"
        );
    }
}
