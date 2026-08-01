use std::{cmp::Ordering, f64::consts::TAU};

use chrono::{DateTime, FixedOffset};
use serde_json::Value;
use tauri::image::Image;

use crate::models::ViewerSnapshot;

const ICON_SIZE: u32 = 36;
const SUPERSAMPLING: u32 = 4;
const RING_RADIUS: f64 = 13.0;
const RING_THICKNESS: f64 = 4.5;
const TRACK_ALPHA: u8 = 70;
const STALE_ARC_ALPHA: u8 = 205;
const ARC_ALPHA: u8 = 255;
const NEUTRAL_ALPHA: u8 = 180;

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum IndicatorState {
    Remaining { percent: u8, stale: bool },
    NoMembership { stale: bool },
    Unknown,
}

/// The native tray representation of the latest sanitized viewer snapshot.
///
/// The exact value lives in the tooltip while the template image encodes it as
/// a ring, so the status remains legible without exposing any account data.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct TrayPresentation {
    state: IndicatorState,
    tooltip: String,
}

impl TrayPresentation {
    /// Generates a 36×36 owned RGBA tray image.
    ///
    /// macOS receives a black template mask and supplies the active menu-bar
    /// foreground. Windows receives an explicit high-contrast color because it
    /// does not implement template-image recoloring.
    #[must_use]
    pub fn image(&self) -> Image<'static> {
        Image::new_owned(render_icon(self.state), ICON_SIZE, ICON_SIZE)
    }

    #[must_use]
    pub fn tooltip(&self) -> &str {
        &self.tooltip
    }

    /// Returns the rounded remaining percentage when a current membership has
    /// a usable weekly-window value.
    #[cfg(test)]
    #[must_use]
    pub fn remaining_percent(&self) -> Option<u8> {
        match self.state {
            IndicatorState::Remaining { percent, .. } => Some(percent),
            IndicatorState::NoMembership { .. } | IndicatorState::Unknown => None,
        }
    }
}

/// Builds the menu-bar presentation from the already-sanitized snapshot that
/// is safe to cross the native/WebView boundary.
#[must_use]
pub fn presentation(snapshot: &ViewerSnapshot) -> TrayPresentation {
    let state = extract_state(snapshot);
    let tooltip = match state {
        IndicatorState::Remaining {
            percent,
            stale: false,
        } => format!("落雪额度 · 会员剩余 {percent}%"),
        IndicatorState::Remaining {
            percent,
            stale: true,
        } => format!("落雪额度 · 上次会员剩余 {percent}%"),
        IndicatorState::NoMembership { stale: false } => "落雪额度 · 暂无会员订阅".into(),
        IndicatorState::NoMembership { stale: true } => "落雪额度 · 上次记录暂无会员订阅".into(),
        IndicatorState::Unknown => "落雪额度 · 会员额度待确认".into(),
    };

    TrayPresentation { state, tooltip }
}

