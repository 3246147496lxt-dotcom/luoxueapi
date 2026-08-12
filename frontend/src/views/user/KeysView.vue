<template>
  <AppLayout variant="chat">
    <section
      class="keys-workspace"
      data-test="keys-workspace"
    >
      <header class="keys-mobile-header md:hidden">
        <div class="keys-mobile-title">
          <KeysLucideIcon name="key" :size="20" aria-hidden="true" />
          <span>{{ t('keys.title') }}</span>
        </div>
        <div class="keys-mobile-actions">
          <button
            type="button"
            class="keys-mobile-refresh"
            :disabled="loading"
            :title="t('common.refresh')"
            :aria-label="t('common.refresh')"
            data-test="key-refresh-mobile"
            @click="loadApiKeys"
          >
            <KeysLucideIcon name="refreshCw" :size="18" :class="loading && 'animate-spin'" />
          </button>
          <button
            type="button"
            class="keys-mobile-create"
            :class="{ 'key-header-create--empty': !loading && apiKeys.length === 0 }"
            data-tour="keys-create-btn"
            data-test="key-create-header-mobile"
            @click="showCreateModal = true"
          >
            <KeysLucideIcon name="plus" :size="16" aria-hidden="true" />
            <span>{{ t('common.create') }}</span>
          </button>
        </div>
      </header>

      <header class="keys-desktop-header hidden md:flex">
        <div class="keys-desktop-title-wrap">
          <div class="keys-desktop-title-icon">
            <KeysLucideIcon name="key" :size="24" aria-hidden="true" />
          </div>
          <div class="keys-desktop-title-copy">
            <h1>{{ t('keys.title') }}</h1>
            <p>{{ t('keys.workspaceSubtitle') }}</p>
          </div>
        </div>
        <div class="keys-desktop-actions">
          <button
            type="button"
            class="keys-desktop-refresh"
            :disabled="loading"
            :title="t('common.refresh')"
            :aria-label="t('common.refresh')"
            data-test="key-refresh"
            @click="loadApiKeys"
          >
            <KeysLucideIcon name="refreshCw" :size="20" :class="loading && 'animate-spin'" />
          </button>
          <button
            type="button"
            class="key-header-create keys-desktop-create"
            :class="{ 'key-header-create--empty': !loading && apiKeys.length === 0 }"
            data-tour="keys-create-btn"
            data-test="key-create-header"
            @click="showCreateModal = true"
          >
            <KeysLucideIcon name="plus" :size="20" aria-hidden="true" />
            <span>{{ t('keys.createKey') }}</span>
          </button>
        </div>
      </header>

      <div class="keys-filter-toolbar">
        <div class="keys-filter-grid">
          <SearchInput
            v-model="filterSearch"
            class="key-filter-search"
            :placeholder="t('keys.searchPlaceholder')"
            data-test="key-filter-search"
            @search="onFilterChange"
          />
          <Select
            class="key-secondary-filter"
            :model-value="filterGroupId"
            :options="groupFilterOptions"
            data-test="key-filter-group"
            @update:model-value="onGroupFilterChange"
          />
          <Select
            class="key-secondary-filter"
            :model-value="filterStatus"
            :options="statusFilterOptions"
            data-test="key-filter-status"
            @update:model-value="onStatusFilterChange"
          />
          <Select
            class="key-secondary-filter key-sort-filter"
            :model-value="sortSelection"
            :options="sortOptions"
            data-test="key-sort"
            @update:model-value="onSortChange"
          />

          <div class="keys-filter-actions">
            <div ref="columnDropdownRef" class="relative min-w-0">
              <button
                type="button"
                class="keys-detail-settings-button"
                :title="t('keys.detailSettings')"
                :aria-expanded="showColumnDropdown"
                aria-controls="key-detail-menu"
                aria-haspopup="menu"
                data-test="key-detail-settings"
                @click="showColumnDropdown = !showColumnDropdown"
              >
                <KeysLucideIcon class="keys-settings-icon keys-settings-icon--mobile" name="slidersHorizontal" :size="18" aria-hidden="true" />
                <KeysLucideIcon class="keys-settings-icon keys-settings-icon--desktop" name="settings2" :size="18" aria-hidden="true" />
                <span>{{ t('keys.detailSettings') }}</span>
              </button>
              <div
                v-if="showColumnDropdown"
                id="key-detail-menu"
                role="menu"
                :aria-label="t('keys.detailSettings')"
                class="keys-detail-menu"
                data-test="key-detail-menu"
              >
                <button
                  v-for="col in toggleableColumns"
                  :key="col.key"
                  type="button"
                  role="menuitemcheckbox"
                  :aria-checked="isColumnVisible(col.key)"
                  class="keys-detail-menu-item"
                  @click="toggleColumn(col.key)"
                >
                  <span>{{ col.label }}</span>
                  <KeysLucideIcon v-if="isColumnVisible(col.key)" name="check" :size="16" class="text-primary-600 dark:text-primary-400" />
                </button>
              </div>
            </div>

            <button
              type="button"
              class="keys-density-button"
              :aria-pressed="compactTable"
              data-test="key-density-toggle"
              @click="compactTable = !compactTable"
            >
              <span>{{ compactTable ? t('keys.compactList') : t('keys.comfortableList') }}</span>
            </button>
          </div>
        </div>
      </div>

      <div v-if="loading" class="min-h-0 flex-1 overflow-hidden" aria-live="polite" :aria-label="t('common.loading')">
        <div class="hidden h-full grid-cols-[minmax(0,3fr)_minmax(0,2fr)] md:grid">
          <div class="keys-loading-master animate-pulse" />
          <div class="keys-loading-detail animate-pulse border-l" />
        </div>
        <div class="keys-loading-mobile h-full space-y-4 overflow-hidden p-4 md:hidden">
          <div v-for="index in 2" :key="index" class="keys-loading-card h-[360px] animate-pulse rounded-lg border" />
        </div>
      </div>

      <div v-else-if="apiKeys.length > 0" class="keys-content">
        <div class="keys-master-pane">
          <div class="min-h-0 flex-1 overflow-hidden">
            <ApiKeyWorkspaceList
              class="hidden h-full md:block"
              :api-keys="apiKeys"
              :usage-stats="usageStats"
              :user-group-rates="userGroupRates"
              :selected-key-id="inspectedKeyId"
              :compact="compactTable"
              :copied-key-id="copiedKeyId"
              :status-updating-ids="statusUpdatingIds"
              :now="now"
              @select="openKeyDetails"
              @copy-key="copyKey"
              @toggle-status="toggleKeyStatus"
              @change-group="openGroupSelector"
            />

            <div class="keys-mobile-list md:hidden" data-test="api-key-mobile-list">
              <ApiKeySummaryCard
                v-for="row in apiKeys"
                :key="row.id"
                :api-key="row"
                :usage="usageStats[row.id]"
                :user-group-rate="row.group ? userGroupRates[row.group.id] : null"
                :copied="copiedKeyId === row.id"
                :status-updating="statusUpdatingKeyIds.has(row.id)"
                :now="now"
                :visible-columns="visibleColumnKeys"
                @open-details="openKeyDetails"
                @copy-key="copyKey"
                @toggle-status="toggleKeyStatus"
                @change-group="openGroupSelector"
              />

              <div v-if="pagination.total > 0" class="keys-mobile-pagination">
                <p>{{ t('keys.workspacePageOf', { page: pagination.page, total: totalPageCount }) }}</p>
                <div>
                  <button
                    type="button"
                    :disabled="pagination.page <= 1"
                    :aria-label="t('pagination.previous')"
                    @click="handlePageChange(pagination.page - 1)"
                  >
                    <KeysLucideIcon name="chevronLeft" :size="16" />
                  </button>
                  <button
                    type="button"
                    :disabled="pagination.page >= totalPageCount"
                    :aria-label="t('pagination.next')"
                    @click="handlePageChange(pagination.page + 1)"
                  >
                    <KeysLucideIcon name="chevronRight" :size="16" />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div v-if="pagination.total > 0" class="keys-desktop-pagination hidden md:block">
            <Pagination
              :page="pagination.page"
              :total="pagination.total"
              :page-size="pagination.page_size"
              :show-page-size-selector="false"
              @update:page="handlePageChange"
              @update:pageSize="handlePageSizeChange"
            />
          </div>
        </div>

        <aside
          class="keys-detail-pane hidden md:flex"
          data-test="key-inline-inspector"
        >
          <ApiKeyInspector
            v-if="inspectedKey"
            class="h-full w-full"
            mode="inline"
            :api-key="inspectedKey"
            :usage="inspectedUsage"
            :user-group-rate="inspectedUserGroupRate"
            :public-settings="publicSettings"
            :copied="copiedKeyId === inspectedKey.id"
            :status-updating="statusUpdatingKeyIds.has(inspectedKey.id)"
            :service-tier-updating="serviceTierUpdatingKeyIds.has(inspectedKey.id)"
            :now="now"
            :show-ccs-import="!publicSettings?.hide_ccs_import_button"
            :visible-columns="visibleColumnKeys"
            @copy-key="copyKey"
            @toggle-status="toggleKeyStatus"
            @toggle-service-tier="toggleServiceTierPreference"
            @change-group="openGroupSelector"
            @reset-quota="confirmResetQuotaFromInspector"
            @reset-rate-limit="confirmResetRateLimitFromTable"
            @use-key="openUseKeyFromInspector"
            @import-ccs="importFromInspector"
            @edit="editKeyFromInspector"
            @delete="deleteKeyFromInspector"
          />
        </aside>
      </div>

      <div v-else class="keys-empty-state">
        <EmptyState :title="t('keys.noKeysYet')" :description="t('keys.createFirstKey')">
          <template #action>
            <button type="button" class="key-empty-create btn btn-primary min-h-11" data-test="key-create-empty" @click="showCreateModal = true">
              <KeysLucideIcon name="plus" :size="20" class="mr-2" aria-hidden="true" />
              {{ t('keys.createKey') }}
            </button>
          </template>
        </EmptyState>
      </div>
    </section>

    <ApiKeyDetailSheet
      :show="showKeyDetailSheet && Boolean(inspectedKey)"
      :title="t('keys.detailTitle')"
      :subtitle="inspectedKey ? `#${inspectedKey.id} · ${inspectedKey.name}` : ''"
      @close="closeKeyDetails"
    >
      <ApiKeyInspector
        v-if="inspectedKey"
        mode="sheet"
        :api-key="inspectedKey"
        :usage="inspectedUsage"
        :user-group-rate="inspectedUserGroupRate"
        :public-settings="publicSettings"
        :copied="copiedKeyId === inspectedKey.id"
        :status-updating="statusUpdatingKeyIds.has(inspectedKey.id)"
        :service-tier-updating="serviceTierUpdatingKeyIds.has(inspectedKey.id)"
        :now="now"
        :show-ccs-import="!publicSettings?.hide_ccs_import_button"
        :visible-columns="visibleColumnKeys"
        @copy-key="copyKey"
        @toggle-status="toggleKeyStatus"
        @toggle-service-tier="toggleServiceTierPreference"
        @change-group="openGroupSelector"
        @reset-quota="confirmResetQuotaFromInspector"
        @reset-rate-limit="confirmResetRateLimitFromTable"
        @use-key="openUseKeyFromInspector"
        @import-ccs="importFromInspector"
        @edit="editKeyFromInspector"
        @delete="deleteKeyFromInspector"
      />
    </ApiKeyDetailSheet>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="closeModals"
    >
      <form id="key-form" @submit.prevent="handleSubmit" class="space-y-5">
        <div>
          <label class="input-label">{{ t('keys.nameLabel') }}</label>
          <input
            v-model="formData.name"
            type="text"
            required
            class="input"
            :placeholder="t('keys.namePlaceholder')"
            data-tour="key-form-name"
          />
        </div>

        <div>
          <label class="input-label">{{ t('keys.groupLabel') }}</label>
          <Select
            v-model="formData.group_id"
            :options="groupOptions"
            :placeholder="t('keys.selectGroup')"
            :searchable="true"
            :search-placeholder="t('keys.searchGroup')"
            data-tour="key-form-group"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
              />
              <span v-else class="keys-text-muted">{{ t('keys.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
        </div>

        <!-- Custom Key Section (only for create) -->
        <div v-if="!showEditModal" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.customKeyLabel') }}</label>
            <button
              type="button"
              @click="formData.use_custom_key = !formData.use_custom_key"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.use_custom_key ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.use_custom_key ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          <div v-if="formData.use_custom_key">
            <input
              v-model="formData.custom_key"
              type="text"
              class="input font-mono"
              :placeholder="t('keys.customKeyPlaceholder')"
              :class="{ 'border-red-500 dark:border-red-500': customKeyError }"
            />
            <p v-if="customKeyError" class="mt-1 text-sm text-red-500">{{ customKeyError }}</p>
            <p v-else class="input-hint">{{ t('keys.customKeyHint') }}</p>
          </div>
        </div>

        <div v-if="showEditModal">
          <label class="input-label">{{ t('keys.statusLabel') }}</label>
          <Select
            v-model="formData.status"
            :options="statusOptions"
            :placeholder="t('keys.selectStatus')"
          />
        </div>

        <!-- IP Restriction Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.ipRestriction') }}</label>
            <button
              type="button"
              @click="formData.enable_ip_restriction = !formData.enable_ip_restriction"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_ip_restriction ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_ip_restriction ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_ip_restriction" class="space-y-4 pt-2">
            <div>
              <label class="input-label">{{ t('keys.ipWhitelist') }}</label>
              <textarea
                v-model="formData.ip_whitelist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipWhitelistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipWhitelistHint') }}</p>
            </div>

            <div>
              <label class="input-label">{{ t('keys.ipBlacklist') }}</label>
              <textarea
                v-model="formData.ip_blacklist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipBlacklistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipBlacklistHint') }}</p>
            </div>
          </div>
        </div>

        <!-- Quota Limit Section -->
        <div class="space-y-3">
          <label class="input-label">{{ t('keys.quotaLimit') }}</label>
          <!-- Switch commented out - always show input, 0 = unlimited
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.quotaLimit') }}</label>
            <button
              type="button"
              @click="formData.enable_quota = !formData.enable_quota"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_quota ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_quota ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          -->

          <div class="space-y-4">
            <div>
              <div class="relative">
                <SnowflakeCreditIcon class="absolute left-3 top-1/2 -translate-y-1/2" size="sm" />
                <input
                  v-model.number="formData.quota"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-10"
                  :placeholder="t('keys.quotaAmountPlaceholder')"
                />
              </div>
              <p class="input-hint">{{ t('keys.quotaAmountHint') }}</p>
            </div>

            <!-- Quota used display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey && selectedKey.quota > 0">
              <label class="input-label">{{ t('keys.quotaUsed') }}</label>
              <div class="flex items-center gap-2">
                <div class="keys-form-usage-surface flex-1 rounded-lg px-3 py-2">
                  <CreditAmount
                    class="keys-text-primary font-medium"
                    :value="selectedKey.quota_used?.toFixed(4) || '0.0000'"
                    icon-size="xs"
                  />
                  <span class="keys-text-muted mx-2">/</span>
                  <CreditAmount
                    class="keys-text-secondary"
                    :value="selectedKey.quota?.toFixed(2) || '0.00'"
                    icon-size="xs"
                  />
                </div>
                <button
                  type="button"
                  @click="confirmResetQuota"
                  class="btn btn-secondary text-sm"
                  :title="t('keys.resetQuotaUsed')"
                >
                  {{ t('keys.reset') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Rate Limit Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.rateLimitSection') }}</label>
            <button
              type="button"
              @click="formData.enable_rate_limit = !formData.enable_rate_limit"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_rate_limit ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_rate_limit ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_rate_limit" class="space-y-4 pt-2">
            <p class="input-hint -mt-2">{{ t('keys.rateLimitHint') }}</p>
            <!-- 5-Hour Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit5h') }}</label>
              <div class="relative">
                <SnowflakeCreditIcon class="absolute left-3 top-1/2 -translate-y-1/2" size="sm" />
                <input
                  v-model.number="formData.rate_limit_5h"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-10"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_5h > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="keys-form-usage-surface flex-1 rounded-lg px-3 py-2 text-sm">
                    <CreditAmount
                      :value="selectedKey.usage_5h?.toFixed(4) || '0.0000'"
                      icon-size="xs"
                      :class="[
                      'font-medium',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'text-red-500' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'text-yellow-500' :
                      'keys-text-primary'
                    ]"
                    />
                    <span class="keys-text-muted mx-2">/</span>
                    <CreditAmount
                      class="keys-text-secondary"
                      :value="selectedKey.rate_limit_5h?.toFixed(2) || '0.00'"
                      icon-size="xs"
                    />
                  </div>
                </div>
                <div class="keys-form-progress-track mt-1 h-1.5 w-full overflow-hidden rounded-full">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'bg-red-500' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_5h / selectedKey.rate_limit_5h) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Daily Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit1d') }}</label>
              <div class="relative">
                <SnowflakeCreditIcon class="absolute left-3 top-1/2 -translate-y-1/2" size="sm" />
                <input
                  v-model.number="formData.rate_limit_1d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-10"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_1d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="keys-form-usage-surface flex-1 rounded-lg px-3 py-2 text-sm">
                    <CreditAmount
                      :value="selectedKey.usage_1d?.toFixed(4) || '0.0000'"
                      icon-size="xs"
                      :class="[
                      'font-medium',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'text-red-500' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'text-yellow-500' :
                      'keys-text-primary'
                    ]"
                    />
                    <span class="keys-text-muted mx-2">/</span>
                    <CreditAmount
                      class="keys-text-secondary"
                      :value="selectedKey.rate_limit_1d?.toFixed(2) || '0.00'"
                      icon-size="xs"
                    />
                  </div>
                </div>
                <div class="keys-form-progress-track mt-1 h-1.5 w-full overflow-hidden rounded-full">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'bg-red-500' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_1d / selectedKey.rate_limit_1d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- 7-Day Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit7d') }}</label>
              <div class="relative">
                <SnowflakeCreditIcon class="absolute left-3 top-1/2 -translate-y-1/2" size="sm" />
                <input
                  v-model.number="formData.rate_limit_7d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-10"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_7d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="keys-form-usage-surface flex-1 rounded-lg px-3 py-2 text-sm">
                    <CreditAmount
                      :value="selectedKey.usage_7d?.toFixed(4) || '0.0000'"
                      icon-size="xs"
                      :class="[
                      'font-medium',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'text-red-500' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'text-yellow-500' :
                      'keys-text-primary'
                    ]"
                    />
                    <span class="keys-text-muted mx-2">/</span>
                    <CreditAmount
                      class="keys-text-secondary"
                      :value="selectedKey.rate_limit_7d?.toFixed(2) || '0.00'"
                      icon-size="xs"
                    />
                  </div>
                </div>
                <div class="keys-form-progress-track mt-1 h-1.5 w-full overflow-hidden rounded-full">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'bg-red-500' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_7d / selectedKey.rate_limit_7d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Reset Rate Limit button (edit mode only) -->
            <div v-if="showEditModal && selectedKey && (selectedKey.rate_limit_5h > 0 || selectedKey.rate_limit_1d > 0 || selectedKey.rate_limit_7d > 0)">
              <button
                type="button"
                @click="confirmResetRateLimit"
                class="btn btn-secondary text-sm"
              >
                {{ t('keys.resetRateLimitUsage') }}
              </button>
            </div>
          </div>
        </div>

        <!-- Expiration Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.expiration') }}</label>
            <button
              type="button"
              @click="formData.enable_expiration = !formData.enable_expiration"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_expiration ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_expiration ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_expiration" class="space-y-4 pt-2">
            <!-- Quick select buttons (for both create and edit mode) -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="days in ['7', '30', '90']"
                :key="days"
                type="button"
                @click="setExpirationDays(parseInt(days))"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === days
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'keys-neutral-choice'
                ]"
              >
                {{ showEditModal ? t('keys.extendDays', { days }) : t('keys.expiresInDays', { days }) }}
              </button>
              <button
                type="button"
                @click="formData.expiration_preset = 'custom'"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === 'custom'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'keys-neutral-choice'
                ]"
              >
                {{ t('keys.customDate') }}
              </button>
            </div>

            <!-- Date picker (always show for precise adjustment) -->
            <div>
              <label class="input-label">{{ t('keys.expirationDate') }}</label>
              <input
                v-model="formData.expiration_date"
                type="datetime-local"
                class="input"
              />
              <p class="input-hint">{{ t('keys.expirationDateHint') }}</p>
            </div>

            <!-- Current expiration display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey?.expires_at" class="text-sm">
              <span class="keys-text-secondary">{{ t('keys.currentExpiration') }}: </span>
              <span class="keys-text-primary font-medium">
                {{ formatDateTime(selectedKey.expires_at) }}
              </span>
            </div>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeModals" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            form="key-form"
            type="submit"
            :disabled="submitting"
            class="btn btn-primary"
            data-tour="key-form-submit"
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
            {{
              submitting
                ? t('keys.saving')
                : showEditModal
                  ? t('common.update')
                  : t('common.create')
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Reset Quota Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetQuotaDialog"
      :title="t('keys.resetQuotaTitle')"
      :message="t('keys.resetQuotaConfirmMessage', { name: selectedKey?.name, used: selectedKey?.quota_used?.toFixed(4) })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetQuotaUsed"
      @cancel="showResetQuotaDialog = false"
    />

    <!-- Reset Rate Limit Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetRateLimitDialog"
      :title="t('keys.resetRateLimitTitle')"
      :message="t('keys.resetRateLimitConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetRateLimitUsage"
      @cancel="showResetRateLimitDialog = false"
    />

    <!-- Fast mode cost confirmation -->
    <ConfirmDialog
      :show="showServiceTierConfirmDialog"
      :title="t('keys.serviceTierEnableTitle')"
      :message="t('keys.serviceTierEnableConfirmMessage')"
      :confirm-text="t('keys.serviceTierEnableConfirm')"
      :cancel-text="t('common.cancel')"
      @confirm="confirmServiceTierEnable"
      @cancel="cancelServiceTierEnable"
    />

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="selectedKey?.key || ''"
      :base-url="publicSettings?.api_base_url || ''"
      :platform="selectedKey?.group?.platform || null"
      :allow-messages-dispatch="selectedKey?.group?.allow_messages_dispatch || false"
      @close="closeUseKeyModal"
    />

    <!-- CCS Client Selection Dialog for Antigravity -->
    <BaseDialog
      :show="showCcsClientSelect"
      :title="t('keys.ccsClientSelect.title')"
      width="narrow"
      @close="closeCcsClientSelect"
    >
      <div class="space-y-4">
        <p class="keys-text-secondary text-sm">
          {{ t('keys.ccsClientSelect.description') }}
	        </p>
	        <div class="grid grid-cols-2 gap-3">
	          <button
	            @click="handleCcsClientSelect('claude')"
	            class="keys-client-option flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all hover:border-primary-500 hover:bg-primary-50 dark:hover:border-primary-500 dark:hover:bg-primary-900/20"
	          >
	            <Icon name="terminal" size="xl" class="keys-text-secondary" />
	            <span class="keys-text-primary font-medium">{{
	              t('keys.ccsClientSelect.claudeCode')
	            }}</span>
	            <span class="keys-text-secondary text-xs">{{
	              t('keys.ccsClientSelect.claudeCodeDesc')
	            }}</span>
	          </button>
	          <button
	            @click="handleCcsClientSelect('gemini')"
	            class="keys-client-option flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all hover:border-primary-500 hover:bg-primary-50 dark:hover:border-primary-500 dark:hover:bg-primary-900/20"
	          >
	            <Icon name="sparkles" size="xl" class="keys-text-secondary" />
	            <span class="keys-text-primary font-medium">{{
	              t('keys.ccsClientSelect.geminiCli')
	            }}</span>
	            <span class="keys-text-secondary text-xs">{{
	              t('keys.ccsClientSelect.geminiCliDesc')
	            }}</span>
	          </button>
	        </div>
	      </div>
      <template #footer>
        <div class="flex justify-end">
          <button @click="closeCcsClientSelect" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Selector Dropdown (Teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div
        v-if="groupSelectorKeyId !== null && dropdownPosition"
        ref="dropdownRef"
        role="dialog"
        :aria-label="t('keys.selectGroup')"
        class="keys-group-selector animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-[min(380px,calc(100vw-2rem))] overflow-hidden rounded-xl shadow-lg duration-200"
        style="pointer-events: auto !important;"
        :style="{
          top: dropdownPosition.top !== undefined ? dropdownPosition.top + 'px' : undefined,
          bottom: dropdownPosition.bottom !== undefined ? dropdownPosition.bottom + 'px' : undefined,
          left: dropdownPosition.left + 'px'
        }"
      >
        <!-- Search box -->
        <div class="keys-group-selector-header border-b p-2">
          <div class="relative">
            <svg class="keys-text-muted absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="groupSearchQuery"
              type="text"
              class="keys-group-search w-full rounded-lg border py-1.5 pl-8 pr-3 text-sm outline-none focus:border-primary-300 focus:ring-1 focus:ring-primary-300 dark:focus:border-primary-600 dark:focus:ring-primary-600"
              :placeholder="t('keys.searchGroup')"
              :aria-label="t('keys.searchGroup')"
              @click.stop
              @keydown.esc.stop="closeGroupSelectorMenu"
            />
          </div>
        </div>
        <!-- Group list -->
        <div class="max-h-80 overflow-y-auto p-1.5" role="listbox" :aria-label="t('keys.selectGroup')">
          <button
            v-for="option in filteredGroupOptions"
            :key="option.value ?? 'null'"
            role="option"
            :aria-selected="selectedKeyForGroup?.group_id === option.value"
            @click="changeSelectedGroup(option.value)"
            :class="[
              'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm transition-colors',
              'keys-group-option border-b last:border-0',
              selectedKeyForGroup?.group_id === option.value ||
              (!selectedKeyForGroup?.group_id && option.value === null)
                ? 'keys-group-option--selected'
                : 'keys-group-option--idle'
            ]"
            :title="option.description || undefined"
          >
            <GroupOptionItem
              :name="option.label"
              :platform="option.platform"
              :subscription-type="option.subscriptionType"
              :rate-multiplier="option.rate"
              :user-rate-multiplier="option.userRate"
              :peak-rate-enabled="option.peakRateEnabled"
              :peak-start="option.peakStart"
              :peak-end="option.peakEnd"
              :peak-rate-multiplier="option.peakRateMultiplier"
              :description="option.description"
              :selected="
                selectedKeyForGroup?.group_id === option.value ||
                (!selectedKeyForGroup?.group_id && option.value === null)
              "
            />
          </button>
          <!-- Empty state when search has no results -->
          <div v-if="filteredGroupOptions.length === 0" class="keys-text-muted py-4 text-center text-sm">
            {{ t('keys.noGroupFound') }}
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { authAPI, keysAPI, usageAPI, userGroupsAPI } from '@/api'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Pagination from '@/components/common/Pagination.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import Icon from '@/components/icons/Icon.vue'
import KeysLucideIcon from '@/components/keys/KeysLucideIcon.vue'
import SnowflakeCreditIcon from '@/components/icons/SnowflakeCreditIcon.vue'
import ApiKeyDetailSheet from '@/components/keys/ApiKeyDetailSheet.vue'
import ApiKeyInspector from '@/components/keys/ApiKeyInspector.vue'
import ApiKeySummaryCard from '@/components/keys/ApiKeySummaryCard.vue'
import ApiKeyWorkspaceList from '@/components/keys/ApiKeyWorkspaceList.vue'
import UseKeyModal from '@/components/keys/UseKeyModal.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useClipboard } from '@/composables/useClipboard'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import { useOnboardingStore } from '@/stores/onboarding'
import type { ApiKey, Group, GroupPlatform, PublicSettings, SubscriptionType, UpdateApiKeyRequest } from '@/types'
import { formatDateTime } from '@/utils/format'
import {
  buildCcSwitchImportDeeplink,
  type CcSwitchClientType
} from '@/utils/ccswitchImport'

const { t } = useI18n()

// Helper to format date for datetime-local input
const formatDateTimeLocal = (isoDate: string): string => {
  const date = new Date(isoDate)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

interface GroupOption {
  value: number
  label: string
  description: string | null
  rate: number
  userRate: number | null
  peakRateEnabled: boolean
  peakStart: string
  peakEnd: string
  peakRateMultiplier: number
  subscriptionType: SubscriptionType
  platform: GroupPlatform
}

const appStore = useAppStore()
const onboardingStore = useOnboardingStore()
const { copyToClipboard: clipboardCopy } = useClipboard()

const allColumns = computed<Column[]>(() => [
  { key: 'name', label: t('common.name'), sortable: true },
  { key: 'id', label: t('keys.id'), sortable: true },
  { key: 'key', label: t('keys.apiKey'), sortable: false },
  { key: 'group', label: t('keys.group'), sortable: false },
  { key: 'current_concurrency', label: t('keys.currentConcurrency'), sortable: true },
  { key: 'usage', label: t('keys.usage'), sortable: false },
  { key: 'rate_limit', label: t('keys.rateLimitColumn'), sortable: false },
  { key: 'expires_at', label: t('keys.expiresAt'), sortable: true },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'last_used_at', label: t('keys.lastUsedAt'), sortable: true },
  { key: 'last_used_ip', label: t('keys.lastUsedIP'), sortable: false },
  { key: 'created_at', label: t('keys.created'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false }
])

const ALWAYS_VISIBLE_COLUMNS = new Set(['name', 'id', 'status', 'usage', 'group', 'key', 'actions'])
const DEFAULT_HIDDEN_COLUMNS: string[] = []
const HIDDEN_COLUMNS_KEY = 'api-key-hidden-columns'
const COLUMN_SETTINGS_VERSION_KEY = 'api-key-column-settings-version'
const COLUMN_SETTINGS_VERSION = 4
const VERSION_NEW_HIDDEN_COLUMNS: Record<number, string[]> = {
  2: ['last_used_ip'],
  3: ['id']
}
const VERSION_NEW_VISIBLE_COLUMNS: Record<number, string[]> = {
  4: ['id', 'rate_limit', 'last_used_at', 'last_used_ip']
}

const toggleableColumns = computed(() =>
  allColumns.value.filter((col) => !ALWAYS_VISIBLE_COLUMNS.has(col.key))
)

const hiddenColumns = reactive<Set<string>>(new Set())

const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
    localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
  } catch (error) {
    console.error('Failed to save API key table columns:', error)
  }
}

