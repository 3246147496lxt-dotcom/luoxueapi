<template>
  <PublicSiteLayout class="skill-detail-page" page="skills">
    <main id="top" class="skill-detail-main">
      <div v-if="loading" class="skill-detail-shell skill-detail-loading" aria-busy="true" :aria-label="t('skills.detail.loading')">
        <div class="skill-detail-loading__hero"><span></span><span></span><span></span></div>
        <div class="skill-detail-loading__grid"><span></span><span></span></div>
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

            <div class="skill-detail-hero__copy">
              <h1>{{ skill.display_name }}</h1>
              <p>{{ skill.summary || t('skills.card.noSummary') }}</p>
            </div>
          </div>
        </header>

        <div class="skill-detail-shell skill-detail-layout">
          <section class="skill-detail-review" aria-labelledby="skill-risk-title">
            <div class="skill-detail-review__heading">
              <div>
                <h2 id="skill-risk-title">{{ t('skills.detail.riskTitle') }}</h2>
                <p>{{ t('skills.detail.riskDescription') }}</p>
              </div>
              <label v-if="availableVersions.length > 1">
                <span>{{ t('skills.detail.versionLabel') }}</span>
                <select v-model="selectedVersionNumber">
                  <option
                    v-for="version in availableVersions"
                    :key="version.version"
                    :value="version.version"
                  >
                    v{{ version.version }}
                  </option>
                </select>
              </label>
            </div>

            <div class="skill-detail-review__risk" :class="{ 'is-clear': !hasRiskSignals }">
              <span aria-hidden="true">
                <Icon :name="hasRiskSignals ? 'exclamationTriangle' : 'shield'" size="md" />
              </span>
              <div>
                <strong>{{ t(hasRiskSignals ? 'skills.detail.riskFound' : 'skills.detail.noExtraRisk') }}</strong>
                <SafeMarkdown v-if="skill.risk_notes" :content="skill.risk_notes" compact />
                <ul v-if="selectedWarnings.length">
                  <li v-for="warning in selectedWarnings" :key="`${warning.code}-${warning.path || ''}`">
                    <Icon name="exclamationCircle" size="xs" aria-hidden="true" />
                    <span>
                      {{ riskWarningLabel(warning.code, warning.message) }}
                      <code v-if="warning.path">{{ warning.path }}</code>
                    </span>
                  </li>
                </ul>
                <p v-if="!skill.risk_notes && !selectedWarnings.length">
                  {{ t('skills.detail.noExtraRiskDescription') }}
                </p>
              </div>
            </div>

            <div v-if="selectedVersion" class="skill-detail-review__package">
              <dl>
                <div><dt>{{ t('skills.detail.version') }}</dt><dd>v{{ selectedVersion.version }}</dd></div>
                <div><dt>{{ t('skills.detail.packageSize') }}</dt><dd>{{ formatBytes(selectedVersion.byte_size) }}</dd></div>
                <div><dt>{{ t('skills.detail.fileCount') }}</dt><dd>{{ selectedVersion.file_count }}</dd></div>
                <div><dt>{{ t('skills.detail.versionDownloads') }}</dt><dd>{{ formatNumber(selectedVersion.download_count) }}</dd></div>
              </dl>

              <div v-if="selectedVersion.sha256" class="skill-detail-review__checksum">
                <span>SHA-256</span>
                <code :title="selectedVersion.sha256">{{ selectedVersion.sha256 }}</code>
              </div>

              <div v-if="selectedVersion.file_manifest.length" class="skill-detail-review__files">
                <div>
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
            </div>

            <p v-else class="skill-detail-muted">{{ t('skills.install.versionUnavailable') }}</p>
          </section>

          <SkillInstallPanel
            class="skill-detail-install"
            :skill="skill"
            :version="skill.current_version ? selectedVersion?.version : undefined"
            :sha256="skill.current_version ? selectedVersion?.sha256 : undefined"
          />

          <section class="skill-detail-overview" aria-labelledby="skill-overview-title">
            <h2 id="skill-overview-title">{{ t('skills.detail.overviewTitle') }}</h2>
            <div class="skill-detail-overview__card">
              <SafeMarkdown
                v-if="skill.description"
                :content="skill.description"
                class="skill-detail-overview__markdown"
              />
              <p v-else class="skill-detail-muted">{{ t('skills.detail.noDescription') }}</p>
            </div>
          </section>

          <aside v-if="hasSupportingInfo" class="skill-detail-info" :aria-labelledby="`${skill.slug}-info-title`">
            <h2 :id="`${skill.slug}-info-title`">{{ t('skills.detail.infoTitle') }}</h2>

            <div v-if="skill.category" class="skill-detail-info__category">
              <span>{{ t('skills.detail.categoryLabel') }}</span>
              <strong>{{ skill.category }}</strong>
            </div>

            <ul v-if="skill.tags.length" class="skill-detail-info__tags" :aria-label="t('skills.card.tags')">
              <li v-for="tag in skill.tags" :key="tag">{{ tag }}</li>
            </ul>

            <div v-if="sourceMeta" class="skill-detail-info__source">
              <span>{{ t('skills.detail.sourceLabel') }}</span>
              <a
                :href="sourceMeta.url"
                target="_blank"
                rel="noopener noreferrer"
                :aria-label="t('skills.detail.openSourceAria', { repository: sourceMeta.repository })"
              >
                <GitHubMark class="skill-detail-info__github" />
                <span>
                  <strong>{{ sourceMeta.repository }}</strong>
                  <small v-if="sourceMeta.stars !== null" :title="t('skills.detail.repositoryStars')">
                    ★ {{ formatNumber(sourceMeta.stars) }}
                  </small>
                </span>
                <Icon name="externalLink" size="xs" aria-hidden="true" />
              </a>
            </div>
          </aside>

          <section v-if="skillMarkdown" class="skill-detail-source" aria-labelledby="skill-source-title">
            <div class="skill-detail-source__heading">
              <h2 id="skill-source-title">SKILL.md</h2>
              <button
                type="button"
                :aria-label="t(markdownCopied ? 'skills.detail.fullCopied' : 'skills.detail.copyFull')"
                @click="copySkillMarkdown"
              >
                <Icon
                  :name="markdownCopied ? 'check' : 'lucideCopy'"
                  size="sm"
                  :stroke-width="2"
                  aria-hidden="true"
                />
                <span aria-live="polite">
                  {{ t(markdownCopied ? 'skills.detail.fullCopied' : 'skills.detail.copyFull') }}
                </span>
              </button>
            </div>
            <SafeMarkdown :content="skillMarkdown" class="skill-detail-source__body" />
          </section>
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
import GitHubMark from '@/components/auth/GitHubMark.vue'
import SafeMarkdown from '@/components/skills/SafeMarkdown.vue'
import SkillInstallPanel from '@/components/skills/SkillInstallPanel.vue'
import { useClipboard } from '@/composables/useClipboard'
import {
  getPublicSkill,
  getPublicSkillVersions,
  type PublicSkill,
  type PublicSkillVersion,
} from '@/api/skills'

