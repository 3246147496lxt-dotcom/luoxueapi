<template>
  <PublicSiteLayout class="skill-market-page" page="skills">
    <!--
      THESIS: Start with the task, then let an official catalog turn intent into one safe local Codex capability.
      OWN-WORLD: Snow Clay surfaces, violet task routes, recessed filters, and precise code moments inside a calm public library.
      STORY: Describe a job, narrow by task, inspect an official Skill, then continue to a versioned local install.
      FIRST VIEWPORT: A task-sized search field leads the page while a compact find-copy-verify proof explains the mechanism beside it.
      FORM: A task compass opens into featured and latest shelves; structural seed 41a98e36, assigned form five.
      FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, and DESIGN.md
    -->
    <main id="top" class="skill-market-main">
      <section class="skill-market-hero" aria-labelledby="skill-market-title">
        <div class="skill-market-shell skill-market-hero__grid">
          <div class="skill-market-hero__copy">
            <h1 id="skill-market-title">
              <span>{{ t('skills.market.titleLine1') }}</span>
              <span>{{ t('skills.market.titleLine2') }}</span>
            </h1>
            <p>{{ t('skills.market.description') }}</p>

            <form class="skill-market-search" role="search" @submit.prevent="applySearchNow">
              <label class="skill-market-sr-only" for="skill-market-search-input">
                {{ t('skills.market.searchLabel') }}
              </label>
              <Icon name="search" size="md" aria-hidden="true" />
              <input
                id="skill-market-search-input"
                v-model="searchInput"
                type="search"
                autocomplete="off"
                :placeholder="t('skills.market.searchPlaceholder')"
              />
              <button type="submit">{{ t('skills.market.searchAction') }}</button>
            </form>

            <ul class="skill-market-trust" :aria-label="t('skills.market.trustLabel')">
              <li><Icon name="shield" size="sm" aria-hidden="true" />{{ t('skills.market.officialOnly') }}</li>
              <li><Icon name="download" size="sm" aria-hidden="true" />{{ t('skills.market.localCodex') }}</li>
              <li><Icon name="badge" size="sm" aria-hidden="true" />{{ t('skills.market.versioned') }}</li>
            </ul>
          </div>

          <div class="skill-market-proof" :aria-label="t('skills.market.preview.ariaLabel')">
            <div class="skill-market-proof__task">
              <Icon name="chat" size="sm" aria-hidden="true" />
              <span>{{ t('skills.market.preview.task') }}</span>
            </div>
            <div class="skill-market-proof__match">
              <span class="skill-market-proof__mark" aria-hidden="true">
                <Icon name="sparkles" size="lg" :stroke-width="1.8" />
              </span>
              <div>
                <strong>{{ t('skills.market.preview.matchTitle') }}</strong>
                <p>{{ t('skills.market.preview.matchDescription') }}</p>
              </div>
              <span class="skill-market-proof__version">v1.0.0</span>
            </div>
            <ol class="skill-market-proof__route">
              <li><span>1</span>{{ t('skills.market.preview.find') }}</li>
              <li><span>2</span>{{ t('skills.market.preview.copy') }}</li>
              <li><span>3</span>{{ t('skills.market.preview.verify') }}</li>
            </ol>
            <p class="skill-market-proof__note">
              <Icon name="infoCircle" size="sm" aria-hidden="true" />
              {{ t('skills.market.preview.note') }}
            </p>
          </div>
        </div>
      </section>

      <section class="skill-market-shell skill-market-compass" aria-labelledby="skill-category-title">
        <div class="skill-market-section-heading skill-market-section-heading--compact">
          <div>
            <h2 id="skill-category-title">{{ t('skills.market.categoriesTitle') }}</h2>
            <p>{{ t('skills.market.categoriesDescription') }}</p>
          </div>
          <button
            v-if="hasFilters"
            type="button"
            class="skill-market-clear"
            @click="clearFilters"
          >
            <Icon name="x" size="xs" aria-hidden="true" />
            {{ t('skills.market.clearFilters') }}
          </button>
        </div>

        <div class="skill-market-categories" role="group" :aria-label="t('skills.market.categoriesTitle')">
          <button
            type="button"
            :class="{ 'is-active': selectedCategory === '' }"
            :aria-pressed="selectedCategory === ''"
            @click="selectCategory('')"
          >
            <Icon name="grid" size="sm" aria-hidden="true" />
            <span>{{ t('skills.categories.all') }}</span>
            <small v-if="catalogTotal > 0">{{ catalogTotal }}</small>
          </button>
          <button
            v-for="category in categories"
            :key="category.slug"
            type="button"
            :class="{ 'is-active': selectedCategory === category.slug }"
            :aria-pressed="selectedCategory === category.slug"
            @click="selectCategory(category.slug)"
          >
            <Icon :name="categoryIcon(category.slug)" size="sm" aria-hidden="true" />
            <span>{{ categoryLabel(category) }}</span>
            <small v-if="category.count != null">{{ category.count }}</small>
          </button>
        </div>
      </section>

      <section
        v-if="showFeatured"
        class="skill-market-featured"
        aria-labelledby="skill-featured-title"
      >
        <div class="skill-market-shell">
          <div class="skill-market-section-heading">
            <div>
              <h2 id="skill-featured-title">{{ t('skills.market.featuredTitle') }}</h2>
              <p>{{ t('skills.market.featuredDescription') }}</p>
            </div>
            <span>{{ t('skills.market.officialSelection') }}</span>
          </div>

          <div class="skill-market-featured__grid">
            <SkillCard
              v-for="skill in featuredSkills"
              :key="skill.slug"
              :skill="skill"
              :category-name="categoryNameFor(skill.category)"
            />
          </div>
        </div>
      </section>

      <section class="skill-market-shell skill-market-results" aria-labelledby="skill-results-title">
        <div class="skill-market-section-heading">
          <div>
            <h2 id="skill-results-title">
              {{ t(hasFilters ? 'skills.market.resultsTitle' : 'skills.market.latestTitle') }}
            </h2>
            <p>{{ t(hasFilters ? 'skills.market.resultsDescription' : 'skills.market.latestDescription') }}</p>
          </div>
          <span v-if="!loading && !errorState" aria-live="polite">
            {{ t('skills.market.resultCount', { count: catalogTotal }) }}
          </span>
        </div>

        <div v-if="loading" class="skill-market-grid" aria-busy="true" :aria-label="t('skills.market.loading')">
          <article v-for="index in 6" :key="index" class="skill-market-skeleton" aria-hidden="true">
            <span></span><span></span><span></span><span></span>
          </article>
        </div>

        <div v-else-if="errorState" class="skill-market-state" role="alert">
          <span aria-hidden="true"><Icon :name="errorState === 'unavailable' ? 'inbox' : 'exclamationCircle'" size="lg" /></span>
          <h2>{{ t(errorState === 'unavailable' ? 'skills.market.unavailableTitle' : 'skills.market.errorTitle') }}</h2>
          <p>{{ t(errorState === 'unavailable' ? 'skills.market.unavailableDescription' : 'skills.market.errorDescription') }}</p>
          <button v-if="errorState === 'error'" type="button" @click="loadCatalog">
            <Icon name="refresh" size="sm" aria-hidden="true" />
            {{ t('skills.market.retry') }}
          </button>
        </div>

        <div v-else-if="skills.length === 0" class="skill-market-state" role="status">
          <span aria-hidden="true"><Icon :name="hasFilters ? 'search' : 'inbox'" size="lg" /></span>
          <h2>{{ t(hasFilters ? 'skills.market.noResultsTitle' : 'skills.market.emptyTitle') }}</h2>
          <p>{{ t(hasFilters ? 'skills.market.noResultsDescription' : 'skills.market.emptyDescription') }}</p>
          <button v-if="hasFilters" type="button" @click="clearFilters">
            {{ t('skills.market.clearFilters') }}
          </button>
        </div>

        <div v-else class="skill-market-grid">
          <SkillCard
            v-for="skill in skills"
            :key="skill.slug"
            :skill="skill"
            :category-name="categoryNameFor(skill.category)"
          />
        </div>

        <nav v-if="pageCount > 1" class="skill-market-pagination" :aria-label="t('skills.market.paginationLabel')">
          <button type="button" :disabled="page <= 1" @click="page -= 1">
            <Icon name="chevronLeft" size="sm" aria-hidden="true" />
            {{ t('skills.market.previous') }}
          </button>
          <span>{{ t('skills.market.pageStatus', { page, pages: pageCount }) }}</span>
          <button type="button" :disabled="page >= pageCount" @click="page += 1">
            {{ t('skills.market.next') }}
            <Icon name="chevronRight" size="sm" aria-hidden="true" />
          </button>
        </nav>
      </section>
    </main>
  </PublicSiteLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PublicSiteLayout from '@/components/public/PublicSiteLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import SkillCard from '@/components/skills/SkillCard.vue'
