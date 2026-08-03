<template>
  <PublicSiteLayout class="skill-detail-page" page="skills">
    <!--
      THESIS: A Skill earns installation only after its purpose, package, risks, and exact local destination are legible together.
      OWN-WORLD: Snow Clay reading surfaces with a violet installation rail and dark code fields reserved for exact paths and hashes.
      STORY: Understand the capability, try its prompts, inspect risk and files, choose a version, then copy or download for local Codex.
      FIRST VIEWPORT: Identity and release facts lead into a reading column paired with the complete three-stage installation guide.
      FORM: An evidence sheet with a persistent action rail, continuing structural seed 41a98e36.
      FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, and DESIGN.md
    -->
    <main id="top" class="skill-detail-main">
      <div v-if="loading" class="skill-detail-shell skill-detail-loading" aria-busy="true" :aria-label="t('skills.detail.loading')">
        <div class="skill-detail-loading__hero"><span></span><span></span><span></span></div>
        <div class="skill-detail-loading__grid">
          <span></span><span></span>
        </div>
      </div>

      <section v-else-if="loadError || !skill" class="skill-detail-shell skill-detail-state" role="alert">
        <span aria-hidden="true"><Icon :name="notFound ? 'inbox' : 'exclamationCircle'" size="lg" /></span>
        <h1>{{ t(notFound ? 'skills.detail.notFoundTitle' : 'skills.detail.errorTitle') }}</h1>
        <p>{{ t(notFound ? 'skills.detail.notFoundDescription' : 'skills.detail.errorDescription') }}</p>
        <div>
          <button v-if="!notFound" type="button" @click="loadSkill">
            <Icon name="refresh" size="sm" aria-hidden="true" />
            {{ t('skills.detail.retry') }}
          </button>
          <RouterLink to="/skills">
            <Icon name="arrowLeft" size="sm" aria-hidden="true" />
            {{ t('skills.detail.backToMarket') }}
          </RouterLink>
        </div>
      </section>

      <template v-else>
        <header class="skill-detail-hero">
          <div class="skill-detail-shell">
            <nav class="skill-detail-breadcrumb" :aria-label="t('skills.detail.breadcrumbLabel')">
              <RouterLink to="/skills">{{ t('skills.navLabel') }}</RouterLink>
              <Icon name="chevronRight" size="xs" aria-hidden="true" />
              <span aria-current="page">{{ skill.display_name }}</span>
            </nav>

            <div class="skill-detail-hero__content">
              <span class="skill-detail-hero__mark" aria-hidden="true">
                <Icon name="sparkles" size="xl" :stroke-width="1.7" />
              </span>
              <div class="skill-detail-hero__copy">
                <h1>{{ skill.display_name }}</h1>
                <p>{{ skill.summary || t('skills.card.noSummary') }}</p>
                <div class="skill-detail-meta">
                  <span v-if="skill.category"><Icon name="grid" size="xs" aria-hidden="true" />{{ skill.category }}</span>
                  <span v-if="skill.current_version"><Icon name="badge" size="xs" aria-hidden="true" />v{{ skill.current_version.version }}</span>
                  <span><Icon name="download" size="xs" aria-hidden="true" />{{ t('skills.card.downloads', { count: formatNumber(skill.download_count) }) }}</span>
                  <span v-if="formattedPublishedAt"><Icon name="calendar" size="xs" aria-hidden="true" />{{ formattedPublishedAt }}</span>
                </div>
                <ul v-if="skill.tags.length" class="skill-detail-tags" :aria-label="t('skills.card.tags')">
                  <li v-for="tag in skill.tags" :key="tag">{{ tag }}</li>
                </ul>
              </div>
            </div>
          </div>
        </header>

        <div class="skill-detail-shell skill-detail-layout">
          <article class="skill-detail-content">
            <section aria-labelledby="skill-capability-title">
              <div class="skill-detail-section-heading">
                <h2 id="skill-capability-title">{{ t('skills.detail.capabilityTitle') }}</h2>
                <p>{{ t('skills.detail.capabilityDescription') }}</p>
              </div>
              <SafeMarkdown
                v-if="skill.description"
                :content="skill.description"
                class="skill-detail-markdown"
              />
              <p v-else class="skill-detail-muted">{{ t('skills.detail.noDescription') }}</p>
            </section>

            <section aria-labelledby="skill-prompts-title">
              <div class="skill-detail-section-heading">
                <h2 id="skill-prompts-title">{{ t('skills.detail.promptsTitle') }}</h2>
                <p>{{ t('skills.detail.promptsDescription') }}</p>
              </div>
              <ul v-if="skill.example_prompts.length" class="skill-detail-prompts">
                <li v-for="(prompt, index) in skill.example_prompts" :key="`${index}-${prompt}`">
                  <Icon name="chat" size="sm" aria-hidden="true" />
                  <span>{{ prompt }}</span>
                  <button
                    type="button"
                    :aria-label="t('skills.detail.copyPromptAria', { prompt })"
                    @click="copyPrompt(prompt, index)"
                  >
                    <Icon :name="copiedPrompt === index ? 'check' : 'copy'" size="sm" aria-hidden="true" />
                    {{ t(copiedPrompt === index ? 'skills.detail.copied' : 'skills.detail.copy') }}
                  </button>
                </li>
              </ul>
              <p v-else class="skill-detail-muted">{{ t('skills.detail.noPrompts') }}</p>
            </section>

            <section aria-labelledby="skill-risk-title">
              <div class="skill-detail-section-heading">
                <h2 id="skill-risk-title">{{ t('skills.detail.riskTitle') }}</h2>
                <p>{{ t('skills.detail.riskDescription') }}</p>
              </div>
              <div class="skill-detail-risk" :class="{ 'skill-detail-risk--clear': !hasRiskSignals }">
                <span aria-hidden="true">
                  <Icon :name="hasRiskSignals ? 'exclamationTriangle' : 'shield'" size="md" />
                </span>
                <div>
                  <strong>{{ t(hasRiskSignals ? 'skills.detail.riskFound' : 'skills.detail.noExtraRisk') }}</strong>
                  <SafeMarkdown v-if="skill.risk_notes" :content="skill.risk_notes" compact />
                  <ul v-if="selectedWarnings.length" class="skill-detail-risk__warnings">
                    <li v-for="warning in selectedWarnings" :key="`${warning.code}-${warning.path || ''}`">
                      <Icon name="exclamationCircle" size="xs" aria-hidden="true" />
                      <span>
                        {{ riskWarningLabel(warning.code, warning.message) }}
                        <code v-if="warning.path">{{ warning.path }}</code>
                      </span>
                    </li>
                  </ul>
                  <p v-if="!skill.risk_notes && !selectedWarnings.length">{{ t('skills.detail.noExtraRiskDescription') }}</p>
                </div>
              </div>
            </section>

            <section aria-labelledby="skill-package-title">
              <div class="skill-detail-section-heading skill-detail-section-heading--with-control">
                <div>
                  <h2 id="skill-package-title">{{ t('skills.detail.packageTitle') }}</h2>
                  <p>{{ t('skills.detail.packageDescription') }}</p>
                </div>
                <label v-if="availableVersions.length > 1">
                  <span>{{ t('skills.detail.versionLabel') }}</span>
                  <select v-model="selectedVersionNumber">
                    <option v-for="version in availableVersions" :key="version.version" :value="version.version">
                      v{{ version.version }}
                    </option>
                  </select>
                </label>
              </div>

              <div v-if="selectedVersion" class="skill-detail-release-facts">
                <dl>
                  <div><dt>{{ t('skills.detail.version') }}</dt><dd>v{{ selectedVersion.version }}</dd></div>
                  <div><dt>{{ t('skills.detail.packageSize') }}</dt><dd>{{ formatBytes(selectedVersion.byte_size) }}</dd></div>
                  <div><dt>{{ t('skills.detail.fileCount') }}</dt><dd>{{ selectedVersion.file_count }}</dd></div>
                  <div><dt>{{ t('skills.detail.versionDownloads') }}</dt><dd>{{ formatNumber(selectedVersion.download_count) }}</dd></div>
                </dl>
                <div v-if="selectedVersion.sha256" class="skill-detail-checksum">
                  <span>SHA-256</span>
                  <code :title="selectedVersion.sha256">{{ selectedVersion.sha256 }}</code>
                  <button type="button" :aria-label="t('skills.detail.copyChecksum')" @click="copyChecksum">
                    <Icon :name="checksumCopied ? 'check' : 'copy'" size="sm" aria-hidden="true" />
                  </button>
                </div>
                <div v-if="selectedVersion.changelog" class="skill-detail-changelog">
                  <strong>{{ t('skills.detail.changelog') }}</strong>
                  <SafeMarkdown :content="selectedVersion.changelog" compact />
                </div>
              </div>

              <div v-if="selectedVersion?.file_manifest.length" class="skill-detail-files">
                <div class="skill-detail-files__heading">
                  <span>{{ t('skills.detail.fileTree') }}</span>
                  <small>{{ t('skills.detail.fileTreeCount', { count: selectedVersion.file_manifest.length }) }}</small>
                </div>
                <ul>
                  <li v-for="file in selectedVersion.file_manifest" :key="file.path">
                    <Icon name="document" size="sm" aria-hidden="true" />
                    <code :title="file.path">{{ file.path }}</code>
                    <span>{{ formatBytes(file.byte_size) }}</span>
                  </li>
                </ul>
              </div>
              <p v-else class="skill-detail-muted">{{ t('skills.detail.noFileManifest') }}</p>
            </section>

            <section v-if="skillMarkdown" aria-labelledby="skill-source-title">
              <details class="skill-detail-source">
                <summary id="skill-source-title">
                  <span>
                    <strong>{{ t('skills.detail.sourceTitle') }}</strong>
                    <small>{{ t('skills.detail.sourceDescription') }}</small>
                  </span>
                  <Icon name="chevronDown" size="sm" aria-hidden="true" />
                </summary>
                <SafeMarkdown :content="skillMarkdown" class="skill-detail-source__body" />
              </details>
            </section>

            <section v-if="availableVersions.length > 1" aria-labelledby="skill-history-title">
              <div class="skill-detail-section-heading">
                <h2 id="skill-history-title">{{ t('skills.detail.historyTitle') }}</h2>
                <p>{{ t('skills.detail.historyDescription') }}</p>
              </div>
              <ol class="skill-detail-history">
                <li v-for="version in availableVersions" :key="version.version">
                  <button type="button" @click="selectedVersionNumber = version.version">
                    <span>v{{ version.version }}</span>
                    <small>{{ formatDate(version.created_at) }}</small>
                    <Icon name="chevronRight" size="xs" aria-hidden="true" />
                  </button>
                </li>
              </ol>
            </section>
          </article>

          <div class="skill-detail-install-rail">
            <SkillInstallPanel
              :skill="skill"
              :version="skill.current_version ? selectedVersion?.version : undefined"
              :sha256="skill.current_version ? selectedVersion?.sha256 : undefined"
            />
          </div>
        </div>

        <section class="skill-detail-shell skill-detail-closing" aria-labelledby="skill-closing-title">
          <div>
            <h2 id="skill-closing-title">{{ t('skills.detail.closingTitle') }}</h2>
            <p>{{ t('skills.detail.closingDescription') }}</p>
          </div>
          <RouterLink to="/skills">
            {{ t('skills.detail.exploreMore') }}
            <Icon name="arrowRight" size="sm" aria-hidden="true" />
          </RouterLink>
        </section>
      </template>
    </main>
  </PublicSiteLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import PublicSiteLayout from '@/components/public/PublicSiteLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import SafeMarkdown from '@/components/skills/SafeMarkdown.vue'
