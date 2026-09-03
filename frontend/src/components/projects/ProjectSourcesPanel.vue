<template>
  <section
    id="project-sources-panel"
    class="project-sources-panel"
    data-ui-component="project-sources-panel"
    role="tabpanel"
    aria-labelledby="project-sources-tab"
    :aria-label="t('projects.sourcesLabel')"
  >
    <div v-if="files.length === 0" class="project-sources-empty" role="status">
      <div class="project-sources-empty__marks" aria-hidden="true">
        <span><Icon name="document" size="sm" /></span>
        <span><Icon name="cloud" size="sm" /></span>
        <span><Icon name="paperclip" size="sm" /></span>
      </div>
      <h2>{{ sourcesEmptyTitle }}</h2>
      <p>{{ sourcesEmptyDescription }}</p>
      <label
        class="project-source-add"
        :class="{ 'project-source-add--disabled': uploading }"
        :aria-disabled="uploading ? 'true' : undefined"
      >
        <span>{{ uploading ? uploadingLabel : addSourceLabel }}</span>
        <input
          type="file"
          multiple
          :disabled="uploading"
          :aria-label="addSourceLabel"
          @change="emit('add-files', $event)"
        />
      </label>
    </div>

    <template v-else>
      <div class="project-sources-panel__toolbar">
        <h2 class="sr-only">{{ t('projects.sourcesLabel') }}</h2>
        <label
          class="project-source-add project-source-add--quiet"
          :class="{ 'project-source-add--disabled': uploading }"
          :aria-disabled="uploading ? 'true' : undefined"
        >
          <Icon name="plus" size="xs" aria-hidden="true" />
          <span>{{ uploading ? uploadingLabel : addSourceLabel }}</span>
          <input
            type="file"
            multiple
            :disabled="uploading"
            :aria-label="addSourceLabel"
            @change="emit('add-files', $event)"
          />
        </label>
      </div>

      <ul class="project-sources-panel__list" role="list">
        <li
          v-for="file in files"
          :key="file.id"
          class="project-source-row"
          :class="{ 'project-source-row--menu-open': openMenuId === file.id }"
        >
          <span class="project-source-row__icon" aria-hidden="true">
            <Icon name="document" size="sm" />
          </span>
          <span class="project-source-row__copy">
            <strong :title="file.name">{{ file.name }}</strong>
            <span v-if="formatFileMeta(file)">{{ formatFileMeta(file) }}</span>
          </span>

          <div class="project-source-row__menu-wrap">
            <button
              type="button"
              class="project-source-row__more"
              :aria-label="fileMenuLabel(file.name)"
              aria-haspopup="menu"
              :aria-expanded="openMenuId === file.id"
              :aria-controls="fileMenuId(file.id)"
              @click="toggleFileMenu(file.id, $event)"
              @keydown.down.prevent="openFileMenu(file.id, $event)"
              @keydown.up.prevent="openFileMenu(file.id, $event)"
            >
              <Icon name="more" size="sm" aria-hidden="true" />
            </button>

            <Transition name="project-source-menu">
              <div
                v-if="openMenuId === file.id"
                :id="fileMenuId(file.id)"
                ref="menuRef"
                class="project-source-menu"
                role="menu"
                @keydown.esc.prevent.stop="closeFileMenu(true)"
              >
                <button type="button" role="menuitem" @click="removeFile(file.id)">
                  <Icon name="x" size="sm" aria-hidden="true" />
                  <span>{{ t('projects.removeFile') }}</span>
                </button>
              </div>
            </Transition>
          </div>
        </li>
      </ul>
    </template>
  </section>
</template>

<script setup lang="ts">
import {
  computed,
  getCurrentInstance,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
} from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { ProjectFile } from '@/types/projects'

withDefaults(defineProps<{
  files: ProjectFile[]
  uploading?: boolean
}>(), {
  uploading: false,
})

const emit = defineEmits<{
  'add-files': [event: Event]
  remove: [id: string]
}>()

const { locale, t, te } = useI18n()
const instanceId = `project-sources-${getCurrentInstance()?.uid ?? 'panel'}`
const openMenuId = ref<string | null>(null)
const menuRef = ref<HTMLElement | null>(null)
let activeMenuTrigger: HTMLButtonElement | null = null

const usesChinese = computed(() => locale.value.toLowerCase().startsWith('zh'))
const sourcesEmptyTitle = computed(() => te('projects.sourcesEmptyTitle')
  ? t('projects.sourcesEmptyTitle')
  : usesChinese.value ? '为 ChatGPT 提供更多背景信息' : 'Give ChatGPT more context')
const sourcesEmptyDescription = computed(() => te('projects.sourcesEmptyDescription')
  ? t('projects.sourcesEmptyDescription')
  : usesChinese.value
    ? '上传数据源、链接云端硬盘或连接应用，为项目提供更深入的背景信息。'
    : 'Upload sources, link cloud storage, or connect apps to give this project more context.')
const addSourceLabel = computed(() => te('projects.addSource')
  ? t('projects.addSource')
  : usesChinese.value ? '添加来源' : 'Add source')
