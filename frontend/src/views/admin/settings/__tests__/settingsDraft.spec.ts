import { describe, expect, it } from "vitest";
import type {
  SystemSettings,
  UpdateSettingsRequest,
} from "@/api/admin/settings";
import {
  createSettingsDraft,
  type SettingsDraft,
} from "../settingsDraft";
import {
  SETTINGS_SECRET_KEYS,
  decodeSettingsDraft,
  encodeSettingsDraft,
} from "../settingsCodec";

function asSystemSettings(
  overrides: Partial<SystemSettings> = {},
): SystemSettings {
  return {
    ...createSettingsDraft(),
    ...overrides,
  } as unknown as SystemSettings;
}

describe("settings draft", () => {
  it("creates independent defaults and accepts localized boundary values", () => {
    const first = createSettingsDraft({
      loginAgreementDocuments: [
        { id: "terms", title: "服务条款", content_md: "正文" },
      ],
      dingtalkCorporateEmailLabel: "钉钉企业邮箱",
      dingtalkNameLabel: "钉钉姓名",
      dingtalkDepartmentLabel: "钉钉部门",
      claudeOAuthSystemPromptBlocks: '[{"type":"text"}]',
    });
    const second = createSettingsDraft();

    expect(first.login_agreement_documents).toEqual([
      { id: "terms", title: "服务条款", content_md: "正文" },
    ]);
    expect(first.dingtalk_connect_sync_corp_email_attr_name).toBe("钉钉企业邮箱");
    expect(first.dingtalk_connect_sync_display_name_attr_name).toBe("钉钉姓名");
    expect(first.dingtalk_connect_sync_dept_attr_name).toBe("钉钉部门");
    expect(first.claude_oauth_system_prompt_blocks).toBe('[{"type":"text"}]');

    first.login_agreement_documents.push({
      id: "privacy",
      title: "Privacy",
      content_md: "",
    });
    first.custom_menu_items.push({
      id: "docs",
      label: "Docs",
      icon_svg: "",
      url: "/docs",
      auth_mode: "none",
      visibility: "user",
      sort_order: 1,
    });
    expect(second.login_agreement_documents).toHaveLength(4);
    expect(second.custom_menu_items).toEqual([]);
    expect(second.table_default_page_size).toBe(20);
  });
});

describe("settings codec", () => {
  it("decodes non-null values, normalizes legacy menu auth, and never hydrates secrets", () => {
    const draft = createSettingsDraft();
    for (const key of SETTINGS_SECRET_KEYS) {
      draft[key] = `stale-${key}`;
    }

    decodeSettingsDraft(
      draft,
      asSystemSettings({
        site_name: "Loaded Site",
        payment_load_balance_strategy: "",
        custom_menu_items: [
          {
            id: "legacy",
            label: "Legacy",
            icon_svg: "",
            url: "/legacy",
            visibility: "user",
            sort_order: 0,
          },
          {
            id: "exchange",
            label: "Exchange",
            icon_svg: "",
            url: "https://example.com",
            auth_mode: "exchange_code",
            visibility: "user",
            sort_order: 1,
          },
        ],
      }),
    );

    expect(draft.site_name).toBe("Loaded Site");
    expect(draft.payment_load_balance_strategy).toBe("round-robin");
    expect(draft.custom_menu_items.map((item) => item.auth_mode)).toEqual([
      "none",
      "exchange_code",
    ]);
    for (const key of SETTINGS_SECRET_KEYS) {
      expect(draft[key]).toBe("");
    }
  });

  it("omits blank secrets and includes only explicitly entered replacements", () => {
    const draft: SettingsDraft = createSettingsDraft();
    draft.smtp_password = "replacement-password";
    draft.google_oauth_client_secret = "replacement-google-secret";

    const base = {
      site_name: "Saved Site",
      smtp_password: undefined,
      turnstile_secret_key: undefined,
    } as UpdateSettingsRequest;
    const encoded = encodeSettingsDraft(base, draft);

    expect(encoded.site_name).toBe("Saved Site");
    expect(encoded.smtp_password).toBe("replacement-password");
    expect(encoded.google_oauth_client_secret).toBe(
      "replacement-google-secret",
    );
    for (const key of SETTINGS_SECRET_KEYS) {
      if (key === "smtp_password" || key === "google_oauth_client_secret") {
        continue;
      }
      expect(Object.prototype.hasOwnProperty.call(encoded, key)).toBe(false);
    }
  });
});