fn extract_state(snapshot: &ViewerSnapshot) -> IndicatorState {
    let Some(overview) = snapshot.overview.as_ref() else {
        return IndicatorState::Unknown;
    };
    let stale = snapshot.status == "stale"
        || overview.get("freshness").and_then(Value::as_str) == Some("stale");
    let Some(subscriptions) = overview.get("subscriptions").and_then(Value::as_array) else {
        return IndicatorState::Unknown;
    };
    if subscriptions.is_empty() {
        return IndicatorState::NoMembership { stale };
    }
    let Some(as_of) = overview
        .get("as_of")
        .and_then(Value::as_str)
        .and_then(parse_timestamp)
    else {
        return IndicatorState::Unknown;
    };

    let mut current: Option<CurrentSubscription<'_>> = None;
    let mut malformed_active_subscription = false;
    for value in subscriptions {
        let Some(object) = value.as_object() else {
            return IndicatorState::Unknown;
        };
        let Some(status) = object.get("status").and_then(Value::as_str) else {
            return IndicatorState::Unknown;
        };
        if status != "active" {
            continue;
        }

        let Some(starts_at) = object
            .get("starts_at")
            .and_then(Value::as_str)
            .and_then(parse_timestamp)
        else {
            malformed_active_subscription = true;
            continue;
        };
        let Some(expires_at) = object
            .get("expires_at")
            .and_then(Value::as_str)
            .and_then(parse_timestamp)
        else {
            malformed_active_subscription = true;
            continue;
        };
        if starts_at > as_of || expires_at <= as_of {
            continue;
        }

        let candidate = CurrentSubscription {
            value,
            starts_at,
            expires_at,
            id: object.get("id").and_then(Value::as_str).unwrap_or_default(),
        };
        if current
            .as_ref()
            .is_none_or(|selected| compare_subscriptions(&candidate, selected).is_gt())
        {
            current = Some(candidate);
        }
    }

    let Some(current) = current else {
        return if malformed_active_subscription {
            IndicatorState::Unknown
        } else {
            IndicatorState::NoMembership { stale }
        };
    };
    let Some(window) = current.value.get("weekly_window") else {
        return IndicatorState::Unknown;
    };
    if window.get("state").and_then(Value::as_str) == Some("unknown") {
        return IndicatorState::Unknown;
    }
    if !matches!(
        window.get("state").and_then(Value::as_str),
        Some("active" | "exhausted")
    ) {
        return IndicatorState::Unknown;
    }
    let Some(used_percent) = window.get("used_percent").and_then(Value::as_f64) else {
        return IndicatorState::Unknown;
    };
    if !used_percent.is_finite() {
        return IndicatorState::Unknown;
    }

    let remaining = (100.0 - used_percent.clamp(0.0, 100.0)).round() as u8;
    IndicatorState::Remaining {
        percent: remaining,
        stale,
    }
}

struct CurrentSubscription<'a> {
    value: &'a Value,
    starts_at: DateTime<FixedOffset>,
    expires_at: DateTime<FixedOffset>,
    id: &'a str,
}

fn parse_timestamp(value: &str) -> Option<DateTime<FixedOffset>> {
    DateTime::parse_from_rfc3339(value).ok()
}

fn compare_subscriptions(
    left: &CurrentSubscription<'_>,
    right: &CurrentSubscription<'_>,
) -> Ordering {
    left.starts_at
        .cmp(&right.starts_at)
        .then_with(|| left.expires_at.cmp(&right.expires_at))
        .then_with(|| compare_subscription_ids(left.id, right.id))
}

fn compare_subscription_ids(left: &str, right: &str) -> Ordering {
    match (normalized_decimal(left), normalized_decimal(right)) {
        (Some(left), Some(right)) => left.len().cmp(&right.len()).then_with(|| left.cmp(right)),
        _ => left.cmp(right),
    }
}

fn normalized_decimal(value: &str) -> Option<&str> {
    if value.is_empty() || !value.bytes().all(|byte| byte.is_ascii_digit()) {
        return None;
    }
    let normalized = value.trim_start_matches('0');
    Some(if normalized.is_empty() {
        "0"
    } else {
        normalized
    })
}

fn render_icon(state: IndicatorState) -> Vec<u8> {
    let mut alpha = vec![0_u8; (ICON_SIZE * ICON_SIZE) as usize];
    match state {
        IndicatorState::Remaining { percent, stale } => {
            draw_ring(&mut alpha, TRACK_ALPHA);
            if percent > 0 {
                draw_arc(
                    &mut alpha,
                    f64::from(percent) / 100.0,
                    if stale { STALE_ARC_ALPHA } else { ARC_ALPHA },
                );
            }
        }
        IndicatorState::NoMembership { .. } => {
            draw_ring(&mut alpha, NEUTRAL_ALPHA);
            draw_line(
                &mut alpha,
                Point { x: 10.5, y: 10.5 },
                Point { x: 25.5, y: 25.5 },
                3.3,
                ARC_ALPHA,
            );
        }
        IndicatorState::Unknown => {
            draw_ring(&mut alpha, NEUTRAL_ALPHA);
            draw_disc(
                &mut alpha,
                Point {
                    x: f64::from(ICON_SIZE) / 2.0,
                    y: f64::from(ICON_SIZE) / 2.0,
                },
                2.3,
                ARC_ALPHA,
            );
        }
    }

    if cfg!(target_os = "macos") {
        return alpha
            .into_iter()
            .flat_map(|alpha| [0, 0, 0, alpha])
            .collect();
    }

    let [red, green, blue] = match state {
        IndicatorState::Remaining { stale: false, .. } => [37, 99, 235],
        IndicatorState::Remaining { stale: true, .. }
        | IndicatorState::NoMembership { stale: true } => [180, 83, 9],
        IndicatorState::NoMembership { stale: false } | IndicatorState::Unknown => [71, 85, 105],
    };
    alpha
        .into_iter()
        .flat_map(|alpha| {
            if alpha == 0 {
                [0, 0, 0, 0]
            } else {
                [red, green, blue, alpha.max(180)]
            }
        })
        .collect()
}

