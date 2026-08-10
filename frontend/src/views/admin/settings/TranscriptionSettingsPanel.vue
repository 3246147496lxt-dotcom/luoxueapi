<template>
  <section v-show="active" class="space-y-6" aria-labelledby="transcription-settings-title">
    <div class="card overflow-hidden">
      <header class="border-b border-gray-100 px-5 py-5 dark:border-dark-700 sm:px-6">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div class="flex min-w-0 items-start gap-3">
            <span class="transcription-icon" aria-hidden="true">
              <Icon name="microphone" size="md" :stroke-width="1.8" />
            </span>
            <div class="min-w-0">
              <h2 id="transcription-settings-title" class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.transcription.title") }}
              </h2>
              <p class="mt-1 max-w-3xl text-sm leading-6 text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.transcription.description") }}
              </p>
            </div>
          </div>

          <span v-if="settings" :class="statusBadgeClass" data-testid="transcription-status">
            <span class="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true"></span>
            {{ statusLabel }}
          </span>
        </div>
      </header>

      <div v-if="loading" class="flex min-h-64 items-center justify-center" role="status">
        <span class="h-7 w-7 animate-spin rounded-full border-2 border-primary-200 border-t-primary-600"></span>
        <span class="sr-only">{{ t("admin.settings.transcription.loading") }}</span>
      </div>

      <div v-else-if="loadError" class="p-5 sm:p-6">
        <div class="rounded-xl bg-red-50 p-4 text-sm text-red-800 dark:bg-red-950/30 dark:text-red-200" role="alert">
          <p class="font-medium">{{ t("admin.settings.transcription.loadFailed") }}</p>
          <p class="mt-1 leading-6">{{ loadError }}</p>
          <button type="button" class="btn btn-secondary mt-4" @click="loadSettings">
            <Icon name="refresh" size="sm" />
            {{ t("common.retry") }}
          </button>
        </div>
      </div>

      <div v-else-if="settings" class="divide-y divide-gray-100 dark:divide-dark-700">
        <div class="grid gap-5 p-5 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:p-6">
          <div>
            <label class="text-sm font-semibold text-gray-900 dark:text-white" for="transcription-enabled">
              {{ t("admin.settings.transcription.enabled") }}
            </label>
            <p class="mt-1 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.transcription.enabledHint") }}
            </p>
          </div>
          <Toggle id="transcription-enabled" v-model="draft.enabled" data-testid="transcription-enabled" />
        </div>

        <div class="grid gap-6 p-5 sm:p-6 lg:grid-cols-[minmax(0,1.15fr)_minmax(18rem,0.85fr)]">
          <div>
            <label class="input-label" for="transcription-model">
              {{ t("admin.settings.transcription.model") }}
              <span class="text-red-500">*</span>
            </label>
            <input
              id="transcription-model"
              v-model="draft.model"
              type="text"
              class="input"
              list="transcription-model-options"
              required
              aria-required="true"
              aria-describedby="transcription-model-hint"
              :aria-invalid="draft.model.trim().length === 0"
              autocomplete="off"
              spellcheck="false"
              :placeholder="t('admin.settings.transcription.modelPlaceholder')"
              data-testid="transcription-model"
              @keydown.enter.prevent="saveSettings"
            />
            <datalist id="transcription-model-options">
              <option v-for="option in settings.model_options" :key="option.id" :value="option.id"></option>
            </datalist>
            <p id="transcription-model-hint" class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.transcription.modelHint") }}
            </p>

            <div v-if="selectedModelOption" class="mt-3 flex flex-wrap items-center gap-2 text-xs">
              <span v-if="availabilityPending" class="status-chip-neutral" data-testid="transcription-model-availability">
                {{ t("admin.settings.transcription.availabilityPending") }}
              </span>
              <span
                v-else
                :class="selectedModelOption.available_group_count > 0 ? 'status-chip-ready' : 'status-chip-warning'"
                data-testid="transcription-model-availability"
              >
                {{
                  selectedModelOption.available_group_count > 0
                    ? t("admin.settings.transcription.availableInGroups", {
                        count: selectedModelOption.available_group_count,
                      })
                    : t("admin.settings.transcription.modelUnavailable")
                }}
              </span>
              <span v-if="!settings.managed" class="status-chip-neutral">
                {{ t("admin.settings.transcription.deploymentDefault") }}
              </span>
            </div>
          </div>

          <div class="rounded-xl bg-primary-50/70 p-4 dark:bg-primary-950/20">
            <p class="text-sm font-semibold text-primary-900 dark:text-primary-100">
              {{ t("admin.settings.transcription.failoverTitle") }}
            </p>
            <p class="mt-1.5 text-sm leading-6 text-primary-800/80 dark:text-primary-200/80">
              {{ t("admin.settings.transcription.failoverHint") }}
            </p>
            <router-link
              to="/admin/accounts?platform=openai"
              class="mt-3 inline-flex items-center gap-1 text-sm font-medium text-primary-700 hover:underline dark:text-primary-300"
            >
              {{ t("admin.settings.transcription.manageAccounts") }}
              <Icon name="arrowRight" size="sm" aria-hidden="true" />
            </router-link>
          </div>
        </div>

        <fieldset class="p-5 sm:p-6">
          <legend class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t("admin.settings.transcription.groups") }}
          </legend>
          <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.transcription.groupsHint") }}
          </p>

          <div v-if="displayGroups.length || missingSelectedGroupIDs.length" class="mt-4 grid gap-3 md:grid-cols-2">
            <label
              v-for="group in displayGroups"
              :key="group.id"
              class="group-option"
              :class="[
                isGroupSelected(group.id) && 'group-option-selected',
                !isEligibleGroup(group) && 'group-option-unavailable',
              ]"
            >
              <input
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="isGroupSelected(group.id)"
                :value="group.id"
                :data-testid="`transcription-group-${group.id}`"
                @change="toggleGroup(group.id)"
              />
              <span class="min-w-0">
                <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">{{ group.name }}</span>
                <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.transcription.groupMeta", { id: group.id, accounts: group.active_account_count ?? 0 }) }}
                </span>
                <span v-if="!isEligibleGroup(group)" class="mt-1 inline-flex text-xs font-semibold text-amber-700 dark:text-amber-300">
                  {{ t("admin.settings.transcription.groupUnavailable") }}
                </span>
              </span>
            </label>

            <label v-for="groupID in missingSelectedGroupIDs" :key="`missing-${groupID}`" class="group-option group-option-unavailable">
              <input
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                checked
                :value="groupID"
                :data-testid="`transcription-group-missing-${groupID}`"
                @change="toggleGroup(groupID)"
              />
              <span class="min-w-0">
                <span class="block text-sm font-medium text-gray-900 dark:text-white">
                  {{ t("admin.settings.transcription.missingGroup", { id: groupID }) }}
                </span>
                <span class="mt-1 inline-flex text-xs font-semibold text-amber-700 dark:text-amber-300">
                  {{ t("admin.settings.transcription.removeMissingGroup") }}
                </span>
              </span>
            </label>
          </div>

          <div v-else class="mt-4 rounded-xl bg-amber-50 p-4 text-sm text-amber-900 dark:bg-amber-950/25 dark:text-amber-100">
            <p class="font-medium">{{ t("admin.settings.transcription.noGroups") }}</p>
            <p class="mt-1 leading-6 text-amber-800/80 dark:text-amber-200/80">
              {{ t("admin.settings.transcription.noGroupsHint") }}
            </p>
          </div>

          <div v-if="groupsLoadError" class="mt-4 rounded-xl bg-amber-50 p-4 text-sm text-amber-900 dark:bg-amber-950/25 dark:text-amber-100" role="status">
            <p class="font-medium">{{ t("admin.settings.transcription.groupsLoadFailed") }}</p>
            <p class="mt-1 leading-6 text-amber-800/80 dark:text-amber-200/80">{{ groupsLoadError }}</p>
          </div>
        </fieldset>

        <div class="grid gap-5 p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.55fr)] lg:items-start">
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t("admin.settings.transcription.dailyQuotaTitle") }}
            </h3>
            <p class="mt-1 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.transcription.dailyQuotaHint") }}
            </p>
          </div>

          <div class="w-full lg:max-w-sm lg:justify-self-end">
            <label class="input-label" for="transcription-daily-quota">
              {{ t("admin.settings.transcription.dailyQuotaLabel") }}
            </label>
            <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3">
              <input
                id="transcription-daily-quota"
                v-model.number="draft.user_daily_audio_seconds"
                type="number"
                class="input min-w-0 tabular-nums"
                :min="dailyAudioSecondsMin"
                :max="dailyAudioSecondsMax"
                step="1"
                inputmode="numeric"
                :aria-invalid="!dailyAudioSecondsValid"
                :aria-describedby="dailyAudioSecondsValid
                  ? 'transcription-daily-quota-unit transcription-daily-quota-hint'
                  : 'transcription-daily-quota-unit transcription-daily-quota-error'"
                data-testid="transcription-daily-quota"
                @keydown.enter.prevent="saveSettings"
              />
              <span
                id="transcription-daily-quota-unit"
                class="whitespace-nowrap text-xs font-medium text-gray-500 dark:text-gray-400"
              >
                {{ t("admin.settings.transcription.dailyQuotaUnit") }}
              </span>
            </div>
            <p
              v-if="!dailyAudioSecondsValid"
              id="transcription-daily-quota-error"
              class="mt-2 text-xs leading-5 text-red-600 dark:text-red-300"
              role="alert"
              data-testid="transcription-daily-quota-error"
            >
              {{
                t("admin.settings.transcription.dailyQuotaInvalid", {
                  min: numberFormatter.format(dailyAudioSecondsMin),
                  max: numberFormatter.format(dailyAudioSecondsMax),
                })
              }}
            </p>
            <p
              v-else
              id="transcription-daily-quota-hint"
              class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400"
            >
              {{
                t("admin.settings.transcription.dailyQuotaEquivalent", {
                  minutes: dailyAudioMinutes,
                })
              }}
            </p>
          </div>
        </div>

        <div class="p-5 sm:p-6">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.transcription.safetyLimits") }}
              </h3>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.transcription.safetyLimitsHint") }}
              </p>
            </div>
            <span class="status-chip-neutral">{{ settings.provider }}</span>
          </div>
          <dl class="mt-4 grid grid-cols-2 gap-x-5 gap-y-4 sm:grid-cols-3 lg:grid-cols-6">
            <div v-for="item in limitItems" :key="item.label">
              <dt class="text-xs leading-5 text-gray-500 dark:text-gray-400">{{ item.label }}</dt>
              <dd class="mt-1 text-sm font-semibold tabular-nums text-gray-900 dark:text-white">{{ item.value }}</dd>
            </div>
          </dl>
        </div>

        <div
          v-if="draft.enabled && !availabilityPending && (!settings.runtime_ready || !settings.catalog_available || !currentDraftModelAvailable)"
          class="mx-5 mb-5 rounded-xl bg-amber-50 p-4 text-sm text-amber-900 dark:bg-amber-950/25 dark:text-amber-100 sm:mx-6 sm:mb-6"
          role="status"
          data-testid="transcription-warning"
        >
          <p class="font-medium">
            {{
              !settings.runtime_ready
                ? t("admin.settings.transcription.runtimeUnavailableTitle")
                : settings.catalog_available
                  ? t("admin.settings.transcription.unavailableTitle")
                  : t("admin.settings.transcription.catalogUnavailableTitle")
            }}
          </p>
          <p class="mt-1 leading-6 text-amber-800/80 dark:text-amber-200/80">
            {{
              !settings.runtime_ready
                ? t("admin.settings.transcription.runtimeUnavailableHint")
                : settings.catalog_available
                  ? t("admin.settings.transcription.unavailableHint")
                  : t("admin.settings.transcription.catalogUnavailableHint")
            }}
          </p>
        </div>

        <div
          v-if="saveError"
          class="mx-5 mb-5 rounded-xl bg-red-50 p-4 text-sm text-red-800 dark:bg-red-950/30 dark:text-red-200 sm:mx-6 sm:mb-6"
          role="alert"
          data-testid="transcription-save-error"
        >
          <p class="font-medium">{{ t("admin.settings.transcription.saveFailed") }}</p>
          <p class="mt-1 leading-6">{{ saveError }}</p>
        </div>

        <footer class="flex flex-col-reverse gap-3 bg-gray-50/70 px-5 py-4 dark:bg-dark-800/40 sm:flex-row sm:items-center sm:justify-between sm:px-6">
          <p class="text-xs leading-5 text-gray-500 dark:text-gray-400" aria-live="polite">
            {{ isDirty ? t("admin.settings.transcription.unsaved") : t("admin.settings.transcription.saved") }}
          </p>
          <div class="flex gap-2 sm:justify-end">
            <button type="button" class="btn btn-secondary flex-1 sm:flex-none" :disabled="saving" @click="reloadSettings">
              <Icon name="refresh" size="sm" />
              {{ t("admin.settings.transcription.reload") }}
            </button>
            <button
              type="button"
              class="btn btn-primary flex-1 sm:flex-none"
              :disabled="saving || !isDirty || !canSave"
              data-testid="transcription-save"
              @click="saveSettings"
            >
              <span v-if="saving" class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"></span>
              {{ saving ? t("common.saving") : t("admin.settings.transcription.save") }}
            </button>
          </div>
        </footer>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { adminAPI } from "@/api/admin";
