<template>
  <div
    v-if="attachments.length"
    ref="rootRef"
    class="chat-message-attachments"
    role="group"
    :aria-label="t('chat.attachments.messageAttachments')"
  >
    <ul
      v-if="imageAttachments.length"
      class="chat-message-attachments__images image-attachments"
      :class="{
        'chat-message-attachments__images--multiple': imageAttachments.length > 1,
      }"
      role="list"
    >
      <li
        v-for="attachment in imageAttachments"
        :key="attachment.id"
        class="chat-message-attachments__image-item image-attachment"
        :class="{
          'chat-message-attachments__image-item--expired': isExpired(attachment),
        }"
      >
        <button
          v-if="imageUrls[attachment.id] && !isExpired(attachment)"
          type="button"
          class="chat-message-attachments__image-trigger"
          :aria-label="t('chat.attachments.viewImage', { name: attachment.name })"
          aria-haspopup="dialog"
          data-test="chat-message-image-trigger"
          @click="openPreview(attachment.id, $event)"
        >
          <img
            :src="imageUrls[attachment.id]"
            alt=""
          />
        </button>
        <div
          v-else
          class="chat-message-attachments__placeholder"
        >
          <Icon name="photo" size="md" aria-hidden="true" />
          <span v-if="isExpired(attachment)">{{ t('chat.attachments.expired') }}</span>
          <span v-else-if="imageErrors.has(attachment.id)">
            {{ t('chat.attachments.loadFailed') }}
          </span>
          <span v-else>{{ t('chat.attachments.loading') }}</span>
        </div>
      </li>
    </ul>

    <ul
      v-if="documentAttachments.length"
      class="chat-message-attachments__documents document-attachments"
      role="list"
    >
      <li
        v-for="attachment in documentAttachments"
        :key="attachment.id"
        class="chat-message-attachments__item"
        :class="{
          'chat-message-attachments__item--expired': isExpired(attachment),
        }"
      >
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
      </li>
    </ul>

    <ChatImagePreviewDialog
      :show="Boolean(previewedImage)"
      :src="previewedImage?.src"
      :name="previewedImage?.attachment.name"
      @close="previewedAttachmentId = null"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getChatAttachmentContent, isAbortError } from '@/api/chat'
import Icon from '@/components/icons/Icon.vue'
import ChatImagePreviewDialog from './ChatImagePreviewDialog.vue'
import { formatAttachmentBytes } from './chatAttachmentUi'
import type { ChatAttachment } from '@/types/chat'

const props = defineProps<{
  attachments: ChatAttachment[]
}>()

const { t } = useI18n()
const imageUrls = ref<Record<string, string>>({})
const imageErrors = ref(new Set<string>())
const expiryTick = ref(Date.now())
const previewedAttachmentId = ref<string | null>(null)
const rootRef = ref<HTMLElement | null>(null)
const canLoadImages = ref(typeof IntersectionObserver !== 'function')
const loadingIds = new Set<string>()
const controllers = new Map<string, AbortController>()
let intersectionObserver: IntersectionObserver | null = null
let expiryTimer: ReturnType<typeof setTimeout> | undefined
const MAX_EXPIRY_TIMER_DELAY_MS = 24 * 60 * 60 * 1000

function isImageAttachment(attachment: ChatAttachment): boolean {
  return attachment.kind === 'image'
    || attachment.mimeType.trim().toLowerCase().startsWith('image/')
}

const imageAttachments = computed(() => props.attachments.filter(isImageAttachment))
const documentAttachments = computed(() => props.attachments.filter((attachment) => (
  !isImageAttachment(attachment)
)))
const previewedImage = computed(() => {
  const attachment = imageAttachments.value.find(({ id }) => id === previewedAttachmentId.value)
  if (!attachment || isExpired(attachment)) return null
  const src = imageUrls.value[attachment.id]
  return src ? { attachment, src } : null
})

watch(previewedImage, (image) => {
  if (!image) previewedAttachmentId.value = null
})

function isExpired(attachment: ChatAttachment): boolean {
  const now = expiryTick.value
  if (attachment.status === 'expired') return true
  const expiration = Date.parse(attachment.expiresAt)
  return Number.isFinite(expiration) && expiration <= now
}

