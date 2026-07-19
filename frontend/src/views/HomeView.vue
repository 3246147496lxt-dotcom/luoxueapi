<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      :title="`${siteName} ${t('home.nav.ariaLabel')}`"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- SECURITY: homeContent is an administrator-only setting. -->
    <div v-else v-html="homeContent"></div>
  </div>

  <PublicSiteLayout v-else class="home-page" page="home">
    <main id="top">
      <section class="hero-section" aria-labelledby="home-hero-title">
        <div class="content-shell hero-layout">
          <div class="hero-copy">
            <p class="status-line">
              <span class="status-dot" aria-hidden="true"></span>
              {{ t('home.hero.status') }}
            </p>
            <h1 id="home-hero-title">{{ t('home.hero.title') }}</h1>
            <p class="hero-description">{{ t('home.hero.description') }}</p>
            <div class="hero-actions">
              <router-link
                data-testid="hero-primary-cta"
                :to="primaryCta.to"
                class="primary-button"
              >
                {{ primaryCta.label }}
                <Icon name="arrowRight" size="sm" aria-hidden="true" />
              </router-link>
              <a v-if="tutorialUrl" :href="tutorialUrl" class="secondary-button">
                {{ t('home.hero.tutorial') }}
              </a>
            </div>
          </div>

          <div class="code-showcase" aria-labelledby="code-example-title">
            <div class="code-copy">
              <h2 id="code-example-title">{{ t('home.codeExample.title') }}</h2>
              <p>{{ t('home.codeExample.description') }}</p>
            </div>

            <div class="code-window">
              <div class="code-toolbar">
                <div
                  class="code-tabs"
                  role="tablist"
                  aria-orientation="horizontal"
                  :aria-label="t('home.codeExample.title')"
                >
                  <button
                    v-for="(tab, index) in codeTabs"
                    :id="`code-tab-${tab.id}`"
                    :key="tab.id"
                    ref="codeTabRefs"
                    type="button"
                    role="tab"
                    :aria-selected="activeCode === tab.id"
                    :aria-controls="`code-panel-${tab.id}`"
                    :tabindex="activeCode === tab.id ? 0 : -1"
                    :class="{ 'is-active': activeCode === tab.id }"
                    @click="selectCodeTab(tab.id)"
                    @keydown="handleCodeTabKeydown($event, index)"
                  >
                    {{ t(tab.labelKey) }}
                  </button>
                </div>
                <button
                  type="button"
                  class="copy-button"
                  :aria-label="copyAriaLabel"
                  @click="copyActiveCode"
                >
                  <Icon :name="copyStatus === 'copied' ? 'check' : 'copy'" size="sm" aria-hidden="true" />
                  {{ copyButtonLabel }}
                </button>
              </div>
              <div
                :id="`code-panel-${activeCode}`"
                role="tabpanel"
                :aria-labelledby="`code-tab-${activeCode}`"
                class="code-scroll"
              >
                <pre><code data-testid="active-code-example" v-text="activeCodeSnippet"></code></pre>
              </div>
              <p class="sr-only" aria-live="polite">{{ copyLiveMessage }}</p>
            </div>
          </div>
        </div>
      </section>

      <div class="content-shell">
        <ul class="fact-rail" :aria-label="t('home.hero.status')">
          <li v-for="fact in factItems" :key="fact.labelKey">
            <Icon :name="fact.icon" size="sm" aria-hidden="true" />
            <span>{{ t(fact.labelKey) }}</span>
          </li>
        </ul>
      </div>

      <section id="capabilities" class="home-section capability-section" aria-labelledby="capabilities-title">
        <div class="content-shell capability-layout">
          <figure class="dashboard-figure">
            <img
              src="/brand/home-dashboard.webp"
              :alt="t('home.capabilities.imageAlt')"
              width="1600"
              height="757"
              loading="lazy"
              decoding="async"
            />
          </figure>

          <div class="capability-copy">
            <div class="section-intro section-intro--left">
              <h2 id="capabilities-title">{{ t('home.capabilities.title') }}</h2>
              <p>{{ t('home.capabilities.description') }}</p>
            </div>
            <ul class="capability-list">
              <li v-for="item in capabilityItems" :key="item.titleKey">
                <span class="capability-marker" aria-hidden="true">
                  <Icon :name="item.icon" size="sm" />
                </span>
                <div>
                  <h3>{{ t(item.titleKey) }}</h3>
                  <p>{{ t(item.descriptionKey) }}</p>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </section>

      <section id="steps" class="home-section steps-section" aria-labelledby="steps-title">
        <div class="content-shell">
          <div class="section-intro">
            <h2 id="steps-title">{{ t('home.steps.title') }}</h2>
            <p>{{ t('home.steps.description') }}</p>
          </div>

          <ol class="steps-list">
            <li v-for="(step, index) in stepItems" :key="step.titleKey">
              <span class="step-number" aria-hidden="true">{{ index + 1 }}</span>
              <h3>{{ t(step.titleKey) }}</h3>
              <p>{{ t(step.descriptionKey) }}</p>
            </li>
          </ol>

          <div class="section-action">
            <a v-if="tutorialUrl" :href="tutorialUrl" class="text-link">
              {{ t('home.steps.tutorial') }}
              <Icon name="externalLink" size="sm" aria-hidden="true" />
            </a>
          </div>
        </div>
      </section>

      <section id="providers" class="home-section provider-section" aria-labelledby="providers-title">
        <div class="content-shell">
          <div class="section-intro">
            <h2 id="providers-title">{{ t('home.providers.title') }}</h2>
            <p>{{ t('home.providers.description') }}</p>
          </div>

          <div class="provider-grid" role="list">
            <div
              v-for="provider in homeProviders"
              :key="provider.id"
              role="listitem"
              :data-provider="provider.id"
              :data-provider-status="provider.supported ? 'supported' : 'unsupported'"
              :aria-label="`${t(provider.labelKey)}：${t(provider.supported ? 'home.providers.supported' : 'home.providers.unsupported')}`"
              class="provider-card"
              :class="provider.supported ? 'provider-card--supported' : 'provider-card--unsupported'"
            >
              <span class="provider-icon" aria-hidden="true">
                <PlatformIcon :platform="provider.platform" size="lg" />
              </span>
              <span class="provider-name">{{ t(provider.labelKey) }}</span>
              <span class="provider-status">
                {{ t(provider.supported ? 'home.providers.supported' : 'home.providers.unsupported') }}
              </span>
            </div>
          </div>

          <p class="provider-note">{{ t('home.providers.note') }}</p>
        </div>
      </section>

      <section id="faq" class="home-section faq-section" aria-labelledby="faq-title">
        <div class="content-shell faq-layout">
          <div class="section-intro section-intro--left faq-heading">
            <h2 id="faq-title">{{ t('home.faq.title') }}</h2>
            <p>{{ t('home.faq.description') }}</p>
          </div>

          <div class="faq-list">
            <details v-for="item in faqItems" :key="item.questionKey">
              <summary>
                <span>{{ t(item.questionKey) }}</span>
                <Icon name="plus" size="sm" aria-hidden="true" />
              </summary>
              <p>{{ t(item.answerKey) }}</p>
            </details>
          </div>
        </div>
      </section>

      <section class="home-section final-cta-section" aria-labelledby="final-cta-title">
        <div class="content-shell">
          <div class="final-cta-panel">
            <div>
              <h2 id="final-cta-title">{{ t('home.cta.title') }}</h2>
              <p>{{ t('home.cta.description') }}</p>
            </div>
            <div class="final-cta-actions">
              <router-link
                data-testid="final-primary-cta"
                :to="primaryCta.to"
                class="primary-button primary-button--light"
              >
                {{ t('home.cta.button') }}
                <Icon name="arrowRight" size="sm" aria-hidden="true" />
              </router-link>
              <a v-if="tutorialUrl" :href="tutorialUrl" class="cta-text-link">
                {{ t('home.cta.tutorial') }}
              </a>
            </div>
          </div>
        </div>
      </section>
    </main>

  </PublicSiteLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import PublicSiteLayout from '@/components/public/PublicSiteLayout.vue'
