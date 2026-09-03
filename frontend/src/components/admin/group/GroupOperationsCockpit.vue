<template>
  <section
    class="groups-cockpit"
    data-admin-page-kind="overview"
    data-test="groups-operations-cockpit"
  >
    <header class="groups-cockpit__header">
      <div class="min-w-0">
        <h1 class="groups-cockpit__title">{{ t("admin.groups.title") }}</h1>
        <p class="groups-cockpit__description">
          {{ t("admin.groups.description") }}
        </p>
      </div>

      <div class="groups-cockpit__header-actions">
        <button
          type="button"
          class="groups-cockpit__icon-button min-h-11 min-w-11"
          data-test="groups-refresh"
          :disabled="refreshing"
          :title="t('common.refresh')"
          :aria-label="t('common.refresh')"
          @click="emit('refresh')"
        >
          <Icon
            name="refresh"
            size="md"
            :class="refreshing ? 'animate-spin' : ''"
          />
        </button>
        <button
          type="button"
          class="groups-cockpit__primary min-h-11"
          data-test="groups-create"
          data-tour="groups-create-btn"
          @click="emit('create')"
        >
          <Icon name="plus" size="md" />
          <span>{{ t("admin.groups.createGroup") }}</span>
        </button>
      </div>
    </header>

    <section
      class="groups-cockpit__metrics"
      :aria-label="t('admin.groups.cockpit.metricsLabel')"
    >
      <article class="groups-cockpit__metric">
        <span class="groups-cockpit__metric-label">
          {{ t("admin.groups.cockpit.totalGroups") }}
        </span>
        <div class="groups-cockpit__metric-value-row">
          <span v-if="overviewLoading" class="groups-cockpit__metric-skeleton" />
          <span v-else-if="overviewError" class="groups-cockpit__metric-value">—</span>
          <span v-else class="groups-cockpit__metric-value">{{ totalGroupCount }}</span>
          <span class="groups-cockpit__metric-hint">
            {{ t("admin.groups.cockpit.totalGroupsHint") }}
          </span>
        </div>
      </article>

      <article class="groups-cockpit__metric">
        <span class="groups-cockpit__metric-label">
          {{ t("admin.groups.cockpit.unavailableGroups") }}
        </span>
        <div class="groups-cockpit__metric-value-row">
          <span v-if="overviewLoading" class="groups-cockpit__metric-skeleton" />
          <span v-else-if="overviewError" class="groups-cockpit__metric-value">—</span>
          <span
            v-else
            class="groups-cockpit__metric-value groups-cockpit__metric-value--danger"
          >
            {{ unavailableGroupCount }}
          </span>
          <span class="groups-cockpit__metric-hint">
            {{
              unavailableGroupCount > 0
                ? t("admin.groups.cockpit.unavailableGroupsHint")
                : t("admin.groups.cockpit.allSchedulable")
            }}
          </span>
        </div>
      </article>

      <article class="groups-cockpit__metric">
        <span class="groups-cockpit__metric-label">
          {{ t("admin.groups.cockpit.capacityWarnings") }}
        </span>
        <div class="groups-cockpit__metric-value-row">
          <span v-if="capacityLoading" class="groups-cockpit__metric-skeleton" />
          <span v-else-if="capacityError" class="groups-cockpit__metric-value">—</span>
          <span
            v-else
            class="groups-cockpit__metric-value"
            :class="capacityWarningCount > 0 && 'groups-cockpit__metric-value--warning'"
          >
            {{ capacityWarningCount }}
          </span>
          <span class="groups-cockpit__metric-hint">
            {{ t("admin.groups.cockpit.capacityWarningsHint") }}
          </span>
        </div>
      </article>

      <article class="groups-cockpit__metric">
        <span class="groups-cockpit__metric-label">
          {{ t("admin.groups.cockpit.todayUsage") }}
        </span>
        <div class="groups-cockpit__metric-value-row">
          <span v-if="usageLoading" class="groups-cockpit__metric-skeleton" />
          <span v-else-if="usageError" class="groups-cockpit__metric-value">—</span>
          <span v-else class="groups-cockpit__metric-value groups-cockpit__metric-value--accent">
            {{ formatCompactCredit(todayUsageTotal) }}
          </span>
          <span class="groups-cockpit__metric-hint groups-cockpit__metric-hint--trend">
            <Icon name="trendingUp" size="xs" :stroke-width="2" />
            {{ t("admin.groups.cockpit.todayUsageHint") }}
          </span>
        </div>
      </article>
    </section>

    <section class="groups-cockpit__toolbar" :aria-label="t('common.filter')">
      <div class="groups-cockpit__search">
        <Icon
          name="search"
          size="md"
          class="groups-cockpit__search-icon"
        />
        <input
          :value="searchQuery"
          type="text"
          :placeholder="t('admin.groups.cockpit.searchPlaceholder')"
          class="groups-cockpit__search-input min-h-11"
          data-test="groups-search"
          @input="handleSearchInput"
        />
      </div>

      <button
        type="button"
        class="groups-cockpit__filter-toggle min-h-11 min-w-11"
        data-test="groups-mobile-filter-toggle"
        :aria-expanded="mobileFiltersExpanded"
        aria-controls="groups-mobile-secondary-filters"
        @click="mobileFiltersExpanded = !mobileFiltersExpanded"
      >
        <Icon name="filter" size="sm" />
        <span>{{ t("common.filter") }}</span>
        <Icon
          name="chevronDown"
          size="xs"
          class="transition-transform motion-reduce:transition-none"
          :class="mobileFiltersExpanded && 'rotate-180'"
        />
      </button>

      <div
        id="groups-mobile-secondary-filters"
        data-test="groups-mobile-secondary-filters"
        class="groups-cockpit__filters"
        :class="mobileFiltersExpanded && 'groups-cockpit__filters--expanded'"
      >
        <Select
          :model-value="filters.platform"
          :options="platformOptions"
          class="groups-cockpit__filter-select"
          @update:model-value="emit('filter', 'platform', String($event ?? ''))"
        />
        <Select
          :model-value="filters.status"
          :options="statusOptions"
          class="groups-cockpit__filter-select"
          @update:model-value="emit('filter', 'status', String($event ?? ''))"
        />
        <Select
          :model-value="filters.is_exclusive"
          :options="exclusiveOptions"
          class="groups-cockpit__filter-select"
          @update:model-value="emit('filter', 'is_exclusive', String($event ?? ''))"
        />
      </div>

      <div class="groups-cockpit__toolbar-tail">
        <span v-if="lastUpdatedAt" class="groups-cockpit__updated-at">
          {{ t("admin.groups.cockpit.lastUpdated", { time: formattedLastUpdated }) }}
        </span>

        <div class="groups-cockpit__preferences">
          <button
            type="button"
            class="groups-cockpit__preferences-trigger min-h-11"
            data-test="groups-preferences-toggle"
            :title="t('admin.groups.columnSettings')"
            :aria-expanded="preferencesOpen"
            aria-controls="groups-preferences-menu"
            @click="preferencesOpen = !preferencesOpen"
          >
            <Icon name="slidersHorizontal" size="md" />
            <span>{{ t("admin.groups.cockpit.displayPreferences") }}</span>
          </button>

          <div
            v-if="preferencesOpen"
            id="groups-preferences-menu"
            class="groups-cockpit__preferences-menu"
            data-test="groups-preferences-menu"
          >
            <p class="groups-cockpit__preferences-heading">
              {{ t("admin.groups.columnSettings") }}
            </p>
            <button
              v-for="preference in columnPreferences"
              :key="preference.key"
              type="button"
              class="groups-cockpit__preference-item"
              @click="emit('toggle-column', preference.key)"
            >
              <span>{{ preference.label }}</span>
              <Icon
                v-if="preference.visible"
                name="check"
                size="sm"
                class="text-primary-500"
                :stroke-width="2"
              />
            </button>
            <div class="groups-cockpit__preferences-divider" />
            <button
              type="button"
              class="groups-cockpit__preference-item groups-cockpit__preference-item--sort"
              data-test="groups-sort-order"
              @click="preferencesOpen = false; emit('sort-order')"
            >
              <span class="flex items-center gap-2">
                <Icon name="arrowsUpDown" size="sm" />
                {{ t("admin.groups.sortOrder") }}
              </span>
              <Icon name="chevronRight" size="xs" />
            </button>
          </div>
        </div>
      </div>
    </section>

    <section
      class="groups-cockpit__ledger"
      data-test="groups-data-table"
      :data-visible-preferences="visiblePreferenceKeys.join(',')"
      :aria-busy="loading"
    >
      <div v-if="loading" class="groups-cockpit__skeleton-list" aria-hidden="true">
        <div v-for="index in 5" :key="index" class="groups-cockpit__skeleton-row">
          <span class="groups-cockpit__skeleton-block groups-cockpit__skeleton-block--wide" />
          <span class="groups-cockpit__skeleton-block" />
          <span class="groups-cockpit__skeleton-block" />
          <span class="groups-cockpit__skeleton-block groups-cockpit__skeleton-block--wide" />
          <span class="groups-cockpit__skeleton-block" />
        </div>
      </div>

      <EmptyState
        v-else-if="groups.length === 0"
        :title="t('admin.groups.noGroupsYet')"
        :description="t('admin.groups.createFirstGroup')"
      />

      <div v-else class="groups-cockpit__ledger-table" role="table">
        <div
          class="groups-cockpit__ledger-header"
          :style="{ gridTemplateColumns: ledgerGridTemplate }"
          role="row"
        >
          <span role="columnheader">{{ t("admin.groups.cockpit.identity") }}</span>
          <span v-if="showPolicyColumn" role="columnheader">
            {{ t("admin.groups.cockpit.policy") }}
          </span>
          <span v-if="showAccountColumn" role="columnheader">
            {{ t("admin.groups.cockpit.accountHealth") }}
          </span>
          <span v-if="showLoadColumn" role="columnheader">
            {{ t("admin.groups.cockpit.capacityAndUsage") }}
          </span>
          <span class="text-right" role="columnheader">
            {{ t("admin.groups.columns.actions") }}
          </span>
        </div>

        <article
          v-for="group in groups"
          :key="group.id"
          class="groups-cockpit__ledger-row"
          :class="expandedGroupId === group.id && 'groups-cockpit__ledger-row--expanded'"
          :style="{ gridTemplateColumns: ledgerGridTemplate }"
          data-test="group-row-actions"
          role="row"
          tabindex="0"
          :aria-label="t('admin.groups.cockpit.rowDisclosure', { name: group.name })"
          :aria-expanded="expandedGroupId === group.id"
          :aria-controls="`groups-row-details-${group.id}`"
          @click="toggleGroupDetails(group.id)"
          @keydown.enter.self.prevent="toggleGroupDetails(group.id)"
          @keydown.space.self.prevent="toggleGroupDetails(group.id)"
        >
          <div
            class="groups-cockpit__cell groups-cockpit__identity"
            :data-label="t('admin.groups.cockpit.identity')"
            role="cell"
          >
            <div class="groups-cockpit__identity-heading">
              <span class="groups-cockpit__group-name">{{ group.name }}</span>
              <span
                v-if="isColumnVisible('status')"
                class="groups-cockpit__status"
                :class="group.status === 'active' ? 'is-active' : 'is-inactive'"
              >
                {{ t(`admin.accounts.status.${group.status}`) }}
              </span>
            </div>
            <div class="groups-cockpit__identity-meta">
              <span v-if="isColumnVisible('platform')" class="inline-flex items-center gap-1.5">
                <PlatformIcon :platform="group.platform" size="xs" />
                {{ t(`admin.groups.platforms.${group.platform}`) }}
              </span>
              <span v-if="isColumnVisible('id')">#{{ group.id }}</span>
            </div>
          </div>

          <div
            v-if="showPolicyColumn"
            class="groups-cockpit__cell groups-cockpit__policy"
            :data-label="t('admin.groups.cockpit.policy')"
            role="cell"
          >
            <span
              v-if="isColumnVisible('billing_type')"
              class="groups-cockpit__pill"
              :class="group.subscription_type === 'subscription' ? 'is-subscription' : 'is-neutral'"
            >
              {{
                group.subscription_type === "subscription"
                  ? t("admin.groups.subscription.subscription")
                  : t("admin.groups.subscription.standard")
              }}
            </span>
            <span
              v-if="isColumnVisible('is_exclusive')"
              class="groups-cockpit__pill is-neutral"
            >
              {{ group.is_exclusive ? t("admin.groups.exclusive") : t("admin.groups.public") }}
            </span>
            <span
              v-if="isColumnVisible('rate_multiplier')"
              class="groups-cockpit__rate"
            >
              {{ group.rate_multiplier }}x
            </span>
          </div>

          <div
            v-if="showAccountColumn"
            class="groups-cockpit__cell groups-cockpit__health"
            :data-label="t('admin.groups.cockpit.accountHealth')"
            role="cell"
          >
            <template v-if="getAccountHealth(group).available">
              <strong>
                {{
                  t("admin.groups.cockpit.accountAvailability", {
                    available: getAccountHealth(group).active,
                    total: getAccountHealth(group).total,
                  })
                }}
              </strong>
              <span
                :class="getAccountHealth(group).limited > 0 ? 'is-warning' : 'is-ready'"
              >
                {{
                  getAccountHealth(group).limited > 0
                    ? t("admin.groups.cockpit.rateLimitedAccounts", {
                        count: getAccountHealth(group).limited,
                      })
                    : t("admin.groups.cockpit.allReady")
                }}
              </span>
            </template>
            <span v-else class="is-muted">{{ t("admin.groups.cockpit.dataUnavailable") }}</span>
          </div>

          <div
            v-if="showLoadColumn"
            class="groups-cockpit__cell groups-cockpit__load"
            :data-label="t('admin.groups.cockpit.capacityAndUsage')"
            role="cell"
          >
            <div v-if="isColumnVisible('capacity')" class="groups-cockpit__load-summary">
              <div class="groups-cockpit__load-labels">
                <span :class="capacityToneClass(getCapacityState(group))">
                  {{ capacitySummaryLabel(group) }}
                </span>
                <CreditAmount
                  v-if="isColumnVisible('usage') && getUsage(group)"
                  :value="formatCost(getUsage(group)!.today_cost)"
                  icon-size="xs"
                  :label="t('admin.groups.cockpit.todayUsageAccessible', { value: formatCost(getUsage(group)!.today_cost) })"
                />
                <span
                  v-else-if="isColumnVisible('usage')"
                  class="is-muted"
                >—</span>
              </div>
              <div class="groups-cockpit__progress" aria-hidden="true">
                <span
                  class="groups-cockpit__progress-fill"
                  :class="capacityProgressClass(getCapacityState(group))"
                  :style="{ width: capacityProgressWidth(getCapacityState(group)) }"
                />
              </div>
            </div>
            <div v-else-if="isColumnVisible('usage')" class="groups-cockpit__usage-only">
              <span>{{ t("admin.groups.usageToday") }}</span>
              <CreditAmount
                v-if="getUsage(group)"
                :value="formatCost(getUsage(group)!.today_cost)"
                icon-size="xs"
              />
              <span v-else>—</span>
            </div>
          </div>

          <div
            class="groups-cockpit__cell groups-cockpit__row-actions"
            :data-label="t('admin.groups.columns.actions')"
            role="cell"
            @click.stop
            @keydown.stop
          >
            <button
              type="button"
              class="groups-cockpit__edit-button min-h-11 min-w-11 lg:min-h-8 lg:min-w-0"
              data-test="groups-row-edit"
              @click.stop="emit('edit', group)"
            >
              {{ t("common.edit") }}
            </button>

            <button
              type="button"
              class="groups-cockpit__row-icon-button min-h-11 min-w-11 lg:min-h-8 lg:min-w-8"
              data-test="groups-row-details-toggle"
              :title="
                expandedGroupId === group.id
                  ? t('admin.groups.cockpit.collapseDetails')
                  : t('admin.groups.cockpit.viewDetails')
              "
              :aria-label="
                expandedGroupId === group.id
                  ? t('admin.groups.cockpit.collapseDetails')
                  : t('admin.groups.cockpit.viewDetails')
              "
              :aria-expanded="expandedGroupId === group.id"
              :aria-controls="`groups-row-details-${group.id}`"
              @click.stop="toggleGroupDetails(group.id)"
            >
              <Icon
                :name="expandedGroupId === group.id ? 'chevronUp' : 'chevronDown'"
                size="sm"
              />
            </button>

            <details class="groups-cockpit__row-menu" @click.stop>
              <summary
                class="groups-cockpit__row-icon-button min-h-11 min-w-11 lg:min-h-8 lg:min-w-8"
                data-test="groups-row-more"
                :aria-label="t('common.more')"
              >
                <Icon name="more" size="sm" />
              </summary>
              <div class="groups-cockpit__row-menu-popover">
                <button
                  type="button"
                  data-test="groups-row-rate-desktop"
                  @click="emit('rate-multipliers', group)"
                >
                  <Icon name="dollar" size="sm" />
                  {{ t("admin.groups.rateMultipliers") }}
                </button>
                <button
                  type="button"
                  data-test="groups-row-rpm-desktop"
                  @click="emit('rpm-overrides', group)"
                >
                  <Icon name="bolt" size="sm" />
                  {{ t("admin.groups.rpmOverrides") }}
                </button>
                <button
                  type="button"
                  class="is-danger"
                  data-test="groups-row-delete-desktop"
                  @click="emit('delete', group)"
                >
                  <Icon name="trash" size="sm" />
                  {{ t("common.delete") }}
                </button>
              </div>
            </details>
          </div>

          <section
            v-if="expandedGroupId === group.id"
            :id="`groups-row-details-${group.id}`"
            class="groups-cockpit__detail"
            role="region"
            :aria-label="t('admin.groups.cockpit.groupDetails', { name: group.name })"
            @click.stop
            @keydown.stop
          >
            <div class="groups-cockpit__detail-section">
              <h2>{{ t("admin.groups.cockpit.quotaControl") }}</h2>
              <div class="groups-cockpit__quota-cards">
                <div class="groups-cockpit__detail-card">
                  <span>{{ t("admin.groups.cockpit.dailyLimit") }}</span>
                  <CreditAmount
                    v-if="group.daily_limit_usd !== null && group.daily_limit_usd !== undefined"
                    :value="formatNullableCost(group.daily_limit_usd)"
                    icon-size="xs"
                  />
                  <strong v-else>—</strong>
                </div>
                <div class="groups-cockpit__detail-card">
                  <span>{{ t("admin.groups.usageTotal") }}</span>
                  <CreditAmount
                    v-if="getUsage(group)"
                    :value="formatCost(getUsage(group)!.total_cost)"
                    icon-size="xs"
                  />
                  <strong v-else>—</strong>
                </div>
              </div>
              <div class="groups-cockpit__quota-meta">
                <span>
                  {{ t("admin.groups.cockpit.weeklyLimit") }}
                  <CreditAmount
                    v-if="group.weekly_limit_usd !== null && group.weekly_limit_usd !== undefined"
                    :value="formatNullableCost(group.weekly_limit_usd)"
                    icon-size="xs"
                  />
                  <strong v-else>—</strong>
                </span>
                <span>
                  {{ t("admin.groups.cockpit.monthlyLimit") }}
                  <CreditAmount
                    v-if="group.monthly_limit_usd !== null && group.monthly_limit_usd !== undefined"
                    :value="formatNullableCost(group.monthly_limit_usd)"
                    icon-size="xs"
                  />
                  <strong v-else>—</strong>
                </span>
              </div>
              <p v-if="group.subscription_type === 'subscription'" class="groups-cockpit__detail-note">
                {{ t("admin.groups.cockpit.perSubscriberLimitNote") }}
              </p>
            </div>

            <div class="groups-cockpit__detail-section groups-cockpit__detail-section--capacity">
              <h2>{{ t("admin.groups.cockpit.capacityDetails") }}</h2>
              <template v-if="capacityMap.get(group.id)">
                <div
                  v-for="dimension in capacityDimensions(capacityMap.get(group.id)!)"
                  :key="dimension.key"
                  class="groups-cockpit__capacity-line"
                >
                  <span>{{ dimension.label }}</span>
                  <strong :class="dimension.ratio !== null && dimension.ratio >= 0.8 && 'is-warning'">
                    {{ dimension.max > 0 ? `${dimension.used} / ${dimension.max}` : t("admin.groups.cockpit.unconfigured") }}
                  </strong>
                </div>
              </template>
              <p v-else class="groups-cockpit__detail-empty">
                {{
                  group.status === "inactive"
                    ? t("admin.groups.cockpit.unmonitored")
                    : t("admin.groups.cockpit.dataUnavailable")
                }}
              </p>
              <div class="groups-cockpit__capacity-line">
                <span>{{ t("admin.groups.cockpit.routingStatus") }}</span>
                <strong :class="group.model_routing_enabled ? 'is-ready' : 'is-muted'">
                  {{
                    group.model_routing_enabled
                      ? t("admin.groups.cockpit.routingRules", { count: routingRuleCount(group) })
                      : t("admin.groups.cockpit.unconfigured")
                  }}
                </strong>
              </div>
            </div>
          </section>
        </article>
      </div>
    </section>

    <div v-if="pagination.total > 0" class="groups-cockpit__pagination">
      <Pagination
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="emit('page-change', $event)"
        @update:pageSize="emit('page-size-change', $event)"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import type { AdminGroup } from "@/types";
