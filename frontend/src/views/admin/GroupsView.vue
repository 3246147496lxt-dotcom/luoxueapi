<template>
  <AppLayout
    variant="home-clay"
    :content-mode="editorMode === 'list' ? 'workbench' : 'contained'"
  >
    <GroupOperationsCockpit
      v-if="editorMode === 'list'"
      data-admin-page-kind="ops"
      :groups="groups"
      :overview-groups="overviewGroups"
      :loading="loading"
      :overview-loading="overviewLoading"
      :overview-error="overviewError"
      :usage-map="usageMap"
      :usage-loading="usageLoading"
      :usage-error="usageError"
      :capacity-map="capacityMap"
      :capacity-loading="capacityLoading"
      :capacity-error="capacityError"
      :refreshing="dashboardRefreshing"
      :search-query="searchQuery"
      :filters="filters"
      :platform-options="platformFilterOptions"
      :status-options="statusOptions"
      :exclusive-options="exclusiveOptions"
      :column-preferences="columnPreferences"
      :pagination="pagination"
      :last-updated-at="lastUpdatedAt"
      @refresh="refreshGroupsDashboard"
      @create="openCreateModal"
      @search="handleCockpitSearch"
      @filter="handleCockpitFilter"
      @toggle-column="toggleColumn"
      @sort-order="openSortModal"
      @edit="handleEdit"
      @rate-multipliers="handleRateMultipliers"
      @rpm-overrides="handleRPMOverrides"
      @delete="handleDelete"
      @page-change="handlePageChange"
      @page-size-change="handlePageSizeChange"
    />

    <!-- Create Group Workspace -->
    <GroupEditorShell
      v-else-if="editorMode === 'create'"
      data-admin-page-kind="form"
      :title="t('admin.groups.createGroup')"
      :description="t('admin.groups.editor.createDescription')"
      :context-label="t('admin.groups.platforms.' + createForm.platform)"
      :back-label="t('admin.groups.editor.backToGroups')"
      :navigation-label="t('admin.groups.editor.navigationLabel')"
      :sections="editorSections"
      :active-section="activeEditorSection"
      :loading="editorLoading"
      :loading-label="t('admin.groups.editor.loading')"
      :dirty="editorDirty"
      :dirty-label="t('admin.groups.editor.dirty')"
      :saved-label="t('admin.groups.editor.saved')"
      @back="closeCreateModal"
      @select-section="selectEditorSection"
    >
      <GroupForm
        mode="create"
        :draft="createForm"
        :active-editor-section="activeEditorSection"
        :editor-validation-error-field="editorValidationErrorField"
        :editor-validation-error-message="editorValidationErrorMessage"
        :platform-options="platformOptions"
        :status-options="editStatusOptions"
        :subscription-type-options="subscriptionTypeOptions"
        :copy-accounts-group-options="copyAccountsGroupOptions"
        :fallback-group-options="fallbackGroupOptions"
        :invalid-request-fallback-options="invalidRequestFallbackOptions"
        :models-list-state="createModelsListState"
        :models-list-loading="createModelsListLoading"
        :models-list-selected-count="createModelsListSelectedCount"
        :image-final-price-preview="createImageFinalPricePreview"
        :video-final-price-preview="createVideoFinalPricePreview"
        :web-search-final-price-preview="createWebSearchFinalPricePreview"
        :model-routing-rules="createModelRoutingRules"
        :account-search-keyword="accountSearchKeyword"
        :account-search-results="accountSearchResults"
        :show-account-dropdown="showAccountDropdown"
        :actions="createGroupFormActions"
        @submit="handleCreateGroup"
      />

      <template #footer>
        <div class="flex justify-end gap-3 pt-4">
          <button
            @click="closeCreateModal"
            type="button"
            class="btn btn-secondary"
          >
            {{ t("common.cancel") }}
          </button>
          <button
            type="submit"
            form="create-group-form"
            :disabled="submitting"
            class="btn btn-primary"
            data-tour="group-form-submit"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ submitting ? t("admin.groups.creating") : t("common.create") }}
          </button>
        </div>
      </template>
    </GroupEditorShell>

    <!-- Edit Group Workspace -->
    <GroupEditorShell
      v-else
      data-admin-page-kind="form"
      :title="t('admin.groups.editGroup')"
      :description="t('admin.groups.editor.editDescription')"
      :context-label="editorContextLabel"
      :back-label="t('admin.groups.editor.backToGroups')"
      :navigation-label="t('admin.groups.editor.navigationLabel')"
      :sections="editorSections"
      :active-section="activeEditorSection"
      :loading="editorLoading"
      :loading-label="t('admin.groups.editor.loading')"
      :dirty="editorDirty"
      :dirty-label="t('admin.groups.editor.dirty')"
      :saved-label="t('admin.groups.editor.saved')"
      @back="closeEditModal"
      @select-section="selectEditorSection"
    >
      <div
        v-if="editorLoadError"
        ref="editorLoadErrorRef"
        class="rounded-2xl border border-red-200 bg-red-50 p-5 text-red-800 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200"
        role="alert"
        tabindex="-1"
        data-test="group-editor-load-error"
      >
        <p class="font-semibold">{{ editorLoadError }}</p>
        <div class="mt-4 flex flex-wrap gap-3">
          <button
            v-if="editorLoadRetryable"
            type="button"
            class="btn btn-primary"
            @click="retryEditorLoad"
          >
            {{ t('admin.groups.editor.retry') }}
          </button>
          <button type="button" class="btn btn-secondary" @click="closeEditModal">
            {{ t('admin.groups.editor.backToGroups') }}
          </button>
        </div>
      </div>
      <GroupForm
        v-else-if="editingGroup"
        mode="edit"
        :draft="editForm"
        :active-editor-section="activeEditorSection"
        :editor-validation-error-field="editorValidationErrorField"
        :editor-validation-error-message="editorValidationErrorMessage"
        :platform-options="platformOptions"
        :status-options="editStatusOptions"
        :subscription-type-options="subscriptionTypeOptions"
        :copy-accounts-group-options="copyAccountsGroupOptionsForEdit"
        :fallback-group-options="fallbackGroupOptionsForEdit"
        :invalid-request-fallback-options="invalidRequestFallbackOptionsForEdit"
        :models-list-state="editModelsListState"
        :models-list-loading="editModelsListLoading"
        :models-list-selected-count="editModelsListSelectedCount"
        :image-final-price-preview="editImageFinalPricePreview"
        :video-final-price-preview="editVideoFinalPricePreview"
        :web-search-final-price-preview="editWebSearchFinalPricePreview"
        :model-routing-rules="editModelRoutingRules"
        :account-search-keyword="accountSearchKeyword"
        :account-search-results="accountSearchResults"
        :show-account-dropdown="showAccountDropdown"
        :actions="editGroupFormActions"
        @submit="handleUpdateGroup"
      />

      <template #footer>
        <div v-if="editingGroup && !editorLoadError" class="flex justify-end gap-3">
          <button
            @click="closeEditModal"
            type="button"
            class="btn btn-secondary"
          >
            {{ t("common.cancel") }}
          </button>
          <button
            type="submit"
            form="edit-group-form"
            :disabled="submitting"
            class="btn btn-primary"
            data-tour="group-form-submit"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ submitting ? t("admin.groups.updating") : t("common.update") }}
          </button>
        </div>
      </template>
    </GroupEditorShell>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.groups.deleteGroup')"
      :message="deleteConfirmMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Sort Order Modal -->
    <BaseDialog
      :show="showSortModal"
      :title="t('admin.groups.sortOrder')"
      width="normal"
      @close="closeSortModal"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t("admin.groups.sortOrderHint") }}
        </p>
        <VueDraggable
          v-model="sortableGroups"
          :animation="200"
          class="space-y-2"
        >
          <div
            v-for="group in sortableGroups"
            :key="group.id"
            class="flex cursor-grab items-center gap-3 rounded-lg border border-gray-200 bg-white p-3 transition-shadow hover:shadow-md active:cursor-grabbing dark:border-dark-600 dark:bg-dark-700"
          >
            <div class="text-gray-400">
              <Icon name="menu" size="md" />
            </div>
            <div class="flex-1">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ group.name }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                <span
                  :class="[
                    'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
                    group.platform === 'anthropic'
                      ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                      : group.platform === 'openai'
                        ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                        : group.platform === 'antigravity'
                          ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                          : group.platform === 'grok'
                            ? 'bg-zinc-200 text-zinc-800 dark:bg-zinc-700 dark:text-zinc-100'
                            : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
                  ]"
                >
                  {{ t("admin.groups.platforms." + group.platform) }}
                </span>
              </div>
            </div>
            <div class="text-sm text-gray-400">#{{ group.id }}</div>
          </div>
        </VueDraggable>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3 pt-4">
          <button
            @click="closeSortModal"
            type="button"
            class="btn btn-secondary"
          >
            {{ t("common.cancel") }}
          </button>
          <button
            @click="saveSortOrder"
            :disabled="sortSubmitting"
            class="btn btn-primary"
          >
            <svg
              v-if="sortSubmitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ sortSubmitting ? t("common.saving") : t("common.save") }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Rate Multipliers Modal -->
    <GroupRateMultipliersModal
      :show="showRateMultipliersModal"
      :group="rateMultipliersGroup"
      @close="showRateMultipliersModal = false"
      @success="refreshGroupsDashboard"
    />

    <!-- Group RPM Overrides Modal -->
    <GroupRPMOverridesModal
      :show="showRPMOverridesModal"
      :group="rpmOverridesGroup"
      @close="showRPMOverridesModal = false"
      @success="refreshGroupsDashboard"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import {
  ref,
  reactive,
  computed,
  nextTick,
  onMounted,
  onUnmounted,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import {
  onBeforeRouteLeave,
  onBeforeRouteUpdate,
  useRoute,
  useRouter,
} from "vue-router";
import { useAppStore } from "@/stores/app";
import { useOnboardingStore } from "@/stores/onboarding";
import { adminAPI } from "@/api/admin";
import type { AdminGroup, GroupPlatform } from "@/types";
import type { Column } from "@/components/common/types";
import AppLayout from "@/components/layout/AppLayout.vue";
import BaseDialog from "@/components/common/BaseDialog.vue";
import ConfirmDialog from "@/components/common/ConfirmDialog.vue";
import Icon from "@/components/icons/Icon.vue";
import GroupRateMultipliersModal from "@/components/admin/group/GroupRateMultipliersModal.vue";
import GroupRPMOverridesModal from "@/components/admin/group/GroupRPMOverridesModal.vue";
import GroupEditorShell from "@/components/admin/group/GroupEditorShell.vue";
import GroupOperationsCockpit from "@/components/admin/group/GroupOperationsCockpit.vue";
import GroupForm from "./groups/GroupForm.vue";
import { VueDraggable } from "vue-draggable-plus";
import { createStableObjectKeyResolver } from "@/utils/stableObjectKey";
import {
  extractApiErrorMessage,
  extractApiErrorStatus,
} from "@/utils/apiError";
import { useKeyedDebouncedSearch } from "@/composables/useKeyedDebouncedSearch";
import { getPersistedPageSize } from "@/composables/usePersistedPageSize";
import {
  messagesDispatchFormStateToConfig,
  resetMessagesDispatchFormState,
} from "./groupsMessagesDispatch";
import {
  buildModelsListConfig,
  createModelsListState as createInitialModelsListState,
  setModelsListCandidates,
} from "./groupsModelsList";
import { createModelsListCandidatesTracker } from "./groupsModelsListCandidates";
import { normalizeSupportedModelScopesForPlatform } from "./groupsSupportedModelScopes";
import {
  getDefaultImagePreviewPrice,
  getDefaultVideoPreviewPrice,
} from "./groupsImagePricing";
import {
  GROUP_EDITOR_SECTION_IDS,
  isCanonicalGroupEditorSection,
  normalizeGroupEditorSection,
  parsePositiveGroupId,
  type GroupEditorSectionId,
} from "./groups/groupEditorRoute";
import {
  createGroupDraft,
  groupToDraft,
  replaceGroupDraft,
  serializeCreateGroupDraft,
  serializeUpdateGroupDraft,
  type GroupDraft,
} from "./groups/groupDraft";
import { useGroupEditor } from "./groups/useGroupEditor";
import type {
  GroupFormActions,
  GroupFormAccount,
  GroupFormRoutingRule,
} from "./groups/groupForm";

const { t } = useI18n();
const appStore = useAppStore();
const onboardingStore = useOnboardingStore();
const route = useRoute();
const router = useRouter();

type GroupEditorMode = "list" | "create" | "edit";

const editorMode = computed<GroupEditorMode>(() => {
  if (route.name === "AdminGroupCreate") return "create";
  if (route.name === "AdminGroupEdit") return "edit";
  return "list";
});

const activeEditorSection = computed(() =>
  normalizeGroupEditorSection(route.query.section),
);

const editorSections = computed(() =>
  GROUP_EDITOR_SECTION_IDS.map((id) => ({
    id,
    label: t(`admin.groups.editor.sections.${id}.label`),
    description: t(`admin.groups.editor.sections.${id}.description`),
    to: {
      path: route.path,
      query: { ...route.query, section: id },
    },
  })),
);

const editorLoading = ref(false);
const editorLoadError = ref("");
const editorLoadRetryable = ref(true);
const editorLoadErrorRef = ref<HTMLElement | null>(null);
const editorBaseline = ref("");
type EditorValidationErrorField = "name" | "rateMultiplier";
const editorValidationErrorField = ref<EditorValidationErrorField | null>(null);
const editorValidationErrorMessage = ref("");

const ALWAYS_VISIBLE_COLUMNS = new Set(["name", "actions"]);
// Default hidden columns (hidden on first load / after schema bumps).
const DEFAULT_HIDDEN_COLUMNS = ["id"];
const HIDDEN_COLUMNS_KEY = "group-hidden-columns";
// Bump when adding new default-hidden columns so existing admins pick them up once.
const COLUMN_SETTINGS_VERSION_KEY = "group-column-settings-version";
const COLUMN_SETTINGS_VERSION = 2;
const VERSION_NEW_HIDDEN_COLUMNS: Record<number, string[]> = {
  2: ["id"],
};

const allColumns = computed<Column[]>(() => [
  { key: "name", label: t("admin.groups.columns.name"), sortable: true },
  { key: "id", label: t("admin.groups.columns.id"), sortable: true },
  {
    key: "platform",
    label: t("admin.groups.columns.platform"),
    sortable: true,
  },
  {
    key: "billing_type",
    label: t("admin.groups.columns.billingType"),
    sortable: true,
  },
  {
    key: "rate_multiplier",
    label: t("admin.groups.columns.rateMultiplier"),
    sortable: true,
  },
  {
    key: "is_exclusive",
    label: t("admin.groups.columns.type"),
    sortable: true,
  },
  {
    key: "account_count",
    label: t("admin.groups.columns.accounts"),
    sortable: true,
  },
  {
    key: "capacity",
    label: t("admin.groups.columns.capacity"),
    sortable: false,
  },
  { key: "usage", label: t("admin.groups.columns.usage"), sortable: false },
  { key: "status", label: t("admin.groups.columns.status"), sortable: true },
  { key: "actions", label: t("admin.groups.columns.actions"), sortable: false },
]);

const toggleableColumns = computed(() =>
  allColumns.value.filter((col) => !ALWAYS_VISIBLE_COLUMNS.has(col.key)),
);
const hiddenColumns = reactive<Set<string>>(new Set());

const getValidHiddenColumnKeys = () =>
  new Set(toggleableColumns.value.map((col) => col.key));

const loadSavedColumns = () => {
  hiddenColumns.clear();
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY);
    const validKeys = getValidHiddenColumnKeys();

    if (saved) {
      const parsed = JSON.parse(saved);
      if (Array.isArray(parsed)) {
        parsed
          .filter(
            (key): key is string =>
              typeof key === "string" && validKeys.has(key),
          )
          .forEach((key) => hiddenColumns.add(key));
      }

      // Existing admins: auto-hide columns newly added as default-hidden.
      const storedVersion = Number(
        localStorage.getItem(COLUMN_SETTINGS_VERSION_KEY) ?? "1",
      );
      if (storedVersion < COLUMN_SETTINGS_VERSION) {
        let mutated = false;
        for (let v = storedVersion + 1; v <= COLUMN_SETTINGS_VERSION; v++) {
          for (const key of VERSION_NEW_HIDDEN_COLUMNS[v] ?? []) {
            if (validKeys.has(key) && !hiddenColumns.has(key)) {
              hiddenColumns.add(key);
              mutated = true;
            }
          }
        }
        if (mutated) {
          saveColumnsToStorage();
        } else {
          localStorage.setItem(
            COLUMN_SETTINGS_VERSION_KEY,
            String(COLUMN_SETTINGS_VERSION),
          );
        }
      }
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach((key) => {
        if (validKeys.has(key)) hiddenColumns.add(key);
      });
      saveColumnsToStorage();
    }
  } catch (error) {
    console.error("Failed to load group column settings:", error);
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key));
  }
};

