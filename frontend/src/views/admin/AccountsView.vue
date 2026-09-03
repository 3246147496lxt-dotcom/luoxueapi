<template>
  <AppLayout variant="home-clay">
    <TablePageLayout class="account-table-page-layout" table-surface="plain">
      <template #header>
        <AdminPageHeader
          :title="t('admin.accounts.title')"
          :description="t('admin.accounts.description')"
        >
          <template #primary-actions>
            <button
              type="button"
              class="btn btn-primary min-h-11 gap-2 px-4"
              @click="showCreate = true"
            >
              <Icon name="plus" size="sm" aria-hidden="true" />
              <span>{{ t('admin.accounts.createAccount') }}</span>
            </button>
          </template>
        </AdminPageHeader>
      </template>

      <template #filters>
        <div class="account-filter-toolbar">
          <AccountTableFilters
            class="min-w-0 flex-1"
            v-model:searchQuery="params.search"
            :filters="params"
            :groups="groups"
            @update:filters="(newFilters) => Object.assign(params, newFilters)"
            @change="debouncedReload"
            @update:searchQuery="debouncedReload"
          />
          <AccountTableActions
            :loading="loading"
            @refresh="handleManualRefresh"
          >
            <template #after>
              <!-- Auto Refresh Dropdown -->
              <div class="relative" ref="autoRefreshDropdownRef">
                <button
                  type="button"
                  @click="
                    showAutoRefreshDropdown = !showAutoRefreshDropdown;
                    showAccountToolsDropdown = false;
                    showColumnDropdown = false
                  "
                  class="account-toolbar-icon-button"
                  :class="{ 'account-toolbar-icon-button--active': autoRefreshEnabled }"
                  data-test="account-auto-refresh-toggle"
                  :title="
                    autoRefreshEnabled
                      ? t('admin.accounts.autoRefreshCountdown', { seconds: autoRefreshCountdown })
                      : t('admin.accounts.autoRefresh')
                  "
                  :aria-label="
                    autoRefreshEnabled
                      ? t('admin.accounts.autoRefreshCountdown', { seconds: autoRefreshCountdown })
                      : t('admin.accounts.autoRefresh')
                  "
                  :aria-expanded="showAutoRefreshDropdown"
                  aria-controls="account-auto-refresh-menu"
                >
                  <Icon name="timer" size="sm" />
                </button>
                <div
                  v-if="showAutoRefreshDropdown"
                  id="account-auto-refresh-menu"
                  class="absolute left-0 z-50 mt-2 w-56 origin-top-left rounded-lg border border-gray-200 bg-white shadow-lg lg:left-auto lg:right-0 lg:origin-top-right dark:border-gray-700 dark:bg-gray-800"
                >
                  <div class="p-2">
                    <button
                      @click="setAutoRefreshEnabled(!autoRefreshEnabled)"
                      class="flex min-h-11 w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                    >
                      <span>{{ t('admin.accounts.enableAutoRefresh') }}</span>
                      <Icon v-if="autoRefreshEnabled" name="check" size="sm" class="text-primary-500" />
                    </button>
                    <div class="my-1 border-t border-gray-100 dark:border-gray-700"></div>
                    <button
                      v-for="sec in autoRefreshIntervals"
                      :key="sec"
                      @click="setAutoRefreshInterval(sec)"
                      class="flex min-h-11 w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                    >
                      <span>{{ autoRefreshIntervalLabel(sec) }}</span>
                      <Icon v-if="autoRefreshIntervalSeconds === sec" name="check" size="sm" class="text-primary-500" />
                    </button>
                  </div>
                </div>
              </div>

              <!-- Column Settings Dropdown -->
              <div class="relative" ref="columnDropdownRef">
                <button
                  type="button"
                  class="account-toolbar-icon-button"
                  data-test="account-columns-toggle"
                  :title="t('admin.accounts.viewColumns')"
                  :aria-label="t('admin.accounts.viewColumns')"
                  :aria-expanded="showColumnDropdown"
                  aria-controls="account-columns-menu"
                  @click="
                    showColumnDropdown = !showColumnDropdown;
                    showAutoRefreshDropdown = false;
                    showAccountToolsDropdown = false
                  "
                >
                  <Icon name="grid" size="sm" />
                </button>
                <div
                  v-if="showColumnDropdown"
                  id="account-columns-menu"
                  class="absolute right-0 z-50 mt-2 w-[min(18rem,calc(100vw-2rem))] origin-top-right overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-800"
                >
                  <div class="flex items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-gray-700">
                    <span class="text-sm font-semibold text-gray-800 dark:text-gray-100">
                      {{ t('admin.accounts.viewColumns') }}
                    </span>
                    <Icon name="grid" size="sm" class="text-gray-400" />
                  </div>
                  <div class="max-h-[60vh] overflow-y-auto p-2">
                    <button
                      v-for="col in toggleableColumns"
                      :key="col.key"
                      type="button"
                      class="flex min-h-11 w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 transition-colors hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                      @click="toggleColumn(col.key)"
                    >
                      <span class="truncate">{{ col.label }}</span>
                      <Icon v-if="isColumnVisible(col.key)" name="check" size="sm" class="text-primary-500" />
                    </button>
                  </div>
                </div>
              </div>

              <!-- More Tools Dropdown -->
              <div class="relative" ref="accountToolsDropdownRef">
                <button
                  type="button"
                  @click="
                    showAccountToolsDropdown = !showAccountToolsDropdown;
                    showAutoRefreshDropdown = false;
                    showColumnDropdown = false
                  "
                  class="account-toolbar-icon-button"
                  data-test="account-tools-toggle"
                  :title="t('admin.accounts.moreActions')"
                  :aria-label="t('admin.accounts.moreActions')"
                  :aria-expanded="showAccountToolsDropdown"
                  aria-controls="account-tools-menu"
                >
                  <Icon name="more" size="sm" />
                </button>
                <div
                  v-if="showAccountToolsDropdown"
                  id="account-tools-menu"
                  class="absolute right-0 z-50 mt-2 w-[min(20rem,calc(100vw-2rem))] origin-top-right overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-800"
                >
                  <div class="max-h-[70vh] overflow-y-auto p-2">
                    <div class="px-2 py-2">
                      <div class="text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
                        {{ t('admin.accounts.dataActions') }}
                      </div>
                    </div>
                    <button class="account-tools-menu-item" @click="openSyncFromCrs">
                      <span class="account-tools-menu-icon bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-300">
                        <Icon name="sync" size="sm" />
                      </span>
                      <span class="flex-1 text-left">{{ t('admin.accounts.syncFromCrs') }}</span>
                    </button>
                    <button class="account-tools-menu-item" @click="openImportData">
                      <span class="account-tools-menu-icon bg-emerald-50 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-300">
                        <Icon name="upload" size="sm" />
                      </span>
                      <span class="flex-1 text-left">{{ t('admin.accounts.dataImport') }}</span>
                    </button>
                    <button class="account-tools-menu-item" @click="openExportDataDialogFromMenu">
                      <span class="account-tools-menu-icon bg-violet-50 text-violet-600 dark:bg-violet-900/30 dark:text-violet-300">
                        <Icon name="download" size="sm" />
                      </span>
                      <span class="flex-1 text-left">
                        {{ selIds.length ? t('admin.accounts.dataExportSelected') : t('admin.accounts.dataExport') }}
                      </span>
                      <span
                        v-if="selIds.length"
                        class="rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/40 dark:text-primary-300"
                      >
                        {{ t('admin.accounts.selectedCount', { count: selIds.length }) }}
                      </span>
                    </button>
                    <button
                      class="account-tools-menu-item"
                      data-test="bulk-edit-filtered"
                      @click="openBulkEditFilteredFromMenu"
                    >
                      <span class="account-tools-menu-icon bg-fuchsia-50 text-fuchsia-600 dark:bg-fuchsia-900/30 dark:text-fuchsia-300">
                        <Icon name="edit" size="sm" />
                      </span>
                      <span class="flex-1 text-left">{{ t('admin.accounts.bulkEdit.title') }}</span>
                    </button>

                    <div class="my-2 border-t border-gray-100 dark:border-gray-700"></div>
                    <div class="px-2 py-2">
                      <div class="text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
                        {{ t('admin.accounts.toolActions') }}
                      </div>
                    </div>
                    <button class="account-tools-menu-item" @click="openErrorPassthrough">
                      <span class="account-tools-menu-icon bg-amber-50 text-amber-600 dark:bg-amber-900/30 dark:text-amber-300">
                        <Icon name="shield" size="sm" />
                      </span>
                      <span class="flex-1 text-left">{{ t('admin.errorPassthrough.title') }}</span>
                    </button>
                    <button class="account-tools-menu-item" @click="openTLSFingerprintProfiles">
                      <span class="account-tools-menu-icon bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-200">
                        <Icon name="lock" size="sm" />
                      </span>
                      <span class="flex-1 text-left">{{ t('admin.tlsFingerprintProfiles.title') }}</span>
                    </button>

                    <div class="my-2 border-t border-gray-100 dark:border-gray-700"></div>
                    <div class="space-y-2 px-3 py-2">
                      <div class="flex items-center justify-between gap-3">
                        <span class="text-sm font-medium text-gray-700 dark:text-gray-200">
                          {{ t('admin.accounts.upstreamBilling.autoProbeSettings') }}
                        </span>
                        <Toggle
                          v-model="upstreamBillingProbeSettings.enabled"
                          :aria-label="t('admin.accounts.upstreamBilling.autoProbeSettings')"
                        />
                      </div>
                      <div class="flex items-center gap-2">
                        <label class="flex-1 text-xs text-gray-500 dark:text-gray-400" for="upstream-billing-probe-interval">
                          {{ t('admin.accounts.upstreamBilling.intervalMinutes') }}
                        </label>
                        <input
                          id="upstream-billing-probe-interval"
                          v-model.number="upstreamBillingProbeSettings.interval_minutes"
                          type="number"
                          min="5"
                          max="1440"
                          class="input h-8 w-20 px-2 text-sm"
                        />
                        <button
                          type="button"
                          class="btn btn-secondary min-h-11 min-w-11 px-2"
                          :disabled="upstreamBillingSettingsLoading || upstreamBillingSettingsSaving"
                          :title="t('common.save')"
                          @click="saveUpstreamBillingProbeSettings"
                        >
                          <Icon name="check" size="sm" />
                        </button>
                      </div>
                    </div>

                  </div>
                </div>
              </div>
            </template>
          </AccountTableActions>
        </div>
        <div
          v-if="hasPendingListSync"
          class="mt-2 flex items-center justify-between rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200"
        >
          <span>{{ t('admin.accounts.listPendingSyncHint') }}</span>
          <button
            class="btn btn-secondary px-2 py-1 text-xs"
            @click="syncPendingListChanges"
          >
            {{ t('admin.accounts.listPendingSyncAction') }}
          </button>
        </div>
      </template>
      <template #table>
        <div class="account-workbench" @keydown.esc="closeAccountInspector">
          <section
            class="account-workbench__table-surface"
            :class="{ 'account-workbench__table-surface--expanded': hasExpandedColumns }"
            :aria-label="t('admin.accounts.title')"
          >
        <AccountBulkActionsBar
          v-if="selIds.length > 0"
          :selected-ids="selIds"
          @delete="handleBulkDelete"
          @reset-status="handleBulkResetStatus"
          @refresh-token="handleBulkRefreshToken"
          @probe-upstream-billing="handleBulkProbeUpstreamBilling"
          @edit-selected="openBulkEditSelected"
          @clear="clearSelection"
          @select-page="selectPage"
          @toggle-schedulable="handleBulkToggleSchedulable"
        />
        <div ref="accountTableRef" class="account-workbench__table-body flex min-h-0 flex-1 flex-col overflow-hidden">
        <DataTable
          ref="dataTableRef"
          :columns="cols"
          :data="accounts"
          :loading="loading"
          row-key="id"
          :server-side-sort="true"
          @sort="handleSort"
          default-sort-key="name"
          default-sort-order="asc"
          :sort-storage-key="ACCOUNT_SORT_STORAGE_KEY"
          :estimate-row-height="60"
          :overscan="5"
          :virtualize-threshold="50"
          mobile-primary-key="name"
          :mobile-visible-keys="ACCOUNT_MOBILE_VISIBLE_KEYS"
          :clickable-rows="true"
          :selected-row-key="selectedAccountId"
          :row-aria-label="accountRowAriaLabel"
          @rowClick="openAccountInspector"
        >
          <template #empty>
            <div class="flex flex-col items-center px-4 py-3 text-center">
              <span class="mb-3 inline-flex h-11 w-11 items-center justify-center rounded-xl bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                <Icon name="inbox" size="lg" />
              </span>
              <p class="font-medium text-gray-900 dark:text-white">{{ t('admin.accounts.noAccounts') }}</p>
              <p class="mt-1 max-w-sm text-sm text-gray-500 dark:text-dark-400">
                {{ t('admin.accounts.noAccountsDescription') }}
              </p>
            </div>
          </template>
          <template #header-select>
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="allVisibleSelected"
              @click.stop
              @change="toggleSelectAllVisible($event)"
            />
          </template>
          <template #cell-select="{ row }">
            <input
              type="checkbox"
              :checked="isSelected(row.id)"
              class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :aria-label="t('admin.accounts.workbench.selectAccountCheckbox', { name: row.name })"
              @click.stop
              @change="toggleSel(row.id)"
            />
          </template>
          <template #cell-id="{ value }">
            <span class="font-mono text-xs text-gray-500 dark:text-gray-400">#{{ value }}</span>
          </template>
          <template #cell-name="{ value }">
            <div class="account-identity-cell">
              <span class="account-identity-cell__name">{{ value }}</span>
            </div>
          </template>
          <template #cell-notes="{ value }">
            <span v-if="value" :title="value" class="block max-w-xs truncate text-sm text-gray-600 dark:text-gray-300">{{ value }}</span>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>
          <template #cell-platform_type="{ row }">
            <div class="flex min-w-0 flex-col gap-1">
              <div class="flex flex-wrap items-center gap-1">
                <PlatformTypeBadge :platform="row.platform" :type="row.type"
                  :auth-mode="getOpenAIAuthMode(row)"
                  :plan-type="getAccountPlanType(row)"
                  :privacy-mode="row.extra?.privacy_mode || row.parent_privacy_mode"
                  :subscription-expires-at="row.credentials?.subscription_expires_at || row.parent_subscription_expires_at" />
                <span
                  v-if="getAntigravityTierLabel(row)"
                  :class="['inline-block rounded px-1.5 py-0.5 text-[10px] font-medium', getAntigravityTierClass(row)]"
                >
                  {{ getAntigravityTierLabel(row) }}
                </span>
              </div>
              <div
                v-if="getOpenAICompactMeta(row)"
                :class="[
                  'inline-flex items-center gap-1.5 pl-0.5 text-[11px] font-medium leading-4',
                  getOpenAICompactMeta(row)?.className
                ]"
                :title="getOpenAICompactTitle(row)"
              >
                <span :class="['h-1.5 w-1.5 rounded-full', getOpenAICompactMeta(row)?.dotClass]" />
                <span>{{ getOpenAICompactMeta(row)?.label }}</span>
              </div>
            </div>
          </template>
          <template #cell-capacity="{ row }">
            <div @click.stop>
              <AccountCapacityCell :account="row" />
            </div>
          </template>
          <template #cell-status="{ row }">
            <div class="account-status-cell">
              <AccountStatusIndicator
                :account="row"
                compact
                @show-temp-unsched="handleShowTempUnsched"
              />
              <button
                type="button"
                class="account-status-cell__switch"
                role="switch"
                :aria-checked="row.schedulable"
                :aria-label="row.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled')"
                :disabled="togglingSchedulable === row.id"
                @click.stop="handleToggleSchedulable(row)"
              >
                <span />
              </button>
            </div>
          </template>
          <template #cell-schedulable="{ row }">
            <button @click.stop="handleToggleSchedulable(row)" :disabled="togglingSchedulable === row.id" class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-dark-800" :class="[row.schedulable ? 'bg-primary-500 hover:bg-primary-600' : 'bg-gray-200 hover:bg-gray-300 dark:bg-dark-600 dark:hover:bg-dark-500']" :title="row.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled')">
              <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out" :class="[row.schedulable ? 'translate-x-4' : 'translate-x-0']" />
            </button>
          </template>
          <template #cell-today_stats="{ row }">
            <AccountTodayStatsCell
              :stats="todayStatsByAccountId[String(row.id)] ?? null"
              :loading="todayStatsLoading"
              :error="todayStatsError"
            />
          </template>
          <template #cell-groups="{ row }">
            <div
              v-if="row.groups?.length"
              class="account-groups-summary"
              :title="accountGroupTitle(row)"
            >
              <GroupBadge
                class="account-groups-summary__badge"
                :name="row.groups[0].name"
                :platform="row.groups[0].platform"
                :subscription-type="row.groups[0].subscription_type"
                :rate-multiplier="row.groups[0].rate_multiplier"
                :show-rate="false"
              />
              <span v-if="row.groups.length > 1" class="account-groups-summary__more">
                +{{ row.groups.length - 1 }}
              </span>
            </div>
            <span v-else class="text-xs text-gray-400 dark:text-dark-500">-</span>
          </template>
          <template #header-usage="{ column }">
            <span>{{ column.label }}</span>
          </template>
          <template #cell-usage="{ row }">
            <div class="account-quota-capacity">
              <AccountUsageCell
                :account="row"
                :today-stats="todayStatsByAccountId[String(row.id)] ?? null"
                :today-stats-loading="todayStatsLoading"
                :manual-refresh-token="usageManualRefreshToken"
                display-mode="summary"
              />
              <span class="account-quota-capacity__capacity">
                {{
                  t('admin.accounts.workbench.concurrencySummary', {
                    current: row.current_concurrency || 0,
                    max: row.concurrency || 0
                  })
                }}
              </span>
            </div>
          </template>
          <template #cell-proxy="{ row }">
            <div class="flex flex-col gap-1" @click.stop>
              <div v-if="row.proxy" class="flex items-center gap-2">
                <span class="text-sm text-gray-700 dark:text-gray-300">{{ row.proxy.name }}</span>
                <span v-if="row.proxy.country_code" class="text-xs text-gray-500 dark:text-gray-400">
                  ({{ row.proxy.country_code }})
                </span>
              </div>
              <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
              <div v-if="row.proxy && row.proxy.expires_at" class="flex items-center gap-2 text-xs">
                <span class="text-gray-600 dark:text-gray-300">{{ formatDateTime(row.proxy.expires_at) }}</span>
                <span :class="proxyExpiryBadge(row.proxy)">{{ proxyExpiryText(row.proxy) }}</span>
              </div>
              <div v-if="row.proxy_fallback_origin_id" class="flex items-center gap-1">
                <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200" :title="t('admin.accounts.fallbackActiveTip', { origin: row.proxy_fallback_origin_name })">
                  {{ t('admin.accounts.fallbackActive') }}
                </span>
                <button class="text-xs px-1.5 py-0.5 rounded border border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700" @click.stop="onRevertFallback(row)">{{ t('admin.accounts.revertProxy') }}</button>
              </div>
            </div>
          </template>
          <template #cell-rate_multiplier="{ row }">
            <span class="text-sm font-mono text-gray-700 dark:text-gray-300">
              {{ (row.rate_multiplier ?? 1).toFixed(2) }}x
            </span>
          </template>
          <template #header-upstream_billing_rate="{ column }">
            <div class="flex items-center">
              <span>{{ column.label }}</span>
              <HelpTooltip :content="t('admin.accounts.upstreamBilling.trustWarning')" width-class="w-80" />
            </div>
          </template>
          <template #cell-upstream_billing_rate="{ row }">
            <div @click.stop>
              <UpstreamBillingRateCell
                :account="row"
                :interval-minutes="upstreamBillingProbeSettings.interval_minutes"
                :now="upstreamBillingNow"
                :probing="probingUpstreamBilling.has(row.id)"
                @probe="handleProbeUpstreamBilling(row)"
              />
            </div>
          </template>
          <template #cell-priority="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ value }}</span>
          </template>
          <template #header-scheduler_score="{ column }">
            <div class="flex items-center">
              <span>{{ column.label }}</span>
              <HelpTooltip :content="t('admin.accounts.schedulerScore.hint')" width-class="w-80" />
            </div>
          </template>
          <template #cell-scheduler_score="{ row }">
            <div v-if="getSchedulerScoreRows(row).length" class="flex min-w-[7rem] flex-col gap-0.5 font-mono text-[11px] leading-4">
              <div
                v-for="score in getSchedulerScoreRows(row)"
                :key="String(score.group_id)"
                class="flex items-center gap-1 whitespace-nowrap text-gray-700 dark:text-gray-300"
                :title="`${formatSchedulerScoreGroup(score)} / ${formatSchedulerScore(score.base_score)} / ${formatStickySchedulerScore(score)}`"
              >
                <span class="max-w-[4.75rem] truncate text-gray-500 dark:text-dark-400">{{ formatSchedulerScoreGroup(score) }}</span>
                <span class="text-gray-300 dark:text-gray-600">/</span>
                <span>{{ formatSchedulerScore(score.base_score) }}</span>
                <span class="text-gray-300 dark:text-gray-600">/</span>
                <span class="text-primary-700 dark:text-primary-300">{{ formatStickySchedulerScore(score) }}</span>
              </div>
            </div>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>
          <template #cell-last_used_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatRelativeTime(value) }}</span>
          </template>
          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>
          <template #cell-expires_at="{ row, value }">
            <div class="flex flex-col items-start gap-1">
              <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatExpiresAt(value) }}</span>
              <div v-if="isExpired(value) || (row.auto_pause_on_expired && value)" class="flex items-center gap-1">
                <span
                  v-if="isExpired(value)"
                  class="inline-flex items-center rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
                >
                  {{ t('admin.accounts.expired') }}
                </span>
                <span
                  v-if="row.auto_pause_on_expired && value"
                  class="inline-flex items-center rounded-md bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                >
                  {{ t('admin.accounts.autoPauseOnExpired') }}
                </span>
              </div>
            </div>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex w-full items-center justify-end" @click.stop>
              <button
                type="button"
                class="account-row-more-button"
                :aria-label="t('common.more')"
                :title="t('common.more')"
                @click.stop="openMenu(row, $event)"
              >
                <Icon name="more" size="sm" class="rotate-90" aria-hidden="true" />
              </button>
            </div>
          </template>
        </DataTable>
        </div>
            <div class="account-workbench__pagination">
              <Pagination
                v-if="pagination.total > 0"
                :page="pagination.page"
                :total="pagination.total"
                :page-size="pagination.page_size"
                @update:page="handlePageChange"
                @update:pageSize="handlePageSizeChange"
              />
            </div>
          </section>

          <aside v-if="isWideInspector" class="account-workbench__inspector-surface">
            <AccountInspector
              :account="selectedAccount"
              :proxy-telemetry="selectedProxyTelemetry"
              :today-stats="selectedTodayStats"
              :today-stats-loading="todayStatsLoading"
              :today-stats-error="todayStatsError"
              :usage-refresh-token="usageManualRefreshToken"
              :toggling-schedulable="togglingSchedulable === selectedAccount?.id"
              mode="inline"
              @close="closeAccountInspector"
              @test="handleTest"
              @stats="handleViewStats"
              @edit="handleEdit"
              @more="openMenu"
              @revert-fallback="onRevertFallback"
              @toggle-schedulable="handleToggleSchedulable"
              @view-proxy="handleViewProxy"
            />
          </aside>
        </div>
      </template>
    </TablePageLayout>

    <Teleport to="body">
      <Transition name="account-inspector-overlay">
        <div
          v-if="selectedAccount && !isWideInspector"
          class="account-inspector-overlay"
          :class="`account-inspector-overlay--${overlayInspectorMode}`"
          data-testid="account-inspector-overlay"
          @click.self="closeAccountInspector"
        >
          <AccountInspector
            ref="overlayInspectorRef"
            :account="selectedAccount"
            :proxy-telemetry="selectedProxyTelemetry"
            :today-stats="selectedTodayStats"
            :today-stats-loading="todayStatsLoading"
            :today-stats-error="todayStatsError"
            :usage-refresh-token="usageManualRefreshToken"
            :toggling-schedulable="togglingSchedulable === selectedAccount.id"
            :mode="overlayInspectorMode"
            @close="closeAccountInspector"
            @test="handleTest"
            @stats="handleViewStats"
            @edit="handleEdit"
            @more="openMenu"
            @revert-fallback="onRevertFallback"
            @toggle-schedulable="handleToggleSchedulable"
            @view-proxy="handleViewProxy"
          />
        </div>
      </Transition>
    </Teleport>
    <CreateAccountModal :show="showCreate" :proxies="proxies" :groups="groups" @close="showCreate = false" @created="reload" />
    <EditAccountModal :show="showEdit" :account="edAcc" :proxies="proxies" :groups="groups" @close="showEdit = false" @updated="handleAccountUpdated" />
    <ReAuthAccountModal :show="showReAuth" :account="reAuthAcc" @close="closeReAuthModal" @reauthorized="handleAccountUpdated" />
    <AccountTestModal
      :show="showTest"
      :account="testingAcc"
      @close="closeTestModal"
      @completed="handleTestCompleted"
    />
    <AccountStatsModal :show="showStats" :account="statsAcc" @close="closeStatsModal" />
    <ScheduledTestsPanel :show="showSchedulePanel" :account-id="scheduleAcc?.id ?? null" :model-options="scheduleModelOptions" @close="closeSchedulePanel" />
    <AccountActionMenu :show="menu.show" :account="menu.acc" :position="menu.pos" @close="menu.show = false" @test="handleTest" @stats="handleViewStats" @schedule="handleSchedule" @duplicate="handleDuplicateAccount" @reauth="handleReAuth" @refresh-token="handleRefresh" @recover-state="handleRecoverState" @reset-quota="handleResetQuota" @set-privacy="handleSetPrivacy" @create-spark-shadow="handleCreateSparkShadow" @delete="handleDelete" />
    <SyncFromCrsModal :show="showSync" @close="showSync = false" @synced="reload" />
    <ImportDataModal :show="showImportData" @close="showImportData = false" @imported="handleDataImported" />
    <BulkEditAccountModal
      :show="showBulkEdit"
      :account-ids="selIds"
      :selected-platforms="selPlatforms"
      :selected-types="selTypes"
      :target="bulkEditTarget ?? undefined"
      :proxies="proxies"
      :groups="groups"
      @close="showBulkEdit = false"
      @updated="handleBulkUpdated"
    />
    <TempUnschedStatusModal :show="showTempUnsched" :account="tempUnschedAcc" @close="showTempUnsched = false" @reset="handleTempUnschedReset" />
    <ConfirmDialog :show="showDeleteDialog" :title="t('admin.accounts.deleteAccount')" :message="t('admin.accounts.deleteConfirm', { name: deletingAcc?.name })" :confirm-text="t('common.delete')" :cancel-text="t('common.cancel')" :danger="true" @confirm="confirmDelete" @cancel="showDeleteDialog = false" />
    <ConfirmDialog :show="showCreateShadowDialog" :title="t('admin.accounts.createSparkShadow')" :message="t('admin.accounts.createSparkShadowConfirm', { name: creatingShadowAcc?.name })" @confirm="confirmCreateSparkShadow" @cancel="showCreateShadowDialog = false" />
    <ConfirmDialog :show="showExportDataDialog" :title="t('admin.accounts.dataExport')" :message="t('admin.accounts.dataExportConfirmMessage')" :confirm-text="t('admin.accounts.dataExportConfirm')" :cancel-text="t('common.cancel')" @confirm="handleExportData" @cancel="showExportDataDialog = false">
      <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
        <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" v-model="includeProxyOnExport" />
        <span>{{ t('admin.accounts.dataExportIncludeProxies') }}</span>
      </label>
    </ConfirmDialog>
    <ErrorPassthroughRulesModal :show="showErrorPassthrough" @close="showErrorPassthrough = false" />
    <TLSFingerprintProfilesModal :show="showTLSFingerprintProfiles" @close="showTLSFingerprintProfiles = false" />
    <TotpStepUpDialog :controller="accountExportStepUp" />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, nextTick, onMounted, onUnmounted, toRaw, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { adminAPI } from '@/api/admin'