import CreditAmount from "@/components/common/CreditAmount.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import Pagination from "@/components/common/Pagination.vue";
import PlatformIcon from "@/components/common/PlatformIcon.vue";
import Select from "@/components/common/Select.vue";
import Icon from "@/components/icons/Icon.vue";

type GroupFilterKey = "platform" | "status" | "is_exclusive";

type GroupUsageSummary = {
  today_cost: number;
  total_cost: number;
};

type GroupCapacitySummary = {
  concurrencyUsed: number;
  concurrencyMax: number;
  sessionsUsed: number;
  sessionsMax: number;
  rpmUsed: number;
  rpmMax: number;
};

type ColumnPreference = {
  key: string;
  label: string;
  visible: boolean;
};

type SelectOption = {
  value: string;
  label: string;
};

type CapacityState = {
  status: "loading" | "unavailable" | "unmonitored" | "unconfigured" | "neutral" | "warning" | "critical";
  ratio: number | null;
  dimension: string;
};

const props = defineProps<{
  groups: AdminGroup[];
  overviewGroups: AdminGroup[];
  loading: boolean;
  overviewLoading: boolean;
  overviewError: boolean;
  usageMap: Map<number, GroupUsageSummary>;
  usageLoading: boolean;
  usageError: boolean;
  capacityMap: Map<number, GroupCapacitySummary>;
  capacityLoading: boolean;
  capacityError: boolean;
  refreshing: boolean;
  searchQuery: string;
  filters: {
    platform: string;
    status: string;
    is_exclusive: string;
  };
  platformOptions: SelectOption[];
  statusOptions: SelectOption[];
  exclusiveOptions: SelectOption[];
  columnPreferences: ColumnPreference[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
  lastUpdatedAt: Date | null;
}>();

const emit = defineEmits<{
  refresh: [];
  create: [];
  search: [value: string];
  filter: [key: GroupFilterKey, value: string];
  "toggle-column": [key: string];
  "sort-order": [];
  edit: [group: AdminGroup];
  "rate-multipliers": [group: AdminGroup];
  "rpm-overrides": [group: AdminGroup];
  delete: [group: AdminGroup];
  "page-change": [page: number];
  "page-size-change": [pageSize: number];
}>();

const { t } = useI18n();
const mobileFiltersExpanded = ref(false);
const preferencesOpen = ref(false);
const expandedGroupId = ref<number | null>(null);
const autoExpansionInitialized = ref(false);

const visiblePreferenceKeys = computed(() =>
  props.columnPreferences
    .filter((preference) => preference.visible)
    .map((preference) => preference.key),
);

const isColumnVisible = (key: string) =>
  props.columnPreferences.find((preference) => preference.key === key)?.visible ??
  ["name", "actions"].includes(key);

const showPolicyColumn = computed(() =>
  ["billing_type", "rate_multiplier", "is_exclusive"].some(isColumnVisible),
);
const showAccountColumn = computed(() => isColumnVisible("account_count"));
const showLoadColumn = computed(() =>
  ["capacity", "usage"].some(isColumnVisible),
);

const ledgerGridTemplate = computed(() => {
  const tracks = ["minmax(220px,1.05fr)"];
  if (showPolicyColumn.value) tracks.push("minmax(210px,.95fr)");
  if (showAccountColumn.value) tracks.push("minmax(180px,.78fr)");
  if (showLoadColumn.value) tracks.push("minmax(260px,1fr)");
  tracks.push("132px");
  return tracks.join(" ");
});

const totalGroupCount = computed(() => props.overviewGroups.length);
const unavailableGroupCount = computed(() =>
  props.overviewGroups.filter((group) => {
    if (group.status !== "active") return false;
    if (typeof group.account_count !== "number" || group.account_count <= 0) {
      return false;
    }
    if (typeof group.active_account_count !== "number") return false;
    return group.active_account_count === 0;
  }).length,
);

const capacityWarningCount = computed(() => {
  let count = 0;
  for (const capacity of props.capacityMap.values()) {
    const ratios = capacityDimensions(capacity)
      .map((dimension) => dimension.ratio)
      .filter((ratio): ratio is number => ratio !== null);
    if (ratios.length > 0 && Math.max(...ratios) >= 0.8) count += 1;
  }
  return count;
});

const todayUsageTotal = computed(() => {
  let total = 0;
  for (const usage of props.usageMap.values()) {
    if (Number.isFinite(usage.today_cost)) total += usage.today_cost;
  }
  return total;
});

const formattedLastUpdated = computed(() => {
  if (!props.lastUpdatedAt) return "";
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(props.lastUpdatedAt);
});

watch(
  () => props.groups.map((group) => group.id).join(","),
  () => {
    if (
      expandedGroupId.value !== null &&
      !props.groups.some((group) => group.id === expandedGroupId.value)
    ) {
      expandedGroupId.value = null;
    }
    if (!autoExpansionInitialized.value && props.groups.length > 0) {
      expandedGroupId.value =
        props.groups.find((group) => group.subscription_type === "subscription")?.id ??
        null;
      autoExpansionInitialized.value = true;
    }
  },
  { immediate: true },
);

const toggleGroupDetails = (groupId: number) => {
  expandedGroupId.value = expandedGroupId.value === groupId ? null : groupId;
};

const handleSearchInput = (event: Event) => {
  emit("search", (event.target as HTMLInputElement).value);
};

const formatCost = (cost: number | null | undefined): string => {
  if (cost === null || cost === undefined || !Number.isFinite(cost)) return "—";
  if (cost >= 1000) return cost.toFixed(0);
  if (cost >= 100) return cost.toFixed(1);
  return cost.toFixed(2);
};

const formatNullableCost = (cost: number | null | undefined) =>
  cost === null || cost === undefined ? "—" : formatCost(cost);

const formatCompactCredit = (value: number) => {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}m`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}k`;
  return formatCost(value);
};

