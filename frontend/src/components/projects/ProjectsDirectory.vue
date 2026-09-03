<template>
  <section
    class="projects-directory"
    :aria-busy="loading ? 'true' : undefined"
    :aria-labelledby="titleId"
  >
    <header class="projects-directory__topbar">
      <h1 :id="titleId">{{ t('projects.title') }}</h1>

      <label class="projects-directory__search">
        <span class="projects-directory__sr-only">{{ t('projects.searchPlaceholder') }}</span>
        <Icon name="search" size="sm" aria-hidden="true" />
        <input
          type="search"
          :value="search"
          autocomplete="off"
          :placeholder="t('projects.searchPlaceholder')"
          :aria-label="t('projects.searchPlaceholder')"
          @input="updateSearch"
        >
      </label>

      <button
        type="button"
        class="projects-directory__create"
        @click="emit('create')"
      >
        <Icon name="plus" size="sm" aria-hidden="true" />
        <span>{{ t('projects.new') }}</span>
      </button>
    </header>

    <div
      class="projects-directory__filters"
      role="group"
      :aria-label="t('projects.filtersLabel')"
    >
      <button
        v-for="option in filterOptions"
        :key="option.value"
        type="button"
        :class="{ 'projects-directory__filter--active': filter === option.value }"
        :aria-pressed="filter === option.value"
        @click="selectFilter(option.value)"
      >
        {{ option.label }}
      </button>
    </div>

    <div
      class="projects-directory__table"
      :class="{ 'projects-directory__table--empty': !loading && visibleProjects.length === 0 && emptyState.kind === 'initial' }"
      role="table"
      :aria-label="t('projects.listLabel')"
    >
      <div class="projects-directory__table-head" role="row">
        <span role="columnheader">{{ t('projects.nameColumn') }}</span>
        <span role="columnheader">{{ t('projects.modifiedColumn') }}</span>
        <span aria-hidden="true"></span>
      </div>
      <p class="projects-directory__mobile-heading">{{ t('projects.nameColumn') }}</p>

      <div
        v-if="loading && projects.length === 0"
        class="projects-directory__skeletons"
        aria-live="polite"
      >
        <span class="projects-directory__sr-only">{{ t('projects.loading') }}</span>
        <div
          v-for="index in 3"
          :key="index"
          class="projects-directory__skeleton-row"
          aria-hidden="true"
        >
          <span class="projects-directory__skeleton-icon"></span>
          <span class="projects-directory__skeleton-copy">
            <span></span>
            <span></span>
          </span>
          <span class="projects-directory__skeleton-date"></span>
        </div>
      </div>

      <div v-else-if="visibleProjects.length" class="projects-directory__rows" role="rowgroup">
        <div
          v-for="(project, index) in visibleProjects"
          :key="project.id"
          class="projects-directory__row"
          :class="{ 'projects-directory__row--menu-open': openMenuId === project.id }"
          role="row"
        >
          <button
            type="button"
            class="projects-directory__row-open"
            :aria-label="t('projects.openProject', { name: project.name })"
            @click="openProject(project.id)"
          >
            <span class="projects-directory__identity" role="cell">
              <span
                class="projects-directory__emoji"
                aria-hidden="true"
              >
                <Icon
                  v-if="usesDefaultProjectIcon(project)"
                  name="chatSidebarProjects"
                  size="sm"
                />
                <template v-else>{{ project.icon }}</template>
              </span>
              <span class="projects-directory__name-wrap">
                <strong :title="project.name">{{ project.name }}</strong>
                <small>{{ formatUpdatedAt(project.updatedAt) }}</small>
              </span>
            </span>
            <time
              class="projects-directory__date"
              role="cell"
              :datetime="dateTime(project.updatedAt)"
            >
              {{ formatUpdatedAt(project.updatedAt) }}
            </time>
          </button>

          <div class="projects-directory__actions" role="cell">
            <button
              :ref="(element) => setMenuTrigger(project.id, element)"
              type="button"
              class="projects-directory__more"
              :aria-label="t('projects.moreActions', { name: project.name })"
              aria-haspopup="menu"
              :aria-expanded="openMenuId === project.id"
              :aria-controls="openMenuId === project.id ? menuId(index) : undefined"
              @click.stop="toggleMenu(project.id)"
            >
              <Icon name="more" size="sm" aria-hidden="true" />
            </button>

            <div
              v-if="openMenuId === project.id"
              :id="menuId(index)"
              :ref="(element) => setMenuElement(project.id, element)"
              class="projects-directory__menu"
              role="menu"
              :aria-label="t('projects.projectActions', { name: project.name })"
              @click.stop
            >
              <button type="button" role="menuitem" @click="editProject(project)">
                {{ t('projects.edit') }}
              </button>
              <button
                type="button"
                role="menuitem"
                class="projects-directory__menu-danger"
                @click="deleteProject(project.id)"
              >
                {{ t('projects.delete') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <div
        v-else
        class="projects-directory__empty"
        :class="{ 'projects-directory__empty--initial': emptyState.kind === 'initial' }"
        aria-live="polite"
      >
        <template v-if="emptyState.kind === 'initial'">
          <span class="projects-directory__empty-icon" aria-hidden="true">
            <Icon name="folderOutline" size="xl" />
          </span>
          <strong>{{ emptyState.title }}</strong>
        </template>
        <template v-else>
          <strong>{{ emptyState.title }}</strong>
          <p>{{ emptyState.description }}</p>
          <button
            v-if="emptyState.showCreate"
            type="button"
            @click="emit('create')"
          >
            {{ t('projects.newProject') }}
          </button>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  useId,
  watch,
} from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Project } from '@/types/projects'

