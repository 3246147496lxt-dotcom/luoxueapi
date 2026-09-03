<template>
  <!--
  THESIS: A durable intake control room; it refuses a detached settings-table workflow.
  OWN-WORLD: Snow Clay canvas, violet selection, quiet status color, dense list-and-inspector controls.
  STORY: The operator sees what is moving, opens one run, resolves exceptions, and publishes eligible Skills.
  FIRST VIEWPORT: Header and local navigation above a narrow run queue and a wide persistent inspector; create is top-right.
  FORM: Grounded control-console structure, surface candidate 6, concept seed 89f5e4b5.
  FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, and DESIGN.md
  -->
  <AppLayout variant="home-clay">
    <main
      class="skill-import-page"
      data-admin-page-kind="ops"
      data-direction-contract="89f5e4b5"
    >
      <header class="skill-import-header">
        <AdminPageHeader
          :title="t('admin.skills.imports.title')"
          :description="t('admin.skills.imports.description')"
        >
          <template #meta>
            <span
              class="skill-import-live-state"
              :class="{ 'skill-import-live-state--active': hasActiveRun }"
              role="status"
              aria-live="polite"
            >
              <span aria-hidden="true"></span>
              {{ hasActiveRun
                ? t('admin.skills.imports.polling.active')
                : t('admin.skills.imports.polling.idle') }}
            </span>
          </template>
          <template #secondary-actions>
            <button type="button" class="btn btn-secondary" :disabled="pageLoading" @click="refreshCurrentView">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': pageLoading }" aria-hidden="true" />
              <span class="ml-1.5">{{ t('admin.skills.imports.actions.refresh') }}</span>
            </button>
          </template>
          <template #primary-actions>
            <template v-if="activeSection === 'imports'">
              <button type="button" class="btn btn-secondary" @click="openRunComposer('upload')">
                <Icon name="upload" size="sm" aria-hidden="true" />
                <span class="ml-1.5">{{ t('admin.skills.imports.actions.uploadManifest') }}</span>
              </button>
              <button type="button" class="btn btn-primary" @click="openRunComposer('source')">
                <Icon name="plus" size="sm" aria-hidden="true" />
                <span class="ml-1.5">{{ t('admin.skills.imports.actions.newRun') }}</span>
              </button>
            </template>
            <template v-else>
              <button type="button" class="btn btn-secondary" @click="openSourceEditor()">
                <Icon name="globe" size="sm" aria-hidden="true" />
                <span class="ml-1.5">{{ t('admin.skills.imports.actions.newSource') }}</span>
              </button>
              <button type="button" class="btn btn-primary" :disabled="enabledSources.length === 0" @click="openScheduleEditor()">
                <Icon name="plus" size="sm" aria-hidden="true" />
                <span class="ml-1.5">{{ t('admin.skills.imports.actions.newSchedule') }}</span>
              </button>
            </template>
          </template>
        </AdminPageHeader>
        <SkillMarketNav :active="activeSection" />
      </header>

      <section
        v-if="activeSection === 'imports' && runComposerOpen"
        class="skill-import-composer"
        aria-labelledby="skill-import-composer-title"
      >
        <div class="skill-import-composer__intro">
          <span class="skill-import-composer__icon" aria-hidden="true">
            <Icon :name="runDraft.kind === 'upload' ? 'upload' : 'sync'" size="md" />
          </span>
          <div>
            <h2 id="skill-import-composer-title">
              {{ runDraft.kind === 'upload'
                ? t('admin.skills.imports.composer.uploadTitle')
                : t('admin.skills.imports.composer.sourceTitle') }}
            </h2>
            <p id="skill-import-composer-hint">
              {{ runDraft.kind === 'upload'
                ? t('admin.skills.imports.composer.uploadHint')
                : t('admin.skills.imports.composer.hint') }}
            </p>
          </div>
        </div>

        <form class="skill-import-composer__form" @submit.prevent="submitRun">
          <label>
            <span>{{ t('admin.skills.imports.fields.source') }}</span>
            <select v-model.number="runDraft.sourceId" class="input" required>
              <option :value="0" disabled>{{ t('admin.skills.imports.fields.selectSource') }}</option>
              <option v-for="source in composerSources" :key="source.id" :value="source.id">
                {{ source.name }}
              </option>
            </select>
          </label>
          <label>
            <span>{{ t('admin.skills.imports.fields.mode') }}</span>
            <select v-model="runDraft.mode" class="input">
              <option value="auto_publish">{{ t('admin.skills.imports.mode.autoPublish') }}</option>
              <option value="review">{{ t('admin.skills.imports.mode.review') }}</option>
              <option value="dry_run">{{ t('admin.skills.imports.mode.dryRun') }}</option>
            </select>
          </label>
          <label>
            <span>{{ t('admin.skills.imports.fields.startRank') }}</span>
            <input v-model.number="runDraft.startRank" class="input" type="number" min="1" required />
          </label>
          <label>
            <span>{{ t('admin.skills.imports.fields.limit') }}</span>
            <input v-model.number="runDraft.limit" class="input" type="number" min="1" max="5000" required />
          </label>
          <label v-if="runDraft.kind === 'upload'" class="skill-import-file-field">
            <span>{{ t('admin.skills.imports.fields.manifestFile') }}</span>
            <input
              ref="manifestInput"
              type="file"
              accept=".json,.csv,.zip,application/json,text/csv,application/zip"
              aria-describedby="skill-import-composer-hint"
              required
              @change="onManifestSelected"
            />
            <span class="skill-import-file-field__button">
              <Icon name="paperclip" size="sm" aria-hidden="true" />
              {{ manifestFile?.name ?? t('admin.skills.imports.composer.chooseFile') }}
            </span>
          </label>
          <div class="skill-import-composer__safe-gate">
            <div>
              <span>{{ t('admin.skills.imports.fields.safeGate') }}</span>
              <small>{{ t('admin.skills.imports.composer.safeGateHint') }}</small>
            </div>
            <Toggle
              v-model="runDraft.safeGate"
              :disabled="runDraft.mode !== 'review'"
              :aria-label="t('admin.skills.imports.fields.safeGate')"
            />
          </div>
          <div class="skill-import-composer__actions">
            <button type="button" class="btn btn-ghost" :disabled="mutating" @click="closeRunComposer">
              {{ t('common.cancel') }}
            </button>
            <button type="submit" class="btn btn-primary" :disabled="mutating || !canSubmitRun">
              <Icon name="arrowRight" size="sm" aria-hidden="true" />
              <span class="ml-1.5">{{ mutating
                ? t('admin.skills.imports.actions.starting')
                : t('admin.skills.imports.actions.start') }}</span>
            </button>
          </div>
        </form>
      </section>

      <section
        v-if="activeSection === 'imports'"
        class="skill-import-workspace"
        :aria-label="t('admin.skills.imports.workspace.runs')"
      >
        <aside class="skill-import-queue">
          <div class="skill-import-queue__toolbar">
            <div class="skill-import-search">
              <Icon name="search" size="sm" aria-hidden="true" />
              <input
                v-model="runFilters.search"
                type="search"
                class="input"
                :placeholder="t('admin.skills.imports.filters.searchRuns')"
                :aria-label="t('admin.skills.imports.filters.searchRuns')"
                @input="scheduleRunSearch"
              />
            </div>
            <select
              v-model="runFilters.status"
              class="input skill-import-status-filter"
              :aria-label="t('admin.skills.imports.filters.status')"
              @change="handleRunStatusChange"
            >
              <option value="">{{ t('admin.skills.imports.filters.allStatuses') }}</option>
              <option v-for="status in canonicalRunStatuses" :key="status" :value="status">
                {{ runStatusLabel(status) }}
              </option>
            </select>
          </div>

          <div v-if="runsLoading" class="skill-import-loading" role="status">
            <Icon name="refresh" size="md" class="animate-spin" aria-hidden="true" />
            <span>{{ t('admin.skills.imports.loading.runs') }}</span>
          </div>
          <div v-else-if="runsError" class="skill-import-error" role="alert">
            <Icon name="exclamationCircle" size="md" aria-hidden="true" />
            <p>{{ runsError }}</p>
            <button type="button" @click="retryLoadRuns">{{ t('common.retry') }}</button>
          </div>
          <div v-else-if="runs.length === 0" class="skill-import-empty">
            <Icon name="inbox" size="lg" aria-hidden="true" />
            <h2>{{ t('admin.skills.imports.empty.runsTitle') }}</h2>
            <p>{{ t('admin.skills.imports.empty.runsDescription') }}</p>
            <button type="button" class="btn btn-primary btn-sm" @click="openRunComposer('source')">
              {{ t('admin.skills.imports.actions.newRun') }}
            </button>
          </div>
          <ol v-else class="skill-import-run-list" :aria-label="t('admin.skills.imports.list.runLabel')">
            <li v-for="run in runs" :key="run.id">
              <button
                type="button"
                class="skill-import-run-row"
                :class="{ 'skill-import-run-row--selected': selectedRunID === run.id }"
                :aria-current="selectedRunID === run.id ? 'true' : undefined"
                @click="selectRun(run.id)"
              >
                <span class="skill-import-run-row__topline">
                  <strong>#{{ run.id }} · {{ sourceName(run.source_id) }}</strong>
                  <span class="skill-import-status" :data-status="canonicalRunStatus(run.status)">
                    <span aria-hidden="true"></span>
                    {{ runStatusLabel(run.status) }}
                  </span>
                </span>
                <span class="skill-import-run-row__meta">
                  <span>{{ triggerLabel(run.trigger_type) }}</span>
                  <time :datetime="run.created_at">{{ formatRelativeTime(run.created_at) }}</time>
                </span>
                <span class="skill-import-run-row__progress" aria-hidden="true">
                  <i :style="{ width: `${runPercent(run)}%` }"></i>
                </span>
                <span class="skill-import-run-row__counts">
                  {{ t('admin.skills.imports.list.runCounts', {
                    ready: run.counts?.prepared ?? 0,
                    published: run.counts?.published ?? 0,
                    exceptions: (run.counts?.blocked ?? 0) + (run.counts?.failed ?? 0),
                  }) }}
                </span>
              </button>
            </li>
          </ol>
          <Pagination
            v-if="runPagination.total > runPagination.page_size"
            class="skill-import-queue-pagination"
            :page="runPagination.page"
            :total="runPagination.total"
            :page-size="runPagination.page_size"
            :show-page-size-selector="false"
            @update:page="handleRunPageChange"
          />
        </aside>

        <article class="skill-import-inspector" :aria-busy="detailLoading">
          <div v-if="detailLoading" class="skill-import-loading skill-import-loading--detail" role="status">
            <Icon name="refresh" size="lg" class="animate-spin" aria-hidden="true" />
            <span>{{ t('admin.skills.imports.loading.detail') }}</span>
          </div>
          <div v-else-if="detailError" class="skill-import-error skill-import-error--detail" role="alert">
            <Icon name="exclamationCircle" size="lg" aria-hidden="true" />
            <h2>{{ t('admin.skills.imports.errors.detailTitle') }}</h2>
            <p>{{ detailError }}</p>
            <button type="button" class="btn btn-primary btn-sm" @click="retrySelectedRun">
              {{ t('common.retry') }}
            </button>
          </div>
          <div v-else-if="!selectedRun" class="skill-import-empty skill-import-empty--detail">
            <Icon name="mousePointerClick" size="xl" aria-hidden="true" />
            <h2>{{ t('admin.skills.imports.empty.detailTitle') }}</h2>
            <p>{{ t('admin.skills.imports.empty.detailDescription') }}</p>
          </div>
          <template v-else>
            <div class="skill-import-inspector__header">
              <div>
                <div class="skill-import-inspector__titleline">
                  <h2 id="skill-import-run-title">{{ t('admin.skills.imports.detail.runTitle', { id: selectedRun.id }) }}</h2>
                  <span class="skill-import-status" :data-status="canonicalRunStatus(selectedRun.status)">
                    <span aria-hidden="true"></span>
                    {{ runStatusLabel(selectedRun.status) }}
                  </span>
                </div>
                <p>
                  {{ sourceName(selectedRun.source_id) }} · {{ modeLabel(selectedRun.mode) }} ·
                  {{ formatDateTime(selectedRun.created_at) }}
                </p>
              </div>
              <div class="skill-import-inspector__actions">
                <button
                  v-if="canCancelSelectedRun"
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="mutating"
                  @click="requestRunAction('cancel')"
                >
                  <Icon name="xCircle" size="sm" aria-hidden="true" />
                  <span class="ml-1.5">{{ t('admin.skills.imports.actions.cancelRun') }}</span>
                </button>
                <button
                  v-if="canRetrySelectedRun"
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="mutating"
                  @click="requestRunAction('retry')"
                >
                  <Icon name="refresh" size="sm" aria-hidden="true" />
                  <span class="ml-1.5">{{ t('admin.skills.imports.actions.retryFailed') }}</span>
                </button>
                <button
                  v-if="canPublishSelectedRun"
                  type="button"
                  class="btn btn-primary btn-sm"
                  :disabled="mutating"
                  @click="requestRunAction('publish')"
                >
                  <Icon name="upload" size="sm" aria-hidden="true" />
                  <span class="ml-1.5">{{ t('admin.skills.imports.actions.publishEligible') }}</span>
                </button>
              </div>
            </div>

            <section class="skill-import-progress" :aria-label="t('admin.skills.imports.detail.progress')">
              <div class="skill-import-progress__summary">
                <div>
                  <strong>{{ runPercent(selectedRun) }}%</strong>
                  <span>{{ t('admin.skills.imports.detail.completed') }}</span>
                </div>
                <p>{{ progressSentence(selectedRun) }}</p>
              </div>
              <div
                class="skill-import-progress__track"
                role="progressbar"
                :aria-label="t('admin.skills.imports.detail.progress')"
                :aria-valuenow="runPercent(selectedRun)"
                aria-valuemin="0"
                aria-valuemax="100"
              >
                <i :style="{ width: `${runPercent(selectedRun)}%` }"></i>
              </div>
              <dl class="skill-import-counts">
                <div v-for="metric in runMetrics" :key="metric.key" :data-tone="metric.tone">
                  <dt>{{ metric.label }}</dt>
                  <dd>{{ metric.value }}</dd>
                </div>
              </dl>
            </section>

            <div v-if="selectedRun.last_error_message" class="skill-import-run-alert" role="alert">
              <Icon name="exclamationTriangle" size="sm" aria-hidden="true" />
              <div>
                <strong>{{ selectedRun.last_error_code || t('admin.skills.imports.errors.runFailed') }}</strong>
                <p>{{ selectedRun.last_error_message }}</p>
              </div>
            </div>

            <div class="skill-import-inspector__tabs" role="tablist" :aria-label="t('admin.skills.imports.detail.tabsLabel')">
              <button
                type="button"
                role="tab"
                id="skill-import-items-tab"
                aria-controls="skill-import-items-panel"
                :aria-selected="detailTab === 'items'"
                :tabindex="detailTab === 'items' ? 0 : -1"
                :class="{ active: detailTab === 'items' }"
                @click="detailTab = 'items'"
                @keydown.right.prevent="focusDetailTab('events')"
              >
                {{ t('admin.skills.imports.detail.items', { count: itemPagination.total }) }}
              </button>
              <button
                type="button"
                role="tab"
                id="skill-import-events-tab"
                aria-controls="skill-import-events-panel"
                :aria-selected="detailTab === 'events'"
                :tabindex="detailTab === 'events' ? 0 : -1"
                :class="{ active: detailTab === 'events' }"
                @click="detailTab = 'events'"
                @keydown.left.prevent="focusDetailTab('items')"
              >
                {{ t('admin.skills.imports.detail.events', { count: eventPagination.total }) }}
              </button>
            </div>

            <section
              v-if="detailTab === 'items'"
              id="skill-import-items-panel"
              class="skill-import-detail-pane"
              role="tabpanel"
              aria-labelledby="skill-import-items-tab"
            >
              <div class="skill-import-detail-toolbar">
                <select
                  v-model="itemStatusFilter"
                  class="input"
                  :aria-label="t('admin.skills.imports.filters.itemStatus')"
                  @change="handleItemStatusChange"
                >
                  <option value="all">{{ t('admin.skills.imports.filters.allItems') }}</option>
                  <option v-for="status in canonicalItemStatuses" :key="status" :value="status">
                    {{ itemStatusLabel(status) }}
                  </option>
                </select>
                <span>{{ t('admin.skills.imports.detail.cohortPublishHint') }}</span>
              </div>
              <div v-if="items.length === 0" class="skill-import-empty skill-import-empty--compact">
                <Icon name="inbox" size="md" aria-hidden="true" />
                <p>{{ t('admin.skills.imports.empty.items') }}</p>
              </div>
              <div v-else class="skill-import-items-table-wrap">
                <table class="skill-import-items-table">
                  <caption class="sr-only">{{ t('admin.skills.imports.detail.itemTableCaption') }}</caption>
                  <thead>
                    <tr>
                      <th>{{ t('admin.skills.imports.columns.rank') }}</th>
                      <th>{{ t('admin.skills.imports.columns.skill') }}</th>
                      <th>{{ t('admin.skills.imports.columns.status') }}</th>
                      <th>{{ t('admin.skills.imports.columns.origin') }}</th>
                      <th>{{ t('admin.skills.imports.columns.result') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="item in items" :key="item.id">
                      <td class="skill-import-rank">{{ item.rank ?? '—' }}</td>
                      <td>
                        <strong>{{ item.desired_skill?.display_name || item.upstream_name }}</strong>
                        <code>{{ item.market_slug || item.desired_skill?.slug || item.stable_key.external_id }}</code>
                      </td>
                      <td>
                        <span class="skill-import-status" :data-status="canonicalItemStatus(item.status)">
                          <span aria-hidden="true"></span>
                          {{ itemStatusLabel(item.status) }}
                        </span>
                      </td>
                      <td>
                        <a
                          v-if="safeOriginURL(item.origin_url)"
                          :href="safeOriginURL(item.origin_url)"
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          {{ originHost(item.origin_url) }}
                          <Icon name="externalLink" size="xs" aria-hidden="true" />
                        </a>
                        <span v-else class="skill-import-origin-missing">
                          {{ t('admin.skills.imports.detail.originUnavailable') }}
                        </span>
                      </td>
                      <td>
                        <span v-if="item.error_message" class="skill-import-item-error" :title="item.error_message">
                          {{ item.error_message }}
                        </span>
                        <span v-else-if="item.license_unverified" class="skill-import-item-warning">
                          {{ t('admin.skills.imports.detail.licenseUnverified') }}
                        </span>
                        <span v-else>{{ item.version_id ? `v#${item.version_id}` : '—' }}</span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <Pagination
                v-if="itemPagination.total > itemPagination.page_size"
                :page="itemPagination.page"
                :total="itemPagination.total"
                :page-size="itemPagination.page_size"
                :show-page-size-selector="false"
                @update:page="handleItemPageChange"
              />
            </section>

            <section
              v-else
              id="skill-import-events-panel"
              class="skill-import-detail-pane"
              role="tabpanel"
              aria-labelledby="skill-import-events-tab"
            >
              <ol v-if="events.length" class="skill-import-event-list">
                <li v-for="event in events" :key="event.id" :data-level="event.level">
                  <span class="skill-import-event-list__rail" aria-hidden="true"></span>
                  <div>
                    <span>
                      <strong>{{ event.event_type }}</strong>
                      <time :datetime="event.created_at">{{ formatDateTime(event.created_at) }}</time>
                    </span>
                    <p>{{ event.message }}</p>
                  </div>
                </li>
              </ol>
              <div v-else class="skill-import-empty skill-import-empty--compact">
                <Icon name="clock" size="md" aria-hidden="true" />
                <p>{{ t('admin.skills.imports.empty.events') }}</p>
              </div>
              <Pagination
                v-if="eventPagination.total > eventPagination.page_size"
                :page="eventPagination.page"
                :total="eventPagination.total"
                :page-size="eventPagination.page_size"
                :show-page-size-selector="false"
                @update:page="handleEventPageChange"
              />
            </section>
          </template>
        </article>
      </section>

      <section
        v-else
        class="skill-import-workspace skill-import-workspace--plans"
        :aria-label="t('admin.skills.imports.workspace.plans')"
      >
        <aside class="skill-import-queue">
          <div class="skill-import-library-switch" role="tablist" :aria-label="t('admin.skills.imports.plans.libraryLabel')">
            <button
              type="button"
              role="tab"
              id="skill-import-schedules-tab"
              aria-controls="skill-import-schedules-panel"
              :aria-selected="planLibrary === 'schedules'"
              :tabindex="planLibrary === 'schedules' ? 0 : -1"
              :class="{ active: planLibrary === 'schedules' }"
              @click="selectPlanLibrary('schedules')"
              @keydown.right.prevent="focusPlanLibrary('sources')"
            >
              {{ t('admin.skills.imports.plans.schedules', { count: schedules.length }) }}
            </button>
            <button
              type="button"
              role="tab"
              id="skill-import-sources-tab"
              aria-controls="skill-import-sources-panel"
              :aria-selected="planLibrary === 'sources'"
              :tabindex="planLibrary === 'sources' ? 0 : -1"
              :class="{ active: planLibrary === 'sources' }"
              @click="selectPlanLibrary('sources')"
              @keydown.left.prevent="focusPlanLibrary('schedules')"
            >
              {{ t('admin.skills.imports.plans.sources', { count: sources.length }) }}
            </button>
          </div>

          <div v-if="plansLoading" class="skill-import-loading" role="status">
            <Icon name="refresh" size="md" class="animate-spin" aria-hidden="true" />
            <span>{{ t('admin.skills.imports.loading.plans') }}</span>
          </div>
          <div v-else-if="plansError" class="skill-import-error" role="alert">
            <p>{{ plansError }}</p>
            <button type="button" @click="retryPlans">{{ t('common.retry') }}</button>
          </div>
          <div
            v-else-if="planLibrary === 'schedules'"
            id="skill-import-schedules-panel"
            class="skill-import-library-list"
            role="tabpanel"
            aria-labelledby="skill-import-schedules-tab"
          >
            <button
              v-for="schedule in schedules"
              :key="schedule.id"
              type="button"
              :class="{ active: editorKind === 'schedule' && scheduleDraft.id === schedule.id }"
              @click="openScheduleEditor(schedule)"
            >
              <span class="skill-import-library-list__topline">
                <strong>{{ schedule.name }}</strong>
                <span :class="schedule.enabled ? 'is-enabled' : 'is-disabled'">
                  {{ schedule.enabled
                    ? t('admin.skills.imports.state.enabled')
                    : t('admin.skills.imports.state.disabled') }}
                </span>
              </span>
              <span>{{ sourceName(schedule.source_id) }}</span>
              <code>{{ schedule.cron_expression }} · {{ schedule.timezone }}</code>
              <small>{{ schedule.next_run_at
                ? t('admin.skills.imports.plans.nextRun', { time: formatDateTime(schedule.next_run_at) })
                : t('admin.skills.imports.plans.notScheduled') }}</small>
            </button>
            <div v-if="schedules.length === 0" class="skill-import-empty skill-import-empty--compact">
              <p>{{ t('admin.skills.imports.empty.schedules') }}</p>
            </div>
          </div>
          <div
            v-else
            id="skill-import-sources-panel"
            class="skill-import-library-list"
            role="tabpanel"
            aria-labelledby="skill-import-sources-tab"
          >
            <button
              v-for="source in sources"
              :key="source.id"
              type="button"
              :class="{ active: editorKind === 'source' && sourceDraft.id === source.id }"
              @click="openSourceEditor(source)"
            >
              <span class="skill-import-library-list__topline">
                <strong>{{ source.name }}</strong>
                <span :class="source.enabled ? 'is-enabled' : 'is-disabled'">
                  {{ source.enabled
                    ? t('admin.skills.imports.state.enabled')
                    : t('admin.skills.imports.state.disabled') }}
                </span>
              </span>
              <span>{{ adapterLabel(source.adapter) }}</span>
              <code>{{ source.namespace }}</code>
              <small>{{ t('admin.skills.imports.plans.priority', { value: source.catalog_priority }) }}</small>
            </button>
            <div v-if="sources.length === 0" class="skill-import-empty skill-import-empty--compact">
              <p>{{ t('admin.skills.imports.empty.sources') }}</p>
            </div>
          </div>
        </aside>

        <article class="skill-import-inspector skill-import-plan-editor">
          <SourceEditor
            v-if="editorKind === 'source'"
            v-model="sourceDraft"
            :saving="mutating"
            @save="saveSource"
            @cancel="closeEditor"
          />
          <ScheduleEditor
            v-else-if="editorKind === 'schedule'"
            v-model="scheduleDraft"
            :sources="sources"
            :saving="mutating"
            @save="saveSchedule"
            @cancel="closeEditor"
            @run="runScheduleNow"
          />
          <div v-else class="skill-import-empty skill-import-empty--detail">
            <Icon name="calendar" size="xl" aria-hidden="true" />
            <h2>{{ t('admin.skills.imports.empty.planEditorTitle') }}</h2>
            <p>{{ t('admin.skills.imports.empty.planEditorDescription') }}</p>
          </div>
        </article>
      </section>
    </main>

    <ConfirmDialog
      :show="Boolean(pendingRunAction)"
      :title="pendingRunActionTitle"
      :message="pendingRunActionMessage"
      :confirm-text="pendingRunActionConfirmText"
      :danger="pendingRunAction === 'cancel'"
      @confirm="performRunAction"
      @cancel="pendingRunAction = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import skillImportAPI, {
  type CreateSkillImportScheduleRequest,
  type CreateSkillImportSourceRequest,
  type SkillImportAdapter,
  type SkillImportItemStatus,
  type SkillImportListResponse,
  type SkillImportMode,
  type SkillImportRun,
  type SkillImportRunItem,
  type SkillImportRunStatus,
  type SkillImportSchedule,
  type SkillImportSource,
  type SkillImportEvent,
} from '@/api/admin/skillImport'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import SkillMarketNav from '@/components/admin/skills/SkillMarketNav.vue'
import SourceEditor, { type SkillImportSourceDraft } from '@/components/admin/skills/SourceEditor.vue'
import ScheduleEditor, { type SkillImportScheduleDraft } from '@/components/admin/skills/ScheduleEditor.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import {
  canonicalItemStatus,
  canonicalRunStatus,
  isSkillImportRunActive,
  skillImportPollDelay,
  type CanonicalSkillImportItemStatus,
  type CanonicalSkillImportRunStatus,
} from '@/features/skillImport/status'

