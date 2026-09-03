<template>
  <article class="skill-card">
    <RouterLink :to="`/skills/${encodeURIComponent(skill.slug)}`" class="skill-card__link">
      <div class="skill-card__body">
        <h3>{{ skill.display_name }}</h3>
        <p>{{ skill.summary || t('skills.card.noSummary') }}</p>
      </div>

      <div class="skill-card__footer">
        <span class="skill-card__category">{{ categoryLabel }}</span>
        <span class="skill-card__action" aria-hidden="true">{{ t('skills.card.view') }}</span>
      </div>
    </RouterLink>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PublicSkill } from '@/api/skills'

const props = withDefaults(defineProps<{
  skill: PublicSkill
  categoryName?: string
}>(), {
  categoryName: '',
})

const { t } = useI18n()

const categoryLabel = computed(() => (
  props.categoryName || props.skill.category || t('skills.labels.uncategorized')
))
</script>

<style scoped>
.skill-card {
  min-width: 0;
  height: 210px;
  min-height: 210px;
  overflow: hidden;
  border: 1px solid var(--skill-border, var(--workspace-border));
  border-radius: 16px;
  background: var(--skill-surface, var(--workspace-card-surface));
  box-shadow: none;
  transition: border-color 180ms ease, box-shadow 180ms ease, transform 180ms ease;
}

.skill-card:hover {
  border-color: var(--skill-border-strong, var(--workspace-border-strong));
  box-shadow: 0 4px 12px rgb(0 0 0 / 0.02);
  transform: translateY(-1px);
}

.skill-card__link {
  height: 100%;
  display: flex;
  flex-direction: column;
  border-radius: 15px;
  padding: 20px;
  color: inherit;
  text-decoration: none;
}

.skill-card__link:focus-visible {
  outline: 2px solid var(--skill-ink, var(--workspace-text));
  outline-offset: -3px;
}

.skill-card__body {
  min-width: 0;
}

.skill-card__body h3 {
  margin: 0;
  overflow: hidden;
  color: var(--skill-ink, var(--workspace-text));
  font-family: var(--workspace-font-ui);
  font-size: 16px;
  font-weight: 600;
  line-height: 1.25;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.skill-card__body p {
  margin: 8px 0 0;
  overflow: hidden;
  color: var(--skill-copy, var(--workspace-text-secondary));
  font-family: var(--workspace-font-ui);
  font-size: 13px;
  line-height: 1.625;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.skill-card__footer {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: auto;
}

.skill-card__category {
  min-width: 0;
  height: 24px;
  display: inline-flex;
  align-items: center;
  overflow: hidden;
  border: 1px solid var(--skill-border, var(--workspace-border));
  border-radius: 6px;
  padding: 0 8px;
  background: var(--skill-surface-subtle, var(--workspace-surface-subtle));
  color: var(--skill-copy, var(--workspace-text-secondary));
  font-size: 11px;
  font-weight: 500;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-card__action {
  width: 72px;
  height: 36px;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--skill-action, #0b8bed);
  border-radius: 999px;
  background: var(--skill-action, #0b8bed);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
  transition: background-color 160ms ease, border-color 160ms ease;
}

.skill-card:hover .skill-card__action,
.skill-card__link:focus-visible .skill-card__action {
  border-color: var(--skill-action-hover, #075985);
  background: var(--skill-action-hover, #075985);
}

@media (prefers-reduced-motion: reduce) {
  .skill-card,
  .skill-card__action {
    transition-duration: 0.01ms;
  }

  .skill-card:hover {
    transform: none;
  }
}
</style>