const loadSavedColumns = () => {
  hiddenColumns.clear()
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      const validColumnKeys = new Set(allColumns.value.map((col) => col.key))
      parsed
        .filter((key) =>
          typeof key === 'string' &&
          validColumnKeys.has(key) &&
          !ALWAYS_VISIBLE_COLUMNS.has(key)
        )
        .forEach((key) => hiddenColumns.add(key))
      const storedVersion = Number(localStorage.getItem(COLUMN_SETTINGS_VERSION_KEY) ?? '1')
      if (storedVersion < COLUMN_SETTINGS_VERSION) {
        for (let v = storedVersion + 1; v <= COLUMN_SETTINGS_VERSION; v++) {
          for (const key of VERSION_NEW_HIDDEN_COLUMNS[v] ?? []) {
            if (validColumnKeys.has(key) && !ALWAYS_VISIBLE_COLUMNS.has(key)) {
              hiddenColumns.add(key)
            }
          }
          for (const key of VERSION_NEW_VISIBLE_COLUMNS[v] ?? []) {
            hiddenColumns.delete(key)
          }
        }
        saveColumnsToStorage()
      } else {
        localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
      }
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
      localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
    }
  } catch (error) {
    console.error('Failed to load API key table columns:', error)
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
  }
}

const toggleColumn = (key: string) => {
  if (ALWAYS_VISIBLE_COLUMNS.has(key)) return
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
}