function documentLabel(attachment: ChatAttachment): string {
  const mimeType = attachment.mimeType.toLowerCase()
  const extension = attachment.name.split('.').pop()?.trim().toLowerCase() ?? ''
  const type = mimeType.includes('pdf') || extension === 'pdf'
    ? 'PDF'
    : mimeType.includes('spreadsheet') || mimeType.includes('excel') || ['xls', 'xlsx'].includes(extension)
      ? 'XLSX'
      : mimeType.includes('presentation') || mimeType.includes('powerpoint') || ['ppt', 'pptx'].includes(extension)
        ? 'PPTX'
        : mimeType.includes('word') || ['doc', 'docx'].includes(extension)
          ? 'DOCX'
          : 'FILE'
  return attachment.pageCount
    ? t('chat.attachments.documentPages', { type, count: attachment.pageCount })
    : type
}

function revokeUrl(id: string): void {
  const url = imageUrls.value[id]
  if (!url) return
  if (previewedAttachmentId.value === id) previewedAttachmentId.value = null
  const next = { ...imageUrls.value }
  delete next[id]
  imageUrls.value = next
  if (typeof URL.revokeObjectURL === 'function') URL.revokeObjectURL(url)
}

function openPreview(id: string, event: MouseEvent): void {
  if (event.currentTarget instanceof HTMLButtonElement) {
    event.currentTarget.focus({ preventScroll: true })
  }
  previewedAttachmentId.value = id
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
    if (!current || !isImageAttachment(current) || isExpired(current)) return
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
    ? imageAttachments.value
        .filter((attachment) => !isExpired(attachment))
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
    `${attachment.id}:${attachment.kind}:${attachment.mimeType}:${attachment.status}:${attachment.expiresAt}`
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
  width: fit-content;
  max-width: 100%;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
  margin: 0 0 0 auto;
}

.chat-message-attachments--with-content {
  margin-bottom: 10px;
}

.chat-message-attachments__images,
.chat-message-attachments__documents {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  margin: 0 0 0 auto;
  padding: 0;
  list-style: none;
}

.chat-message-attachments__images {
  width: fit-content;
  max-width: min(760px, 72vw);
}

.chat-message-attachments__images--multiple {
  width: min(760px, 72vw);
}

.chat-message-attachments__image-item {
  display: block;
  width: fit-content;
  max-width: min(760px, 72vw);
  flex: 0 1 auto;
  margin-left: auto;
}

.chat-message-attachments__images--multiple .chat-message-attachments__image-item {
  max-width: min(360px, calc(50% - 4px));
}

.chat-message-attachments__image-trigger {
  display: block;
  width: fit-content;
  max-width: 100%;
  height: auto;
  margin-left: auto;
  overflow: hidden;
  border: 0;
  border-radius: 28px;
  padding: 0;
  background: transparent;
  box-shadow: none;
  cursor: zoom-in;
}

.chat-message-attachments__image-trigger:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 42%, transparent);
  outline-offset: 2px;
}

.chat-message-attachments__image-trigger img {
  display: block;
  width: auto;
  height: auto;
  max-width: 100%;
  max-height: 560px;
  object-fit: contain;
}

.chat-message-attachments__placeholder {
  display: flex;
  width: fit-content;
  max-width: 100%;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border-radius: 20px;
  padding: 20px 24px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.chat-message-attachments__image-item--expired .chat-message-attachments__placeholder {
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
}

.chat-message-attachments__documents {
  width: fit-content;
  max-width: 100%;
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
  .chat-message-attachments,
  .chat-message-attachments__images,
  .chat-message-attachments__images--multiple,
  .chat-message-attachments__documents,
  .chat-message-attachments__image-item {
    max-width: 100%;
  }

  .chat-message-attachments__images--multiple {
    width: 100%;
  }

  .chat-message-attachments__images--multiple .chat-message-attachments__image-item {
    max-width: calc(50% - 4px);
  }

  .chat-message-attachments__item {
    width: 100%;
    max-width: 100%;
  }
}
</style>
