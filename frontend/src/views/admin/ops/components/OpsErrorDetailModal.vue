<template>
  <Teleport to="body">
    <Transition name="ops-error-detail-drawer">
      <aside
        v-if="show"
        ref="drawerRef"
        class="ops-error-detail-drawer"
        role="dialog"
        :aria-labelledby="titleId"
        :aria-describedby="subtitleId"
        tabindex="-1"
        data-testid="ops-error-detail-drawer"
      >
        <header class="ops-error-detail-drawer__header">
          <div class="ops-error-detail-drawer__heading">
            <div v-if="detail" class="ops-error-detail-drawer__badges" aria-hidden="true">
              <span class="ops-error-detail-drawer__status" :class="statusClass">
                {{ detail.status_code }}
              </span>
              <span class="ops-error-detail-drawer__badge" :class="phasePresentation?.className">
                {{ phaseLabel }}
              </span>
              <span
                class="ops-error-detail-drawer__badge"
                :class="errorCategoryBadgeClass(categoryCode)"
              >
                {{ categoryLabel }}
              </span>
              <span
                v-if="priority"
                class="ops-error-detail-drawer__priority"
                :class="getSeverityClass(priority)"
              >
                {{ priority }}
              </span>
            </div>
            <h2 :id="titleId">{{ title }}</h2>
            <p :id="subtitleId" :title="requestId || undefined">
              {{ requestId || t('admin.ops.errorDetail.noErrorSelected') }}
            </p>
          </div>

          <div class="ops-error-detail-drawer__header-actions">
            <span class="ops-error-detail-drawer__readonly">
              <Icon name="lock" size="xs" aria-hidden="true" />
              {{ t('admin.ops.errorLog.readOnly') }}
            </span>
            <button
              type="button"
              class="ops-error-detail-drawer__icon-button"
              :title="t('common.close')"
              :aria-label="t('common.close')"
              data-testid="ops-error-detail-drawer-close"
              @click="close"
            >
              <Icon name="x" size="sm" :stroke-width="2" />
            </button>
          </div>
        </header>

        <div
          class="ops-error-detail-drawer__content"
          :aria-busy="loading"
          aria-live="polite"
        >
          <div v-if="loading" class="ops-error-detail-drawer__state" role="status">
            <span class="ops-error-detail-drawer__spinner" aria-hidden="true" />
            <span>{{ t('admin.ops.errorDetail.loading') }}</span>
          </div>

          <div v-else-if="!detail" class="ops-error-detail-drawer__state">
            <Icon name="document" size="lg" aria-hidden="true" />
            <span>{{ emptyText }}</span>
          </div>

          <template v-else>
            <section class="ops-error-detail-drawer__summary">
              <div class="ops-error-detail-drawer__section-heading">
                <h3>{{ t('admin.ops.errorDetail.message') }}</h3>
                <button
                  v-if="detail.message"
                  type="button"
                  class="ops-error-detail-drawer__copy-button"
                  :aria-label="`${t('common.copy')} ${t('admin.ops.errorDetail.message')}`"
                  :title="t('common.copy')"
                  @click="copyValue(detail.message)"
                >
                  <Icon name="copy" size="xs" :stroke-width="2" />
                  <span>{{ t('common.copy') }}</span>
                </button>
              </div>
              <p class="ops-error-detail-drawer__message">
                {{ detail.message || '—' }}
              </p>

              <div class="ops-error-detail-drawer__request-id-row">
                <span>{{ t('admin.ops.errorDetail.requestId') }}</span>
                <code :title="requestId || undefined">{{ requestId || '—' }}</code>
                <button
                  v-if="requestId"
                  type="button"
                  class="ops-error-detail-drawer__icon-button ops-error-detail-drawer__icon-button--compact"
                  :aria-label="`${t('common.copy')} ${t('admin.ops.errorDetail.requestId')}`"
                  :title="t('common.copy')"
                  @click="copyValue(requestId)"
                >
                  <Icon name="copy" size="xs" :stroke-width="2" />
                </button>
              </div>
            </section>

            <section class="ops-error-detail-drawer__section">
              <h3>{{ t('admin.ops.errorDetail.classification') }}</h3>
              <dl class="ops-error-detail-drawer__fact-grid">
                <div>
                  <dt>{{ t('admin.ops.errorDetail.status') }}</dt>
                  <dd>{{ detail.status_code }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.phase') }}</dt>
                  <dd>{{ phaseLabel }}</dd>
                </div>
                <div>
                  <dt>{{ t('usage.errors.category') }}</dt>
                  <dd>{{ categoryLabel }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorLog.priority') }}</dt>
                  <dd>{{ detail.severity || '—' }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.classificationKeys.owner') }}</dt>
                  <dd>{{ ownerLabel }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.classificationKeys.source') }}</dt>
                  <dd>{{ sourceLabel }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.resolution') }}</dt>
                  <dd>
                    {{ detail.resolved ? t('admin.ops.errorDetails.resolved') : t('admin.ops.errorDetails.unresolved') }}
                  </dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.businessLimited') }}</dt>
                  <dd>{{ detail.is_business_limited ? t('common.yes') : t('common.no') }}</dd>
                </div>
              </dl>
            </section>

            <section class="ops-error-detail-drawer__section">
              <h3>{{ t('admin.ops.errorDetail.basicInfo') }}</h3>
              <dl class="ops-error-detail-drawer__fact-list">
                <div>
                  <dt>{{ t('admin.ops.errorDetail.time') }}</dt>
                  <dd>{{ formatDateTime(detail.created_at) }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.user') }}</dt>
                  <dd>{{ userDisplay }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorLog.apiKey') }}</dt>
                  <dd>
                    {{ apiKeyDisplay }}
                    <span v-if="detail.api_key_deleted" class="ops-error-detail-drawer__deleted-key">
                      {{ t('admin.ops.errorLog.keyDeletedBadge') }}
                    </span>
                  </dd>
                </div>
                <div v-if="detail.api_key_prefix">
                  <dt>{{ t('admin.ops.errorDetail.apiKeyPrefix') }}</dt>
                  <dd><code>{{ detail.api_key_prefix }}</code></dd>
                </div>
                <div v-if="detail.attempted_key_prefix">
                  <dt>{{ t('admin.ops.errorDetail.attemptedKeyPrefix') }}</dt>
                  <dd><code>{{ detail.attempted_key_prefix }}</code></dd>
                </div>
                <div v-if="detail.deleted_key_owner_email">
                  <dt>{{ t('admin.ops.errorDetail.deletedKeyOwner') }}</dt>
                  <dd>
                    {{ detail.deleted_key_owner_email }}
                    <span v-if="detail.deleted_key_name"> · {{ detail.deleted_key_name }}</span>
                    <span class="ops-error-detail-drawer__deleted-key">
                      {{ t('admin.ops.errorDetail.keyDeletedBadge') }}
                    </span>
                  </dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.account') }}</dt>
                  <dd>{{ accountDisplay }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.platform') }}</dt>
                  <dd>{{ detail.platform || '—' }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.group') }}</dt>
                  <dd>{{ groupDisplay }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.model') }}</dt>
                  <dd class="ops-error-detail-drawer__model-value">{{ modelDisplay || '—' }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.inboundEndpoint') }}</dt>
                  <dd><code>{{ detail.inbound_endpoint || '—' }}</code></dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.upstreamEndpoint') }}</dt>
                  <dd><code>{{ detail.upstream_endpoint || '—' }}</code></dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.requestPath') }}</dt>
                  <dd><code>{{ detail.request_path || '—' }}</code></dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorDetail.requestType') }}</dt>
                  <dd>{{ requestTypeLabel }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.errorLog.ip') }}</dt>
                  <dd><code>{{ detail.client_ip || '—' }}</code></dd>
                </div>
                <div>
                  <dt>{{ t('usage.userAgent') }}</dt>
                  <dd>{{ detail.user_agent || '—' }}</dd>
                </div>
              </dl>
            </section>

            <section v-if="timingRows.length" class="ops-error-detail-drawer__section">
              <h3>{{ t('admin.ops.errorDetail.timings') }}</h3>
              <dl class="ops-error-detail-drawer__fact-grid">
                <div v-for="row in timingRows" :key="row.label">
                  <dt>{{ row.label }}</dt>
                  <dd>{{ row.value }}</dd>
                </div>
              </dl>
            </section>

            <section class="ops-error-detail-drawer__section">
              <h3>{{ t('admin.ops.errorDetail.responseBody') }}</h3>
              <pre class="ops-error-detail-drawer__code"><code>{{ prettyJSON(primaryResponseBody) }}</code></pre>
            </section>

            <section v-if="showUpstreamList" class="ops-error-detail-drawer__section">
              <div class="ops-error-detail-drawer__section-heading">
                <h3>{{ t('admin.ops.errorDetails.upstreamErrors') }}</h3>
                <span v-if="correlatedUpstreamLoading" class="ops-error-detail-drawer__loading-label">
                  {{ t('common.loading') }}
                </span>
              </div>

              <p
                v-if="!correlatedUpstreamLoading && !correlatedUpstreamErrors.length"
                class="ops-error-detail-drawer__empty-related"
              >
                {{ t('common.noData') }}
              </p>

              <ol v-else class="ops-error-detail-drawer__related-list">
                <li v-for="(event, index) in correlatedUpstreamErrors" :key="event.id">
                  <div class="ops-error-detail-drawer__related-heading">
                    <div>
                      <strong>#{{ index + 1 }}</strong>
                      <span>{{ event.type || phaseLabelFor(event) }}</span>
                    </div>
                    <span class="ops-error-detail-drawer__related-status" :class="statusCodeBadgeClass(event.status_code)">
                      {{ event.status_code }}
                    </span>
                  </div>
                  <p v-if="event.message">{{ event.message }}</p>
                  <div class="ops-error-detail-drawer__related-request">
                    <code>{{ eventIdentifier(event) || '—' }}</code>
                    <button
                      v-if="getUpstreamResponsePreview(event)"
                      type="button"
                      class="ops-error-detail-drawer__expand-button"
                      :aria-expanded="expandedUpstreamDetailIds.has(event.id)"
                      @click="toggleUpstreamDetail(event.id)"
                    >
                      <Icon
                        :name="expandedUpstreamDetailIds.has(event.id) ? 'chevronDown' : 'chevronRight'"
                        size="xs"
                        :stroke-width="2"
                      />
                      {{
                        expandedUpstreamDetailIds.has(event.id)
                          ? t('admin.ops.errorDetail.responsePreview.collapse')
                          : t('admin.ops.errorDetail.responsePreview.expand')
                      }}
                    </button>
                  </div>
                  <pre
                    v-if="expandedUpstreamDetailIds.has(event.id)"
                    class="ops-error-detail-drawer__code ops-error-detail-drawer__code--compact"
                  ><code>{{ prettyJSON(getUpstreamResponsePreview(event)) }}</code></pre>
                </li>
              </ol>
            </section>
          </template>
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores'
import { opsAPI, type OpsErrorDetail, type OpsErrorLog } from '@/api/admin/ops'
import { formatDateTime } from '@/utils/format'
import { mapErrorCategory } from '@/utils/errorCategory'
import { statusCodeBadgeClass } from '@/utils/errorBadges'
import { getSeverityClass } from '../utils/opsFormatters'
import { resolvePrimaryResponseBody, resolveUpstreamPayload } from '../utils/errorDetailResponse'
import {
  errorCategoryBadgeClass,
  errorOwnerLabelKey,
  errorPhasePresentation,
  normalizedErrorPriority
} from '../utils/errorPresentation'

interface Props {
  show: boolean
  errorId: number | null
  errorType?: 'request' | 'upstream'
}

interface Emits {
  (e: 'update:show', value: boolean): void
}

const props = withDefaults(defineProps<Props>(), {
  errorType: 'request'
})
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const titleId = 'ops-error-detail-drawer-title'
const subtitleId = 'ops-error-detail-drawer-subtitle'
const drawerRef = ref<HTMLElement | null>(null)
const loading = ref(false)
const detail = ref<OpsErrorDetail | null>(null)
const correlatedUpstream = ref<OpsErrorDetail[]>([])
const correlatedUpstreamLoading = ref(false)
const expandedUpstreamDetailIds = ref(new Set<number>())
let detailRequestToken = 0
let correlatedRequestToken = 0
let previousActiveElement: HTMLElement | null = null

const showUpstreamList = computed(() => props.errorType === 'request')
const correlatedUpstreamErrors = computed(() => correlatedUpstream.value)
const requestId = computed(() => eventIdentifier(detail.value))
const primaryResponseBody = computed(() => resolvePrimaryResponseBody(detail.value, props.errorType))
const priority = computed(() => normalizedErrorPriority(detail.value?.severity))
const categoryCode = computed(() => mapErrorCategory(detail.value?.phase, detail.value?.type))
const categoryLabel = computed(() => t(`usage.errors.categories.${categoryCode.value}`))
const phasePresentation = computed(() => detail.value ? errorPhasePresentation(detail.value) : null)
const phaseLabel = computed(() => {
  const presentation = phasePresentation.value
  if (!presentation) return t('common.unknown')
  return presentation.labelKey
    ? t(presentation.labelKey)
    : (presentation.fallback || t('common.unknown'))
})
const statusClass = computed(() => statusCodeBadgeClass(detail.value?.status_code ?? 0))
const ownerLabel = computed(() => {
  const owner = String(detail.value?.error_owner || '').trim()
  const labelKey = errorOwnerLabelKey(owner)
  return labelKey ? t(labelKey) : (owner || t('common.unknown'))
})
const sourceLabel = computed(() => {
  const source = String(detail.value?.error_source || '').trim()
  return source === 'upstream_http'
    ? t('admin.ops.errorDetail.source.upstream_http')
    : (source || t('common.unknown'))
})
const title = computed(() => props.errorId
  ? t('admin.ops.errorDetail.titleWithId', { id: String(props.errorId) })
  : t('admin.ops.errorDetail.title'))
const emptyText = computed(() => t('admin.ops.errorDetail.noErrorSelected'))

const userDisplay = computed(() => {
  const current = detail.value
  if (!current) return '—'
  const userId = current.user_id ?? current.deleted_key_owner_user_id
  const email = String(current.user_email || current.deleted_key_owner_email || '').trim()
  if (email && userId != null) return `${email} · #${userId}`
  if (email) return email
  return userId != null ? `#${userId}` : '—'
})

const apiKeyDisplay = computed(() => {
  const current = detail.value
  if (!current) return '—'
  if (current.api_key_name && current.api_key_id != null) return `${current.api_key_name} · #${current.api_key_id}`
  return current.api_key_name || (current.api_key_id != null ? `#${current.api_key_id}` : '—')
})

const accountDisplay = computed(() => {
  const current = detail.value
  if (!current) return '—'
  if (current.account_name && current.account_id != null) return `${current.account_name} · #${current.account_id}`
  return current.account_name || (current.account_id != null ? `#${current.account_id}` : '—')
})

const groupDisplay = computed(() => {
  const current = detail.value
  if (!current) return '—'
  if (current.group_name && current.group_id != null) return `${current.group_name} · #${current.group_id}`
  return current.group_name || (current.group_id != null ? `#${current.group_id}` : '—')
})

const modelDisplay = computed(() => {
  const current = detail.value
  if (!current) return ''
  const requested = String(current.requested_model || '').trim()
  const upstream = String(current.upstream_model || '').trim()
  if (requested && upstream && requested !== upstream) return `${requested} → ${upstream}`
  return upstream || requested || String(current.model || '').trim()
})

const requestTypeLabel = computed(() => formatRequestTypeLabel(detail.value?.request_type))

const timingRows = computed(() => {
  const current = detail.value
  if (!current) return []
  return [
    { label: t('admin.ops.errorDetail.auth'), value: current.auth_latency_ms },
    { label: t('admin.ops.errorDetail.routing'), value: current.routing_latency_ms },
    { label: t('admin.ops.errorDetail.upstream'), value: current.upstream_latency_ms },
    { label: t('admin.ops.errorDetail.response'), value: current.response_latency_ms }
  ]
    .filter((row): row is { label: string; value: number } => typeof row.value === 'number')
    .map((row) => ({ label: row.label, value: `${row.value} ms` }))
})

function eventIdentifier(log: Pick<OpsErrorLog, 'request_id' | 'client_request_id'> | null): string {
  return String(log?.request_id || log?.client_request_id || '').trim()
}

function phaseLabelFor(log: OpsErrorLog): string {
  const presentation = errorPhasePresentation(log)
  return presentation.labelKey
    ? t(presentation.labelKey)
    : (presentation.fallback || t('common.unknown'))
}

function formatRequestTypeLabel(type: number | null | undefined): string {
  switch (type) {
    case 1: return t('admin.ops.errorDetail.requestTypeSync')
    case 2: return t('admin.ops.errorDetail.requestTypeStream')
    case 3: return t('admin.ops.errorDetail.requestTypeWs')
    default: return t('admin.ops.errorDetail.requestTypeUnknown')
  }
}

function getUpstreamResponsePreview(event: OpsErrorDetail): string {
  return resolveUpstreamPayload(event) || String(event.error_body || '').trim()
}

function toggleUpstreamDetail(id: number) {
  const next = new Set(expandedUpstreamDetailIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedUpstreamDetailIds.value = next
}

function prettyJSON(raw?: string): string {
  if (!raw) return t('common.noData')
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

function close() {
  emit('update:show', false)
}

function copyValue(value: string) {
  void copyToClipboard(value)
}

async function fetchDetail(id: number, requestToken: number) {
  loading.value = true
  detail.value = null
  try {
    const response = props.errorType === 'upstream'
      ? await opsAPI.getUpstreamErrorDetail(id)
      : await opsAPI.getRequestErrorDetail(id)
    if (requestToken === detailRequestToken) detail.value = response
  } catch (error: any) {
    if (requestToken !== detailRequestToken) return
    detail.value = null
    appStore.showError(error?.message || t('admin.ops.failedToLoadErrorDetail'))
  } finally {
    if (requestToken === detailRequestToken) loading.value = false
  }
}

async function fetchCorrelatedUpstreamErrors(requestErrorId: number, requestToken: number) {
  correlatedUpstreamLoading.value = true
  try {
    const response = await opsAPI.listRequestErrorUpstreamErrors(
      requestErrorId,
      { page: 1, page_size: 100, view: 'all' },
      { include_detail: true }
    )
    if (requestToken === correlatedRequestToken) correlatedUpstream.value = response.items || []
  } catch (error) {
    if (requestToken !== correlatedRequestToken) return
    console.error('[OpsErrorDetailDrawer] Failed to load correlated upstream errors', error)
    correlatedUpstream.value = []
  } finally {
    if (requestToken === correlatedRequestToken) correlatedUpstreamLoading.value = false
  }
}

function restoreFocus() {
  const target = previousActiveElement
  previousActiveElement = null
  if (!target?.isConnected) return
  const activeElement = document.activeElement as HTMLElement | null
  const focusIsInsideDrawer = !!activeElement && !!drawerRef.value?.contains(activeElement)
  if (activeElement !== document.body && !focusIsInsideDrawer) return
  try {
    target.focus({ preventScroll: true })
  } catch {
    target.focus()
  }
}

async function focusDrawer() {
  await nextTick()
  if (!props.show) return
  const target = drawerRef.value
  if (!target) return
  try {
    target.focus({ preventScroll: true })
  } catch {
    target.focus()
  }
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (!props.show || event.key !== 'Escape') return
  const activeModal = document.querySelector<HTMLElement>('[role="dialog"][aria-modal="true"]')
  if (activeModal && activeModal !== drawerRef.value) return
  event.preventDefault()
  event.stopPropagation()
  close()
}

watch(
  () => props.show,
  (isOpen, wasOpen) => {
    if (isOpen && !wasOpen && typeof document !== 'undefined') {
      previousActiveElement = document.activeElement as HTMLElement | null
      void focusDrawer()
    } else if (!isOpen && wasOpen) {
      restoreFocus()
    }
  },
  { immediate: true }
)

watch(
  () => [props.show, props.errorId, props.errorType] as const,
  ([isOpen, id]) => {
    if (!isOpen || typeof id !== 'number' || id <= 0) {
      detailRequestToken += 1
      correlatedRequestToken += 1
      loading.value = false
      correlatedUpstreamLoading.value = false
      detail.value = null
      correlatedUpstream.value = []
      expandedUpstreamDetailIds.value = new Set()
      return
    }

    expandedUpstreamDetailIds.value = new Set()
    const nextDetailToken = ++detailRequestToken
    void fetchDetail(id, nextDetailToken)

    if (props.errorType === 'request') {
      const nextCorrelatedToken = ++correlatedRequestToken
      void fetchCorrelatedUpstreamErrors(id, nextCorrelatedToken)
    } else {
      correlatedRequestToken += 1
      correlatedUpstream.value = []
      correlatedUpstreamLoading.value = false
    }
  },
  { immediate: true }
)

onMounted(() => {
  document.addEventListener('keydown', handleDocumentKeydown, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleDocumentKeydown, true)
  if (props.show) restoreFocus()
})
</script>

<style scoped>
.ops-error-detail-drawer {
  position: fixed;
  top: 0.75rem;
  right: 0.75rem;
  bottom: 0.75rem;
  z-index: 90;
  display: flex;
  width: min(42rem, calc(100vw - 1.5rem));
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
  border-radius: var(--lx-clay-radius-overlay);
  outline: none;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-overlay);
  font-family: var(--lx-clay-font-ui);
}

.ops-error-detail-drawer:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 32%, transparent);
  outline-offset: -3px;
}

.ops-error-detail-drawer__header,
.ops-error-detail-drawer__section-heading,
.ops-error-detail-drawer__header-actions,
.ops-error-detail-drawer__badges,
.ops-error-detail-drawer__request-id-row,
.ops-error-detail-drawer__related-heading,
.ops-error-detail-drawer__related-request {
  display: flex;
  align-items: center;
}

.ops-error-detail-drawer__header {
  flex: 0 0 auto;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 1.125rem 1.25rem;
  background: var(--lx-clay-surface);
}

.ops-error-detail-drawer__heading {
  min-width: 0;
}

.ops-error-detail-drawer__heading h2,
.ops-error-detail-drawer__section h3,
.ops-error-detail-drawer__summary h3 {
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
}

.ops-error-detail-drawer__heading h2 {
  margin-top: 0.45rem;
  font-size: 1.125rem;
  font-weight: 850;
  line-height: 1.35;
  letter-spacing: -0.02em;
}

.ops-error-detail-drawer__heading p {
  max-width: 32rem;
  margin: 0.2rem 0 0;
  overflow: hidden;
  color: var(--lx-clay-text-muted);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.6875rem;
  line-height: 1.1rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-error-detail-drawer__badges {
  flex-wrap: wrap;
  gap: 0.35rem;
}

.ops-error-detail-drawer__status,
.ops-error-detail-drawer__badge,
.ops-error-detail-drawer__priority,
.ops-error-detail-drawer__readonly,
.ops-error-detail-drawer__deleted-key,
.ops-error-detail-drawer__related-status {
  display: inline-flex;
  min-height: 1.5rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  padding: 0.2rem 0.48rem;
  font-size: 0.6875rem;
  font-weight: 750;
  line-height: 1rem;
  white-space: nowrap;
}

.ops-error-detail-drawer__status {
  min-width: 2.8rem;
  font-variant-numeric: tabular-nums;
}

.ops-error-detail-drawer__header-actions {
  flex: 0 0 auto;
  gap: 0.5rem;
}

.ops-error-detail-drawer__readonly {
  gap: 0.3rem;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-error-detail-drawer__icon-button,
.ops-error-detail-drawer__copy-button,
.ops-error-detail-drawer__expand-button {
  display: inline-flex;
  min-width: 2.75rem;
  min-height: 2.75rem;
  align-items: center;
  justify-content: center;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text-secondary);
  transition: color 150ms ease, background-color 150ms ease, transform 150ms ease;
}

.ops-error-detail-drawer__icon-button:hover,
.ops-error-detail-drawer__copy-button:hover,
.ops-error-detail-drawer__expand-button:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-error-detail-drawer__icon-button:active,
.ops-error-detail-drawer__copy-button:active,
.ops-error-detail-drawer__expand-button:active {
  transform: scale(0.97);
}

.ops-error-detail-drawer__icon-button:focus-visible,
.ops-error-detail-drawer__copy-button:focus-visible,
.ops-error-detail-drawer__expand-button:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 30%, transparent);
  outline-offset: 2px;
}

.ops-error-detail-drawer__icon-button--compact {
  margin: -0.5rem;
}

.ops-error-detail-drawer__content {
  min-height: 0;
  flex: 1 1 auto;
  overflow-y: auto;
  overscroll-behavior: contain;
  background: var(--lx-clay-surface);
}

.ops-error-detail-drawer__state {
  display: flex;
  min-height: 18rem;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  color: var(--lx-clay-text-muted);
  font-size: 0.875rem;
}

.ops-error-detail-drawer__spinner {
  width: 1.5rem;
  height: 1.5rem;
  border: 2px solid var(--lx-clay-border-strong);
  border-top-color: var(--lx-clay-accent);
  border-radius: 50%;
  animation: ops-error-detail-spin 800ms linear infinite;
}

.ops-error-detail-drawer__summary,
.ops-error-detail-drawer__section {
  padding: 1.25rem;
}

.ops-error-detail-drawer__summary {
  background: color-mix(in srgb, var(--lx-clay-recessed) 78%, var(--lx-clay-surface));
}

.ops-error-detail-drawer__section {
  border-top: 1px solid var(--lx-clay-border);
}

.ops-error-detail-drawer__section-heading {
  justify-content: space-between;
  gap: 1rem;
}

.ops-error-detail-drawer__section h3,
.ops-error-detail-drawer__summary h3 {
  font-size: 0.875rem;
  font-weight: 800;
  line-height: 1.25rem;
  letter-spacing: -0.01em;
}

.ops-error-detail-drawer__copy-button,
.ops-error-detail-drawer__expand-button {
  gap: 0.35rem;
  padding: 0 0.7rem;
  font-size: 0.75rem;
  font-weight: 700;
}

.ops-error-detail-drawer__message {
  margin: 0.75rem 0 0;
  color: var(--lx-clay-text);
  font-size: 0.9375rem;
  font-weight: 650;
  line-height: 1.55;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.ops-error-detail-drawer__request-id-row {
  min-width: 0;
  gap: 0.65rem;
  margin-top: 1rem;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
}

.ops-error-detail-drawer__request-id-row code {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  color: var(--lx-clay-text-secondary);
  font-family: var(--lx-clay-font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-error-detail-drawer__fact-grid,
.ops-error-detail-drawer__fact-list {
  margin: 0.85rem 0 0;
}

.ops-error-detail-drawer__fact-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.85rem 1.25rem;
}

.ops-error-detail-drawer__fact-grid > div,
.ops-error-detail-drawer__fact-list > div {
  min-width: 0;
}

.ops-error-detail-drawer__fact-list > div {
  display: grid;
  grid-template-columns: minmax(7rem, 0.32fr) minmax(0, 1fr);
  gap: 1rem;
  padding: 0.62rem 0;
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-error-detail-drawer__fact-list > div:last-child {
  border-bottom: 0;
}

.ops-error-detail-drawer dt {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 700;
  line-height: 1.15rem;
}

.ops-error-detail-drawer dd {
  min-width: 0;
  margin: 0.18rem 0 0;
  color: var(--lx-clay-text);
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.3rem;
  overflow-wrap: anywhere;
}

.ops-error-detail-drawer__fact-list dd {
  margin-top: 0;
  text-align: right;
}

.ops-error-detail-drawer dd code {
  font-family: var(--lx-clay-font-mono);
  font-size: 0.75rem;
}

.ops-error-detail-drawer__deleted-key {
  margin-left: 0.4rem;
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.ops-error-detail-drawer__model-value {
  font-family: var(--lx-clay-font-mono);
  font-size: 0.75rem !important;
}

.ops-error-detail-drawer__code {
  max-height: 30rem;
  margin: 0.85rem 0 0;
  overflow: auto;
  border-radius: var(--lx-clay-radius-ops);
  padding: 1rem;
  color: #f8f5fc;
  background: var(--lx-clay-code-canvas);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.75rem;
  line-height: 1.55;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.ops-error-detail-drawer__code--compact {
  max-height: 15rem;
  padding: 0.75rem;
}

.ops-error-detail-drawer__loading-label,
.ops-error-detail-drawer__empty-related {
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
}

.ops-error-detail-drawer__empty-related {
  margin: 0.85rem 0 0;
}

.ops-error-detail-drawer__related-list {
  display: grid;
  gap: 0;
  margin: 0.75rem 0 0;
  padding: 0;
  list-style: none;
}

.ops-error-detail-drawer__related-list > li {
  padding: 0.9rem 0;
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-error-detail-drawer__related-list > li:last-child {
  border-bottom: 0;
}

.ops-error-detail-drawer__related-heading,
.ops-error-detail-drawer__related-request {
  justify-content: space-between;
  gap: 0.75rem;
}

.ops-error-detail-drawer__related-heading > div {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
}

.ops-error-detail-drawer__related-heading strong {
  color: var(--lx-clay-text);
  font-size: 0.75rem;
}

.ops-error-detail-drawer__related-heading span,
.ops-error-detail-drawer__related-list p,
.ops-error-detail-drawer__related-request code {
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
}

.ops-error-detail-drawer__related-list p {
  margin: 0.55rem 0 0;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.ops-error-detail-drawer__related-request {
  min-width: 0;
  margin-top: 0.45rem;
}

.ops-error-detail-drawer__related-request code {
  min-width: 0;
  overflow: hidden;
  font-family: var(--lx-clay-font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-error-detail-drawer-enter-active,
.ops-error-detail-drawer-leave-active {
  transition: transform 220ms cubic-bezier(0.22, 1, 0.36, 1), opacity 180ms ease-out;
}

.ops-error-detail-drawer-enter-from,
.ops-error-detail-drawer-leave-to {
  opacity: 0.7;
  transform: translateX(2rem);
}

@keyframes ops-error-detail-spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 640px) {
  .ops-error-detail-drawer {
    inset: 0;
    width: 100%;
    border-radius: 0;
  }

  .ops-error-detail-drawer__header,
  .ops-error-detail-drawer__summary,
  .ops-error-detail-drawer__section {
    padding-right: 1rem;
    padding-left: 1rem;
  }

  .ops-error-detail-drawer__readonly {
    width: 2.75rem;
    padding: 0;
    overflow: hidden;
    color: var(--lx-clay-accent-deep);
  }

  .ops-error-detail-drawer__readonly :deep(svg) {
    flex: 0 0 auto;
  }

  .ops-error-detail-drawer__readonly {
    font-size: 0;
  }

  .ops-error-detail-drawer__fact-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-error-detail-drawer__fact-list > div {
    grid-template-columns: minmax(5.5rem, 0.36fr) minmax(0, 1fr);
    gap: 0.75rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-error-detail-drawer-enter-active,
  .ops-error-detail-drawer-leave-active,
  .ops-error-detail-drawer__icon-button,
  .ops-error-detail-drawer__copy-button,
  .ops-error-detail-drawer__expand-button {
    transition-duration: 0.01ms;
  }

  .ops-error-detail-drawer__spinner {
    animation-duration: 1.5s;
  }
}
</style>
