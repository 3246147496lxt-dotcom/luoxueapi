<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <template v-for="(node, index) in iconNodes" :key="`${name}-${index}`">
      <path v-if="node.tag === 'path'" v-bind="node.attrs" />
      <circle v-else-if="node.tag === 'circle'" v-bind="node.attrs" />
      <rect v-else v-bind="node.attrs" />
    </template>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type IconName =
  | 'check'
  | 'chevronDown'
  | 'chevronLeft'
  | 'chevronRight'
  | 'copy'
  | 'eye'
  | 'eyeOff'
  | 'info'
  | 'key'
  | 'penLine'
  | 'plus'
  | 'refreshCw'
  | 'settings2'
  | 'slidersHorizontal'
  | 'sparkles'
  | 'terminal'
  | 'trash2'
  | 'uploadCloud'
  | 'x'

type SvgNode = {
  tag: 'path' | 'circle' | 'rect'
  attrs: Record<string, string | number>
}

const props = withDefaults(defineProps<{
  name: IconName
  size?: number
  strokeWidth?: number
}>(), {
  size: 20,
  strokeWidth: 2,
})

// Nodes mirror the Lucide 0.562 icon set used by the approved Superdesign draft.
const icons: Record<IconName, SvgNode[]> = {
  check: [
    { tag: 'path', attrs: { d: 'M20 6 9 17l-5-5' } },
  ],
  chevronDown: [
    { tag: 'path', attrs: { d: 'm6 9 6 6 6-6' } },
  ],
  chevronLeft: [
    { tag: 'path', attrs: { d: 'm15 18-6-6 6-6' } },
  ],
  chevronRight: [
    { tag: 'path', attrs: { d: 'm9 18 6-6-6-6' } },
  ],
  copy: [
    { tag: 'rect', attrs: { width: 14, height: 14, x: 8, y: 8, rx: 2, ry: 2 } },
    { tag: 'path', attrs: { d: 'M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2' } },
  ],
  eye: [
    { tag: 'path', attrs: { d: 'M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0' } },
    { tag: 'circle', attrs: { cx: 12, cy: 12, r: 3 } },
  ],
  eyeOff: [
    { tag: 'path', attrs: { d: 'M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49' } },
    { tag: 'path', attrs: { d: 'M14.084 14.158a3 3 0 0 1-4.242-4.242' } },
    { tag: 'path', attrs: { d: 'M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143' } },
    { tag: 'path', attrs: { d: 'm2 2 20 20' } },
  ],
  info: [
    { tag: 'circle', attrs: { cx: 12, cy: 12, r: 10 } },
    { tag: 'path', attrs: { d: 'M12 16v-4' } },
    { tag: 'path', attrs: { d: 'M12 8h.01' } },
  ],
  key: [
    { tag: 'path', attrs: { d: 'm15.5 7.5 2.3 2.3a1 1 0 0 0 1.4 0l2.1-2.1a1 1 0 0 0 0-1.4L19 4' } },
    { tag: 'path', attrs: { d: 'm21 2-9.6 9.6' } },
    { tag: 'circle', attrs: { cx: 7.5, cy: 15.5, r: 5.5 } },
  ],
  penLine: [
    { tag: 'path', attrs: { d: 'M13 21h8' } },
    { tag: 'path', attrs: { d: 'M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z' } },
  ],
  plus: [
    { tag: 'path', attrs: { d: 'M5 12h14' } },
    { tag: 'path', attrs: { d: 'M12 5v14' } },
  ],
  refreshCw: [
    { tag: 'path', attrs: { d: 'M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8' } },
    { tag: 'path', attrs: { d: 'M21 3v5h-5' } },
    { tag: 'path', attrs: { d: 'M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16' } },
    { tag: 'path', attrs: { d: 'M8 16H3v5' } },
  ],
  settings2: [
    { tag: 'path', attrs: { d: 'M14 17H5' } },
    { tag: 'path', attrs: { d: 'M19 7h-9' } },
    { tag: 'circle', attrs: { cx: 17, cy: 17, r: 3 } },
    { tag: 'circle', attrs: { cx: 7, cy: 7, r: 3 } },
  ],
  slidersHorizontal: [
    { tag: 'path', attrs: { d: 'M10 5H3' } },
    { tag: 'path', attrs: { d: 'M12 19H3' } },
    { tag: 'path', attrs: { d: 'M14 3v4' } },
    { tag: 'path', attrs: { d: 'M16 17v4' } },
    { tag: 'path', attrs: { d: 'M21 12h-9' } },
    { tag: 'path', attrs: { d: 'M21 19h-5' } },
    { tag: 'path', attrs: { d: 'M21 5h-7' } },
    { tag: 'path', attrs: { d: 'M8 10v4' } },
    { tag: 'path', attrs: { d: 'M8 12H3' } },
  ],
  sparkles: [
    { tag: 'path', attrs: { d: 'M11.017 2.814a1 1 0 0 1 1.966 0l1.051 5.558a2 2 0 0 0 1.594 1.594l5.558 1.051a1 1 0 0 1 0 1.966l-5.558 1.051a2 2 0 0 0-1.594 1.594l-1.051 5.558a1 1 0 0 1-1.966 0l-1.051-5.558a2 2 0 0 0-1.594-1.594l-5.558-1.051a1 1 0 0 1 0-1.966l5.558-1.051a2 2 0 0 0 1.594-1.594z' } },
    { tag: 'path', attrs: { d: 'M20 2v4' } },
    { tag: 'path', attrs: { d: 'M22 4h-4' } },
    { tag: 'circle', attrs: { cx: 4, cy: 20, r: 2 } },
  ],
  terminal: [
    { tag: 'path', attrs: { d: 'M12 19h8' } },
    { tag: 'path', attrs: { d: 'm4 17 6-6-6-6' } },
  ],
  trash2: [
    { tag: 'path', attrs: { d: 'M10 11v6' } },
    { tag: 'path', attrs: { d: 'M14 11v6' } },
    { tag: 'path', attrs: { d: 'M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6' } },
    { tag: 'path', attrs: { d: 'M3 6h18' } },
    { tag: 'path', attrs: { d: 'M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2' } },
  ],
  uploadCloud: [
    { tag: 'path', attrs: { d: 'M12 13v8' } },
    { tag: 'path', attrs: { d: 'M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242' } },
    { tag: 'path', attrs: { d: 'm8 17 4-4 4 4' } },
  ],
  x: [
    { tag: 'path', attrs: { d: 'M18 6 6 18' } },
    { tag: 'path', attrs: { d: 'm6 6 12 12' } },
  ],
}

const iconNodes = computed(() => icons[props.name])
</script>
