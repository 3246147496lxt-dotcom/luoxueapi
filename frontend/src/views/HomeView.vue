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
      <section class="content-shell hero-section" aria-labelledby="home-hero-title">
        <div class="hero-copy">
          <h1 id="home-hero-title" class="hero-title">
            <span class="hero-title-lead">{{ heroTitleParts.lead }}</span><br v-if="heroTitleParts.accent">
            <span v-if="heroTitleParts.accent" class="hero-title-accent">{{ heroTitleParts.accent }}</span>
          </h1>

          <p class="hero-description">{{ t('home.hero.description') }}</p>

          <div class="hero-actions">
            <router-link
              data-testid="hero-primary-cta"
              :to="primaryCta.to"
              class="clay-button-primary hero-primary-action"
            >
              {{ primaryCta.label }}
              <Icon name="lucideSparkles" size="sm" :stroke-width="2" aria-hidden="true" />
            </router-link>
            <a v-if="tutorialUrl" :href="tutorialUrl" class="clay-recessed hero-secondary-action">
              {{ t('home.hero.tutorial') }}
            </a>
          </div>
        </div>

        <div class="hero-stage">
          <div class="code-well" aria-labelledby="code-example-title">
            <h2 id="code-example-title" class="sr-only">{{ t('home.codeExample.title') }}</h2>
            <div class="code-window-header">
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
                  aria-controls="code-example-panel"
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
                data-testid="copy-code-button"
                :aria-label="copyAriaLabel"
                @click="copyActiveCode"
              >
                <Icon
                  :name="copyStatus === 'copied' ? 'check' : 'lucideCopy'"
                  size="xs"
                  :stroke-width="2"
                  aria-hidden="true"
                />
                {{ copyButtonLabel }}
              </button>
            </div>

            <p class="code-helper">{{ t('home.codeExample.description') }}</p>

            <div
              id="code-example-panel"
              role="tabpanel"
              :aria-labelledby="`code-tab-${activeCode}`"
              class="code-scroll"
              tabindex="0"
            >
              <pre><code data-testid="active-code-example" v-text="activeCodeSnippet"></code></pre>
            </div>
            <p class="sr-only" aria-live="polite">{{ copyLiveMessage }}</p>
          </div>
        </div>
      </section>

      <div class="content-shell fact-section">
        <ul class="clay-card fact-rail" :aria-label="t('home.hero.status')">
          <li v-for="(fact, index) in factItems" :key="fact.labelKey" :class="`fact-item--${index + 1}`">
            <Icon :name="fact.icon" size="lg" :stroke-width="2" aria-hidden="true" />
            <span>{{ t(fact.labelKey) }}</span>
          </li>
        </ul>
      </div>

      <section id="steps" class="content-shell home-section" aria-labelledby="steps-title">
        <div class="section-intro">
          <h2 id="steps-title">{{ t('home.steps.title') }}</h2>
          <p>{{ t('home.steps.description') }}</p>
        </div>

        <ol class="clay-card steps-list">
          <li v-for="(step, index) in stepItems" :key="step.titleKey" class="step-item">
            <span class="step-node" aria-hidden="true">{{ index + 1 }}</span>
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
      </section>

      <section id="capabilities" class="content-shell home-section" aria-labelledby="capabilities-title">
        <div class="section-intro">
          <h2 id="capabilities-title">{{ t('home.capabilities.title') }}</h2>
          <p>{{ t('home.capabilities.description') }}</p>
        </div>

        <div class="capability-layout">
          <figure class="clay-card dashboard-figure">
            <div class="clay-recessed dashboard-frame">
              <img
                class="dashboard-image dashboard-image--light"
                src="/brand/home-dashboard.webp"
                :alt="t('home.capabilities.imageAlt')"
                width="1600"
                height="757"
                loading="lazy"
                decoding="async"
              />
              <img
                class="dashboard-image dashboard-image--dark"
                src="/brand/home-dashboard-dark.png"
                :alt="t('home.capabilities.imageAlt')"
                width="1600"
                height="757"
                loading="lazy"
                decoding="async"
              />
            </div>
          </figure>

          <ul class="clay-card capability-list">
            <li v-for="(item, index) in capabilityItems" :key="item.titleKey" class="capability-item">
              <span class="clay-recessed capability-marker" :class="`capability-marker--${index + 1}`" aria-hidden="true">
                <Icon :name="item.icon" size="md" />
              </span>
              <h3>{{ t(item.titleKey) }}</h3>
              <p>{{ t(item.descriptionKey) }}</p>
            </li>
          </ul>
        </div>
      </section>

      <section id="providers" class="content-shell home-section" aria-labelledby="providers-title">
        <div class="clay-card provider-panel">
          <div class="provider-intro">
            <h2 id="providers-title">{{ t('home.providers.title') }}</h2>
            <p>{{ t('home.providers.description') }} {{ t('home.providers.note') }}</p>
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
              <span class="clay-recessed provider-icon" aria-hidden="true">
                <PlatformIcon :platform="provider.platform" size="lg" />
              </span>
              <span class="provider-name">{{ t(provider.labelKey) }}</span>
              <span class="provider-status">
                {{ t(provider.supported ? 'home.providers.supported' : 'home.providers.unsupported') }}
              </span>
            </div>
          </div>
        </div>
      </section>

      <section id="faq" class="content-shell home-section faq-layout" aria-labelledby="faq-title">
        <div class="faq-heading">
          <h2 id="faq-title">{{ t('home.faq.title') }}</h2>
          <p>{{ t('home.faq.description') }}</p>
        </div>

        <div class="clay-card faq-list">
          <details
            v-for="(item, index) in faqItems"
            :key="item.questionKey"
            :open="index === 0"
            @toggle="handleFaqToggle"
          >
            <summary>
              <span>{{ t(item.questionKey) }}</span>
              <Icon name="chevronDown" size="sm" aria-hidden="true" />
            </summary>
            <p>{{ t(item.answerKey) }}</p>
          </details>
        </div>
      </section>

      <section class="content-shell home-section final-cta-section" aria-labelledby="final-cta-title">
        <div class="clay-card final-cta-panel">
          <span class="final-cta-glow" aria-hidden="true"></span>
          <div class="final-cta-copy">
            <h2 id="final-cta-title">{{ t('home.cta.title') }}</h2>
            <p>{{ t('home.cta.description') }}</p>
          </div>
          <div class="final-cta-actions">
            <router-link
              data-testid="final-primary-cta"
              :to="primaryCta.to"
              class="final-primary-action"
            >
              {{ t('home.cta.button') }}
              <Icon name="arrowRight" size="sm" aria-hidden="true" />
            </router-link>
            <a v-if="tutorialUrl" :href="tutorialUrl" class="cta-text-link">
              {{ t('home.cta.tutorial') }}
            </a>
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
import { resolveTutorialUrl } from '@/utils/documentationUrl'
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
const heroTitleParts = computed(() => {
  const title = t('home.hero.title')
  const [, lead = title, accent = ''] = title.match(/^(.*?)[，,]\s*(.+)$/) || []
  return { lead, accent }
})