const saveColumnsToStorage = () => {
  try {
    const validKeys = getValidHiddenColumnKeys();
    const keys = [...hiddenColumns].filter((key) => validKeys.has(key));
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify(keys));
    localStorage.setItem(
      COLUMN_SETTINGS_VERSION_KEY,
      String(COLUMN_SETTINGS_VERSION),
    );
  } catch (error) {
    console.error("Failed to save group column settings:", error);
  }
};

const isColumnVisible = (key: string) => !hiddenColumns.has(key);
const columnPreferences = computed(() =>
  toggleableColumns.value.map((column) => ({
    key: column.key,
    label: column.label,
    visible: isColumnVisible(column.key),
  })),
);

const toggleColumn = (key: string) => {
  const validKeys = getValidHiddenColumnKeys();
  if (!validKeys.has(key)) return;

  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key);
  } else {
    hiddenColumns.add(key);
  }
  saveColumnsToStorage();
};

if (typeof window !== "undefined") {
  loadSavedColumns();
}

// Filter options
const statusOptions = computed(() => [
  { value: "", label: t("admin.groups.cockpit.activeStatus") },
  { value: "active", label: t("admin.accounts.status.active") },
  { value: "inactive", label: t("admin.accounts.status.inactive") },
]);

const exclusiveOptions = computed(() => [
  { value: "", label: t("admin.groups.cockpit.accessType") },
  { value: "true", label: t("admin.groups.exclusive") },
  { value: "false", label: t("admin.groups.nonExclusive") },
]);

