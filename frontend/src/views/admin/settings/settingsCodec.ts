import type {
  SystemSettings,
  UpdateSettingsRequest,
} from "@/api/admin/settings";
import type { SettingsDraft } from "./settingsDraft";

export const SETTINGS_SECRET_KEYS = [
  "smtp_password",
  "turnstile_secret_key",
  "linuxdo_connect_client_secret",
  "dingtalk_connect_client_secret",
  "wechat_connect_app_secret",
  "wechat_connect_open_app_secret",
  "wechat_connect_mp_app_secret",
  "wechat_connect_mobile_app_secret",
  "oidc_connect_client_secret",
  "github_oauth_client_secret",
  "google_oauth_client_secret",
] as const;

export type SettingsSecretKey = (typeof SETTINGS_SECRET_KEYS)[number];

/**
 * Secrets returned by the settings API are represented by `*_configured` flags.
 * The editable draft must never hydrate a secret value from a response.
 */
export function clearSettingsDraftSecrets(
  draft: Pick<SettingsDraft, SettingsSecretKey>,
): void {
  for (const key of SETTINGS_SECRET_KEYS) {
    draft[key] = "";
  }
}

/**
 * Applies a settings response without allowing null values to erase local defaults.
 * Secret inputs are always reset, preserving the "blank means keep existing" contract.
 */
export function decodeSettingsDraft(
  draft: SettingsDraft,
  settings: SystemSettings,
  options: { skipKeys?: readonly string[] } = {},
): SettingsDraft {
  const values: Record<string, unknown> = {
    ...settings,
    payment_load_balance_strategy:
      settings.payment_load_balance_strategy || "round-robin",
  };

  for (const [key, value] of Object.entries(values)) {
    if (options.skipKeys?.includes(key)) {
      continue;
    }
    if (value !== null && value !== undefined) {
      (draft as Record<string, unknown>)[key] = value;
    }
  }

  draft.custom_menu_items = (settings.custom_menu_items || []).map((item) => ({
    ...item,
    auth_mode: item.auth_mode === "exchange_code" ? "exchange_code" : "none",
  }));
  clearSettingsDraftSecrets(draft);
  return draft;
}

/**
 * Finalizes an already validated settings payload. Non-empty secret inputs replace
 * the stored secret; blank inputs are omitted so the backend retains its value.
 */
export function encodeSettingsDraft(
  payload: UpdateSettingsRequest,
  draft: Pick<SettingsDraft, SettingsSecretKey>,
): UpdateSettingsRequest {
  const encoded: UpdateSettingsRequest = { ...payload };
  const target = encoded as Record<string, unknown>;

  for (const key of SETTINGS_SECRET_KEYS) {
    const value = draft[key];
    if (value) {
      target[key] = value;
    } else {
      delete target[key];
    }
  }

  return encoded;
}