import { useTableLoader } from '@/composables/useTableLoader'
import { useSwipeSelect, type SwipeSelectVirtualContext } from '@/composables/useSwipeSelect'
import { useTableSelection } from '@/composables/useTableSelection'
import { useStepUp, isStepUpBlocked, isStepUpCancelled, stepUpBlockReason } from '@/composables/useStepUp'
import {
  invalidateAccountUsageHealthSnapshot,
  requestAccountUsage
} from '@/composables/useAccountUsageHealth'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Toggle from '@/components/common/Toggle.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { CreateAccountModal, EditAccountModal, BulkEditAccountModal, SyncFromCrsModal, TempUnschedStatusModal } from '@/components/account'
import AccountTableActions from '@/components/admin/account/AccountTableActions.vue'
import AccountTableFilters from '@/components/admin/account/AccountTableFilters.vue'
import AccountBulkActionsBar from '@/components/admin/account/AccountBulkActionsBar.vue'
import AccountActionMenu from '@/components/admin/account/AccountActionMenu.vue'
import AccountInspector from '@/components/admin/account/AccountInspector.vue'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'
import ReAuthAccountModal from '@/components/admin/account/ReAuthAccountModal.vue'
import AccountTestModal from '@/components/admin/account/AccountTestModal.vue'
import AccountStatsModal from '@/components/admin/account/AccountStatsModal.vue'
import ScheduledTestsPanel from '@/components/admin/account/ScheduledTestsPanel.vue'
import type { SelectOption } from '@/components/common/Select.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import AccountUsageCell from '@/components/account/AccountUsageCell.vue'
import AccountTodayStatsCell from '@/components/account/AccountTodayStatsCell.vue'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import UpstreamBillingRateCell from '@/components/account/UpstreamBillingRateCell.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import ErrorPassthroughRulesModal from '@/components/admin/ErrorPassthroughRulesModal.vue'
import TLSFingerprintProfilesModal from '@/components/admin/TLSFingerprintProfilesModal.vue'
import { buildOpenAIUsageRefreshKey } from '@/utils/accountUsageRefresh'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import { proxyExpiryBadgeClass, proxyExpiryLabelKey } from '@/utils/proxyExpiry'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Account, AccountPlatform, AccountSchedulerGroupScore, AccountType, Proxy as AccountProxy, AdminGroup, WindowStats, ClaudeModel, UpstreamBillingProbeSettings, UpstreamBillingProbeSnapshot } from '@/types'