const platformOptions = computed(() => [
  { value: "anthropic", label: "Anthropic" },
  { value: "openai", label: "OpenAI" },
  { value: "gemini", label: "Gemini" },
  { value: "antigravity", label: "Antigravity" },
  { value: "grok", label: "Grok" },
]);

const platformFilterOptions = computed(() => [
  { value: "", label: t("admin.groups.cockpit.allPlatforms") },
  { value: "anthropic", label: "Anthropic" },
  { value: "openai", label: "OpenAI" },
  { value: "gemini", label: "Gemini" },
  { value: "antigravity", label: "Antigravity" },
  { value: "grok", label: "Grok" },
]);

const editStatusOptions = computed(() => [
  { value: "active", label: t("admin.accounts.status.active") },
  { value: "inactive", label: t("admin.accounts.status.inactive") },
]);

const subscriptionTypeOptions = computed(() => [
  { value: "standard", label: t("admin.groups.subscription.standard") },
  { value: "subscription", label: t("admin.groups.subscription.subscription") },
]);

// 降级分组选项（创建时）- 仅包含 anthropic 平台且未启用 claude_code_only 的分组
const fallbackGroupOptions = computed(() => {
  const options: { value: number | null; label: string }[] = [
    { value: null, label: t("admin.groups.claudeCode.noFallback") },
  ];
  const eligibleGroups = groups.value.filter(
    (g) =>
      g.platform === "anthropic" &&
      !g.claude_code_only &&
      g.status === "active",
  );
  eligibleGroups.forEach((g) => {
    options.push({ value: g.id, label: g.name });
  });
  return options;
});

// 降级分组选项（编辑时）- 排除自身
const fallbackGroupOptionsForEdit = computed(() => {
  const options: { value: number | null; label: string }[] = [
    { value: null, label: t("admin.groups.claudeCode.noFallback") },
  ];
  const currentId = editingGroup.value?.id;
  const eligibleGroups = groups.value.filter(
    (g) =>
      g.platform === "anthropic" &&
      !g.claude_code_only &&
      g.status === "active" &&
      g.id !== currentId,
  );
  eligibleGroups.forEach((g) => {
    options.push({ value: g.id, label: g.name });
  });
  return options;
});

// 无效请求兜底分组选项（创建时）- 仅包含 anthropic 平台、非订阅且未配置兜底的分组
const invalidRequestFallbackOptions = computed(() => {
  const options: { value: number | null; label: string }[] = [
    { value: null, label: t("admin.groups.invalidRequestFallback.noFallback") },
  ];
  const eligibleGroups = groups.value.filter(
    (g) =>
      g.platform === "anthropic" &&
      g.status === "active" &&
      g.subscription_type !== "subscription" &&
      g.fallback_group_id_on_invalid_request === null,
  );
  eligibleGroups.forEach((g) => {
    options.push({ value: g.id, label: g.name });
  });
  return options;
});

// 无效请求兜底分组选项（编辑时）- 排除自身
const invalidRequestFallbackOptionsForEdit = computed(() => {
  const options: { value: number | null; label: string }[] = [
    { value: null, label: t("admin.groups.invalidRequestFallback.noFallback") },
  ];
  const currentId = editingGroup.value?.id;
  const eligibleGroups = groups.value.filter(
    (g) =>
      g.platform === "anthropic" &&
      g.status === "active" &&
      g.subscription_type !== "subscription" &&
      g.fallback_group_id_on_invalid_request === null &&
      g.id !== currentId,
  );
  eligibleGroups.forEach((g) => {
    options.push({ value: g.id, label: g.name });
  });
  return options;
});

// 复制账号的源分组选项（创建时）- 仅包含相同平台且有账号的分组
const copyAccountsGroupOptions = computed(() => {
  const eligibleGroups = groups.value.filter(
    (g) => g.platform === createForm.platform && (g.account_count || 0) > 0,
  );
  return eligibleGroups.map((g) => ({
    value: g.id,
    label: `${g.name} (${t("admin.groups.accountsCount", { count: g.account_count || 0 })})`,
  }));
});

// 复制账号的源分组选项（编辑时）- 仅包含相同平台且有账号的分组，排除自身
const copyAccountsGroupOptionsForEdit = computed(() => {
  const currentId = editingGroup.value?.id;
  const eligibleGroups = groups.value.filter(
    (g) =>
      g.platform === editForm.platform &&
      (g.account_count || 0) > 0 &&
      g.id !== currentId,
  );
  return eligibleGroups.map((g) => ({
    value: g.id,
    label: `${g.name} (${t("admin.groups.accountsCount", { count: g.account_count || 0 })})`,
  }));
});

const groups = ref<AdminGroup[]>([]);
const overviewGroups = ref<AdminGroup[]>([]);
const loading = ref(false);
const overviewLoading = ref(false);
const overviewError = ref(false);
type GroupUsageSummary = {
  today_cost: number;
  total_cost: number;
};

const usageMap = ref<Map<number, GroupUsageSummary>>(new Map());
const usageLoading = ref(false);
const usageError = ref(false);
const capacityMap = ref<
  Map<
    number,
    {
      concurrencyUsed: number;
      concurrencyMax: number;
      sessionsUsed: number;
      sessionsMax: number;
      rpmUsed: number;
      rpmMax: number;
    }
  >
>(new Map());
const capacityLoading = ref(false);
const capacityError = ref(false);
const lastUpdatedAt = ref<Date | null>(null);
const dashboardRefreshing = computed(
  () =>
    loading.value ||
    overviewLoading.value ||
    usageLoading.value ||
    capacityLoading.value,
);
const searchQuery = ref("");
const filters = reactive({
  platform: "",
  status: "",
  is_exclusive: "",
});

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0,
});
const sortState = reactive({
  sort_by: "sort_order",
  sort_order: "asc" as "asc" | "desc",
});