import SkillInstallPanel from '@/components/skills/SkillInstallPanel.vue'
import { useClipboard } from '@/composables/useClipboard'
import {
  getPublicSkill,
  getPublicSkillVersions,
  type PublicSkill,
  type PublicSkillVersion,
} from '@/api/skills'

const route = useRoute()
const { t, te, locale } = useI18n()
const { copyToClipboard } = useClipboard()
const skill = ref<PublicSkill | null>(null)
const loading = ref(true)
const loadError = ref(false)
const notFound = ref(false)
const selectedVersionNumber = ref('')
const copiedPrompt = ref<number | null>(null)
const checksumCopied = ref(false)
let controller: AbortController | null = null
let copiedTimer: ReturnType<typeof setTimeout> | null = null
let checksumTimer: ReturnType<typeof setTimeout> | null = null

const slug = computed(() => {
  const value = route.params.slug
  return Array.isArray(value) ? value[0] || '' : String(value || '')
})

const availableVersions = computed<PublicSkillVersion[]>(() => {
  if (!skill.value) return []
  const all = [...skill.value.versions]
  if (
    skill.value.current_version
    && !all.some((version) => version.version === skill.value?.current_version?.version)
  ) {
    all.unshift(skill.value.current_version)
  }
  return all.sort((a, b) => b.created_at.localeCompare(a.created_at))
})