type ProjectFilter = 'all' | 'mine' | 'shared'

const props = withDefaults(defineProps<{
  projects: Project[]
  loading?: boolean
  search: string
  filter: ProjectFilter
}>(), {
  loading: false,
})

const emit = defineEmits<{
  'update:search': [value: string]
  'update:filter': [value: ProjectFilter]
  create: []
  open: [id: string]
  edit: [project: Project]
  delete: [id: string]
}>()

const { t, locale } = useI18n()

const filterOptions = computed<ReadonlyArray<{ value: ProjectFilter, label: string }>>(() => [
  { value: 'all', label: t('projects.filterAll') },
  { value: 'mine', label: t('projects.filterMine') },
  { value: 'shared', label: t('projects.filterShared') },
])

const componentId = useId().replace(/:/g, '')
const titleId = `projects-directory-title-${componentId}`
const openMenuId = ref<string | null>(null)
const menuTriggers = new Map<string, HTMLButtonElement>()
const menuElements = new Map<string, HTMLElement>()

const scopedProjects = computed(() => (
  // Project currently has no ownership or sharing metadata. Treat every
  // supplied project as user-owned and never invent rows for the shared tab.
  props.filter === 'shared' ? [] : props.projects
))

const visibleProjects = computed(() => {
  const query = props.search.trim().toLocaleLowerCase()
  if (!query) return scopedProjects.value
  return scopedProjects.value.filter((project) => (
    project.name.toLocaleLowerCase().includes(query)
  ))
})

const emptyState = computed(() => {
  if (props.filter === 'shared') {
    return {
      title: t('projects.sharedEmptyTitle'),
      description: t('projects.sharedEmptyDescription'),
      showCreate: false,
      kind: 'shared' as const,
    }
  }
  if (props.search.trim()) {
    return {
      title: t('projects.searchEmptyTitle'),
      description: t('projects.searchEmptyDescription'),
      showCreate: false,
      kind: 'search' as const,
    }
  }
  return {
    title: t('projects.noProjects'),
    description: '',
    showCreate: false,
    kind: 'initial' as const,
  }
})

function updateSearch(event: Event): void {
  closeMenu()
  const target = event.target
  emit('update:search', target instanceof HTMLInputElement ? target.value : '')
}

function selectFilter(value: ProjectFilter): void {
  closeMenu()
  emit('update:filter', value)
}