const isColumnVisible = (key: string) => !hiddenColumns.has(key)

const visibleColumnKeys = computed(() =>
  allColumns.value
    .filter((column) => isColumnVisible(column.key))
    .map((column) => column.key)
)

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const submitting = ref(false)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})
const userGroupRates = ref<Record<number, number>>({})

const pagination = ref({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})
const totalPageCount = computed(() => Math.max(
  1,
  pagination.value.pages || Math.ceil(pagination.value.total / pagination.value.page_size)
))
const sortState = ref({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const sortSelection = computed(() => `${sortState.value.sort_by}:${sortState.value.sort_order}`)

const sortOptions = computed(() => [
  { value: 'created_at:desc', label: t('keys.sortCreatedDesc') },
  { value: 'created_at:asc', label: t('keys.sortCreatedAsc') },
  { value: 'id:desc', label: t('keys.sortIdDesc') },
  { value: 'id:asc', label: t('keys.sortIdAsc') },
  { value: 'name:asc', label: t('keys.sortNameAsc') },
  { value: 'name:desc', label: t('keys.sortNameDesc') },
  { value: 'status:asc', label: t('keys.sortStatusAsc') },
  { value: 'status:desc', label: t('keys.sortStatusDesc') },
  { value: 'expires_at:asc', label: t('keys.sortExpirationAsc') },
  { value: 'expires_at:desc', label: t('keys.sortExpirationDesc') },
  { value: 'last_used_at:desc', label: t('keys.sortLastUsedDesc') },
  { value: 'last_used_at:asc', label: t('keys.sortLastUsedAsc') },
  { value: 'current_concurrency:desc', label: t('keys.sortConcurrencyDesc') },
  { value: 'current_concurrency:asc', label: t('keys.sortConcurrencyAsc') }
])

// Filter state
const filterSearch = ref('')
const filterStatus = ref('')
const filterGroupId = ref<string | number>('')

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showResetQuotaDialog = ref(false)
const showResetRateLimitDialog = ref(false)
const showUseKeyModal = ref(false)
const showCcsClientSelect = ref(false)
const showColumnDropdown = ref(false)
const compactTable = ref(false)
const statusUpdatingKeyIds = reactive(new Set<number>())
const statusUpdatingIds = computed(() => Array.from(statusUpdatingKeyIds))
const serviceTierUpdatingKeyIds = reactive(new Set<number>())
const showServiceTierConfirmDialog = ref(false)
const pendingServiceTierKey = ref<ApiKey | null>(null)
const pendingCcsRow = ref<ApiKey | null>(null)
const selectedKey = ref<ApiKey | null>(null)
const inspectedKeyId = ref<number | null>(null)
const showKeyDetailSheet = ref(false)
const hasInlineInspector = ref(true)
const copiedKeyId = ref<number | null>(null)
const groupSelectorKeyId = ref<number | null>(null)
const publicSettings = ref<PublicSettings | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const columnDropdownRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<{ top?: number; bottom?: number; left: number } | null>(null)
let abortController: AbortController | null = null
let inlineInspectorMediaQuery: MediaQueryList | null = null

const inspectedKey = computed(() => {
  if (inspectedKeyId.value === null) return null
  return apiKeys.value.find((key) => key.id === inspectedKeyId.value) ?? null
})

const inspectedUsage = computed(() => {
  if (!inspectedKey.value) return undefined
  return usageStats.value[String(inspectedKey.value.id)]
})

const inspectedUserGroupRate = computed(() => {
  const groupId = inspectedKey.value?.group?.id
  return groupId === undefined ? null : userGroupRates.value[groupId] ?? null
})

// Get the currently selected key for group change
const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const formData = ref({
  name: '',
  group_id: null as number | null,
  status: 'active' as 'active' | 'inactive',
  use_custom_key: false,
  custom_key: '',
  enable_ip_restriction: false,
  ip_whitelist: '',
  ip_blacklist: '',
  // Quota settings (empty = unlimited)
  enable_quota: false,
  quota: null as number | null,
  // Rate limit settings
  enable_rate_limit: false,
  rate_limit_5h: null as number | null,
  rate_limit_1d: null as number | null,
  rate_limit_7d: null as number | null,
  enable_expiration: false,
  expiration_preset: '30' as '7' | '30' | '90' | 'custom',
  expiration_date: ''
})

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return ''
  }
  const key = formData.value.custom_key
  if (key.length < 16) {
    return t('keys.customKeyTooShort')
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t('keys.customKeyInvalidChars')
  }
  return ''
})

