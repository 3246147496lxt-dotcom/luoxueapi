mod cache;
mod cloud;
mod credentials;
mod error;
mod file_replace;
mod models;
mod process_lock;
mod state;
mod tray_indicator;

use std::sync::{
    atomic::{AtomicBool, Ordering},
    Mutex,
};

use chrono::{DateTime, Utc};
use error::{AppError, CommandError, CommandResult};
use models::{PairingState, ViewerSnapshot};
use process_lock::ProcessLock;
use state::AppRuntime;
use tauri::{
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    AppHandle, Emitter, LogicalPosition, LogicalSize, Manager, Monitor, PhysicalPosition, Rect,
    State, WebviewWindow, WindowEvent, Wry,
};

const MAIN_PANEL_COMPACT_WIDTH: f64 = 80.0;
const MAIN_PANEL_COMPACT_HEIGHT: f64 = 80.0;
const MAIN_PANEL_EXPANDED_WIDTH: f64 = 314.0;
const MAIN_PANEL_EXPANDED_HEIGHT: f64 = 314.0;
const TRAY_PANEL_WIDTH: f64 = 406.0;
const TRAY_PANEL_HEIGHT: f64 = 500.0;
const TRAY_DETAIL_PANEL_WIDTH: f64 = 822.0;
const TRAY_PANEL_GAP: f64 = 6.0;
const SCREEN_MARGIN: f64 = 8.0;

type WindowCommandResult<T> = Result<T, String>;

#[derive(Clone, Copy, Debug, PartialEq)]
struct LogicalBounds {
    position: LogicalPosition<f64>,
    size: LogicalSize<f64>,
}

#[derive(Default)]
struct TrayClickState {
    panel_was_visible_on_mouse_down: AtomicBool,
}

#[derive(Default)]
struct TrayIndicatorUpdateState {
    last_revision: Mutex<Option<DateTime<Utc>>>,
}

fn main_window(app: &AppHandle) -> WindowCommandResult<tauri::WebviewWindow> {
    app.get_webview_window("main")
        .ok_or_else(|| "main quota window is unavailable".to_string())
}

#[tauri::command]
fn hide_panel(app: AppHandle) -> WindowCommandResult<()> {
    main_window(&app)?
        .hide()
        .map_err(|error| format!("failed to hide quota window: {error}"))
}

#[tauri::command]
fn hide_tray_panel(app: AppHandle) -> WindowCommandResult<()> {
    if let Some(window) = app.get_webview_window("tray") {
        let resize_result = set_tray_window_size(&window, tray_panel_size());
        let hide_result = window
            .hide()
            .map_err(|error| format!("failed to hide tray quota panel: {error}"));
        resize_result?;
        hide_result?;
    }
    Ok(())
}

#[tauri::command]
fn set_tray_detail_open(app: AppHandle, open: bool) -> WindowCommandResult<bool> {
    resize_tray_panel(&app, open)?;
    Ok(open)
}

#[tauri::command]
fn open_main_panel(app: AppHandle) {
    show_panel(&app);
}

#[tauri::command]
fn set_main_panel_expanded(app: AppHandle, expanded: bool) -> WindowCommandResult<bool> {
    resize_main_panel(&app, expanded)?;
    Ok(expanded)
}

#[tauri::command]
async fn get_viewer_snapshot(
    app: AppHandle,
    state: State<'_, AppRuntime>,
    timezone: String,
) -> CommandResult<ViewerSnapshot> {
    let timezone = validate_display_timezone(&timezone)?;
    let snapshot = state.snapshot(timezone).await.map_err(CommandError::from)?;
    update_tray_indicator(&app, &snapshot);
    Ok(snapshot)
}