import { useClipboard } from '@/composables/useClipboard'
import { sanitizeUrl } from '@/utils/url'
import type { GroupPlatform } from '@/types'

type CodeTab = 'curl' | 'python'
type CopyStatus = 'idle' | 'copied' | 'failed'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '落雪API')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const apiBaseUrl = computed(() => appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const isHomeContentUrl = computed(() => /^https?:\/\//i.test(homeContent.value.trim()))

const tutorialUrl = computed(() => {
  const base = (docUrl.value || '/tutorial-docs/').replace(/#.*$/, '')
  return `${base}#quick-start`
})

function normalizeApiV1Base(value: string): string {
  const fallback = typeof window === 'undefined' ? 'https://luoxueapi.cc' : window.location.origin
  const base = value.trim().replace(/\/+$/, '') || fallback
  return /\/v1$/i.test(base) ? base : `${base}/v1`
}

const apiV1Base = computed(() => normalizeApiV1Base(apiBaseUrl.value))
const chatCompletionsUrl = computed(() => `${apiV1Base.value}/chat/completions`)
const codeExamples = computed<Record<CodeTab, string>>(() => ({
  curl: `curl "${chatCompletionsUrl.value}" \\
  -H "Authorization: Bearer <YOUR_API_KEY>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "<YOUR_MODEL>",
    "messages": [{"role": "user", "content": "你好"}]
  }'`,
  python: `from openai import OpenAI

client = OpenAI(
    api_key="<YOUR_API_KEY>",
    base_url="${apiV1Base.value}",
)

response = client.chat.completions.create(
    model="<YOUR_MODEL>",
    messages=[{"role": "user", "content": "你好"}],
)

print(response.choices[0].message.content)`
}))

const codeTabs = [
  { id: 'curl', labelKey: 'home.codeExample.tabs.curl' },
  { id: 'python', labelKey: 'home.codeExample.tabs.python' }
] as const

const factItems = [
  { labelKey: 'home.facts.gpt', icon: 'checkCircle' },
  { labelKey: 'home.facts.usage', icon: 'chart' },
  { labelKey: 'home.facts.quota', icon: 'key' }
] as const

const capabilityItems = [
  {
    titleKey: 'home.capabilities.items.usage.title',
    descriptionKey: 'home.capabilities.items.usage.description',
    icon: 'chart'
  },
  {
    titleKey: 'home.capabilities.items.keys.title',
    descriptionKey: 'home.capabilities.items.keys.description',
    icon: 'key'
  },
  {
    titleKey: 'home.capabilities.items.diagnostics.title',
    descriptionKey: 'home.capabilities.items.diagnostics.description',
    icon: 'shield'
  }
] as const

const stepItems = [
  {
    titleKey: 'home.steps.items.account.title',
    descriptionKey: 'home.steps.items.account.description'
  },
  {
    titleKey: 'home.steps.items.key.title',
    descriptionKey: 'home.steps.items.key.description'
  },
  {
    titleKey: 'home.steps.items.client.title',
    descriptionKey: 'home.steps.items.client.description'
  }
] as const

const faqItems = ['models', 'group', 'endpoint', 'billing', 'client'].map((id) => ({
  questionKey: `home.faq.items.${id}.question`,
  answerKey: `home.faq.items.${id}.answer`
}))

const homeProviders: ReadonlyArray<{
  id: string
  labelKey: string
  platform: GroupPlatform
  supported: boolean
}> = [
  { id: 'gpt', labelKey: 'home.providers.gpt', platform: 'openai', supported: true },
  { id: 'claude', labelKey: 'home.providers.claude', platform: 'anthropic', supported: false },
  { id: 'gemini', labelKey: 'home.providers.gemini', platform: 'gemini', supported: false },
  {
    id: 'antigravity',
    labelKey: 'home.providers.antigravity',
    platform: 'antigravity',
    supported: false
  }
]

const isAuthenticated = computed(() => authStore.isAuthenticated)
const registrationEnabled = computed(
  () => appStore.cachedPublicSettings?.registration_enabled === true
)
const primaryCta = computed(() => {
  if (isAuthenticated.value) {
    return { to: '/keys', label: t('home.hero.createKey') }
  }
  if (registrationEnabled.value) {
    return { to: '/register', label: t('home.hero.register') }
  }
  return { to: '/login', label: t('home.hero.login') }
})
const codeTabRefs = ref<HTMLButtonElement[]>([])
const activeCode = ref<CodeTab>('curl')
const copyStatus = ref<CopyStatus>('idle')
let copyResetTimer: ReturnType<typeof setTimeout> | undefined

const activeCodeSnippet = computed(() => codeExamples.value[activeCode.value])
const activeCodeLanguage = computed(() =>
  t(activeCode.value === 'curl' ? 'home.codeExample.tabs.curl' : 'home.codeExample.tabs.python')
)
const copyButtonLabel = computed(() => {
  if (copyStatus.value === 'copied') return t('home.codeExample.copied')
  if (copyStatus.value === 'failed') return t('home.codeExample.copyFailed')
  return t('home.codeExample.copy')
})
const copyAriaLabel = computed(() =>
  t(copyStatus.value === 'copied' ? 'home.codeExample.copiedAria' : 'home.codeExample.copyAria', {
    language: activeCodeLanguage.value
  })
)
const copyLiveMessage = computed(() => (copyStatus.value === 'idle' ? '' : copyButtonLabel.value))

function selectCodeTab(tab: CodeTab) {
  activeCode.value = tab
  copyStatus.value = 'idle'
}

function handleCodeTabKeydown(event: KeyboardEvent, currentIndex: number) {
  let nextIndex: number | undefined

  switch (event.key) {
    case 'ArrowRight':
      nextIndex = (currentIndex + 1) % codeTabs.length
      break
    case 'ArrowLeft':
      nextIndex = (currentIndex - 1 + codeTabs.length) % codeTabs.length
      break
    case 'Home':
      nextIndex = 0
      break
    case 'End':
      nextIndex = codeTabs.length - 1
      break
    default:
      return
  }

  event.preventDefault()
  selectCodeTab(codeTabs[nextIndex].id)
  codeTabRefs.value[nextIndex]?.focus()
}

async function copyActiveCode() {
  if (copyResetTimer) clearTimeout(copyResetTimer)
  const didCopy = await copyToClipboard(activeCodeSnippet.value, t('home.codeExample.copied'))
  copyStatus.value = didCopy ? 'copied' : 'failed'
  copyResetTimer = setTimeout(() => {
    copyStatus.value = 'idle'
  }, 2000)
}

onBeforeUnmount(() => {
  if (copyResetTimer) clearTimeout(copyResetTimer)
})
</script>

<style scoped>
.home-page {
  --page: #ffffff;
  --surface: #ffffff;
  --surface-soft: #f4f7f5;
  --surface-accent: #f0fdfa;
  --ink: #0f172a;
  --copy: #475569;
  --muted: #64748b;
  --border: #dde4e0;
  --accent: #0f766e;
  --accent-strong: #0b5f59;
  --accent-ink: #115e59;
  --code: #101713;
  --code-toolbar: #17201c;
  --code-copy: #dff4ed;
  min-height: 100vh;
  overflow-x: clip;
  background: var(--page);
  color: var(--ink);
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC",
    "Microsoft YaHei", sans-serif;
}

.home-page.public-site-page--dark {
  --page: #0e1211;
  --surface: #111816;
  --surface-soft: #1a211e;
  --surface-accent: #17312e;
  --ink: #f4f7f5;
  --copy: #c1cbc6;
  --muted: #9aa6a0;
  --border: #28312c;
  --accent: #5eead4;
  --accent-strong: #99f6e4;
  --accent-ink: #99f6e4;
  --code: #090d0b;
  --code-toolbar: #111816;
  --code-copy: #dff4ed;
}

.home-page :where(a, button, summary):focus-visible {
  outline: 3px solid color-mix(in srgb, var(--accent) 72%, white);
  outline-offset: 3px;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.content-shell,
.home-nav,
.mobile-menu-inner {
  width: min(1120px, calc(100% - 48px));
  margin-inline: auto;
}

.home-header {
  position: sticky;
  top: 0;
  z-index: 40;
  border-bottom: 1px solid transparent;
  background: var(--page);
  transition:
    border-color 170ms ease,
    box-shadow 170ms ease;
}

.home-header--elevated {
  border-bottom-color: var(--border);
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.08);
}

.public-site-page--dark .home-header--elevated {
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.34);
}

.home-nav {
  min-height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.brand-link,
.header-actions,
.desktop-nav-links,
.hero-actions,
.final-cta-actions,
.mobile-menu-footer,
.header-account-link,
.primary-button,
.secondary-button,
.text-link,
.cta-text-link {
  display: flex;
  align-items: center;
}

.brand-link {
  min-height: 44px;
  min-width: 0;
  gap: 12px;
  color: var(--ink);
  text-decoration: none;
}

.brand-mark {
  width: 40px;
  height: 40px;
  flex: 0 0 auto;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}

.brand-mark img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.brand-name {
  overflow: hidden;
  color: var(--ink);
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.02em;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.desktop-nav-links {
  gap: clamp(18px, 2vw, 30px);
}

.desktop-nav-links a,
.mobile-menu-panel a,
.home-footer a {
  color: var(--copy);
  text-decoration: none;
  transition: color 150ms ease;
}

.desktop-nav-links a {
  padding-block: 12px;
  font-size: 14px;
  font-weight: 600;
}

.desktop-nav-links a:hover,
.mobile-menu-panel a:hover,
.home-footer a:hover {
  color: var(--accent-ink);
}

.header-actions {
  flex: 0 0 auto;
  gap: 8px;
}

.icon-button {
  width: 44px;
  height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--copy);
  cursor: pointer;
  transition:
    background-color 150ms ease,
    border-color 150ms ease,
    color 150ms ease;
}

.icon-button:hover {
  border-color: var(--border);
  background: var(--surface-soft);
  color: var(--ink);
}

.header-account-link {
  min-height: 44px;
  gap: 7px;
  border-radius: 10px;
  background: var(--ink);
  color: var(--page);
  padding: 0 16px;
  font-size: 13px;
  font-weight: 700;
  text-decoration: none;
  transition: transform 150ms ease;
}

.header-account-link:hover {
  transform: translateY(-1px);
}

.mobile-menu-button,
.mobile-menu-panel {
  display: none;
}

.hero-section {
  padding-block: clamp(72px, 9vw, 112px) clamp(56px, 7vw, 88px);
}

.hero-layout {
  display: grid;
  grid-template-columns: minmax(0, 0.82fr) minmax(520px, 1.18fr);
  align-items: center;
  gap: clamp(48px, 7vw, 88px);
}

.hero-copy,
.code-showcase,
.capability-copy,
.dashboard-figure,
.faq-heading,
.faq-list {
  min-width: 0;
}

.status-line {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin: 0 0 20px;
  color: var(--accent-ink);
  font-size: 14px;
  font-weight: 700;
}

.status-dot {
  width: 9px;
  height: 9px;
  border-radius: 999px;
  background: var(--accent);
  box-shadow: 0 0 0 5px var(--surface-accent);
}

.hero-copy h1 {
  max-width: 11ch;
  margin: 0;
  color: var(--ink);
  font-size: clamp(2.5rem, 5.1vw, 4rem);
  font-weight: 760;
  letter-spacing: -0.038em;
  line-height: 1.08;
  text-wrap: balance;
}

.hero-description {
  max-width: 35rem;
  margin: 26px 0 0;
  color: var(--copy);
  font-size: 17px;
  line-height: 1.85;
  text-wrap: pretty;
}

.hero-actions {
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 34px;
}

.primary-button,
.secondary-button {
  min-height: 48px;
  justify-content: center;
  gap: 9px;
  border-radius: 10px;
  padding: 0 20px;
  font-size: 14px;
  font-weight: 700;
  text-decoration: none;
  transition:
    transform 150ms ease,
    background-color 150ms ease,
    border-color 150ms ease;
}

.primary-button {
  border: 1px solid var(--ink);
  background: var(--ink);
  color: var(--page);
}

.primary-button:hover,
.secondary-button:hover {
  transform: translateY(-2px);
}

.secondary-button {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--ink);
}

.secondary-button:hover {
  border-color: var(--accent);
}

.code-showcase {
  display: grid;
  gap: 20px;
}

.code-copy h2 {
  margin: 0;
  color: var(--ink);
  font-size: 18px;
  font-weight: 720;
  letter-spacing: -0.02em;
}

.code-copy p {
  max-width: 46rem;
  margin: 8px 0 0;
  color: var(--copy);
  font-size: 14px;
  line-height: 1.65;
}

.code-window {
  min-width: 0;
  overflow: hidden;
  border: 1px solid #28312c;
  border-radius: 14px;
  background: var(--code);
  box-shadow: 0 22px 54px rgba(15, 23, 42, 0.18);
}

.public-site-page--dark .code-window {
  box-shadow: 0 22px 54px rgba(0, 0, 0, 0.38);
}

.code-toolbar {
  min-height: 54px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid #28312c;
  background: var(--code-toolbar);
  padding: 7px 10px 7px 14px;
}

.code-tabs {
  display: flex;
  align-items: center;
  gap: 4px;
}

.code-tabs button,
.copy-button {
  min-height: 44px;
  border: 0;
  border-radius: 8px;
  color: #9fb1a9;
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
}

.code-tabs button {
  background: transparent;
  padding-inline: 12px;
}

.code-tabs button.is-active {
  background: #26322d;
  color: #ffffff;
}

.copy-button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #26322d;
  padding-inline: 11px;
  white-space: nowrap;
}

.code-scroll {
  max-width: 100%;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
}

.code-scroll pre {
  min-width: 540px;
  margin: 0;
  padding: 24px;
}

.code-scroll code {
  display: block;
  color: var(--code-copy);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  line-height: 1.75;
  white-space: pre;
}

.fact-rail {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  margin: 0;
  border-block: 1px solid var(--border);
  padding: 0;
  list-style: none;
}

.fact-rail li {
  min-height: 86px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--ink);
  font-size: 14px;
  font-weight: 680;
}