const route = useRoute()
const router = useRouter()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const proxies = ref<AccountProxy[]>([])
const groups = ref<AdminGroup[]>([])
const accountTableRef = ref<HTMLElement | null>(null)
const dataTableRef = ref<InstanceType<typeof DataTable> | null>(null)
const overlayInspectorRef = ref<InstanceType<typeof AccountInspector> | null>(null)
const selectedAccountId = ref<number | null>(null)
const viewportWidth = ref(typeof window === 'undefined' ? 1440 : window.innerWidth)
const isWideInspector = computed(() => viewportWidth.value >= 1280)
const overlayInspectorMode = computed<'drawer' | 'sheet'>(() =>
  viewportWidth.value >= 1024 ? 'drawer' : 'sheet'
)
let inspectorTriggerElement: HTMLElement | null = null
let previousBodyOverflow = ''
let inspectorBodyLocked = false
type AccountBulkEditTarget =
  | {
      mode: 'selected'
      accountIds: number[]
      selectedPlatforms: AccountPlatform[]
      selectedTypes: AccountType[]
    }
  | {
      mode: 'filtered'
      filters: {
        platform?: string
        type?: string
        status?: string
        group?: string
        search?: string
        privacy_mode?: string
        sort_by?: string
        sort_order?: AccountSortOrder
      }
      previewCount: number
      selectedPlatforms: AccountPlatform[]
      selectedTypes: AccountType[]
    }
const selPlatforms = computed<AccountPlatform[]>(() => {
  const platforms = new Set(
    accounts.value
      .filter(a => isSelected(a.id))
      .map(a => a.platform)
  )
  return [...platforms]
})
const selTypes = computed<AccountType[]>(() => {
  const types = new Set(
    accounts.value
      .filter(a => isSelected(a.id))
      .map(a => a.type)
  )
  return [...types]
})
const showCreate = ref(false)
const showEdit = ref(false)
const showSync = ref(false)
const showImportData = ref(false)
const showExportDataDialog = ref(false)
const includeProxyOnExport = ref(true)
const showBulkEdit = ref(false)
const bulkEditTarget = ref<AccountBulkEditTarget | null>(null)
const showTempUnsched = ref(false)
const showDeleteDialog = ref(false)
const showCreateShadowDialog = ref(false)
const showReAuth = ref(false)
const showTest = ref(false)
const showStats = ref(false)
const showErrorPassthrough = ref(false)
const showTLSFingerprintProfiles = ref(false)
const edAcc = ref<Account | null>(null)
const tempUnschedAcc = ref<Account | null>(null)
const deletingAcc = ref<Account | null>(null)
const creatingShadowAcc = ref<Account | null>(null)
const reAuthAcc = ref<Account | null>(null)
const testingAcc = ref<Account | null>(null)
const statsAcc = ref<Account | null>(null)
const showSchedulePanel = ref(false)
const scheduleAcc = ref<Account | null>(null)
const scheduleModelOptions = ref<SelectOption[]>([])
const togglingSchedulable = ref<number | null>(null)
const menu = reactive<{show:boolean, acc:Account|null, pos:{top:number, left:number}|null}>({ show: false, acc: null, pos: null })
const exportingData = ref(false)
const upstreamBillingProbeSettings = reactive<UpstreamBillingProbeSettings>({
  enabled: true,
  interval_minutes: 30
})
const upstreamBillingSettingsLoading = ref(false)
const upstreamBillingSettingsSaving = ref(false)
const probingUpstreamBilling = reactive(new Set<number>())
const upstreamBillingNow = ref(Date.now())
useIntervalFn(() => { upstreamBillingNow.value = Date.now() }, 60_000)

// Account tools dropdown
const showAccountToolsDropdown = ref(false)
const accountToolsDropdownRef = ref<HTMLElement | null>(null)
const showColumnDropdown = ref(false)
const columnDropdownRef = ref<HTMLElement | null>(null)
const hiddenColumns = reactive<Set<string>>(new Set())
const DEFAULT_HIDDEN_COLUMNS = [
  'id',
  'platform_type',
  'capacity',
  'schedulable',
  'today_stats',
  'proxy',
  'notes',
  'priority',
  'scheduler_score',
  'rate_multiplier',
  'upstream_billing_rate',
  'created_at',
  'expires_at'
]
const HIDDEN_COLUMNS_KEY = 'account-hidden-columns'
// One-time migration keeps existing admins on the compact workbench column set.
const HIDDEN_COLUMNS_VERSION_KEY = 'account-hidden-columns-version'
const HIDDEN_COLUMNS_CURRENT_VERSION = 'account-workbench-v2'
const COMPACT_WORKBENCH_COLUMN_KEYS = new Set([
  'select',
  'name',
  'usage',
  'status',
  'groups',
  'last_used_at',
  'actions'
])

// Sorting settings
const ACCOUNT_SORT_STORAGE_KEY = 'account-table-sort'
const ACCOUNT_MOBILE_VISIBLE_KEYS: string[] = ['name', 'usage', 'status', 'groups', 'last_used_at']
type AccountSortOrder = 'asc' | 'desc'
type AccountSortState = {
  sort_by: string
  sort_order: AccountSortOrder
}
const ACCOUNT_SORTABLE_KEYS = new Set([
  'id',
  'name',
  'status',
  'schedulable',
  'priority',
  'rate_multiplier',
  'last_used_at',
  'created_at',
  'expires_at'
])
// The combined status column renders effective health, not the persisted account.status value.
const accountSortRequestKey = (key: string) => key === 'status' ? 'effective_status' : key
const loadInitialAccountSortState = (): AccountSortState => {
  const fallback: AccountSortState = { sort_by: 'name', sort_order: 'asc' }
  try {
    const raw = localStorage.getItem(ACCOUNT_SORT_STORAGE_KEY)
    if (!raw) return fallback
    const parsed = JSON.parse(raw) as { key?: string; order?: string }
    const key = typeof parsed.key === 'string' ? parsed.key : ''
    if (!ACCOUNT_SORTABLE_KEYS.has(key)) return fallback
    return {
      sort_by: key,
      sort_order: parsed.order === 'desc' ? 'desc' : 'asc'
    }
  } catch {
    return fallback
  }
}
const sortState = reactive<AccountSortState>(loadInitialAccountSortState())

// Auto refresh settings
const showAutoRefreshDropdown = ref(false)
const autoRefreshDropdownRef = ref<HTMLElement | null>(null)
const AUTO_REFRESH_STORAGE_KEY = 'account-auto-refresh'
const autoRefreshIntervals = [5, 10, 15, 30] as const
const autoRefreshEnabled = ref(false)
const autoRefreshIntervalSeconds = ref<(typeof autoRefreshIntervals)[number]>(30)
const autoRefreshCountdown = ref(0)
const autoRefreshETag = ref<string | null>(null)
const autoRefreshFetching = ref(false)
const AUTO_REFRESH_SILENT_WINDOW_MS = 15000
const autoRefreshSilentUntil = ref(0)
const hasPendingListSync = ref(false)
const todayStatsByAccountId = ref<Record<string, WindowStats>>({})
const todayStatsLoading = ref(false)
const todayStatsError = ref<string | null>(null)
const todayStatsReqSeq = ref(0)
const pendingTodayStatsRefresh = ref(false)
const usageManualRefreshToken = ref(0)

const buildDefaultTodayStats = (): WindowStats => ({
  requests: 0,
  tokens: 0,
  cost: 0,
  standard_cost: 0,
  user_cost: 0
})

const refreshTodayStatsBatch = async () => {
  // Why this checks both columns:
  // - today_stats column shows dedicated today's metrics.
  // - usage column also embeds today's stats for Key/Bedrock rows.
  // So we only skip fetching when BOTH columns are hidden.
  if (hiddenColumns.has('today_stats') && hiddenColumns.has('usage')) {
    todayStatsLoading.value = false
    todayStatsError.value = null
    return
  }

  const accountIDs = accounts.value.map(account => account.id)
  const reqSeq = ++todayStatsReqSeq.value
  if (accountIDs.length === 0) {
    todayStatsByAccountId.value = {}
    todayStatsError.value = null
    todayStatsLoading.value = false
    return
  }

  todayStatsLoading.value = true
  todayStatsError.value = null

  try {
    const result = await adminAPI.accounts.getBatchTodayStats(accountIDs)
    if (reqSeq !== todayStatsReqSeq.value) return
    const serverStats = result.stats ?? {}
    const nextStats: Record<string, WindowStats> = {}
    for (const accountID of accountIDs) {
      const key = String(accountID)
      nextStats[key] = serverStats[key] ?? buildDefaultTodayStats()
    }
    todayStatsByAccountId.value = nextStats
  } catch (error) {
    if (reqSeq !== todayStatsReqSeq.value) return
    todayStatsError.value = 'Failed'
    console.error('Failed to load account today stats:', error)
  } finally {
    if (reqSeq === todayStatsReqSeq.value) {
      todayStatsLoading.value = false
    }
  }
}

const autoRefreshIntervalLabel = (sec: number) => {
  if (sec === 5) return t('admin.accounts.refreshInterval5s')
  if (sec === 10) return t('admin.accounts.refreshInterval10s')
  if (sec === 15) return t('admin.accounts.refreshInterval15s')
  if (sec === 30) return t('admin.accounts.refreshInterval30s')
  return `${sec}s`
}

const formatSchedulerScore = (value: unknown): string => {
  const num = Number(value)
  if (!Number.isFinite(num)) return '-'
  return num.toFixed(6).replace(/\.?0+$/, '')
}

const formatStickySchedulerScore = (score: AccountSchedulerGroupScore): string => {
  if (!score) return '-'
  if (score.sticky_score_infinity) return '+∞'
  return formatSchedulerScore(score.sticky_score)
}

const getSchedulerScoreRows = (account: Account): AccountSchedulerGroupScore[] => {
  const groupRows = Array.isArray(account.scheduler_scores)
    ? account.scheduler_scores.filter(score => score.group_id != null)
    : []
  if (groupRows.length) return groupRows
  // 未分组账号没有分组维度分数，回退展示后端返回的基础分
  if (account.scheduler_score) {
    return [{ group_id: null, ...account.scheduler_score }]
  }
  return []
}

