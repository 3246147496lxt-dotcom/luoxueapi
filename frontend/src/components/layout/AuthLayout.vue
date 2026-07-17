<template>
  <div class="min-h-[100dvh] bg-[#edf0ee] text-gray-900 dark:bg-dark-950 dark:text-gray-100">
    <main
      class="mx-auto grid min-h-[100dvh] w-full max-w-[1480px] grid-cols-1 gap-4 p-3 sm:gap-5 sm:p-5 lg:grid-cols-[minmax(0,1.15fr)_minmax(440px,0.85fr)] lg:gap-6 lg:px-8 lg:py-6"
    >
      <section
        class="order-1 flex min-h-[220px] flex-col px-3 py-4 sm:px-5 lg:min-h-[calc(100dvh-3rem)] lg:px-7 lg:py-8 xl:px-10 xl:py-10"
        :aria-label="siteName"
      >
        <div class="flex items-center gap-3.5 lg:gap-4">
            <div
              class="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-2xl border border-white bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800 lg:h-16 lg:w-16"
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
              <p class="mt-0.5 text-sm leading-5 text-gray-500 dark:text-dark-400 lg:hidden">
                {{ siteSubtitle }}
              </p>
            </div>
          </div>

          <div class="my-auto hidden py-12 lg:block">
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

          <p class="mt-6 hidden text-xs text-gray-400 dark:text-dark-500 lg:block">
            &copy; {{ currentYear }} {{ siteName }}
          </p>
      </section>

      <section
        class="order-2 flex min-h-[calc(100dvh-1.5rem)] flex-col rounded-[24px] border border-white/80 bg-white px-5 py-7 shadow-[0_24px_80px_rgba(30,60,52,0.08)] dark:border-dark-700 dark:bg-dark-900 dark:shadow-none sm:min-h-[calc(100dvh-2.5rem)] sm:px-8 sm:py-9 lg:min-h-[calc(100dvh-3rem)] lg:px-10 xl:px-14"
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
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const appStore = useAppStore()

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

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