.fact-rail li + li {
  border-left: 1px solid var(--border);
}

.fact-rail svg {
  color: var(--accent);
}

.home-section {
  scroll-margin-top: 84px;
  padding-block: clamp(76px, 9vw, 108px);
}

.section-intro {
  max-width: 680px;
  margin: 0 auto 48px;
  text-align: center;
}

.section-intro--left {
  margin-inline: 0;
  text-align: left;
}

.section-intro h2,
.final-cta-panel h2 {
  margin: 0;
  color: var(--ink);
  font-size: clamp(2rem, 3.7vw, 2.8rem);
  font-weight: 740;
  letter-spacing: -0.035em;
  line-height: 1.18;
  text-wrap: balance;
}

.section-intro p,
.final-cta-panel p {
  max-width: 65ch;
  margin: 18px auto 0;
  color: var(--copy);
  font-size: 16px;
  line-height: 1.75;
  text-wrap: pretty;
}

.section-intro--left p {
  margin-inline: 0;
}

.capability-section,
.provider-section {
  background: var(--surface-soft);
}

.capability-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.14fr) minmax(340px, 0.86fr);
  align-items: center;
  gap: clamp(44px, 7vw, 86px);
}

.dashboard-figure {
  margin: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: #ffffff;
}

.dashboard-figure img {
  width: 100%;
  height: auto;
  display: block;
}

