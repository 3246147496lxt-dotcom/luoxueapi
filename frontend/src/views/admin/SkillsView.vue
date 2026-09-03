<template>
  <AppLayout variant="home-clay">
    <TablePageLayout class="skills-page" table-surface="card">
      <template #header>
        <AdminPageHeader
          :title="t('admin.skills.title')"
          :description="t('admin.skills.description')"
        >
          <template #meta>
            <span>{{ t('admin.skills.resultCount', { count: pagination.total }) }}</span>
          </template>
          <template #secondary-actions>
            <button type="button" class="btn btn-secondary" :disabled="loading || configLoading" @click="refreshAll">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading || configLoading }" aria-hidden="true" />
              <span class="ml-1.5">{{ t('admin.skills.refresh') }}</span>
            </button>
          </template>
          <template #primary-actions>
            <button type="button" class="btn btn-primary" @click="openCreate">
              <Icon name="plus" size="sm" aria-hidden="true" />
              <span class="ml-1.5">{{ t('admin.skills.newSkill') }}</span>
            </button>
          </template>
        </AdminPageHeader>
      </template>

      <template #actions>
        <div class="skills-local-navigation">
          <SkillMarketNav active="catalog" />
        </div>
        <section class="skills-workflow-strip" aria-labelledby="skills-workflow-title">
          <div class="skills-workflow-strip__icon" aria-hidden="true">
            <Icon name="cube" size="md" />
          </div>
          <div>
            <h2 id="skills-workflow-title">{{ t('admin.skills.workflowTitle') }}</h2>
            <p>{{ t('admin.skills.workflowDescription') }}</p>
          </div>
          <div
            class="skills-market-gate"
            :class="{ 'skills-market-gate--enabled': marketplaceEnabled && !appStore.backendModeEnabled }"
          >
            <div>
              <span>{{ t('admin.skills.marketplace.title') }}</span>
              <strong>{{ appStore.backendModeEnabled
                ? t('admin.skills.marketplace.backendModeBlocked')
                : marketplaceEnabled
                  ? t('admin.skills.marketplace.enabled')
                  : t('admin.skills.marketplace.disabled') }}</strong>
              <p v-if="configError" role="alert">{{ configError }}</p>
              <small v-else-if="appStore.backendModeEnabled">
                {{ t('admin.skills.marketplace.backendModeHint') }}
              </small>
              <small v-else>{{ marketplaceEnabled
                ? t('admin.skills.marketplace.hintEnabled')
                : t('admin.skills.marketplace.hintDisabled') }}</small>
            </div>
            <Toggle
              :model-value="marketplaceEnabled"
              :disabled="configLoading"
              :aria-label="t('admin.skills.marketplace.title')"
              @update:model-value="updateMarketplaceConfig"
            />
          </div>
          <div class="skills-workflow-strip__stages" aria-hidden="true">
            <span><Icon name="edit" size="xs" /> 1</span>
            <i></i>
            <span><Icon name="upload" size="xs" /> 2</span>
            <i></i>
            <span><Icon name="shield" size="xs" /> 3</span>
            <i></i>
            <span><Icon name="globe" size="xs" /> 4</span>
          </div>
        </section>
      </template>

      <template #filters>
        <div class="skills-filters">
          <div class="skills-search">
            <Icon name="search" size="sm" aria-hidden="true" />
            <input
              v-model="searchQuery"
              type="search"
              class="input"
              :placeholder="t('admin.skills.searchPlaceholder')"
              @input="scheduleSearch"
              @keydown.enter.prevent="resetAndLoad"
            />
          </div>
          <Select
            v-model="statusFilter"
            class="skills-filter-select"
            :options="statusOptions"
            @change="resetAndLoad"
          />
          <Select
            v-model="featuredFilter"
            class="skills-filter-select"
            :options="featuredOptions"
            @change="resetAndLoad"
          />
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="skills"
          :loading="loading"
          :error="loadError"
          row-key="id"
          mobile-primary-key="skill"
          :mobile-visible-keys="['version', 'status', 'downloads']"
        >
          <template #cell-skill="{ row }">
            <div class="skills-name-cell">
              <div class="skills-name-cell__icon">
                <img v-if="safeIconUrl(row.icon)" :src="safeIconUrl(row.icon)" alt="" />
                <Icon v-else name="cube" size="sm" aria-hidden="true" />
              </div>
              <div class="skills-name-cell__copy">
                <div>
                  <span>{{ row.display_name }}</span>
                  <Icon
                    v-if="row.featured"
                    name="sparkles"
                    size="xs"
                    class="skills-featured-icon"
                    :title="t('admin.skills.featured')"
                  />
                </div>
                <code>${{ row.slug }}</code>
                <p>{{ row.summary }}</p>
              </div>
            </div>
          </template>

          <template #cell-category="{ row }">
            <div class="skills-category-cell">
              <span>{{ row.category || '—' }}</span>
              <p v-if="row.tags?.length">{{ row.tags.slice(0, 2).join(' · ') }}</p>
            </div>
          </template>

          <template #cell-version="{ row }">
            <div v-if="listVersion(row)" class="skills-version-cell">
              <strong>v{{ listVersion(row)?.version }}</strong>
              <span
                :class="listVersion(row)?.validation_report.valid
                  ? 'skills-validation-state--ready'
                  : 'skills-validation-state--blocked'"
              >
                <Icon
                  :name="listVersion(row)?.validation_report.valid ? 'checkCircle' : 'exclamationTriangle'"
                  size="xs"
                  aria-hidden="true"
                />
                {{ listVersion(row)?.validation_report.valid
                  ? t('admin.skills.validationState.ready')
                  : t('admin.skills.validationState.blocked') }}
              </span>
            </div>
            <span v-else class="skills-muted-state">
              {{ t('admin.skills.noVersion') }}
            </span>
          </template>

          <template #cell-status="{ row }">
            <SkillStatusBadge :status="row.status" />
          </template>

          <template #cell-featured="{ row }">
            <span :class="row.featured ? 'skills-featured-label' : 'skills-muted-state'">
              {{ row.featured ? t('admin.skills.featured') : t('admin.skills.notFeatured') }}
            </span>
          </template>

          <template #cell-downloads="{ row }">
            <span class="skills-download-count">{{ formatNumber(row.download_count) }}</span>
          </template>

          <template #cell-updated_at="{ row }">
            <time class="skills-updated-at" :datetime="row.updated_at">
              {{ formatDateTime(row.updated_at) }}
            </time>
          </template>

          <template #cell-actions="{ row }">
            <div class="skills-row-actions" @click.stop>
              <button
                type="button"
                class="skills-row-actions__primary"
                @click="openEditor(row)"
              >
                <Icon name="edit" size="sm" aria-hidden="true" />
                <span>{{ t('admin.skills.edit') }}</span>
              </button>
              <button
                v-if="canQuickPublish(row)"
                type="button"
                class="skills-row-actions__icon skills-row-actions__icon--publish"
                :title="t('admin.skills.publish')"
                :disabled="operatingId === row.id"
                @click="requestPublish(row)"
              >
                <Icon name="upload" size="sm" aria-hidden="true" />
              </button>
              <button
                v-if="row.status !== 'archived'"
                type="button"
                class="skills-row-actions__icon skills-row-actions__icon--archive"
                :title="t('admin.skills.archive')"
                :disabled="operatingId === row.id"
                @click="requestArchive(row)"
              >
                <Icon name="ban" size="sm" aria-hidden="true" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.skills.emptyTitle')"
              :description="t('admin.skills.emptyDescription')"
              :action-text="t('admin.skills.newSkill')"
              @action="openCreate"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <ConfirmDialog
      :show="Boolean(pendingAction)"
      :title="confirmTitle"
      :message="confirmMessage"
      :confirm-text="confirmButtonText"
      :danger="pendingAction?.kind === 'archive'"
      @confirm="performPendingAction"
      @cancel="pendingAction = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import skillsAPI, {
  type AdminSkill,
  type AdminSkillVersion,
  type SkillStatus,
} from '@/api/admin/skills'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import SkillStatusBadge from '@/components/admin/skills/SkillStatusBadge.vue'
import SkillMarketNav from '@/components/admin/skills/SkillMarketNav.vue'

