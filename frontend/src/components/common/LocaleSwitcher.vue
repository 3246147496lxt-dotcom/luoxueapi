<template>
  <div class="relative" ref="dropdownRef">
    <button
      :id="triggerId"
      ref="triggerRef"
      type="button"
      @click="toggleDropdown"
      @keydown="handleTriggerKeydown"
      :disabled="switching"
      class="relative flex items-center justify-center rounded-full text-[#007bff] transition-colors duration-200 after:absolute after:-inset-1.5 after:content-[''] hover:bg-[rgba(46,50,56,0.05)] active:bg-gray-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60 dark:text-[#5aa2ff] dark:hover:bg-white/[0.08] dark:active:bg-dark-700 dark:focus-visible:ring-offset-dark-900"
      :class="[
        compact
          ? 'h-8 w-8'
          : 'h-11 min-h-11 w-11 min-w-11',
        { 'bg-gray-100 dark:bg-dark-800': isOpen }
      ]"
      :title="currentLocale?.name"
      :aria-label="currentLocale?.name"
      aria-haspopup="menu"
      :aria-expanded="isOpen"
      :aria-controls="menuId"
    >
      <LanguageIcon class="locale-switcher-icon h-5 w-5" />
    </button>

    <transition name="dropdown">
      <div
        v-if="isOpen"
        :id="menuId"
        ref="menuRef"
        role="menu"
        aria-orientation="vertical"
        :aria-labelledby="triggerId"
        class="absolute right-0 z-50 mt-1 w-36 overflow-hidden rounded-lg border border-gray-200 bg-white p-1 shadow-lg dark:border-dark-700 dark:bg-dark-800"
        @keydown="handleMenuKeydown"
      >
        <button
          v-for="(locale, index) in availableLocales"
          :key="locale.code"
          ref="optionRefs"
          type="button"
          role="menuitemradio"
          :aria-checked="locale.code === currentLocaleCode"
          tabindex="-1"
          :disabled="switching"
          @click="selectLocale(locale.code)"
          @focus="focusedIndex = index"
          class="flex min-h-11 w-full items-center gap-2 rounded-md px-3 text-left text-sm text-gray-700 transition-colors hover:bg-gray-100 focus-visible:bg-gray-100 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-60 dark:text-gray-200 dark:hover:bg-dark-700 dark:focus-visible:bg-dark-700"
          :class="{
            'bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-400':
              locale.code === currentLocaleCode
          }"
        >
          <span class="text-base" aria-hidden="true">{{ locale.flag }}</span>
          <span>{{ locale.name }}</span>
          <Icon
            v-if="locale.code === currentLocaleCode"
            name="check"
            size="sm"
            class="ml-auto text-primary-500"
            aria-hidden="true"
          />
        </button>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import LanguageIcon from '@/components/icons/LanguageIcon.vue'
import { setLocale, availableLocales } from '@/i18n'

withDefaults(defineProps<{
  compact?: boolean
}>(), {
  compact: false,
})

const { locale } = useI18n()

const instanceId = Math.random().toString(36).slice(2, 9)
const triggerId = `locale-switcher-trigger-${instanceId}`
const menuId = `locale-switcher-menu-${instanceId}`
const isOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const optionRefs = ref<HTMLButtonElement[]>([])
const focusedIndex = ref(-1)
const switching = ref(false)

const currentLocaleCode = computed(() => locale.value)
const currentLocale = computed(() => availableLocales.find((l) => l.code === locale.value))

function toggleDropdown() {
  if (isOpen.value) {
    void closeDropdown()
    return
  }

  void openDropdown('current')
}

type InitialFocus = 'current' | 'first' | 'last'

async function openDropdown(initialFocus: InitialFocus) {
  if (switching.value) return

  isOpen.value = true
  await nextTick()

  if (initialFocus === 'first') {
    focusOption(0)
    return
  }

  if (initialFocus === 'last') {
    focusOption(availableLocales.length - 1)
    return
  }

  const currentIndex = availableLocales.findIndex((item) => item.code === currentLocaleCode.value)
  focusOption(currentIndex >= 0 ? currentIndex : 0)
}

async function closeDropdown(restoreFocus = false) {
  isOpen.value = false
  focusedIndex.value = -1
  await nextTick()

  if (restoreFocus) {
    triggerRef.value?.focus()
  }
}

function focusOption(index: number) {
  const optionCount = optionRefs.value.length
  if (optionCount === 0) return

  const normalizedIndex = (index + optionCount) % optionCount
  focusedIndex.value = normalizedIndex
  optionRefs.value[normalizedIndex]?.focus()
}

function handleTriggerKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    void openDropdown('first')
    return
  }

  if (event.key === 'ArrowUp') {
    event.preventDefault()
    void openDropdown('last')
    return
  }

  if (event.key === 'Escape' && isOpen.value) {
    event.preventDefault()
    event.stopPropagation()
    void closeDropdown(true)
  }
}

function handleMenuKeydown(event: KeyboardEvent) {
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      focusOption(focusedIndex.value + 1)
      break
    case 'ArrowUp':
      event.preventDefault()
      focusOption(focusedIndex.value - 1)
      break
    case 'Home':
      event.preventDefault()
      focusOption(0)
      break
    case 'End':
      event.preventDefault()
      focusOption(optionRefs.value.length - 1)
      break
    case 'Escape':
      event.preventDefault()
      event.stopPropagation()
      void closeDropdown(true)
      break
    case 'Tab':
      void closeDropdown()
      break
  }
}

async function selectLocale(code: string) {
  if (switching.value) return

  if (code === currentLocaleCode.value) {
    await closeDropdown(true)
    return
  }

  switching.value = true
  try {
    await setLocale(code)
  } finally {
    switching.value = false
  }

  await closeDropdown(true)
}

function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    void closeDropdown()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.15s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}

@media (prefers-reduced-motion: reduce) {
  .dropdown-enter-active,
  .dropdown-leave-active {
    transition: none !important;
  }
}
</style>