fn draw_ring(alpha: &mut [u8], opacity: u8) {
    draw_shape(alpha, opacity, |point| {
        let distance = point.distance_to(center());
        (distance - RING_RADIUS).abs() <= RING_THICKNESS / 2.0
    });
}

fn draw_arc(alpha: &mut [u8], fraction: f64, opacity: u8) {
    let end_angle = fraction.clamp(0.0, 1.0) * TAU;
    draw_shape(alpha, opacity, |point| {
        let offset_x = point.x - center().x;
        let offset_y = point.y - center().y;
        let distance = offset_x.hypot(offset_y);
        let angle = offset_x.atan2(-offset_y).rem_euclid(TAU);
        (distance - RING_RADIUS).abs() <= RING_THICKNESS / 2.0 && angle <= end_angle
    });

    let cap_radius = RING_THICKNESS / 2.0;
    draw_disc(
        alpha,
        Point {
            x: center().x,
            y: center().y - RING_RADIUS,
        },
        cap_radius,
        opacity,
    );
    if end_angle < TAU {
        draw_disc(
            alpha,
            Point {
                x: center().x + end_angle.sin() * RING_RADIUS,
                y: center().y - end_angle.cos() * RING_RADIUS,
            },
            cap_radius,
            opacity,
        );
    }
}

fn draw_disc(alpha: &mut [u8], disc_center: Point, radius: f64, opacity: u8) {
    draw_shape(alpha, opacity, |point| {
        point.distance_to(disc_center) <= radius
    });
}

fn draw_line(alpha: &mut [u8], start: Point, end: Point, thickness: f64, opacity: u8) {
    draw_shape(alpha, opacity, |point| {
        distance_to_segment(point, start, end) <= thickness / 2.0
    });
}

fn draw_shape<F>(alpha: &mut [u8], opacity: u8, contains: F)
where
    F: Fn(Point) -> bool,
{
    let sample_count = SUPERSAMPLING * SUPERSAMPLING;
    for y in 0..ICON_SIZE {
        for x in 0..ICON_SIZE {
            let mut covered = 0_u32;
            for sample_y in 0..SUPERSAMPLING {
                for sample_x in 0..SUPERSAMPLING {
                    let point = Point {
                        x: f64::from(x) + (f64::from(sample_x) + 0.5) / f64::from(SUPERSAMPLING),
                        y: f64::from(y) + (f64::from(sample_y) + 0.5) / f64::from(SUPERSAMPLING),
                    };
                    covered += u32::from(contains(point));
                }
            }
            if covered == 0 {
                continue;
            }
            let coverage = f64::from(covered) / f64::from(sample_count);
            let sampled_alpha = (f64::from(opacity) * coverage).round() as u8;
            let index = (y * ICON_SIZE + x) as usize;
            alpha[index] = alpha[index].max(sampled_alpha);
        }
    }
}

fn center() -> Point {
    Point {
        x: f64::from(ICON_SIZE) / 2.0,
        y: f64::from(ICON_SIZE) / 2.0,
    }
}

