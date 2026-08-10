<template>
  <ul
    v-if="attachments.length"
    ref="rootRef"
    class="chat-message-attachments"
    :aria-label="t('chat.attachments.messageAttachments')"
  >
    <li
      v-for="attachment in attachments"
      :key="attachment.id"
      class="chat-message-attachments__item"
      :class="{
        'chat-message-attachments__item--image': attachment.kind === 'image',
        'chat-message-attachments__item--expired': isExpired(attachment),
      }"
    >
      <template v-if="attachment.kind === 'image'">
        <img
          v-if="imageUrls[attachment.id] && !isExpired(attachment)"
          :src="imageUrls[attachment.id]"
          :alt="attachment.name"
        />
        <div v-else class="chat-message-attachments__placeholder">
          <Icon name="photo" size="md" />
          <span v-if="isExpired(attachment)">{{ t('chat.attachments.expired') }}</span>
          <span v-else-if="imageErrors.has(attachment.id)">
            {{ t('chat.attachments.loadFailed') }}
          </span>
          <span v-else>{{ t('chat.attachments.loading') }}</span>
        </div>
        <span class="chat-message-attachments__image-name" :title="attachment.name">
          {{ attachment.name }}
        </span>
      </template>

      <template v-else>
        <div class="chat-message-attachments__document-icon" aria-hidden="true">
          <Icon name="document" size="md" />
        </div>
        <div class="chat-message-attachments__document-copy">
          <strong :title="attachment.name">{{ attachment.name }}</strong>
          <span>
            {{ documentLabel(attachment) }} · {{ formatAttachmentBytes(attachment.size) }}
          </span>
          <span v-if="isExpired(attachment)" class="chat-message-attachments__expired">
            {{ t('chat.attachments.expired') }}
          </span>
        </div>
      </template>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getChatAttachmentContent, isAbortError } from '@/api/chat'
import Icon from '@/components/icons/Icon.vue'
import { formatAttachmentBytes } from './chatAttachmentUi'
import type { ChatAttachment } from '@/types/chat'

const props = defineProps<{
  attachments: ChatAttachment[]
}>()

const { t } = useI18n()
const imageUrls = ref<Record<string, string>>({})
const imageErrors = ref(new Set<string>())
const expiryTick = ref(Date.now())
const rootRef = ref<HTMLElement | null>(null)
const canLoadImages = ref(typeof IntersectionObserver !== 'function')
const loadingIds = new Set<string>()
const controllers = new Map<string, AbortController>()
let intersectionObserver: IntersectionObserver | null = null
let expiryTimer: ReturnType<typeof setTimeout> | undefined
const MAX_EXPIRY_TIMER_DELAY_MS = 24 * 60 * 60 * 1000

function isExpired(attachment: ChatAttachment): boolean {
  const now = expiryTick.value
  if (attachment.status === 'expired') return true
  const expiration = Date.parse(attachment.expiresAt)
  return Number.isFinite(expiration) && expiration <= now
}

function documentLabel(attachment: ChatAttachment): string {
  const type = attachment.mimeType.includes('pdf') ? 'PDF' : 'DOCX'
  return attachment.pageCount
    ? t('chat.attachments.documentPages', { type, count: attachment.pageCount })
    : type
}

function revokeUrl(id: string): void {
  const url = imageUrls.value[id]
  if (!url) return
  if (typeof URL.revokeObjectURL === 'function') URL.revokeObjectURL(url)
  const next = { ...imageUrls.value }
  delete next[id]
  imageUrls.value = next
}

async function loadImage(attachment: ChatAttachment): Promise<void> {
  if (loadingIds.has(attachment.id) || imageUrls.value[attachment.id]) return
  loadingIds.add(attachment.id)
  const controller = new AbortController()
  controllers.set(attachment.id, controller)
  imageErrors.value = new Set([...imageErrors.value].filter((id) => id !== attachment.id))
  try {
    const blob = await getChatAttachmentContent(attachment.id, controller.signal)
    if (controller.signal.aborted || typeof URL.createObjectURL !== 'function') return
    const current = props.attachments.find(({ id }) => id === attachment.id)
    if (!current || current.kind !== 'image' || isExpired(current)) return
    const url = URL.createObjectURL(blob)
    revokeUrl(attachment.id)
    imageUrls.value = { ...imageUrls.value, [attachment.id]: url }
  } catch (error) {
    if (!controller.signal.aborted && !isAbortError(error)) {
      imageErrors.value = new Set(imageErrors.value).add(attachment.id)
    }
  } finally {
    loadingIds.delete(attachment.id)
    if (controllers.get(attachment.id) === controller) controllers.delete(attachment.id)
  }
}