type FeaturedFilter = 'all' | boolean
type PendingAction = { kind: 'publish' | 'archive'; skill: AdminSkill }

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const skills = ref<AdminSkill[]>([])
const loading = ref(false)
const loadError = ref<string | null>(null)
const marketplaceEnabled = ref(false)
const configLoading = ref(false)
const configError = ref<string | null>(null)
const operatingId = ref<number | null>(null)
const searchQuery = ref('')
const statusFilter = ref<SkillStatus | 'all'>('all')
const featuredFilter = ref<FeaturedFilter>('all')
const pagination = ref({ page: 1, page_size: 20, total: 0 })
const pendingAction = ref<PendingAction | null>(null)
let searchTimer: ReturnType<typeof setTimeout> | null = null

const columns = computed<Column[]>(() => [
  { key: 'skill', label: t('admin.skills.columns.skill') },
  { key: 'category', label: t('admin.skills.columns.category') },
  { key: 'version', label: t('admin.skills.columns.version') },
  { key: 'status', label: t('admin.skills.columns.status') },
  { key: 'featured', label: t('admin.skills.columns.featured') },
  { key: 'downloads', label: t('admin.skills.columns.downloads') },
  { key: 'updated_at', label: t('admin.skills.columns.updatedAt') },
  { key: 'actions', label: t('common.actions'), class: 'text-right' },
])

