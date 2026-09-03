<template>
  <div
    class="skill-markdown"
    :class="{ 'skill-markdown--compact': compact }"
    v-html="sanitizedHtml"
  ></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'

const props = withDefaults(defineProps<{
  content: string
  compact?: boolean
}>(), {
  compact: false,
})

const sanitizedHtml = computed(() => {
  const markdown = props.content.trim()
  if (!markdown) return ''

  const rendered = marked.parse(markdown, {
    breaks: true,
    gfm: true,
  })

  const sanitized = DOMPurify.sanitize(String(rendered), {
    ALLOWED_TAGS: [
      'p', 'h1', 'h2', 'h3', 'h4', 'ul', 'ol', 'li', 'strong', 'em',
      'code', 'pre', 'blockquote', 'a', 'hr', 'table', 'thead', 'tbody',
      'tr', 'th', 'td', 'br',
    ],
    ALLOWED_ATTR: ['href', 'title'],
    ALLOWED_URI_REGEXP: /^https?:\/\//i,
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
  })

  if (typeof document === 'undefined') return sanitized
  const template = document.createElement('template')
  template.innerHTML = sanitized
  template.content.querySelectorAll('a').forEach((link) => {
    const href = link.getAttribute('href') || ''
    if (!/^https?:\/\//i.test(href)) {
      link.removeAttribute('href')
      return
    }
    link.setAttribute('rel', 'noopener noreferrer nofollow')
  })
  return template.innerHTML
})
</script>

<style scoped>
.skill-markdown {
  color: var(--lx-clay-text-secondary);
  font-size: 15px;
  line-height: 1.78;
  overflow-wrap: anywhere;
}

.skill-markdown :deep(> :first-child) {
  margin-top: 0;
}

.skill-markdown :deep(> :last-child) {
  margin-bottom: 0;
}

.skill-markdown :deep(h1),
.skill-markdown :deep(h2),
.skill-markdown :deep(h3),
.skill-markdown :deep(h4) {
  margin: 1.8em 0 0.6em;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-weight: 900;
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.skill-markdown :deep(h1) { font-size: 26px; }
.skill-markdown :deep(h2) { font-size: 22px; }
.skill-markdown :deep(h3) { font-size: 18px; }
.skill-markdown :deep(h4) { font-size: 16px; }

.skill-markdown :deep(p),
.skill-markdown :deep(ul),
.skill-markdown :deep(ol),
.skill-markdown :deep(blockquote),
.skill-markdown :deep(pre),
.skill-markdown :deep(table) {
  margin: 0 0 1em;
}

.skill-markdown :deep(ul),
.skill-markdown :deep(ol) {
  padding-left: 1.5em;
}

.skill-markdown :deep(li + li) {
  margin-top: 0.35em;
}

.skill-markdown :deep(a) {
  color: var(--lx-clay-accent-deep);
  font-weight: 700;
  text-underline-offset: 3px;
}

.skill-markdown :deep(code) {
  border-radius: 6px;
  padding: 0.12em 0.38em;
  background: var(--lx-clay-recessed);
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.9em;
}

.skill-markdown :deep(pre) {
  overflow-x: auto;
  border-radius: 12px;
  padding: 16px;
  background: var(--lx-clay-code-canvas);
  color: #f8f5fc;
}

.skill-markdown :deep(pre code) {
  padding: 0;
  background: transparent;
  color: inherit;
}

.skill-markdown :deep(blockquote) {
  border-left: 1px solid var(--lx-clay-accent);
  padding-left: 16px;
  color: var(--lx-clay-text-muted);
}

.skill-markdown :deep(table) {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.skill-markdown :deep(th),
.skill-markdown :deep(td) {
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 9px 10px;
  text-align: left;
}

.skill-markdown--compact {
  font-size: 14px;
  line-height: 1.7;
}
</style>