const uploadingLabel = computed(() => te('projects.addingSource')
  ? t('projects.addingSource')
  : usesChinese.value ? '正在添加…' : 'Adding…')

const dateFormatter = computed(() => new Intl.DateTimeFormat(locale.value, {
  year: 'numeric',
  month: 'short',
  day: 'numeric',
}))

const numberFormatter = computed(() => new Intl.NumberFormat(locale.value, {
  maximumFractionDigits: 1,
}))

function safeId(value: string): string {
  return value.replace(/[^a-zA-Z0-9_-]/g, '-')
}

function fileMenuId(id: string): string {
  return `${instanceId}-menu-${safeId(id)}`
}

function fileMenuLabel(name: string): string {
  return te('projects.fileActions')
    ? t('projects.fileActions', { name })
    : usesChinese.value ? `${name} 的文件操作` : `File actions for ${name}`
}

function formatFileSize(bytes?: number): string {
  if (bytes == null || !Number.isFinite(bytes) || bytes < 0) return ''
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let value = bytes / 1024
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  return `${numberFormatter.value.format(value)} ${units[unitIndex]}`
}

function formatFileDate(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : dateFormatter.value.format(date)
}

function formatMimeType(value?: string): string {
  if (!value) return ''
  const subtype = value.split('/')[1]?.split(';')[0]
  return subtype ? subtype.toUpperCase() : value
}

function formatFileMeta(file: ProjectFile): string {
  const metadata = [formatFileSize(file.size), formatFileDate(file.createdAt)].filter(Boolean)
  if (metadata.length === 0) metadata.push(formatMimeType(file.mimeType))
  return metadata.filter(Boolean).join(' · ')
}

async function openFileMenu(id: string, event: Event): Promise<void> {
  activeMenuTrigger = event.currentTarget as HTMLButtonElement
  openMenuId.value = id
  await nextTick()
  menuRef.value?.querySelector<HTMLButtonElement>('[role="menuitem"]')?.focus()
}

function toggleFileMenu(id: string, event: MouseEvent): void {
  if (openMenuId.value === id) {
    closeFileMenu()
    return
  }
  void openFileMenu(id, event)
}

function closeFileMenu(restoreFocus = false): void {
  if (!openMenuId.value) return
  openMenuId.value = null
  if (restoreFocus) {
    void nextTick(() => activeMenuTrigger?.focus({ preventScroll: true }))
  }
}

function removeFile(id: string): void {
  emit('remove', id)
  closeFileMenu(true)
}

function handleDocumentPointerDown(event: PointerEvent): void {
  if (!openMenuId.value) return
  const target = event.target as Node | null
  if (
    target
    && !menuRef.value?.contains(target)
    && !activeMenuTrigger?.contains(target)
  ) {
    closeFileMenu()
  }
}

function handleDocumentKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Escape' || !openMenuId.value) return
  event.preventDefault()
  closeFileMenu(true)
}

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown, true)
  document.addEventListener('keydown', handleDocumentKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown, true)
  document.removeEventListener('keydown', handleDocumentKeydown)
})
</script>

<style scoped>
.project-sources-panel {
  --project-source-text: var(--workspace-text);
  --project-source-secondary: var(--workspace-text-secondary);
  --project-source-muted: var(--workspace-text-muted);
  --project-source-hover: var(--workspace-hover);
  --project-source-surface: var(--workspace-popover-surface);
  --project-source-border: var(--workspace-popover-border);
  --project-source-border-strong: var(--workspace-border-strong);
  --project-source-focus: var(--workspace-popover-focus);
  --project-source-action: #0d0d0d;
  --project-source-action-text: #fff;
  width: min(100%, 768px);
  color: var(--project-source-text);
}

:global(html.light .project-sources-panel) {
  --project-source-hover: #f3f3f3;
  --project-source-surface: #fff;
  --project-source-border: rgb(0 0 0 / 0.12);
  --project-source-border-strong: rgb(0 0 0 / 0.2);
  --project-source-action: #0d0d0d;
  --project-source-action-text: #fff;
}

:global(html.dark .project-sources-panel) {
  --project-source-hover: #212121;
  --project-source-surface: #2f2f2f;
  --project-source-border: rgb(255 255 255 / 0.12);
  --project-source-border-strong: rgb(255 255 255 / 0.2);
  --project-source-action: #f4f4f4;
  --project-source-action-text: #171717;
}

.project-sources-empty {
  display: grid;
  width: 100%;
  min-height: 320px;
  align-content: center;
  justify-items: center;
  padding: 40px 28px;
  border: 1px dashed var(--project-source-border-strong);
  border-radius: 14px;
  text-align: center;
}

.project-sources-empty__marks {
  display: flex;
  min-height: 42px;
  align-items: center;
  justify-content: center;
  margin-bottom: 18px;
  padding-left: 12px;
}

.project-sources-empty__marks span {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  margin-left: -12px;
  border: 1px solid var(--project-source-border);
  border-radius: 10px;
  color: var(--project-source-secondary);
  background: var(--project-source-surface);
  box-shadow: 0 3px 10px rgb(0 0 0 / 0.1);
}