.capability-list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.capability-list li {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr);
  gap: 15px;
  border-top: 1px solid var(--border);
  padding-block: 22px;
}

.capability-list li:last-child {
  border-bottom: 1px solid var(--border);
}

.capability-marker {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--surface-accent);
  color: var(--accent-ink);
}

.capability-list h3,
.steps-list h3 {
  margin: 0;
  color: var(--ink);
  font-size: 16px;
  font-weight: 720;
  line-height: 1.5;
}

.capability-list p,
.steps-list p {
  margin: 6px 0 0;
  color: var(--copy);
  font-size: 14px;
  line-height: 1.7;
}

.steps-list {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin: 0;
  border-block: 1px solid var(--border);
  padding: 0;
  list-style: none;
}

.steps-list li {
  min-width: 0;
  padding: 32px clamp(22px, 3vw, 38px) 36px;
}

.steps-list li + li {
  border-left: 1px solid var(--border);
}

.step-number {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 24px;
  border-radius: 999px;
  background: var(--ink);
  color: var(--page);
  font-size: 13px;
  font-weight: 760;
}

.section-action {
  display: flex;
  justify-content: center;
  margin-top: 32px;
}

.text-link,
.cta-text-link {
  min-height: 44px;
  gap: 7px;
  color: var(--accent-ink);
  font-size: 14px;
  font-weight: 700;
  text-decoration: none;
}