const getUsage = (group: AdminGroup) => props.usageMap.get(group.id);

const getAccountHealth = (group: AdminGroup) => {
  const fieldsPresent = [
    group.account_count,
    group.active_account_count,
    group.rate_limited_account_count,
  ].some((value) => typeof value === "number");

  return {
    available: fieldsPresent,
    total: group.account_count ?? 0,
    active: group.active_account_count ?? 0,
    limited: group.rate_limited_account_count ?? 0,
  };
};

const capacityDimensions = (capacity: GroupCapacitySummary) => [
  {
    key: "concurrency",
    label: t("admin.groups.cockpit.concurrency"),
    used: capacity.concurrencyUsed,
    max: capacity.concurrencyMax,
    ratio:
      capacity.concurrencyMax > 0
        ? Math.max(0, capacity.concurrencyUsed) / capacity.concurrencyMax
        : null,
  },
  {
    key: "sessions",
    label: t("admin.groups.cockpit.sessions"),
    used: capacity.sessionsUsed,
    max: capacity.sessionsMax,
    ratio:
      capacity.sessionsMax > 0
        ? Math.max(0, capacity.sessionsUsed) / capacity.sessionsMax
        : null,
  },
  {
    key: "rpm",
    label: "RPM",
    used: capacity.rpmUsed,
    max: capacity.rpmMax,
    ratio:
      capacity.rpmMax > 0
        ? Math.max(0, capacity.rpmUsed) / capacity.rpmMax
        : null,
  },
];