function dateValue(value: number): Date | null {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function sameCalendarDay(left: Date, right: Date): boolean {
  return left.getFullYear() === right.getFullYear()
    && left.getMonth() === right.getMonth()
    && left.getDate() === right.getDate()
}

function formatUpdatedAt(value: number): string {
  const date = dateValue(value)
  if (!date) return t('projects.recentlyUpdated')

  const now = new Date()
  if (sameCalendarDay(date, now)) return t('projects.today')

  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (sameCalendarDay(date, yesterday)) return t('projects.yesterday')

  return new Intl.DateTimeFormat(locale.value, {
    year: date.getFullYear() === now.getFullYear() ? undefined : 'numeric',
    month: 'numeric',
    day: 'numeric',
  }).format(date)
}

function dateTime(value: number): string | undefined {
  return dateValue(value)?.toISOString()
}

function usesDefaultProjectIcon(project: Project): boolean {
  const icon = project.icon?.trim()
  return !icon || icon === 'folder' || icon === '📁'
}

function menuId(index: number): string {
  return `projects-directory-menu-${componentId}-${index}`
}

function setMenuTrigger(id: string, element: unknown): void {
  if (element instanceof HTMLButtonElement) menuTriggers.set(id, element)
  else menuTriggers.delete(id)
}

function setMenuElement(id: string, element: unknown): void {
  if (element instanceof HTMLElement) menuElements.set(id, element)
  else menuElements.delete(id)
}

function toggleMenu(id: string): void {
  openMenuId.value = openMenuId.value === id ? null : id
}

function closeMenu(restoreFocus = false): void {
  const id = openMenuId.value
  if (!id) return
  openMenuId.value = null
  menuElements.delete(id)
  if (restoreFocus) void nextTick(() => menuTriggers.get(id)?.focus())
}

function openProject(id: string): void {
  closeMenu()
  emit('open', id)
}

function editProject(project: Project): void {
  closeMenu()
  emit('edit', project)
}

function deleteProject(id: string): void {
  closeMenu()
  emit('delete', id)
}

function handleOutsidePointer(event: PointerEvent): void {
  const id = openMenuId.value
  const target = event.target
  if (!id || !(target instanceof Node)) return
  if (menuElements.get(id)?.contains(target) || menuTriggers.get(id)?.contains(target)) return
  closeMenu()
}

function handleEscape(event: KeyboardEvent): void {
  if (event.key !== 'Escape' || !openMenuId.value) return
  event.preventDefault()
  event.stopPropagation()
  closeMenu(true)
}

watch(
  () => props.projects.map((project) => project.id),
  (ids) => {
    if (openMenuId.value && !ids.includes(openMenuId.value)) closeMenu()
  },
)

watch(() => [props.filter, props.search], () => closeMenu())

onMounted(() => {
  document.addEventListener('pointerdown', handleOutsidePointer, true)
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleOutsidePointer, true)
  document.removeEventListener('keydown', handleEscape)
  menuTriggers.clear()
  menuElements.clear()
})
</script>

<style scoped>
.projects-directory {
  --projects-directory-text: var(--workspace-text);
  --projects-directory-text-secondary: var(--workspace-text-secondary);
  --projects-directory-text-muted: var(--workspace-text-muted);
  --projects-directory-control: var(--workspace-surface-subtle);
  --projects-directory-hover: var(--workspace-hover);
  --projects-directory-selected: var(--workspace-selected);
  --projects-directory-border: var(--workspace-border);
  --projects-directory-border-strong: var(--workspace-border-strong);
  --projects-directory-menu: var(--workspace-popup-surface);
  --projects-directory-on-primary: var(--workspace-canvas);
  width: min(100%, 768px);
  margin: 0 auto;
  color: var(--projects-directory-text);
  font-family: var(--workspace-font-ui);
}

.projects-directory__topbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 240px auto;
  align-items: center;
  gap: 12px;
}

.projects-directory h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 500;
  line-height: 34px;
  letter-spacing: -0.02em;
}

.projects-directory__search {
  display: flex;
  width: 240px;
  height: 36px;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--projects-directory-border);
  border-radius: 999px;
  padding: 0 12px;
  color: var(--projects-directory-text-muted);
  background: var(--projects-directory-control);
}

.projects-directory__search:focus-within {
  border-color: var(--projects-directory-border-strong);
  outline: 2px solid color-mix(in srgb, var(--projects-directory-text) 12%, transparent);
  outline-offset: 1px;
}

.projects-directory__search svg {
  width: 16px;
  height: 16px;
  flex: 0 0 16px;
}

.projects-directory__search input {
  width: 100%;
  min-width: 0;
  border: 0;
  padding: 0;
  color: var(--projects-directory-text);
  background: transparent;
  font: inherit;
  font-size: 14px;
  line-height: 20px;
  outline: none;
}

.projects-directory__search input::placeholder {
  color: var(--projects-directory-text-muted);
  opacity: 1;
}

.projects-directory__search input::-webkit-search-cancel-button {
  opacity: 0.72;
}

.projects-directory__create {
  display: inline-flex;
  height: 36px;
  align-items: center;
  justify-content: center;
  gap: 5px;
  border: 0;
  border-radius: 999px;
  padding: 0 12px;
  color: var(--projects-directory-on-primary);
  background: var(--projects-directory-text);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  white-space: nowrap;
  cursor: pointer;
}

.projects-directory__create:hover {
  opacity: 0.88;
}

.projects-directory__create svg {
  display: none;
}

.projects-directory__filters {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 52px;
}

.projects-directory__filters button {
  height: 36px;
  border: 0;
  border-radius: 999px;
  padding: 0 16px;
  color: var(--projects-directory-text-secondary);
  background: transparent;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  white-space: nowrap;
  cursor: pointer;
}