import type { WebChatTranscriptionSettings } from "@/api/admin/settings";
import type { AdminGroup } from "@/types";
import Icon from "@/components/icons/Icon.vue";
import Toggle from "@/components/common/Toggle.vue";
import { useAppStore } from "@/stores";
import { extractApiErrorMessage } from "@/utils/apiError";

const props = defineProps<{
  active: boolean;
}>();
const emit = defineEmits<{
  "dirty-change": [dirty: boolean];
}>();

const { t, locale } = useI18n();
const appStore = useAppStore();
const loading = ref(false);
const saving = ref(false);
const loaded = ref(false);
const loadError = ref("");
const groupsLoadError = ref("");
const saveError = ref("");
const settings = ref<WebChatTranscriptionSettings | null>(null);
const groups = ref<AdminGroup[]>([]);
const draft = reactive({
  enabled: false,
  model: "",
  group_ids: [] as number[],
  user_daily_audio_seconds: "" as number | "",
});

const displayGroups = computed(() =>
  groups.value.filter(
    (group) =>
      draft.group_ids.includes(group.id) ||
      (group.platform === "openai" &&
        group.subscription_type === "standard" &&
        group.status === "active"),
  ),
);

function isEligibleGroup(group: AdminGroup): boolean {
  return (
    group.platform === "openai" &&
    group.subscription_type === "standard" &&
    group.status === "active"
  );
}

