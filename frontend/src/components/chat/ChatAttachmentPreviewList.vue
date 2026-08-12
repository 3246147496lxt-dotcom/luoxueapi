<template>
  <section
    v-if="items.length"
    class="chat-attachment-preview"
    :class="{
      'chat-attachment-preview--single-image': isSingleImage,
      'chat-attachment-preview--multiple': items.length > 1,
    }"
    :aria-label="t('chat.attachments.selected')"
  >
    <ul class="chat-attachment-preview__list" role="list">
      <li
        v-for="item in items"
        :key="item.key"
        class="chat-attachment-preview__item"
        :class="[
          `chat-attachment-preview__item--${item.kind}`,
          `chat-attachment-preview__item--${item.state}`,
          {
            'chat-attachment-preview__item--error': itemIsInvalid(item),
          },
        ]"
        :data-kind="item.kind"
        :data-state="item.state"
        data-test="chat-attachment-item"
      >
        <div class="chat-attachment-preview__visual" aria-hidden="true">
          <img
            v-if="item.kind === 'image' && item.previewUrl"
            :src="item.previewUrl"
            alt=""
          />
          <Icon v-else :name="item.kind === 'image' ? 'photo' : 'document'" size="lg" />

          <span
            v-if="item.state === 'uploading' || item.state === 'processing'"
            class="chat-attachment-preview__scrim"
          >
            <span class="chat-attachment-preview__state-icon">
              <Icon name="refresh" size="sm" />
            </span>
            <span v-if="item.state === 'uploading'" class="chat-attachment-preview__progress-copy">
              {{ item.progress }}%
            </span>
          </span>
          <span
            v-else-if="itemIsInvalid(item)"
            class="chat-attachment-preview__scrim chat-attachment-preview__scrim--error"
          >
            <span class="chat-attachment-preview__state-icon">
              <Icon name="exclamationCircle" size="sm" />
            </span>
          </span>
        </div>

        <div v-if="item.kind === 'document'" class="chat-attachment-preview__document-copy">
          <strong :title="item.file.name">{{ item.file.name }}</strong>
          <span>{{ documentMeta(item) }}</span>
        </div>

        <span class="sr-only" aria-live="polite">
          {{ item.file.name }}: {{ statusText(item) }}
        </span>

        <progress
          v-if="item.state === 'uploading'"
          class="sr-only"
          max="100"
          :value="item.progress"
          :aria-label="t('chat.attachments.uploadProgress', { progress: item.progress })"
        ></progress>

        <button
          v-if="item.state === 'error'"
          type="button"
          class="chat-attachment-preview__retry"
          :aria-label="t('chat.attachments.retry', { name: item.file.name })"
          :title="t('chat.attachments.retry', { name: item.file.name })"
          data-test="chat-attachment-retry"
          @click="$emit('retry', item.key)"
        >
          <span class="chat-attachment-preview__action-icon">
            <Icon name="refresh" size="xs" aria-hidden="true" />
          </span>
        </button>

        <ChatControlTooltip
          :label="t('chat.attachments.removeFile')"
          :accessible="false"
          contents
        >
          <button
            type="button"
            class="chat-attachment-preview__remove"
            :aria-label="t('chat.attachments.remove', { name: item.file.name })"
            data-test="chat-attachment-remove"
            @click="$emit('remove', item.key)"
          >
            <span class="chat-attachment-preview__action-icon">
              <Icon name="x" size="xs" aria-hidden="true" />
            </span>
          </button>
        </ChatControlTooltip>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import ChatControlTooltip from './ChatControlTooltip.vue'
import {
  formatAttachmentBytes,
  type ChatAttachmentDraft,
} from './chatAttachmentUi'

const props = withDefaults(defineProps<{
  items: ChatAttachmentDraft[]
  supportsVision?: boolean
}>(), {
  supportsVision: false,
})

defineEmits<{
  cancel: [key: string]
  retry: [key: string]
  remove: [key: string]
}>()

const { t } = useI18n()

const isSingleImage = computed(() => (
  props.items.length === 1 && props.items[0]?.kind === 'image'
))

function itemIsInvalid(item: ChatAttachmentDraft): boolean {
  return item.state === 'error' || (item.kind === 'image' && !props.supportsVision)
}

function statusText(item: ChatAttachmentDraft): string {
  if (item.kind === 'image' && !props.supportsVision) {
    return t('chat.attachments.errors.visionUnsupported')
  }
  if (item.state === 'uploading') {
    return t('chat.attachments.uploading', { progress: item.progress })
  }
  if (item.state === 'processing') return t('chat.attachments.processing')
  if (item.state === 'ready') return t('chat.attachments.ready')
  return t(item.errorKey ?? 'chat.attachments.errors.uploadFailed', item.errorArgs ?? {})
}

function documentMeta(item: ChatAttachmentDraft): string {
  const status = item.state === 'ready' ? formatAttachmentBytes(item.file.size) : statusText(item)
  return isDocx(item) && item.state === 'ready'
    ? `${status} · ${t('chat.attachments.docxTextOnly')}`
    : status
}

function isDocx(item: ChatAttachmentDraft): boolean {
  return item.file.name.toLowerCase().endsWith('.docx')
}
</script>

<style scoped>
.chat-attachment-preview {
  width: 100%;
  min-width: 0;
  margin: 0;
}

.chat-attachment-preview__list {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 0;
  padding: 0 6px 0 0;
  overflow-x: auto;
  overflow-y: hidden;
  list-style: none;
  scrollbar-width: none;
  overscroll-behavior-inline: contain;
}

.chat-attachment-preview__list::-webkit-scrollbar {
  display: none;
}

.chat-attachment-preview__item {
  position: relative;
  flex: 0 0 auto;
  min-width: 0;
  color: var(--chat-composer-primary-fg, var(--lx-clay-text));
}