const getCapacityState = (group: AdminGroup): CapacityState => {
  if (group.status === "inactive") {
    return { status: "unmonitored", ratio: null, dimension: "" };
  }
  const capacity = props.capacityMap.get(group.id);
  if (!capacity) {
    if (props.capacityLoading) {
      return { status: "loading", ratio: null, dimension: "" };
    }
    return { status: "unavailable", ratio: null, dimension: "" };
  }
  const configured = capacityDimensions(capacity).filter(
    (dimension) => dimension.ratio !== null,
  );
  if (configured.length === 0) {
    return { status: "unconfigured", ratio: null, dimension: "" };
  }
  const riskiest = [...configured].sort(
    (left, right) => (right.ratio ?? 0) - (left.ratio ?? 0),
  )[0]!;
  const ratio = riskiest.ratio ?? 0;
  return {
    status: ratio >= 1 ? "critical" : ratio >= 0.8 ? "warning" : "neutral",
    ratio,
    dimension: riskiest.label,
  };
};

const capacitySummaryLabel = (group: AdminGroup) => {
  const state = getCapacityState(group);
  if (state.status === "loading") return t("common.loading");
  if (state.status === "unmonitored") return t("admin.groups.cockpit.unmonitored");
  if (state.status === "unconfigured") return t("admin.groups.cockpit.unconfigured");
  if (state.status === "unavailable") return t("admin.groups.cockpit.dataUnavailable");
  return t("admin.groups.cockpit.capacityLoad", {
    dimension: state.dimension,
    percent: Math.round((state.ratio ?? 0) * 100),
  });
};