const statusOptions = computed(() => [
  { value: 'all', label: t('admin.skills.filters.allStatuses') },
  { value: 'draft', label: t('admin.skills.status.draft') },
  { value: 'published', label: t('admin.skills.status.published') },
  { value: 'archived', label: t('admin.skills.status.archived') },
])

const featuredOptions = computed(() => [
  { value: 'all', label: t('admin.skills.filters.allFeatured') },
  { value: true, label: t('admin.skills.filters.featuredOnly') },
  { value: false, label: t('admin.skills.filters.regularOnly') },
])

const confirmTitle = computed(() => {
  const action = pendingAction.value
  return action?.kind === 'archive'
    ? t('admin.skills.archiveTitle')
    : t('admin.skills.publishTitle')
})

const confirmMessage = computed(() => {
  const action = pendingAction.value
  if (!action) return ''
  if (action.kind === 'archive') {
    return t('admin.skills.archiveConfirm', { name: action.skill.display_name })
  }
  return t('admin.skills.publishConfirm', {
    name: action.skill.display_name,
    version: listVersion(action.skill)?.version ?? '—',
  })
})

const confirmButtonText = computed(() => {
  const action = pendingAction.value
  return action?.kind === 'archive' ? t('admin.skills.archive') : t('admin.skills.publish')
})