type DetailTab = 'items' | 'events'
type PlanLibrary = 'schedules' | 'sources'
type EditorKind = 'schedule' | 'source' | null
type PendingRunAction = 'cancel' | 'retry' | 'publish' | null

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const sources = ref<SkillImportSource[]>([])
const schedules = ref<SkillImportSchedule[]>([])
const runs = ref<SkillImportRun[]>([])
const selectedRun = ref<SkillImportRun | null>(null)
const selectedRunID = ref<number | null>(null)
const items = ref<SkillImportRunItem[]>([])
const events = ref<SkillImportEvent[]>([])
const itemPagination = ref({ total: 0, page: 1, page_size: 100 })
const eventPagination = ref({ total: 0, page: 1, page_size: 100 })
const runPagination = ref({ total: 0, page: 1, page_size: 50 })
const pageLoading = ref(false)
const runsLoading = ref(false)
const plansLoading = ref(false)
const detailLoading = ref(false)
const runsError = ref<string | null>(null)
const plansError = ref<string | null>(null)
const detailError = ref<string | null>(null)
const mutating = ref(false)
const runComposerOpen = ref(false)
const manifestFile = ref<File | null>(null)
const manifestInput = ref<HTMLInputElement | null>(null)
const detailTab = ref<DetailTab>('items')
const itemStatusFilter = ref<SkillImportItemStatus | 'all'>('all')
const pendingRunAction = ref<PendingRunAction>(null)
const planLibrary = ref<PlanLibrary>('schedules')
const editorKind = ref<EditorKind>(null)
const runFilters = reactive({ search: '', status: '' })
const runDraft = reactive({
  kind: 'source' as 'source' | 'upload',
  sourceId: 0,
  mode: 'auto_publish' as SkillImportMode,
  startRank: 1,
  limit: 500,
  safeGate: true,
})
const sourceDraft = ref<SkillImportSourceDraft>(emptySourceDraft())
const scheduleDraft = ref<SkillImportScheduleDraft>(emptyScheduleDraft())