.text-link:hover,
.cta-text-link:hover {
  text-decoration: underline;
  text-underline-offset: 4px;
}

.provider-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.provider-card {
  min-width: 0;
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr);
  align-items: center;
  gap: 2px 12px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  padding: 18px;
}

.provider-card--supported {
  border-color: color-mix(in srgb, var(--accent) 48%, var(--border));
  background: var(--surface-accent);
}

.provider-icon {
  grid-row: span 2;
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  color: var(--ink);
}

.provider-card--unsupported .provider-icon {
  filter: grayscale(1);
  opacity: 0.58;
}

.provider-name {
  overflow: hidden;
  color: var(--ink);
  font-size: 14px;
  font-weight: 720;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-status {
  color: var(--copy);
  font-size: 12px;
  font-weight: 650;
}

.provider-card--supported .provider-status {
  color: var(--accent-ink);
}

.provider-note {
  margin: 24px 0 0;
  color: var(--copy);
  font-size: 13px;
  line-height: 1.7;
  text-align: center;
}

.faq-layout {
  display: grid;
  grid-template-columns: minmax(260px, 0.7fr) minmax(0, 1.3fr);
  align-items: start;
  gap: clamp(48px, 8vw, 104px);
}

.faq-heading {
  position: sticky;
  top: 112px;
  margin-bottom: 0;
}

.faq-list {
  border-top: 1px solid var(--border);
}

.faq-list details {
  border-bottom: 1px solid var(--border);
}

.faq-list summary {
  min-height: 78px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  color: var(--ink);
  cursor: pointer;
  font-size: 16px;
  font-weight: 690;
  list-style: none;
}

.faq-list summary::-webkit-details-marker {
  display: none;
}

.faq-list summary svg {
  flex: 0 0 auto;
  color: var(--accent-ink);
  transition: transform 180ms ease;
}

.faq-list details[open] summary svg {
  transform: rotate(45deg);
}

.faq-list details > p {
  max-width: 68ch;
  margin: -4px 0 0;
  padding: 0 42px 24px 0;
  color: var(--copy);
  font-size: 14px;
  line-height: 1.78;
}

.final-cta-section {
  padding-top: 32px;
}

.final-cta-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 40px;
  border-radius: 16px;
  background: #0f172a;
  padding: clamp(36px, 6vw, 64px);
}

