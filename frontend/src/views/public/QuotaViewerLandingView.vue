<template>
  <!--
    THESIS: Put renewal confidence on the desktop by making quota timing continuously visible.
    OWN-WORLD: Luoxue Snow Clay, expressed as a lavender desktop instrument rather than a generic SaaS landing page.
    STORY: See the quota, understand the reset, connect read-only, and renew on your own timing.
    FIRST VIEWPORT: A decisive product promise beside one dominant, faithful lavender quota-viewer proof, followed by the five-stage runway.
    FORM: The product journey follows seed 48ea3de4: 账户 → 安装 → 配对 → 桌面 → 续费.
    FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, and DESIGN.md
  -->
  <PublicSiteLayout class="quota-viewer-page" page="quota-viewer">
    <main id="top" class="quota-viewer-main">
      <section class="quota-hero" aria-labelledby="quota-viewer-title">
        <div class="quota-shell quota-hero-grid">
          <div class="quota-hero-copy">
            <h1 id="quota-viewer-title" :aria-label="t('quotaViewerLanding.hero.title')">
              <span>{{ t('quotaViewerLanding.hero.titleLineOne') }}</span>
              <span>{{ t('quotaViewerLanding.hero.titleLineTwo') }}</span>
            </h1>
            <p class="quota-hero-lede">{{ t('quotaViewerLanding.hero.description') }}</p>

            <div class="quota-hero-actions" :aria-label="t('quotaViewerLanding.hero.actionsAria')">
              <button
                type="button"
                class="quota-download-button quota-download-button--primary"
                :disabled="downloadState === 'preparing'"
                data-testid="download-macos"
                @click="startDownload('macos')"
              >
                <span class="quota-platform-symbol" aria-hidden="true">
                  <svg viewBox="0 0 24 24" fill="none">
                    <rect x="4" y="3.5" width="16" height="12" rx="2.25" />
                    <path d="M2.75 19.25h18.5M9 19.25l.6-2.25h4.8l.6 2.25" />
                  </svg>
                </span>
                <span>{{ macosDownloadLabel }}</span>
                <Icon v-if="activePlatform !== 'macos'" name="download" size="sm" aria-hidden="true" />
                <span v-else class="quota-button-spinner" aria-hidden="true"></span>
              </button>
              <button
                type="button"
                class="quota-download-button quota-download-button--secondary"
                :disabled="downloadState === 'preparing'"
                data-testid="download-windows"
                @click="startDownload('windows')"
              >
                <span class="quota-platform-symbol" aria-hidden="true">
                  <svg viewBox="0 0 24 24" fill="currentColor" stroke="none">
                    <path d="M3 4.6 10.7 3.5v7.8H3V4.6Zm8.7-1.25L21 2v9.3h-9.3V3.35ZM3 12.3h7.7v7.85L3 19.05V12.3Zm8.7 0H21v9.3l-9.3-1.3v-8Z" />
                  </svg>
                </span>
                <span>{{ windowsDownloadLabel }}</span>
                <Icon v-if="activePlatform !== 'windows'" name="download" size="sm" aria-hidden="true" />
                <span v-else class="quota-button-spinner" aria-hidden="true"></span>
              </button>
            </div>

            <p class="quota-open-access">
              <Icon name="checkCircle" size="sm" aria-hidden="true" />
              {{ t('quotaViewerLanding.hero.openAccess') }}
            </p>

            <ul class="quota-trust-list" :aria-label="t('quotaViewerLanding.hero.trustAria')">
              <li>
                <Icon name="download" size="sm" aria-hidden="true" />
                {{ t('quotaViewerLanding.hero.directDownload') }}
              </li>
              <li>
                <Icon name="shield" size="sm" aria-hidden="true" />
                {{ t('quotaViewerLanding.hero.readOnly') }}
              </li>
            </ul>

            <RouterLink
              v-if="authStore.isAuthenticated"
              to="/quota-viewer/devices"
              class="quota-manage-link"
            >
              {{ t('quotaViewerLanding.hero.manageDevices') }}
              <Icon name="arrowRight" size="sm" aria-hidden="true" />
            </RouterLink>

            <p
              v-if="downloadState !== 'idle'"
              class="quota-download-status"
              :class="`quota-download-status--${downloadState}`"
              role="status"
              aria-live="polite"
            >
              {{ downloadStatusMessage }}
            </p>
          </div>

          <div class="quota-product-stage" :aria-label="t('quotaViewerLanding.preview.ariaLabel')">
            <div class="quota-stage-orbit quota-stage-orbit--one" aria-hidden="true"></div>
            <div class="quota-stage-orbit quota-stage-orbit--two" aria-hidden="true"></div>
            <article class="quota-app-window">
              <header class="quota-app-titlebar">
                <div class="quota-window-dots" aria-hidden="true">
                  <span></span><span></span><span></span>
                </div>
                <strong>{{ t('quotaViewerLanding.preview.appName') }}</strong>
                <span class="quota-connected-state">
                  <span aria-hidden="true"></span>
                  {{ t('quotaViewerLanding.preview.connected') }}
                </span>
              </header>

              <div class="quota-app-body">
                <div class="quota-account-line">
                  <span class="quota-account-mark" aria-hidden="true">
                    <svg viewBox="0 0 36 36" fill="none">
                      <path d="M18 4.75 21.1 13l8.15 3.1-8.15 3.1L18 27.45l-3.1-8.25-8.15-3.1 8.15-3.1L18 4.75Z" />
                      <path d="M26.8 24.2 28 27.4l3.2 1.2-3.2 1.2-1.2 3.2-1.2-3.2-3.2-1.2 3.2-1.2 1.2-3.2Z" />
                    </svg>
                  </span>
                  <div>
                    <span>{{ t('quotaViewerLanding.preview.account') }}</span>
                    <strong>Pro</strong>
                  </div>
                </div>

                <div class="quota-ring-block">
                  <div class="quota-ring" role="img" :aria-label="t('quotaViewerLanding.preview.week')">
                    <div class="quota-ring-center">
                      <strong>32%</strong>
                      <span>{{ t('quotaViewerLanding.preview.week') }}</span>
                    </div>
                  </div>
                  <p>{{ t('quotaViewerLanding.preview.reset') }}</p>
                </div>

                <div class="quota-month-line">
                  <span>{{ t('quotaViewerLanding.preview.month') }}</span>
                  <span class="quota-month-bar" aria-hidden="true"><i></i></span>
                </div>

                <div class="quota-readonly-line">
                  <Icon name="shield" size="sm" aria-hidden="true" />
                  <span>{{ t('quotaViewerLanding.preview.readOnly') }}</span>
                </div>
              </div>
            </article>
          </div>
        </div>

        <ol class="quota-shell quota-runway" :aria-label="t('quotaViewerLanding.runway.ariaLabel')">
          <li v-for="(step, index) in runwaySteps" :key="step.id">
            <span class="quota-runway-number">{{ index + 1 }}</span>
            <div>
              <strong>{{ t(step.titleKey) }}</strong>
              <span>{{ t(step.descriptionKey) }}</span>
            </div>
          </li>
        </ol>
      </section>

      <section class="quota-section quota-states-section" aria-labelledby="quota-states-title">
        <div class="quota-shell quota-states-layout">
          <div class="quota-section-heading quota-section-heading--left">
            <h2 id="quota-states-title">{{ t('quotaViewerLanding.states.title') }}</h2>
            <p>{{ t('quotaViewerLanding.states.description') }}</p>
          </div>
          <ul class="quota-state-spectrum" :aria-label="t('quotaViewerLanding.states.ariaLabel')">
            <li v-for="state in quotaStates" :key="state.id" :class="`quota-state--${state.id}`">
              <span class="quota-state-ring" :style="{ '--state-progress': `${state.value}%` }">
                <strong>{{ state.value }}%</strong>
              </span>
              <span>{{ t(state.labelKey) }}</span>
            </li>
          </ul>
        </div>
      </section>

      <section class="quota-section quota-value-section" aria-labelledby="quota-value-title">
        <div class="quota-shell quota-value-panel">
          <div class="quota-section-heading quota-section-heading--left">
            <h2 id="quota-value-title">{{ t('quotaViewerLanding.value.title') }}</h2>
            <p>{{ t('quotaViewerLanding.value.description') }}</p>
          </div>
          <div class="quota-value-list">
            <article v-for="(item, index) in valueItems" :key="item.id">
              <span class="quota-value-number">0{{ index + 1 }}</span>
              <div>
                <h3>{{ t(item.titleKey) }}</h3>
                <p>{{ t(item.descriptionKey) }}</p>
              </div>
              <Icon :name="item.icon" size="md" aria-hidden="true" />
            </article>
          </div>
        </div>
      </section>

      <section id="download" class="quota-section quota-download-section" aria-labelledby="quota-download-title">
        <div class="quota-shell">
          <div class="quota-section-heading">
            <h2 id="quota-download-title">{{ t('quotaViewerLanding.download.title') }}</h2>
            <p>{{ t('quotaViewerLanding.download.description') }}</p>
          </div>

          <div class="quota-release-list">
            <article v-for="release in releases" :key="release.platform" class="quota-release-row">
              <span class="quota-release-platform" :class="`quota-release-platform--${release.platform}`" aria-hidden="true">
                <svg v-if="release.platform === 'macos'" viewBox="0 0 24 24" fill="none">
                  <rect x="4" y="3.5" width="16" height="12" rx="2.25" />
                  <path d="M2.75 19.25h18.5M9 19.25l.6-2.25h4.8l.6 2.25" />
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="currentColor" stroke="none">
                  <path d="M3 4.6 10.7 3.5v7.8H3V4.6Zm8.7-1.25L21 2v9.3h-9.3V3.35ZM3 12.3h7.7v7.85L3 19.05V12.3Zm8.7 0H21v9.3l-9.3-1.3v-8Z" />
                </svg>
              </span>
              <div class="quota-release-identity">
                <div>
                  <h3>{{ t(`quotaViewerLanding.download.${release.platform}.name`) }}</h3>
                  <span>{{ t(`quotaViewerLanding.download.${release.platform}.build`) }}</span>
                </div>
                <p>
                  <span>{{ t(`quotaViewerLanding.download.${release.platform}.version`) }}</span>
                  <span>{{ t(`quotaViewerLanding.download.${release.platform}.size`) }}</span>
                  <span>{{ t(`quotaViewerLanding.download.${release.platform}.signature`) }}</span>
                </p>
              </div>
              <div class="quota-release-checksum">
                <span>{{ t('quotaViewerLanding.download.checksum') }}</span>
                <code :title="release.sha256">{{ shortenedChecksum(release.sha256) }}</code>
              </div>
              <button
                type="button"
                class="quota-release-action"
                :disabled="downloadState === 'preparing'"
                :data-testid="`release-download-${release.platform}`"
                @click="startDownload(release.platform)"
              >
                <Icon name="download" size="sm" aria-hidden="true" />
                {{ release.platform === 'macos' ? macosDownloadLabel : windowsDownloadLabel }}
              </button>
            </article>
          </div>

          <p class="quota-signing-disclosure">
            <Icon name="infoCircle" size="sm" aria-hidden="true" />
            {{ t('quotaViewerLanding.download.disclosure') }}
          </p>
        </div>
      </section>

      <section class="quota-section quota-closing-section" aria-labelledby="quota-closing-title">
        <div class="quota-shell quota-closing-panel">
          <div>
            <h2 id="quota-closing-title">{{ t('quotaViewerLanding.closing.title') }}</h2>
            <p>{{ t('quotaViewerLanding.closing.description') }}</p>
          </div>
          <button
            type="button"
            class="quota-closing-action"
            :disabled="downloadState === 'preparing'"
            @click="startDownload(preferredPlatform)"
          >
            <Icon name="download" size="sm" aria-hidden="true" />
            {{ preferredPlatform === 'macos' ? macosDownloadLabel : windowsDownloadLabel }}
          </button>
        </div>
      </section>
    </main>
  </PublicSiteLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import PublicSiteLayout from '@/components/public/PublicSiteLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import {
  QUOTA_VIEWER_RELEASES,
  isQuotaViewerPlatform,
  issueQuotaViewerInstallerDownload,
  resolveQuotaViewerInstallerDownloadURL,
  type QuotaViewerPlatform,
} from '@/api/quotaViewer'