const normalizedDraftGroupIDs = computed(() =>
  [...new Set(draft.group_ids)].sort((a, b) => a - b),
);

const normalizedSavedGroupIDs = computed(() =>
  [...new Set(settings.value?.group_ids ?? [])].sort((a, b) => a - b),
);

const missingSelectedGroupIDs = computed(() => {
  const known = new Set(groups.value.map((group) => group.id));
  return normalizedDraftGroupIDs.value.filter((groupID) => !known.has(groupID));
});

const groupSelectionDirty = computed(
  () =>
    JSON.stringify(normalizedDraftGroupIDs.value) !==
    JSON.stringify(normalizedSavedGroupIDs.value),
);

const isDirty = computed(() => {
  const current = settings.value;
  if (!current) return false;
  return (
    draft.enabled !== current.enabled ||
    draft.model.trim() !== current.model ||
    Number(draft.user_daily_audio_seconds) !== current.user_daily_audio_seconds ||
    groupSelectionDirty.value
  );
});

const canSave = computed(
  () =>
    draft.model.trim().length > 0 &&
    dailyAudioSecondsValid.value &&
    (!draft.enabled || normalizedDraftGroupIDs.value.length > 0),
);

const selectedModelOption = computed(() => {
  const model = draft.model.trim();
  return settings.value?.model_options.find(
    (option) => option.id.trim() === model,
  );
});

