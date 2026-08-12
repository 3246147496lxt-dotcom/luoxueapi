<template>
  <AppLayout variant="home-clay">
    <div class="skill-editor-page" data-admin-page-kind="form">
      <AdminPageHeader
        :title="isCreate
          ? t('admin.skills.editor.createTitle')
          : t('admin.skills.editor.editTitle', { name: draft.display_name || skill?.slug || 'Skill' })"
        :description="isCreate
          ? t('admin.skills.editor.createDescription')
          : t('admin.skills.editor.editDescription')"
      >
        <template #meta>
          <SkillStatusBadge v-if="skill" :status="skill.status" />
          <span class="skill-editor-save-state" :class="{ 'skill-editor-save-state--dirty': isDirty }">
            <i aria-hidden="true"></i>
            {{ isDirty ? t('admin.skills.editor.unsaved') : t('admin.skills.editor.saved') }}
          </span>
          <span v-if="skill?.current_version" class="skill-editor-version-meta">
            {{ t('admin.skills.currentVersion', { version: skill.current_version.version }) }}
          </span>
        </template>
        <template #secondary-actions>
          <button type="button" class="btn btn-secondary" @click="goBack">
            <Icon name="arrowLeft" size="sm" aria-hidden="true" />
            <span class="ml-1.5">{{ t('admin.skills.editor.back') }}</span>
          </button>
          <button
            v-if="skillId && skill?.status !== 'archived'"
            type="button"
            class="btn btn-ghost skill-editor-archive-action"
            :disabled="isOperating || isDirty"
            @click="requestArchive"
          >
            <Icon name="ban" size="sm" aria-hidden="true" />
            <span class="ml-1.5">{{ t('admin.skills.archive') }}</span>
          </button>
        </template>
        <template #primary-actions>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="saving || loading || skill?.status === 'archived'"
            data-testid="save-skill-draft"
            @click="saveDraft()"
          >
            <Icon name="document" size="sm" aria-hidden="true" />
            <span class="ml-1.5">
              {{ saving ? t('admin.skills.editor.saving') : t('admin.skills.editor.saveDraft') }}
            </span>
          </button>
          <button
            v-if="skill?.status !== 'published' && skill?.status !== 'archived'"
            type="button"
            class="btn btn-primary"
            :disabled="!canPublish || isOperating"
            data-testid="publish-skill"
            @click="requestPublish"
          >
            <Icon name="upload" size="sm" aria-hidden="true" />
            <span class="ml-1.5">
              {{ publishing ? t('admin.skills.editor.publishing') : t('admin.skills.editor.publish') }}
            </span>
          </button>
        </template>
      </AdminPageHeader>

      <div
        v-if="skill?.status === 'archived'"
        class="skill-editor-archived-notice"
        role="status"
      >
        <Icon name="ban" size="sm" aria-hidden="true" />
        {{ t('admin.skills.editor.archivedNotice') }}
      </div>

      <div v-if="validationErrors.length" ref="validationAlert" class="skill-editor-validation" role="alert" tabindex="-1">
        <Icon name="exclamationTriangle" size="sm" aria-hidden="true" />
        <div>
          <p>{{ t('admin.skills.editor.validationTitle') }}</p>
          <ul>
            <li v-for="message in validationErrors" :key="message">{{ message }}</li>
          </ul>
        </div>
      </div>

      <section v-if="loading" class="skill-editor-loading" aria-live="polite">
        <div></div>
        <div>
          <span v-for="index in 5" :key="index"></span>
        </div>
      </section>

      <section v-else-if="loadError" class="skill-editor-load-error">
        <Icon name="exclamationCircle" size="xl" aria-hidden="true" />
        <h2>{{ t('admin.skills.detailLoadFailed') }}</h2>
        <p>{{ loadError }}</p>
        <button type="button" class="btn btn-primary" @click="loadSkill">
          {{ t('admin.skills.editor.retry') }}
        </button>
      </section>

      <div v-else class="skill-editor-workspace">
        <aside class="skill-editor-nav" :aria-label="t('admin.skills.editor.navigation')">
          <div class="skill-editor-nav__heading">
            <Icon name="sparkles" size="sm" aria-hidden="true" />
            <h2>{{ t('admin.skills.editor.navigation') }}</h2>
          </div>
          <nav>
            <button
              v-for="section in editorSections"
              :key="section.id"
              type="button"
              :class="{ 'skill-editor-nav__item--active': activeSection === section.id }"
              @click="scrollToSection(section.id)"
            >
              <span class="skill-editor-nav__icon">
                <Icon :name="section.icon" size="sm" aria-hidden="true" />
              </span>
              <span class="skill-editor-nav__copy">
                <strong>{{ section.label }}</strong>
                <small>{{ section.complete
                  ? t('admin.skills.editor.complete')
                  : t('admin.skills.editor.needsAttention') }}</small>
              </span>
              <Icon
                :name="section.complete ? 'checkCircle' : 'exclamationCircle'"
                size="sm"
                :class="section.complete ? 'skill-editor-nav__complete' : 'skill-editor-nav__incomplete'"
                aria-hidden="true"
              />
            </button>
          </nav>
        </aside>

        <main class="skill-editor-content">
          <section id="skill-editor-basics" class="skill-editor-section" aria-labelledby="skill-editor-basics-title">
            <header class="skill-editor-section__heading">
              <div class="skill-editor-section__icon"><Icon name="edit" size="md" aria-hidden="true" /></div>
              <div>
                <h2 id="skill-editor-basics-title">{{ t('admin.skills.editor.sections.basics') }}</h2>
                <p>{{ t('admin.skills.editor.sections.basicsHint') }}</p>
              </div>
            </header>

            <div class="skill-editor-fields skill-editor-fields--two">
              <div>
                <label for="skill-display-name" class="input-label">
                  {{ t('admin.skills.editor.fields.displayName') }}
                </label>
                <input
                  id="skill-display-name"
                  v-model="draft.display_name"
                  type="text"
                  class="input"
                  :disabled="isArchived"
                  :placeholder="t('admin.skills.editor.fields.displayNamePlaceholder')"
                />
              </div>
              <div>
                <label for="skill-slug" class="input-label">{{ t('admin.skills.editor.fields.slug') }}</label>
                <input
                  id="skill-slug"
                  v-model="draft.slug"
                  type="text"
                  class="input skill-editor-mono-input"
                  :disabled="slugLocked || isArchived"
                  :placeholder="t('admin.skills.editor.fields.slugPlaceholder')"
                />
                <p class="input-hint">{{ t('admin.skills.editor.fields.slugHint') }}</p>
              </div>
              <div>
                <label for="skill-category" class="input-label">
                  {{ t('admin.skills.editor.fields.category') }}
                </label>
                <input
                  id="skill-category"
                  v-model="draft.category"
                  type="text"
                  class="input"
                  :disabled="isArchived"
                  :placeholder="t('admin.skills.editor.fields.categoryPlaceholder')"
                />
              </div>
              <div>
                <label for="skill-sort-order" class="input-label">
                  {{ t('admin.skills.editor.fields.sortOrder') }}
                </label>
                <input
                  id="skill-sort-order"
                  v-model.number="draft.sort_order"
                  type="number"
                  class="input"
                  :disabled="isArchived"
                  min="0"
                  step="1"
                />
              </div>
              <div class="skill-editor-fields__wide">
                <label for="skill-icon" class="input-label">{{ t('admin.skills.editor.fields.icon') }}</label>
                <input
                  id="skill-icon"
                  v-model="draft.icon"
                  type="url"
                  class="input"
                  :disabled="isArchived"
                  :placeholder="t('admin.skills.editor.fields.iconPlaceholder')"
                />
              </div>
              <div class="skill-editor-fields__wide">
                <label for="skill-origin-url" class="input-label">
                  {{ t('admin.skills.editor.fields.originUrl') }}
                </label>
                <input
                  id="skill-origin-url"
                  v-model="draft.origin_url"
                  type="url"
                  class="input"
                  :disabled="isArchived"
                  :placeholder="t('admin.skills.editor.fields.originUrlPlaceholder')"
                />
                <p class="input-hint">{{ t('admin.skills.editor.fields.originUrlHint') }}</p>
              </div>
              <div class="skill-editor-fields__wide">
                <label for="skill-source-url" class="input-label">
                  {{ t('admin.skills.editor.fields.sourceUrl') }}
                </label>
                <input
                  id="skill-source-url"
                  v-model="draft.source_url"
                  type="url"
                  class="input"
                  :disabled="isArchived"
                  :placeholder="t('admin.skills.editor.fields.sourceUrlPlaceholder')"
                />
                <p class="input-hint">{{ t('admin.skills.editor.fields.sourceUrlHint') }}</p>
              </div>
              <div class="skill-editor-toggle-row skill-editor-fields__wide">
                <div>
                  <label class="input-label">{{ t('admin.skills.editor.fields.featured') }}</label>
                  <p class="input-hint">{{ t('admin.skills.editor.fields.featuredHint') }}</p>
                </div>
                <Toggle
                  v-model="draft.featured"
                  :disabled="isArchived"
                  :aria-label="t('admin.skills.editor.fields.featured')"
                />
              </div>
            </div>
          </section>

          <section id="skill-editor-package" class="skill-editor-section" aria-labelledby="skill-editor-package-title">
            <header class="skill-editor-section__heading">
              <div class="skill-editor-section__icon"><Icon name="cube" size="md" aria-hidden="true" /></div>
              <div>
                <h2 id="skill-editor-package-title">{{ t('admin.skills.editor.sections.package') }}</h2>
                <p>{{ t('admin.skills.editor.sections.packageHint') }}</p>
              </div>
            </header>

            <div class="skill-editor-package-grid">
              <SkillPackageUploader
                ref="packageUploader"
                :disabled="isArchived"
                :uploading="uploading"
                @upload="handleVersionUpload"
              />
              <div class="skill-editor-report-column">
                <SkillValidationReportPanel
                  :version="reportVersion"
                  :report="uploadFailureReport"
                />
                <div v-if="uploadErrorMessage" class="skill-editor-upload-error" role="alert">
                  <Icon name="exclamationCircle" size="sm" aria-hidden="true" />
                  <div>
                    <strong>{{ uploadErrorMessage }}</strong>
                    <p>{{ t('admin.skills.editor.package.repackageHint') }}</p>
                  </div>
                </div>
              </div>
            </div>

            <div class="skill-editor-versions">
              <div class="skill-editor-versions__heading">
                <div>
                  <h3>{{ t('admin.skills.editor.versions.title') }}</h3>
                  <p>{{ t('admin.skills.editor.versions.hint') }}</p>
                </div>
                <span>{{ t('admin.skills.versionCount', { count: versions.length }) }}</span>
              </div>

              <div v-if="!versions.length" class="skill-editor-versions__empty">
                <Icon name="inbox" size="lg" aria-hidden="true" />
                <p>{{ t('admin.skills.editor.versions.empty') }}</p>
              </div>

              <div v-else class="skill-editor-version-list">
                <article
                  v-for="version in versions"
                  :key="version.id"
                  class="skill-editor-version-row"
                  :class="{ 'skill-editor-version-row--selected': selectedVersionId === version.id }"
                >
                  <label class="skill-editor-version-row__select">
                    <input
                      v-model="selectedVersionId"
                      type="radio"
                      name="skill-release-version"
                      :value="version.id"
                      :disabled="version.status === 'yanked'"
                      @change="uploadFailureReport = null"
                    />
                    <span></span>
                    <span class="sr-only">{{ t('admin.skills.editor.versions.select') }} v{{ version.version }}</span>
                  </label>
                  <div class="skill-editor-version-row__identity">
                    <div>
                      <strong>v{{ version.version }}</strong>
                      <SkillStatusBadge :status="version.status" />
                    </div>
                    <p>{{ version.changelog }}</p>
                    <small>
                      {{ t('admin.skills.editor.versions.createdAt', { time: formatDateTime(version.created_at) }) }}
                      <span aria-hidden="true"> · </span>
                      {{ t('admin.skills.editor.versions.downloadCount', { count: version.download_count }) }}
                    </small>
                  </div>
                  <div class="skill-editor-version-row__checks">
                    <span :class="version.validation_report.valid ? 'is-valid' : 'is-invalid'">
                      <Icon
                        :name="version.validation_report.valid ? 'checkCircle' : 'exclamationTriangle'"
                        size="sm"
                        aria-hidden="true"
                      />
                      {{ version.validation_report.valid
                        ? t('admin.skills.validationState.ready')
                        : t('admin.skills.validationState.blocked') }}
                    </span>
                    <code :title="version.sha256">{{ compactHash(version.sha256) }}</code>
                  </div>
                  <div class="skill-editor-version-row__actions">
                    <button
                      v-if="skill?.status === 'published' && version.status === 'available'"
                      type="button"
                      class="btn btn-secondary btn-sm"
                      :disabled="!version.validation_report.valid || isOperating || isDirty"
                      @click="requestActivate(version)"
                    >
                      {{ t('admin.skills.editor.versions.activate') }}
                    </button>
                    <button
                      v-if="version.status !== 'yanked'"
                      type="button"
                      class="skill-editor-yank-button"
                      :disabled="version.status === 'active' || isOperating || isDirty"
                      :title="version.status === 'active'
                        ? t('admin.skills.editor.versions.activeYankHint')
                        : t('admin.skills.editor.versions.yank')"
                      @click="requestYank(version)"
                    >
                      <Icon name="ban" size="sm" aria-hidden="true" />
                      <span>{{ t('admin.skills.editor.versions.yank') }}</span>
                    </button>
                  </div>
                </article>
              </div>
            </div>
          </section>

          <section id="skill-editor-content" class="skill-editor-section" aria-labelledby="skill-editor-content-title">
            <header class="skill-editor-section__heading">
              <div class="skill-editor-section__icon"><Icon name="document" size="md" aria-hidden="true" /></div>
              <div>
                <h2 id="skill-editor-content-title">{{ t('admin.skills.editor.sections.content') }}</h2>
                <p>{{ t('admin.skills.editor.sections.contentHint') }}</p>
              </div>
            </header>

            <div class="skill-editor-fields">
              <div>
                <label for="skill-summary" class="input-label">{{ t('admin.skills.editor.fields.summary') }}</label>
                <input
                  id="skill-summary"
                  v-model="draft.summary"
                  type="text"
                  class="input"
                  :disabled="isArchived"
                  :placeholder="t('admin.skills.editor.fields.summaryPlaceholder')"
                />
              </div>
              <div>
                <label for="skill-description" class="input-label">
                  {{ t('admin.skills.editor.fields.description') }}
                </label>
                <textarea
                  id="skill-description"
                  v-model="draft.description"
                  class="input"
                  rows="5"
                  :disabled="isArchived"
                  :placeholder="t('admin.skills.editor.fields.descriptionPlaceholder')"
                ></textarea>
              </div>
              <div>
                <label for="skill-tags" class="input-label">{{ t('admin.skills.editor.fields.tags') }}</label>
                <input
                  id="skill-tags"
                  v-model="tagsText"
                  type="text"
                  class="input"
                  :disabled="isArchived"
                  :placeholder="t('admin.skills.editor.fields.tagsPlaceholder')"
                />
                <p class="input-hint">{{ t('admin.skills.editor.fields.tagsHint') }}</p>
              </div>
            </div>
          </section>

          <section id="skill-editor-release" class="skill-editor-section" aria-labelledby="skill-editor-release-title">
            <header class="skill-editor-section__heading">
              <div class="skill-editor-section__icon"><Icon name="globe" size="md" aria-hidden="true" /></div>
              <div>
                <h2 id="skill-editor-release-title">{{ t('admin.skills.editor.sections.release') }}</h2>
                <p>{{ t('admin.skills.editor.sections.releaseHint') }}</p>
              </div>
            </header>

            <div class="skill-editor-release-grid">
              <div>
                <h3 class="skill-editor-subheading">{{ t('admin.skills.editor.preview.title') }}</h3>
                <SkillPreviewCard :skill="previewDraft" />
              </div>
              <div class="skill-editor-preflight">
                <div class="skill-editor-preflight__heading">
                  <div>
                    <h3>{{ t('admin.skills.editor.preflight.title') }}</h3>
                    <p>{{ t('admin.skills.editor.preflight.hint') }}</p>
                  </div>
                  <Icon name="shield" size="md" aria-hidden="true" />
                </div>
                <ul>
                  <li v-for="check in preflightChecks" :key="check.key" :class="{ 'is-passed': check.passed }">
                    <Icon :name="check.passed ? 'checkCircle' : 'exclamationCircle'" size="sm" aria-hidden="true" />
                    <span>{{ check.label }}</span>
                  </li>
                </ul>
                <div class="skill-editor-preflight__result" :class="{ 'is-ready': preflightBlockCount === 0 }">
                  <strong>
                    {{ preflightBlockCount === 0
                      ? t('admin.skills.editor.preflight.ready')
                      : t('admin.skills.editor.preflight.blocked', { count: preflightBlockCount }) }}
                  </strong>
                  <button
                    v-if="skill?.status !== 'published' && skill?.status !== 'archived'"
                    type="button"
                    class="btn btn-primary"
                    :disabled="!canPublish || isOperating"
                    @click="requestPublish"
                  >
                    <Icon name="upload" size="sm" aria-hidden="true" />
                    <span class="ml-1.5">{{ t('admin.skills.editor.publish') }}</span>
                  </button>
                </div>
              </div>
            </div>
          </section>
        </main>
      </div>

      <ConfirmDialog
        :show="Boolean(pendingAction)"
        :title="confirmTitle"
        :message="confirmMessage"
        :confirm-text="confirmButtonText"
        :danger="pendingAction?.kind === 'archive' || pendingAction?.kind === 'yank'"
        @confirm="performPendingAction"
        @cancel="pendingAction = null"
      />

    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import skillsAPI, {
  type AdminSkill,
  type AdminSkillVersion,
  type CreateSkillRequest,
  type SkillValidationReport,
  type UploadSkillVersionRequest,
} from '@/api/admin/skills'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import SkillPackageUploader from '@/components/admin/skills/SkillPackageUploader.vue'
import SkillPreviewCard from '@/components/admin/skills/SkillPreviewCard.vue'
import SkillStatusBadge from '@/components/admin/skills/SkillStatusBadge.vue'
import SkillValidationReportPanel from '@/components/admin/skills/SkillValidationReport.vue'