let searchTimer: ReturnType<typeof setTimeout> | null = null
let pollTimer: ReturnType<typeof setTimeout> | null = null
let pollFailureCount = 0

const canonicalRunStatuses: CanonicalSkillImportRunStatus[] = [
  'queued', 'discovering', 'preparing', 'waiting_retry', 'ready', 'awaiting_review',
  'publishing', 'succeeded', 'partial_succeeded', 'failed', 'cancelled',
]
const canonicalItemStatuses: CanonicalSkillImportItemStatus[] = [
  'queued', 'processing', 'ready', 'unchanged', 'blocked', 'failed', 'published', 'skipped', 'cancelled',
]

const activeSection = computed<'imports' | 'schedules'>(() =>
  route.query.tab === 'schedules' ? 'schedules' : 'imports',
)
const enabledSources = computed(() => sources.value.filter((source) => source.enabled))
const composerSources = computed(() => runDraft.kind === 'upload'
  ? enabledSources.value.filter((source) => source.adapter === 'manifest')
  : enabledSources.value)
const hasActiveRun = computed(() => runs.value.some((run) => isSkillImportRunActive(run.status))
  || Boolean(selectedRun.value && isSkillImportRunActive(selectedRun.value.status)))
const canSubmitRun = computed(() => runDraft.sourceId > 0
  && composerSources.value.some((source) => source.id === runDraft.sourceId)
  && runDraft.startRank > 0
  && runDraft.limit > 0
  && (runDraft.kind !== 'upload' || Boolean(manifestFile.value)))