type DownloadState = 'idle' | 'preparing' | 'started' | 'failed'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

const downloadState = ref<DownloadState>('idle')
const activePlatform = ref<QuotaViewerPlatform | null>(null)
let autoDownloadHandled = false
let statusTimer: ReturnType<typeof setTimeout> | null = null

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '落雪API')
const preferredPlatform = computed<QuotaViewerPlatform>(() => {
  if (typeof navigator !== 'undefined' && /Windows/i.test(navigator.userAgent)) return 'windows'
  return 'macos'
})
const macosDownloadLabel = computed(() => t(authStore.isAuthenticated
  ? 'quotaViewerLanding.hero.macos'
  : 'quotaViewerLanding.hero.loginMacos'))
const windowsDownloadLabel = computed(() => t(authStore.isAuthenticated
  ? 'quotaViewerLanding.hero.windows'
  : 'quotaViewerLanding.hero.loginWindows'))
const downloadStatusMessage = computed(() => {
  if (downloadState.value === 'preparing') return t('quotaViewerLanding.status.preparing')
  if (downloadState.value === 'started') return t('quotaViewerLanding.status.started')
  if (downloadState.value === 'failed') return t('quotaViewerLanding.status.failed')
  return ''
})

const runwaySteps = [
  { id: 'account', titleKey: 'quotaViewerLanding.runway.items.account.title', descriptionKey: 'quotaViewerLanding.runway.items.account.description' },
  { id: 'install', titleKey: 'quotaViewerLanding.runway.items.install.title', descriptionKey: 'quotaViewerLanding.runway.items.install.description' },
  { id: 'pair', titleKey: 'quotaViewerLanding.runway.items.pair.title', descriptionKey: 'quotaViewerLanding.runway.items.pair.description' },
  { id: 'desktop', titleKey: 'quotaViewerLanding.runway.items.desktop.title', descriptionKey: 'quotaViewerLanding.runway.items.desktop.description' },
  { id: 'renew', titleKey: 'quotaViewerLanding.runway.items.renew.title', descriptionKey: 'quotaViewerLanding.runway.items.renew.description' },
] as const