const capacityToneClass = (state: CapacityState) => ({
  "is-warning": state.status === "warning",
  "is-critical": state.status === "critical",
  "is-muted": ["loading", "unavailable", "unmonitored", "unconfigured"].includes(
    state.status,
  ),
});

const capacityProgressClass = (state: CapacityState) => ({
  "is-warning": state.status === "warning",
  "is-critical": state.status === "critical",
  "is-neutral": state.status === "neutral",
});

const capacityProgressWidth = (state: CapacityState) =>
  state.ratio === null ? "0%" : `${Math.min(100, state.ratio * 100)}%`;

const routingRuleCount = (group: AdminGroup) =>
  group.model_routing ? Object.keys(group.model_routing).length : 0;
</script>

<style scoped>
.groups-cockpit {
  width: 100%;
  min-height: calc(100dvh - var(--app-shell-top-offset));
  padding: 32px;
  container-type: inline-size;
  color: var(--lx-clay-text);
  background: var(--lx-clay-canvas);
  font-family: var(--lx-clay-font-ui);
}

.groups-cockpit__header,
.groups-cockpit__header-actions,
.groups-cockpit__toolbar,
.groups-cockpit__toolbar-tail,
.groups-cockpit__preferences-trigger,
.groups-cockpit__primary,
.groups-cockpit__identity-heading,
.groups-cockpit__identity-meta,
.groups-cockpit__policy,
.groups-cockpit__load-labels,
.groups-cockpit__row-actions,
.groups-cockpit__quota-meta {
  display: flex;
  align-items: center;
}

.groups-cockpit__header {
  justify-content: space-between;
  gap: 20px;
}

.groups-cockpit__title {
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 1.875rem;
  font-weight: 800;
  line-height: 1.2;
  letter-spacing: -0.025em;
}

.groups-cockpit__description {
  margin: 5px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.875rem;
  line-height: 1.45;
}

.groups-cockpit__header-actions {
  flex: 0 0 auto;
  gap: 12px;
}

.groups-cockpit__icon-button,
.groups-cockpit__filter-toggle,
.groups-cockpit__preferences-trigger,
.groups-cockpit__edit-button,
.groups-cockpit__row-icon-button {
  border: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface);
  cursor: pointer;
}

.groups-cockpit__icon-button {
  display: inline-grid;
  place-items: center;
  border-radius: 12px;
}

.groups-cockpit__icon-button:hover,
.groups-cockpit__filter-toggle:hover,
.groups-cockpit__preferences-trigger:hover,
.groups-cockpit__row-icon-button:hover {
  border-color: var(--lx-clay-border-strong);
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.groups-cockpit__primary {
  justify-content: center;
  gap: 8px;
  padding: 0 22px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-on-accent) 12%, transparent);
  border-radius: 12px;
  color: var(--lx-clay-on-accent);
  background: linear-gradient(145deg, var(--lx-clay-accent-gradient-start), var(--lx-clay-accent-deep));
  box-shadow: var(--lx-clay-shadow-primary);
  font-size: 0.875rem;
  font-weight: 800;
  cursor: pointer;
}