const canCancelSelectedRun = computed(() => Boolean(selectedRun.value && (
  isSkillImportRunActive(selectedRun.value.status)
  || ['ready', 'awaiting_review'].includes(canonicalRunStatus(selectedRun.value.status))
)))
const canRetrySelectedRun = computed(() => Boolean(selectedRun.value
  && ['failed', 'partial_succeeded'].includes(canonicalRunStatus(selectedRun.value.status))))
const publishableItemCount = computed(() => Math.max(
  selectedRun.value?.counts?.prepared ?? 0,
  0,
))
const canPublishSelectedRun = computed(() => Boolean(selectedRun.value
  && selectedRun.value.mode !== 'dry_run'
  && publishableItemCount.value > 0
  && ['ready', 'awaiting_review'].includes(canonicalRunStatus(selectedRun.value.status))))

const runMetrics = computed(() => {
  const counts = selectedRun.value?.counts
  return [
    { key: 'discovered', label: t('admin.skills.imports.metrics.discovered'), value: counts?.discovered ?? 0, tone: 'neutral' },
    { key: 'prepared', label: t('admin.skills.imports.metrics.prepared'), value: counts?.prepared ?? 0, tone: 'violet' },
    { key: 'published', label: t('admin.skills.imports.metrics.published'), value: counts?.published ?? 0, tone: 'success' },
    { key: 'unchanged', label: t('admin.skills.imports.metrics.unchanged'), value: counts?.unchanged ?? 0, tone: 'neutral' },
    { key: 'blocked', label: t('admin.skills.imports.metrics.blocked'), value: counts?.blocked ?? 0, tone: 'warning' },
    { key: 'failed', label: t('admin.skills.imports.metrics.failed'), value: counts?.failed ?? 0, tone: 'danger' },
  ]
})

