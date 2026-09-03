import { beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent, h } from "vue";
import { flushPromises, mount } from "@vue/test-utils";

import TranscriptionSettingsPanel from "../TranscriptionSettingsPanel.vue";

const {
  getWebChatTranscriptionSettings,
  updateWebChatTranscriptionSettings,
  getGroups,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  getWebChatTranscriptionSettings: vi.fn(),
  updateWebChatTranscriptionSettings: vi.fn(),
  getGroups: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}));

vi.mock("@/api/admin", () => ({
  adminAPI: {
    settings: {
      getWebChatTranscriptionSettings,
      updateWebChatTranscriptionSettings,
    },
    groups: {
      getAllIncludingInactive: getGroups,
    },
  },
}));

vi.mock("@/stores", () => ({
  useAppStore: () => ({ showError, showSuccess }),
}));

vi.mock("@/utils/apiError", () => ({
  extractApiErrorMessage: (_error: unknown, fallback: string) => fallback,
}));

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string | number>) =>
      `${key}${params ? `:${JSON.stringify(params)}` : ""}`,
    locale: { value: "zh-CN" },
  }),
}));

const ToggleStub = defineComponent({
  inheritAttrs: false,
  props: {
    modelValue: { type: Boolean, default: false },
  },
  emits: ["update:modelValue"],
  setup(props, { attrs, emit }) {
    return () =>
      h("input", {
        ...attrs,
        type: "checkbox",
        checked: props.modelValue,
        onChange: (event: Event) =>
          emit(
            "update:modelValue",
            (event.target as HTMLInputElement).checked,
          ),
      });
  },
});

const baseSettings = () => ({
  enabled: true,
  provider: "openai_compatible",
  model: "TeleAI/TeleSpeechASR",
  group_ids: [2],
  user_daily_audio_seconds: 1200,
  daily_audio_seconds_min: 1,
  daily_audio_seconds_max: 86400,
  managed: true,
  runtime_ready: true,
  catalog_available: true,
  selected_model_available: true,
  model_options: [
    {
      id: "TeleAI/TeleSpeechASR",
      available_group_count: 1,
      availability_checked: true,
    },
    {
      id: "FunAudioLLM/SenseVoiceSmall",
      available_group_count: 0,
      availability_checked: false,
    },
  ],
  limits: {
    max_upload_bytes: 10 * 1024 * 1024,
    max_duration_seconds: 120,
    max_concurrent_global: 8,
    max_concurrent_per_user: 1,
    user_requests_per_minute: 10,
    user_daily_audio_seconds: 1200,
    request_timeout_seconds: 45,
  },
});

const groups = [
  {
    id: 2,
    name: "Voice providers",
    platform: "openai",
    status: "active",
    subscription_type: "standard",
    active_account_count: 2,
  },
  {
    id: 3,
    name: "Backup voice providers",
    platform: "openai",
    status: "active",
    subscription_type: "standard",
    active_account_count: 1,
  },
];

function mountPanel(active = true) {
  return mount(TranscriptionSettingsPanel, {
    props: { active },
    global: {
      stubs: {
        Icon: true,
        Toggle: ToggleStub,
        RouterLink: {
          template: "<a><slot /></a>",
        },
      },
    },
  });
}