import {
  listPublicSkills,
  type PublicSkill,
  type SkillCategory,
} from '@/api/skills'

const PAGE_SIZE = 12

const { t, te } = useI18n()
const searchInput = ref('')
const appliedSearch = ref('')
const selectedCategory = ref('')
const skills = ref<PublicSkill[]>([])
const featuredSkills = ref<PublicSkill[]>([])
const categories = ref<SkillCategory[]>([])
const catalogTotal = ref(0)
const page = ref(1)
const loading = ref(true)
const errorState = ref<'unavailable' | 'error' | null>(null)
let searchTimer: ReturnType<typeof setTimeout> | null = null
let catalogController: AbortController | null = null
let featuredController: AbortController | null = null

const hasFilters = computed(() => Boolean(appliedSearch.value || selectedCategory.value))
const showFeatured = computed(() => !hasFilters.value && featuredSkills.value.length > 0)
const pageCount = computed(() => Math.max(1, Math.ceil(catalogTotal.value / PAGE_SIZE)))

watch(searchInput, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    appliedSearch.value = searchInput.value.trim()
  }, 320)
})

watch([appliedSearch, selectedCategory, page], loadCatalog)

function applySearchNow() {
  if (searchTimer) clearTimeout(searchTimer)
  const nextSearch = searchInput.value.trim()
  const pageWillChange = page.value !== 1
  page.value = 1
  if (nextSearch === appliedSearch.value) {
    if (!pageWillChange) loadCatalog()
  } else {
    appliedSearch.value = nextSearch
  }
}