const pendingRunActionTitle = computed(() => pendingRunAction.value
  ? t(`admin.skills.imports.confirm.${pendingRunAction.value}Title`)
  : '')
const pendingRunActionMessage = computed(() => pendingRunAction.value && selectedRun.value
  ? t(`admin.skills.imports.confirm.${pendingRunAction.value}Message`, {
      id: selectedRun.value.id,
      count: publishableItemCount.value,
    })
  : '')
const pendingRunActionConfirmText = computed(() => pendingRunAction.value
  ? t(`admin.skills.imports.confirm.${pendingRunAction.value}Action`)
  : '')

function emptySourceDraft(): SkillImportSourceDraft {
  return {
    id: null,
    name: '',
    adapter: 'skills_sh',
    namespace: '',
    baseUrl: '',
    sourceConfig: '{}',
    catalogPriority: 0,
    enabled: true,
  }
}

function emptyScheduleDraft(): SkillImportScheduleDraft {
  return {
    id: null,
    sourceId: sources.value.find((source) => source.enabled)?.id ?? sources.value[0]?.id ?? 0,
    name: '',
    enabled: false,
    cronExpression: '0 3 * * *',
    timezone: 'Asia/Shanghai',
    startRank: 1,
    limit: 500,
    concurrency: 2,
    safeGate: true,
    publishPolicy: 'auto_publish',
    metadataPolicy: 'refresh',
    requireAllValid: false,
    allowLicenseUnverified: true,
    maxBlockedItems: 500,
    maxFailedItems: 500,
  }
}