let abortController: AbortController | null = null;

const showDeleteDialog = ref(false);
const showSortModal = ref(false);
const submitting = ref(false);
const sortSubmitting = ref(false);
const editingGroup = ref<AdminGroup | null>(null);
const deletingGroup = ref<AdminGroup | null>(null);
const showRateMultipliersModal = ref(false);
const rateMultipliersGroup = ref<AdminGroup | null>(null);
const showRPMOverridesModal = ref(false);
const rpmOverridesGroup = ref<AdminGroup | null>(null);
const sortableGroups = ref<AdminGroup[]>([]);
const createModelsListState = reactive(createInitialModelsListState());
const editModelsListState = reactive(createInitialModelsListState());
const createModelsListLoading = ref(false);
const editModelsListLoading = ref(false);
const modelsListCandidatesTracker = createModelsListCandidatesTracker();
const createModelsListSelectedCount = computed(
  () => createModelsListState.items.filter((item) => item.selected).length,
);
const editModelsListSelectedCount = computed(
  () => editModelsListState.items.filter((item) => item.selected).length,
);

const createForm = reactive<GroupDraft>(createGroupDraft());

// 共享表单与路由搜索使用同一组窄类型。
type SimpleAccount = GroupFormAccount;
type ModelRoutingRule = GroupFormRoutingRule;

// 创建表单的模型路由规则
const createModelRoutingRules = ref<ModelRoutingRule[]>([]);

// 编辑表单的模型路由规则
const editModelRoutingRules = ref<ModelRoutingRule[]>([]);

// 规则对象稳定 key（避免使用 index 导致状态错位）
const resolveCreateRuleKey =
  createStableObjectKeyResolver<ModelRoutingRule>("create-rule");
const resolveEditRuleKey =
  createStableObjectKeyResolver<ModelRoutingRule>("edit-rule");

const getCreateRuleRenderKey = (rule: ModelRoutingRule) =>
  resolveCreateRuleKey(rule);
const getEditRuleRenderKey = (rule: ModelRoutingRule) =>
  resolveEditRuleKey(rule);

const getCreateRuleSearchKey = (rule: ModelRoutingRule) =>
  `create-${resolveCreateRuleKey(rule)}`;
const getEditRuleSearchKey = (rule: ModelRoutingRule) =>
  `edit-${resolveEditRuleKey(rule)}`;

const getRuleSearchKey = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  return isEdit ? getEditRuleSearchKey(rule) : getCreateRuleSearchKey(rule);
};

// 账号搜索相关状态
const accountSearchKeyword = ref<Record<string, string>>({});
const accountSearchResults = ref<Record<string, SimpleAccount[]>>({});
const showAccountDropdown = ref<Record<string, boolean>>({});

const clearAccountSearchStateByKey = (key: string) => {
  delete accountSearchKeyword.value[key];
  delete accountSearchResults.value[key];
  delete showAccountDropdown.value[key];
};

const clearAllAccountSearchState = () => {
  accountSearchKeyword.value = {};
  accountSearchResults.value = {};
  showAccountDropdown.value = {};
};

const accountSearchRunner = useKeyedDebouncedSearch<SimpleAccount[]>({
  delay: 300,
  search: async (keyword, { signal }) => {
    const res = await adminAPI.accounts.list(
      1,
      20,
      {
        search: keyword,
        platform: "anthropic",
      },
      { signal },
    );
    return res.items.map((account) => ({ id: account.id, name: account.name }));
  },
  onSuccess: (key, result) => {
    accountSearchResults.value[key] = result;
  },
  onError: (key) => {
    accountSearchResults.value[key] = [];
  },
});

// 搜索账号（仅限 anthropic 平台）
const searchAccounts = (key: string) => {
  accountSearchRunner.trigger(key, accountSearchKeyword.value[key] || "");
};

const searchAccountsByRule = (
  rule: ModelRoutingRule,
  isEdit: boolean = false,
) => {
  searchAccounts(getRuleSearchKey(rule, isEdit));
};

// 选择账号
const selectAccount = (
  rule: ModelRoutingRule,
  account: SimpleAccount,
  isEdit: boolean = false,
) => {
  if (!rule) return;

  // 检查是否已选择
  if (!rule.accounts.some((a) => a.id === account.id)) {
    rule.accounts.push(account);
  }

  // 清空搜索
  const key = getRuleSearchKey(rule, isEdit);
  accountSearchKeyword.value[key] = "";
  showAccountDropdown.value[key] = false;
};

// 移除已选账号
const removeSelectedAccount = (
  rule: ModelRoutingRule,
  accountId: number,
  _isEdit: boolean = false,
) => {
  if (!rule) return;

  rule.accounts = rule.accounts.filter((a) => a.id !== accountId);
};

// 处理账号搜索输入框聚焦
const onAccountSearchFocus = (
  rule: ModelRoutingRule,
  isEdit: boolean = false,
) => {
  const key = getRuleSearchKey(rule, isEdit);
  showAccountDropdown.value[key] = true;
  // 如果没有搜索结果，触发一次搜索
  if (!accountSearchResults.value[key]?.length) {
    searchAccounts(key);
  }
};

// 添加创建表单的路由规则
const addCreateRoutingRule = () => {
  createModelRoutingRules.value.push({ pattern: "", accounts: [] });
};

// 删除创建表单的路由规则
const removeCreateRoutingRule = (rule: ModelRoutingRule) => {
  const index = createModelRoutingRules.value.indexOf(rule);
  if (index === -1) return;

  const key = getCreateRuleSearchKey(rule);
  accountSearchRunner.clearKey(key);
  clearAccountSearchStateByKey(key);
  createModelRoutingRules.value.splice(index, 1);
};

// 添加编辑表单的路由规则
const addEditRoutingRule = () => {
  editModelRoutingRules.value.push({ pattern: "", accounts: [] });
};

// 删除编辑表单的路由规则
const removeEditRoutingRule = (rule: ModelRoutingRule) => {
  const index = editModelRoutingRules.value.indexOf(rule);
  if (index === -1) return;

  const key = getEditRuleSearchKey(rule);
  accountSearchRunner.clearKey(key);
  clearAccountSearchStateByKey(key);
  editModelRoutingRules.value.splice(index, 1);
};

const createGroupFormActions: GroupFormActions = {
  clearValidationError: (field) => clearEditorValidationError(field),
  getRuleRenderKey: (rule) => getCreateRuleRenderKey(rule),
  getRuleSearchKey: (rule) => getCreateRuleSearchKey(rule),
  searchAccountsByRule: (rule) => searchAccountsByRule(rule),
  onAccountSearchFocus: (rule) => onAccountSearchFocus(rule),
  selectAccount: (rule, account) => selectAccount(rule, account),
  removeSelectedAccount: (rule, accountId) =>
    removeSelectedAccount(rule, accountId),
  addRoutingRule: () => addCreateRoutingRule(),
  removeRoutingRule: (rule) => removeCreateRoutingRule(rule),
};

const editGroupFormActions: GroupFormActions = {
  clearValidationError: (field) => clearEditorValidationError(field),
  getRuleRenderKey: (rule) => getEditRuleRenderKey(rule),
  getRuleSearchKey: (rule) => getEditRuleSearchKey(rule),
  searchAccountsByRule: (rule) => searchAccountsByRule(rule, true),
  onAccountSearchFocus: (rule) => onAccountSearchFocus(rule, true),
  selectAccount: (rule, account) => selectAccount(rule, account, true),
  removeSelectedAccount: (rule, accountId) =>
    removeSelectedAccount(rule, accountId, true),
  addRoutingRule: () => addEditRoutingRule(),
  removeRoutingRule: (rule) => removeEditRoutingRule(rule),
};

const resetModelsListState = (
  state: typeof createModelsListState,
  config?: Parameters<typeof createInitialModelsListState>[0],
) => {
  const fresh = createInitialModelsListState(config);
  state.enabled = fresh.enabled;
  state.savedModels = fresh.savedModels;
  state.items = fresh.items;
};