type EditorSectionId = 'basics' | 'package' | 'content' | 'release'
type PendingAction =
  | { kind: 'publish'; version: AdminSkillVersion }
  | { kind: 'activate'; version: AdminSkillVersion }
  | { kind: 'yank'; version: AdminSkillVersion }
  | { kind: 'archive' }

type ValidationErrorPayload = {
  validation_report?: SkillValidationReport | string
  data?: { validation_report?: SkillValidationReport | string }
  details?: { validation_report?: SkillValidationReport | string }
}

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()

const routeId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id
const parsedId = Number(routeId)
const skillId = ref<number | null>(Number.isInteger(parsedId) && parsedId > 0 ? parsedId : null)
const isCreate = computed(() => skillId.value === null)
const skill = ref<AdminSkill | null>(null)
const loading = ref(false)
const loadError = ref<string | null>(null)
const saving = ref(false)
const uploading = ref(false)
const publishing = ref(false)
const operating = ref(false)
const validationErrors = ref<string[]>([])
const validationAlert = ref<HTMLElement | null>(null)
const activeSection = ref<EditorSectionId>('basics')
const selectedVersionId = ref<number | null>(null)
const uploadFailureReport = ref<SkillValidationReport | null>(null)
const uploadErrorMessage = ref('')
const packageUploader = ref<InstanceType<typeof SkillPackageUploader> | null>(null)
const pendingAction = ref<PendingAction | null>(null)
const allowRouteLeave = ref(false)
const savedSnapshot = ref('')
const tagsText = ref('')