function idempotencyKey(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

async function collectAllPages<T>(
  loader: (params: { page: number; page_size: number }) => Promise<SkillImportListResponse<T>>,
): Promise<T[]> {
  const first = await loader({ page: 1, page_size: 100 })
  const result = [...first.items]
  const pageSize = Math.max(first.page_size, 1)
  const pageCount = Math.max(first.pages, Math.ceil(first.total / pageSize), 1)
  for (let page = 2; page <= pageCount; page += 1) {
    const response = await loader({ page, page_size: pageSize })
    result.push(...response.items)
  }
  return result
}

function clearSelectedRun(): void {
  selectedRunID.value = null
  selectedRun.value = null
  items.value = []
  events.value = []
  itemPagination.value = { ...itemPagination.value, total: 0, page: 1 }
  eventPagination.value = { ...eventPagination.value, total: 0, page: 1 }
}

function resetRunFilters(): void {
  runFilters.search = ''
  runFilters.status = ''
  runPagination.value.page = 1
}

async function loadSourcesAndSchedules(): Promise<void> {
  plansLoading.value = true
  plansError.value = null
  try {
    const [allSources, allSchedules] = await Promise.all([
      collectAllPages((params) => skillImportAPI.listSources(params)),
      collectAllPages((params) => skillImportAPI.listSchedules(params)),
    ])
    sources.value = allSources
    schedules.value = allSchedules
    if (!runDraft.sourceId) runDraft.sourceId = enabledSources.value[0]?.id ?? 0
  } catch (error: unknown) {
    plansError.value = extractApiErrorMessage(error, t('admin.skills.imports.errors.plans'))
  } finally {
    plansLoading.value = false
  }
}

async function loadRuns(showLoading = false): Promise<void> {
  if (showLoading) runsLoading.value = true
  runsError.value = null
  try {
    const response = await skillImportAPI.listRuns({
      page: runPagination.value.page,
      page_size: runPagination.value.page_size,
      search: runFilters.search.trim() || undefined,
      status: runFilters.status || undefined,
    })
    runs.value = response.items
    runPagination.value = { total: response.total, page: response.page, page_size: response.page_size }
    if (selectedRunID.value && !runs.value.some((run) => run.id === selectedRunID.value)) {
      clearSelectedRun()
    }
    if (!selectedRunID.value && runs.value[0]) await selectRun(runs.value[0].id)
  } catch (error: unknown) {
    runsError.value = extractApiErrorMessage(error, t('admin.skills.imports.errors.runs'))
    throw error
  } finally {
    runsLoading.value = false
  }
}

async function selectRun(id: number): Promise<void> {
  if (selectedRunID.value !== id) {
    selectedRunID.value = id
    itemPagination.value.page = 1
    eventPagination.value.page = 1
  }
  await refreshSelectedRun(true)
}

async function refreshSelectedRun(showLoading = false): Promise<void> {
  if (!selectedRunID.value) return
  if (showLoading) detailLoading.value = true
  detailError.value = null
  try {
    const [run] = await Promise.all([
      skillImportAPI.getRun(selectedRunID.value),
      loadRunItems(selectedRunID.value),
      loadRunEvents(selectedRunID.value),
    ])
    selectedRun.value = run
  } catch (error: unknown) {
    detailError.value = extractApiErrorMessage(error, t('admin.skills.imports.errors.detail'))
    throw error
  } finally {
    detailLoading.value = false
  }
}

async function loadRunItems(runID: number): Promise<void> {
  const response = await skillImportAPI.listItems(runID, {
    page: itemPagination.value.page,
    page_size: itemPagination.value.page_size,
    status: itemStatusFilter.value,
  })
  items.value = response.items
  itemPagination.value = { total: response.total, page: response.page, page_size: response.page_size }
}

async function loadRunEvents(runID: number): Promise<void> {
  const response = await skillImportAPI.listEvents(runID, {
    page: eventPagination.value.page,
    page_size: eventPagination.value.page_size,
  })
  events.value = response.items
  eventPagination.value = { total: response.total, page: response.page, page_size: response.page_size }
}

async function loadItemsWithFeedback(): Promise<void> {
  if (!selectedRunID.value) return
  try {
    await loadRunItems(selectedRunID.value)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.skills.imports.errors.items')))
  }
}

async function loadEventsWithFeedback(): Promise<void> {
  if (!selectedRunID.value) return
  try {
    await loadRunEvents(selectedRunID.value)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.skills.imports.errors.events')))
  }
}

function retryLoadRuns(): void {
  void loadRuns(true).catch(() => undefined)
}

function retrySelectedRun(): void {
  void refreshSelectedRun(true).catch(() => undefined)
}

function retryPlans(): void {
  void loadSourcesAndSchedules()
}

async function handleItemStatusChange(): Promise<void> {
  itemPagination.value.page = 1
  await loadItemsWithFeedback()
}

async function handleItemPageChange(page: number): Promise<void> {
  itemPagination.value.page = page
  await loadItemsWithFeedback()
}

async function handleEventPageChange(page: number): Promise<void> {
  eventPagination.value.page = page
  await loadEventsWithFeedback()
}

async function refreshCurrentView(): Promise<void> {
  pageLoading.value = true
  try {
    if (activeSection.value === 'imports') {
      await Promise.all([loadSourcesAndSchedules(), loadRuns(false)])
      if (selectedRunID.value) await refreshSelectedRun(false)
    } else {
      await loadSourcesAndSchedules()
    }
  } catch {
    // Child loaders already expose contextual recovery messages.
  } finally {
    pageLoading.value = false
    schedulePoll()
  }
}

function scheduleRunSearch(): void {
  if (searchTimer) clearTimeout(searchTimer)
  runPagination.value.page = 1
  searchTimer = setTimeout(() => void loadRuns(true).catch(() => undefined), 280)
}

function handleRunStatusChange(): void {
  runPagination.value.page = 1
  void loadRuns(true).catch(() => undefined)
}

function handleRunPageChange(page: number): void {
  runPagination.value.page = page
  void loadRuns(true).catch(() => undefined)
}

function rejectManifestFile(input: HTMLInputElement, errorKey: string): void {
  manifestFile.value = null
  input.value = ''
  appStore.showError(t(errorKey))
}

function openRunComposer(kind: 'source' | 'upload'): void {
  runDraft.kind = kind
  runDraft.safeGate = true
  runComposerOpen.value = true
  manifestFile.value = null
  if (!composerSources.value.some((source) => source.id === runDraft.sourceId)) {
    runDraft.sourceId = composerSources.value[0]?.id ?? 0
  }
}

function closeRunComposer(): void {
  runComposerOpen.value = false
  manifestFile.value = null
  if (manifestInput.value) manifestInput.value.value = ''
}

function onManifestSelected(event: Event): void {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] ?? null
  if (!file) {
    manifestFile.value = null
    return
  }
  const extension = file.name.split('.').pop()?.toLowerCase()
  if (!extension || !['json', 'csv', 'zip'].includes(extension)) {
    rejectManifestFile(input, 'admin.skills.imports.errors.fileType')
    return
  }
  if (file.size === 0) {
    rejectManifestFile(input, 'admin.skills.imports.errors.fileEmpty')
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    rejectManifestFile(input, 'admin.skills.imports.errors.fileTooLarge')
    return
  }
  manifestFile.value = file
}

async function submitRun(): Promise<void> {
  if (!canSubmitRun.value || mutating.value) return
  mutating.value = true
  try {
    const selection = { start_rank: runDraft.startRank, limit: runDraft.limit }
    const runConfig = {
      safe_gate: runDraft.mode === 'review' ? runDraft.safeGate : true,
      auto_publish_gate: {
        require_all_valid: false,
        allow_license_unverified: true,
        max_blocked_items: runDraft.limit,
        max_failed_items: runDraft.limit,
      },
    }
    const run = runDraft.kind === 'upload'
      ? await skillImportAPI.uploadRun({
          source_id: runDraft.sourceId,
          mode: runDraft.mode,
          selection,
          run_config: runConfig,
          publish_policy: runDraft.mode === 'auto_publish' ? 'auto_publish' : 'review',
          metadata_policy: 'refresh',
          file: manifestFile.value!,
        }, idempotencyKey('skill-import-upload'))
      : await skillImportAPI.createRun({
          source_id: runDraft.sourceId,
          mode: runDraft.mode,
          selection,
          run_config: runConfig,
          publish_policy: runDraft.mode === 'auto_publish' ? 'auto_publish' : 'review',
          metadata_policy: 'refresh',
        }, idempotencyKey('skill-import-run'))
    closeRunComposer()
    appStore.showSuccess(t('admin.skills.imports.success.runStarted', { id: run.id }))
    await refreshAfterRunStarted(run)
    schedulePoll()
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.skills.imports.errors.startRun')))
  } finally {
    mutating.value = false
  }
}