#[tauri::command]
async fn refresh_quota_overview(
    app: AppHandle,
    state: State<'_, AppRuntime>,
    timezone: String,
) -> CommandResult<ViewerSnapshot> {
    let timezone = validate_display_timezone(&timezone)?;
    let snapshot = state.refresh_overview(timezone).await;
    update_tray_indicator(&app, &snapshot);
    Ok(snapshot)
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
async fn disconnect_quota_account(
    app: AppHandle,
    state: State<'_, AppRuntime>,
) -> CommandResult<()> {
    state.disconnect().await.map_err(CommandError::from)?;
    update_tray_indicator(&app, &ViewerSnapshot::disconnected());
    Ok(())
}

fn show_panel(app: &AppHandle) {
    let Ok(window) = main_window(app) else {
        return;
    };

    if let Some(tray_window) = app.get_webview_window("tray") {
        hide_tray_window(&tray_window);
    }
    if resize_main_panel(app, true).is_err() {
        return;
    }
    let _ = window.emit("quota-panel-shown", ());
    let _ = window.show();
    let _ = window.unminimize();
    let _ = window.set_focus();
}

fn main_panel_size(expanded: bool) -> LogicalSize<f64> {
    if expanded {
        LogicalSize::new(MAIN_PANEL_EXPANDED_WIDTH, MAIN_PANEL_EXPANDED_HEIGHT)
    } else {
        LogicalSize::new(MAIN_PANEL_COMPACT_WIDTH, MAIN_PANEL_COMPACT_HEIGHT)
    }
}

fn centered_panel_position(
    current_bounds: LogicalBounds,
    work_area: LogicalBounds,
    target_size: LogicalSize<f64>,
) -> LogicalPosition<f64> {
    let minimum_x = work_area.position.x + SCREEN_MARGIN;
    let maximum_x =
        (work_area.position.x + work_area.size.width - target_size.width - SCREEN_MARGIN)
            .max(minimum_x);
    let minimum_y = work_area.position.y + SCREEN_MARGIN;
    let maximum_y =
        (work_area.position.y + work_area.size.height - target_size.height - SCREEN_MARGIN)
            .max(minimum_y);
    let centered_x =
        current_bounds.position.x + (current_bounds.size.width - target_size.width) / 2.0;
    let centered_y =
        current_bounds.position.y + (current_bounds.size.height - target_size.height) / 2.0;

    LogicalPosition::new(
        centered_x.clamp(minimum_x, maximum_x),
        centered_y.clamp(minimum_y, maximum_y),
    )
}

fn resize_main_panel(app: &AppHandle, expanded: bool) -> WindowCommandResult<()> {
    let window = main_window(app)?;
    let monitor = window
        .current_monitor()
        .map_err(|error| format!("failed to resolve current quota monitor: {error}"))?
        .or_else(|| app.primary_monitor().ok().flatten())
        .ok_or_else(|| "no monitor is available for the quota window".to_string())?;
    let scale_factor = monitor.scale_factor();
    let current_bounds = LogicalBounds {
        position: window
            .outer_position()
            .map_err(|error| format!("failed to read quota window position: {error}"))?
            .to_logical::<f64>(scale_factor),
        size: window
            .outer_size()
            .map_err(|error| format!("failed to read quota window size: {error}"))?
            .to_logical::<f64>(scale_factor),
    };
    let work_area = monitor.work_area();
    let work_area = LogicalBounds {
        position: work_area.position.to_logical::<f64>(scale_factor),
        size: work_area.size.to_logical::<f64>(scale_factor),
    };
    let target_size = main_panel_size(expanded);
    let target_position = centered_panel_position(current_bounds, work_area, target_size);

    window
        .set_size(target_size)
        .map_err(|error| format!("failed to resize quota window: {error}"))?;
    if let Err(error) = window.set_position(target_position) {
        let _ = window.set_size(current_bounds.size);
        let _ = window.set_position(current_bounds.position);
        return Err(format!("failed to reposition quota window: {error}"));
    }
    let _ = window.emit("quota-main-layout-changed", expanded);
    Ok(())
}

fn tray_window(app: &AppHandle) -> Option<WebviewWindow> {
    app.get_webview_window("tray")
}

fn tray_panel_size() -> LogicalSize<f64> {
    LogicalSize::new(TRAY_PANEL_WIDTH, TRAY_PANEL_HEIGHT)
}

fn tray_detail_panel_size() -> LogicalSize<f64> {
    LogicalSize::new(TRAY_DETAIL_PANEL_WIDTH, TRAY_PANEL_HEIGHT)
}

fn clamp_panel_position(
    position: LogicalPosition<f64>,
    work_area: LogicalBounds,
    panel_size: LogicalSize<f64>,
    minimum_y_gap: f64,
) -> LogicalPosition<f64> {
    let minimum_x = work_area.position.x + SCREEN_MARGIN;
    let maximum_x =
        (work_area.position.x + work_area.size.width - panel_size.width - SCREEN_MARGIN)
            .max(minimum_x);
    let minimum_y = work_area.position.y + minimum_y_gap;
    let maximum_y =
        (work_area.position.y + work_area.size.height - panel_size.height - SCREEN_MARGIN)
            .max(minimum_y);

    LogicalPosition::new(
        position.x.clamp(minimum_x, maximum_x),
        position.y.clamp(minimum_y, maximum_y),
    )
}

fn resize_tray_panel(app: &AppHandle, detail_open: bool) -> WindowCommandResult<()> {
    let window = tray_window(app).ok_or_else(|| "tray quota window is unavailable".to_string())?;
    let monitor = window
        .current_monitor()
        .map_err(|error| format!("failed to resolve current tray monitor: {error}"))?
        .or_else(|| app.primary_monitor().ok().flatten())
        .ok_or_else(|| "no monitor is available for the tray quota window".to_string())?;
    let scale_factor = monitor.scale_factor();
    let current_position = window
        .outer_position()
        .map_err(|error| format!("failed to read tray quota window position: {error}"))?
        .to_logical::<f64>(scale_factor);
    let current_size = window
        .outer_size()
        .map_err(|error| format!("failed to read tray quota window size: {error}"))?
        .to_logical::<f64>(scale_factor);
    let work_area = monitor.work_area();
    let work_area = LogicalBounds {
        position: work_area.position.to_logical::<f64>(scale_factor),
        size: work_area.size.to_logical::<f64>(scale_factor),
    };
    let target_size = if detail_open {
        tray_detail_panel_size()
    } else {
        tray_panel_size()
    };
    let target_position =
        clamp_panel_position(current_position, work_area, target_size, TRAY_PANEL_GAP);

    window
        .set_size(target_size)
        .map_err(|error| format!("failed to resize tray quota panel: {error}"))?;
    if let Err(error) = window.set_position(target_position) {
        let _ = window.set_size(current_size);
        let _ = window.set_position(current_position);
        return Err(format!("failed to reposition tray quota panel: {error}"));
    }
    let _ = window.emit("quota-tray-layout-changed", detail_open);
    Ok(())
}

fn set_tray_window_size(
    window: &WebviewWindow,
    panel_size: LogicalSize<f64>,
) -> WindowCommandResult<()> {
    window
        .set_size(panel_size)
        .map_err(|error| format!("failed to resize tray quota panel: {error}"))
}

fn clamp_tray_panel_position(
    tray_rect: LogicalBounds,
    work_area: LogicalBounds,
    panel_size: LogicalSize<f64>,
) -> LogicalPosition<f64> {
    let minimum_x = work_area.position.x + SCREEN_MARGIN;
    let maximum_x =
        (work_area.position.x + work_area.size.width - panel_size.width - SCREEN_MARGIN)
            .max(minimum_x);
    let minimum_y = work_area.position.y + TRAY_PANEL_GAP;
    let maximum_y =
        (work_area.position.y + work_area.size.height - panel_size.height - SCREEN_MARGIN)
            .max(minimum_y);

    LogicalPosition::new(
        (tray_rect.position.x + tray_rect.size.width / 2.0 - panel_size.width / 2.0)
            .clamp(minimum_x, maximum_x),
        (tray_rect.position.y + tray_rect.size.height + TRAY_PANEL_GAP).clamp(minimum_y, maximum_y),
    )
}

fn tray_panel_position(
    app: &AppHandle,
    click_position: PhysicalPosition<f64>,
    tray_rect: Rect,
    panel_size: LogicalSize<f64>,
) -> Option<LogicalPosition<f64>> {
    let monitor = app
        .monitor_from_point(click_position.x, click_position.y)
        .ok()
        .flatten()
        .or_else(|| app.primary_monitor().ok().flatten())?;
    Some(tray_panel_position_for_monitor(
        tray_rect, &monitor, panel_size,
    ))
}

fn tray_panel_position_for_monitor(
    tray_rect: Rect,
    monitor: &Monitor,
    panel_size: LogicalSize<f64>,
) -> LogicalPosition<f64> {
    let scale_factor = monitor.scale_factor();
    let tray_rect = LogicalBounds {
        position: tray_rect.position.to_logical::<f64>(scale_factor),
        size: tray_rect.size.to_logical::<f64>(scale_factor),
    };
    let work_area = monitor.work_area();
    let work_area = LogicalBounds {
        position: work_area.position.to_logical::<f64>(scale_factor),
        size: work_area.size.to_logical::<f64>(scale_factor),
    };
    clamp_tray_panel_position(tray_rect, work_area, panel_size)
}

fn hide_tray_window(window: &WebviewWindow) {
    let _ = set_tray_window_size(window, tray_panel_size());
    let _ = window.hide();
}

fn toggle_tray_panel(app: &AppHandle, click_position: PhysicalPosition<f64>, tray_rect: Rect) {
    let Some(window) = tray_window(app) else {
        return;
    };

    if window.is_visible().unwrap_or(false) {
        hide_tray_window(&window);
        return;
    }

    let panel_size = tray_panel_size();
    let _ = set_tray_window_size(&window, panel_size);
    if let Some(position) = tray_panel_position(app, click_position, tray_rect, panel_size) {
        let _ = window.set_position(position);
    }
    let _ = window.emit("quota-tray-shown", ());
    let _ = window.show();
    let _ = window.unminimize();
    let _ = window.set_focus();
}

fn remember_tray_panel_mouse_down(app: &AppHandle) {
    let visible = tray_window(app)
        .and_then(|window| window.is_visible().ok())
        .unwrap_or(false);
    app.state::<TrayClickState>()
        .panel_was_visible_on_mouse_down
        .store(visible, Ordering::Release);
}

fn handle_tray_panel_mouse_up(
    app: &AppHandle,
    click_position: PhysicalPosition<f64>,
    tray_rect: Rect,
) {
    let was_visible = app
        .state::<TrayClickState>()
        .panel_was_visible_on_mouse_down
        .swap(false, Ordering::AcqRel);
    if was_visible {
        if let Some(window) = tray_window(app) {
            hide_tray_window(&window);
        }
        return;
    }
    toggle_tray_panel(app, click_position, tray_rect);
}

fn tray_menu(app: &AppHandle) -> tauri::Result<Menu<Wry>> {
    let open = MenuItem::with_id(app, "open", "打开额度面板", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "退出落雪额度", true, None::<&str>)?;
    Menu::with_items(app, &[&open, &quit])
}

fn update_tray_indicator(app: &AppHandle, snapshot: &ViewerSnapshot) {
    let update_state = app.state::<TrayIndicatorUpdateState>();
    let mut last_revision = update_state
        .last_revision
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    if !accept_tray_snapshot(&mut last_revision, snapshot) {
        return;
    }

    let presentation = tray_indicator::presentation(snapshot);
    if let Some(tray) = app.tray_by_id("quota") {
        let _ =
            tray.set_icon_with_as_template(Some(presentation.image()), cfg!(target_os = "macos"));
        let _ = tray.set_tooltip(Some(presentation.tooltip()));
    }
}

fn accept_tray_snapshot(
    last_revision: &mut Option<DateTime<Utc>>,
    snapshot: &ViewerSnapshot,
) -> bool {
    if snapshot.overview.is_some() {
        let candidate = snapshot_revision(snapshot);
        match (last_revision.as_ref(), candidate) {
            (Some(last), Some(candidate)) if candidate < *last => return false,
            (_, Some(candidate)) => *last_revision = Some(candidate),
            (Some(_), None) => return false,
            (None, None) => {}
        }
        return true;
    }

    if matches!(snapshot.status.as_str(), "disconnected" | "auth-invalid") {
        *last_revision = None;
        return true;
    }

    last_revision.is_none()
}

fn snapshot_revision(snapshot: &ViewerSnapshot) -> Option<DateTime<Utc>> {
    snapshot
        .overview
        .as_ref()
        .and_then(|overview| overview.get("generated_at"))
        .and_then(serde_json::Value::as_str)
        .and_then(|value| DateTime::parse_from_rfc3339(value).ok())
        .map(|value| value.with_timezone(&Utc))
        .or_else(|| {
            snapshot
                .fetched_at
                .as_deref()
                .and_then(|value| DateTime::parse_from_rfc3339(value).ok())
                .map(|value| value.with_timezone(&Utc))
        })
}

fn install_tray(app: &AppHandle) -> tauri::Result<()> {
    let menu = tray_menu(app)?;
    let initial_presentation = tray_indicator::presentation(&ViewerSnapshot::disconnected());
    TrayIconBuilder::with_id("quota")
        .icon(initial_presentation.image())
        .icon_as_template(cfg!(target_os = "macos"))
        .tooltip(initial_presentation.tooltip())
        .menu(&menu)
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| match event.id.as_ref() {
            "open" => show_panel(app),
            "quit" => app.exit(0),
            _ => {}
        })
        .on_tray_icon_event(|tray, event| match event {
            TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Down,
                ..
            } => remember_tray_panel_mouse_down(tray.app_handle()),
            TrayIconEvent::Click {
                position,
                rect,
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } => handle_tray_panel_mouse_up(tray.app_handle(), position, rect),
            _ => {}
        })
        .build(app)?;
    Ok(())
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, _, _| {
            show_panel(app);
        }))
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![
            hide_panel,
            hide_tray_panel,
            open_main_panel,
            set_main_panel_expanded,
            set_tray_detail_open,
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
            app.manage(TrayClickState::default());
            app.manage(TrayIndicatorUpdateState::default());

            #[cfg(target_os = "macos")]
            app.handle()
                .set_activation_policy(tauri::ActivationPolicy::Accessory)?;

            let runtime = AppRuntime::new(
                cache_directory.join("quota-overview.json"),
                cache_directory.join("credentials-v1.json"),
            )
            .map_err(|error| std::io::Error::other(error.to_string()))?;
            app.manage(runtime);
            install_tray(app.handle())?;
            Ok(())
        })
        .on_window_event(|window, event| {
            if window.label() == "tray" && matches!(event, WindowEvent::Focused(false)) {
                let _ = window.set_size(tray_panel_size());
                let _ = window.hide();
                return;
            }
            if let WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                if window.label() == "tray" {
                    let _ = window.set_size(tray_panel_size());
                    let _ = window.hide();
                } else {
                    let _ = window.hide();
                }
            }
        })
        .run(tauri::generate_context!())
        .expect("failed to run Luoxue quota viewer");
}