.groups-cockpit__metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-top: 24px;
  padding: 20px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 24px;
  background: var(--lx-clay-overlay-surface-soft);
  box-shadow: var(--lx-clay-shadow-flat);
  backdrop-filter: blur(8px);
}

.groups-cockpit__metric {
  min-width: 0;
  padding: 0 24px;
}

.groups-cockpit__metric + .groups-cockpit__metric {
  border-left: 1px solid var(--lx-clay-border);
}

.groups-cockpit__metric-label {
  display: block;
  color: var(--lx-clay-text-subtle);
  font-size: 0.6875rem;
  font-weight: 800;
  letter-spacing: 0.12em;
}

.groups-cockpit__metric-value-row {
  display: flex;
  min-height: 32px;
  align-items: baseline;
  gap: 9px;
  margin-top: 4px;
}

.groups-cockpit__metric-value {
  color: var(--lx-clay-text);
  font-size: 1.5rem;
  font-weight: 900;
  line-height: 1.2;
  letter-spacing: -0.025em;
  tab-size: 4;
}

.groups-cockpit__metric-value--danger {
  color: #ef4444;
}

.groups-cockpit__metric-value--warning {
  color: var(--lx-clay-warning-bright);
}

.groups-cockpit__metric-value--accent {
  color: var(--lx-clay-accent);
}

.groups-cockpit__metric-hint {
  overflow: hidden;
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.groups-cockpit__metric-hint--trend {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  color: var(--lx-clay-success);
  font-weight: 800;
}

.groups-cockpit__metric-skeleton,
.groups-cockpit__skeleton-block {
  border-radius: 8px;
  background: linear-gradient(90deg, var(--lx-clay-recessed), var(--lx-clay-surface), var(--lx-clay-recessed));
  background-size: 200% 100%;
  animation: groups-cockpit-shimmer 1.4s ease-in-out infinite;
}

.groups-cockpit__metric-skeleton {
  width: 72px;
  height: 28px;
}

.groups-cockpit__toolbar {
  position: relative;
  z-index: 8;
  gap: 12px;
  margin-top: 24px;
  padding: 12px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 16px;
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.groups-cockpit__search {
  position: relative;
  width: min(100%, 384px);
  max-width: 384px;
  flex: 1 1 320px;
}

.groups-cockpit__search-icon {
  position: absolute;
  top: 50%;
  left: 12px;
  z-index: 1;
  color: var(--lx-clay-text-subtle);
  transform: translateY(-50%);
  pointer-events: none;
}

.groups-cockpit__search-input {
  width: 100%;
  padding: 0 14px 0 40px;
  border: 1px solid transparent;
  border-radius: 12px;
  outline: none;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface-soft);
  font-size: 0.875rem;
}

.groups-cockpit__search-input:focus {
  border-color: var(--lx-clay-accent);
  background: var(--lx-clay-surface);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--lx-clay-accent) 16%, transparent);
}

.groups-cockpit__filters {
  display: none;
  gap: 8px;
  margin-left: 6px;
  padding-left: 14px;
  border-left: 1px solid var(--lx-clay-border);
}

.groups-cockpit__filters--expanded {
  display: grid;
}

.groups-cockpit__filter-select {
  width: 104px;
}

.groups-cockpit__filter-select:last-child {
  width: 112px;
}

.groups-cockpit__filter-toggle {
  display: flex;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;
  border-radius: 12px;
  font-size: 0.8125rem;
  font-weight: 750;
}

@container (min-width: 1050px) {
  .groups-cockpit__filter-toggle {
    display: none;
  }

  .groups-cockpit__filters,
  .groups-cockpit__filters--expanded {
    display: flex;
  }

  .groups-cockpit__search-input,
  .groups-cockpit__preferences-trigger {
    min-height: 40px;
    height: 40px;
  }

  .groups-cockpit__filter-select :deep(.select-trigger) {
    min-height: 40px;
    height: 40px;
    gap: 4px;
    padding: 0 10px 0 12px;
  }
}

.groups-cockpit__toolbar-tail {
  min-width: 0;
  flex: 0 1 auto;
  gap: 14px;
  margin-left: auto;
}

.groups-cockpit__updated-at {
  overflow: hidden;
  color: var(--lx-clay-text-subtle);
  font-size: 0.6875rem;
  font-style: italic;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.groups-cockpit__preferences {
  position: relative;
}

.groups-cockpit__preferences-trigger {
  justify-content: center;
  gap: 8px;
  padding: 0 14px;
  border-radius: 12px;
  font-size: 0.8125rem;
  font-weight: 800;
  white-space: nowrap;
}

.groups-cockpit__preferences-menu,
.groups-cockpit__row-menu-popover {
  position: absolute;
  right: 0;
  z-index: 30;
  border: 1px solid var(--lx-clay-border);
  border-radius: 14px;
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-overlay);
}

.groups-cockpit__preferences-menu {
  top: calc(100% + 8px);
  width: 236px;
  padding: 8px;
}