function requestRunAction(action: Exclude<PendingRunAction, null>): void {
  pendingRunAction.value = action
}

async function performRunAction(): Promise<void> {
  const action = pendingRunAction.value
  const run = selectedRun.value
  if (!action || !run || mutating.value) return
  pendingRunAction.value = null
  mutating.value = true
  try {
    if (action === 'cancel') {
      selectedRun.value = await skillImportAPI.cancelRun(run.id, idempotencyKey('skill-import-cancel'))
    } else if (action === 'retry') {
      selectedRun.value = await skillImportAPI.retryFailed(run.id, idempotencyKey('skill-import-retry'))
    } else {
      await skillImportAPI.publishRun(
        run.id,
        { item_ids: [] },
        idempotencyKey('skill-import-publish'),
      )
    }
    appStore.showSuccess(t(`admin.skills.imports.success.${action}`))
    await refreshAfterMutation()
    schedulePoll()
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t(`admin.skills.imports.errors.${action}`)))
  } finally {
    mutating.value = false
  }
}

function selectPlanLibrary(library: PlanLibrary): void {
  planLibrary.value = library
  editorKind.value = null
  void router.replace({ query: { ...route.query, tab: 'schedules', library } })
}

async function focusPlanLibrary(library: PlanLibrary): Promise<void> {
  selectPlanLibrary(library)
  await nextTick()
  document.getElementById(`skill-import-${library}-tab`)?.focus()
}

async function focusDetailTab(tab: DetailTab): Promise<void> {
  detailTab.value = tab
  await nextTick()
  document.getElementById(`skill-import-${tab}-tab`)?.focus()
}

function openSourceEditor(source?: SkillImportSource): void {
  planLibrary.value = 'sources'
  editorKind.value = 'source'
  sourceDraft.value = source
    ? {
        id: source.id,
        name: source.name,
        adapter: source.adapter,
        namespace: source.namespace,
        baseUrl: source.base_url,
        sourceConfig: JSON.stringify(source.source_config ?? {}, null, 2),
        catalogPriority: source.catalog_priority ?? 0,
        enabled: source.enabled,
      }
    : emptySourceDraft()
}

function openScheduleEditor(schedule?: SkillImportSchedule): void {
  planLibrary.value = 'schedules'
  editorKind.value = 'schedule'
  if (!schedule) {
    scheduleDraft.value = emptyScheduleDraft()
    return
  }
  const selection = schedule.selection ?? {}
  const runConfig = schedule.run_config ?? {}
  const gate = runConfig.auto_publish_gate ?? {
    require_all_valid: false,
    allow_license_unverified: true,
    max_blocked_items: 500,
    max_failed_items: 500,
  }
  scheduleDraft.value = {
    id: schedule.id,
    sourceId: schedule.source_id,
    name: schedule.name,
    enabled: schedule.enabled,
    cronExpression: schedule.cron_expression,
    timezone: schedule.timezone,
    startRank: numberValue(selection.start_rank, 1),
    limit: numberValue(selection.limit, 500),
    concurrency: numberValue(runConfig.concurrency, 2),
    safeGate: booleanValue(runConfig.safe_gate, true),
    publishPolicy: schedule.publish_policy,
    metadataPolicy: schedule.metadata_policy,
    requireAllValid: booleanValue(gate.require_all_valid, false),
    allowLicenseUnverified: booleanValue(gate.allow_license_unverified, true),
    maxBlockedItems: numberValue(gate.max_blocked_items, 500),
    maxFailedItems: numberValue(gate.max_failed_items, 500),
  }
}

function closeEditor(): void {
  editorKind.value = null
}

async function saveSource(): Promise<void> {
  if (mutating.value) return
  let sourceConfig: Record<string, unknown>
  try {
    const parsed: unknown = JSON.parse(sourceDraft.value.sourceConfig)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('not an object')
    sourceConfig = parsed as Record<string, unknown>
  } catch {
    appStore.showError(t('admin.skills.imports.errors.invalidJson'))
    return
  }
  const request: CreateSkillImportSourceRequest = {
    name: sourceDraft.value.name.trim(),
    adapter: sourceDraft.value.adapter,
    namespace: sourceDraft.value.namespace.trim(),
    base_url: sourceDraft.value.baseUrl.trim(),
    source_config: sourceConfig,
    catalog_priority: sourceDraft.value.catalogPriority,
    enabled: sourceDraft.value.enabled,
  }
  mutating.value = true
  try {
    const source = sourceDraft.value.id
      ? await skillImportAPI.updateSource(sourceDraft.value.id, request)
      : await skillImportAPI.createSource(request, idempotencyKey('skill-import-source'))
    appStore.showSuccess(t('admin.skills.imports.success.sourceSaved'))
    await loadSourcesAndSchedules()
    openSourceEditor(source)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.skills.imports.errors.saveSource')))
  } finally {
    mutating.value = false
  }
}

async function saveSchedule(): Promise<void> {
  if (mutating.value) return
  const request = scheduleRequest()
  mutating.value = true
  try {
    const schedule = scheduleDraft.value.id
      ? await skillImportAPI.updateSchedule(scheduleDraft.value.id, request)
      : await skillImportAPI.createSchedule(request, idempotencyKey('skill-import-schedule-create'))
    appStore.showSuccess(t('admin.skills.imports.success.scheduleSaved'))
    await loadSourcesAndSchedules()
    openScheduleEditor(schedule)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.skills.imports.errors.saveSchedule')))
  } finally {
    mutating.value = false
  }
}

function scheduleRequest(): CreateSkillImportScheduleRequest {
  return {
    source_id: scheduleDraft.value.sourceId,
    name: scheduleDraft.value.name.trim(),
    enabled: scheduleDraft.value.enabled,
    cron_expression: scheduleDraft.value.cronExpression.trim(),
    timezone: scheduleDraft.value.timezone.trim(),
    selection: {
      start_rank: scheduleDraft.value.startRank,
      limit: scheduleDraft.value.limit,
    },
    run_config: {
      safe_gate: scheduleDraft.value.publishPolicy === 'review' ? scheduleDraft.value.safeGate : true,
      concurrency: scheduleDraft.value.concurrency,
      auto_publish_gate: {
        require_all_valid: scheduleDraft.value.requireAllValid,
        allow_license_unverified: scheduleDraft.value.allowLicenseUnverified,
        max_blocked_items: scheduleDraft.value.maxBlockedItems,
        max_failed_items: scheduleDraft.value.maxFailedItems,
      },
    },
    publish_policy: scheduleDraft.value.publishPolicy,
    metadata_policy: scheduleDraft.value.metadataPolicy,
  }
}

