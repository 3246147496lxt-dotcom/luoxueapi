mod cache;
mod cloud;
mod error;
mod keychain;
mod models;
mod process_lock;
mod state;

use error::{AppError, CommandError, CommandResult};
use models::{PairingState, ViewerSnapshot};
use process_lock::ProcessLock;
use state::AppRuntime;
use tauri::{
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    AppHandle, Emitter, LogicalSize, Manager, State, WindowEvent, Wry,
};

const COMPACT_WIDTH: f64 = 406.0;
const DETAIL_WIDTH: f64 = 790.0;
const PANEL_HEIGHT: f64 = 620.0;

type WindowCommandResult<T> = Result<T, String>;

fn main_window(app: &AppHandle) -> WindowCommandResult<tauri::WebviewWindow> {
    app.get_webview_window("main")
        .ok_or_else(|| "main quota window is unavailable".to_string())
}

#[tauri::command]
fn set_detail_open(app: AppHandle, open: bool) -> WindowCommandResult<()> {
    let window = main_window(&app)?;
    let width = if open { DETAIL_WIDTH } else { COMPACT_WIDTH };
    window
        .set_size(LogicalSize::new(width, PANEL_HEIGHT))
        .map_err(|error| format!("failed to resize quota window: {error}"))
}

#[tauri::command]
fn hide_panel(app: AppHandle) -> WindowCommandResult<()> {
    main_window(&app)?
        .hide()
        .map_err(|error| format!("failed to hide quota window: {error}"))
}

#[tauri::command]
async fn get_viewer_snapshot(
    state: State<'_, AppRuntime>,
    timezone: String,
) -> CommandResult<ViewerSnapshot> {
    let timezone = validate_display_timezone(&timezone)?;
    state.snapshot(timezone).await.map_err(CommandError::from)
}

#[tauri::command]
async fn refresh_quota_overview(
    state: State<'_, AppRuntime>,
    timezone: String,
) -> CommandResult<ViewerSnapshot> {
    let timezone = validate_display_timezone(&timezone)?;
    Ok(state.refresh_overview(timezone).await)
}

fn validate_display_timezone(timezone: &str) -> CommandResult<&str> {
    let timezone = timezone.trim();
    if timezone.is_empty() || timezone.len() > 100 {
        return Err(AppError::Message("display timezone is invalid".into()).into());
    }
    Ok(timezone)
}

#[tauri::command]
async fn start_quota_pairing(
    app: AppHandle,
    state: State<'_, AppRuntime>,
) -> CommandResult<PairingState> {
    state.start_pairing(&app).await.map_err(CommandError::from)
}

#[tauri::command]
async fn open_quota_pairing_page(
    app: AppHandle,
    state: State<'_, AppRuntime>,
) -> CommandResult<()> {
    state
        .open_pairing_page(&app)
        .await
        .map_err(CommandError::from)
}

#[tauri::command]
async fn poll_quota_pairing(state: State<'_, AppRuntime>) -> CommandResult<PairingState> {
    state.poll_pairing().await.map_err(CommandError::from)
}

#[tauri::command]
async fn cancel_quota_pairing(state: State<'_, AppRuntime>) -> CommandResult<()> {
    state.cancel_pairing().await.map_err(CommandError::from)
}

#[tauri::command]
async fn disconnect_quota_account(state: State<'_, AppRuntime>) -> CommandResult<()> {
    state.disconnect().await.map_err(CommandError::from)
}

fn show_panel(app: &AppHandle) {
    let Ok(window) = main_window(app) else {
        return;
    };

    let _ = window.set_size(LogicalSize::new(COMPACT_WIDTH, PANEL_HEIGHT));
    let _ = window.emit("quota-panel-shown", ());
    let _ = window.show();
    let _ = window.unminimize();
    let _ = window.set_focus();
}

fn toggle_panel(app: &AppHandle) {
    let Ok(window) = main_window(app) else {
        return;
    };

    if window.is_visible().unwrap_or(false) {
        let _ = window.hide();
    } else {
        show_panel(app);
    }
}

fn tray_menu(app: &AppHandle) -> tauri::Result<Menu<Wry>> {
    let open = MenuItem::with_id(app, "open", "打开额度面板", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "退出落雪额度", true, None::<&str>)?;
    Menu::with_items(app, &[&open, &quit])
}

fn install_tray(app: &AppHandle) -> tauri::Result<()> {
    let menu = tray_menu(app)?;
    let mut builder = TrayIconBuilder::with_id("quota")
        .tooltip("落雪额度")
        .menu(&menu)
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| match event.id.as_ref() {
            "open" => show_panel(app),
            "quit" => app.exit(0),
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if matches!(
                event,
                TrayIconEvent::Click {
                    button: MouseButton::Left,
                    button_state: MouseButtonState::Up,
                    ..
                }
            ) {
                toggle_panel(tray.app_handle());
            }
        });

    if let Some(icon) = app.default_window_icon() {
        builder = builder.icon(icon.clone());
    }

    builder.build(app)?;
    Ok(())
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, _, _| {
            show_panel(app);
        }))
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![
            set_detail_open,
            hide_panel,
            get_viewer_snapshot,
            refresh_quota_overview,
            start_quota_pairing,
            open_quota_pairing_page,
            poll_quota_pairing,
            cancel_quota_pairing,
            disconnect_quota_account
        ])
        .setup(|app| {
            let cache_directory = app.path().app_data_dir()?;
            let process_lock =
                ProcessLock::acquire(&cache_directory.join("quota-viewer-process.lock"))?;
            app.manage(process_lock);

            #[cfg(target_os = "macos")]
            app.handle()
                .set_activation_policy(tauri::ActivationPolicy::Accessory)?;

            let runtime = AppRuntime::new(cache_directory.join("quota-overview.json"))
                .map_err(|error| std::io::Error::other(error.to_string()))?;
            app.manage(runtime);
            install_tray(app.handle())?;
            Ok(())
        })
        .on_window_event(|window, event| {
            if let WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = window.hide();
            }
        })
        .run(tauri::generate_context!())
        .expect("failed to run Luoxue quota viewer");
}

#[cfg(test)]
mod tests {
    use super::{COMPACT_WIDTH, DETAIL_WIDTH, PANEL_HEIGHT};

    #[test]
    fn detail_layout_matches_two_reference_panels() {
        assert_eq!(COMPACT_WIDTH, 374.0 + 32.0);
        assert_eq!(DETAIL_WIDTH, 374.0 * 2.0 + 10.0 + 32.0);
        assert_eq!(PANEL_HEIGHT, 588.0 + 32.0);
    }
}