const tutorialUrl = computed(() => resolveTutorialUrl(docUrl.value))

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
  { labelKey: 'home.facts.gpt', icon: 'lucideCheckCircle' },
  { labelKey: 'home.facts.usage', icon: 'lucideBarChart3' },
  { labelKey: 'home.facts.quota', icon: 'lucideShieldCheck' }
] as const

const capabilityItems = [
  {
    titleKey: 'home.capabilities.items.usage.title',
    descriptionKey: 'home.capabilities.items.usage.description',
    icon: 'search'
  },
  {
    titleKey: 'home.capabilities.items.keys.title',
    descriptionKey: 'home.capabilities.items.keys.description',
    icon: 'cog'
  },
  {
    titleKey: 'home.capabilities.items.diagnostics.title',
    descriptionKey: 'home.capabilities.items.diagnostics.description',
    icon: 'zap'
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
let copyRequestId = 0
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
  copyRequestId += 1
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

function handleFaqToggle(event: Event) {
  const current = event.currentTarget as HTMLDetailsElement
  if (!current.open) return

  current.parentElement?.querySelectorAll<HTMLDetailsElement>('details[open]').forEach((detail) => {
    if (detail !== current) detail.removeAttribute('open')
  })
}

async function copyActiveCode() {
  if (copyResetTimer) clearTimeout(copyResetTimer)
  const requestId = ++copyRequestId
  const snippet = activeCodeSnippet.value
  const didCopy = await copyToClipboard(snippet, t('home.codeExample.copied'))
  if (requestId !== copyRequestId) return

  copyStatus.value = didCopy ? 'copied' : 'failed'
  copyResetTimer = setTimeout(() => {
    copyStatus.value = 'idle'
  }, 2000)
}

onBeforeUnmount(() => {
  copyRequestId += 1
  if (copyResetTimer) clearTimeout(copyResetTimer)
})
</script>

<style scoped src="./HomeView.clay.css"></style>