const statusOptions = computed(() => [
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') }
])

const shouldSubmitEditStatus = (key: ApiKey, status: 'active' | 'inactive') => {
  if (key.status === 'quota_exhausted' || key.status === 'expired') {
    return status === 'active'
  }
  return true
}

// Filter dropdown options
const groupFilterOptions = computed(() => [
  { value: '', label: t('keys.allGroups') },
  { value: 0, label: t('keys.noGroup') },
  ...groups.value.map((g) => ({ value: g.id, label: g.name }))
])

const statusFilterOptions = computed(() => [
  { value: '', label: t('keys.allStatus') },
  { value: 'active', label: t('keys.status.active') },
  { value: 'inactive', label: t('keys.status.inactive') },
  { value: 'quota_exhausted', label: t('keys.status.quota_exhausted') },
  { value: 'expired', label: t('keys.status.expired') }
])

const onFilterChange = () => {
  pagination.value.page = 1
  loadApiKeys()
}

const onGroupFilterChange = (value: string | number | boolean | null) => {
  filterGroupId.value = value as string | number
  onFilterChange()
}

const onStatusFilterChange = (value: string | number | boolean | null) => {
  filterStatus.value = value as string
  onFilterChange()
}

const onSortChange = (value: string | number | boolean | null) => {
  if (typeof value !== 'string') return
  const [sortBy, sortOrder] = value.split(':')
  if (!sortBy || (sortOrder !== 'asc' && sortOrder !== 'desc')) return
  handleSort(sortBy, sortOrder)
}