#[cfg(test)]
mod tests {
    use super::{
        accept_tray_snapshot, centered_panel_position, clamp_panel_position,
        clamp_tray_panel_position, main_panel_size, tray_detail_panel_size, tray_panel_size,
        LogicalBounds, MAIN_PANEL_COMPACT_HEIGHT, MAIN_PANEL_COMPACT_WIDTH,
        MAIN_PANEL_EXPANDED_HEIGHT, MAIN_PANEL_EXPANDED_WIDTH, SCREEN_MARGIN,
        TRAY_DETAIL_PANEL_WIDTH, TRAY_PANEL_GAP, TRAY_PANEL_HEIGHT, TRAY_PANEL_WIDTH,
    };
    use crate::models::ViewerSnapshot;
    use chrono::{TimeZone, Utc};
    use serde_json::json;
    use tauri::{LogicalPosition, LogicalSize};

    #[test]
    fn floating_panel_and_tray_use_independent_dimensions() {
        assert_eq!(
            main_panel_size(false),
            LogicalSize::new(MAIN_PANEL_COMPACT_WIDTH, MAIN_PANEL_COMPACT_HEIGHT)
        );
        assert_eq!(
            main_panel_size(true),
            LogicalSize::new(MAIN_PANEL_EXPANDED_WIDTH, MAIN_PANEL_EXPANDED_HEIGHT)
        );
        assert_eq!(
            tray_panel_size(),
            LogicalSize::new(TRAY_PANEL_WIDTH, TRAY_PANEL_HEIGHT)
        );
        assert_eq!(
            tray_detail_panel_size(),
            LogicalSize::new(TRAY_DETAIL_PANEL_WIDTH, TRAY_PANEL_HEIGHT)
        );
    }