const availabilityPending = computed(() => {
  const savedModel = settings.value?.model ?? "";
  return (
    groupSelectionDirty.value ||
    draft.model.trim() !== savedModel ||
    (selectedModelOption.value != null &&
      !selectedModelOption.value.availability_checked)
  );
});

const currentDraftModelAvailable = computed(
  () =>
    settings.value?.runtime_ready === true &&
    (selectedModelOption.value?.available_group_count ?? 0) > 0,
);

const statusLabel = computed(() => {
  if (!draft.enabled) return t("admin.settings.transcription.statusDisabled");
  if (availabilityPending.value)
    return t("admin.settings.transcription.statusPending");
  if (currentDraftModelAvailable.value) return t("admin.settings.transcription.statusReady");
  return t("admin.settings.transcription.statusUnavailable");
});

const statusBadgeClass = computed(() => {
  const base = "inline-flex shrink-0 items-center gap-2 self-start rounded-full px-3 py-1.5 text-xs font-semibold";
  if (!draft.enabled) return `${base} bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300`;
  if (availabilityPending.value)
    return `${base} bg-primary-50 text-primary-700 dark:bg-primary-950/30 dark:text-primary-300`;
  if (currentDraftModelAvailable.value) return `${base} bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300`;
  return `${base} bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300`;
});