// Convert groups to Select options format with rate multiplier and subscription type
const groupOptions = computed(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    rate: group.rate_multiplier,
    userRate: userGroupRates.value[group.id] ?? null,
    peakRateEnabled: group.peak_rate_enabled,
    peakStart: group.peak_start,
    peakEnd: group.peak_end,
    peakRateMultiplier: group.peak_rate_multiplier,
    subscriptionType: group.subscription_type,
    platform: group.platform
  }))
)

// Group dropdown search
const groupSearchQuery = ref('')
const filteredGroupOptions = computed(() => {
  const query = groupSearchQuery.value.trim().toLowerCase()
  if (!query) return groupOptions.value
  return groupOptions.value.filter((opt) => {
    return opt.label.toLowerCase().includes(query) ||
      (opt.description && opt.description.toLowerCase().includes(query))
  })
})

const copyToClipboard = async (text: string, keyId: number) => {
  const success = await clipboardCopy(text, t('keys.copied'))
  if (success) {
    copiedKeyId.value = keyId
    setTimeout(() => {
      copiedKeyId.value = null
    }, 800)
  }
}

const copyKey = (key: ApiKey) => copyToClipboard(key.key, key.id)

const closeKeyDetails = () => {
  showKeyDetailSheet.value = false
  closeGroupSelectorMenu()
}

const openKeyDetails = (key: ApiKey) => {
  inspectedKeyId.value = key.id
  if (!hasInlineInspector.value) {
    showKeyDetailSheet.value = true
  }
}

const syncInspectedKey = (items: ApiKey[]) => {
  if (items.length === 0) {
    inspectedKeyId.value = null
    showKeyDetailSheet.value = false
    return
  }

  const currentId = inspectedKeyId.value
  if (currentId !== null && items.some((key) => key.id === currentId)) return

  inspectedKeyId.value = items[0].id
  if (currentId !== null) {
    showKeyDetailSheet.value = false
  }
}

const handleInlineInspectorChange = (event: MediaQueryListEvent) => {
  hasInlineInspector.value = event.matches
  if (event.matches) {
    showKeyDetailSheet.value = false
  }
}

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const { name, code } = error as { name?: string; code?: string }
  return name === 'AbortError' || code === 'ERR_CANCELED'
}

const loadApiKeys = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const { signal } = controller
  loading.value = true
  try {
    // Build filters
    const filters: {
      search?: string
      status?: string
      group_id?: number | string
      sort_by?: string
      sort_order?: 'asc' | 'desc'
    } = {}
    if (filterSearch.value) filters.search = filterSearch.value
    if (filterStatus.value) filters.status = filterStatus.value
    if (filterGroupId.value !== '') filters.group_id = filterGroupId.value
    filters.sort_by = sortState.value.sort_by
    filters.sort_order = sortState.value.sort_order

    const response = await keysAPI.list(pagination.value.page, pagination.value.page_size, filters, {
      signal
    })
    if (signal.aborted) return
    apiKeys.value = response.items
    syncInspectedKey(response.items)
    pagination.value.total = response.total
    pagination.value.pages = response.pages
    usageStats.value = {}

    // Load usage stats for all API keys in the list
    if (response.items.length > 0) {
      const keyIds = response.items.map((k) => k.id)
      try {
        const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, { signal })
        if (signal.aborted) return
        usageStats.value = usageResponse.stats
      } catch (e) {
        if (!isAbortError(e)) {
          console.error('Failed to load usage stats:', e)
        }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    appStore.showError(t('keys.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
    }
  }
}

const loadGroups = async () => {
  try {
    groups.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const loadUserGroupRates = async () => {
  try {
    userGroupRates.value = await userGroupsAPI.getUserGroupRates()
  } catch (error) {
    console.error('Failed to load user group rates:', error)
  }
}

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings()
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
}

const openUseKeyModal = (key: ApiKey) => {
  selectedKey.value = key
  showUseKeyModal.value = true
}

const closeUseKeyModal = () => {
  showUseKeyModal.value = false
  selectedKey.value = null
}

const openUseKeyFromInspector = (key: ApiKey) => {
  showKeyDetailSheet.value = false
  openUseKeyModal(key)
}

const editKeyFromInspector = (key: ApiKey) => {
  showKeyDetailSheet.value = false
  editKey(key)
}

const deleteKeyFromInspector = (key: ApiKey) => {
  showKeyDetailSheet.value = false
  confirmDelete(key)
}

const importFromInspector = (key: ApiKey) => {
  showKeyDetailSheet.value = false
  importToCcswitch(key)
}

const confirmResetQuotaFromInspector = (key: ApiKey) => {
  showKeyDetailSheet.value = false
  selectedKey.value = key
  showResetQuotaDialog.value = true
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadApiKeys()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.page_size = pageSize
  pagination.value.page = 1
  loadApiKeys()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.value.sort_by = key
  sortState.value.sort_order = order
  pagination.value.page = 1
  loadApiKeys()
}

const editKey = (key: ApiKey) => {
  selectedKey.value = key
  const hasIPRestriction = (key.ip_whitelist?.length > 0) || (key.ip_blacklist?.length > 0)
  const hasExpiration = !!key.expires_at
  formData.value = {
    name: key.name,
    group_id: key.group_id,
    status: key.status === 'quota_exhausted' || key.status === 'expired' ? 'inactive' : key.status,
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join('\n'),
    ip_blacklist: (key.ip_blacklist || []).join('\n'),
    enable_quota: key.quota > 0,
    quota: key.quota > 0 ? key.quota : null,
    enable_rate_limit: (key.rate_limit_5h > 0) || (key.rate_limit_1d > 0) || (key.rate_limit_7d > 0),
    rate_limit_5h: key.rate_limit_5h || null,
    rate_limit_1d: key.rate_limit_1d || null,
    rate_limit_7d: key.rate_limit_7d || null,
    enable_expiration: hasExpiration,
    expiration_preset: 'custom',
    expiration_date: key.expires_at ? formatDateTimeLocal(key.expires_at) : ''
  }
  showEditModal.value = true
}

const toggleKeyStatus = async (key: ApiKey) => {
  if (statusUpdatingKeyIds.has(key.id)) return
  const newStatus = key.status === 'active' ? 'inactive' : 'active'
  statusUpdatingKeyIds.add(key.id)
  try {
    await keysAPI.toggleStatus(key.id, newStatus)
    appStore.showSuccess(
      newStatus === 'active' ? t('keys.keyEnabledSuccess') : t('keys.keyDisabledSuccess')
    )
    await loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToUpdateStatus'))
  } finally {
    statusUpdatingKeyIds.delete(key.id)
  }
}

const updateServiceTierPreference = async (key: ApiKey, preference: 'standard' | 'priority') => {
  if (serviceTierUpdatingKeyIds.has(key.id)) return
  serviceTierUpdatingKeyIds.add(key.id)
  try {
    const updatedKey = await keysAPI.update(key.id, {
      service_tier_preference: preference
    })
    const localIndex = apiKeys.value.findIndex((item) => item.id === key.id)
    if (localIndex >= 0 && updatedKey) {
      // Keep list-only relations/usage fields if the update response is compact.
      apiKeys.value[localIndex] = { ...apiKeys.value[localIndex], ...updatedKey }
    }
    appStore.showSuccess(
      preference === 'priority'
        ? t('keys.serviceTierEnabledSuccess')
        : t('keys.serviceTierDisabledSuccess')
    )
    await loadApiKeys()
  } catch (error) {
    // Keep the previous value in place so a failed request naturally rolls back.
    appStore.showError(t('keys.serviceTierUpdateFailed'))
  } finally {
    serviceTierUpdatingKeyIds.delete(key.id)
  }
}

const toggleServiceTierPreference = (key: ApiKey) => {
  if (key.group?.platform !== 'openai' || serviceTierUpdatingKeyIds.has(key.id)) return
  const isPriority = key.service_tier_preference === 'priority'
  if (isPriority) {
    void updateServiceTierPreference(key, 'standard')
    return
  }
  pendingServiceTierKey.value = key
  showServiceTierConfirmDialog.value = true
}

const cancelServiceTierEnable = () => {
  showServiceTierConfirmDialog.value = false
  pendingServiceTierKey.value = null
}

const confirmServiceTierEnable = () => {
  const key = pendingServiceTierKey.value
  cancelServiceTierEnable()
  if (key) {
    void updateServiceTierPreference(key, 'priority')
  }
}

const openGroupSelector = (key: ApiKey, event: MouseEvent) => {
  if (groupSelectorKeyId.value === key.id) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  } else {
    const buttonEl = event.currentTarget as HTMLElement | null
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      const dropdownEstHeight = 400 // estimated max dropdown height
      const dropdownWidth = Math.min(380, window.innerWidth - 32)
      const spaceBelow = window.innerHeight - rect.bottom
      const spaceAbove = rect.top
      const left = Math.min(
        Math.max(16, rect.left),
        Math.max(16, window.innerWidth - dropdownWidth - 16)
      )

      if (spaceBelow < dropdownEstHeight && spaceAbove > spaceBelow) {
        // Not enough space below, pop upward
        dropdownPosition.value = {
          bottom: window.innerHeight - rect.top + 4,
          left
        }
      } else {
        // Default: pop downward
        dropdownPosition.value = {
          top: rect.bottom + 4,
          left
        }
      }
    }
    groupSelectorKeyId.value = key.id
    groupSearchQuery.value = ''
  }
}

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
  if (key.group_id === newGroupId) return

  try {
    await keysAPI.update(key.id, { group_id: newGroupId })
    appStore.showSuccess(t('keys.groupChangedSuccess'))
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToChangeGroup'))
  }
}