.chat-attachment-preview__item--image {
  width: 56px;
  height: 56px;
}

.chat-attachment-preview--single-image .chat-attachment-preview__item--image {
  width: 144px;
  height: 144px;
}

.chat-attachment-preview__item--document {
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
  width: min(240px, calc(100vw - 72px));
  height: 72px;
  border: 1px solid var(--workspace-border);
  border-radius: 14px;
  padding: 11px 36px 11px 11px;
  background: var(--workspace-surface-subtle);
}

.chat-attachment-preview__visual {
  position: relative;
  display: grid;
  width: 100%;
  height: 100%;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--workspace-border);
  border-radius: 14px;
  color: var(--chat-composer-muted-fg, var(--lx-clay-text-muted));
  background: var(--workspace-surface-subtle);
}

.chat-attachment-preview__item--document .chat-attachment-preview__visual {
  width: 48px;
  height: 48px;
  border: 0;
  border-radius: 10px;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.chat-attachment-preview__visual img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.chat-attachment-preview__scrim {
  position: absolute;
  inset: 0;
  display: grid;
  place-content: center;
  justify-items: center;
  gap: 3px;
  color: #fff;
  background: rgb(13 13 13 / 48%);
}

.chat-attachment-preview__scrim--error {
  background: rgb(127 29 29 / 50%);
}

.chat-attachment-preview__state-icon {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border-radius: 50%;
  background: rgb(0 0 0 / 58%);
}

.chat-attachment-preview__item--uploading .chat-attachment-preview__state-icon,
.chat-attachment-preview__item--processing .chat-attachment-preview__state-icon {
  animation: chat-attachment-spin 900ms linear infinite;
}

.chat-attachment-preview__progress-copy {
  font-size: 11px;
  font-weight: 600;
  line-height: 14px;
}

.chat-attachment-preview__document-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.chat-attachment-preview__document-copy strong,
.chat-attachment-preview__document-copy span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-attachment-preview__document-copy strong {
  font-size: 14px;
  font-weight: 500;
  line-height: 18px;
}

.chat-attachment-preview__document-copy span {
  color: var(--chat-composer-muted-fg, var(--lx-clay-text-muted));
  font-size: 12px;
  font-weight: 400;
  line-height: 16px;
}

.chat-attachment-preview__remove,
.chat-attachment-preview__retry {
  position: absolute;
  z-index: 2;
  display: grid;
  border: 0;
  padding: 0;
  place-items: center;
  color: #fff;
  background: transparent;
  cursor: pointer;
  opacity: 0;
  transition: opacity 120ms ease;
}

.chat-attachment-preview__remove {
  inset-block-start: 0;
  inset-inline-end: 0;
  width: 44px;
  height: 44px;
}

.chat-attachment-preview__retry {
  inset-block-end: 2px;
  inset-inline-start: 2px;
  width: 34px;
  height: 34px;
}

.chat-attachment-preview__remove::before,
.chat-attachment-preview__retry::before {
  position: absolute;
  border: 2px solid var(--chat-composer-surface, #fff);
  border-radius: 50%;
  background: #212121;
  box-shadow: 0 2px 8px rgb(0 0 0 / 16%);
  content: '';
  transition: background-color 120ms ease, transform 120ms ease;
}

.chat-attachment-preview__remove::before {
  inset-block-start: 5px;
  inset-inline-end: 5px;
  width: 22px;
  height: 22px;
}

.chat-attachment-preview__retry::before {
  inset-block-end: 3px;
  inset-inline-start: 3px;
  width: 24px;
  height: 24px;
}

.chat-attachment-preview__action-icon {
  position: absolute;
  z-index: 1;
  display: grid;
  place-items: center;
}

.chat-attachment-preview__remove .chat-attachment-preview__action-icon {
  inset-block-start: 10px;
  inset-inline-end: 10px;
  width: 12px;
  height: 12px;
}

.chat-attachment-preview__retry .chat-attachment-preview__action-icon {
  inset-block-end: 9px;
  inset-inline-start: 9px;
  width: 12px;
  height: 12px;
}

.chat-attachment-preview__item:hover .chat-attachment-preview__remove,
.chat-attachment-preview__item:hover .chat-attachment-preview__retry,
.chat-attachment-preview__item:focus-within .chat-attachment-preview__remove,
.chat-attachment-preview__item:focus-within .chat-attachment-preview__retry {
  opacity: 1;
}

.chat-attachment-preview__remove:hover::before,
.chat-attachment-preview__retry:hover::before {
  background: #0d0d0d;
  transform: scale(1.04);
}

.chat-attachment-preview__remove:focus-visible,
.chat-attachment-preview__retry:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: -5px;
}

.chat-attachment-preview__item--error .chat-attachment-preview__visual,
.chat-attachment-preview__item--error.chat-attachment-preview__item--document {
  border-color: color-mix(in srgb, var(--lx-clay-danger) 52%, var(--lx-clay-border));
}

@media (hover: none), (pointer: coarse) {
  .chat-attachment-preview__list {
    scroll-snap-type: x proximity;
  }

  .chat-attachment-preview__item {
    scroll-snap-align: start;
  }

  .chat-attachment-preview__remove,
  .chat-attachment-preview__retry {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chat-attachment-preview__item--uploading .chat-attachment-preview__state-icon,
  .chat-attachment-preview__item--processing .chat-attachment-preview__state-icon {
    animation: none;
  }

  .chat-attachment-preview__remove,
  .chat-attachment-preview__retry,
  .chat-attachment-preview__remove::before,
  .chat-attachment-preview__retry::before {
    transition: none;
  }
}

@keyframes chat-attachment-spin {
  to {
    transform: rotate(1turn);
  }
}
</style>