    #[test]
    fn floating_panel_resize_preserves_its_center() {
        let work_area = LogicalBounds {
            position: LogicalPosition::new(0.0, 24.0),
            size: LogicalSize::new(1440.0, 876.0),
        };
        let compact_bounds = LogicalBounds {
            position: LogicalPosition::new(680.0, 422.0),
            size: main_panel_size(false),
        };
        let expanded_position =
            centered_panel_position(compact_bounds, work_area, main_panel_size(true));

        assert_eq!(expanded_position, LogicalPosition::new(563.0, 305.0));

        let expanded_bounds = LogicalBounds {
            position: expanded_position,
            size: main_panel_size(true),
        };
        assert_eq!(
            centered_panel_position(expanded_bounds, work_area, main_panel_size(false)),
            compact_bounds.position
        );
    }

    #[test]
    fn floating_panel_resize_stays_inside_the_active_work_area() {
        let work_area = LogicalBounds {
            position: LogicalPosition::new(-1440.0, 24.0),
            size: LogicalSize::new(1440.0, 876.0),
        };
        let target_size = main_panel_size(true);
        let near_left = centered_panel_position(
            LogicalBounds {
                position: LogicalPosition::new(-1440.0, 24.0),
                size: main_panel_size(false),
            },
            work_area,
            target_size,
        );
        let near_right = centered_panel_position(
            LogicalBounds {
                position: LogicalPosition::new(-80.0, 820.0),
                size: main_panel_size(false),
            },
            work_area,
            target_size,
        );

        assert_eq!(
            near_left,
            LogicalPosition::new(
                work_area.position.x + SCREEN_MARGIN,
                work_area.position.y + SCREEN_MARGIN
            )
        );
        assert_eq!(
            near_right,
            LogicalPosition::new(
                work_area.position.x + work_area.size.width - target_size.width - SCREEN_MARGIN,
                work_area.position.y + work_area.size.height - target_size.height - SCREEN_MARGIN
            )
        );
    }