const numberFormatter = computed(() =>
  new Intl.NumberFormat(locale.value.startsWith("zh") ? "zh-CN" : "en-US"),
);

const dailyAudioSecondsMin = computed(
  () => settings.value?.daily_audio_seconds_min ?? 1,
);
const dailyAudioSecondsMax = computed(
  () => settings.value?.daily_audio_seconds_max ?? 86_400,
);
const dailyAudioSecondsValid = computed(() => {
  const value = Number(draft.user_daily_audio_seconds);
  return (
    draft.user_daily_audio_seconds !== "" &&
    Number.isInteger(value) &&
    value >= dailyAudioSecondsMin.value &&
    value <= dailyAudioSecondsMax.value
  );
});
const dailyAudioMinutes = computed(() => {
  const value = Number(draft.user_daily_audio_seconds);
  if (!Number.isFinite(value)) return "0";
  return new Intl.NumberFormat(
    locale.value.startsWith("zh") ? "zh-CN" : "en-US",
    { maximumFractionDigits: 2 },
  ).format(value / 60);
});

const limitItems = computed(() => {
  const limits = settings.value?.limits;
  if (!limits) return [];
  return [
    {
      label: t("admin.settings.transcription.maxUpload"),
      value: `${Math.round(limits.max_upload_bytes / 1024 / 1024)} MB`,
    },
    {
      label: t("admin.settings.transcription.maxDuration"),
      value: t("admin.settings.transcription.secondsValue", {
        value: limits.max_duration_seconds,
      }),
    },
    {
      label: t("admin.settings.transcription.globalConcurrency"),
      value: numberFormatter.value.format(limits.max_concurrent_global),
    },
    {
      label: t("admin.settings.transcription.userConcurrency"),
      value: numberFormatter.value.format(limits.max_concurrent_per_user),
    },
    {
      label: t("admin.settings.transcription.userRPM"),
      value: numberFormatter.value.format(limits.user_requests_per_minute),
    },
    {
      label: t("admin.settings.transcription.requestTimeout"),
      value: t("admin.settings.transcription.secondsValue", {
        value: limits.request_timeout_seconds,
      }),
    },
  ];
});

function applySettings(value: WebChatTranscriptionSettings): void {
  const userDailyAudioSeconds =
    value.user_daily_audio_seconds ??
    value.limits?.user_daily_audio_seconds ??
    1200;
  settings.value = {
    ...value,
    user_daily_audio_seconds: userDailyAudioSeconds,
    daily_audio_seconds_min: value.daily_audio_seconds_min ?? 1,
    daily_audio_seconds_max: value.daily_audio_seconds_max ?? 86_400,
  };
  draft.enabled = value.enabled;
  draft.model = value.model;
  // Be defensive with older servers that encoded an empty Go slice as null.
  draft.group_ids = [...(value.group_ids ?? [])];
  draft.user_daily_audio_seconds = userDailyAudioSeconds;
}

function isGroupSelected(groupID: number): boolean {
  return draft.group_ids.includes(groupID);
}

function toggleGroup(groupID: number): void {
  draft.group_ids = isGroupSelected(groupID)
    ? draft.group_ids.filter((id) => id !== groupID)
    : [...draft.group_ids, groupID];
}

async function loadSettings(): Promise<void> {
  loading.value = true;
  loadError.value = "";
  groupsLoadError.value = "";
  saveError.value = "";
  const [settingsResult, groupsResult] = await Promise.allSettled([
    adminAPI.settings.getWebChatTranscriptionSettings(),
    adminAPI.groups.getAllIncludingInactive(),
  ]);
  if (settingsResult.status === "rejected") {
    loadError.value = extractApiErrorMessage(
      settingsResult.reason,
      t("admin.settings.transcription.loadFailed"),
    );
    loading.value = false;
    return;
  }

  applySettings(settingsResult.value);
  loaded.value = true;
  if (groupsResult.status === "fulfilled") {
    groups.value = groupsResult.value;
  } else {
    groups.value = [];
    groupsLoadError.value = extractApiErrorMessage(
      groupsResult.reason,
      t("admin.settings.transcription.groupsLoadFailed"),
    );
  }
  loading.value = false;
}