function syncImages(): void {
  const desired = new Map(canLoadImages.value
    ? props.attachments
        .filter((attachment) => attachment.kind === 'image' && !isExpired(attachment))
        .map((attachment) => [attachment.id, attachment] as const)
    : [])

  for (const id of Object.keys(imageUrls.value)) {
    if (!desired.has(id)) revokeUrl(id)
  }
  for (const [id, controller] of controllers) {
    if (!desired.has(id)) {
      controller.abort()
      controllers.delete(id)
      loadingIds.delete(id)
    }
  }
  imageErrors.value = new Set([...imageErrors.value].filter((id) => desired.has(id)))
  for (const attachment of desired.values()) void loadImage(attachment)
}

function scheduleExpiryRefresh(): void {
  if (expiryTimer) clearTimeout(expiryTimer)
  expiryTimer = undefined
  const now = Date.now()
  expiryTick.value = now
  const nearestExpiration = props.attachments.reduce((nearest, attachment) => {
    if (attachment.status === 'expired') return nearest
    const expiration = Date.parse(attachment.expiresAt)
    if (!Number.isFinite(expiration) || expiration <= now) return nearest
    return nearest === null || expiration < nearest ? expiration : nearest
  }, null as number | null)
  if (nearestExpiration === null) return

  const delay = Math.min(
    MAX_EXPIRY_TIMER_DELAY_MS,
    Math.max(20, nearestExpiration - now + 20),
  )
  expiryTimer = setTimeout(() => {
    expiryTick.value = Date.now()
    syncImages()
    scheduleExpiryRefresh()
  }, delay)
}

function syncAttachmentState(): void {
  expiryTick.value = Date.now()
  syncImages()
  scheduleExpiryRefresh()
}

watch(
  () => props.attachments.map((attachment) => (
    `${attachment.id}:${attachment.status}:${attachment.expiresAt}`
  )).join('|'),
  syncAttachmentState,
  { immediate: true },
)

onMounted(() => {
  if (canLoadImages.value) {
    syncImages()
    return
  }
  const root = rootRef.value
  if (!root) return
  intersectionObserver = new IntersectionObserver((entries) => {
    if (!entries.some((entry) => entry.isIntersecting || entry.intersectionRatio > 0)) return
    canLoadImages.value = true
    intersectionObserver?.unobserve(root)
    intersectionObserver?.disconnect()
    intersectionObserver = null
    syncImages()
  }, { rootMargin: '320px 0px' })
  intersectionObserver.observe(root)
})

onBeforeUnmount(() => {
  intersectionObserver?.disconnect()
  intersectionObserver = null
  if (expiryTimer) clearTimeout(expiryTimer)
  for (const controller of controllers.values()) controller.abort()
  controllers.clear()
  for (const id of Object.keys(imageUrls.value)) revokeUrl(id)
})
</script>

<style scoped>
.chat-message-attachments {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0 0 10px;
  padding: 0;
  list-style: none;
}

.chat-message-attachments__item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: min(260px, 100%);
  max-width: 360px;
  overflow: hidden;
  border: 1px solid var(--workspace-border);
  border-radius: 12px;
  padding: 10px 12px;
  color: var(--workspace-text);
  background: var(--workspace-surface);
}

.chat-message-attachments__item--image {
  position: relative;
  display: block;
  width: min(320px, 100%);
  min-width: 180px;
  padding: 0;
  background: var(--workspace-surface-subtle);
}

.chat-message-attachments__item--image img,
.chat-message-attachments__placeholder {
  display: flex;
  width: 100%;
  aspect-ratio: 4 / 3;
  align-items: center;
  justify-content: center;
  object-fit: cover;
}

.chat-message-attachments__placeholder {
  flex-direction: column;
  gap: 7px;
  color: var(--workspace-text-secondary);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.chat-message-attachments__image-name {
  position: absolute;
  right: 7px;
  bottom: 7px;
  left: 7px;
  overflow: hidden;
  border-radius: 8px;
  padding: 5px 7px;
  color: var(--workspace-text);
  background: color-mix(in srgb, var(--workspace-surface) 92%, transparent);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-message-attachments__item--expired {
  border-color: color-mix(in srgb, var(--lx-clay-warning) 42%, var(--lx-clay-border));
  background: var(--lx-clay-warning-soft);
}

.chat-message-attachments__document-icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  flex: none;
  border-radius: 9px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
}

.chat-message-attachments__document-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.chat-message-attachments__document-copy strong {
  overflow: hidden;
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-message-attachments__document-copy span {
  color: var(--workspace-text-secondary);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.chat-message-attachments__document-copy .chat-message-attachments__expired {
  color: var(--lx-clay-warning);
}

@media (max-width: 640px) {
  .chat-message-attachments__item,
  .chat-message-attachments__item--image {
    width: 100%;
    max-width: 100%;
  }
}
</style>