const formatSchedulerScoreGroup = (score: AccountSchedulerGroupScore): string => {
  if ('group_name' in score && score.group_name) return score.group_name
  if ('group_id' in score && score.group_id != null) return `#${score.group_id}`
  return t('admin.accounts.schedulerScore.ungrouped')
}

const loadSavedColumns = () => {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      parsed.forEach(key => {
        hiddenColumns.add(key)
      })
      // Older saved layouts can recreate the former wide table. Apply the
      // approved compact defaults once while preserving any extra hidden fields.
      if (localStorage.getItem(HIDDEN_COLUMNS_VERSION_KEY) !== HIDDEN_COLUMNS_CURRENT_VERSION) {
        DEFAULT_HIDDEN_COLUMNS.forEach(key => hiddenColumns.add(key))
        ;['usage', 'status', 'groups', 'last_used_at'].forEach(key => hiddenColumns.delete(key))
        localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
        localStorage.setItem(HIDDEN_COLUMNS_VERSION_KEY, HIDDEN_COLUMNS_CURRENT_VERSION)
      }
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach(key => {
        hiddenColumns.add(key)
      })
      localStorage.setItem(HIDDEN_COLUMNS_VERSION_KEY, HIDDEN_COLUMNS_CURRENT_VERSION)
    }
  } catch (e) {
    console.error('Failed to load saved columns:', e)
    DEFAULT_HIDDEN_COLUMNS.forEach(key => {
      hiddenColumns.add(key)
    })
  }
}

const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
    localStorage.setItem(HIDDEN_COLUMNS_VERSION_KEY, HIDDEN_COLUMNS_CURRENT_VERSION)
  } catch (e) {
    console.error('Failed to save columns:', e)
  }
}

const loadSavedAutoRefresh = () => {
  try {
    const saved = localStorage.getItem(AUTO_REFRESH_STORAGE_KEY)
    if (!saved) return
    const parsed = JSON.parse(saved) as { enabled?: boolean; interval_seconds?: number }
    autoRefreshEnabled.value = parsed.enabled === true
    const interval = Number(parsed.interval_seconds)
    if (autoRefreshIntervals.includes(interval as any)) {
      autoRefreshIntervalSeconds.value = interval as any
    }
  } catch (e) {
    console.error('Failed to load saved auto refresh settings:', e)
  }
}

const saveAutoRefreshToStorage = () => {
  try {
    localStorage.setItem(
      AUTO_REFRESH_STORAGE_KEY,
      JSON.stringify({
        enabled: autoRefreshEnabled.value,
        interval_seconds: autoRefreshIntervalSeconds.value
      })
    )
  } catch (e) {
    console.error('Failed to save auto refresh settings:', e)
  }
}

if (typeof window !== 'undefined') {
  loadSavedColumns()
  loadSavedAutoRefresh()
}

const setAutoRefreshEnabled = (enabled: boolean) => {
  autoRefreshEnabled.value = enabled
  saveAutoRefreshToStorage()
  if (enabled) {
    autoRefreshCountdown.value = autoRefreshIntervalSeconds.value
    resumeAutoRefresh()
  } else {
    pauseAutoRefresh()
    autoRefreshCountdown.value = 0
  }
}

const setAutoRefreshInterval = (seconds: (typeof autoRefreshIntervals)[number]) => {
  autoRefreshIntervalSeconds.value = seconds
  saveAutoRefreshToStorage()
  if (autoRefreshEnabled.value) {
    autoRefreshCountdown.value = seconds
  }
}

const toggleColumn = (key: string) => {
  const wasHidden = hiddenColumns.has(key)
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
  if ((key === 'today_stats' || key === 'usage') && wasHidden) {
    refreshTodayStatsBatch().catch((error) => {
      console.error('Failed to load account today stats after showing column:', error)
    })
  }
  if (key === 'scheduler_score') {
    // The server only returns scheduler scores when this column is visible, so reload the current page immediately.
    syncAccountListDerivedParams()
    load().catch((error) => {
      console.error('Failed to reload accounts after toggling scheduler score column:', error)
    })
  }
}

const isColumnVisible = (key: string) => !hiddenColumns.has(key)
const shouldIncludeSchedulerScore = () => isColumnVisible('scheduler_score')
const syncAccountListDerivedParams = () => {
  // Keep every load path, including auto-refresh and sorting, aligned with the current column visibility.
  const requestParams = params as any
  requestParams.include_scheduler_score = shouldIncludeSchedulerScore() ? '1' : '0'
}

const {
  items: accounts,
  loading,
  params,
  pagination,
  load: baseLoad,
  reload: baseReload,
  debouncedReload: baseDebouncedReload,
  handlePageChange: baseHandlePageChange,
  handlePageSizeChange: baseHandlePageSizeChange
} = useTableLoader<Account, any>({
  fetchFn: adminAPI.accounts.list,
  initialParams: {
    platform: '',
    type: '',
    status: '',
    privacy_mode: '',
    group: '',
    search: '',
    include_scheduler_score: shouldIncludeSchedulerScore() ? '1' : '0',
    sort_by: accountSortRequestKey(sortState.sort_by),
    sort_order: sortState.sort_order
  }
})

const proxyTelemetryById = computed(() =>
  new Map(proxies.value.map(proxy => [proxy.id, proxy]))
)
const selectedAccount = computed(() =>
  accounts.value.find(account => account.id === selectedAccountId.value) ?? null
)
const selectedProxyTelemetry = computed(() => {
  const proxyId = selectedAccount.value?.proxy_id
  if (proxyId == null) return null
  const proxy = proxyTelemetryById.value.get(proxyId)
  if (!proxy) return null
  return {
    id: proxy.id,
    name: proxy.name,
    status: proxy.status
  }
})
const selectedTodayStats = computed(() => {
  const accountId = selectedAccount.value?.id
  if (accountId == null) return null
  return todayStatsByAccountId.value[String(accountId)] ?? null
})

const allowedAccountPlatforms = new Set(['anthropic', 'openai', 'gemini', 'antigravity', 'grok'])
const allowedAccountHealthFilters = new Set([
  'active',
  'inactive',
  'error',
  'rate_limited',
  'temp_unschedulable',
  'unschedulable',
  'overloaded',
  'expired',
  'quota_exhausted'
])
let syncingAccountRoute = false
let accountRouteReady = false

const accountRouteString = (key: string): string => {
  const value = route.query[key]
  if (typeof value === 'string') return value.trim()
  if (Array.isArray(value) && typeof value[0] === 'string') return value[0].trim()
  return ''
}

const positiveRouteID = (key: string): number | null => {
  const raw = accountRouteString(key)
  if (!raw) return null
  const value = Number.parseInt(raw, 10)
  return Number.isFinite(value) && value > 0 ? value : null
}

const applyAccountRouteFilters = async () => {
  const nextQuery = { ...route.query }
  let normalized = false

  const platformQuery = accountRouteString('platform')
  if (platformQuery && !allowedAccountPlatforms.has(platformQuery)) {
    delete nextQuery.platform
    normalized = true
  }
  params.platform = allowedAccountPlatforms.has(platformQuery) ? platformQuery : ''

  const healthQuery = accountRouteString('health')
  if (healthQuery && !allowedAccountHealthFilters.has(healthQuery)) {
    delete nextQuery.health
    normalized = true
  }
  params.status = allowedAccountHealthFilters.has(healthQuery) ? healthQuery : ''

  const groupQuery = accountRouteString('group')
  const validGroup = groupQuery === 'ungrouped' || (Number.isFinite(Number(groupQuery)) && Number(groupQuery) > 0)
  if (groupQuery && !validGroup) {
    delete nextQuery.group
    normalized = true
  }
  params.group = validGroup ? groupQuery : ''

  const accountID = positiveRouteID('account_id')
  const proxyID = positiveRouteID('proxy_id')
  if (accountRouteString('account_id') && accountID === null) {
    delete nextQuery.account_id
    normalized = true
  }
  if (accountRouteString('proxy_id') && proxyID === null) {
    delete nextQuery.proxy_id
    normalized = true
  }

  if (accountID !== null) {
    params.search = `#${accountID}`
  } else if (proxyID !== null) {
    params.search = `proxy:${proxyID}`
  } else {
    params.search = accountRouteString('search')
  }

  if (normalized) {
    syncingAccountRoute = true
    try {
      await router.replace({ query: nextQuery })
    } finally {
      syncingAccountRoute = false
    }
  }
}

