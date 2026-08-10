<template>
  <article class="skill-card">
    <RouterLink :to="`/skills/${encodeURIComponent(skill.slug)}`" class="skill-card__link">
      <div class="skill-card__body">
        <span class="skill-card__category">{{ categoryLabel }}</span>
        <h3>{{ skill.display_name }}</h3>
      </div>

      <div v-if="visibleTags.length" class="skill-card__meta">
        <ul class="skill-card__tags" :aria-label="t('skills.card.tags')">
          <li v-for="tag in visibleTags" :key="tag">{{ tag }}</li>
        </ul>
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
const hiddenMarketplaceTags = new Set(['anthropic', 'codex'])
const preferredMarketplaceTags = ['html/css', 'ui/ux']

const categoryLabel = computed(() => (
  props.categoryName || props.skill.category || t('skills.labels.uncategorized')
))
const visibleTags = computed(() => {
  const candidates = props.skill.tags.filter(
    (tag) => !hiddenMarketplaceTags.has(tag.trim().toLowerCase()),
  )
  const byNormalizedTag = new Map(
    candidates.map((tag) => [tag.trim().toLowerCase(), tag] as const),
  )
  const preferred = preferredMarketplaceTags
    .map((tag) => byNormalizedTag.get(tag))
    .filter((tag): tag is string => Boolean(tag))
  const remaining = candidates.filter(
    (tag) => !preferredMarketplaceTags.includes(tag.trim().toLowerCase()),
  )
  return [...preferred, ...remaining].slice(0, 2)
})
</script>

<style scoped>
.skill-card {
  min-width: 0;
  overflow: visible;
  border: 1px solid var(--lx-clay-border);
  border-radius: 14px;
  background: var(--lx-clay-surface-elevated);
  box-shadow: none;
  transition: border-color 180ms ease, box-shadow 180ms ease, transform 180ms ease;
}

.skill-card:hover {
  border-color: color-mix(in srgb, var(--lx-clay-accent-deep) 35%, var(--lx-clay-border));
  box-shadow: 0 10px 26px color-mix(in srgb, var(--lx-clay-accent-deep) 8%, transparent);
  transform: translateY(-1px);
}

.skill-card__link {
  min-height: 89.5px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(130px, auto);
  align-items: center;
  gap: 40px;
  border-radius: 13px;
  padding: 18px 24px;
  color: inherit;
  text-decoration: none;
}

.skill-card__link:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent-deep) 65%, transparent);
  outline-offset: 3px;
}

.skill-card__body {
  min-width: 0;
}

.skill-card__category {
  display: block;
  overflow: hidden;
  margin-bottom: 6px;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.02em;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-card__body h3 {
  overflow: hidden;
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 18px;
  font-weight: 900;
  letter-spacing: -0.015em;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-card__meta {
  min-width: 130px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.skill-card__tags {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.skill-card__tags li {
  border-radius: 6px;
  padding: 3px 8px;
  background: var(--lx-clay-recessed);
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  font-weight: 750;
  line-height: 1.3;
  white-space: nowrap;
}

@media (max-width: 1023px) {
  .skill-card__link {
    min-height: 157px;
    grid-template-columns: minmax(0, 1fr);
    align-content: center;
    gap: 29px;
    padding: 24px;
  }

  .skill-card__meta {
    min-width: 0;
    justify-content: flex-start;
    border-top: 1px solid var(--lx-clay-border);
    padding-top: 16px;
  }

  .skill-card__tags {
    justify-content: flex-start;
  }
}

@media (max-width: 767px) {
  .skill-card__link {
    min-height: 141px;
    gap: 30.5px;
    padding: 16px;
  }

  .skill-card__meta {
    padding-top: 14px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skill-card {
    transition-duration: 0.01ms;
  }
}
</style>
