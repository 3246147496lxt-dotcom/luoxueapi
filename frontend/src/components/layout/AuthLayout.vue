<template>
  <div
    class="auth-layout min-h-[100dvh] bg-[#edf0ee] text-gray-900 dark:bg-dark-950 dark:text-gray-100"
    :class="{
      'auth-layout--snow': isSnowVariant,
      'auth-layout--dark': isSnowVariant && isDark
    }"
  >
    <main
      class="auth-layout-grid mx-auto grid min-h-[100dvh] w-full max-w-[1480px] grid-cols-1 gap-4 p-3 sm:gap-5 sm:p-5 lg:grid-cols-[minmax(0,1.15fr)_minmax(440px,0.85fr)] lg:gap-6 lg:px-8 lg:py-6"
    >
      <section
        class="auth-story order-1 flex min-h-[220px] flex-col px-3 py-4 sm:px-5 lg:min-h-[calc(100dvh-3rem)] lg:px-7 lg:py-8 xl:px-10 xl:py-10"
        :aria-label="siteName"
      >
        <div class="auth-brand-bar flex items-center justify-between gap-4">
          <div class="auth-brand flex min-w-0 items-center gap-3.5 lg:gap-4">
            <div
              class="auth-brand-mark flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-2xl border border-white bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800 lg:h-16 lg:w-16"
            >
              <img
                :src="siteLogo || '/logo.png'"
                :alt="siteName"
                class="h-full w-full object-contain"
              />
            </div>
            <div class="min-w-0">
              <p class="break-words text-xl font-semibold tracking-tight text-gray-950 dark:text-white lg:text-2xl">
                {{ siteName }}
              </p>
              <p
                v-if="!isSnowVariant"
                class="mt-0.5 text-sm leading-5 text-gray-500 dark:text-dark-400 lg:hidden"
              >
                {{ siteSubtitle }}
              </p>
            </div>
          </div>

          <div v-if="isSnowVariant" class="auth-toolbar flex shrink-0 items-center gap-2">
            <LocaleSwitcher icon-variant="lucide" />
            <button
              type="button"
              class="auth-tool-button"
              :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
              :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
              :aria-pressed="isDark"
              @click="toggleTheme"
            >
              <Icon
                :name="isDark ? 'lucideSun' : 'lucideMoon'"
                size="md"
                :stroke-width="2"
                aria-hidden="true"
              />
            </button>
          </div>
        </div>

        <div v-if="isSnowVariant" class="auth-snow-stage my-auto hidden py-8 lg:flex">
          <div class="auth-snow-hero">
            <div class="auth-snow-copy">
              <p class="auth-snow-title">{{ siteSubtitle }}</p>
            </div>

            <div class="auth-snow-media" aria-hidden="true">
              <img
                src="/brand/luoxue-snowflake-3d.png"
                alt=""
                width="640"
                height="640"
              />
            </div>
          </div>

          <ul class="auth-feature-rail">
            <li>
              <Icon name="grid" size="md" aria-hidden="true" />
              <span>{{ t('nav.dashboard') }}</span>
            </li>
            <li>
              <Icon name="key" size="md" aria-hidden="true" />
              <span>{{ t('nav.apiKeys') }}</span>
            </li>
            <li>
              <Icon name="chart" size="md" aria-hidden="true" />
              <span>{{ t('nav.usage') }}</span>
            </li>
          </ul>
        </div>

        <div v-else class="my-auto hidden py-12 lg:block">
          <h1
            class="max-w-[650px] text-balance text-[clamp(2.6rem,4.7vw,4.8rem)] font-semibold leading-[1.04] tracking-[-0.045em] text-gray-950 dark:text-white"
          >
            {{ siteSubtitle }}
          </h1>

          <div class="mt-10 grid max-w-[680px] grid-cols-[1.15fr_0.85fr] gap-4 xl:mt-12 xl:gap-5">
            <article
              class="row-span-2 flex min-h-[286px] flex-col rounded-[24px] border border-gray-200/80 bg-white p-6 shadow-[0_18px_50px_rgba(30,60,52,0.06)] dark:border-dark-700 dark:bg-dark-800 xl:min-h-[312px] xl:p-7"
            >
              <div
                class="flex h-11 w-11 items-center justify-center rounded-2xl bg-primary-50 text-primary-700 dark:bg-primary-950/60 dark:text-primary-300"
              >
                <Icon name="grid" size="lg" />
              </div>
              <div class="mt-auto">
                <p class="text-3xl font-semibold tracking-[-0.035em] text-gray-950 dark:text-white">
                  {{ t('nav.dashboard') }}
                </p>
                <div class="mt-5 flex flex-wrap items-center gap-x-5 gap-y-2 border-t border-gray-100 pt-5 text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <span class="flex items-center gap-2">
                    <Icon name="server" size="sm" class="text-primary-700 dark:text-primary-300" />
                    {{ t('nav.availableChannels') }}
                  </span>
                  <span class="flex items-center gap-2">
                    <Icon name="cube" size="sm" class="text-primary-700 dark:text-primary-300" />
                    {{ t('nav.subscriptions') }}
                  </span>
                </div>
              </div>
            </article>

            <article
              class="flex min-h-[135px] flex-col justify-between rounded-[24px] bg-[#dff4e9] p-5 text-gray-950 dark:bg-primary-950/50 dark:text-white xl:min-h-[146px] xl:p-6"
            >
              <Icon name="key" size="lg" class="text-primary-800 dark:text-primary-300" />
              <p class="text-lg font-semibold tracking-tight">{{ t('nav.apiKeys') }}</p>
            </article>

            <article
              class="flex min-h-[135px] flex-col justify-between rounded-[24px] bg-[#fff4be] p-5 text-gray-950 dark:bg-[#3a3420] dark:text-white xl:min-h-[146px] xl:p-6"
            >
              <Icon name="chart" size="lg" class="text-[#75631d] dark:text-[#f0dc82]" />
              <p class="text-lg font-semibold tracking-tight">{{ t('nav.usage') }}</p>
            </article>
          </div>
        </div>

        <p class="auth-story-footer mt-6 hidden text-xs text-gray-400 dark:text-dark-500 lg:block">
          &copy; {{ currentYear }} {{ siteName }}
        </p>
      </section>

      <section
        class="auth-form-panel order-2 flex min-h-[calc(100dvh-1.5rem)] flex-col rounded-[24px] border border-white/80 bg-white px-5 py-7 shadow-[0_24px_80px_rgba(30,60,52,0.08)] dark:border-dark-700 dark:bg-dark-900 dark:shadow-none sm:min-h-[calc(100dvh-2.5rem)] sm:px-8 sm:py-9 lg:min-h-[calc(100dvh-3rem)] lg:px-10 xl:px-14"
      >
        <div class="flex flex-1 items-center">
          <div class="mx-auto w-full max-w-[470px] py-5 lg:py-8">
            <slot />
          </div>
        </div>

        <div
          v-if="$slots.footer"
          class="mx-auto mt-3 w-full max-w-[470px] border-t border-gray-100 pt-6 text-center text-sm dark:border-dark-700"
        >
          <slot name="footer" />
        </div>

        <p class="mt-6 text-center text-xs text-gray-400 dark:text-dark-500 lg:hidden">
          &copy; {{ currentYear }} {{ siteName }}
        </p>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const appStore = useAppStore()

