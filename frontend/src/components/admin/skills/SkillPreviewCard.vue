<template>
  <article class="skill-preview" data-testid="skill-preview-card">
    <header class="skill-preview__header">
      <div class="skill-preview__icon">
        <img v-if="safeIconUrl" :src="safeIconUrl" alt="" />
        <Icon v-else name="cube" size="md" aria-hidden="true" />
      </div>
      <div class="skill-preview__identity">
        <div class="skill-preview__labels">
          <span>{{ t('admin.skills.editor.preview.official') }}</span>
          <span v-if="skill.category">{{ skill.category }}</span>
        </div>
        <h3>{{ skill.display_name || t('admin.skills.editor.fields.displayNamePlaceholder') }}</h3>
        <code v-if="skill.slug">${{ skill.slug }}</code>
      </div>
    </header>

    <p class="skill-preview__summary" :class="{ 'skill-preview__placeholder': !skill.summary }">
      {{ skill.summary || t('admin.skills.editor.preview.noSummary') }}
    </p>
    <p class="skill-preview__description" :class="{ 'skill-preview__placeholder': !skill.description }">
      {{ skill.description || t('admin.skills.editor.preview.noDescription') }}
    </p>

    <div v-if="skill.tags?.length" class="skill-preview__tags" aria-label="Tags">
      <span v-for="tag in skill.tags" :key="tag">{{ tag }}</span>
    </div>

    <section class="skill-preview__examples">
      <h4>{{ t('admin.skills.editor.preview.examples') }}</h4>
      <ol v-if="skill.example_prompts?.length">
        <li v-for="prompt in skill.example_prompts" :key="prompt">
          <Icon name="chat" size="sm" aria-hidden="true" />
          <span>{{ prompt }}</span>
        </li>
      </ol>
      <p v-else class="skill-preview__placeholder">
        {{ t('admin.skills.editor.preview.noExamples') }}
      </p>
    </section>

    <section class="skill-preview__risk">
      <h4>
        <Icon name="shield" size="sm" aria-hidden="true" />
        {{ t('admin.skills.editor.preview.risks') }}
      </h4>
      <p :class="{ 'skill-preview__placeholder': !skill.risk_notes }">
        {{ skill.risk_notes || t('admin.skills.editor.preview.noRisks') }}
      </p>
    </section>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CreateSkillRequest } from '@/api/admin/skills'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  skill: Partial<CreateSkillRequest>
}>()

const { t } = useI18n()

const safeIconUrl = computed(() => {
  const value = props.skill.icon?.trim() ?? ''
  if (value.startsWith('/') || value.startsWith('https://')) return value
  return ''
})
</script>

<style scoped>
.skill-preview {
  overflow: hidden;
  padding: 20px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
}

.skill-preview__header {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 13px;
}

.skill-preview__icon {
  display: flex;
  width: 46px;
  height: 46px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

html.dark .skill-preview__icon {
  color: var(--lx-clay-accent);
}

.skill-preview__icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.skill-preview__identity {
  min-width: 0;
}

.skill-preview__labels {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.skill-preview__labels span,
.skill-preview__tags span {
  display: inline-flex;
  min-height: 22px;
  align-items: center;
  padding: 3px 8px;
  border-radius: 999px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  font-size: 0.6875rem;
  font-weight: 700;
}

.skill-preview__labels span:first-child {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

html.dark .skill-preview__labels span:first-child {
  color: var(--lx-clay-accent);
}

.skill-preview h3 {
  margin: 7px 0 0;
  font-family: var(--lx-clay-font-display);
  font-size: 1.125rem;
  font-weight: 900;
  line-height: 1.25;
  letter-spacing: -0.02em;
  text-wrap: balance;
}

.skill-preview__identity code {
  display: block;
  margin-top: 3px;
  color: var(--lx-clay-text-muted);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.6875rem;
}

.skill-preview__summary {
  margin: 16px 0 0;
  color: var(--lx-clay-text-secondary);
  font-size: 0.875rem;
  font-weight: 750;
  line-height: 1.55;
}

.skill-preview__description {
  margin: 8px 0 0;
  color: var(--lx-clay-text-secondary);
  font-size: 0.8125rem;
  line-height: 1.6;
  white-space: pre-line;
}

.skill-preview__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 14px;
}

.skill-preview__examples,
.skill-preview__risk {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--lx-clay-border);
}

.skill-preview h4 {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 0 0 9px;
  color: var(--lx-clay-text);
  font-size: 0.75rem;
  font-weight: 800;
}

.skill-preview__examples ol {
  display: grid;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.skill-preview__examples li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 9px 10px;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
  font-size: 0.75rem;
  line-height: 1.5;
}

.skill-preview__examples li :deep(svg) {
  flex: 0 0 auto;
  margin-top: 1px;
  color: var(--lx-clay-accent-deep);
}

.skill-preview__risk p,
.skill-preview__examples > p {
  margin: 0;
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
  line-height: 1.6;
  white-space: pre-line;
}

.skill-preview__placeholder {
  color: var(--lx-clay-text-muted) !important;
  font-style: italic;
}
</style>