const changeSelectedGroup = (newGroupId: number | null) => {
  const key = selectedKeyForGroup.value
  if (!key) {
    closeGroupSelectorMenu()
    return
  }
  changeGroup(key, newGroupId)
}

const closeGroupSelectorMenu = () => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
}

const closeGroupSelector = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside the dropdown or the trigger button
  if (!target.closest('.group\\/dropdown') && !dropdownRef.value?.contains(target)) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  }
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(target)) {
    showColumnDropdown.value = false
  }
}

const handleEscapeKey = (event: KeyboardEvent) => {
  if (event.key !== 'Escape') return
  closeGroupSelectorMenu()
  showColumnDropdown.value = false
}

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key
  showDeleteDialog.value = true
}

const handleSubmit = async () => {
  // Validate group_id is required
  if (formData.value.group_id === null) {
    appStore.showError(t('keys.groupRequired'))
    return
  }

  // Validate custom key if enabled
  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t('keys.customKeyRequired'))
      return
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value)
      return
    }
  }

  // Parse IP lists only if IP restriction is enabled
  const parseIPList = (text: string): string[] =>
    text.split('\n').map(ip => ip.trim()).filter(ip => ip.length > 0)
  const ipWhitelist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_whitelist) : []
  const ipBlacklist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_blacklist) : []

  // Calculate quota value (null/empty/0 = unlimited, stored as 0)
  const quota = formData.value.quota && formData.value.quota > 0 ? formData.value.quota : 0

  // Calculate expiration
  let expiresInDays: number | undefined
  let expiresAt: string | null | undefined
  if (formData.value.enable_expiration && formData.value.expiration_date) {
    if (!showEditModal.value) {
      // Create mode: calculate days from date
      const expDate = new Date(formData.value.expiration_date)
      const now = new Date()
      const diffDays = Math.ceil((expDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
      expiresInDays = diffDays > 0 ? diffDays : 1
    } else {
      // Edit mode: use custom date directly
      expiresAt = new Date(formData.value.expiration_date).toISOString()
    }
  } else if (showEditModal.value) {
    // Edit mode: if expiration disabled or date cleared, send empty string to clear
    expiresAt = ''
  }

  // Calculate rate limit values (send 0 when toggle is off)
  const rateLimitData = formData.value.enable_rate_limit ? {
    rate_limit_5h: formData.value.rate_limit_5h && formData.value.rate_limit_5h > 0 ? formData.value.rate_limit_5h : 0,
    rate_limit_1d: formData.value.rate_limit_1d && formData.value.rate_limit_1d > 0 ? formData.value.rate_limit_1d : 0,
    rate_limit_7d: formData.value.rate_limit_7d && formData.value.rate_limit_7d > 0 ? formData.value.rate_limit_7d : 0,
  } : { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 }

  submitting.value = true
  try {
    if (showEditModal.value && selectedKey.value) {
      const updates: UpdateApiKeyRequest = {
        name: formData.value.name,
        group_id: formData.value.group_id,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota: quota,
        expires_at: expiresAt,
        rate_limit_5h: rateLimitData.rate_limit_5h,
        rate_limit_1d: rateLimitData.rate_limit_1d,
        rate_limit_7d: rateLimitData.rate_limit_7d,
      }
      if (shouldSubmitEditStatus(selectedKey.value, formData.value.status)) {
        updates.status = formData.value.status
      }
      await keysAPI.update(selectedKey.value.id, updates)
      appStore.showSuccess(t('keys.keyUpdatedSuccess'))
    } else {
      const customKey = formData.value.use_custom_key ? formData.value.custom_key : undefined
      await keysAPI.create(
        formData.value.name,
        formData.value.group_id,
        customKey,
        ipWhitelist,
        ipBlacklist,
        quota,
        expiresInDays,
        rateLimitData
      )
      appStore.showSuccess(t('keys.keyCreatedSuccess'))
      // Only advance tour if active, on submit step, and creation succeeded
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500)
      }
    }
    closeModals()
    loadApiKeys()
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToSave')
    appStore.showError(errorMsg)
    // Don't advance tour on error
  } finally {
    submitting.value = false
  }
}

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return

  try {
    await keysAPI.delete(selectedKey.value.id)
    appStore.showSuccess(t('keys.keyDeletedSuccess'))
    showDeleteDialog.value = false
    loadApiKeys()
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t('keys.failedToDelete')
    appStore.showError(errorMsg)
  }
}

const closeModals = () => {
  showCreateModal.value = false
  showEditModal.value = false
  selectedKey.value = null
  formData.value = {
    name: '',
    group_id: null,
    status: 'active',
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: false,
    ip_whitelist: '',
    ip_blacklist: '',
    enable_quota: false,
    quota: null,
    enable_rate_limit: false,
    rate_limit_5h: null,
    rate_limit_1d: null,
    rate_limit_7d: null,
    enable_expiration: false,
    expiration_preset: '30',
    expiration_date: ''
  }
}

// Show reset quota confirmation dialog
const confirmResetQuota = () => {
  showResetQuotaDialog.value = true
}

// Set expiration date based on quick select days
const setExpirationDays = (days: number) => {
  formData.value.expiration_preset = days.toString() as '7' | '30' | '90'
  const expDate = new Date()
  expDate.setDate(expDate.getDate() + days)
  formData.value.expiration_date = formatDateTimeLocal(expDate.toISOString())
}

// Reset quota used for an API key
const resetQuotaUsed = async () => {
  if (!selectedKey.value) return
  showResetQuotaDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_quota: true })
    appStore.showSuccess(t('keys.quotaResetSuccess'))
    // Update local state
    if (selectedKey.value) {
      selectedKey.value.quota_used = 0
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetQuota')
    appStore.showError(errorMsg)
  }
}

// Show reset rate limit confirmation dialog (from edit modal)
const confirmResetRateLimit = () => {
  showResetRateLimitDialog.value = true
}

// Show reset rate limit confirmation dialog (from table row)
const confirmResetRateLimitFromTable = (row: ApiKey) => {
  showKeyDetailSheet.value = false
  selectedKey.value = row
  showResetRateLimitDialog.value = true
}

// Reset rate limit usage for an API key
const resetRateLimitUsage = async () => {
  if (!selectedKey.value) return
  showResetRateLimitDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_rate_limit_usage: true })
    appStore.showSuccess(t('keys.rateLimitResetSuccess'))
    // Refresh key data
    await loadApiKeys()
    // Update the editing key with fresh data
    const refreshedKey = apiKeys.value.find(k => k.id === selectedKey.value!.id)
    if (refreshedKey) {
      selectedKey.value = refreshedKey
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetRateLimit')
    appStore.showError(errorMsg)
  }
}

const importToCcswitch = (row: ApiKey) => {
  const platform = row.group?.platform || 'anthropic'

  // For antigravity platform, show client selection dialog
  if (platform === 'antigravity') {
    pendingCcsRow.value = row
    showCcsClientSelect.value = true
    return
  }

  // For other platforms, execute directly
  executeCcsImport(row, platform === 'gemini' ? 'gemini' : 'claude')
}