    #[test]
    fn tray_panel_centers_below_the_menu_bar_icon() {
        let compact_size = LogicalSize::new(TRAY_PANEL_WIDTH, TRAY_PANEL_HEIGHT);
        let position = clamp_tray_panel_position(
            LogicalBounds {
                position: LogicalPosition::new(800.0, 0.0),
                size: LogicalSize::new(24.0, 24.0),
            },
            LogicalBounds {
                position: LogicalPosition::new(0.0, 24.0),
                size: LogicalSize::new(1440.0, 876.0),
            },
            compact_size,
        );

        assert_eq!(
            position,
            LogicalPosition::new(800.0 + 12.0 - TRAY_PANEL_WIDTH / 2.0, 24.0 + TRAY_PANEL_GAP)
        );
    }

    #[test]
    fn tray_panel_stays_inside_the_active_screen_work_area() {
        let work_area = LogicalBounds {
            position: LogicalPosition::new(-1440.0, 24.0),
            size: LogicalSize::new(1440.0, 876.0),
        };
        let panel_size = tray_panel_size();
        let left_position = clamp_tray_panel_position(
            LogicalBounds {
                position: LogicalPosition::new(-1440.0, 0.0),
                size: LogicalSize::new(24.0, 24.0),
            },
            work_area,
            panel_size,
        );
        let right_position = clamp_tray_panel_position(
            LogicalBounds {
                position: LogicalPosition::new(-20.0, 0.0),
                size: LogicalSize::new(20.0, 24.0),
            },
            work_area,
            panel_size,
        );

        assert_eq!(left_position.x, work_area.position.x + SCREEN_MARGIN);
        assert_eq!(
            right_position.x,
            work_area.position.x + work_area.size.width - panel_size.width - SCREEN_MARGIN
        );
        for position in [left_position, right_position] {
            assert!(position.x >= work_area.position.x);
            assert!(position.x + panel_size.width <= work_area.position.x + work_area.size.width);
            assert!(position.y >= work_area.position.y);
            assert!(position.y + panel_size.height <= work_area.position.y + work_area.size.height);
        }
    }