const quotaStates = [
  { id: 'healthy', value: 82, labelKey: 'quotaViewerLanding.states.healthy' },
  { id: 'steady', value: 54, labelKey: 'quotaViewerLanding.states.steady' },
  { id: 'attention', value: 18, labelKey: 'quotaViewerLanding.states.attention' },
  { id: 'reset', value: 2, labelKey: 'quotaViewerLanding.states.reset' },
] as const

const valueItems = [
  { id: 'glance', titleKey: 'quotaViewerLanding.value.items.glance.title', descriptionKey: 'quotaViewerLanding.value.items.glance.description', icon: 'search' },
  { id: 'renewal', titleKey: 'quotaViewerLanding.value.items.renewal.title', descriptionKey: 'quotaViewerLanding.value.items.renewal.description', icon: 'calendar' },
  { id: 'safety', titleKey: 'quotaViewerLanding.value.items.safety.title', descriptionKey: 'quotaViewerLanding.value.items.safety.description', icon: 'shield' },
] as const

const releases = [QUOTA_VIEWER_RELEASES.macos, QUOTA_VIEWER_RELEASES.windows]

function queryString(value: unknown): string {
  if (Array.isArray(value)) return typeof value[0] === 'string' ? value[0] : ''
  return typeof value === 'string' ? value : ''
}