const draft = reactive<CreateSkillRequest>(emptyDraft())

const versions = computed(() => {
  const source = skill.value?.versions ?? []
  const current = skill.value?.current_version
  if (current && !source.some((version) => version.id === current.id)) return [current, ...source]
  return [...source].sort((a, b) => b.id - a.id)
})

const selectedVersion = computed(() =>
  versions.value.find((version) => version.id === selectedVersionId.value) ?? null,
)

const reportVersion = computed(() => uploadFailureReport.value ? null : selectedVersion.value)
const isArchived = computed(() => skill.value?.status === 'archived')
const slugLocked = computed(() => versions.value.length > 0)
const isOperating = computed(() => saving.value || uploading.value || publishing.value || operating.value)
const previewDraft = computed<CreateSkillRequest>(() => buildPayload())
const isDirty = computed(() => savedSnapshot.value !== JSON.stringify(buildPayload()))

const basicComplete = computed(() => validateForSave(false).length === 0)
const packageComplete = computed(() => Boolean(selectedVersion.value?.validation_report.valid))
const contentComplete = computed(() => Boolean(
  draft.summary.trim()
  && draft.description.trim()
  && draft.category.trim(),
))

const editorSections = computed(() => [
  {
    id: 'basics' as const,
    icon: 'edit' as const,
    label: t('admin.skills.editor.sections.basics'),
    complete: basicComplete.value,
  },
  {
    id: 'package' as const,
    icon: 'cube' as const,
    label: t('admin.skills.editor.sections.package'),
    complete: packageComplete.value,
  },
  {
    id: 'content' as const,
    icon: 'document' as const,
    label: t('admin.skills.editor.sections.content'),
    complete: contentComplete.value,
  },
  {
    id: 'release' as const,
    icon: 'globe' as const,
    label: t('admin.skills.editor.sections.release'),
    complete: preflightBlockCount.value === 0,
  },
])