.projects-directory__filters button:hover {
  color: var(--projects-directory-text);
  background: var(--projects-directory-hover);
}

.projects-directory__filters .projects-directory__filter--active {
  padding-inline: 16px;
  color: var(--projects-directory-text);
  background: var(--projects-directory-selected);
}

.projects-directory__table {
  margin-top: 20px;
}

.projects-directory__table--empty .projects-directory__table-head,
.projects-directory__table--empty .projects-directory__mobile-heading {
  display: none;
}

.projects-directory__table-head {
  display: grid;
  min-height: 36px;
  grid-template-columns: minmax(0, 1fr) 160px 44px;
  align-items: center;
  gap: 8px;
  color: var(--projects-directory-text-secondary);
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
}

.projects-directory__mobile-heading {
  display: none;
  margin: 0;
  color: var(--projects-directory-text-secondary);
  font-size: 14px;
  line-height: 20px;
}

.projects-directory__rows {
  position: relative;
}

.projects-directory__row {
  position: relative;
  display: grid;
  height: 60px;
  grid-template-columns: minmax(0, 1fr) 44px;
  align-items: center;
  gap: 8px;
  border-radius: 10px;
}

.projects-directory__row:hover,
.projects-directory__row:focus-within {
  background: var(--projects-directory-hover);
}

.projects-directory__row--menu-open {
  z-index: 3;
  background: var(--projects-directory-hover);
}