function shortenedChecksum(checksum: string): string {
  return `${checksum.slice(0, 12)}…${checksum.slice(-10)}`
}

async function clearDownloadIntent(): Promise<void> {
  if (!Object.prototype.hasOwnProperty.call(route.query, 'download')) return
  const nextQuery = { ...route.query }
  delete nextQuery.download
  await router.replace({ path: route.path, query: nextQuery, hash: route.hash })
}

async function persistDownloadIntent(platform: QuotaViewerPlatform): Promise<void> {
  const queryKeys = Object.keys(route.query)
  if (queryKeys.length === 1 && queryKeys[0] === 'download' && queryString(route.query.download) === platform) {
    return
  }
  await router.replace({ path: route.path, query: { download: platform }, hash: route.hash })
}

function triggerInstallerDownload(url: string): void {
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.setAttribute('download', '')
  anchor.style.display = 'none'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}

async function startDownload(platform: QuotaViewerPlatform, automatic = false): Promise<void> {
  if (downloadState.value === 'preparing') return

  if (!authStore.isAuthenticated) {
    const redirect = `/quota-viewer?download=${platform}`
    await router.push({ path: '/login', query: { redirect } })
    return
  }

  downloadState.value = 'preparing'
  activePlatform.value = platform
  if (statusTimer) clearTimeout(statusTimer)

  try {
    // Keep the requested platform in the URL before the authenticated call so
    // a fatal token-refresh failure can return from login and resume it.
    await persistDownloadIntent(platform)
    const release = await issueQuotaViewerInstallerDownload(platform)
    if (
      release.platform !== platform
      || release.version !== QUOTA_VIEWER_RELEASES[platform].version
      || release.sha256 !== QUOTA_VIEWER_RELEASES[platform].sha256
      || release.filename !== QUOTA_VIEWER_RELEASES[platform].filename
      || release.size !== QUOTA_VIEWER_RELEASES[platform].sizeBytes
      || release.architecture !== QUOTA_VIEWER_RELEASES[platform].architecture
      || release.signing_status !== QUOTA_VIEWER_RELEASES[platform].signingStatus
    ) {
      throw new Error('Quota viewer release metadata mismatch')
    }
    const downloadURL = resolveQuotaViewerInstallerDownloadURL(platform, release.download_path)
    await clearDownloadIntent()
    triggerInstallerDownload(downloadURL)
    downloadState.value = 'started'
    statusTimer = setTimeout(() => {
      downloadState.value = 'idle'
    }, 8000)
  } catch (error) {
    console.error('Failed to start quota viewer download', error)
    downloadState.value = 'failed'
  } finally {
    activePlatform.value = null
    if (automatic) autoDownloadHandled = true
  }
}