const preflightChecks = computed(() => [
  {
    key: 'metadata',
    label: t('admin.skills.editor.preflight.metadata'),
    passed: contentComplete.value,
  },
  {
    key: 'version',
    label: t('admin.skills.editor.preflight.version'),
    passed: Boolean(selectedVersion.value && selectedVersion.value.status !== 'yanked'),
  },
  {
    key: 'validation',
    label: t('admin.skills.editor.preflight.validation'),
    passed: selectedVersion.value?.validation_report.valid === true,
  },
])

const preflightBlockCount = computed(() => preflightChecks.value.filter((check) => !check.passed).length)
const canPublish = computed(() => Boolean(
  skillId.value
  && skill.value?.status === 'draft'
  && contentComplete.value
  && selectedVersion.value?.status !== 'yanked'
  && selectedVersion.value?.validation_report.valid === true,
))

const confirmTitle = computed(() => {
  switch (pendingAction.value?.kind) {
    case 'publish': return t('admin.skills.publishTitle')
    case 'activate': return t('admin.skills.editor.versions.activateTitle')
    case 'yank': return t('admin.skills.editor.versions.yankTitle')
    case 'archive': return t('admin.skills.archiveTitle')
    default: return ''
  }
})

const confirmMessage = computed(() => {
  const action = pendingAction.value
  if (!action) return ''
  if (action.kind === 'archive') {
    return t('admin.skills.archiveConfirm', { name: draft.display_name || draft.slug })
  }
  if (action.kind === 'publish') {
    return t('admin.skills.publishConfirm', {
      name: draft.display_name || draft.slug,
      version: action.version.version,
    })
  }
  return action.kind === 'activate'
    ? t('admin.skills.editor.versions.activateConfirm', { version: action.version.version })
    : t('admin.skills.editor.versions.yankConfirm', { version: action.version.version })
})