const props = withDefaults(defineProps<{
  variant?: 'default' | 'snow'
}>(), {
  variant: 'default'
})

const isSnowVariant = computed(() => props.variant === 'snow')
const isDark = ref(document.documentElement.classList.contains('dark'))

const siteName = computed(() => appStore.siteName || '落雪API')
const siteLogo = computed(() =>
  sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true })
)
const siteSubtitle = computed(
  () =>
    appStore.cachedPublicSettings?.site_subtitle ||
    'Subscription to API Conversion Platform'
)
const currentYear = computed(() => new Date().getFullYear())

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
@font-face {
  font-family: "Nunito";
  src: url("/fonts/nunito-latin-variable.woff2") format("woff2");
  font-weight: 700 900;
  font-style: normal;
  font-display: swap;
}

@font-face {
  font-family: "DM Sans";
  src: url("/fonts/dm-sans-latin-variable.woff2") format("woff2");
  font-weight: 400 700;
  font-style: normal;
  font-display: swap;
}

.auth-layout--snow {
  --auth-canvas: #f4f1fa;
  --auth-surface: #ffffff;
  --auth-recessed: #efebf5;
  --auth-text: #332f3a;
  --auth-copy: #635f69;
  --auth-border: rgba(91, 80, 112, 0.14);
  --auth-violet: #7c3aed;
  --auth-violet-light: #a78bfa;
  --auth-blue: #0b8bed;
  --auth-blue-deep: #075985;
  color: var(--auth-text);
  background:
    radial-gradient(circle at 12% 18%, rgba(11, 139, 237, 0.12), transparent 29rem),
    radial-gradient(circle at 76% 78%, rgba(124, 58, 237, 0.12), transparent 31rem),
    var(--auth-canvas);
  font-family: "DM Sans", "PingFang SC", "Microsoft YaHei", sans-serif;
}