.projects-directory__row-open {
  display: grid;
  width: 100%;
  height: 60px;
  min-width: 0;
  grid-template-columns: minmax(0, 1fr) 160px;
  align-items: center;
  gap: 8px;
  border: 0;
  border-radius: 10px 0 0 10px;
  padding: 0;
  color: inherit;
  background: transparent;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.projects-directory__identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.projects-directory__emoji {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  border: 1px solid var(--projects-directory-border);
  border-radius: 8px;
  background: var(--projects-directory-control);
  font-size: 17px;
  line-height: 1;
}

.projects-directory__name-wrap {
  display: grid;
  min-width: 0;
}

.projects-directory__name-wrap strong {
  overflow: hidden;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.projects-directory__name-wrap small {
  display: none;
  color: var(--projects-directory-text-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 16px;
}

.projects-directory__date {
  color: var(--projects-directory-text-secondary);
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
}

.projects-directory__actions {
  position: relative;
  display: grid;
  width: 44px;
  height: 60px;
  place-items: center;
}

.projects-directory__more {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 0;
  border-radius: 9px;
  padding: 0;
  color: var(--projects-directory-text-secondary);
  background: transparent;
  opacity: 0;
  cursor: pointer;
  transition: opacity 120ms ease, color 120ms ease, background-color 120ms ease;
}

.projects-directory__row:hover .projects-directory__more,
.projects-directory__row:focus-within .projects-directory__more,
.projects-directory__row--menu-open .projects-directory__more {
  opacity: 1;
}

.projects-directory__more:hover,
.projects-directory__more[aria-expanded='true'] {
  color: var(--projects-directory-text);
  background: var(--projects-directory-selected);
}

.projects-directory__menu {
  position: absolute;
  z-index: 5;
  top: 48px;
  right: 0;
  display: grid;
  width: 128px;
  border: 1px solid var(--projects-directory-border);
  border-radius: 12px;
  padding: 6px;
  background: var(--projects-directory-menu);
  box-shadow: 0 10px 30px rgb(0 0 0 / 0.14);
}

.projects-directory__menu button {
  width: 100%;
  min-height: 36px;
  border: 0;
  border-radius: 8px;
  padding: 7px 10px;
  color: var(--projects-directory-text);
  background: transparent;
  font: inherit;
  font-size: 14px;
  line-height: 20px;
  text-align: left;
  cursor: pointer;
}

.projects-directory__menu button:hover,
.projects-directory__menu button:focus-visible {
  background: var(--projects-directory-hover);
}

.projects-directory__menu .projects-directory__menu-danger {
  color: #e02e2a;
}

.projects-directory__empty {
  display: grid;
  min-height: 240px;
  place-items: center;
  align-content: center;
  padding: 48px 16px;
  text-align: center;
}

.projects-directory__empty--initial {
  min-height: 224px;
  box-sizing: border-box;
  grid-auto-rows: max-content;
  gap: 16px;
  padding: 64px 24px;
}

.projects-directory__empty-icon {
  display: grid;
  width: 56px;
  height: 56px;
  place-items: center;
  border: 1px solid var(--projects-directory-border);
  border-radius: 16px;
  color: var(--projects-directory-text);
  background: #e8e8e8;
}

.projects-directory__empty-icon svg {
  width: 32px;
  height: 32px;
  stroke-width: 1.45;
}

.projects-directory__empty--initial strong {
  font-size: 16px;
  line-height: 24px;
}

.projects-directory__empty strong {
  font-size: 15px;
  font-weight: 500;
  line-height: 22px;
}

.projects-directory__empty p {
  max-width: 340px;
  margin: 6px 0 0;
  color: var(--projects-directory-text-secondary);
  font-size: 13px;
  line-height: 20px;
}

.projects-directory__empty button {
  height: 36px;
  margin-top: 18px;
  border: 1px solid var(--projects-directory-border-strong);
  border-radius: 999px;
  padding: 0 14px;
  color: var(--projects-directory-text);
  background: transparent;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.projects-directory__empty button:hover {
  background: var(--projects-directory-hover);
}

.projects-directory__skeletons {
  display: grid;
}

.projects-directory__skeleton-row {
  display: grid;
  height: 60px;
  grid-template-columns: 32px minmax(0, 1fr) 160px 44px;
  align-items: center;
  gap: 12px;
}

.projects-directory__skeleton-icon,
.projects-directory__skeleton-copy span,
.projects-directory__skeleton-date {
  display: block;
  border-radius: 8px;
  background: var(--projects-directory-control);
  animation: projects-directory-pulse 1.25s ease-in-out infinite alternate;
}

.projects-directory__skeleton-icon {
  width: 32px;
  height: 32px;
}

.projects-directory__skeleton-copy {
  display: grid;
  gap: 5px;
}

.projects-directory__skeleton-copy span:first-child {
  width: min(180px, 70%);
  height: 12px;
}

.projects-directory__skeleton-copy span:last-child {
  display: none;
  width: 56px;
  height: 9px;
}

.projects-directory__skeleton-date {
  width: 52px;
  height: 12px;
}

.projects-directory button:focus-visible {
  outline: 2px solid var(--projects-directory-text-secondary);
  outline-offset: 2px;
}

.projects-directory__sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  clip-path: inset(50%);
  white-space: nowrap;
}

@keyframes projects-directory-pulse {
  from { opacity: 0.55; }
  to { opacity: 1; }
}

:global(html.dark .projects-directory) {
  --projects-directory-text: var(--workspace-text);
  --projects-directory-text-secondary: var(--workspace-text-secondary);
  --projects-directory-text-muted: var(--workspace-text-muted);
  --projects-directory-control: var(--workspace-surface-subtle);
  --projects-directory-hover: var(--workspace-hover);
  --projects-directory-selected: var(--workspace-selected);
  --projects-directory-border: var(--workspace-border);
  --projects-directory-border-strong: var(--workspace-border-strong);
  --projects-directory-menu: var(--workspace-popup-surface);
  --projects-directory-on-primary: var(--workspace-canvas);
}

:global(html.dark .projects-directory__empty-icon) {
  color: #f2f2f2;
  background: #2f2f2f;
  border-color: rgb(255 255 255 / 0.12);
}

@media (max-width: 640px) {
  .projects-directory__topbar {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 0 12px;
  }

  .projects-directory__search {
    width: 100%;
    grid-column: 1 / -1;
    grid-row: 2;
    margin-top: 16px;
  }

  .projects-directory__create {
    grid-column: 2;
    grid-row: 1;
  }

  .projects-directory__filters {
    gap: 2px;
    margin-top: 4px;
  }

  .projects-directory__filters button {
    padding: 0 16px;
  }

  .projects-directory__table {
    margin-top: 52px;
  }

  .projects-directory__table-head {
    display: none;
  }

  .projects-directory__mobile-heading {
    display: block;
    min-height: 28px;
  }

  .projects-directory__empty--initial {
    min-height: 224px;
    padding-inline: 0;
  }

  .projects-directory__row {
    grid-template-columns: minmax(0, 1fr) 44px;
  }

  .projects-directory__row-open {
    grid-template-columns: minmax(0, 1fr);
  }

  .projects-directory__name-wrap small {
    display: block;
  }

  .projects-directory__date {
    display: none;
  }

  .projects-directory__menu {
    top: 48px;
  }

  .projects-directory__more {
    opacity: 1;
  }

  .projects-directory__skeleton-row {
    grid-template-columns: 32px minmax(0, 1fr) 44px;
  }

  .projects-directory__skeleton-copy span:last-child {
    display: block;
  }

  .projects-directory__skeleton-date {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .projects-directory__skeleton-icon,
  .projects-directory__skeleton-copy span,
  .projects-directory__skeleton-date {
    animation: none;
  }
}
</style>