const confirmButtonText = computed(() => {
  switch (pendingAction.value?.kind) {
    case 'publish': return t('admin.skills.publish')
    case 'activate': return t('admin.skills.editor.versions.activate')
    case 'yank': return t('admin.skills.editor.versions.yank')
    case 'archive': return t('admin.skills.archive')
    default: return t('common.confirm')
  }
})

function emptyDraft(): CreateSkillRequest {
  return {
    slug: '',
    display_name: '',
    summary: '',
    description: '',
    category: '',
    tags: [],
    icon: '',
    origin_url: '',
    source_url: '',
    example_prompts: [],
    risk_notes: '',
    featured: false,
    sort_order: 0,
  }
}

function buildPayload(): CreateSkillRequest {
  return {
    slug: draft.slug.trim().toLowerCase(),
    display_name: draft.display_name.trim(),
    summary: draft.summary.trim(),
    description: draft.description.trim(),
    category: draft.category.trim(),
    tags: uniqueValues(tagsText.value),
    icon: draft.icon.trim(),
    origin_url: draft.origin_url?.trim() || '',
    source_url: draft.source_url?.trim() || '',
    example_prompts: [...(draft.example_prompts ?? [])],
    risk_notes: draft.risk_notes ?? '',
    featured: draft.featured,
    sort_order: Number.isFinite(Number(draft.sort_order)) ? Math.max(0, Number(draft.sort_order)) : 0,
  }
}