const executeCcsImport = (row: ApiKey, clientType: CcSwitchClientType) => {
  const baseUrl = publicSettings.value?.api_base_url || window.location.origin
  const platform = row.group?.platform || 'anthropic'

  const usageScript = `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`
  const providerName = (publicSettings.value?.site_name || '落雪API').trim() || '落雪API'
  const deeplink = buildCcSwitchImportDeeplink({
    baseUrl,
    platform,
    clientType,
    providerName,
    apiKey: row.key,
    usageScript
  })

  try {
    window.open(deeplink, '_self')

    // Check if the protocol handler worked by detecting if we're still focused
    setTimeout(() => {
      if (document.hasFocus()) {
        // Still focused means the protocol handler likely failed
        appStore.showError(t('keys.ccSwitchNotInstalled'))
      }
    }, 100)
  } catch (error) {
    appStore.showError(t('keys.ccSwitchNotInstalled'))
  }
}

const handleCcsClientSelect = (clientType: CcSwitchClientType) => {
  if (pendingCcsRow.value) {
    executeCcsImport(pendingCcsRow.value, clientType)
  }
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

const closeCcsClientSelect = () => {
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

onMounted(() => {
  loadSavedColumns()
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    inlineInspectorMediaQuery = window.matchMedia('(min-width: 768px)')
    hasInlineInspector.value = inlineInspectorMediaQuery.matches
    inlineInspectorMediaQuery.addEventListener('change', handleInlineInspectorChange)
  }
  loadApiKeys()
  loadGroups()
  loadUserGroupRates()
  loadPublicSettings()
  document.addEventListener('click', closeGroupSelector)
  document.addEventListener('keydown', handleEscapeKey)
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  abortController?.abort()
  document.removeEventListener('click', closeGroupSelector)
  document.removeEventListener('keydown', handleEscapeKey)
  inlineInspectorMediaQuery?.removeEventListener('change', handleInlineInspectorChange)
  inlineInspectorMediaQuery = null
  if (resetTimer) clearInterval(resetTimer)
})
</script>

<style scoped>
.keys-workspace {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  color: var(--workspace-text);
  background: var(--workspace-canvas);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0;
  -webkit-font-smoothing: antialiased;
}

.keys-mobile-header,
.keys-desktop-header,
.keys-filter-toolbar {
  flex: 0 0 auto;
  border-bottom: 1px solid var(--workspace-border);
  background: var(--workspace-card-surface);
}

.keys-mobile-header {
  display: flex;
  height: 56px;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
}

.keys-mobile-title,
.keys-mobile-actions,
.keys-desktop-title-wrap,
.keys-desktop-actions {
  display: flex;
  align-items: center;
}

.keys-mobile-title {
  gap: 8px;
  min-width: 0;
  color: var(--workspace-text);
  font-size: var(--workspace-type-brand-size);
  font-weight: var(--workspace-type-brand-weight);
  line-height: 24px;
}

.keys-mobile-title > :first-child {
  color: #7c3aed;
}

.keys-mobile-actions {
  gap: 8px;
}

.keys-mobile-refresh,
.keys-mobile-create,
.keys-desktop-refresh,
.keys-desktop-create,
.keys-detail-settings-button,
.keys-density-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  outline: none;
  transition: all 300ms cubic-bezier(0.4, 0, 0.2, 1);
}

.keys-mobile-refresh {
  width: 40px;
  height: 40px;
  color: var(--workspace-text-muted);
  border-radius: 12px;
}

.keys-mobile-refresh :deep(svg) {
  width: 18px;
  height: 18px;
}

.keys-mobile-create {
  height: 36px;
  gap: 6px;
  padding: 0 16px;
  border-radius: 12px;
  color: #fff;
  background: #7c3aed;
  box-shadow: 0 10px 15px -3px rgb(124 58 237 / 0.3), 0 4px 6px -4px rgb(124 58 237 / 0.3);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.keys-mobile-create :deep(svg) {
  width: 12px;
  height: 12px;
}

.keys-mobile-refresh:active,
.keys-mobile-create:active {
  transform: scale(0.95);
}

.keys-desktop-header {
  min-height: 89px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 32px;
}

.keys-desktop-title-wrap {
  min-width: 0;
  gap: 16px;
}

.keys-desktop-title-icon {
  display: grid;
  width: 48px;
  height: 48px;
  flex: 0 0 48px;
  place-items: center;
  border-radius: 16px;
  color: #7c3aed;
  background: #ede9fe;
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.05);
}

.keys-desktop-title-copy {
  min-width: 0;
}

.keys-desktop-title-copy h1 {
  margin: 0;
  color: var(--workspace-text);
  font-size: var(--workspace-type-page-title-size);
  font-weight: var(--workspace-type-page-title-weight);
  line-height: 28px;
}

.keys-desktop-title-copy p {
  overflow: hidden;
  margin: 0;
  color: var(--workspace-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 16px;
  letter-spacing: 0;
  text-overflow: ellipsis;
  text-transform: uppercase;
  white-space: nowrap;
}

.keys-desktop-actions {
  gap: 12px;
}

.keys-desktop-refresh {
  width: 48px;
  height: 48px;
  flex: 0 0 48px;
  border-radius: 12px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
}

.keys-desktop-refresh:hover {
  background: var(--workspace-hover);
}

.keys-desktop-create {
  height: 48px;
  gap: 8px;
  padding: 0 24px;
  border-radius: 12px;
  color: #fff;
  background: #7c3aed;
  box-shadow: 0 10px 15px -3px rgb(124 58 237 / 0.3), 0 4px 6px -4px rgb(124 58 237 / 0.3);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.keys-desktop-create:hover {
  transform: translateY(-2px);
}

.keys-desktop-create:active {
  transform: scale(0.95);
}

.keys-filter-toolbar {
  padding: 16px;
}

.keys-filter-grid {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 8px;
}

.key-filter-search,
.key-secondary-filter {
  min-width: 0;
}

.key-filter-search {
  margin-bottom: 4px;
}

.key-filter-search :deep(.input),
.key-secondary-filter :deep(.select-trigger) {
  width: 100%;
  min-height: 44px;
  border: 0;
  border-radius: 12px;
  color: var(--workspace-text);
  background: var(--workspace-surface-subtle);
  box-shadow: none;
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
}

.key-filter-search :deep(.input) {
  height: 44px;
  padding-left: 48px;
}

.key-filter-search :deep(> div > div) {
  padding-left: 16px;
}

.key-filter-search :deep(> div > div svg) {
  width: 16px;
  height: 16px;
}

.key-secondary-filter :deep(.select-trigger) {
  height: 44px;
  padding: 0 16px;
}

.key-secondary-filter :deep(.select-icon) {
  display: none;
}

.key-filter-search :deep(.input:focus),
.key-secondary-filter :deep(.select-trigger:focus-visible),
.key-secondary-filter :deep(.select-trigger-open) {
  border-color: transparent;
  outline: none;
  box-shadow: 0 0 0 2px rgb(139 92 246 / 0.2);
}

.keys-filter-actions {
  display: flex;
  min-width: 0;
  height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 4px;
}

.keys-detail-settings-button,
.keys-density-button {
  height: 40px;
  border-radius: 8px;
  font-weight: var(--workspace-type-navigation-weight);
}

.keys-detail-settings-button {
  gap: 8px;
  padding: 0 12px;
  color: #6d28d9;
  background: #f5f3ff;
  font-size: var(--workspace-type-navigation-size);
}

.keys-detail-settings-button :deep(svg) {
  width: 18px;
  height: 18px;
}

.keys-settings-icon--desktop {
  display: none;
}

.keys-density-button {
  padding: 0 12px;
  border: 1px solid var(--workspace-border);
  color: var(--workspace-text-muted);
  background: var(--workspace-surface-subtle);
  font-size: var(--workspace-type-navigation-size);
  letter-spacing: 0;
  text-transform: uppercase;
}

.keys-detail-settings-button:focus-visible,
.keys-density-button:focus-visible,
.keys-mobile-refresh:focus-visible,
.keys-mobile-create:focus-visible,
.keys-desktop-refresh:focus-visible,
.keys-desktop-create:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: 2px;
}

.keys-detail-menu {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 50;
  width: 224px;
  max-height: 320px;
  overflow-y: auto;
  margin-top: 8px;
  border: 1px solid var(--workspace-border);
  border-radius: 12px;
  padding: 8px 0;
  background: var(--workspace-popup-surface);
  box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);
}

.keys-detail-menu-item {
  display: flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  color: var(--workspace-text-secondary);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-align: left;
}

.keys-detail-menu-item:hover {
  background: var(--workspace-hover);
}

.keys-content {
  display: flex;
  min-height: 0;
  flex: 1 1 0;
  overflow: hidden;
  background: var(--workspace-canvas);
}

.keys-master-pane {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  background: var(--workspace-canvas);
}

.keys-detail-pane {
  width: 40%;
  min-width: 0;
  flex: 0 0 40%;
  overflow: hidden;
  border-left: 1px solid var(--workspace-border);
  background: var(--workspace-canvas);
}