function selectCategory(category: string) {
  page.value = 1
  selectedCategory.value = category
}

function clearFilters() {
  searchInput.value = ''
  appliedSearch.value = ''
  selectedCategory.value = ''
  page.value = 1
}

function categoryLabel(category: SkillCategory) {
  const key = `skills.categories.${category.slug}`
  return te(key) ? t(key) : category.label
}

function categoryNameFor(slug: string) {
  const category = categories.value.find((item) => item.slug === slug)
  return category ? categoryLabel(category) : slug
}

function categoryIcon(slug: string): 'terminal' | 'shield' | 'document' | 'beaker' | 'grid' {
  const normalized = slug.toLowerCase()
  if (normalized.includes('test') || normalized.includes('debug')) return 'beaker'
  if (normalized.includes('quality') || normalized.includes('security')) return 'shield'
  if (normalized.includes('doc') || normalized.includes('data')) return 'document'
  if (normalized.includes('api') || normalized.includes('automation')) return 'terminal'
  return 'grid'
}

async function loadCatalog() {
  catalogController?.abort()
  const controller = new AbortController()
  catalogController = controller
  loading.value = true
  errorState.value = null
  try {
    const result = await listPublicSkills({
      search: appliedSearch.value,
      category: selectedCategory.value,
      page: page.value,
      page_size: PAGE_SIZE,
    }, { signal: controller.signal })
    skills.value = result.items
    catalogTotal.value = result.total
    if (result.categories.length) categories.value = result.categories
  } catch (error) {
    if (!controller.signal.aborted) {
      const status = extractErrorStatus(error)
      errorState.value = status === 404 ? 'unavailable' : 'error'
    }
  } finally {
    if (!controller.signal.aborted) loading.value = false
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

async function loadFeatured() {
  featuredController?.abort()
  const controller = new AbortController()
  featuredController = controller
  try {
    const result = await listPublicSkills({ featured: true, page: 1, page_size: 4 }, {
      signal: controller.signal,
    })
    featuredSkills.value = result.items
    if (!categories.value.length && result.categories.length) categories.value = result.categories
  } catch {
    // The latest list remains useful when the optional featured shelf cannot load.
  }
}

onMounted(() => {
  loadCatalog()
  loadFeatured()
})

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
  catalogController?.abort()
  featuredController?.abort()
})
</script>

<style src="./SkillMarketplaceView.css"></style>