async function loadSkills(): Promise<void> {
  loading.value = true
  loadError.value = null
  try {
    const response = await skillsAPI.list({
      status: statusFilter.value,
      search: searchQuery.value,
      featured: featuredFilter.value,
      page: pagination.value.page,
      page_size: pagination.value.page_size,
    })
    skills.value = response.items
    pagination.value = {
      page: response.page,
      page_size: response.page_size,
      total: response.total,
    }
  } catch (error: unknown) {
    loadError.value = extractApiErrorMessage(error, t('admin.skills.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadConfig(): Promise<void> {
  configLoading.value = true
  configError.value = null
  try {
    const config = await skillsAPI.getConfig()
    marketplaceEnabled.value = config.enabled === true
  } catch (error: unknown) {
    configError.value = extractApiErrorMessage(error, t('admin.skills.marketplace.loadFailed'))
  } finally {
    configLoading.value = false
  }
}

function refreshAll(): void {
  void Promise.all([loadSkills(), loadConfig()])
}

function scheduleSearch(): void {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(resetAndLoad, 280)
}

function resetAndLoad(): void {
  pagination.value.page = 1
  void loadSkills()
}

function handlePageChange(page: number): void {
  pagination.value.page = page
  void loadSkills()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.value.page = 1
  pagination.value.page_size = pageSize
  void loadSkills()
}

function openCreate(): void {
  void router.push('/admin/skills/new')
}

function openEditor(skill: AdminSkill): void {
  void router.push(`/admin/skills/${skill.id}/edit`)
}

function canQuickPublish(skill: AdminSkill): boolean {
  return skill.status === 'draft'
    && Boolean(listVersion(skill)?.validation_report.valid)
    && Boolean(skill.summary.trim())
    && Boolean(skill.description.trim())
    && Boolean(skill.category.trim())
}

function listVersion(skill: AdminSkill): AdminSkillVersion | null {
  return skill.current_version ?? skill.latest_version ?? null
}

function requestPublish(skill: AdminSkill): void {
  if (!listVersion(skill)?.validation_report.valid) {
    appStore.showError(t('admin.skills.noPublishableVersion'))
    return
  }
  pendingAction.value = { kind: 'publish', skill }
}

function requestArchive(skill: AdminSkill): void {
  pendingAction.value = { kind: 'archive', skill }
}

async function updateMarketplaceConfig(enabled: boolean): Promise<void> {
  if (enabled === marketplaceEnabled.value || configLoading.value) return
  const previous = marketplaceEnabled.value
  marketplaceEnabled.value = enabled
  configLoading.value = true
  configError.value = null
  try {
    const updated = await skillsAPI.updateConfig(enabled)
    marketplaceEnabled.value = updated.enabled === true
    try {
      await appStore.refreshPublicSettingsAfterMutation()
    } catch {
      // The config mutation already succeeded. A public-settings refresh is
      // best-effort and must not turn that success into a false failure.
    }
    appStore.showSuccess(updated.enabled
      ? t('admin.skills.marketplace.enabledSuccess')
      : t('admin.skills.marketplace.disabledSuccess'))
  } catch (error: unknown) {
    marketplaceEnabled.value = previous
    const message = extractApiErrorMessage(error, t('admin.skills.marketplace.updateFailed'))
    configError.value = message
    appStore.showError(message)
  } finally {
    configLoading.value = false
  }
}

async function performPendingAction(): Promise<void> {
  const action = pendingAction.value
  if (!action || operatingId.value !== null) return
  pendingAction.value = null

  operatingId.value = action.skill.id
  try {
    const updated = action.kind === 'archive'
      ? await skillsAPI.archive(action.skill.id)
      : await skillsAPI.publish(action.skill.id, listVersion(action.skill)!.id)
    replaceSkill(updated)
    appStore.showSuccess(action.kind === 'archive'
      ? t('admin.skills.archiveSuccess')
      : t('admin.skills.publishSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(
      error,
      action.kind === 'archive'
        ? t('admin.skills.archiveFailed')
        : t('admin.skills.publishFailed'),
    ))
  } finally {
    operatingId.value = null
  }
}

function replaceSkill(updated: AdminSkill): void {
  const index = skills.value.findIndex((item) => item.id === updated.id)
  if (index >= 0) skills.value.splice(index, 1, updated)
}

function safeIconUrl(value: string): string {
  const icon = value?.trim() ?? ''
  return icon.startsWith('/') || icon.startsWith('https://') ? icon : ''
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat().format(value || 0)
}

onMounted(refreshAll)

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<style scoped src="./SkillsView.css"></style>