.auth-layout--snow .auth-layout-grid {
  max-width: 1440px;
  gap: clamp(22px, 3vw, 42px);
}

.auth-layout--snow .auth-story {
  min-width: 0;
}

.auth-layout--snow .auth-brand-bar {
  position: relative;
  z-index: 2;
  padding: 10px 12px;
  border: 1px solid var(--auth-border);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.7);
  box-shadow:
    12px 12px 28px rgba(160, 150, 180, 0.15),
    -8px -8px 18px rgba(255, 255, 255, 0.72),
    inset 4px 4px 10px rgba(139, 92, 246, 0.025),
    inset -4px -4px 10px rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
}

.auth-layout--snow .auth-brand-mark,
.auth-layout--snow .auth-form-panel,
.auth-layout--snow .auth-snow-media,
.auth-layout--snow .auth-feature-rail {
  border-color: var(--auth-border);
  background: var(--auth-surface);
  box-shadow:
    16px 16px 32px rgba(160, 150, 180, 0.18),
    -10px -10px 24px rgba(255, 255, 255, 0.88),
    inset 6px 6px 12px rgba(139, 92, 246, 0.025),
    inset -6px -6px 12px rgba(255, 255, 255, 0.9);
}

.auth-layout--snow .auth-brand-mark {
  border-radius: 18px;
}

.auth-layout--snow .auth-brand p:first-child {
  color: var(--auth-text);
  font-family: "Nunito", "PingFang SC", sans-serif;
  font-weight: 900;
}

.auth-layout--snow .auth-toolbar :deep(button),
.auth-layout--snow .auth-tool-button {
  display: inline-flex;
  width: 44px;
  min-width: 44px;
  height: 44px;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--auth-border);
  border-radius: 16px;
  color: var(--auth-blue-deep);
  background: var(--auth-recessed);
  box-shadow:
    inset 6px 6px 12px rgba(91, 80, 112, 0.09),
    inset -6px -6px 12px rgba(255, 255, 255, 0.74);
  cursor: pointer;
  transition:
    color 180ms ease,
    transform 180ms cubic-bezier(0.22, 1, 0.36, 1),
    box-shadow 180ms ease;
}

.auth-layout--snow .auth-toolbar :deep(button:hover),
.auth-layout--snow .auth-tool-button:hover {
  color: var(--auth-violet);
  transform: translateY(-2px);
}

.auth-layout--snow .auth-toolbar :deep(button:active),
.auth-layout--snow .auth-tool-button:active {
  transform: scale(0.96);
}

.auth-layout--snow .auth-toolbar :deep(button:focus-visible),
.auth-layout--snow .auth-tool-button:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--auth-violet) 42%, transparent);
  outline-offset: 3px;
}

.auth-layout--snow .auth-snow-stage {
  width: 100%;
  max-width: 720px;
  flex-direction: column;
  gap: 28px;
}

.auth-layout--snow .auth-snow-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(250px, 0.9fr);
  align-items: center;
  gap: 18px;
}

.auth-layout--snow .auth-snow-copy {
  min-width: 0;
}

.auth-layout--snow .auth-snow-title {
  max-width: 11ch;
  margin: 0;
  color: var(--auth-text);
  font-family: "Nunito", "PingFang SC", sans-serif;
  font-size: clamp(3rem, 4.4vw, 4.35rem);
  font-weight: 900;
  line-height: 1.04;
  letter-spacing: -0.035em;
  text-wrap: balance;
}

.auth-layout--snow .auth-snow-media {
  position: relative;
  width: min(100%, 310px);
  justify-self: end;
  padding: 10px;
  border: 1px solid var(--auth-border);
  border-radius: 46px;
  animation: auth-snow-float 8s ease-in-out infinite;
}

.auth-layout--snow .auth-snow-media::before {
  position: absolute;
  z-index: -1;
  inset: 12% -8% -10%;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(11, 139, 237, 0.18), transparent 68%);
  filter: blur(20px);
  content: "";
}