.groups-cockpit__preferences-heading {
  margin: 0;
  padding: 8px 10px 6px;
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.groups-cockpit__preference-item {
  display: flex;
  width: 100%;
  min-height: 40px;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 10px;
  border: 0;
  border-radius: 10px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font: inherit;
  font-size: 0.8125rem;
  text-align: left;
  cursor: pointer;
}

.groups-cockpit__preference-item:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.groups-cockpit__preferences-divider {
  height: 1px;
  margin: 6px 4px;
  background: var(--lx-clay-border);
}

.groups-cockpit__preference-item--sort {
  font-weight: 750;
}

.groups-cockpit__ledger {
  position: relative;
  z-index: 1;
  margin-top: 24px;
  overflow: auto;
  border: 1px solid var(--lx-clay-border);
  border-radius: 16px;
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.groups-cockpit__ledger-table {
  min-width: 1020px;
}

.groups-cockpit__ledger-header,
.groups-cockpit__ledger-row,
.groups-cockpit__skeleton-row {
  display: grid;
  min-width: 0;
}

.groups-cockpit__ledger-header {
  padding: 0 24px;
  border-bottom: 1px solid var(--lx-clay-border);
  border-radius: 16px 16px 0 0;
  color: var(--lx-clay-text-subtle);
  background: color-mix(in srgb, var(--lx-clay-surface) 68%, var(--lx-clay-recessed));
  font-size: 0.6875rem;
  font-weight: 850;
  letter-spacing: 0.09em;
}

.groups-cockpit__ledger-header > span {
  padding: 15px 0;
}

.groups-cockpit__ledger-row {
  position: relative;
  padding: 0 24px;
  border-bottom: 1px solid color-mix(in srgb, var(--lx-clay-border) 72%, transparent);
  cursor: pointer;
  transition: background-color 150ms ease;
}

.groups-cockpit__ledger-row:last-child {
  border-bottom: 0;
}

.groups-cockpit__ledger-row:hover,
.groups-cockpit__ledger-row--expanded {
  background: color-mix(in srgb, var(--lx-clay-surface) 92%, var(--lx-clay-accent-soft));
}

.groups-cockpit__ledger-row:focus-visible {
  z-index: 2;
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: -2px;
}

.groups-cockpit__cell {
  min-width: 0;
  padding: 19px 0;
}

.groups-cockpit__identity {
  align-self: center;
}

.groups-cockpit__identity-heading {
  min-width: 0;
  gap: 8px;
}

.groups-cockpit__group-name {
  overflow: hidden;
  color: var(--lx-clay-text);
  font-size: 0.875rem;
  font-weight: 850;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.groups-cockpit__status,
.groups-cockpit__pill {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  border-radius: 999px;
  font-size: 0.6875rem;
  font-weight: 800;
  line-height: 1.3;
}

.groups-cockpit__status {
  padding: 2px 7px;
}

.groups-cockpit__status.is-active {
  color: var(--lx-clay-success-text);
  background: var(--lx-clay-success-soft);
}

.groups-cockpit__status.is-inactive {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.groups-cockpit__identity-meta {
  gap: 7px;
  margin-top: 5px;
  color: var(--lx-clay-text-subtle);
  font-size: 0.6875rem;
}

.groups-cockpit__identity-meta > span + span::before {
  content: "·";
  margin-right: 7px;
}

.groups-cockpit__policy {
  align-content: center;
  align-items: center;
  align-self: center;
  flex-wrap: wrap;
  gap: 6px;
}

.groups-cockpit__pill {
  padding: 3px 8px;
}

.groups-cockpit__pill.is-subscription {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.groups-cockpit__pill.is-neutral {
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
}

.groups-cockpit__rate {
  color: var(--lx-clay-accent);
  font-size: 0.75rem;
  font-weight: 850;
}

.groups-cockpit__health {
  display: flex;
  align-self: center;
  flex-direction: column;
  gap: 4px;
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
}

.groups-cockpit__health strong {
  color: var(--lx-clay-text);
  font-weight: 800;
}

.groups-cockpit__health span,
.groups-cockpit__load-labels,
.groups-cockpit__usage-only {
  font-size: 0.6875rem;
  font-weight: 700;
}

.is-ready {
  color: var(--lx-clay-success-text) !important;
}

.is-warning {
  color: var(--lx-clay-warning) !important;
}

.is-critical,
.is-danger {
  color: var(--lx-clay-danger) !important;
}

.is-muted {
  color: var(--lx-clay-text-subtle) !important;
}

.groups-cockpit__load {
  align-self: center;
  padding-right: 22px;
}

.groups-cockpit__load-summary {
  display: grid;
  gap: 8px;
}

.groups-cockpit__load-labels {
  justify-content: space-between;
  gap: 12px;
  color: var(--lx-clay-text-muted);
}

.groups-cockpit__progress {
  height: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--lx-clay-recessed);
}

.groups-cockpit__progress-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--lx-clay-text-subtle);
  transition: width 180ms ease;
}

.groups-cockpit__progress-fill.is-neutral {
  background: var(--lx-clay-success-bright);
}

.groups-cockpit__progress-fill.is-warning {
  background: var(--lx-clay-warning-bright);
}

.groups-cockpit__progress-fill.is-critical {
  background: var(--lx-clay-danger);
}

.groups-cockpit__usage-only {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  color: var(--lx-clay-text-muted);
}

.groups-cockpit__row-actions {
  position: relative;
  align-self: center;
  justify-content: flex-end;
  gap: 5px;
  cursor: default;
}

.groups-cockpit__edit-button {
  padding: 0 14px;
  border-color: color-mix(in srgb, var(--lx-clay-accent) 22%, var(--lx-clay-border));
  border-radius: 9px;
  color: var(--lx-clay-accent);
  font-size: 0.75rem;
  font-weight: 800;
}

.groups-cockpit__edit-button:hover {
  border-color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.groups-cockpit__row-icon-button {
  display: inline-grid;
  flex: 0 0 auto;
  place-items: center;
  padding: 0;
  border-color: transparent;
  border-radius: 8px;
  background: transparent;
  list-style: none;
}

.groups-cockpit__row-icon-button::-webkit-details-marker {
  display: none;
}

.groups-cockpit__row-menu {
  position: relative;
}

.groups-cockpit__row-menu-popover {
  top: calc(100% + 6px);
  width: 190px;
  padding: 7px;
}

.groups-cockpit__ledger-row:last-child .groups-cockpit__row-menu-popover {
  top: auto;
  bottom: calc(100% + 6px);
}

.groups-cockpit__row-menu-popover button {
  display: flex;
  width: 100%;
  min-height: 38px;
  align-items: center;
  gap: 9px;
  padding: 8px 10px;
  border: 0;
  border-radius: 9px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font: inherit;
  font-size: 0.75rem;
  text-align: left;
  cursor: pointer;
}

.groups-cockpit__row-menu-popover button:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.groups-cockpit__row-menu-popover button.is-danger:hover {
  background: var(--lx-clay-danger-soft);
}

.groups-cockpit__detail {
  display: grid;
  grid-column: 1 / -1;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0;
  margin: 0 16px 18px;
  padding: 22px 24px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-accent) 16%, var(--lx-clay-border));
  border-radius: 16px;
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.groups-cockpit__detail-section {
  min-width: 0;
  padding: 0 24px;
}

.groups-cockpit__detail-section:first-child {
  padding-left: 0;
}

.groups-cockpit__detail-section:last-child {
  padding-right: 0;
}

.groups-cockpit__detail-section + .groups-cockpit__detail-section {
  border-left: 1px solid var(--lx-clay-border);
}

.groups-cockpit__detail-section h2 {
  margin: 0 0 15px;
  color: var(--lx-clay-accent);
  font-family: var(--lx-clay-font-ui);
  font-size: 0.6875rem;
  font-weight: 900;
  letter-spacing: 0.1em;
}

.groups-cockpit__quota-cards {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.groups-cockpit__detail-card {
  min-width: 0;
  padding: 12px;
  border-radius: 12px;
  background: var(--lx-clay-surface-soft);
}

.groups-cockpit__detail-card > span {
  display: block;
  margin-bottom: 6px;
  color: var(--lx-clay-text-subtle);
  font-size: 0.625rem;
  font-weight: 800;
}

.groups-cockpit__detail-card :deep([data-testid="credit-amount"]),
.groups-cockpit__detail-card > strong {
  color: var(--lx-clay-text);
  font-size: 0.8125rem;
  font-weight: 900;
}

.groups-cockpit__quota-meta {
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 12px;
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 650;
}

.groups-cockpit__quota-meta > span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.groups-cockpit__detail-note,
.groups-cockpit__detail-empty {
  margin: 12px 0 0;
  color: var(--lx-clay-text-subtle);
  font-size: 0.6875rem;
  line-height: 1.5;
}

.groups-cockpit__capacity-line {
  display: flex;
  min-height: 29px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
}

.groups-cockpit__capacity-line strong {
  color: var(--lx-clay-text);
  font-weight: 800;
  white-space: nowrap;
}

.groups-cockpit__pagination {
  margin-top: 16px;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: 16px;
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.groups-cockpit__pagination :deep(.pagination) {
  border-top: 0;
  background: transparent !important;
}

.groups-cockpit__skeleton-list {
  padding: 0 24px;
}

.groups-cockpit__skeleton-row {
  grid-template-columns: 1.05fr .95fr .78fr 1fr 104px;
  gap: 24px;
  padding: 22px 0;
  border-bottom: 1px solid var(--lx-clay-border);
}

.groups-cockpit__skeleton-block {
  height: 30px;
}

.groups-cockpit__skeleton-block--wide {
  height: 40px;
}

@keyframes groups-cockpit-shimmer {
  0% { background-position: 100% 0; }
  100% { background-position: -100% 0; }
}

@media (max-width: 1279px) {
  .groups-cockpit {
    padding: 24px;
  }
}

@container (max-width: 760px) {
  .groups-cockpit__detail {
    grid-template-columns: minmax(0, 1fr);
    gap: 18px;
  }

  .groups-cockpit__detail-section,
  .groups-cockpit__detail-section:first-child,
  .groups-cockpit__detail-section:last-child {
    padding: 0;
  }

  .groups-cockpit__detail-section + .groups-cockpit__detail-section {
    padding-top: 18px;
    border-top: 1px solid var(--lx-clay-border);
    border-left: 0;
  }
}

@container (max-width: 1049px) {
  .groups-cockpit__toolbar {
    align-items: stretch;
    flex-wrap: wrap;
  }

  .groups-cockpit__toolbar-tail {
    flex: 1 1 auto;
  }

  .groups-cockpit__updated-at {
    display: none;
  }

  .groups-cockpit__filters {
    order: 4;
    width: 100%;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    margin-left: 0;
    padding: 12px 0 0;
    border-top: 1px solid var(--lx-clay-border);
    border-left: 0;
  }

  .groups-cockpit__filter-select,
  .groups-cockpit__filter-select:last-child {
    width: 100%;
  }
}

@media (max-width: 1023px) {
  .groups-cockpit__metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 18px 0;
  }

  .groups-cockpit__metric:nth-child(3) {
    border-left: 0;
  }

  .groups-cockpit__search {
    width: auto;
    max-width: none;
  }

  .groups-cockpit__ledger-header {
    display: none;
  }

  .groups-cockpit__ledger {
    overflow: visible;
  }

  .groups-cockpit__ledger-table {
    min-width: 0;
  }

  .groups-cockpit__ledger-row {
    grid-template-columns: minmax(0, 1fr) !important;
    gap: 0;
    padding: 18px;
  }

  .groups-cockpit__cell {
    display: grid;
    grid-template-columns: 132px minmax(0, 1fr);
    align-items: center;
    gap: 12px;
    padding: 11px 0;
    border-bottom: 1px solid color-mix(in srgb, var(--lx-clay-border) 70%, transparent);
  }

  .groups-cockpit__cell::before {
    content: attr(data-label);
    grid-column: 1;
    grid-row: 1 / -1;
    color: var(--lx-clay-text-subtle);
    font-size: 0.6875rem;
    font-weight: 800;
    letter-spacing: 0.06em;
  }

  .groups-cockpit__cell > * {
    grid-column: 2;
  }

  .groups-cockpit__identity {
    align-items: start;
  }

  .groups-cockpit__identity-heading,
  .groups-cockpit__identity-meta {
    grid-row: auto;
  }

  .groups-cockpit__row-actions {
    grid-template-columns: 132px repeat(3, max-content);
    justify-content: flex-start;
    border-bottom: 0;
  }

  .groups-cockpit__row-actions > * {
    grid-column: auto;
  }

  .groups-cockpit__detail {
    margin: 10px 0 0;
  }
}

@media (max-width: 767px) {
  .groups-cockpit {
    padding: 20px 16px 32px;
  }

  .groups-cockpit__header {
    align-items: stretch;
    flex-direction: column;
  }

  .groups-cockpit__header-actions {
    width: 100%;
  }

  .groups-cockpit__primary {
    flex: 1 1 auto;
  }

  .groups-cockpit__title {
    font-size: 1.625rem;
  }

  .groups-cockpit__metrics {
    margin-top: 20px;
    padding: 16px 8px;
    border-radius: 18px;
  }

  .groups-cockpit__metric {
    padding: 0 12px;
  }

  .groups-cockpit__metric-value-row {
    align-items: flex-start;
    flex-direction: column;
    gap: 2px;
  }

  .groups-cockpit__toolbar {
    margin-top: 20px;
  }

  .groups-cockpit__search {
    flex-basis: calc(100% - 104px);
  }

  .groups-cockpit__filter-toggle {
    flex: 0 0 auto;
  }

  .groups-cockpit__filters {
    grid-template-columns: minmax(0, 1fr);
  }

  .groups-cockpit__toolbar-tail {
    width: 100%;
  }

  .groups-cockpit__preferences,
  .groups-cockpit__preferences-trigger {
    width: 100%;
  }

  .groups-cockpit__preferences-menu {
    width: 100%;
  }

  .groups-cockpit__ledger {
    margin-top: 20px;
    border-radius: 14px;
  }

  .groups-cockpit__cell {
    grid-template-columns: minmax(0, 1fr);
    gap: 6px;
  }

  .groups-cockpit__cell::before,
  .groups-cockpit__cell > * {
    grid-column: 1;
  }

  .groups-cockpit__cell::before {
    grid-row: auto;
  }

  .groups-cockpit__row-actions {
    grid-template-columns: repeat(3, max-content);
  }

  .groups-cockpit__row-actions::before {
    grid-column: 1 / -1;
  }

  .groups-cockpit__row-actions > * {
    grid-column: auto;
  }

  .groups-cockpit__detail {
    padding: 18px;
  }

  .groups-cockpit__quota-cards {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (prefers-reduced-motion: reduce) {
  .groups-cockpit *,
  .groups-cockpit *::before,
  .groups-cockpit *::after {
    scroll-behavior: auto !important;
    transition-duration: 1ms !important;
    animation-duration: 1ms !important;
  }
}
</style>