function applySkill(nextSkill: AdminSkill): void {
  skill.value = nextSkill
  Object.assign(draft, {
    slug: nextSkill.slug ?? '',
    display_name: nextSkill.display_name ?? '',
    summary: nextSkill.summary ?? '',
    description: nextSkill.description ?? '',
    category: nextSkill.category ?? '',
    tags: nextSkill.tags ?? [],
    icon: nextSkill.icon ?? '',
    origin_url: nextSkill.origin_url ?? '',
    source_url: nextSkill.source_url ?? '',
    example_prompts: [...(nextSkill.example_prompts ?? [])],
    risk_notes: nextSkill.risk_notes ?? '',
    featured: Boolean(nextSkill.featured),
    sort_order: nextSkill.sort_order ?? 0,
  })
  tagsText.value = (nextSkill.tags ?? []).join(', ')
  savedSnapshot.value = JSON.stringify(buildPayload())

  const selectable = nextSkill.current_version
    ?? nextSkill.versions?.find((version) => version.status === 'available' && version.validation_report.valid)
    ?? nextSkill.versions?.[0]
    ?? null
  if (!selectedVersionId.value || !versions.value.some((version) => version.id === selectedVersionId.value)) {
    selectedVersionId.value = selectable?.id ?? null
  }
}

async function loadSkill(): Promise<void> {
  if (!skillId.value) {
    savedSnapshot.value = JSON.stringify(buildPayload())
    return
  }
  loading.value = true
  loadError.value = null
  try {
    applySkill(await skillsAPI.getById(skillId.value))
  } catch (error: unknown) {
    loadError.value = extractApiErrorMessage(error, t('admin.skills.detailLoadFailed'))
  } finally {
    loading.value = false
  }
}