interface SkillSourceMeta {
  url: string
  repository: string
  stars: number | null
}

const route = useRoute()
const { t, te } = useI18n()
const { copyToClipboard } = useClipboard()
const skill = ref<PublicSkill | null>(null)
const loading = ref(true)
const loadError = ref(false)
const notFound = ref(false)
const selectedVersionNumber = ref('')
const markdownCopied = ref(false)
let controller: AbortController | null = null
let markdownTimer: ReturnType<typeof setTimeout> | null = null

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

const rawSkillMarkdown = computed(() => selectedVersion.value?.skill_md.trim() || '')
const skillMarkdown = computed(() => stripFrontmatter(rawSkillMarkdown.value))
const selectedWarnings = computed(() => selectedVersion.value?.validation_report.warnings ?? [])
const hasRiskSignals = computed(() => Boolean(skill.value?.risk_notes || selectedWarnings.value.length))
const sourceMeta = computed<SkillSourceMeta | null>(() => {
  if (!skill.value) return null
  const url = safeSourceURL(skill.value.source_url || '')
  if (!url) return null
  return {
    url,
    repository: skill.value.source_repository || repositoryName(url) || skill.value.display_name,
    stars: skill.value.repository_stars ?? null,
  }
})
const hasSupportingInfo = computed(() => Boolean(
  skill.value?.category || skill.value?.tags.length || sourceMeta.value,
))

watch(slug, loadSkill, { immediate: true })

function formatNumber(value: number) {
  return new Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 1 })
    .format(value)
    .toLowerCase()
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '—'
  const units = ['B', 'KB', 'MB', 'GB']
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  const amount = value / 1024 ** index
  return `${new Intl.NumberFormat(undefined, { maximumFractionDigits: index === 0 ? 0 : 1 }).format(amount)} ${units[index]}`
}

function stripFrontmatter(markdown: string) {
  return markdown.replace(/^---\s*\n[\s\S]*?\n---\s*(?:\n|$)/, '').trim()
}

function safeSourceURL(value: string) {
  try {
    const url = new URL(value)
    return url.protocol === 'https:' ? url.href : ''
  } catch {
    return ''
  }
}

function repositoryName(value: string) {
  try {
    const url = new URL(value)
    if (url.hostname.toLowerCase() !== 'github.com') return url.hostname
    return url.pathname.split('/').filter(Boolean).slice(0, 2).join('/')
  } catch {
    return ''
  }
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

async function copySkillMarkdown() {
  if (!rawSkillMarkdown.value) return
  const copied = await copyToClipboard(rawSkillMarkdown.value, t('skills.detail.fullCopySuccess'))
  if (!copied) return
  markdownCopied.value = true
  if (markdownTimer) clearTimeout(markdownTimer)
  markdownTimer = setTimeout(() => {
    markdownCopied.value = false
  }, 2200)
}

onBeforeUnmount(() => {
  controller?.abort()
  if (markdownTimer) clearTimeout(markdownTimer)
})
</script>

<style src="./SkillDetailView.css"></style>
