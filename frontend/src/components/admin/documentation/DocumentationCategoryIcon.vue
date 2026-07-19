<template>
  <span
    class="inline-flex flex-shrink-0 items-center justify-center"
    :class="size === 'md' ? 'h-6 w-6' : 'h-4 w-4'"
    aria-hidden="true"
  >
    <span
      v-if="safeIconSvg"
      class="block h-full w-full [&>svg]:block [&>svg]:h-full [&>svg]:w-full"
      v-html="safeIconSvg"
    ></span>
    <Icon v-else :name="iconName" :size="size" />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { DocumentationIconKey } from '@/api/admin/documentation'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeDocumentationIconSvg } from '@/utils/documentationSvg'

const props = withDefaults(defineProps<{
  icon: DocumentationIconKey | string
  iconSvg?: string
  size?: 'sm' | 'md'
}>(), {
  iconSvg: '',
  size: 'sm',
})

type DocumentationIconName = 'key' | 'terminal' | 'server' | 'wallet' | 'upload' | 'questionCircle' | 'book' | 'cpu'

const safeIconSvg = computed(() => {
  return sanitizeDocumentationIconSvg(props.iconSvg)
})

const iconName = computed<DocumentationIconName>(() => {
  const icons: Record<string, DocumentationIconName> = {
    key: 'key',
    client: 'terminal',
    api: 'server',
    wallet: 'wallet',
    image: 'upload',
    help: 'questionCircle',
    book: 'book',
    code: 'cpu',
    terminal: 'terminal',
  }
  return icons[props.icon] ?? 'questionCircle'
})
</script>