async function saveDraft(showSuccess = true): Promise<AdminSkill | null> {
  const errors = validateForSave(true)
  if (errors.length) return null
  const wasCreate = skillId.value === null
  saving.value = true
  try {
    const payload = buildPayload()
    const saved = skillId.value
      ? await skillsAPI.update(skillId.value, payload)
      : await skillsAPI.create(payload)
    skillId.value = saved.id
    applySkill(saved)
    validationErrors.value = []
    if (wasCreate) {
      allowRouteLeave.value = true
      try {
        await router.replace(`/admin/skills/${saved.id}/edit`)
      } finally {
        allowRouteLeave.value = false
      }
    }
    if (showSuccess) {
      appStore.showSuccess(t(wasCreate
        ? 'admin.skills.editor.createSuccess'
        : 'admin.skills.editor.saveSuccess'))
    }
    return saved
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.skills.editor.saveFailed')))
    return null
  } finally {
    saving.value = false
  }
}

function validateForSave(focusAlert: boolean): string[] {
  const errors: string[] = []
  if (!draft.display_name.trim()) errors.push(t('admin.skills.editor.validation.displayNameRequired'))
  if (!draft.slug.trim()) {
    errors.push(t('admin.skills.editor.validation.slugRequired'))
  } else if (
    draft.slug.trim().length > 64
    || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(draft.slug.trim().toLowerCase())
  ) {
    errors.push(t('admin.skills.editor.validation.slugInvalid'))
  }
  if (focusAlert) showValidationErrors(errors)
  return errors
}

function validateForPublish(): string[] {
  const errors: string[] = []
  if (!draft.summary.trim()) errors.push(t('admin.skills.editor.validation.summaryRequired'))
  if (!draft.description.trim()) errors.push(t('admin.skills.editor.validation.descriptionRequired'))
  if (!draft.category.trim()) errors.push(t('admin.skills.editor.validation.categoryRequired'))
  if (!selectedVersion.value || selectedVersion.value.status === 'yanked') {
    errors.push(t('admin.skills.editor.validation.versionRequired'))
  } else if (selectedVersion.value.validation_report.valid !== true) {
    errors.push(t('admin.skills.editor.validation.versionInvalid'))
  }
  showValidationErrors(errors)
  return errors
}

function showValidationErrors(errors: string[]): void {
  validationErrors.value = [...new Set(errors)]
  if (!errors.length) return
  void nextTick(() => validationAlert.value?.focus())
}

async function handleVersionUpload(request: UploadSkillVersionRequest): Promise<void> {
  uploadFailureReport.value = null
  uploadErrorMessage.value = ''

  if (!skillId.value || isDirty.value) {
    const saved = await saveDraft(false)
    if (!saved) return
  }
  if (!skillId.value) return

  uploading.value = true
  try {
    const version = await skillsAPI.uploadVersion(skillId.value, request)
    const currentSkill = skill.value
    if (currentSkill) {
      const nextVersions = (currentSkill.versions ?? []).filter((item) => item.id !== version.id)
      skill.value = { ...currentSkill, versions: [version, ...nextVersions] }
    }
    selectedVersionId.value = version.id
    packageUploader.value?.reset()
    appStore.showSuccess(t('admin.skills.editor.package.uploadSuccess'))
  } catch (error: unknown) {
    uploadFailureReport.value = extractRejectedValidationReport(error)
    uploadErrorMessage.value = extractApiErrorMessage(error, t('admin.skills.editor.package.uploadFailed'))
    appStore.showError(uploadErrorMessage.value)
  } finally {
    uploading.value = false
  }
}

async function requestPublish(): Promise<void> {
  if (validateForPublish().length || !selectedVersion.value) return
  const version = selectedVersion.value
  if (isDirty.value) {
    const saved = await saveDraft(false)
    if (!saved) return
  }
  pendingAction.value = { kind: 'publish', version }
}