async function runScheduleNow(): Promise<void> {
  const source = sources.value.find((candidate) => candidate.id === scheduleDraft.value.sourceId)
  if (!scheduleDraft.value.id || !source?.enabled || mutating.value) return
  mutating.value = true
  try {
    await skillImportAPI.updateSchedule(scheduleDraft.value.id, scheduleRequest())
    const run = await skillImportAPI.runSchedule(
      scheduleDraft.value.id,
      idempotencyKey('skill-import-schedule'),
    )
    appStore.showSuccess(t('admin.skills.imports.success.runStarted', { id: run.id }))
    await refreshAfterRunStarted(run, true)
    schedulePoll()
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.skills.imports.errors.startRun')))
  } finally {
    mutating.value = false
  }
}

async function refreshAfterRunStarted(run: SkillImportRun, navigateToRuns = false): Promise<void> {
  try {
    if (navigateToRuns) await router.push('/admin/skills/imports')
    resetRunFilters()
    await loadRuns(false)
    await selectRun(run.id)
  } catch {
    // The run was created successfully; contextual loaders already expose refresh recovery.
    selectedRunID.value = run.id
    selectedRun.value = run
  }
}

async function refreshAfterMutation(): Promise<void> {
  const results = await Promise.allSettled([loadRuns(false), refreshSelectedRun(false)])
  if (results.some((result) => result.status === 'rejected')) {
    appStore.showError(t('admin.skills.imports.errors.refreshAfterMutation'))
  }
}

function schedulePoll(): void {
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = null
  if (document.visibilityState === 'hidden' || !hasActiveRun.value || activeSection.value !== 'imports') return
  pollTimer = setTimeout(() => void pollRuns(), skillImportPollDelay(pollFailureCount))
}

async function pollRuns(): Promise<void> {
  if (document.visibilityState === 'hidden') return
  try {
    await loadRuns(false)
    if (selectedRunID.value) await refreshSelectedRun(false)
    pollFailureCount = 0
  } catch {
    pollFailureCount += 1
  } finally {
    schedulePoll()
  }
}

function onVisibilityChange(): void {
  if (document.visibilityState === 'hidden') {
    if (pollTimer) clearTimeout(pollTimer)
    pollTimer = null
    return
  }
  if (hasActiveRun.value) void pollRuns()
}

function runStatusLabel(status: SkillImportRunStatus | CanonicalSkillImportRunStatus): string {
  return t(`admin.skills.imports.runStatus.${canonicalRunStatus(status as SkillImportRunStatus)}`)
}

function itemStatusLabel(status: SkillImportItemStatus | CanonicalSkillImportItemStatus): string {
  return t(`admin.skills.imports.itemStatus.${canonicalItemStatus(status as SkillImportItemStatus)}`)
}

function triggerLabel(trigger: string): string {
  const supported = ['manual', 'scheduled', 'retry', 'bootstrap']
  return supported.includes(trigger)
    ? t(`admin.skills.imports.trigger.${trigger}`)
    : trigger
}

function modeLabel(mode: string): string {
  const key = mode === 'auto_publish' ? 'autoPublish' : mode === 'dry_run' ? 'dryRun' : 'review'
  return t(`admin.skills.imports.mode.${key}`)
}

function adapterLabel(adapter: SkillImportAdapter): string {
  return t(`admin.skills.imports.adapter.${adapter}`)
}

function sourceName(sourceID: number): string {
  return sources.value.find((source) => source.id === sourceID)?.name
    ?? t('admin.skills.imports.list.unknownSource', { id: sourceID })
}

function runPercent(run: SkillImportRun): number {
  if (typeof run.progress?.percent === 'number') return Math.max(0, Math.min(100, Math.round(run.progress.percent)))
  if (['succeeded', 'partial_succeeded', 'failed', 'cancelled'].includes(canonicalRunStatus(run.status))) return 100
  const total = Math.max(run.counts?.requested ?? 0, run.counts?.discovered ?? 0, 1)
  const completed = (run.counts?.prepared ?? 0)
    + (run.counts?.blocked ?? 0) + (run.counts?.failed ?? 0) + (run.counts?.skipped ?? 0)
  return Math.max(0, Math.min(99, Math.round((completed / total) * 100)))
}

function progressSentence(run: SkillImportRun): string {
  const counts = run.counts
  return t('admin.skills.imports.detail.progressSentence', {
    discovered: counts?.discovered ?? 0,
    requested: counts?.requested ?? 0,
    exceptions: (counts?.blocked ?? 0) + (counts?.failed ?? 0),
  })
}

function formatRelativeTime(value: string): string {
  const timestamp = new Date(value).getTime()
  if (!Number.isFinite(timestamp)) return value
  const seconds = Math.round((timestamp - Date.now()) / 1000)
  const formatter = new Intl.RelativeTimeFormat(locale.value, { numeric: 'auto' })
  if (Math.abs(seconds) < 60) return formatter.format(seconds, 'second')
  const minutes = Math.round(seconds / 60)
  if (Math.abs(minutes) < 60) return formatter.format(minutes, 'minute')
  const hours = Math.round(minutes / 60)
  if (Math.abs(hours) < 24) return formatter.format(hours, 'hour')
  return formatter.format(Math.round(hours / 24), 'day')
}

function safeOriginURL(value: string): string {
  try {
    const url = new URL(value)
    return url.protocol === 'https:' && url.hostname ? url.href : ''
  } catch {
    return ''
  }
}

function originHost(value: string): string {
  const url = safeOriginURL(value)
  return url ? new URL(url).host : t('admin.skills.imports.detail.originUnavailable')
}

function numberValue(value: unknown, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function booleanValue(value: unknown, fallback: boolean): boolean {
  return typeof value === 'boolean' ? value : fallback
}

watch(activeSection, async (section) => {
  closeRunComposer()
  if (section === 'schedules') {
    planLibrary.value = route.query.library === 'sources' ? 'sources' : 'schedules'
    await loadSourcesAndSchedules()
  } else {
    await refreshCurrentView()
  }
  schedulePoll()
})

watch(() => runDraft.mode, (mode) => {
  if (mode !== 'review') runDraft.safeGate = true
})

watch(() => scheduleDraft.value.publishPolicy, (policy) => {
  if (policy !== 'review') scheduleDraft.value.safeGate = true
})

onMounted(async () => {
  document.addEventListener('visibilitychange', onVisibilityChange)
  planLibrary.value = route.query.library === 'sources' ? 'sources' : 'schedules'
  await refreshCurrentView()
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onVisibilityChange)
  if (searchTimer) clearTimeout(searchTimer)
  if (pollTimer) clearTimeout(pollTimer)
})
</script>

<style scoped src="./SkillImportsView.css"></style>