const loadModelsListCandidates = async (
  mode: "create" | "edit",
  groupID: number,
  platform: GroupPlatform,
) => {
  const request = { mode, groupID, platform };
  const requestID = modelsListCandidatesTracker.next(request);
  const state = mode === "create" ? createModelsListState : editModelsListState;
  const loadingRef = mode === "create" ? createModelsListLoading : editModelsListLoading;
  loadingRef.value = true;
  try {
    const models = await adminAPI.groups.getModelsListCandidates(groupID, platform);
    if (!modelsListCandidatesTracker.isCurrent(requestID, request)) {
      return;
    }
    setModelsListCandidates(state, models);
  } catch (error) {
    if (!modelsListCandidatesTracker.isCurrent(requestID, request)) {
      return;
    }
    console.error("Error loading group models list candidates:", error);
  } finally {
    if (modelsListCandidatesTracker.isCurrent(requestID, request)) {
      loadingRef.value = false;
    }
  }
};

// 将 UI 格式的路由规则转换为 API 格式
const convertRoutingRulesToApiFormat = (
  rules: ModelRoutingRule[],
): Record<string, number[]> | null => {
  const result: Record<string, number[]> = {};
  let hasValidRules = false;

  for (const rule of rules) {
    const pattern = rule.pattern.trim();
    if (!pattern) continue;

    const accountIds = rule.accounts.map((a) => a.id).filter((id) => id > 0);

    if (accountIds.length > 0) {
      result[pattern] = accountIds;
      hasValidRules = true;
    }
  }

  return hasValidRules ? result : null;
};

// 将 API 格式的路由规则转换为 UI 格式（需要加载账号名称）
const convertApiFormatToRoutingRules = async (
  apiFormat: Record<string, number[]> | null,
): Promise<ModelRoutingRule[]> => {
  if (!apiFormat) return [];

  return Promise.all(
    Object.entries(apiFormat).map(async ([pattern, accountIds]) => {
      const accounts = await Promise.all(
        accountIds.map(async (id): Promise<SimpleAccount> => {
          try {
            const account = await adminAPI.accounts.getById(id);
            return { id: account.id, name: account.name };
          } catch {
            return { id, name: `#${id}` };
          }
        }),
      );
      return { pattern, accounts };
    }),
  );
};

const editForm = reactive<GroupDraft>(createGroupDraft());

const groupEditor = useGroupEditor({
  drafts: { createDraft: createForm, editDraft: editForm },
  baseline: editorBaseline,
  snapshotExtras: (mode) => {
    const modelsState =
      mode === "create" ? createModelsListState : editModelsListState;
    const routingRules =
      mode === "create"
        ? createModelRoutingRules.value
        : editModelRoutingRules.value;
    return {
      modelsListConfig: buildModelsListConfig(modelsState),
      modelRouting: routingRules.map((rule) => ({
        pattern: rule.pattern,
        accountIds: rule.accounts.map((account) => account.id),
      })),
    };
  },
});

const buildEditorSnapshot = groupEditor.snapshot;

const captureEditorBaseline = async (
  mode: Exclude<GroupEditorMode, "list">,
) => {
  await nextTick();
  if (editorMode.value === mode) {
    groupEditor.capture(mode);
  }
};

const editorDirty = computed(() => {
  if (
    editorMode.value === "list" ||
    editorLoading.value ||
    !editorBaseline.value
  ) {
    return false;
  }
  return groupEditor.isDirty(editorMode.value);
});

const shouldWarnUnsavedEditor = computed(
  () =>
    editorMode.value !== "list" &&
    editorDirty.value &&
    !editorLoading.value &&
    !submitting.value,
);

const confirmUnsavedEditorNavigation = () => {
  if (!shouldWarnUnsavedEditor.value) return true;
  return window.confirm(t("admin.groups.editor.leaveDirtyConfirm"));
};

const handleEditorBeforeUnload = (event: BeforeUnloadEvent) => {
  if (!shouldWarnUnsavedEditor.value) return;
  event.preventDefault();
  event.returnValue = "";
};

onBeforeRouteLeave(() => confirmUnsavedEditorNavigation());

onBeforeRouteUpdate((to, from) => {
  const sameEditorIdentity =
    to.name === from.name &&
    String(to.params.id ?? "") === String(from.params.id ?? "");
  if (sameEditorIdentity) return true;
  return confirmUnsavedEditorNavigation();
});

const editorContextLabel = computed(() => {
  if (!editingGroup.value) return "";
  return `#${editingGroup.value.id} · ${t(
    `admin.groups.platforms.${editingGroup.value.platform}`,
  )}`;
});

type ImagePricingFormState = {
  platform: GroupPlatform;
  allow_image_generation: boolean;
  allow_batch_image_generation: boolean;
  rate_multiplier: GroupDraft["rate_multiplier"];
  image_rate_independent: boolean;
  image_rate_multiplier: GroupDraft["image_rate_multiplier"];
  batch_image_discount_multiplier: GroupDraft["batch_image_discount_multiplier"];
  batch_image_hold_multiplier: GroupDraft["batch_image_hold_multiplier"];
  image_price_1k: number | string | null;
  image_price_2k: number | string | null;
  image_price_4k: number | string | null;
  peak_rate_enabled: boolean;
  peak_start: string;
  peak_end: string;
  peak_rate_multiplier: GroupDraft["peak_rate_multiplier"];
};

type VideoPricingFormState = {
  platform: GroupPlatform;
  rate_multiplier: GroupDraft["rate_multiplier"];
  video_rate_independent: boolean;
  video_rate_multiplier: GroupDraft["video_rate_multiplier"];
  video_price_480p: number | string | null;
  video_price_720p: number | string | null;
  video_price_1080p: number | string | null;
};

const imagePricingTiers = [
  { key: "image_price_1k", label: "1K" },
  { key: "image_price_2k", label: "2K" },
  { key: "image_price_4k", label: "4K" },
] as const;

const videoPricingTiers = [
  { key: "video_price_480p", label: "480p" },
  { key: "video_price_720p", label: "720p" },
  { key: "video_price_1080p", label: "1080p" },
] as const;

