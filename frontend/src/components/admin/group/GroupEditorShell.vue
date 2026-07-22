<template>
  <div class="group-editor-shell">
    <header class="group-editor-header">
      <button
        type="button"
        class="group-editor-back"
        :aria-label="backLabel"
        @click="emit('back')"
      >
        <Icon name="arrowLeft" size="sm" />
        <span>{{ backLabel }}</span>
      </button>

      <div class="group-editor-heading">
        <div class="group-editor-title-row">
          <div>
            <h1>{{ title }}</h1>
            <p v-if="description">{{ description }}</p>
          </div>
          <span v-if="contextLabel" class="group-editor-context">{{ contextLabel }}</span>
        </div>
      </div>
    </header>

    <div class="group-editor-layout">
      <nav class="group-editor-nav" :aria-label="navigationLabel">
        <template v-for="section in sections" :key="section.id">
          <RouterLink
            v-if="section.to"
            :id="`group-editor-nav-${section.id}`"
            :to="section.to"
            class="group-editor-nav-item"
            :class="{ 'group-editor-nav-item--active': section.id === activeSection }"
            :aria-current="section.id === activeSection ? 'page' : undefined"
          >
            <span class="group-editor-nav-label">{{ section.label }}</span>
            <span v-if="section.description" class="group-editor-nav-description">
              {{ section.description }}
            </span>
          </RouterLink>
          <button
            v-else
            :id="`group-editor-nav-${section.id}`"
            type="button"
            class="group-editor-nav-item"
            :class="{ 'group-editor-nav-item--active': section.id === activeSection }"
            :aria-current="section.id === activeSection ? 'step' : undefined"
            @click="emit('select-section', section.id)"
          >
            <span class="group-editor-nav-label">{{ section.label }}</span>
            <span v-if="section.description" class="group-editor-nav-description">
              {{ section.description }}
            </span>
          </button>
        </template>
      </nav>

      <section class="group-editor-content" :aria-busy="loading">
        <div
          v-if="loading"
          class="group-editor-loading"
          role="status"
          aria-live="polite"
        >
          <span class="sr-only">{{ loadingLabel }}</span>
          <span class="group-editor-loading-line group-editor-loading-line--title"></span>
          <span class="group-editor-loading-line"></span>
          <span class="group-editor-loading-line"></span>
          <span class="group-editor-loading-line group-editor-loading-line--short"></span>
        </div>
        <div v-else>
          <header class="group-editor-section-heading">
            <h2
              id="group-editor-active-title"
              ref="activeSectionHeading"
              tabindex="-1"
            >
              {{ currentSection?.label }}
            </h2>
            <p v-if="currentSection?.description">
              {{ currentSection.description }}
            </p>
          </header>
          <slot />
        </div>
      </section>
    </div>

    <footer class="group-editor-footer">
      <div
        class="group-editor-save-state"
        role="status"
        aria-live="polite"
        aria-atomic="true"
      >
        <span
          class="group-editor-save-dot"
          :class="{ 'group-editor-save-dot--dirty': dirty }"
          aria-hidden="true"
        ></span>
        <span>{{ dirty ? dirtyLabel : savedLabel }}</span>
      </div>
      <div class="group-editor-footer-actions">
        <slot name="footer" />
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'

export interface GroupEditorSection {
  id: string
  label: string
  description?: string
  to?: RouteLocationRaw
}

const props = withDefaults(
  defineProps<{
    title: string
    description?: string
    contextLabel?: string
    backLabel: string
    navigationLabel: string
    sections: GroupEditorSection[]
    activeSection: string
    loading?: boolean
    loadingLabel: string
    dirty?: boolean
    dirtyLabel: string
    savedLabel: string
  }>(),
  {
    description: '',
    contextLabel: '',
    loading: false,
    dirty: false
  }
)

const emit = defineEmits<{
  back: []
  'select-section': [section: string]
}>()

const activeSectionHeading = ref<HTMLElement | null>(null)
const currentSection = computed(() =>
  props.sections.find((section) => section.id === props.activeSection)
)

watch(
  () => [props.activeSection, props.loading] as const,
  async ([, loading], [, previousLoading]) => {
    if (loading) return
    await nextTick()
    if (previousLoading || activeSectionHeading.value) {
      activeSectionHeading.value?.focus({ preventScroll: true })
    }
  },
  { flush: 'post' }
)
</script>

<style scoped>
.group-editor-shell {
  width: min(100%, 1180px);
  margin-inline: auto;
  padding: 1rem 1rem max(1rem, env(safe-area-inset-bottom));
  scroll-margin-top: 5.5rem;
}

.group-editor-header {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 1rem;
}

.group-editor-back {
  display: inline-flex;
  width: fit-content;
  min-height: 44px;
  align-items: center;
  gap: 0.5rem;
  border-radius: var(--lx-clay-radius-control);
  padding: 0.625rem 0.75rem;
  color: var(--lx-clay-text-secondary);
  font-weight: 700;
  transition:
    background-color 150ms ease,
    color 150ms ease;
}

.group-editor-back:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.group-editor-back:focus-visible,
.group-editor-nav-item:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 32%, transparent);
  outline-offset: 2px;
}

.group-editor-heading {
  min-width: 0;
}

.group-editor-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.group-editor-title-row h1 {
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 1.75rem;
  font-weight: 800;
  line-height: 1.2;
  text-wrap: balance;
}

.group-editor-title-row p {
  max-width: 68ch;
  margin-top: 0.375rem;
  color: var(--lx-clay-text-secondary);
  line-height: 1.6;
  text-wrap: pretty;
}