.final-cta-panel h2 {
  color: #ffffff;
}

.final-cta-panel p {
  margin-inline: 0;
  color: #cbd5e1;
}

.final-cta-actions {
  flex: 0 0 auto;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 12px 20px;
}

.primary-button--light {
  border-color: #ffffff;
  background: #ffffff;
  color: #0f172a;
}

.cta-text-link {
  color: #ccfbf1;
}

.home-footer {
  border-top: 1px solid var(--border);
  padding-block: 30px;
}

.footer-layout {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.home-footer p {
  margin: 0;
  color: var(--muted);
  font-size: 13px;
}

.home-footer nav {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
}

.home-footer a {
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  font-size: 13px;
  font-weight: 620;
}

.mobile-menu-enter-active,
.mobile-menu-leave-active {
  transition:
    opacity 170ms ease,
    transform 170ms cubic-bezier(0.22, 1, 0.36, 1);
}

.mobile-menu-enter-from,
.mobile-menu-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

@media (max-width: 1023px) {
  .desktop-nav-links,
  .desktop-action {
    display: none;
  }

  .mobile-menu-button {
    display: inline-flex;
  }

  .mobile-menu-panel {
    position: absolute;
    top: 100%;
    right: 0;
    left: 0;
    display: block;
    border-bottom: 1px solid var(--border);
    background: var(--page);
    box-shadow: 0 20px 34px rgba(15, 23, 42, 0.1);
  }

  .mobile-menu-inner {
    display: grid;
    padding-block: 12px 20px;
  }

  .mobile-menu-inner > a {
    min-height: 52px;
    display: flex;
    align-items: center;
    border-bottom: 1px solid var(--border);
    font-size: 15px;
    font-weight: 650;
  }

  .mobile-menu-footer {
    justify-content: space-between;
    gap: 16px;
    padding-top: 16px;
  }

  .mobile-menu-footer > a {
    min-height: 44px;
    gap: 7px;
    border-radius: 10px;
    background: var(--ink);
    color: var(--page);
    padding-inline: 16px;
    font-size: 13px;
    font-weight: 700;
  }

  .hero-layout,
  .capability-layout {
    grid-template-columns: minmax(0, 1fr);
  }

  .hero-copy {
    max-width: 760px;
  }

  .hero-copy h1 {
    max-width: 13ch;
  }

  .code-showcase {
    width: 100%;
  }

  .capability-copy {
    display: grid;
    grid-template-columns: minmax(230px, 0.7fr) minmax(0, 1.3fr);
    gap: 48px;
  }

  .provider-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .content-shell,
  .home-nav,
  .mobile-menu-inner {
    width: min(100% - 32px, 1120px);
  }

  .home-nav {
    min-height: 64px;
  }

  .brand-link {
    gap: 9px;
  }

  .brand-mark {
    width: 38px;
    height: 38px;
  }

  .brand-name {
    max-width: 128px;
    font-size: 16px;
  }

  .hero-section {
    padding-block: 58px 52px;
  }

  .hero-layout {
    gap: 48px;
  }

  .hero-copy h1 {
    max-width: 12ch;
    font-size: clamp(2.25rem, 10vw, 2.5rem);
    line-height: 1.12;
  }

  .hero-description {
    margin-top: 22px;
    font-size: 15px;
    line-height: 1.8;
  }

  .hero-actions {
    align-items: stretch;
    margin-top: 28px;
  }

  .hero-actions .primary-button,
  .hero-actions .secondary-button {
    flex: 1 1 190px;
  }

  .code-toolbar {
    align-items: stretch;
    flex-direction: column;
    gap: 8px;
    padding: 10px;
  }

  .code-tabs button {
    flex: 1;
  }

  .code-tabs,
  .copy-button {
    width: 100%;
  }

  .copy-button {
    min-height: 44px;
    justify-content: center;
  }

  .code-scroll pre {
    min-width: 510px;
    padding: 20px;
  }

  .fact-rail {
    grid-template-columns: 1fr;
  }

  .fact-rail li {
    min-height: 62px;
    justify-content: flex-start;
    padding-inline: 8px;
  }

  .fact-rail li + li {
    border-top: 1px solid var(--border);
    border-left: 0;
  }

  .home-section {
    padding-block: 68px;
  }

  .section-intro {
    margin-bottom: 36px;
    text-align: left;
  }

  .section-intro h2,
  .final-cta-panel h2 {
    font-size: clamp(1.8rem, 8vw, 2.25rem);
  }

  .section-intro p,
  .final-cta-panel p {
    margin-inline: 0;
    font-size: 15px;
  }

  .capability-copy {
    display: block;
  }

  .capability-copy .section-intro {
    margin-top: 38px;
  }

  .steps-list {
    grid-template-columns: 1fr;
  }

  .steps-list li {
    padding-inline: 8px;
  }

  .steps-list li + li {
    border-top: 1px solid var(--border);
    border-left: 0;
  }

  .provider-grid {
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  }

  .provider-card {
    padding: 15px;
  }

  .faq-layout {
    grid-template-columns: 1fr;
    gap: 20px;
  }

  .faq-heading {
    position: static;
  }

  .faq-list summary {
    min-height: 72px;
    font-size: 15px;
  }

  .faq-list details > p {
    padding-right: 0;
  }

  .final-cta-section {
    padding-top: 10px;
  }

  .final-cta-panel {
    align-items: stretch;
    flex-direction: column;
    padding: 32px 24px;
  }

  .final-cta-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .footer-layout {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 359px) {
  .brand-name {
    max-width: 92px;
  }

  .header-actions {
    gap: 2px;
  }

  .provider-grid {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-page *,
  .home-page *::before,
  .home-page *::after {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}
</style>