const normalizePreviewNumber = (value: number | string | null | undefined, fallback = 0) => {
  if (value === null || value === undefined || value === "") {
    return fallback;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const parsePreviewPrice = (value: number | string | null | undefined) => {
  if (value === null || value === undefined || value === "") {
    return null;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : null;
};

const formatImagePricePreview = (value: number | string | null | undefined) => {
  if (value === null || value === undefined || value === "") {
    return t("admin.groups.imagePricing.notConfigured");
  }
  const price = Number(value);
  if (!Number.isFinite(price) || price < 0) {
    return t("admin.groups.imagePricing.notConfigured");
  }
  return `$${price.toFixed(6).replace(/0+$/, "").replace(/\.$/, "")}`;
};

const formatVideoPricePreview = (value: number | string | null | undefined) => {
  if (value === null || value === undefined || value === "") {
    return t("admin.groups.videoPricing.notConfigured");
  }
  const price = Number(value);
  if (!Number.isFinite(price) || price < 0) {
    return t("admin.groups.videoPricing.notConfigured");
  }
  return `$${price.toFixed(6).replace(/0+$/, "").replace(/\.$/, "")}`;
};

const buildImageFinalPricePreview = (form: ImagePricingFormState) => {
  const imageMultiplier = form.image_rate_independent
    ? normalizePreviewNumber(form.image_rate_multiplier, 1)
    : normalizePreviewNumber(form.rate_multiplier, 1);
  const multiplier = imageMultiplier;
  return imagePricingTiers.map((tier) => {
    const basePrice =
      parsePreviewPrice(form[tier.key]) ??
      getDefaultImagePreviewPrice(form.platform, tier.key);
    return {
      label: tier.label,
      value: basePrice !== null
        ? formatImagePricePreview(basePrice * multiplier)
        : t("admin.groups.imagePricing.notConfigured"),
    };
  });
};

const buildVideoFinalPricePreview = (form: VideoPricingFormState) => {
  const multiplier = form.video_rate_independent
    ? normalizePreviewNumber(form.video_rate_multiplier, 1)
    : normalizePreviewNumber(form.rate_multiplier, 1);
  return videoPricingTiers.map((tier) => {
    const basePrice =
      parsePreviewPrice(form[tier.key]) ??
      getDefaultVideoPreviewPrice(form.platform, tier.key);
    return {
      label: tier.label,
      value: basePrice !== null
        ? formatVideoPricePreview(basePrice * multiplier)
        : t("admin.groups.videoPricing.notConfigured"),
    };
  });
};

const createImageFinalPricePreview = computed(() =>
  buildImageFinalPricePreview(createForm),
);
const editImageFinalPricePreview = computed(() =>
  buildImageFinalPricePreview(editForm),
);
const createVideoFinalPricePreview = computed(() =>
  buildVideoFinalPricePreview(createForm),
);
const editVideoFinalPricePreview = computed(() =>
  buildVideoFinalPricePreview(editForm),
);

// Codex 网页搜索单次默认价（与后端 defaultWebSearchPricePerCall 一致，官方 $10/1000 次）
const DEFAULT_WEB_SEARCH_PRICE_PER_CALL = 0.01;

const buildWebSearchFinalPricePreview = (form: {
  web_search_price_per_call: number | string | null;
  rate_multiplier: number | string | null;
}) => {
  const basePrice =
    parsePreviewPrice(form.web_search_price_per_call) ??
    DEFAULT_WEB_SEARCH_PRICE_PER_CALL;
  const multiplier = normalizePreviewNumber(form.rate_multiplier, 1);
  return formatImagePricePreview(basePrice * multiplier);
};

const createWebSearchFinalPricePreview = computed(() =>
  buildWebSearchFinalPricePreview(createForm),
);
const editWebSearchFinalPricePreview = computed(() =>
  buildWebSearchFinalPricePreview(editForm),
);

const resetDisabledBatchImagePricing = (
  form: Pick<
    ImagePricingFormState,
    "platform" | "allow_image_generation" | "allow_batch_image_generation" | "batch_image_discount_multiplier" | "batch_image_hold_multiplier"
  >,
) => {
  if (form.platform !== "gemini" || !form.allow_image_generation) {
    form.allow_batch_image_generation = false;
  }
  if (!form.allow_batch_image_generation) {
    form.batch_image_discount_multiplier = 0.5;
    form.batch_image_hold_multiplier = 0.6;
  }
};

// 根据分组类型返回不同的删除确认消息
const deleteConfirmMessage = computed(() => {
  if (!deletingGroup.value) {
    return "";
  }
  if (deletingGroup.value.subscription_type === "subscription") {
    return t("admin.groups.deleteConfirmSubscription", {
      name: deletingGroup.value.name,
    });
  }
  return t("admin.groups.deleteConfirm", { name: deletingGroup.value.name });
});

const loadGroups = async () => {
  if (abortController) {
    abortController.abort();
  }
  const currentController = new AbortController();
  abortController = currentController;
  const { signal } = currentController;
  loading.value = true;
  try {
    const response = await adminAPI.groups.list(
      pagination.page,
      pagination.page_size,
      {
        platform: (filters.platform as GroupPlatform) || undefined,
        status: filters.status as any,
        is_exclusive: filters.is_exclusive
          ? filters.is_exclusive === "true"
          : undefined,
        search: searchQuery.value.trim() || undefined,
        sort_by: sortState.sort_by,
        sort_order: sortState.sort_order,
      },
      { signal },
    );
    if (signal.aborted) return;
    groups.value = response.items;
    pagination.total = response.total;
    pagination.pages = response.pages;
  } catch (error: any) {
    if (
      signal.aborted ||
      error?.name === "AbortError" ||
      error?.code === "ERR_CANCELED"
    ) {
      return;
    }
    appStore.showError(t("admin.groups.failedToLoad"));
    console.error("Error loading groups:", error);
  } finally {
    if (abortController === currentController && !signal.aborted) {
      loading.value = false;
    }
  }
};

const loadOverviewGroups = async () => {
  overviewLoading.value = true;
  overviewError.value = false;
  try {
    overviewGroups.value = await adminAPI.groups.getAllIncludingInactive();
  } catch (error) {
    overviewGroups.value = [];
    overviewError.value = true;
    console.error("Error loading group overview:", error);
  } finally {
    overviewLoading.value = false;
  }
};

let editorReferenceLoadId = 0;
const loadEditorReferenceGroups = async () => {
  const requestId = ++editorReferenceLoadId;
  try {
    const references = await adminAPI.groups.getAllIncludingInactive();
    if (requestId !== editorReferenceLoadId || editorMode.value === "list") {
      return;
    }
    groups.value = references;
  } catch (error) {
    if (requestId !== editorReferenceLoadId || editorMode.value === "list") {
      return;
    }
    console.error("Error loading group editor references:", error);
  }
};

const loadUsageSummary = async () => {
  usageLoading.value = true;
  usageError.value = false;
  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    const data = await adminAPI.groups.getUsageSummary(tz);
    const map = new Map<number, GroupUsageSummary>();
    for (const item of data) {
      map.set(item.group_id, {
        today_cost: item.today_cost,
        total_cost: item.total_cost,
      });
    }
    usageMap.value = map;
  } catch (error) {
    usageMap.value = new Map();
    usageError.value = true;
    console.error("Error loading group usage summary:", error);
  } finally {
    usageLoading.value = false;
  }
};

const loadCapacitySummary = async () => {
  capacityLoading.value = true;
  capacityError.value = false;
  try {
    const data = await adminAPI.groups.getCapacitySummary();
    const map = new Map<
      number,
      {
        concurrencyUsed: number;
        concurrencyMax: number;
        sessionsUsed: number;
        sessionsMax: number;
        rpmUsed: number;
        rpmMax: number;
      }
    >();
    for (const item of data) {
      map.set(item.group_id, {
        concurrencyUsed: item.concurrency_used,
        concurrencyMax: item.concurrency_max,
        sessionsUsed: item.sessions_used,
        sessionsMax: item.sessions_max,
        rpmUsed: item.rpm_used,
        rpmMax: item.rpm_max,
      });
    }
    capacityMap.value = map;
  } catch (error) {
    capacityMap.value = new Map();
    capacityError.value = true;
    console.error("Error loading group capacity summary:", error);
  } finally {
    capacityLoading.value = false;
  }
};

const refreshGroupsDashboard = async () => {
  await Promise.allSettled([
    loadGroups(),
    loadOverviewGroups(),
    loadUsageSummary(),
    loadCapacitySummary(),
  ]);
  lastUpdatedAt.value = new Date();
};

let searchTimeout: ReturnType<typeof setTimeout>;
const handleSearch = () => {
  clearTimeout(searchTimeout);
  searchTimeout = setTimeout(() => {
    pagination.page = 1;
    loadGroups();
  }, 300);
};

const handleCockpitSearch = (value: string) => {
  searchQuery.value = value;
  handleSearch();
};

const handleCockpitFilter = (
  key: "platform" | "status" | "is_exclusive",
  value: string,
) => {
  filters[key] = value;
  pagination.page = 1;
  void loadGroups();
};

const handlePageChange = (page: number) => {
  pagination.page = page;
  loadGroups();
};

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize;
  pagination.page = 1;
  loadGroups();
};

const openCreateModal = () => {
  void router.push({
    name: "AdminGroupCreate",
    query: { section: "general" },
  });
};

const resetCreateForm = () => {
  createModelRoutingRules.value.forEach((rule) => {
    accountSearchRunner.clearKey(getCreateRuleSearchKey(rule));
  });
  clearAllAccountSearchState();
  replaceGroupDraft(createForm, createGroupDraft());
  resetModelsListState(createModelsListState);
  createModelRoutingRules.value = [];
};

const closeCreateModal = () => {
  void router.push({ name: "AdminGroups" });
};

const clearEditorValidationError = (field?: EditorValidationErrorField) => {
  if (field && editorValidationErrorField.value !== field) return;
  editorValidationErrorField.value = null;
  editorValidationErrorMessage.value = "";
};

const focusEditorValidationError = async (
  mode: Exclude<GroupEditorMode, "list">,
  field: EditorValidationErrorField,
) => {
  if (activeEditorSection.value !== "general") {
    await router.push({
      query: { ...route.query, section: "general" },
    });
  }
  await nextTick();
  const id = `${mode}-group-${field === "name" ? "name" : "rate-multiplier"}`;
  document.getElementById(id)?.focus();
};

const validateEditorBasics = async (
  mode: Exclude<GroupEditorMode, "list">,
): Promise<boolean> => {
  const form = mode === "create" ? createForm : editForm;
  if (!form.name.trim()) {
    editorValidationErrorField.value = "name";
    editorValidationErrorMessage.value = t("admin.groups.nameRequired");
    appStore.showError(editorValidationErrorMessage.value);
    await focusEditorValidationError(mode, "name");
    return false;
  }

  const rateMultiplier = Number(form.rate_multiplier);
  if (!Number.isFinite(rateMultiplier) || rateMultiplier < 0.001) {
    editorValidationErrorField.value = "rateMultiplier";
    editorValidationErrorMessage.value = t(
      "admin.groups.editor.rateMultiplierInvalid",
    );
    appStore.showError(editorValidationErrorMessage.value);
    await focusEditorValidationError(mode, "rateMultiplier");
    return false;
  }

  clearEditorValidationError();
  return true;
};

const handleCreateGroup = async () => {
  if (!(await validateEditorBasics("create"))) return;
  submitting.value = true;
  try {
    const createPayload = serializeCreateGroupDraft(createForm, {
      modelRouting: convertRoutingRulesToApiFormat(
        createModelRoutingRules.value,
      ),
      modelsListConfig: buildModelsListConfig(createModelsListState),
      supportedModelScopes: normalizeSupportedModelScopesForPlatform(
        createForm.platform,
        createForm.supported_model_scopes,
      ),
      messagesDispatchModelConfig:
        createForm.platform === "openai"
          ? messagesDispatchFormStateToConfig(createForm)
          : undefined,
    });
    const created = await adminAPI.groups.create(createPayload);
    const shouldAdvanceOnboarding = onboardingStore.isCurrentStep(
      '[data-tour="group-form-submit"]',
    );
    editorBaseline.value = buildEditorSnapshot("create");
    appStore.showSuccess(t("admin.groups.groupCreated"));
    await router.replace({
      name: "AdminGroupEdit",
      params: { id: String(created.id) },
      query: { ...route.query, section: activeEditorSection.value },
    });
    // Only advance tour if active, on submit step, and creation succeeded
    if (shouldAdvanceOnboarding) {
      onboardingStore.nextStep(500);
    }
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t("admin.groups.failedToCreate")));
    console.error("Error creating group:", error);
    // Don't advance tour on error
  } finally {
    submitting.value = false;
  }
};