.group-editor-context {
  flex: none;
  border: 1px solid var(--lx-clay-border);
  border-radius: 999px;
  padding: 0.375rem 0.625rem;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
  font-size: 0.75rem;
  font-weight: 700;
}

.group-editor-layout {
  display: grid;
  gap: 1rem;
}

.group-editor-nav {
  display: flex;
  position: sticky;
  z-index: 12;
  top: 5.5rem;
  gap: 0.5rem;
  overflow-x: auto;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  padding: 0.5rem;
  background: color-mix(in srgb, var(--lx-clay-surface) 94%, transparent);
  box-shadow: var(--lx-clay-shadow-flat);
  backdrop-filter: blur(14px);
  scrollbar-width: none;
}

.group-editor-nav::-webkit-scrollbar {
  display: none;
}

.group-editor-nav-item {
  display: flex;
  min-width: max-content;
  min-height: 44px;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--lx-clay-radius-control);
  padding: 0.625rem 0.75rem;
  color: var(--lx-clay-text-secondary);
  text-decoration: none;
  text-align: left;
  transition:
    border-color 150ms ease,
    background-color 150ms ease,
    color 150ms ease;
}

.group-editor-nav-item:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.group-editor-nav-item--active {
  border-color: color-mix(in srgb, var(--lx-clay-accent) 24%, var(--lx-clay-border));
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.group-editor-nav-label {
  font-size: 0.875rem;
  font-weight: 750;
}

.group-editor-nav-description {
  display: none;
  margin-top: 0.125rem;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  line-height: 1.4;
}

.group-editor-content {
  min-width: 0;
  overflow-x: clip;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  padding: 1rem;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-surface);
}

.group-editor-section-heading {
  margin-bottom: 1.5rem;
  border-bottom: 1px solid var(--lx-clay-border);
  padding-bottom: 1rem;
}

.group-editor-section-heading h2 {
  color: var(--lx-clay-text);
  font-size: 1.125rem;
  font-weight: 800;
  line-height: 1.35;
  scroll-margin-block-start: 7rem;
}

.group-editor-section-heading h2:focus {
  outline: none;
}

.group-editor-section-heading h2:focus-visible {
  border-radius: 0.375rem;
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 32%, transparent);
  outline-offset: 4px;
}

.group-editor-section-heading p {
  margin-top: 0.25rem;
  color: var(--lx-clay-text-secondary);
  font-size: 0.875rem;
  line-height: 1.55;
}

.group-editor-loading {
  display: grid;
  gap: 1rem;
  padding: 0.5rem;
}

.group-editor-loading-line {
  width: 100%;
  height: 2.75rem;
  border-radius: var(--lx-clay-radius-control);
  background: var(--lx-clay-recessed);
  animation: group-editor-pulse 1.4s ease-in-out infinite;
}

.group-editor-loading-line--title {
  width: 52%;
  height: 1.5rem;
}

.group-editor-loading-line--short {
  width: 68%;
}

.group-editor-footer {
  display: flex;
  position: sticky;
  z-index: 20;
  bottom: 0;
  min-height: 72px;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 1rem;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  padding: 0.75rem max(1rem, env(safe-area-inset-right)) max(0.75rem, env(safe-area-inset-bottom)) max(1rem, env(safe-area-inset-left));
  background: color-mix(in srgb, var(--lx-clay-surface) 94%, transparent);
  box-shadow: 0 -12px 32px rgba(60, 44, 91, 0.08);
  backdrop-filter: blur(16px);
}

.group-editor-save-state {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 0.5rem;
  color: var(--lx-clay-text-muted);
  font-size: 0.8125rem;
}

.group-editor-save-dot {
  width: 0.5rem;
  height: 0.5rem;
  flex: none;
  border-radius: 999px;
  background: var(--lx-clay-success);
}

.group-editor-save-dot--dirty {
  background: var(--lx-clay-warning);
}

.group-editor-footer-actions {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: flex-end;
}

@keyframes group-editor-pulse {
  0%,
  100% {
    opacity: 0.55;
  }
  50% {
    opacity: 1;
  }
}

@media (min-width: 768px) {
  .group-editor-shell {
    padding: 1.5rem;
  }

  .group-editor-header {
    flex-direction: row;
    align-items: flex-start;
  }

  .group-editor-heading {
    flex: 1;
  }

  .group-editor-content {
    padding: 1.5rem;
  }

  .group-editor-footer {
    padding-inline: 1.5rem;
  }
}

@media (max-width: 639px) {
  .group-editor-title-row {
    flex-wrap: wrap;
  }

  .group-editor-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .group-editor-footer-actions,
  .group-editor-footer-actions :deep(> *) {
    width: 100%;
  }

  .group-editor-footer-actions :deep(.btn) {
    min-height: 44px;
    flex: 1 1 0;
    justify-content: center;
  }
}

@media (min-width: 1024px) {
  .group-editor-shell {
    padding-top: 2rem;
  }

  .group-editor-layout {
    grid-template-columns: 220px minmax(0, 1fr);
    align-items: start;
    gap: 1.25rem;
  }

  .group-editor-nav {
    top: 6.25rem;
    flex-direction: column;
    overflow: visible;
    padding: 0.625rem;
    backdrop-filter: none;
  }

  .group-editor-nav-item {
    width: 100%;
    min-width: 0;
    padding: 0.75rem;
  }

  .group-editor-nav-description {
    display: block;
  }

  .group-editor-content {
    padding: 2rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .group-editor-back,
  .group-editor-nav-item {
    transition: none;
  }

  .group-editor-loading-line {
    animation: none;
  }
}
</style>