function requestActivate(version: AdminSkillVersion): void {
  if (version.validation_report.valid !== true) {
    appStore.showError(t('admin.skills.editor.validation.versionInvalid'))
    return
  }
  pendingAction.value = { kind: 'activate', version }
}

function requestYank(version: AdminSkillVersion): void {
  if (version.status === 'active') return
  pendingAction.value = { kind: 'yank', version }
}

function requestArchive(): void {
  pendingAction.value = { kind: 'archive' }
}

async function performPendingAction(): Promise<void> {
  const action = pendingAction.value
  if (!action || !skillId.value || isOperating.value) return
  pendingAction.value = null
  const id = skillId.value
  publishing.value = action.kind === 'publish'
  operating.value = action.kind !== 'publish'
  try {
    const updated = action.kind === 'publish'
      ? await skillsAPI.publish(id, action.version.id)
      : action.kind === 'activate'
        ? await skillsAPI.activateVersion(id, action.version.id)
        : action.kind === 'yank'
          ? await skillsAPI.yankVersion(id, action.version.id)
          : await skillsAPI.archive(id)
    applySkill(updated)
    const successKey = action.kind === 'publish'
      ? 'admin.skills.publishSuccess'
      : action.kind === 'activate'
        ? 'admin.skills.editor.versions.activateSuccess'
        : action.kind === 'yank'
          ? 'admin.skills.editor.versions.yankSuccess'
          : 'admin.skills.archiveSuccess'
    appStore.showSuccess(t(successKey))
    if (action.kind === 'archive') await router.push('/admin/skills')
  } catch (error: unknown) {
    const failureKey = action.kind === 'publish'
      ? 'admin.skills.publishFailed'
      : action.kind === 'activate'
        ? 'admin.skills.editor.versions.activateFailed'
        : action.kind === 'yank'
          ? 'admin.skills.editor.versions.yankFailed'
          : 'admin.skills.archiveFailed'
    appStore.showError(extractApiErrorMessage(error, t(failureKey)))
  } finally {
    publishing.value = false
    operating.value = false
  }
}

function extractRejectedValidationReport(error: unknown): SkillValidationReport | null {
  if (!error || typeof error !== 'object') return null
  const candidate = error as ValidationErrorPayload & {
    metadata?: ValidationErrorPayload
    response?: { data?: ValidationErrorPayload }
  }
  const report = candidate.validation_report
    ?? candidate.data?.validation_report
    ?? candidate.details?.validation_report
    ?? candidate.metadata?.validation_report
    ?? candidate.metadata?.data?.validation_report
    ?? candidate.metadata?.details?.validation_report
    ?? candidate.response?.data?.validation_report
    ?? candidate.response?.data?.data?.validation_report
    ?? candidate.response?.data?.details?.validation_report
  return normalizeRejectedValidationReport(report)
}

function normalizeRejectedValidationReport(
  value: SkillValidationReport | string | undefined,
): SkillValidationReport | null {
  let report: SkillValidationReport | undefined
  if (typeof value === 'string') {
    try {
      report = JSON.parse(value) as SkillValidationReport
    } catch {
      return null
    }
  } else {
    report = value
  }
  if (!report || typeof report.valid !== 'boolean') return null
  return {
    valid: report.valid,
    errors: Array.isArray(report.errors) ? report.errors : [],
    warnings: Array.isArray(report.warnings) ? report.warnings : [],
  }
}

function uniqueValues(value: string): string[] {
  return [...new Set(value.split(/[,，\n]/).map((item) => item.trim()).filter(Boolean))]
}

function scrollToSection(section: EditorSectionId): void {
  activeSection.value = section
  document.getElementById(`skill-editor-${section}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function goBack(): void {
  if (isDirty.value && !window.confirm(t('admin.skills.editor.leaveDirty'))) return
  void router.push('/admin/skills')
}

function handleBeforeUnload(event: BeforeUnloadEvent): void {
  if (!isDirty.value) return
  event.preventDefault()
  event.returnValue = ''
}

function compactHash(hash: string): string {
  if (!hash) return '—'
  return hash.length > 16 ? `${hash.slice(0, 8)}…${hash.slice(-6)}` : hash
}

onBeforeRouteLeave(() => {
  if (allowRouteLeave.value || !isDirty.value) return true
  return window.confirm(t('admin.skills.editor.leaveDirty'))
})

onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  void loadSkill()
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', handleBeforeUnload)
})
</script>

<style scoped src="./SkillEditorView.css"></style>
