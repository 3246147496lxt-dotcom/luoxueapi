<template>
  <div
    ref="rootRef"
    class="library-file-visual"
    :class="[
      `library-file-visual--${file.type}`,
      { 'library-file-visual--compact': compact },
    ]"
    aria-hidden="true"
  >
    <img v-if="thumbnailUrl && !thumbnailFailed" :src="thumbnailUrl" alt="" @error="onThumbnailError" />
    <template v-else>
      <Icon :name="file.type === 'image' ? 'photo' : 'document'" :size="compact ? 'md' : 'xl'" />
      <span v-if="file.type !== 'image'" class="library-file-visual__extension">
        {{ libraryFileExtension(file) }}
      </span>
    </template>
    <span v-if="loading" class="library-file-visual__loading"></span>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { isLibraryAbortError } from '@/api/library'
import type { LibraryFile } from '@/types/library'
import { libraryFileExtension } from './libraryFileUi'
import { libraryThumbnailLoader } from './libraryThumbnailLoader'

const props = withDefaults(defineProps<{
  file: LibraryFile
  compact?: boolean
}>(), {
  compact: false,
})

const thumbnailUrl = ref('')
const thumbnailFailed = ref(false)
const loading = ref(false)
const rootRef = ref<HTMLElement | null>(null)
const canLoadThumbnail = ref(typeof IntersectionObserver !== 'function')
let controller: AbortController | null = null
let intersectionObserver: IntersectionObserver | null = null

function revokeThumbnail(): void {
  if (!thumbnailUrl.value) return
  if (typeof URL.revokeObjectURL === 'function') URL.revokeObjectURL(thumbnailUrl.value)
  thumbnailUrl.value = ''
}

function onThumbnailError(): void {
  thumbnailFailed.value = true
  revokeThumbnail()
}

function resetThumbnail(): void {
  controller?.abort()
  controller = null
  revokeThumbnail()
  thumbnailFailed.value = false
  loading.value = false
}

async function syncThumbnail(): Promise<void> {
  resetThumbnail()
  if (props.file.type !== 'image' || !canLoadThumbnail.value) return
  const nextController = new AbortController()
  controller = nextController
  loading.value = true
  try {
    const blob = await libraryThumbnailLoader.load({
      id: props.file.id,
      updatedAt: props.file.updatedAt,
    }, nextController.signal)
    if (nextController.signal.aborted || controller !== nextController) return
    if (typeof URL.createObjectURL !== 'function') {
      thumbnailFailed.value = true
      return
    }
    thumbnailUrl.value = URL.createObjectURL(blob)
  } catch (error) {
    if (!nextController.signal.aborted && !isLibraryAbortError(error)) {
      thumbnailFailed.value = true
    }
  } finally {
    if (controller === nextController) {
      controller = null
      loading.value = false
    }
  }
}

watch(
  () => [props.file.id, props.file.updatedAt, props.file.type, canLoadThumbnail.value] as const,
  () => void syncThumbnail(),
  { immediate: true },
)

onMounted(() => {
  if (canLoadThumbnail.value) return
  const root = rootRef.value
  if (!root || typeof IntersectionObserver !== 'function') {
    canLoadThumbnail.value = true
    return
  }
  intersectionObserver = new IntersectionObserver((entries) => {
    if (!entries.some((entry) => entry.isIntersecting || entry.intersectionRatio > 0)) return
    canLoadThumbnail.value = true
    intersectionObserver?.disconnect()
    intersectionObserver = null
  }, { rootMargin: '240px 0px' })
  intersectionObserver.observe(root)
})

onBeforeUnmount(() => {
  intersectionObserver?.disconnect()
  intersectionObserver = null
  resetThumbnail()
})
</script>

<style scoped>
.library-file-visual {
  position: relative;
  display: grid;
  width: 100%;
  height: 100%;
  place-items: center;
  overflow: hidden;
  border-radius: 10px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface);
}

.library-file-visual--compact {
  box-sizing: border-box;
  border: 1px solid color-mix(in srgb, var(--workspace-text) 10%, transparent);
  border-radius: 8px;
  background: var(--workspace-surface-subtle);
}

.library-file-visual img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.library-file-visual__extension {
  position: absolute;
  right: 7px;
  bottom: 6px;
  border-radius: 4px;
  padding: 2px 4px;
  color: var(--workspace-text-muted);
  background: var(--workspace-canvas);
  font-size: 9px;
  font-weight: 600;
  line-height: 1;
  letter-spacing: 0.03em;
}

.library-file-visual--compact .library-file-visual__extension {
  right: 3px;
  bottom: 3px;
  padding: 1px 2px;
  font-size: 7px;
}

.library-file-visual__loading {
  position: absolute;
  width: 18px;
  height: 18px;
  border: 2px solid color-mix(in srgb, var(--workspace-text-muted) 40%, transparent);
  border-top-color: var(--workspace-text-secondary);
  border-radius: 50%;
  animation: library-visual-spin 800ms linear infinite;
}

@keyframes library-visual-spin {
  to { transform: rotate(1turn); }
}

@media (prefers-reduced-motion: reduce) {
  .library-file-visual__loading { animation: none; }
}
</style>
