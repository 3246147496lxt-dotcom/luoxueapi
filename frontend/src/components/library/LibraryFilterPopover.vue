<template>
  <Teleport to="body">
    <Transition name="library-popover">
      <div v-if="show" class="library-filter-layer">
        <button
          type="button"
          class="library-filter-layer__backdrop"
          :aria-label="t('library.actions.close')"
          tabindex="-1"
          @click="emit('close')"
        ></button>
        <form
          ref="panelRef"
          class="library-filter-popover"
          role="dialog"
          :aria-label="t('library.filters.title')"
          :style="positionStyle"
          @submit.prevent="apply"
          @keydown.esc.prevent.stop="emit('close')"
        >
          <div class="library-filter-popover__header">
            <strong>{{ t('library.filters.title') }}</strong>
            <button type="button" @click="reset">{{ t('library.filters.reset') }}</button>
          </div>

          <fieldset>
            <legend>{{ t('library.filters.source') }}</legend>
            <label v-for="option in sourceOptions" :key="option.value">
              <input v-model="draftSource" type="radio" name="library-source" :value="option.value" />
              <span>{{ t(option.label) }}</span>
              <Icon v-if="draftSource === option.value" name="check" size="sm" aria-hidden="true" />
            </label>
          </fieldset>

          <fieldset>
            <legend>{{ t('library.filters.type') }}</legend>
            <label v-for="option in typeOptions" :key="option.value">
              <input v-model="draftType" type="radio" name="library-type" :value="option.value" />
              <span>{{ t(option.label) }}</span>
              <Icon v-if="draftType === option.value" name="check" size="sm" aria-hidden="true" />
            </label>
          </fieldset>

          <fieldset>
            <legend>{{ t('library.filters.sort') }}</legend>
            <label v-for="option in sortOptions" :key="option.value">
              <input v-model="draftSort" type="radio" name="library-sort" :value="option.value" />
              <span>{{ t(option.label) }}</span>
              <Icon v-if="draftSort === option.value" name="check" size="sm" aria-hidden="true" />
            </label>
          </fieldset>

          <button class="library-filter-popover__apply" type="submit">
            {{ t('library.filters.apply') }}
          </button>
        </form>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch, type CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { LibraryFileType, LibrarySort, LibrarySource } from '@/types/library'

const props = defineProps<{
  show: boolean
  anchor?: HTMLElement | null
  source: LibrarySource
  type: LibraryFileType
  sort: LibrarySort
}>()

const emit = defineEmits<{
  close: []
  apply: [value: { source: LibrarySource; type: LibraryFileType; sort: LibrarySort }]
}>()

const { t } = useI18n()
const panelRef = ref<HTMLElement | null>(null)
const draftSource = ref<LibrarySource>(props.source)
const draftType = ref<LibraryFileType>(props.type)
const draftSort = ref<LibrarySort>(props.sort)
const anchorRect = ref<DOMRect | null>(null)

const sourceOptions = [
  { value: 'all', label: 'library.filters.sources.all' },
  { value: 'uploaded', label: 'library.filters.sources.uploaded' },
  { value: 'generated', label: 'library.filters.sources.generated' },
] as const
const typeOptions = [
  { value: 'all', label: 'library.filters.types.all' },
  { value: 'image', label: 'library.filters.types.image' },
  { value: 'pdf', label: 'library.filters.types.pdf' },
  { value: 'document', label: 'library.filters.types.document' },
  { value: 'spreadsheet', label: 'library.filters.types.spreadsheet' },
  { value: 'presentation', label: 'library.filters.types.presentation' },
  { value: 'other', label: 'library.filters.types.other' },
] as const
const sortOptions = [
  { value: 'updated_desc', label: 'library.filters.sorts.updatedDesc' },
  { value: 'updated_asc', label: 'library.filters.sorts.updatedAsc' },
  { value: 'name_asc', label: 'library.filters.sorts.nameAsc' },
  { value: 'name_desc', label: 'library.filters.sorts.nameDesc' },
  { value: 'size_asc', label: 'library.filters.sorts.sizeAsc' },
  { value: 'size_desc', label: 'library.filters.sorts.sizeDesc' },
] as const

const positionStyle = computed<CSSProperties>(() => {
  const rect = anchorRect.value
  const width = Math.min(326, window.innerWidth - 24)
  if (!rect) return { top: '72px', right: '12px', width: `${width}px` }
  const left = Math.max(12, Math.min(rect.right - width, window.innerWidth - width - 12))
  const below = rect.bottom + 8
  const top = Math.min(below, Math.max(12, window.innerHeight - 260))
  return {
    top: `${top}px`,
    left: `${left}px`,
    width: `${width}px`,
    maxHeight: `${Math.max(220, window.innerHeight - top - 12)}px`,
  }
})

watch(() => props.show, async (visible) => {
  if (!visible) return
  draftSource.value = props.source
  draftType.value = props.type
  draftSort.value = props.sort
  anchorRect.value = props.anchor?.getBoundingClientRect() ?? null
  await nextTick()
  panelRef.value?.querySelector<HTMLInputElement>('input:checked')?.focus({ preventScroll: true })
})

function reset(): void {
  draftSource.value = 'all'
  draftType.value = 'all'
  draftSort.value = 'updated_desc'
}

function apply(): void {
  emit('apply', {
    source: draftSource.value,
    type: draftType.value,
    sort: draftSort.value,
  })
}
</script>

<style scoped>
.library-filter-layer {
  position: fixed;
  z-index: 80;
  inset: 0;
}

.library-filter-layer__backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background: transparent;
}

.library-filter-popover {
  position: fixed;
  z-index: 1;
  overflow: auto;
  overscroll-behavior: contain;
  border: 1px solid var(--workspace-popover-border);
  border-radius: 16px;
  padding: 10px;
  color: var(--workspace-popover-text);
  background: var(--workspace-popover-surface);
  box-shadow: var(--workspace-popover-shadow);
}

.library-filter-popover__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 38px;
  padding: 0 8px;
}

.library-filter-popover__header strong {
  font-size: 15px;
  font-weight: 600;
}

.library-filter-popover__header button {
  border: 0;
  padding: 5px 6px;
  color: var(--workspace-popover-text-secondary);
  background: transparent;
  font-size: 13px;
  cursor: pointer;
}

.library-filter-popover fieldset {
  margin: 4px 0 0;
  border: 0;
  border-top: 1px solid var(--workspace-popover-divider);
  padding: 12px 0 2px;
}

.library-filter-popover legend {
  padding: 0 8px 7px;
  color: var(--workspace-text-muted);
  font-size: 12px;
  font-weight: 600;
}

.library-filter-popover label {
  display: grid;
  min-height: 36px;
  grid-template-columns: minmax(0, 1fr) 18px;
  align-items: center;
  border-radius: 9px;
  padding: 0 8px;
  font-size: 14px;
  cursor: pointer;
}

.library-filter-popover label:hover {
  background: var(--workspace-popover-hover);
}

.library-filter-popover input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.library-filter-popover label:has(input:focus-visible) {
  outline: 2px solid var(--workspace-popover-focus);
  outline-offset: -2px;
}

.library-filter-popover__apply {
  width: 100%;
  min-height: 38px;
  margin-top: 8px;
  border: 0;
  border-radius: 999px;
  color: var(--workspace-canvas);
  background: var(--workspace-text);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.library-popover-enter-active,
.library-popover-leave-active {
  transition: opacity 120ms ease;
}

.library-popover-enter-from,
.library-popover-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .library-popover-enter-active,
  .library-popover-leave-active { transition: none; }
}
</style>