.auth-layout--snow .auth-snow-media img {
  display: block;
  width: 100%;
  height: auto;
  border-radius: 37px;
}

.auth-layout--snow .auth-feature-rail {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin: 0;
  padding: 8px;
  overflow: hidden;
  border: 1px solid var(--auth-border);
  border-radius: 28px;
  list-style: none;
}

.auth-layout--snow .auth-feature-rail li {
  display: flex;
  min-width: 0;
  min-height: 74px;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 14px 12px;
  color: var(--auth-text);
  font-size: 14px;
  font-weight: 700;
  text-align: center;
}

.auth-layout--snow .auth-feature-rail li + li {
  border-left: 1px solid var(--auth-border);
}

.auth-layout--snow .auth-feature-rail li:nth-child(1) svg {
  color: var(--auth-blue);
}

.auth-layout--snow .auth-feature-rail li:nth-child(2) svg {
  color: var(--auth-violet);
}

.auth-layout--snow .auth-feature-rail li:nth-child(3) svg {
  color: #10b981;
}

.auth-layout--snow .auth-form-panel {
  position: relative;
  isolation: isolate;
  overflow: hidden;
  border-radius: 40px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
}

.auth-layout--snow .auth-form-panel::before {
  position: absolute;
  z-index: -1;
  top: -180px;
  right: -160px;
  width: 380px;
  height: 380px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(167, 139, 250, 0.14), transparent 68%);
  pointer-events: none;
  content: "";
}

.auth-layout--snow .auth-story-footer {
  color: var(--auth-copy);
}

@keyframes auth-snow-float {
  0%,
  100% {
    transform: translateY(0);
  }

  50% {
    transform: translateY(-10px);
  }
}

@media (max-width: 1023px) {
  .auth-layout--snow .auth-layout-grid {
    gap: 12px;
    padding: 12px;
  }

  .auth-layout--snow .auth-story {
    min-height: auto;
    padding: 0;
  }

  .auth-layout--snow .auth-form-panel {
    min-height: auto;
    border-radius: 30px;
  }
}

@media (max-width: 479px) {
  .auth-layout--snow .auth-layout-grid {
    padding: 8px;
  }

  .auth-layout--snow .auth-brand-bar {
    padding: 8px 9px;
    border-radius: 22px;
  }

  .auth-layout--snow .auth-brand-mark {
    width: 44px;
    height: 44px;
    border-radius: 16px;
  }

  .auth-layout--snow .auth-brand p:first-child {
    font-size: 18px;
  }

  .auth-layout--snow .auth-toolbar {
    gap: 4px;
  }
}

.auth-layout--snow.auth-layout--dark {
  --auth-canvas: #17131f;
  --auth-surface: #251e2f;
  --auth-recessed: #120f18;
  --auth-text: #f8f5fc;
  --auth-copy: #c5bccf;
  --auth-border: rgba(255, 255, 255, 0.12);
  --auth-violet: #a78bfa;
  --auth-blue: #38bdf8;
  --auth-blue-deep: #8dccff;
}

.auth-layout--snow.auth-layout--dark .auth-brand-bar,
.auth-layout--snow.auth-layout--dark .auth-brand-mark,
.auth-layout--snow.auth-layout--dark .auth-form-panel,
.auth-layout--snow.auth-layout--dark .auth-snow-media,
.auth-layout--snow.auth-layout--dark .auth-feature-rail {
  background: rgba(37, 30, 47, 0.9);
  box-shadow:
    14px 14px 30px rgba(0, 0, 0, 0.34),
    -8px -8px 20px rgba(255, 255, 255, 0.025),
    inset 5px 5px 10px rgba(255, 255, 255, 0.02);
}

.auth-layout--snow.auth-layout--dark .auth-toolbar :deep(button),
.auth-layout--snow.auth-layout--dark .auth-tool-button {
  color: var(--auth-blue-deep);
  background: var(--auth-recessed);
  box-shadow:
    inset 6px 6px 12px rgba(0, 0, 0, 0.3),
    inset -6px -6px 12px rgba(255, 255, 255, 0.025);
}

@media (prefers-reduced-motion: reduce) {
  .auth-layout--snow .auth-snow-media {
    animation: none;
  }

  .auth-layout--snow .auth-toolbar :deep(button),
  .auth-layout--snow .auth-tool-button {
    transition-duration: 0.01ms;
  }
}
</style>