    #[test]
    fn tray_detail_panel_clamps_inside_the_active_work_area() {
        let work_area = LogicalBounds {
            position: LogicalPosition::new(0.0, 24.0),
            size: LogicalSize::new(1440.0, 876.0),
        };
        let detail_size = tray_detail_panel_size();
        let unchanged = clamp_panel_position(
            LogicalPosition::new(100.0, 30.0),
            work_area,
            detail_size,
            TRAY_PANEL_GAP,
        );
        let clamped = clamp_panel_position(
            LogicalPosition::new(1200.0, 800.0),
            work_area,
            detail_size,
            TRAY_PANEL_GAP,
        );

        assert_eq!(unchanged, LogicalPosition::new(100.0, 30.0));
        assert_eq!(
            clamped,
            LogicalPosition::new(
                work_area.position.x + work_area.size.width - detail_size.width - SCREEN_MARGIN,
                work_area.position.y + work_area.size.height - detail_size.height - SCREEN_MARGIN
            )
        );
    }

    #[test]
    fn older_server_snapshot_cannot_replace_a_newer_tray_indicator() {
        let mut last_fetched_at = Some(
            Utc.with_ymd_and_hms(2026, 7, 31, 12, 0, 0)
                .single()
                .unwrap(),
        );
        let older = ViewerSnapshot {
            status: "stale".into(),
            overview: Some(json!({
                "generated_at": "2026-07-31T11:59:59Z"
            })),
            source: Some("cache".into()),
            // This older request completed later. Its local fetch timestamp
            // must not let it overwrite newer server data from another window.
            fetched_at: Some("2026-07-31T12:01:00Z".into()),
            error_code: None,
            message: None,
        };

        assert!(!accept_tray_snapshot(&mut last_fetched_at, &older));
        assert_eq!(
            last_fetched_at,
            Some(
                Utc.with_ymd_and_hms(2026, 7, 31, 12, 0, 0)
                    .single()
                    .unwrap()
            )
        );
    }

    #[test]
    fn transient_missing_data_preserves_the_last_known_indicator() {
        let mut last_fetched_at = Some(
            Utc.with_ymd_and_hms(2026, 7, 31, 12, 0, 0)
                .single()
                .unwrap(),
        );
        let unavailable =
            ViewerSnapshot::without_data("unavailable", Some("HTTP_503".into()), None);

        assert!(!accept_tray_snapshot(&mut last_fetched_at, &unavailable));
        assert!(last_fetched_at.is_some());
    }

    #[test]
    fn disconnect_clears_the_tray_indicator_revision() {
        let mut last_fetched_at = Some(
            Utc.with_ymd_and_hms(2026, 7, 31, 12, 0, 0)
                .single()
                .unwrap(),
        );

        assert!(accept_tray_snapshot(
            &mut last_fetched_at,
            &ViewerSnapshot::disconnected()
        ));
        assert_eq!(last_fetched_at, None);
    }
}