async function reloadSettings(): Promise<void> {
  if (
    isDirty.value &&
    !window.confirm(t("admin.settings.transcription.discardConfirm"))
  ) {
    return;
  }
  await loadSettings();
}

async function saveSettings(): Promise<void> {
  if (!canSave.value || saving.value) return;
  saving.value = true;
  saveError.value = "";
  try {
    const updated = await adminAPI.settings.updateWebChatTranscriptionSettings({
      enabled: draft.enabled,
      model: draft.model.trim(),
      group_ids: normalizedDraftGroupIDs.value,
      user_daily_audio_seconds: Number(draft.user_daily_audio_seconds),
    });
    applySettings(updated);
    appStore.showSuccess(t("admin.settings.transcription.saveSuccess"));
  } catch (error) {
    saveError.value = extractApiErrorMessage(
      error,
      t("admin.settings.transcription.saveFailed"),
    );
    appStore.showError(saveError.value);
  } finally {
    saving.value = false;
  }
}

watch(
  () => props.active,
  (active) => {
    if (active && !loaded.value && !loading.value) {
      void loadSettings();
    }
  },
  { immediate: true },
);

watch(
  isDirty,
  (dirty) => {
    emit("dirty-change", dirty);
  },
  { immediate: true },
);

watch(
  () => [
    draft.enabled,
    draft.model,
    draft.user_daily_audio_seconds,
    ...normalizedDraftGroupIDs.value,
  ],
  () => {
    saveError.value = "";
  },
);
</script>

<style scoped>
.transcription-icon {
  display: inline-flex;
  height: 2.5rem;
  width: 2.5rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 0.75rem;
  background: color-mix(in srgb, var(--color-primary-100, #ede9fe) 86%, white);
  color: var(--color-primary-700, #6d28d9);
}

.group-option {
  display: flex;
  min-width: 0;
  cursor: pointer;
  align-items: flex-start;
  gap: 0.75rem;
  border-radius: 0.75rem;
  background: rgb(249 250 251 / 0.8);
  padding: 0.875rem;
  outline: 1px solid rgb(229 231 235);
  outline-offset: -1px;
  transition:
    background-color 160ms ease,
    outline-color 160ms ease;
}

.group-option:hover {
  background: rgb(245 243 255 / 0.7);
  outline-color: rgb(196 181 253);
}

.group-option:focus-within {
  outline: 2px solid rgb(124 58 237);
  outline-offset: 2px;
}

.group-option-selected {
  background: rgb(245 243 255);
  outline-color: rgb(167 139 250);
}

.group-option-unavailable {
  background: rgb(255 251 235 / 0.85);
  outline-color: rgb(252 211 77);
}

.status-chip-ready,
.status-chip-warning,
.status-chip-neutral {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  padding: 0.25rem 0.625rem;
  font-weight: 600;
}

.status-chip-ready {
  background: rgb(236 253 245);
  color: rgb(4 120 87);
}

.status-chip-warning {
  background: rgb(255 251 235);
  color: rgb(180 83 9);
}

.status-chip-neutral {
  background: rgb(243 244 246);
  color: rgb(75 85 99);
}

:global(.dark) .transcription-icon {
  background: rgb(76 29 149 / 0.35);
  color: rgb(196 181 253);
}

:global(.dark) .group-option {
  background: rgb(31 41 55 / 0.55);
  outline-color: rgb(55 65 81);
}

:global(.dark) .group-option:hover,
:global(.dark) .group-option-selected {
  background: rgb(76 29 149 / 0.2);
  outline-color: rgb(109 40 217);
}

:global(.dark) .group-option-unavailable {
  background: rgb(120 53 15 / 0.2);
  outline-color: rgb(180 83 9);
}

:global(.dark) .status-chip-ready {
  background: rgb(6 78 59 / 0.35);
  color: rgb(110 231 183);
}

:global(.dark) .status-chip-warning {
  background: rgb(120 53 15 / 0.35);
  color: rgb(252 211 77);
}

:global(.dark) .status-chip-neutral {
  background: rgb(55 65 81);
  color: rgb(209 213 219);
}

@media (prefers-reduced-motion: reduce) {
  .group-option {
    transition: none;
  }
}
</style>