const selectedVersion = computed(() => availableVersions.value.find(
  (version) => version.version === selectedVersionNumber.value,
) ?? availableVersions.value[0] ?? null)

const selectedWarnings = computed(() => selectedVersion.value?.validation_report.warnings ?? [])
const hasRiskSignals = computed(() => Boolean(skill.value?.risk_notes || selectedWarnings.value.length))

const formattedPublishedAt = computed(() => skill.value?.published_at
  ? formatDate(skill.value.published_at)
  : '')

const skillMarkdown = computed(() => stripFrontmatter(selectedVersion.value?.skill_md || ''))

watch(slug, loadSkill, { immediate: true })

function formatNumber(value: number) {
  return new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 1 }).format(value)
}

function formatDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(locale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date)
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '—'
  const units = ['B', 'KB', 'MB', 'GB']
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  const amount = value / 1024 ** index
  return `${new Intl.NumberFormat(locale.value, { maximumFractionDigits: index === 0 ? 0 : 1 }).format(amount)} ${units[index]}`
}

function stripFrontmatter(markdown: string) {
  return markdown.replace(/^---\s*\n[\s\S]*?\n---\s*(?:\n|$)/, '').trim()
}

function riskWarningLabel(code: string, fallback: string) {
  const key = `skills.detail.validationWarnings.${code}`
  return te(key) ? t(key) : fallback || code
}