.keys-mobile-list {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
  background: var(--workspace-canvas);
  scrollbar-color: var(--workspace-border-strong) transparent;
  scrollbar-width: thin;
}

.keys-mobile-list::-webkit-scrollbar {
  width: 5px;
}

.keys-mobile-list::-webkit-scrollbar-thumb {
  border-radius: 10px;
  background: var(--workspace-border-strong);
}

.keys-mobile-list > * + * {
  margin-top: 16px;
}

.keys-mobile-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 0 32px;
}

.keys-mobile-pagination p {
  margin: 0;
  color: var(--workspace-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  letter-spacing: 0;
  text-transform: uppercase;
}

.keys-mobile-pagination > div {
  display: flex;
  gap: 8px;
}

.keys-mobile-pagination button {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  border: 1px solid var(--workspace-border);
  border-radius: 12px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-card-surface);
}

.keys-mobile-pagination button:disabled {
  opacity: 0.4;
}

.keys-desktop-pagination {
  flex: 0 0 auto;
  overflow: hidden;
}

.keys-desktop-pagination :deep(.pagination) {
  min-height: 73px;
  border-top: 1px solid var(--workspace-border);
  padding: 16px 32px;
  background: var(--workspace-card-surface);
}

.keys-desktop-pagination :deep(.pagination-summary) {
  color: var(--workspace-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  letter-spacing: 0;
  text-transform: uppercase;
}

.keys-desktop-pagination :deep(.pagination-nav) {
  display: flex;
  gap: 8px;
  border-radius: 0;
  box-shadow: none;
}

.keys-desktop-pagination :deep(.pagination-nav .pagination-button) {
  display: inline-flex;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  margin: 0;
  border: 1px solid var(--workspace-border);
  border-radius: 12px;
  padding: 0;
  color: var(--workspace-text-muted);
  background: var(--workspace-card-surface);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.keys-desktop-pagination :deep(.pagination-nav .pagination-button--active) {
  border-color: #7c3aed;
  color: #fff;
  background: #7c3aed;
  box-shadow: 0 4px 6px -1px rgb(124 58 237 / 0.2);
}

.keys-empty-state {
  display: flex;
  min-height: 0;
  flex: 1 1 0;
  align-items: center;
  justify-content: center;
  overflow-y: auto;
  padding: 24px;
  background: var(--workspace-canvas);
}

.keys-loading-master {
  background: var(--workspace-surface-subtle);
}

.keys-loading-mobile {
  background: var(--workspace-canvas);
}

.keys-loading-detail {
  border-color: var(--workspace-border);
  background: var(--workspace-surface-subtle);
}

.keys-loading-card,
.keys-client-option {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
}

.keys-switch-track,
.keys-form-progress-track {
  background: var(--workspace-border-strong);
}

.keys-switch-thumb {
  background: var(--workspace-light-surface);
}

.keys-form-usage-surface,
.keys-neutral-choice {
  background: var(--workspace-surface-subtle);
}

.keys-neutral-choice {
  color: var(--workspace-text-secondary);
}

.keys-neutral-choice:hover {
  background: var(--workspace-hover);
}

.keys-text-primary {
  color: var(--workspace-text);
}

.keys-text-secondary {
  color: var(--workspace-text-secondary);
}

.keys-text-muted {
  color: var(--workspace-text-muted);
}

.keys-group-selector {
  border: 1px solid var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-popup-surface);
}

.keys-group-selector-header,
.keys-group-option {
  border-color: var(--workspace-border);
}

.keys-group-search {
  border-color: var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-surface-subtle);
}

.keys-group-search::placeholder {
  color: var(--workspace-text-muted);
}

.keys-group-option--selected {
  background: var(--workspace-selected);
}

.keys-group-option--idle:hover {
  background: var(--workspace-hover);
}

:global(.app-layout--snow-shell:has(.keys-workspace)) {
  --app-shell-top-offset: 0px;
}

:global(.app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat) {
  position: relative;
  z-index: 0;
  background: var(--workspace-canvas) !important;
  box-shadow: 0 25px 50px -12px rgb(0 0 0 / 0.25);
}

:global(.app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat .app-main-content) {
  padding: 0;
}

:global(.app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat .app-main-content > div) {
  max-width: none;
}

:global(.dark) .keys-workspace,
:global(.dark) .keys-mobile-header,
:global(.dark) .keys-desktop-header,
:global(.dark) .keys-filter-toolbar,
:global(.dark) .keys-content,
:global(.dark) .keys-master-pane {
  color: var(--workspace-text);
  background: var(--workspace-canvas);
}

:global(.dark .app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat) {
  background: var(--workspace-canvas) !important;
}

:global(.dark) .keys-mobile-header,
:global(.dark) .keys-desktop-header,
:global(.dark) .keys-filter-toolbar,
:global(.dark) .keys-detail-pane {
  border-color: var(--workspace-border);
}

:global(.dark) .keys-mobile-title,
:global(.dark) .keys-desktop-title-copy h1 {
  color: var(--workspace-text);
}

:global(.dark) .keys-desktop-title-icon {
  color: #a78bfa;
  background: rgb(76 29 149 / 0.3);
}

:global(.dark) .keys-desktop-refresh,
:global(.dark) .keys-filter-toolbar,
:global(.dark) .keys-mobile-header,
:global(.dark) .keys-desktop-header {
  background: var(--workspace-card-surface);
}

:global(.dark) .key-filter-search :deep(.input),
:global(.dark) .key-secondary-filter :deep(.select-trigger),
:global(.dark) .keys-density-button {
  border-color: var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-surface-subtle);
}

:global(.dark) .keys-detail-settings-button {
  color: #a78bfa;
  background: rgb(76 29 149 / 0.1);
}

:global(.dark) .keys-detail-menu {
  border-color: var(--workspace-border);
  background: var(--workspace-popup-surface);
}

:global(.dark) .keys-detail-menu-item {
  color: var(--workspace-text-secondary);
}

:global(.dark) .keys-detail-menu-item:hover {
  background: var(--workspace-hover);
}

:global(.dark) .keys-detail-pane {
  background: var(--workspace-canvas);
}

:global(.dark) .keys-mobile-list,
:global(.dark) .keys-empty-state {
  background: var(--workspace-canvas);
}

:global(.dark) .keys-mobile-pagination button,
:global(.dark) .keys-desktop-pagination :deep(.pagination),
:global(.dark) .keys-desktop-pagination :deep(.pagination-nav .pagination-button) {
  border-color: var(--workspace-border);
  color: var(--workspace-text-secondary);
  background: var(--workspace-card-surface);
}

@media (min-width: 768px) {
  .keys-settings-icon--mobile {
    display: none;
  }

  .keys-settings-icon--desktop {
    display: inline-flex;
  }

  .keys-filter-toolbar {
    padding: 16px 32px;
  }

  .keys-filter-grid {
    flex-direction: row;
    align-items: center;
    gap: 8px;
  }

  .key-filter-search {
    max-width: 512px;
    flex: 1 1 260px;
    margin: 0 8px 0 0;
  }

  .key-secondary-filter {
    width: 110px;
    flex: 0 0 110px;
  }

  .key-sort-filter {
    width: 152px;
    flex-basis: 152px;
  }

  .key-filter-search :deep(.input),
  .key-secondary-filter :deep(.select-trigger) {
    border: 1px solid var(--workspace-border);
  }

  .key-secondary-filter :deep(.select-trigger) {
    gap: 8px;
    padding-right: 12px;
    padding-left: 12px;
    background: var(--workspace-card-surface);
  }

  .key-secondary-filter :deep(.select-icon) {
    display: inline-flex;
  }

  .key-secondary-filter :deep(.select-icon svg) {
    width: 16px;
    height: 16px;
  }

  .keys-mobile-header {
    display: none;
  }

  .keys-filter-actions {
    width: 44px;
    height: 44px;
    flex: 0 0 44px;
    margin: 0;
  }

  .keys-detail-settings-button {
    width: 44px;
    height: 44px;
    padding: 0;
    border: 1px solid var(--workspace-border);
    border-radius: 12px;
    color: var(--workspace-text-secondary);
    background: var(--workspace-card-surface);
  }

  .keys-detail-settings-button:hover {
    background: var(--workspace-hover);
  }

  .keys-detail-settings-button span,
  .keys-density-button {
    display: none;
  }

  .keys-detail-menu {
    right: 0;
    left: auto;
  }

  .key-empty-create {
    display: none;
  }

  :global(.dark) .key-secondary-filter :deep(.select-trigger),
  :global(.dark) .keys-detail-settings-button {
    border-color: var(--workspace-border);
    color: var(--workspace-text-secondary);
    background: var(--workspace-surface-subtle);
  }
}

@media (max-width: 1023px) {
  :global(.app-layout--snow-shell:has(.keys-workspace) [data-testid='app-mobile-header']) {
    display: none !important;
  }

  :global(.app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat) {
    height: 100dvh;
  }
}

@media (prefers-reduced-motion: reduce) {
  .keys-mobile-refresh,
  .keys-mobile-create,
  .keys-desktop-refresh,
  .keys-desktop-create,
  .keys-detail-settings-button,
  .keys-density-button {
    transition: none;
  }
}
</style>