type ManagedHeadElement = {
  element: HTMLElement
  attribute: 'content' | 'href'
  previousValue: string | null
  created: boolean
}
const managedHead = new Map<string, ManagedHeadElement>()

function updateHeadElement(
  key: string,
  selector: string,
  create: () => HTMLElement,
  attribute: 'content' | 'href',
  value: string,
): void {
  let managed = managedHead.get(key)
  if (!managed) {
    let element = document.head.querySelector<HTMLElement>(selector)
    const created = !element
    if (!element) {
      element = create()
      document.head.appendChild(element)
    }
    managed = { element, attribute, previousValue: element.getAttribute(attribute), created }
    managedHead.set(key, managed)
  }
  managed.element.setAttribute(attribute, value)
}

function updateDocumentMeta(): void {
  const description = t('quotaViewerLanding.meta.description')
  const title = `${t('quotaViewerLanding.meta.title')} - ${siteName.value}`
  const canonicalURL = new URL('/quota-viewer', window.location.origin).toString()

  updateHeadElement('description', 'meta[name="description"]', () => {
    const element = document.createElement('meta')
    element.setAttribute('name', 'description')
    return element
  }, 'content', description)
  for (const [property, value] of [
    ['og:title', title],
    ['og:description', description],
    ['og:type', 'website'],
    ['og:url', canonicalURL],
  ] as const) {
    updateHeadElement(property, `meta[property="${property}"]`, () => {
      const element = document.createElement('meta')
      element.setAttribute('property', property)
      return element
    }, 'content', value)
  }
  updateHeadElement('canonical', 'link[rel="canonical"]', () => {
    const element = document.createElement('link')
    element.setAttribute('rel', 'canonical')
    return element
  }, 'href', canonicalURL)
}

function restoreDocumentMeta(): void {
  managedHead.forEach(({ element, attribute, previousValue, created }) => {
    if (created) element.remove()
    else if (previousValue == null) element.removeAttribute(attribute)
    else element.setAttribute(attribute, previousValue)
  })
  managedHead.clear()
}

watch([locale, siteName], () => {
  if (managedHead.size) updateDocumentMeta()
})

onMounted(async () => {
  updateDocumentMeta()
  const requestedPlatform = queryString(route.query.download).toLowerCase()
  if (!isQuotaViewerPlatform(requestedPlatform) || autoDownloadHandled) return

  if (!authStore.isAuthenticated) {
    await router.replace({ path: '/login', query: { redirect: route.fullPath } })
    return
  }
  autoDownloadHandled = true
  await startDownload(requestedPlatform, true)
})

onBeforeUnmount(() => {
  if (statusTimer) clearTimeout(statusTimer)
  restoreDocumentMeta()
})
</script>

<style scoped src="./QuotaViewerLandingView.css"></style>