describe("TranscriptionSettingsPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getWebChatTranscriptionSettings.mockResolvedValue(baseSettings());
    getGroups.mockResolvedValue(groups);
    updateWebChatTranscriptionSettings.mockImplementation(async (payload) => ({
      ...baseSettings(),
      ...payload,
      managed: true,
    }));
  });

  it("loads only after its settings tab becomes active", async () => {
    const wrapper = mountPanel(false);
    await flushPromises();

    expect(getWebChatTranscriptionSettings).not.toHaveBeenCalled();
    expect(getGroups).not.toHaveBeenCalled();

    await wrapper.setProps({ active: true });
    await flushPromises();

    expect(getWebChatTranscriptionSettings).toHaveBeenCalledTimes(1);
    expect(getGroups).toHaveBeenCalledWith();
    expect(
      (wrapper.get("[data-testid='transcription-model']").element as HTMLInputElement)
        .value,
    ).toBe("TeleAI/TeleSpeechASR");
  });

  it("loads the daily audio quota without marking the panel dirty", async () => {
    const wrapper = mountPanel();
    await flushPromises();

    const quotaInput = wrapper.get(
      "[data-testid='transcription-daily-quota']",
    );
    expect(
      (quotaInput.element as HTMLInputElement).value,
    ).toBe("1200");
    expect(quotaInput.attributes("aria-describedby")?.split(" ")).toContain(
      "transcription-daily-quota-unit",
    );
    expect(wrapper.get("#transcription-daily-quota-unit").text()).toContain(
      "dailyQuotaUnit",
    );
    expect(
      (
        wrapper.get("[data-testid='transcription-save']")
          .element as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(wrapper.emitted("dirty-change")?.at(-1)).toEqual([false]);
  });

  it("treats a legacy null group list as an empty selection", async () => {
    getWebChatTranscriptionSettings.mockResolvedValue({
      ...baseSettings(),
      enabled: false,
      group_ids: null,
    });

    const wrapper = mountPanel();
    await flushPromises();

    expect(wrapper.find("[role='alert']").exists()).toBe(false);
    expect(
      (wrapper.get("[data-testid='transcription-enabled']").element as HTMLInputElement)
        .checked,
    ).toBe(false);
  });

  it("keeps a selected incompatible group visible so it can be removed", async () => {
    getGroups.mockResolvedValue([
      ...groups,
      {
        id: 9,
        name: "Changed platform",
        platform: "anthropic",
        status: "active",
        subscription_type: "exclusive",
        active_account_count: 1,
      },
    ]);
    getWebChatTranscriptionSettings.mockResolvedValue({
      ...baseSettings(),
      group_ids: [9],
    });

    const wrapper = mountPanel();
    await flushPromises();

    const incompatible = wrapper.get("[data-testid='transcription-group-9']");
    expect((incompatible.element as HTMLInputElement).checked).toBe(true);
    await incompatible.setValue(false);
    expect((incompatible.element as HTMLInputElement).checked).toBe(false);
  });

  it("saves the selected model and resource groups immediately", async () => {
    const wrapper = mountPanel();
    await flushPromises();

    await wrapper
      .get("[data-testid='transcription-model']")
      .setValue(" FunAudioLLM/SenseVoiceSmall ");
    await wrapper
      .get("[data-testid='transcription-group-3']")
      .setValue(true);
    expect(
      wrapper.get("[data-testid='transcription-model-availability']").text(),
    ).toContain("availabilityPending");
    await wrapper
      .get("[data-testid='transcription-model']")
      .trigger("keydown.enter");
    await flushPromises();

    expect(updateWebChatTranscriptionSettings).toHaveBeenCalledWith({
      enabled: true,
      model: "FunAudioLLM/SenseVoiceSmall",
      group_ids: [2, 3],
      user_daily_audio_seconds: 1200,
    });
    expect(showSuccess).toHaveBeenCalledTimes(1);
  });

  it("saves a changed daily audio quota in seconds", async () => {
    const wrapper = mountPanel();
    await flushPromises();

    await wrapper
      .get("[data-testid='transcription-daily-quota']")
      .setValue("3600");
    await wrapper.get("[data-testid='transcription-save']").trigger("click");
    await flushPromises();

    expect(updateWebChatTranscriptionSettings).toHaveBeenCalledWith({
      enabled: true,
      model: "TeleAI/TeleSpeechASR",
      group_ids: [2],
      user_daily_audio_seconds: 3600,
    });
    expect(wrapper.emitted("dirty-change")?.at(-1)).toEqual([false]);
  });

  it.each([
    ["zero", "0"],
    ["above the server maximum", "86401"],
    ["empty", ""],
    ["fractional", "1.5"],
  ])("rejects an invalid daily audio quota (%s)", async (_case, value) => {
    const wrapper = mountPanel();
    await flushPromises();

    const input = wrapper.get("[data-testid='transcription-daily-quota']");
    await input.setValue(value);

    expect(input.attributes("aria-invalid")).toBe("true");
    expect(
      (
        wrapper.get("[data-testid='transcription-save']")
          .element as HTMLButtonElement
      ).disabled,
    ).toBe(true);

    const error = wrapper.get(
      "[data-testid='transcription-daily-quota-error']",
    );
    expect(error.attributes("role")).toBe("alert");
    expect(error.attributes("id")).toBeTruthy();
    expect(input.attributes("aria-describedby")?.split(" ")).toContain(
      error.attributes("id"),
    );
    expect(input.attributes("aria-describedby")?.split(" ")).toContain(
      "transcription-daily-quota-unit",
    );
  });

  it("keeps an unavailable saved model visible and explains the outage", async () => {
    getWebChatTranscriptionSettings.mockResolvedValue({
      ...baseSettings(),
      model: "provider/expired-model",
      selected_model_available: false,
      model_options: [
        {
          id: "provider/expired-model",
          available_group_count: 0,
          availability_checked: true,
        },
      ],
    });

    const wrapper = mountPanel();
    await flushPromises();

    expect(
      (wrapper.get("[data-testid='transcription-model']").element as HTMLInputElement)
        .value,
    ).toBe("provider/expired-model");
    expect(wrapper.find("[data-testid='transcription-warning']").exists()).toBe(
      true,
    );
    expect(wrapper.get("[data-testid='transcription-status']").text()).toContain(
      "statusUnavailable",
    );
  });

  it("surfaces save failures without discarding the draft", async () => {
    updateWebChatTranscriptionSettings.mockRejectedValueOnce(
      new Error("provider unavailable"),
    );
    const wrapper = mountPanel();
    await flushPromises();

    await wrapper
      .get("[data-testid='transcription-model']")
      .setValue("FunAudioLLM/SenseVoiceSmall");
    await wrapper.get("[data-testid='transcription-save']").trigger("click");
    await flushPromises();

    expect(showError).toHaveBeenCalledTimes(1);
    expect(wrapper.find("[data-testid='transcription-save-error']").exists()).toBe(
      true,
    );
    expect(
      (wrapper.get("[data-testid='transcription-model']").element as HTMLInputElement)
        .value,
    ).toBe("FunAudioLLM/SenseVoiceSmall");
  });

  it("keeps a changed daily quota when saving fails", async () => {
    updateWebChatTranscriptionSettings.mockRejectedValueOnce(
      new Error("settings unavailable"),
    );
    const wrapper = mountPanel();
    await flushPromises();

    const input = wrapper.get("[data-testid='transcription-daily-quota']");
    await input.setValue("3600");
    await wrapper.get("[data-testid='transcription-save']").trigger("click");
    await flushPromises();

    expect((input.element as HTMLInputElement).value).toBe("3600");
    expect(wrapper.emitted("dirty-change")?.at(-1)).toEqual([true]);
    expect(
      wrapper.find("[data-testid='transcription-save-error']").exists(),
    ).toBe(true);
  });

  it("allows the daily quota to be changed while transcription is disabled", async () => {
    getWebChatTranscriptionSettings.mockResolvedValue({
      ...baseSettings(),
      enabled: false,
    });
    const wrapper = mountPanel();
    await flushPromises();

    await wrapper
      .get("[data-testid='transcription-daily-quota']")
      .setValue("3600");
    await wrapper.get("[data-testid='transcription-save']").trigger("click");
    await flushPromises();

    expect(updateWebChatTranscriptionSettings).toHaveBeenCalledWith({
      enabled: false,
      model: "TeleAI/TeleSpeechASR",
      group_ids: [2],
      user_daily_audio_seconds: 3600,
    });
  });

  it("does not repeat the editable daily quota in the safety-limit list", async () => {
    const wrapper = mountPanel();
    await flushPromises();

    expect(
      wrapper.findAll("[data-testid='transcription-daily-quota']"),
    ).toHaveLength(1);
    expect(wrapper.text()).not.toContain(
      "admin.settings.transcription.dailySeconds",
    );
  });

  it("lets an operator remove an inactive group and disable transcription", async () => {
    getWebChatTranscriptionSettings.mockResolvedValue({
      ...baseSettings(),
      group_ids: [99],
      selected_model_available: false,
      model_options: [
        {
          id: "TeleAI/TeleSpeechASR",
          available_group_count: 0,
          availability_checked: true,
        },
      ],
    });
    getGroups.mockResolvedValue([
      {
        ...groups[0],
        id: 99,
        name: "Expired provider group",
        status: "inactive",
      },
    ]);

    const wrapper = mountPanel();
    await flushPromises();

    expect(wrapper.get("[data-testid='transcription-group-99']").attributes("data-testid")).toBe(
      "transcription-group-99",
    );
    await wrapper
      .get("[data-testid='transcription-enabled']")
      .setValue(false);
    await wrapper.get("[data-testid='transcription-group-99']").setValue(false);
    await wrapper.get("[data-testid='transcription-save']").trigger("click");
    await flushPromises();

    expect(updateWebChatTranscriptionSettings).toHaveBeenCalledWith({
      enabled: false,
      model: "TeleAI/TeleSpeechASR",
      group_ids: [],
      user_daily_audio_seconds: 1200,
    });
  });
});