const syncAccountFiltersToRoute = async () => {
  if (syncingAccountRoute) return
  const nextQuery: Record<string, any> = { ...route.query }
  for (const key of ['platform', 'health', 'group', 'proxy_id', 'account_id', 'search']) delete nextQuery[key]

  if (params.platform) nextQuery.platform = params.platform
  if (params.status) nextQuery.health = params.status
  if (params.group) nextQuery.group = params.group

  const search = String(params.search || '').trim()
  const accountMatch = search.match(/^#(\d+)$/)
  const proxyMatch = search.match(/^proxy:(\d+)$/i)
  if (accountMatch) nextQuery.account_id = accountMatch[1]
  else if (proxyMatch) nextQuery.proxy_id = proxyMatch[1]
  else if (search) nextQuery.search = search

  const currentKeys = Object.keys(route.query)
  const nextKeys = Object.keys(nextQuery)
  const same = currentKeys.length === nextKeys.length && nextKeys.every((key) => String(route.query[key] ?? '') === String(nextQuery[key] ?? ''))
  if (same) return

  syncingAccountRoute = true
  try {
    await router.replace({ query: nextQuery })
  } finally {
    syncingAccountRoute = false
  }
}

const {
  selectedIds: selIds,
  allVisibleSelected,
  isSelected,
  setSelectedIds,
  select,
  deselect,
  toggle: toggleSel,
  clear: clearSelection,
  removeMany: removeSelectedAccounts,
  toggleVisible,
  selectVisible: selectPage,
  batchUpdate
} = useTableSelection<Account>({
  rows: accounts,
  getId: (account) => account.id
})

const swipeVirtualContext: SwipeSelectVirtualContext = {
  getVirtualizer: () => dataTableRef.value?.virtualizer ?? null,
  getSortedData: () => dataTableRef.value?.sortedData ?? accounts.value,
  getRowId: (row: any) => row.id,
}

useSwipeSelect(accountTableRef, {
  isSelected,
  select,
  deselect,
  batchUpdate
}, swipeVirtualContext)

const accountRowAriaLabel = (account: Account) =>
  t('admin.accounts.workbench.inspectAccount', { name: account.name })

const lockInspectorBody = () => {
  if (isWideInspector.value || inspectorBodyLocked || typeof document === 'undefined') return
  previousBodyOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  inspectorBodyLocked = true
}

const unlockInspectorBody = () => {
  if (!inspectorBodyLocked || typeof document === 'undefined') return
  document.body.style.overflow = previousBodyOverflow
  previousBodyOverflow = ''
  inspectorBodyLocked = false
}

const openAccountInspector = async (
  account: Account,
  event?: MouseEvent | KeyboardEvent
) => {
  const eventTarget = event?.currentTarget
  const rowElement = accountTableRef.value?.querySelector<HTMLElement>(
    `[data-row-id="${account.id}"]`
  ) ?? null
  const activeElement = document.activeElement
  inspectorTriggerElement = eventTarget instanceof HTMLElement
    ? eventTarget
    : rowElement
      ?? (activeElement instanceof HTMLElement && activeElement !== document.body
        ? activeElement
        : null)
  selectedAccountId.value = account.id
  if (!isWideInspector.value) {
    lockInspectorBody()
    await nextTick()
    overlayInspectorRef.value?.focus()
  }
}

const closeAccountInspector = () => {
  const trigger = inspectorTriggerElement
  selectedAccountId.value = null
  inspectorTriggerElement = null
  unlockInspectorBody()
  void nextTick(() => trigger?.focus())
}

const handleViewProxy = (proxyId: number) => {
  if (!Number.isInteger(proxyId) || proxyId <= 0) return
  void router.push({
    name: 'AdminProxies',
    query: {
      focus_id: String(proxyId),
      open: 'health'
    }
  })
}

const handleInspectorViewportResize = () => {
  const wasWide = isWideInspector.value
  viewportWidth.value = window.innerWidth
  if (wasWide === isWideInspector.value || !selectedAccount.value) return
  if (isWideInspector.value) {
    unlockInspectorBody()
  } else {
    lockInspectorBody()
    void nextTick(() => overlayInspectorRef.value?.focus())
  }
}

watch([selectedAccount, loading], ([account, isLoading]) => {
  if (!account && !isLoading && selectedAccountId.value !== null) {
    closeAccountInspector()
  }
})

const resetAutoRefreshCache = () => {
  autoRefreshETag.value = null
}

const isFirstLoad = ref(true)
let initialInspectorSelectionHandled = false

const load = async () => {
  const requestParams = params as any
  syncAccountListDerivedParams()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = false
  if (isFirstLoad.value) {
    requestParams.lite = '1'
  }
  await baseLoad()
  if (!initialInspectorSelectionHandled) {
    initialInspectorSelectionHandled = true
    if (isWideInspector.value && accounts.value.length > 0) {
      selectedAccountId.value = accounts.value[0].id
    }
  }
  if (isFirstLoad.value) {
    isFirstLoad.value = false
    delete requestParams.lite
  }
  await refreshTodayStatsBatch()
}

const reload = async () => {
  syncAccountListDerivedParams()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = false
  await baseReload()
  await refreshTodayStatsBatch()
}

const debouncedReload = () => {
  syncAccountListDerivedParams()
  void syncAccountFiltersToRoute()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = true
  baseDebouncedReload()
}

const handlePageChange = (page: number) => {
  syncAccountListDerivedParams()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = true
  baseHandlePageChange(page)
}

const handlePageSizeChange = (size: number) => {
  syncAccountListDerivedParams()
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = true
  baseHandlePageSizeChange(size)
}

const handleSort = (key: string, order: AccountSortOrder) => {
  sortState.sort_by = key
  sortState.sort_order = order
  const requestParams = params as any
  requestParams.sort_by = accountSortRequestKey(key)
  requestParams.sort_order = order
  syncAccountListDerivedParams()
  pagination.page = 1
  hasPendingListSync.value = false
  resetAutoRefreshCache()
  pendingTodayStatsRefresh.value = true
  load()
}

watch(loading, (isLoading, wasLoading) => {
  if (wasLoading && !isLoading && pendingTodayStatsRefresh.value) {
    pendingTodayStatsRefresh.value = false
    refreshTodayStatsBatch().catch((error) => {
      console.error('Failed to refresh account today stats after table load:', error)
    })
  }
})

watch(
  () => route.query,
  async () => {
    if (!accountRouteReady || syncingAccountRoute) return
    await applyAccountRouteFilters()
    pagination.page = 1
    await reload()
  }
)

const isAnyModalOpen = computed(() => {
  return (
    showCreate.value ||
    showEdit.value ||
    showSync.value ||
    showImportData.value ||
    showExportDataDialog.value ||
    showBulkEdit.value ||
    showTempUnsched.value ||
    showDeleteDialog.value ||
    showReAuth.value ||
    showTest.value ||
    showStats.value ||
    showSchedulePanel.value ||
    showErrorPassthrough.value ||
    showTLSFingerprintProfiles.value
  )
})

const enterAutoRefreshSilentWindow = () => {
  autoRefreshSilentUntil.value = Date.now() + AUTO_REFRESH_SILENT_WINDOW_MS
  autoRefreshCountdown.value = autoRefreshIntervalSeconds.value
}

const inAutoRefreshSilentWindow = () => {
  return Date.now() < autoRefreshSilentUntil.value
}

const shouldReplaceAutoRefreshRow = (current: Account, next: Account) => {
  return (
    current.updated_at !== next.updated_at ||
    current.current_concurrency !== next.current_concurrency ||
    current.current_window_cost !== next.current_window_cost ||
    current.active_sessions !== next.active_sessions ||
    current.schedulable !== next.schedulable ||
    current.status !== next.status ||
    current.rate_limit_reset_at !== next.rate_limit_reset_at ||
    current.overload_until !== next.overload_until ||
    current.temp_unschedulable_until !== next.temp_unschedulable_until ||
    buildOpenAIUsageRefreshKey(current) !== buildOpenAIUsageRefreshKey(next)
  )
}

const syncAccountRefs = (nextAccount: Account) => {
  if (edAcc.value?.id === nextAccount.id) edAcc.value = nextAccount
  if (reAuthAcc.value?.id === nextAccount.id) reAuthAcc.value = nextAccount
  if (tempUnschedAcc.value?.id === nextAccount.id) tempUnschedAcc.value = nextAccount
  if (deletingAcc.value?.id === nextAccount.id) deletingAcc.value = nextAccount
  if (menu.acc?.id === nextAccount.id) menu.acc = nextAccount
}

const mergeAccountsIncrementally = (nextRows: Account[]) => {
  const currentRows = accounts.value
  const currentByID = new Map(currentRows.map(row => [row.id, row]))
  let changed = nextRows.length !== currentRows.length
  const mergedRows = nextRows.map((nextRow) => {
    const currentRow = currentByID.get(nextRow.id)
    if (!currentRow) {
      changed = true
      return nextRow
    }
    if (shouldReplaceAutoRefreshRow(currentRow, nextRow)) {
      changed = true
      syncAccountRefs(nextRow)
      return nextRow
    }
    return currentRow
  })
  if (!changed) {
    for (let i = 0; i < mergedRows.length; i += 1) {
      if (mergedRows[i].id !== currentRows[i]?.id) {
        changed = true
        break
      }
    }
  }
  if (changed) {
    accounts.value = mergedRows
  }
}

const refreshAccountsIncrementally = async () => {
  if (autoRefreshFetching.value) return
  syncAccountListDerivedParams()
  autoRefreshFetching.value = true
  try {
    const result = await adminAPI.accounts.listWithEtag(
      pagination.page,
      pagination.page_size,
      toRaw(params) as {
        platform?: string
        type?: string
        status?: string
        privacy_mode?: string
        group?: string
        search?: string
        sort_by?: string
        sort_order?: AccountSortOrder

      },
      { etag: autoRefreshETag.value }
    )

    if (result.etag) {
      autoRefreshETag.value = result.etag
    }
    if (!result.notModified && result.data) {
      pagination.total = result.data.total || 0
      pagination.pages = result.data.pages || 0
      mergeAccountsIncrementally(result.data.items || [])
      hasPendingListSync.value = false
    }

    await refreshTodayStatsBatch()
  } catch (error) {
    console.error('Auto refresh failed:', error)
  } finally {
    autoRefreshFetching.value = false
  }
}

const handleManualRefresh = async () => {
  await load()
  // Force usage cells to refetch /usage on explicit user refresh.
  usageManualRefreshToken.value += 1
}

const closeAccountToolsDropdown = () => {
  showAccountToolsDropdown.value = false
}

const openSyncFromCrs = () => {
  closeAccountToolsDropdown()
  showSync.value = true
}

const openImportData = () => {
  closeAccountToolsDropdown()
  showImportData.value = true
}

const openExportDataDialogFromMenu = () => {
  closeAccountToolsDropdown()
  openExportDataDialog()
}

const openErrorPassthrough = () => {
  closeAccountToolsDropdown()
  showErrorPassthrough.value = true
}

const openTLSFingerprintProfiles = () => {
  closeAccountToolsDropdown()
  showTLSFingerprintProfiles.value = true
}

const loadUpstreamBillingProbeSettings = async () => {
  upstreamBillingSettingsLoading.value = true
  try {
    Object.assign(upstreamBillingProbeSettings, await adminAPI.accounts.getUpstreamBillingProbeSettings())
  } catch (error) {
    console.error('Failed to load upstream billing probe settings:', error)
  } finally {
    upstreamBillingSettingsLoading.value = false
  }
}

const saveUpstreamBillingProbeSettings = async () => {
  upstreamBillingSettingsSaving.value = true
  try {
    const saved = await adminAPI.accounts.updateUpstreamBillingProbeSettings({ ...upstreamBillingProbeSettings })
    Object.assign(upstreamBillingProbeSettings, saved)
    appStore.showSuccess(t('admin.accounts.upstreamBilling.settingsSaved'))
  } catch (error) {
    console.error('Failed to save upstream billing probe settings:', error)
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.upstreamBilling.settingsFailed')))
  } finally {
    upstreamBillingSettingsSaving.value = false
  }
}

const syncPendingListChanges = async () => {
  hasPendingListSync.value = false
  await load()
  // Keep behavior consistent with manual refresh.
  usageManualRefreshToken.value += 1
}

const { pause: pauseAutoRefresh, resume: resumeAutoRefresh } = useIntervalFn(
  async () => {
    if (!autoRefreshEnabled.value) return
    if (document.hidden) return
    if (loading.value || autoRefreshFetching.value) return
    if (isAnyModalOpen.value) return
    if (menu.show || showAccountToolsDropdown.value || showAutoRefreshDropdown.value) return
    if (inAutoRefreshSilentWindow()) {
      autoRefreshCountdown.value = Math.max(
        0,
        Math.ceil((autoRefreshSilentUntil.value - Date.now()) / 1000)
      )
      return
    }

    if (autoRefreshCountdown.value <= 0) {
      autoRefreshCountdown.value = autoRefreshIntervalSeconds.value
      await refreshAccountsIncrementally()
      return
    }

    autoRefreshCountdown.value -= 1
  },
  1000,
  { immediate: false }
)

// Fresh billing/quota snapshots are authoritative. Imported credential tiers
// can be stale, so they remain fallbacks together with legacy plan_type fields.
function getAccountPlanType(row: any): string | undefined {
  if (!row) return undefined
  if (row.platform === 'grok') {
    const extra = (row.extra || {}) as Record<string, any>
    const billing = extra.grok_billing_snapshot as Record<string, any> | undefined
    const quota = extra.grok_quota_snapshot as Record<string, any> | undefined
    return (
      billing?.plan ||
      quota?.subscription_tier ||
      row.credentials?.subscription_tier ||
      extra.subscription_tier ||
      row.credentials?.plan_type ||
      row.parent_plan_type ||
      undefined
    )
  }
  return row.credentials?.plan_type || row.parent_plan_type || undefined
}

function getOpenAIAuthMode(row: any): string | undefined {
  if (!row || row.platform !== 'openai' || row.type !== 'oauth') return undefined
  const authMode = row.credentials?.auth_mode
  return typeof authMode === 'string' && authMode.trim() ? authMode : undefined
}

const accountGroupTitle = (account: Account): string =>
  account.groups?.map(group => group.name).join(', ') ?? ''

// Antigravity 订阅等级辅助函数
function getAntigravityTierFromRow(row: any): string | null {
  if (row.platform !== 'antigravity') return null
  const extra = row.extra as Record<string, unknown> | undefined
  if (!extra) return null
  const lca = extra.load_code_assist as Record<string, unknown> | undefined
  if (!lca) return null
  const paid = lca.paidTier as Record<string, unknown> | undefined
  if (paid && typeof paid.id === 'string') return paid.id
  const current = lca.currentTier as Record<string, unknown> | undefined
  if (current && typeof current.id === 'string') return current.id
  return null
}

function getAntigravityTierLabel(row: any): string | null {
  const tier = getAntigravityTierFromRow(row)
  switch (tier) {
    case 'free-tier': return t('admin.accounts.tier.free')
    case 'g1-pro-tier': return t('admin.accounts.tier.pro')
    case 'g1-ultra-tier': return t('admin.accounts.tier.ultra')
    default: return null
  }
}

type OpenAICompactBadgeState = 'active' | 'blocked' | 'auto'

function getOpenAICompactState(row: any): OpenAICompactBadgeState | null {
  if (row.platform !== 'openai' || (row.type !== 'oauth' && row.type !== 'apikey')) return null
  const extra = row.extra as Record<string, unknown> | undefined
  const mode = typeof extra?.openai_compact_mode === 'string' ? extra.openai_compact_mode : 'auto'
  if (mode === 'force_on') return 'active'
  if (mode === 'force_off') return 'blocked'
  if (typeof extra?.openai_compact_supported === 'boolean') {
    return extra.openai_compact_supported ? 'active' : 'blocked'
  }
  return 'auto'
}

function getOpenAICompactMeta(row: any): { label: string; className: string; dotClass: string } | null {
  const state = getOpenAICompactState(row)
  if (!state) return null
  switch (state) {
    case 'active':
      return {
        label: t('admin.accounts.openai.compactSupported'),
        className: 'text-emerald-600 dark:text-emerald-300',
        dotClass: 'bg-emerald-500 shadow-[0_0_0_2px_rgba(16,185,129,0.14)]'
      }
    case 'blocked':
      return {
        label: t('admin.accounts.openai.compactUnsupported'),
        className: 'text-rose-600 dark:text-rose-300',
        dotClass: 'bg-rose-500 shadow-[0_0_0_2px_rgba(244,63,94,0.14)]'
      }
    case 'auto':
      return {
        label: t('admin.accounts.openai.compactAuto'),
        className: 'text-slate-500 dark:text-slate-400',
        dotClass: 'bg-slate-300 dark:bg-slate-500'
      }
  }
}

function getOpenAICompactTitle(row: any): string {
  const extra = row.extra as Record<string, unknown> | undefined
  const checkedAt = typeof extra?.openai_compact_checked_at === 'string' ? extra.openai_compact_checked_at : ''
  const label = getOpenAICompactMeta(row)?.label || ''
  if (!checkedAt) return label
  return `${label} | ${t('admin.accounts.openai.compactLastChecked')}: ${formatDateTime(new Date(checkedAt))}`
}

function getAntigravityTierClass(row: any): string {
  const tier = getAntigravityTierFromRow(row)
  switch (tier) {
    case 'free-tier': return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    case 'g1-pro-tier': return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    case 'g1-ultra-tier': return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
    default: return ''
  }
}

// All available columns
const allColumns = computed(() => {
  const c = [
    { key: 'select', label: '', sortable: false, class: 'account-table-col--select' },
    { key: 'name', label: t('admin.accounts.workbench.identity'), sortable: true, class: 'account-table-col--identity' },
    { key: 'id', label: t('admin.accounts.columns.id'), sortable: true },
    { key: 'platform_type', label: t('admin.accounts.columns.platformType'), sortable: false },
    { key: 'capacity', label: t('admin.accounts.columns.capacity'), sortable: false },
    { key: 'usage', label: t('admin.accounts.workbench.quotaCapacity'), sortable: false, class: 'account-table-col--quota' },
    { key: 'status', label: t('admin.accounts.workbench.statusScheduling'), sortable: true, class: 'account-table-col--status' },
    { key: 'schedulable', label: t('admin.accounts.columns.schedulable'), sortable: true },
    { key: 'today_stats', label: t('admin.accounts.columns.todayStats'), sortable: false }
  ]
  if (!authStore.isSimpleMode) {
    c.push({
      key: 'groups',
      label: t('admin.accounts.workbench.effectiveGroups'),
      sortable: false,
      class: 'account-table-col--groups'
    })
  }
  c.push(
    { key: 'proxy', label: t('admin.accounts.columns.proxy'), sortable: false },
    { key: 'priority', label: t('admin.accounts.columns.priority'), sortable: true },
    { key: 'scheduler_score', label: t('admin.accounts.columns.schedulerScore'), sortable: false },
    { key: 'rate_multiplier', label: t('admin.accounts.columns.billingRateMultiplier'), sortable: true },
    { key: 'upstream_billing_rate', label: t('admin.accounts.columns.upstreamBillingRate'), sortable: false },
    {
      key: 'last_used_at',
      label: t('admin.accounts.workbench.recentActivity'),
      sortable: true,
      class: 'account-table-col--activity'
    },
    { key: 'created_at', label: t('admin.accounts.columns.createdAt'), sortable: true },
    { key: 'expires_at', label: t('admin.accounts.columns.expiresAt'), sortable: true },
    { key: 'notes', label: t('admin.accounts.columns.notes'), sortable: false },
    {
      key: 'actions',
      label: t('admin.accounts.columns.actions'),
      sortable: false,
      class: 'account-table-col--actions'
    }
  )
  return c
})

// Columns that can be toggled (exclude select, name, and actions)
const toggleableColumns = computed(() =>
  allColumns.value.filter(col => col.key !== 'select' && col.key !== 'name' && col.key !== 'actions')
)

// Filtered columns based on visibility
const cols = computed(() =>
  allColumns.value.filter(col =>
    col.key === 'select' || col.key === 'name' || col.key === 'actions' || !hiddenColumns.has(col.key)
  )
)
const hasExpandedColumns = computed(() =>
  cols.value.some(column => !COMPACT_WORKBENCH_COLUMN_KEYS.has(column.key))
)

const handleEdit = (a: Account) => { edAcc.value = a; showEdit.value = true }
const openMenu = (a: Account, e: MouseEvent) => {
  menu.acc = a

  const target = e.currentTarget as HTMLElement
  if (target) {
    const rect = target.getBoundingClientRect()
    const menuWidth = 200
    const menuHeight = 240
    const padding = 8
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    let left: number
    let top: number

    if (viewportWidth < 768) {
      // 居中显示,水平位置
      left = Math.max(padding, Math.min(
        rect.left + rect.width / 2 - menuWidth / 2,
        viewportWidth - menuWidth - padding
      ))

      // 优先显示在按钮下方
      top = rect.bottom + 4

      // 如果下方空间不够,显示在上方
      if (top + menuHeight > viewportHeight - padding) {
        top = rect.top - menuHeight - 4
        // 如果上方也不够,就贴在视口顶部
        if (top < padding) {
          top = padding
        }
      }
    } else {
      left = Math.max(padding, Math.min(
        e.clientX - menuWidth,
        viewportWidth - menuWidth - padding
      ))
      top = e.clientY
      if (top + menuHeight > viewportHeight - padding) {
        top = viewportHeight - menuHeight - padding
      }
    }

    menu.pos = { top, left }
  } else {
    menu.pos = { top: e.clientY, left: e.clientX - 200 }
  }

  menu.show = true
}
const toggleSelectAllVisible = (event: Event) => {
  const target = event.target as HTMLInputElement
  toggleVisible(target.checked)
}
const handleBulkDelete = async () => { if(!confirm(t('common.confirm'))) return; try { await Promise.all(selIds.value.map(id => adminAPI.accounts.delete(id))); clearSelection(); reload() } catch (error) { console.error('Failed to bulk delete accounts:', error) } }
const handleBulkResetStatus = async () => {
  if (!confirm(t('common.confirm'))) return
  try {
    const result = await adminAPI.accounts.batchClearError(selIds.value)
    if (result.failed > 0) {
      appStore.showError(t('admin.accounts.bulkActions.partialSuccess', { success: result.success, failed: result.failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.bulkActions.resetStatusSuccess', { count: result.success }))
      clearSelection()
    }
    reload()
  } catch (error) {
    console.error('Failed to bulk reset status:', error)
    appStore.showError(String(error))
  }
}
const handleBulkRefreshToken = async () => {
  if (!confirm(t('common.confirm'))) return
  try {
    const result = await adminAPI.accounts.batchRefresh(selIds.value)
    if (result.failed > 0) {
      appStore.showError(t('admin.accounts.bulkActions.partialSuccess', { success: result.success, failed: result.failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.bulkActions.refreshTokenSuccess', { count: result.success }))
      clearSelection()
    }
    reload()
  } catch (error) {
    console.error('Failed to bulk refresh token:', error)
    appStore.showError(String(error))
  }
}
const handleBulkProbeUpstreamBilling = async () => {
  const accountIDs = [...selIds.value]
  if (accountIDs.length === 0) {
    appStore.showError(t('admin.accounts.upstreamBilling.noEligibleAccounts'))
    return
  }
  if (accountIDs.length > 20) {
    appStore.showError(t('admin.accounts.upstreamBilling.batchLimit'))
    return
  }
  accountIDs.forEach(id => probingUpstreamBilling.add(id))
  try {
    const results = await adminAPI.accounts.probeUpstreamBillingBatch(accountIDs)
    results.forEach(result => {
      if (result.snapshot) patchUpstreamBillingSnapshot(result.account_id, result.snapshot)
    })
    const failed = results.filter(result => result.error).length
    if (failed > 0) {
      appStore.showError(t('admin.accounts.upstreamBilling.batchPartial', { success: results.length - failed, failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.upstreamBilling.batchCompleted', { count: results.length }))
    }
  } catch (error) {
    console.error('Failed to probe upstream billing in batch:', error)
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.upstreamBilling.probeFailed')))
  } finally {
    accountIDs.forEach(id => probingUpstreamBilling.delete(id))
  }
}
const updateSchedulableInList = (accountIds: number[], schedulable: boolean) => {
  if (accountIds.length === 0) return
  const idSet = new Set(accountIds)
  accounts.value = accounts.value.map((account) => (idSet.has(account.id) ? { ...account, schedulable } : account))
}
const normalizeBulkSchedulableResult = (
  result: {
    success?: number
    failed?: number
    success_ids?: number[]
    failed_ids?: number[]
    results?: Array<{ account_id: number; success: boolean }>
  },
  accountIds: number[]
) => {
  const responseSuccessIds = Array.isArray(result.success_ids) ? result.success_ids : []
  const responseFailedIds = Array.isArray(result.failed_ids) ? result.failed_ids : []
  if (responseSuccessIds.length > 0 || responseFailedIds.length > 0) {
    return {
      successIds: responseSuccessIds,
      failedIds: responseFailedIds,
      successCount: typeof result.success === 'number' ? result.success : responseSuccessIds.length,
      failedCount: typeof result.failed === 'number' ? result.failed : responseFailedIds.length,
      hasIds: true,
      hasCounts: true
    }
  }

  const results = Array.isArray(result.results) ? result.results : []
  if (results.length > 0) {
    const successIds = results.filter(item => item.success).map(item => item.account_id)
    const failedIds = results.filter(item => !item.success).map(item => item.account_id)
    return {
      successIds,
      failedIds,
      successCount: typeof result.success === 'number' ? result.success : successIds.length,
      failedCount: typeof result.failed === 'number' ? result.failed : failedIds.length,
      hasIds: true,
      hasCounts: true
    }
  }

  const hasExplicitCounts = typeof result.success === 'number' || typeof result.failed === 'number'
  const successCount = typeof result.success === 'number' ? result.success : 0
  const failedCount = typeof result.failed === 'number' ? result.failed : 0
  if (hasExplicitCounts && failedCount === 0 && successCount === accountIds.length && accountIds.length > 0) {
    return {
      successIds: accountIds,
      failedIds: [],
      successCount,
      failedCount,
      hasIds: true,
      hasCounts: true
    }
  }

  return {
    successIds: [],
    failedIds: [],
    successCount,
    failedCount,
    hasIds: false,
    hasCounts: hasExplicitCounts
  }
}
const handleBulkToggleSchedulable = async (schedulable: boolean) => {
  const accountIds = [...selIds.value]
  try {
    const result = await adminAPI.accounts.bulkUpdate(accountIds, { schedulable })
    const { successIds, failedIds, successCount, failedCount, hasIds, hasCounts } = normalizeBulkSchedulableResult(result, accountIds)
    if (!hasIds && !hasCounts) {
      appStore.showError(t('admin.accounts.bulkSchedulableResultUnknown'))
      setSelectedIds(accountIds)
      load().catch((error) => {
        console.error('Failed to refresh accounts:', error)
      })
      return
    }
    if (successIds.length > 0) {
      updateSchedulableInList(successIds, schedulable)
    }
    if (successCount > 0 && failedCount === 0) {
      const message = schedulable
        ? t('admin.accounts.bulkSchedulableEnabled', { count: successCount })
        : t('admin.accounts.bulkSchedulableDisabled', { count: successCount })
      appStore.showSuccess(message)
    }
    if (failedCount > 0) {
      const message = hasCounts || hasIds
        ? t('admin.accounts.bulkSchedulablePartial', { success: successCount, failed: failedCount })
        : t('admin.accounts.bulkSchedulableResultUnknown')
      appStore.showError(message)
      setSelectedIds(failedIds.length > 0 ? failedIds : accountIds)
    } else {
      if (hasIds) clearSelection()
      else setSelectedIds(accountIds)
    }
  } catch (error) {
    console.error('Failed to bulk toggle schedulable:', error)
    appStore.showError(t('common.error'))
  }
}
const buildBulkEditFilterSnapshot = () => {
  const rawParams = toRaw(params) as Record<string, unknown>
  const sortOrder: AccountSortOrder = rawParams.sort_order === 'desc' ? 'desc' : 'asc'
  return {
    platform: typeof rawParams.platform === 'string' ? rawParams.platform : '',
    type: typeof rawParams.type === 'string' ? rawParams.type : '',
    status: typeof rawParams.status === 'string' ? rawParams.status : '',
    group: typeof rawParams.group === 'string' ? rawParams.group : '',
    search: typeof rawParams.search === 'string' ? rawParams.search : '',
    privacy_mode: typeof rawParams.privacy_mode === 'string' ? rawParams.privacy_mode : '',
    sort_by: typeof rawParams.sort_by === 'string' ? rawParams.sort_by : '',
    sort_order: sortOrder
  }
}

const collectSelectionMetadata = (rows: Account[]) => {
  const selectedPlatforms = Array.from(new Set(rows.map(account => account.platform)))
  const selectedTypes = Array.from(new Set(rows.map(account => account.type)))
  return { selectedPlatforms, selectedTypes }
}

const openBulkEditSelected = () => {
  bulkEditTarget.value = {
    mode: 'selected',
    accountIds: [...selIds.value],
    selectedPlatforms: [...selPlatforms.value],
    selectedTypes: [...selTypes.value]
  }
  showBulkEdit.value = true
}

const openBulkEditFiltered = async () => {
  const filters = buildBulkEditFilterSnapshot()
  const preview = await adminAPI.accounts.list(1, 100, filters)
  const { selectedPlatforms, selectedTypes } = collectSelectionMetadata(preview.items)
  bulkEditTarget.value = {
    mode: 'filtered',
    filters,
    previewCount: preview.total,
    selectedPlatforms,
    selectedTypes
  }
  showBulkEdit.value = true
}

const openBulkEditFilteredFromMenu = async () => {
  closeAccountToolsDropdown()
  await openBulkEditFiltered()
}

const handleBulkUpdated = () => {
  showBulkEdit.value = false
  bulkEditTarget.value = null
  clearSelection()
  reload()
}
const handleDataImported = () => { showImportData.value = false; reload() }
const ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE = 'ungrouped'
const ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE = '__unset__'
const buildAccountQueryFilters = () => ({
  platform: params.platform || '',
  type: params.type || '',
  status: params.status || '',
  group: params.group || '',
  privacy_mode: params.privacy_mode || '',
  search: params.search || '',
  sort_by: accountSortRequestKey(sortState.sort_by),
  sort_order: sortState.sort_order
})
const accountMatchesCurrentFilters = (account: Account) => {
  const filters = buildAccountQueryFilters()
  if (filters.platform && account.platform !== filters.platform) return false
  if (filters.type && account.type !== filters.type) return false
  if (filters.status) {
    const now = Date.now()
    const rateLimitResetAt = account.rate_limit_reset_at ? new Date(account.rate_limit_reset_at).getTime() : Number.NaN
    const isRateLimited = Number.isFinite(rateLimitResetAt) && rateLimitResetAt > now
    const tempUnschedUntil = account.temp_unschedulable_until ? new Date(account.temp_unschedulable_until).getTime() : Number.NaN
    const isTempUnschedulable = Number.isFinite(tempUnschedUntil) && tempUnschedUntil > now

    if (filters.status === 'active') {
      if (account.status !== 'active' || isRateLimited || isTempUnschedulable || !account.schedulable) return false
    } else if (filters.status === 'rate_limited') {
      if (account.status !== 'active' || !isRateLimited || isTempUnschedulable) return false
    } else if (filters.status === 'temp_unschedulable') {
      if (account.status !== 'active' || !isTempUnschedulable) return false
    } else if (filters.status === 'unschedulable') {
      if (account.status !== 'active' || account.schedulable || isRateLimited || isTempUnschedulable) return false
    } else if (account.status !== filters.status) {
      return false
    }
  }
  if (filters.group) {
    const groupIds = account.group_ids ?? account.groups?.map((group) => group.id) ?? []
    if (filters.group === ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE) {
      if (groupIds.length > 0) return false
    } else if (!groupIds.includes(Number(filters.group))) {
      return false
    }
  }
  const privacyMode = typeof account.extra?.privacy_mode === 'string' ? account.extra.privacy_mode : ''
  if (filters.privacy_mode) {
    if (filters.privacy_mode === ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE) {
      if (privacyMode.trim() !== '') return false
    } else if (privacyMode !== filters.privacy_mode) {
      return false
    }
  }
  const search = String(filters.search || '').trim().toLowerCase()
  if (search && !account.name.toLowerCase().includes(search)) return false
  return true
}
const mergeRuntimeFields = (oldAccount: Account, updatedAccount: Account): Account => ({
  ...updatedAccount,
  current_concurrency: updatedAccount.current_concurrency ?? oldAccount.current_concurrency,
  current_window_cost: updatedAccount.current_window_cost ?? oldAccount.current_window_cost,
  active_sessions: updatedAccount.active_sessions ?? oldAccount.active_sessions
})

const syncPaginationAfterLocalRemoval = () => {
  const nextTotal = Math.max(0, pagination.total - 1)
  pagination.total = nextTotal
  pagination.pages = nextTotal > 0 ? Math.ceil(nextTotal / pagination.page_size) : 0

  const maxPage = Math.max(1, pagination.pages || 1)

  if (pagination.page > maxPage) {
    pagination.page = maxPage
  }
  // 行被本地移除后不立刻全量补页，改为提示用户手动同步。
  hasPendingListSync.value = nextTotal > 0
}

const patchAccountInList = (updatedAccount: Account) => {
  const index = accounts.value.findIndex(account => account.id === updatedAccount.id)
  if (index === -1) return
  const mergedAccount = mergeRuntimeFields(accounts.value[index], updatedAccount)
  if (!accountMatchesCurrentFilters(mergedAccount)) {
    accounts.value = accounts.value.filter(account => account.id !== mergedAccount.id)
    syncPaginationAfterLocalRemoval()
    removeSelectedAccounts([mergedAccount.id])
    if (menu.acc?.id === mergedAccount.id) {
      menu.show = false
      menu.acc = null
    }
    return
  }
  const nextAccounts = [...accounts.value]
  nextAccounts[index] = mergedAccount
  accounts.value = nextAccounts
  syncAccountRefs(mergedAccount)
}
const patchUpstreamBillingSnapshot = (accountID: number, snapshot: UpstreamBillingProbeSnapshot) => {
  const account = accounts.value.find(item => item.id === accountID)
  if (!account) return
  patchAccountInList({
    ...account,
    extra: { ...account.extra, upstream_billing_probe: snapshot }
  })
}
const handleProbeUpstreamBilling = async (account: Account) => {
  if (probingUpstreamBilling.has(account.id)) return
  probingUpstreamBilling.add(account.id)
  try {
    const result = await adminAPI.accounts.probeUpstreamBilling(account.id)
    if (result.snapshot) patchUpstreamBillingSnapshot(account.id, result.snapshot)
  } catch (error) {
    console.error('Failed to probe upstream billing:', error)
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.upstreamBilling.probeFailed')))
  } finally {
    probingUpstreamBilling.delete(account.id)
  }
}
const handleAccountUpdated = (updatedAccount: Account) => {
  patchAccountInList(updatedAccount)
  enterAutoRefreshSilentWindow()
}
const formatExportTimestamp = () => {
  const now = new Date()
  const pad2 = (value: number) => String(value).padStart(2, '0')
  return `${now.getFullYear()}${pad2(now.getMonth() + 1)}${pad2(now.getDate())}${pad2(now.getHours())}${pad2(now.getMinutes())}${pad2(now.getSeconds())}`
}
const openExportDataDialog = () => {
  includeProxyOnExport.value = true
  showExportDataDialog.value = true
}
const handleExportData = async () => {
  if (exportingData.value) return
  exportingData.value = true
  try {
    const dataPayload = await accountExportStepUp.run(() => adminAPI.accounts.exportData(
      selIds.value.length > 0
        ? { ids: selIds.value, includeProxies: includeProxyOnExport.value }
        : {
            includeProxies: includeProxyOnExport.value,
            filters: buildAccountQueryFilters()
          }
    ))
    const timestamp = formatExportTimestamp()
    const filename = `sub2api-account-${timestamp}.json`
    const blob = new Blob([JSON.stringify(dataPayload, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    link.click()
    URL.revokeObjectURL(url)
    // spark 影子账号被后端排除出备份(其凭据透传母账号、调度配置不可经凭据型导入重建);
    // 跳过非零时明确提示用户,避免「下载成功但少了账号」的静默丢失。
    if (dataPayload.skipped_shadows && dataPayload.skipped_shadows > 0) {
      appStore.showWarning(t('admin.accounts.dataExportedSkippedShadows', { count: dataPayload.skipped_shadows }))
    } else {
      appStore.showSuccess(t('admin.accounts.dataExported'))
    }
  } catch (error: any) {
    if (isStepUpCancelled(error)) {
      // 用户主动取消 step-up 验证，静默返回，不弹错误提示。
    } else if (isStepUpBlocked(error)) {
      appStore.showError(
        stepUpBlockReason(error) === 'STEP_UP_ADMIN_API_KEY_FORBIDDEN'
          ? t('stepUp.adminApiKeyForbidden')
          : t('stepUp.notEnabled')
      )
    } else {
      appStore.showError(error?.message || t('admin.accounts.dataExportFailed'))
    }
  } finally {
    exportingData.value = false
    showExportDataDialog.value = false
  }
}
const accountExportStepUp = useStepUp()
const closeTestModal = () => { showTest.value = false; testingAcc.value = null }
const closeStatsModal = () => { showStats.value = false; statsAcc.value = null }
const closeReAuthModal = () => { showReAuth.value = false; reAuthAcc.value = null }
const handleTest = (a: Account) => { testingAcc.value = a; showTest.value = true }
let accountTestRefreshVersion = 0
const handleTestCompleted = async () => {
  const accountID = testingAcc.value?.id
  if (!accountID) return
  invalidateAccountUsageHealthSnapshot(accountID)
  const refreshVersion = ++accountTestRefreshVersion
  try {
    const updated = await adminAPI.accounts.getById(accountID)
    if (refreshVersion !== accountTestRefreshVersion) return
    patchAccountInList(updated)
    enterAutoRefreshSilentWindow()
    try {
      await requestAccountUsage(updated, {
        source: 'active',
        bypassCache: true,
        force: true
      }).promise
    } catch (error) {
      console.error('Failed to refresh account usage after connection test:', error)
    }
  } catch (error) {
    console.error('Failed to refresh account after connection test:', error)
  }
}
const handleViewStats = (a: Account) => { statsAcc.value = a; showStats.value = true }
const handleSchedule = async (a: Account) => {
  scheduleAcc.value = a
  scheduleModelOptions.value = []
  showSchedulePanel.value = true
  try {
    const models = await adminAPI.accounts.getAvailableModels(a.id)
    scheduleModelOptions.value = models.map((m: ClaudeModel) => ({ value: m.id, label: m.display_name || m.id }))
  } catch {
    scheduleModelOptions.value = []
  }
}
const closeSchedulePanel = () => { showSchedulePanel.value = false; scheduleAcc.value = null; scheduleModelOptions.value = [] }
const handleReAuth = (a: Account) => { reAuthAcc.value = a; showReAuth.value = true }
const duplicatingAccountIDs = new Set<number>()
const handleDuplicateAccount = async (a: Account) => {
  if (duplicatingAccountIDs.has(a.id)) return
  duplicatingAccountIDs.add(a.id)
  try {
    const duplicate = await adminAPI.accounts.duplicate(a.id)
    appStore.showSuccess(t('admin.accounts.duplicateSuccess', { name: duplicate.name }))
    reload()
  } catch (error: any) {
    console.error('Failed to duplicate account:', error)
    appStore.showError(error?.message || t('admin.accounts.duplicateFailed'))
  } finally {
    duplicatingAccountIDs.delete(a.id)
  }
}
const handleRefresh = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.refreshCredentials(a.id)
    patchAccountInList(updated)
    enterAutoRefreshSilentWindow()
  } catch (error) {
    console.error('Failed to refresh credentials:', error)
  }
}
const handleRecoverState = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.recoverState(a.id)
    patchAccountInList(updated)
    enterAutoRefreshSilentWindow()
    appStore.showSuccess(t('admin.accounts.recoverStateSuccess'))
  } catch (error: any) {
    console.error('Failed to recover account state:', error)
    appStore.showError(error?.message || t('admin.accounts.recoverStateFailed'))
  }
}
const handleResetQuota = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.resetAccountQuota(a.id)
    patchAccountInList(updated)
    enterAutoRefreshSilentWindow()
    appStore.showSuccess(t('common.success'))
  } catch (error) {
    console.error('Failed to reset quota:', error)
  }
}

const privacyResultMessageKey = (account: Account): { type: 'success' | 'error'; key: string } => {
  const mode = typeof account.extra?.privacy_mode === 'string' ? account.extra.privacy_mode : ''
  if (account.platform === 'openai') {
    switch (mode) {
      case 'training_off':
        return { type: 'success', key: 'admin.accounts.privacyTrainingOff' }
      case 'training_set_cf_blocked':
        return { type: 'error', key: 'admin.accounts.privacyCfBlocked' }
      default:
        return { type: 'error', key: 'admin.accounts.privacyFailed' }
    }
  }
  if (account.platform === 'antigravity') {
    if (mode === 'privacy_set') {
      return { type: 'success', key: 'admin.accounts.privacyAntigravitySet' }
    }
    return { type: 'error', key: 'admin.accounts.privacyAntigravityFailed' }
  }
  return { type: 'error', key: 'admin.accounts.privacyFailed' }
}

const handleSetPrivacy = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.setPrivacy(a.id)
    patchAccountInList(updated)
    enterAutoRefreshSilentWindow()
    const result = privacyResultMessageKey(updated)
    if (result.type === 'success') {
      appStore.showSuccess(t(result.key))
    } else {
      appStore.showError(t(result.key))
    }
  } catch (error: any) {
    console.error('Failed to set privacy:', error)
    appStore.showError(error?.response?.data?.message || t('admin.accounts.privacyFailed'))
  }
}
const onRevertFallback = async (a: Account) => {
  try {
    await adminAPI.accounts.revertProxyFallback(a.id)
    appStore.showSuccess(t('admin.accounts.revertProxySuccess'))
    reload()
  } catch (error: any) {
    console.error('Failed to revert proxy fallback:', error)
    appStore.showError(error?.response?.data?.message || t('admin.accounts.revertProxyFailed'))
  }
}
const handleCreateSparkShadow = (a: Account) => {
  creatingShadowAcc.value = a
  showCreateShadowDialog.value = true
}
const confirmCreateSparkShadow = async () => {
  const a = creatingShadowAcc.value
  if (!a) return
  try {
    await adminAPI.accounts.createSparkShadow(a.id, { name: `${a.name} (Spark)` })
    showCreateShadowDialog.value = false
    creatingShadowAcc.value = null
    appStore.showSuccess(t('admin.accounts.createSparkShadowSuccess'))
    reload()
  } catch (error: any) {
    console.error('Failed to create spark shadow:', error)
    appStore.showError(error?.response?.data?.message || t('admin.accounts.createSparkShadowFailed'))
  }
}
const handleDelete = (a: Account) => { deletingAcc.value = a; showDeleteDialog.value = true }
const confirmDelete = async () => { if(!deletingAcc.value) return; try { await adminAPI.accounts.delete(deletingAcc.value.id); showDeleteDialog.value = false; deletingAcc.value = null; reload() } catch (error) { console.error('Failed to delete account:', error) } }
const handleToggleSchedulable = async (a: Account) => {
  const nextSchedulable = !a.schedulable
  togglingSchedulable.value = a.id
  try {
    const updated = await adminAPI.accounts.setSchedulable(a.id, nextSchedulable)
    updateSchedulableInList([a.id], updated?.schedulable ?? nextSchedulable)
    enterAutoRefreshSilentWindow()
  } catch (error) {
    console.error('Failed to toggle schedulable:', error)
    appStore.showError(t('admin.accounts.failedToToggleSchedulable'))
  } finally {
    togglingSchedulable.value = null
  }
}
const handleShowTempUnsched = (a: Account) => { tempUnschedAcc.value = a; showTempUnsched.value = true }
const handleTempUnschedReset = async (updated: Account) => {
  showTempUnsched.value = false
  tempUnschedAcc.value = null
  patchAccountInList(updated)
  enterAutoRefreshSilentWindow()
}
const formatExpiresAt = (value: number | null) => {
  if (!value) return '-'
  return formatDateTime(
    new Date(value * 1000),
    {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    },
    'sv-SE'
  )
}
const isExpired = (value: number | null) => {
  if (!value) return false
  return value * 1000 <= Date.now()
}
// 所绑定代理的有效期(逻辑同 /admin/proxies,见 utils/proxyExpiry)
const proxyExpiryBadge = (p: AccountProxy): string => proxyExpiryBadgeClass(p.expires_at, p.status)
const proxyExpiryText = (p: AccountProxy): string => {
  const { key, params } = proxyExpiryLabelKey(p.expires_at, p.status)
  return params ? t(key, params) : t(key)
}

// 滚动时关闭操作菜单（不关闭列设置下拉菜单）
const handleScroll = () => {
  menu.show = false
}

// 点击外部关闭顶部下拉菜单
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (accountToolsDropdownRef.value && !accountToolsDropdownRef.value.contains(target)) {
    showAccountToolsDropdown.value = false
  }
  if (autoRefreshDropdownRef.value && !autoRefreshDropdownRef.value.contains(target)) {
    showAutoRefreshDropdown.value = false
  }
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(target)) {
    showColumnDropdown.value = false
  }
}

onMounted(async () => {
  viewportWidth.value = window.innerWidth
  await applyAccountRouteFilters()
  accountRouteReady = true
  load()
  loadUpstreamBillingProbeSettings()
  try {
    const loadProxies = adminAPI.proxies.getAllWithCount ?? adminAPI.proxies.getAll
    const [p, g] = await Promise.all([loadProxies(), adminAPI.groups.getAll()])
    proxies.value = p
    groups.value = g
  } catch (error) {
    console.error('Failed to load proxies/groups:', error)
  }
  window.addEventListener('resize', handleInspectorViewportResize)
  window.addEventListener('scroll', handleScroll, true)
  document.addEventListener('click', handleClickOutside)

  if (autoRefreshEnabled.value) {
    autoRefreshCountdown.value = autoRefreshIntervalSeconds.value
    resumeAutoRefresh()
  } else {
    pauseAutoRefresh()
  }
})

onUnmounted(() => {
  unlockInspectorBody()
  window.removeEventListener('resize', handleInspectorViewportResize)
  window.removeEventListener('scroll', handleScroll, true)
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.account-filter-toolbar {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 12px;
}

.account-toolbar-icon-button {
  display: inline-flex;
  width: 44px;
  height: 44px;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--lx-clay-border);
  border-radius: 12px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface);
  box-shadow: 0 5px 13px rgb(70 55 96 / 0.06);
  transition:
    color 150ms ease-out,
    background-color 150ms ease-out,
    border-color 150ms ease-out;
}

.account-toolbar-icon-button:hover,
.account-toolbar-icon-button--active {
  color: var(--lx-clay-accent-deep);
  border-color: color-mix(in srgb, var(--lx-clay-accent) 34%, var(--lx-clay-border));
  background: var(--lx-clay-accent-soft);
}

.account-toolbar-icon-button:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--lx-clay-accent) 55%, transparent);
  outline-offset: 2px;
}

