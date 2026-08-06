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
          <SkillInstallPanel
            class="skill-detail-install"
            :skill="skill"
            :version="currentVersion?.version"
            :sha256="currentVersion?.sha256"
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
} from '@/api/skills'

interface SkillSourceMeta {
  url: string
  repository: string
  stars: number | null
}

const route = useRoute()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const skill = ref<PublicSkill | null>(null)
const loading = ref(true)
const loadError = ref(false)
const notFound = ref(false)
const markdownCopied = ref(false)
let controller: AbortController | null = null
let markdownTimer: ReturnType<typeof setTimeout> | null = null

const slug = computed(() => {
  const value = route.params.slug
  return Array.isArray(value) ? value[0] || '' : String(value || '')
})

const currentVersion = computed(() => {
  const current = skill.value?.current_version
  if (!current) return null
  return skill.value?.versions.find((version) => version.version === current.version) ?? current
})

const rawSkillMarkdown = computed(() => currentVersion.value?.skill_md.trim() || '')
const skillMarkdown = computed(() => stripFrontmatter(rawSkillMarkdown.value))
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

function stripFrontmatter(markdown: string) {
  return markdown.replace(/^---\s*\n[\s\S]*?\n---\s*(?:\n|$)/, '').trim()
}

function safeSourceURL(value: string) {
  try {
    const url = new URL(value)
    return url.protocol === 'https:' && url.hostname.toLowerCase() === 'github.com'
      ? url.href
      : ''
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