async function loadSkill() {
  if (!slug.value) {
    loading.value = false
    notFound.value = true
    return
  }

  controller?.abort()
  const requestController = new AbortController()
  controller = requestController
  loading.value = true
  loadError.value = false
  notFound.value = false
  skill.value = null

  try {
    const detail = await getPublicSkill(slug.value, { signal: requestController.signal })
    if (!detail.versions.length) {
      try {
        detail.versions = await getPublicSkillVersions(slug.value, { signal: requestController.signal })
      } catch {
        // The current immutable version still supports installation if history is unavailable.
      }
    }
    skill.value = detail
    selectedVersionNumber.value = detail.current_version?.version
      || detail.versions[0]?.version
      || ''
  } catch (error) {
    if (requestController.signal.aborted) return
    const status = extractErrorStatus(error)
    notFound.value = status === 404
    loadError.value = true
  } finally {
    if (!requestController.signal.aborted) loading.value = false
  }
}

function extractErrorStatus(error: unknown): number | undefined {
  if (!error || typeof error !== 'object') return undefined
  const candidate = error as {
    status?: unknown
    response?: { status?: unknown }
  }
  if (typeof candidate.status === 'number') return candidate.status
  return typeof candidate.response?.status === 'number' ? candidate.response.status : undefined
}

async function copyPrompt(prompt: string, index: number) {
  const copied = await copyToClipboard(prompt, t('skills.detail.promptCopySuccess'))
  if (!copied) return
  copiedPrompt.value = index
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => {
    copiedPrompt.value = null
  }, 2200)
}

async function copyChecksum() {
  if (!selectedVersion.value?.sha256) return
  const copied = await copyToClipboard(selectedVersion.value.sha256, t('skills.detail.checksumCopySuccess'))
  if (!copied) return
  checksumCopied.value = true
  if (checksumTimer) clearTimeout(checksumTimer)
  checksumTimer = setTimeout(() => {
    checksumCopied.value = false
  }, 2200)
}

onBeforeUnmount(() => {
  controller?.abort()
  if (copiedTimer) clearTimeout(copiedTimer)
  if (checksumTimer) clearTimeout(checksumTimer)
})
</script>

<style src="./SkillDetailView.css"></style>