const hydrateEditGroup = async (
  group: AdminGroup,
  isCurrent: () => boolean = () => true,
): Promise<boolean> => {
  editingGroup.value = group;
  replaceGroupDraft(editForm, groupToDraft(group));
  resetModelsListState(editModelsListState, group.models_list_config);
  // 加载模型路由规则（异步加载账号名称）
  const routingRules = await convertApiFormatToRoutingRules(
    group.model_routing,
  );
  if (!isCurrent()) return false;
  editModelRoutingRules.value = routingRules;
  await loadModelsListCandidates("edit", group.id, group.platform);
  return isCurrent();
};

const handleEdit = (group: AdminGroup) => {
  void router.push({
    name: "AdminGroupEdit",
    params: { id: String(group.id) },
    query: { section: "general" },
  });
};

const resetEditForm = () => {
  editModelRoutingRules.value.forEach((rule) => {
    accountSearchRunner.clearKey(getEditRuleSearchKey(rule));
  });
  clearAllAccountSearchState();
  editingGroup.value = null;
  editModelRoutingRules.value = [];
  replaceGroupDraft(editForm, createGroupDraft());
  resetModelsListState(editModelsListState);
};

const closeEditModal = () => {
  void router.push({ name: "AdminGroups" });
};

const editorSectionAnchorId = (section: GroupEditorSectionId) =>
  `group-editor-section-${section}`;

const scrollToEditorSection = async (
  section: GroupEditorSectionId,
  behavior: "auto" | "smooth" = "smooth",
) => {
  await nextTick();
  await new Promise<void>((resolve) => {
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => resolve());
    });
  });
  const target =
    section === "general"
      ? document.querySelector(".group-editor-shell")
      : document.getElementById(editorSectionAnchorId(section));
  if (!target) return;
  const reduceMotion =
    typeof window !== "undefined" &&
    window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
  const offset =
    section === "general" ? 88 : window.innerWidth < 1024 ? 172 : 100;
  const top = Math.max(
    0,
    target.getBoundingClientRect().top + window.scrollY - offset,
  );
  window.scrollTo({
    top,
    behavior: reduceMotion ? "auto" : behavior,
  });
};

const selectEditorSection = async (section: string) => {
  if (!isCanonicalGroupEditorSection(section)) return;
  if (activeEditorSection.value === section) {
    await scrollToEditorSection(section);
    return;
  }
  await router.push({
    query: { ...route.query, section },
  });
  await scrollToEditorSection(section);
};

let editorLoadRequestId = 0;

const prepareCreateEditor = async () => {
  const requestId = ++editorLoadRequestId;
  editorLoading.value = true;
  editorLoadError.value = "";
  editorLoadRetryable.value = true;
  editorBaseline.value = "";
  clearEditorValidationError();
  resetCreateForm();
  await loadModelsListCandidates("create", 0, createForm.platform);
  if (requestId !== editorLoadRequestId || editorMode.value !== "create") {
    return;
  }
  editorLoading.value = false;
  await captureEditorBaseline("create");
  await scrollToEditorSection(activeEditorSection.value, "auto");
};

const prepareEditEditor = async () => {
  const requestId = ++editorLoadRequestId;
  const groupId = parsePositiveGroupId(route.params.id);
  editorLoading.value = true;
  editorLoadError.value = "";
  editorLoadRetryable.value = true;
  editorBaseline.value = "";
  clearEditorValidationError();
  resetEditForm();

  if (groupId === null) {
    editorLoading.value = false;
    editorLoadRetryable.value = false;
    editorLoadError.value = t("admin.groups.editor.notFound");
    await nextTick();
    editorLoadErrorRef.value?.focus();
    return;
  }

  try {
    const group = await adminAPI.groups.getById(groupId);
    if (requestId !== editorLoadRequestId || editorMode.value !== "edit") {
      return;
    }
    const hydrated = await hydrateEditGroup(
      group,
      () => requestId === editorLoadRequestId && editorMode.value === "edit",
    );
    if (!hydrated) {
      return;
    }
    editorLoading.value = false;
    await captureEditorBaseline("edit");
    await scrollToEditorSection(activeEditorSection.value, "auto");
  } catch (error) {
    if (requestId !== editorLoadRequestId || editorMode.value !== "edit") {
      return;
    }
    editorLoading.value = false;
    const status = extractApiErrorStatus(error);
    if (status === 403) {
      editorLoadRetryable.value = false;
      editorLoadError.value = t("admin.groups.editor.forbidden");
    } else if (status === 404) {
      editorLoadRetryable.value = false;
      editorLoadError.value = t("admin.groups.editor.notFound");
    } else if (status === 0) {
      editorLoadRetryable.value = true;
      editorLoadError.value = t("admin.groups.editor.networkError");
    } else if (status !== undefined && status >= 500) {
      editorLoadRetryable.value = true;
      editorLoadError.value = t("admin.groups.editor.serverError");
    } else {
      editorLoadRetryable.value = true;
      editorLoadError.value = extractApiErrorMessage(
        error,
        t("admin.groups.editor.loadFailed"),
      );
    }
    await nextTick();
    editorLoadErrorRef.value?.focus();
    console.error("Error loading group editor:", error);
  }
};

const retryEditorLoad = () => {
  if (editorMode.value === "edit") {
    void prepareEditEditor();
  }
};

const editorRouteKey = computed(
  () => `${editorMode.value}:${String(route.params.id ?? "")}`,
);