fn distance_to_segment(point: Point, start: Point, end: Point) -> f64 {
    let segment_x = end.x - start.x;
    let segment_y = end.y - start.y;
    let length_squared = segment_x * segment_x + segment_y * segment_y;
    if length_squared == 0.0 {
        return point.distance_to(start);
    }
    let projection =
        ((point.x - start.x) * segment_x + (point.y - start.y) * segment_y) / length_squared;
    let projection = projection.clamp(0.0, 1.0);
    point.distance_to(Point {
        x: start.x + projection * segment_x,
        y: start.y + projection * segment_y,
    })
}

#[derive(Clone, Copy)]
struct Point {
    x: f64,
    y: f64,
}

impl Point {
    fn distance_to(self, other: Self) -> f64 {
        (self.x - other.x).hypot(self.y - other.y)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    fn snapshot(status: &str, overview: Option<Value>) -> ViewerSnapshot {
        ViewerSnapshot {
            status: status.into(),
            overview,
            source: Some("network".into()),
            fetched_at: Some("2026-07-30T12:00:00Z".into()),
            error_code: None,
            message: None,
        }
    }

    fn overview(subscriptions: Value) -> Value {
        json!({
            "as_of": "2026-07-30T12:00:00Z",
            "freshness": "fresh",
            "subscriptions": subscriptions
        })
    }

    fn membership(id: &str, starts_at: &str, expires_at: &str, used_percent: Value) -> Value {
        json!({
            "id": id,
            "status": "active",
            "starts_at": starts_at,
            "expires_at": expires_at,
            "weekly_window": {
                "state": "active",
                "used_percent": used_percent
            }
        })
    }

    #[test]
    fn selects_newest_current_membership() {
        let older = membership(
            "20",
            "2026-07-01T00:00:00Z",
            "2026-08-01T00:00:00Z",
            json!(15.0),
        );
        let newer = membership(
            "21",
            "2026-07-15T00:00:00Z",
            "2026-08-15T00:00:00Z",
            json!(68.0),
        );
        let result = presentation(&snapshot("ready", Some(overview(json!([older, newer])))));

        assert_eq!(result.remaining_percent(), Some(32));
        assert_eq!(result.tooltip(), "落雪额度 · 会员剩余 32%");
    }

    #[test]
    fn newest_current_membership_uses_expiry_then_numeric_id_as_tie_breakers() {
        let shorter = membership(
            "999999999999999999999999999999999999998",
            "2026-07-15T00:00:00Z",
            "2026-08-01T00:00:00Z",
            json!(10.0),
        );
        let lower_id = membership(
            "999999999999999999999999999999999999998",
            "2026-07-15T00:00:00Z",
            "2026-08-15T00:00:00Z",
            json!(20.0),
        );
        let higher_id = membership(
            "999999999999999999999999999999999999999",
            "2026-07-15T00:00:00Z",
            "2026-08-15T00:00:00Z",
            json!(30.0),
        );
        let result = presentation(&snapshot(
            "ready",
            Some(overview(json!([higher_id, shorter, lower_id]))),
        ));

        assert_eq!(result.remaining_percent(), Some(70));
    }

    #[test]
    fn rounds_and_clamps_remaining_percent() {
        for (used, expected) in [
            (json!(-4.0), 100),
            (json!(0.0), 100),
            (json!(67.6), 32),
            (json!(99.5), 1),
            (json!(140.0), 0),
        ] {
            let result = presentation(&snapshot(
                "ready",
                Some(overview(json!([membership(
                    "1",
                    "2026-07-01T00:00:00Z",
                    "2026-08-01T00:00:00Z",
                    used
                )]))),
            ));
            assert_eq!(result.remaining_percent(), Some(expected));
        }
    }

    #[test]
    fn stale_snapshot_is_explicit_in_tooltip() {
        let result = presentation(&snapshot(
            "stale",
            Some(overview(json!([membership(
                "1",
                "2026-07-01T00:00:00Z",
                "2026-08-01T00:00:00Z",
                json!(68.0)
            )]))),
        ));

        assert_eq!(result.remaining_percent(), Some(32));
        assert_eq!(result.tooltip(), "落雪额度 · 上次会员剩余 32%");
    }

    #[test]
    fn empty_or_inactive_memberships_are_not_presented_as_zero_percent() {
        let empty = presentation(&snapshot("ready", Some(overview(json!([])))));
        assert_eq!(empty.remaining_percent(), None);
        assert_eq!(empty.tooltip(), "落雪额度 · 暂无会员订阅");

        let mut expired = membership(
            "1",
            "2026-06-01T00:00:00Z",
            "2026-07-01T00:00:00Z",
            json!(100.0),
        );
        expired["status"] = json!("expired");
        let inactive = presentation(&snapshot("ready", Some(overview(json!([expired])))));
        assert_eq!(inactive.remaining_percent(), None);
        assert_eq!(inactive.tooltip(), "落雪额度 · 暂无会员订阅");
    }

    #[test]
    fn missing_or_unknown_membership_data_stays_unknown() {
        let disconnected = presentation(&ViewerSnapshot::disconnected());
        assert_eq!(disconnected.remaining_percent(), None);
        assert_eq!(disconnected.tooltip(), "落雪额度 · 会员额度待确认");

        let mut unknown = membership(
            "1",
            "2026-07-01T00:00:00Z",
            "2026-08-01T00:00:00Z",
            Value::Null,
        );
        unknown["weekly_window"]["state"] = json!("unknown");
        let unknown = presentation(&snapshot("ready", Some(overview(json!([unknown])))));
        assert_eq!(unknown.remaining_percent(), None);
        assert_eq!(unknown.tooltip(), "落雪额度 · 会员额度待确认");
    }

    #[cfg(target_os = "macos")]
    #[test]
    fn image_is_a_black_36_pixel_template_mask() {
        let result = presentation(&snapshot(
            "ready",
            Some(overview(json!([membership(
                "1",
                "2026-07-01T00:00:00Z",
                "2026-08-01T00:00:00Z",
                json!(68.0)
            )]))),
        ));
        let image = result.image();

        assert_eq!(image.width(), 36);
        assert_eq!(image.height(), 36);
        assert_eq!(image.rgba().len(), 36 * 36 * 4);
        assert!(image
            .rgba()
            .chunks_exact(4)
            .all(|pixel| pixel[..3] == [0, 0, 0]));
    }

    #[cfg(not(target_os = "macos"))]
    #[test]
    fn image_has_a_visible_non_template_palette() {
        let result = presentation(&snapshot(
            "ready",
            Some(overview(json!([membership(
                "1",
                "2026-07-01T00:00:00Z",
                "2026-08-01T00:00:00Z",
                json!(68.0)
            )]))),
        ));
        let image = result.image();

        assert!(image
            .rgba()
            .chunks_exact(4)
            .any(|pixel| pixel[3] > 0 && pixel[..3] != [0, 0, 0]));
    }

    #[test]
    fn arc_length_and_neutral_states_have_distinct_alpha_masks() {
        let full = TrayPresentation {
            state: IndicatorState::Remaining {
                percent: 100,
                stale: false,
            },
            tooltip: String::new(),
        }
        .image();
        let partial = TrayPresentation {
            state: IndicatorState::Remaining {
                percent: 32,
                stale: false,
            },
            tooltip: String::new(),
        }
        .image();
        let no_membership = TrayPresentation {
            state: IndicatorState::NoMembership { stale: false },
            tooltip: String::new(),
        }
        .image();
        let unknown = TrayPresentation {
            state: IndicatorState::Unknown,
            tooltip: String::new(),
        }
        .image();

        assert!(opaque_pixel_count(&full) > opaque_pixel_count(&partial));
        assert_ne!(no_membership.rgba(), unknown.rgba());
        assert_ne!(partial.rgba(), no_membership.rgba());
    }

    fn opaque_pixel_count(image: &Image<'_>) -> usize {
        image
            .rgba()
            .chunks_exact(4)
            .filter(|pixel| pixel[3] >= 200)
            .count()
    }
}