.account-tools-menu-item {
  @apply flex min-h-11 w-full items-center gap-3 rounded-md px-3 py-2 text-sm text-gray-700 transition-colors hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700;
}

.account-tools-menu-icon {
  @apply inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md;
}

.account-workbench {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  min-width: 0;
  min-height: 0;
  flex: 1;
  gap: 20px;
}

.account-workbench__table-surface,
.account-workbench__inspector-surface {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: 14px;
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.account-workbench__table-surface > :deep(.account-bulk-actions-bar) {
  flex: none;
}

.account-workbench__pagination {
  flex: none;
  padding: 10px 14px;
  border-top: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface);
}

.account-identity-cell {
  display: flex;
  min-width: 0;
  max-width: 100%;
  flex-direction: column;
  gap: 3px;
  overflow: hidden;
}

.account-identity-cell__name {
  overflow: hidden;
  color: var(--lx-clay-text);
  font-size: 0.875rem;
  font-weight: 700;
  line-height: 1.25rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-quota-capacity {
  display: flex;
  min-width: 0;
  max-width: 100%;
  flex-direction: column;
  gap: 2px;
}

.account-quota-capacity__capacity {
  overflow: hidden;
  color: var(--lx-clay-text-muted);
  font-size: 0.625rem;
  line-height: 0.875rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-quota-capacity :deep(.account-usage-summary) {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.account-quota-capacity :deep(.account-usage-summary__top) {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 0.65rem;
  font-weight: 650;
  line-height: 0.875rem;
}

.account-quota-capacity :deep(.account-usage-summary__label) {
  flex: none;
  color: var(--lx-clay-text-muted);
}

.account-quota-capacity :deep(.account-usage-summary__value) {
  overflow: hidden;
  color: var(--lx-clay-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-quota-capacity :deep(.account-usage-summary__track) {
  display: flex;
  width: 100%;
  height: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--lx-clay-recessed);
}

.account-quota-capacity :deep(.account-usage-summary__track > span) {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--lx-clay-success-bright);
  transition: width 220ms ease-out;
}

.account-quota-capacity :deep(.account-usage-summary[data-level='warning'] .account-usage-summary__value) {
  color: var(--lx-clay-warning);
}

.account-quota-capacity :deep(.account-usage-summary[data-level='warning'] .account-usage-summary__track > span) {
  background: var(--lx-clay-warning-bright);
}

.account-quota-capacity :deep(.account-usage-summary[data-level='danger'] .account-usage-summary__value) {
  color: var(--lx-clay-danger);
}

.account-quota-capacity :deep(.account-usage-summary[data-level='danger'] .account-usage-summary__track > span) {
  background: var(--lx-clay-danger);
}

.account-quota-capacity :deep(.account-usage-summary[data-level='empty'] .account-usage-summary__track > span) {
  background: transparent;
}

.account-status-cell {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
}

.account-status-cell > :deep(.account-status-summary) {
  min-width: 0;
  flex: 1 1 auto;
}

.account-status-cell__switch {
  position: relative;
  width: 32px;
  height: 44px;
  min-height: 44px;
  flex: none;
  margin-left: auto;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
}

.account-status-cell__switch::before {
  position: absolute;
  top: 13px;
  left: 0;
  width: 32px;
  height: 18px;
  border-radius: 999px;
  background: var(--lx-clay-recessed-strong);
  content: '';
  transition: background-color 160ms ease-out;
}

.account-status-cell__switch[aria-checked='true']::before {
  background: var(--lx-clay-accent);
}

.account-status-cell__switch:disabled {
  cursor: wait;
  opacity: 0.55;
}

.account-status-cell__switch span {
  position: absolute;
  top: 15px;
  left: 2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: white;
  box-shadow: 0 1px 3px rgb(15 23 42 / 0.24);
  transition: transform 160ms ease-out;
}

.account-status-cell__switch[aria-checked='true'] span {
  transform: translateX(14px);
}

.account-groups-summary {
  display: flex;
  min-width: 0;
  max-width: 100%;
  align-items: center;
  gap: 4px;
  overflow: hidden;
}

.account-groups-summary :deep(.account-groups-summary__badge) {
  min-width: 0;
  max-width: 100%;
  padding: 2px 6px;
  font-size: 0.625rem;
  line-height: 0.875rem;
}

.account-groups-summary__more {
  flex: none;
  color: var(--lx-clay-text-muted);
  font-size: 0.625rem;
}

.account-row-more-button {
  display: inline-flex;
  width: 30px;
  height: 30px;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--lx-clay-text-muted);
  transition:
    color 150ms ease-out,
    background-color 150ms ease-out;
}

.account-row-more-button:hover {
  color: var(--lx-clay-text);
  background: color-mix(in srgb, var(--lx-clay-text) 5%, transparent);
}

.account-workbench__table-surface :deep(.data-table--desktop) {
  /* Keep the second sticky column aligned with the percentage-based select column. */
  --select-col-width: 5%;
  min-height: 0;
  flex: 1;
  overflow-x: hidden;
}

.account-workbench__table-surface :deep(.data-table-table) {
  width: 100%;
  min-width: 0 !important;
  table-layout: fixed;
}

.account-workbench__table-surface :deep(.data-table-header-cell) {
  padding: 10px 12px;
  letter-spacing: 0;
  text-transform: none;
  white-space: nowrap;
  overflow: hidden;
  font-size: 0.75rem;
  font-weight: 700;
  text-overflow: ellipsis;
}

.account-workbench__table-surface :deep(.data-table-cell) {
  padding: 8px 10px;
  overflow: hidden;
  white-space: normal;
  vertical-align: middle;
}

.account-workbench__table-surface :deep(.data-table-row--selected > td) {
  background: color-mix(in srgb, var(--lx-clay-accent) 8%, var(--lx-clay-surface)) !important;
}

.account-workbench__table-surface :deep(.data-table-row--selected:focus-visible) {
  outline: 2px solid color-mix(in srgb, var(--lx-clay-accent) 48%, transparent);
  outline-offset: -2px;
}

.account-workbench__table-surface :deep(.account-table-col--select) {
  width: 5%;
  padding-right: 6px;
  padding-left: 6px;
  text-align: center;
}

.account-workbench__table-surface :deep(.account-table-col--identity) {
  width: 25%;
}

.account-workbench__table-surface :deep(.account-table-col--quota) {
  width: 24%;
}

.account-workbench__table-surface :deep(.account-table-col--status) {
  width: 18%;
}

.account-workbench__table-surface :deep(.account-table-col--groups) {
  width: 10%;
}

.account-workbench__table-surface :deep(.account-table-col--activity) {
  width: 11%;
}

.account-workbench__table-surface :deep(.account-table-col--actions) {
  width: 7%;
  padding-right: 8px;
  padding-left: 8px;
  text-align: right;
}

.account-workbench__table-surface :deep(.account-table-col--actions .data-table-header-content) {
  justify-content: flex-end;
}

.account-workbench__table-surface :deep(.data-table-row + .data-table-row) {
  border-color: color-mix(in srgb, var(--lx-clay-border) 45%, transparent);
}

.account-workbench__table-surface--expanded :deep(.data-table--desktop) {
  /* Expanded tables switch the select column to a fixed width. */
  --select-col-width: 40px;
  overflow-x: auto;
}

.account-workbench__table-surface--expanded :deep(.data-table-table) {
  min-width: max-content !important;
  table-layout: auto;
}

.account-workbench__table-surface--expanded :deep(.account-table-col--select) {
  width: 40px;
  min-width: 40px;
  max-width: 40px;
}

.account-workbench__table-surface--expanded :deep(.account-table-col--identity),
.account-workbench__table-surface--expanded :deep(.account-table-col--quota) {
  min-width: 190px;
}

.account-workbench__table-surface--expanded :deep(.account-table-col--status) {
  min-width: 110px;
}

.account-workbench__table-surface--expanded :deep(.account-table-col--groups) {
  min-width: 96px;
}

.account-workbench__table-surface--expanded :deep(.account-table-col--activity) {
  min-width: 90px;
}

.account-workbench__table-surface--expanded :deep(.account-table-col--actions) {
  min-width: 50px;
}

.account-inspector-overlay {
  position: fixed;
  z-index: 45;
  inset: 0;
  display: flex;
  background: color-mix(in srgb, var(--lx-clay-text) 28%, transparent);
}

.account-inspector-overlay--drawer {
  top: var(--app-shell-top-offset);
  align-items: stretch;
  justify-content: flex-end;
}

.account-inspector-overlay--drawer :deep(.account-inspector) {
  width: min(420px, calc(100vw - 24px));
  height: 100%;
  border-left: 1px solid var(--lx-clay-border-strong);
  box-shadow: -16px 0 36px color-mix(in srgb, var(--lx-clay-text) 18%, transparent);
}

.account-inspector-overlay--sheet {
  align-items: flex-end;
  justify-content: stretch;
}

.account-inspector-overlay--sheet :deep(.account-inspector) {
  width: 100%;
  max-height: min(86dvh, 760px);
  border-top: 1px solid var(--lx-clay-border-strong);
  border-radius: 16px 16px 0 0;
  box-shadow: 0 -18px 42px color-mix(in srgb, var(--lx-clay-text) 18%, transparent);
}

.account-inspector-overlay-enter-active,
.account-inspector-overlay-leave-active {
  transition: opacity 180ms ease-out;
}

.account-inspector-overlay-enter-active :deep(.account-inspector),
.account-inspector-overlay-leave-active :deep(.account-inspector) {
  transition: transform 220ms cubic-bezier(0.16, 1, 0.3, 1);
}

.account-inspector-overlay-enter-from,
.account-inspector-overlay-leave-to {
  opacity: 0;
}

.account-inspector-overlay--drawer.account-inspector-overlay-enter-from :deep(.account-inspector),
.account-inspector-overlay--drawer.account-inspector-overlay-leave-to :deep(.account-inspector) {
  transform: translateX(100%);
}

.account-inspector-overlay--sheet.account-inspector-overlay-enter-from :deep(.account-inspector),
.account-inspector-overlay--sheet.account-inspector-overlay-leave-to :deep(.account-inspector) {
  transform: translateY(100%);
}

@media (min-width: 1280px) {
  .account-workbench {
    grid-template-columns: minmax(0, 64fr) minmax(320px, 36fr);
  }

  .account-workbench__inspector-surface :deep(.account-inspector) {
    height: 100%;
  }
}

@media (min-width: 1024px) {
  .account-table-page-layout {
    height: calc(100dvh - var(--app-shell-top-offset) - var(--app-main-block-padding));
  }
}

@media (max-width: 1023px) {
  .account-workbench,
  .account-workbench__table-surface {
    min-height: auto;
  }

  .account-workbench__table-surface {
    overflow: visible;
    border: 0;
    border-radius: 0;
    background: transparent;
    box-shadow: none;
  }

  .account-workbench__table-body {
    overflow: visible;
  }

  .account-workbench__table-surface :deep(.data-table-mobile-row) {
    padding: 14px;
  }

  .account-workbench__table-surface :deep(.data-table-mobile-field) {
    align-items: flex-start;
  }

  .account-workbench__table-surface :deep(.data-table-mobile-value) {
    min-width: 0;
    max-width: 70%;
  }

  .account-workbench__pagination {
    margin-top: 12px;
    border: 1px solid var(--lx-clay-border);
    border-radius: 12px;
  }
}

@media (max-width: 639px) {
  .account-filter-toolbar {
    flex-direction: column;
  }

  .account-inspector-overlay {
    background: color-mix(in srgb, var(--lx-clay-text) 34%, transparent);
  }

  .account-workbench__table-surface :deep(.data-table-mobile-value) {
    max-width: 64%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .account-status-cell__switch,
  .account-status-cell__switch span,
  .account-inspector-overlay,
  .account-inspector-overlay :deep(.account-inspector) {
    transition-duration: 0.01ms;
  }
}
</style>