watch(
  editorRouteKey,
  () => {
    if (editorMode.value === "create") {
      void loadEditorReferenceGroups();
      void prepareCreateEditor();
      return;
    }
    if (editorMode.value === "edit") {
      void loadEditorReferenceGroups();
      void prepareEditEditor();
      return;
    }
    editorReferenceLoadId += 1;
    editorLoadRequestId += 1;
    editorLoading.value = false;
    editorLoadError.value = "";
    editorLoadRetryable.value = true;
    editorBaseline.value = "";
    clearEditorValidationError();
    resetCreateForm();
    resetEditForm();
    void refreshGroupsDashboard();
  },
  { immediate: true },
);

watch(
  () => `${editorMode.value}:${String(route.query.section ?? "")}`,
  () => {
    if (editorMode.value === "list") return;
    const section = route.query.section;
    if (!isCanonicalGroupEditorSection(section)) {
      void router.replace({
        query: { ...route.query, section: activeEditorSection.value },
      });
      return;
    }
    void scrollToEditorSection(section, "auto");
  },
  { immediate: true },
);

const handleUpdateGroup = async () => {
  if (!editingGroup.value) return;
  if (!(await validateEditorBasics("edit"))) return;

  submitting.value = true;
  try {
    const updatePayload = serializeUpdateGroupDraft(editForm, {
      modelRouting: convertRoutingRulesToApiFormat(
        editModelRoutingRules.value,
      ),
      modelsListConfig: buildModelsListConfig(editModelsListState),
      supportedModelScopes: normalizeSupportedModelScopesForPlatform(
        editForm.platform,
        editForm.supported_model_scopes,
      ),
      messagesDispatchModelConfig:
        editForm.platform === "openai"
          ? messagesDispatchFormStateToConfig(editForm)
          : undefined,
    });
    const updated = await adminAPI.groups.update(
      editingGroup.value.id,
      updatePayload,
    );
    editingGroup.value = updated;
    await captureEditorBaseline("edit");
    const cachedIndex = groups.value.findIndex((group) => group.id === updated.id);
    if (cachedIndex >= 0) {
      groups.value[cachedIndex] = updated;
    }
    appStore.showSuccess(t("admin.groups.groupUpdated"));
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t("admin.groups.failedToUpdate")));
    console.error("Error updating group:", error);
  } finally {
    submitting.value = false;
  }
};

const handleRateMultipliers = (group: AdminGroup) => {
  rateMultipliersGroup.value = group;
  showRateMultipliersModal.value = true;
};

const handleRPMOverrides = (group: AdminGroup) => {
  rpmOverridesGroup.value = group;
  showRPMOverridesModal.value = true;
};

const handleDelete = (group: AdminGroup) => {
  deletingGroup.value = group;
  showDeleteDialog.value = true;
};

const confirmDelete = async () => {
  if (!deletingGroup.value) return;

  try {
    await adminAPI.groups.delete(deletingGroup.value.id);
    appStore.showSuccess(t("admin.groups.groupDeleted"));
    showDeleteDialog.value = false;
    deletingGroup.value = null;
    void refreshGroupsDashboard();
  } catch (error: any) {
    appStore.showError(
      error.response?.data?.detail || t("admin.groups.failedToDelete"),
    );
    console.error("Error deleting group:", error);
  }
};

// 监听 subscription_type 变化，订阅模式时 is_exclusive 默认为 true；标准模式清空高峰配置
watch(
  () => createForm.subscription_type,
  (newVal) => {
    if (newVal === "subscription") {
      createForm.is_exclusive = true;
      createForm.fallback_group_id_on_invalid_request = null;
    } else {
      createForm.peak_rate_enabled = false;
      createForm.peak_start = "";
      createForm.peak_end = "";
      createForm.peak_rate_multiplier = 1.0;
    }
  },
);

// 编辑表单：切回标准模式时清空高峰配置，避免残留随更新请求提交被后端拒绝
watch(
  () => editForm.subscription_type,
  (newVal) => {
    if (newVal !== "subscription") {
      editForm.peak_rate_enabled = false;
      editForm.peak_start = "";
      editForm.peak_end = "";
      editForm.peak_rate_multiplier = 1.0;
    }
  },
);

watch(
  () => createForm.platform,
  (newVal) => {
    if (!["anthropic", "antigravity"].includes(newVal)) {
      createForm.fallback_group_id_on_invalid_request = null;
    }
    if (newVal !== "openai") {
      resetMessagesDispatchFormState(createForm);
    }
    if (!["openai", "antigravity", "anthropic", "gemini"].includes(newVal)) {
      createForm.require_oauth_only = false;
      createForm.require_privacy_set = false;
    }
    resetDisabledBatchImagePricing(createForm);
    if (!editorLoading.value) {
      resetModelsListState(createModelsListState);
      void loadModelsListCandidates("create", 0, newVal);
    }
  },
);

watch(
  () => createForm.allow_image_generation,
  () => {
    resetDisabledBatchImagePricing(createForm);
  },
);

watch(
  () => createForm.allow_batch_image_generation,
  () => {
    resetDisabledBatchImagePricing(createForm);
  },
);

watch(
  () => editForm.platform,
  (newVal) => {
    if (!["anthropic", "antigravity"].includes(newVal)) {
      editForm.fallback_group_id_on_invalid_request = null;
    }
    if (newVal !== "openai") {
      resetMessagesDispatchFormState(editForm);
    }
    if (!["openai", "antigravity", "anthropic", "gemini"].includes(newVal)) {
      editForm.require_oauth_only = false;
      editForm.require_privacy_set = false;
    }
    resetDisabledBatchImagePricing(editForm);
    if (editingGroup.value && !editorLoading.value) {
      resetModelsListState(editModelsListState, editForm.platform === editingGroup.value.platform ? editingGroup.value.models_list_config : undefined);
      void loadModelsListCandidates("edit", editingGroup.value.id, newVal);
    }
  },
);

watch(
  () => editForm.allow_image_generation,
  () => {
    resetDisabledBatchImagePricing(editForm);
  },
);

watch(
  () => editForm.allow_batch_image_generation,
  () => {
    resetDisabledBatchImagePricing(editForm);
  },
);

// 点击外部关闭账号搜索下拉框
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement;
  // 检查是否点击在下拉框或输入框内
  if (!target.closest(".account-search-container")) {
    Object.keys(showAccountDropdown.value).forEach((key) => {
      showAccountDropdown.value[key] = false;
    });
  }
};

// 打开排序弹窗
const openSortModal = async () => {
  try {
    // 获取所有分组（不分页）
    const allGroups = await adminAPI.groups.getAll();
    // 按 sort_order 排序
    sortableGroups.value = [...allGroups].sort(
      (a, b) => a.sort_order - b.sort_order,
    );
    showSortModal.value = true;
  } catch (error) {
    appStore.showError(t("admin.groups.failedToLoad"));
    console.error("Error loading groups for sorting:", error);
  }
};

// 关闭排序弹窗
const closeSortModal = () => {
  showSortModal.value = false;
  sortableGroups.value = [];
};

// 保存排序
const saveSortOrder = async () => {
  sortSubmitting.value = true;
  try {
    const updates = sortableGroups.value.map((g, index) => ({
      id: g.id,
      sort_order: index * 10,
    }));
    await adminAPI.groups.updateSortOrder(updates);
    appStore.showSuccess(t("admin.groups.sortOrderUpdated"));
    closeSortModal();
    void refreshGroupsDashboard();
  } catch (error: any) {
    appStore.showError(
      error.response?.data?.detail || t("admin.groups.failedToUpdateSortOrder"),
    );
    console.error("Error updating sort order:", error);
  } finally {
    sortSubmitting.value = false;
  }
};

onMounted(() => {
  window.addEventListener("beforeunload", handleEditorBeforeUnload);
  document.addEventListener("click", handleClickOutside);
});

onUnmounted(() => {
  window.removeEventListener("beforeunload", handleEditorBeforeUnload);
  document.removeEventListener("click", handleClickOutside);
  accountSearchRunner.clearAll();
  clearAllAccountSearchState();
});
</script>
