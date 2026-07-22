<template>
  <div class="documentation-preview">
    <div
      v-if="content.tutorials.length"
      class="flex gap-1 overflow-x-auto rounded-xl bg-gray-100 p-1 dark:bg-dark-700"
      role="tablist"
      :aria-label="t('admin.documentation.previewCategories')"
    >
      <button
        v-for="(tutorial, index) in content.tutorials"
        :id="`documentation-preview-tab-${tutorial.id}`"
        :key="tutorial.id"
        type="button"
        role="tab"
        class="min-h-10 flex-1 whitespace-nowrap rounded-lg px-3 text-sm font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40"
        :class="
          selectedTutorial?.id === tutorial.id
            ? 'bg-white text-gray-950 dark:bg-dark-800 dark:text-white'
            : 'text-gray-600 hover:text-gray-950 dark:text-dark-300 dark:hover:text-white'
        "
        :aria-selected="selectedTutorial?.id === tutorial.id"
        :aria-controls="`documentation-preview-panel-${tutorial.id}`"
        :tabindex="selectedTutorial?.id === tutorial.id ? 0 : -1"
        @click="selectedId = tutorial.id"
        @keydown.left.prevent="moveTab(index, -1)"
        @keydown.right.prevent="moveTab(index, 1)"
      >
        <DocumentationCategoryIcon
          :icon="tutorial.icon"
          :icon-svg="tutorial.icon_svg"
          class="mr-1 inline-block align-text-bottom"
        />
        {{ tutorial.tab_label || t('admin.documentation.untitledCategory') }}
      </button>
    </div>

    <div
      v-if="selectedTutorial"
      :id="`documentation-preview-panel-${selectedTutorial.id}`"
      class="mt-7"
      role="tabpanel"
      :aria-labelledby="`documentation-preview-tab-${selectedTutorial.id}`"
    >
      <header class="border-b border-gray-200 pb-5 dark:border-dark-700">
        <h3 class="text-xl font-semibold text-gray-950 dark:text-white">
          {{ selectedTutorial.tab_label || t('admin.documentation.untitledCategory') }}
        </h3>
        <p v-if="selectedTutorial.description" class="mt-1.5 max-w-3xl text-sm leading-6 text-gray-600 dark:text-dark-300">
          {{ selectedTutorial.description }}
        </p>
      </header>

      <ol v-if="selectedTutorial.steps.length" class="divide-y divide-gray-200 dark:divide-dark-700">
        <li
          v-for="(step, index) in selectedTutorial.steps"
          :key="`${selectedTutorial.id}-${index}`"
          class="flex gap-4 py-6 sm:gap-5"
        >
          <span
            class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gray-950 text-sm font-semibold text-white dark:bg-white dark:text-gray-950"
            aria-hidden="true"
          >
            {{ index + 1 }}
          </span>
          <div class="min-w-0 flex-1">
            <h4 class="text-base font-semibold text-gray-950 dark:text-white">
              {{ step.title || t('admin.documentation.untitledStep') }}
            </h4>
            <p v-if="step.description" class="mt-1 max-w-3xl whitespace-pre-line text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ step.description }}
            </p>

            <div
              v-if="step.note && step.note_placement !== 'after-image'"
              class="mt-4 rounded-lg border px-3.5 py-3 text-sm leading-6"
              :class="noteClasses(step.note.tone)"
            >
              {{ step.note.text }}
            </div>

            <figure v-if="step.image && safeUrl(step.image.src)" class="mt-4 max-w-3xl">
              <img
                :src="safeUrl(step.image.src)"
                :alt="step.image.alt"
                class="h-auto max-w-full rounded border border-gray-200 dark:border-dark-700"
              />
              <figcaption v-if="step.image.caption" class="mt-1.5 text-xs text-gray-500 dark:text-dark-400">
                {{ step.image.caption }}
              </figcaption>
            </figure>

            <div
              v-if="step.note && step.note_placement === 'after-image'"
              class="mt-4 rounded-lg border px-3.5 py-3 text-sm leading-6"
              :class="noteClasses(step.note.tone)"
            >
              {{ step.note.text }}
            </div>

            <div v-if="step.code" class="mt-4 max-w-3xl overflow-hidden rounded-xl border border-gray-800 bg-[#101713]">
              <div class="border-b border-white/10 px-4 py-2 text-xs font-medium text-[#9fb7ae]">
                {{ step.code.label || t('admin.documentation.code') }}
              </div>
              <pre class="overflow-x-auto p-4 text-sm leading-6 text-[#dff4ed]"><code>{{ step.code.value }}</code></pre>
            </div>

            <a
              v-if="step.link && safeUrl(step.link.href)"
              :href="safeUrl(step.link.href)"
              target="_blank"
              rel="noopener noreferrer"
              class="mt-4 inline-flex items-center gap-1.5 rounded-md py-1 text-sm font-medium text-primary-700 hover:text-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-primary-300 dark:hover:text-primary-200"
            >
              {{ step.link.label || step.link.href }}
              <Icon name="externalLink" size="xs" aria-hidden="true" />
            </a>
          </div>
        </li>
      </ol>

      <div v-else class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">
        {{ t('admin.documentation.noStepsPreview') }}
      </div>
    </div>

    <div v-else class="py-16 text-center">
      <Icon name="book" size="xl" class="mx-auto text-gray-400" aria-hidden="true" />
      <p class="mt-3 text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.documentation.emptyPreview') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DocumentationContent, DocumentationNoteTone } from '@/api/admin/documentation'
import Icon from '@/components/icons/Icon.vue'
import DocumentationCategoryIcon from '@/components/admin/documentation/DocumentationCategoryIcon.vue'
import { sanitizeUrl } from '@/utils/url'

const props = defineProps<{ content: DocumentationContent }>()
const { t } = useI18n()
const selectedId = ref('')

const selectedTutorial = computed(
  () => props.content.tutorials.find((tutorial) => tutorial.id === selectedId.value) ?? props.content.tutorials[0] ?? null,
)

watch(
  () => props.content.tutorials.map((tutorial) => tutorial.id).join('|'),
  () => {
    if (!props.content.tutorials.some((tutorial) => tutorial.id === selectedId.value)) {
      selectedId.value = props.content.tutorials[0]?.id ?? ''
    }
  },
  { immediate: true },
)

function safeUrl(value: string): string {
  return sanitizeUrl(value, { allowRelative: true })
}

function noteClasses(tone: DocumentationNoteTone): string {
  return tone === 'warning'
    ? 'border-amber-200 bg-amber-50 text-amber-950 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-100'
    : 'info-surface'
}

async function moveTab(index: number, offset: -1 | 1): Promise<void> {
  const length = props.content.tutorials.length
  if (!length) return
  const nextIndex = (index + offset + length) % length
  const nextTutorial = props.content.tutorials[nextIndex]
  selectedId.value = nextTutorial.id
  await nextTick()
  document.getElementById(`documentation-preview-tab-${nextTutorial.id}`)?.focus()
}
</script>