.project-sources-empty__marks span:first-child {
  transform: rotate(-6deg);
}

.project-sources-empty__marks span:last-child {
  transform: rotate(6deg);
}

.project-sources-empty h2 {
  margin: 0;
  color: var(--project-source-text);
  font-size: 16px;
  font-weight: 600;
  line-height: 24px;
  letter-spacing: -0.01em;
}

.project-sources-empty p {
  max-width: 520px;
  margin: 7px 0 22px;
  color: var(--project-source-secondary);
  font-size: 13px;
  line-height: 20px;
}

.project-source-add {
  position: relative;
  display: inline-flex;
  min-height: 32px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 0;
  border-radius: 999px;
  padding: 0 13px;
  color: var(--project-source-action-text);
  background: var(--project-source-action);
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;
  cursor: pointer;
}

.project-source-add--quiet {
  border: 1px solid var(--project-source-border-strong);
  color: var(--project-source-text);
  background: transparent;
  font-weight: 500;
}

.project-source-add:hover:not(.project-source-add--disabled) {
  opacity: 0.9;
}

.project-source-add--quiet:hover:not(.project-source-add--disabled) {
  background: var(--project-source-hover);
  opacity: 1;
}

.project-source-add--disabled {
  cursor: wait;
  opacity: 0.55;
}

.project-source-add input {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  margin: -1px;
  padding: 0;
  border: 0;
  clip: rect(0 0 0 0);
  clip-path: inset(50%);
  white-space: nowrap;
}

.project-source-add:focus-within {
  outline: 2px solid var(--project-source-focus);
  outline-offset: 3px;
}

.project-sources-panel__toolbar {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: flex-end;
  margin-bottom: 4px;
}

.project-sources-panel__list {
  width: 100%;
  margin: 0;
  padding: 0;
  list-style: none;
}

.project-source-row {
  position: relative;
  display: flex;
  min-height: 60px;
  align-items: center;
  gap: 12px;
  padding: 7px 0 7px 10px;
  background: transparent;
}

.project-source-row:hover,
.project-source-row:focus-within,
.project-source-row--menu-open {
  background: var(--project-source-hover);
}

.project-source-row__icon {
  display: grid;
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  place-items: center;
  border-radius: 8px;
  color: var(--project-source-secondary);
  background: color-mix(in srgb, var(--project-source-text) 7%, transparent);
}

.project-source-row__copy {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 2px;
}

.project-source-row__copy strong {
  overflow: hidden;
  color: var(--project-source-text);
  font-size: 14px;
  font-weight: 500;
  line-height: 21px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-source-row__copy > span {
  overflow: hidden;
  color: var(--project-source-secondary);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-source-row__menu-wrap {
  position: relative;
  display: grid;
  flex: 0 0 48px;
  place-items: center;
}

.project-source-row__more {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 0;
  border-radius: 9px;
  padding: 0;
  color: var(--project-source-secondary);
  background: transparent;
  cursor: pointer;
}

.project-source-row__more:hover,
.project-source-row__more[aria-expanded='true'] {
  color: var(--project-source-text);
  background: color-mix(in srgb, var(--project-source-text) 8%, transparent);
}

.project-source-row__more:focus-visible,
.project-source-menu button:focus-visible {
  outline: 2px solid var(--project-source-focus);
  outline-offset: 2px;
}

.project-source-menu {
  position: absolute;
  z-index: 35;
  top: 40px;
  right: 6px;
  width: max-content;
  min-width: 170px;
  padding: 5px;
  border: 1px solid var(--project-source-border);
  border-radius: 12px;
  color: var(--project-source-text);
  background: var(--project-source-surface);
  box-shadow: 0 6px 18px rgb(0 0 0 / 0.12);
}

.project-source-menu button {
  display: flex;
  width: 100%;
  min-height: 36px;
  align-items: center;
  gap: 9px;
  border: 0;
  border-radius: 8px;
  padding: 0 10px;
  color: inherit;
  background: transparent;
  font: inherit;
  font-size: 13px;
  white-space: nowrap;
  cursor: pointer;
}

.project-source-menu button:hover {
  background: var(--project-source-hover);
}

.project-source-menu-enter-active,
.project-source-menu-leave-active {
  transition: opacity 100ms ease, transform 120ms ease;
  transform-origin: top right;
}

.project-source-menu-enter-from,
.project-source-menu-leave-to {
  opacity: 0;
  transform: translateY(-3px) scale(0.985);
}

@media (max-width: 700px) {
  .project-sources-panel {
    width: 100%;
  }

  .project-sources-empty {
    min-height: clamp(276px, 42dvh, 320px);
    padding: 36px 20px;
    border-radius: 12px;
  }

  .project-sources-empty p {
    max-width: 310px;
  }

  .project-source-row {
    min-height: 58px;
    padding-left: 0;
  }

  .project-source-row__menu-wrap {
    flex-basis: 42px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .project-source-menu-enter-active,
  .project-source-menu-leave-active {
    transition: none;
  }
}
</style>
