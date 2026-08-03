<template>
  <article class="skill-card" :class="{ 'skill-card--featured': featuredLayout }">
    <RouterLink :to="`/skills/${encodeURIComponent(skill.slug)}`" class="skill-card__link">
      <div class="skill-card__topline">
        <span class="skill-card__mark" aria-hidden="true">
          <Icon :name="categoryIcon" size="md" :stroke-width="1.8" />
        </span>
        <span v-if="skill.featured" class="skill-card__badge">{{ t('skills.labels.featured') }}</span>
        <span class="skill-card__category">{{ categoryLabel }}</span>
      </div>

      <div class="skill-card__body">
        <h3>{{ skill.display_name }}</h3>
        <p>{{ skill.summary || t('skills.card.noSummary') }}</p>
      </div>

      <ul v-if="visibleTags.length" class="skill-card__tags" :aria-label="t('skills.card.tags')">
        <li v-for="tag in visibleTags" :key="tag">{{ tag }}</li>
        <li v-if="skill.tags.length > visibleTags.length">+{{ skill.tags.length - visibleTags.length }}</li>
      </ul>

      <div class="skill-card__meta">
        <span v-if="skill.current_version">v{{ skill.current_version.version }}</span>
        <span v-if="skill.download_count > 0">
          {{ t('skills.card.downloads', { count: compactNumber(skill.download_count) }) }}
        </span>
        <span v-if="formattedDate">{{ formattedDate }}</span>
        <Icon name="arrowRight" size="sm" aria-hidden="true" />
      </div>
    </RouterLink>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { PublicSkill } from '@/api/skills'

const props = withDefaults(defineProps<{
  skill: PublicSkill
  featuredLayout?: boolean
  categoryName?: string
}>(), {
  featuredLayout: false,
  categoryName: '',
})

const { t, locale } = useI18n()

const categoryLabel = computed(() => props.categoryName || props.skill.category || t('skills.labels.uncategorized'))
const categoryIcon = computed<'terminal' | 'shield' | 'document' | 'beaker' | 'cube'>(() => {
  const category = props.skill.category.toLowerCase()
  if (category.includes('test') || category.includes('debug')) return 'beaker'
  if (category.includes('quality') || category.includes('security')) return 'shield'
  if (category.includes('doc') || category.includes('data')) return 'document'
  if (category.includes('api') || category.includes('automation')) return 'terminal'
  return 'cube'
})
const visibleTags = computed(() => props.skill.tags.slice(0, props.featuredLayout ? 4 : 3))
const formattedDate = computed(() => {
  const raw = props.skill.published_at || props.skill.updated_at
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(locale.value, {
    month: 'short',
    day: 'numeric',
  }).format(date)
})

function compactNumber(value: number) {
  return new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 1 }).format(value)
}
</script>

<style scoped>
.skill-card {
  min-width: 0;
  border-radius: 16px;
  background: var(--lx-clay-surface-elevated);
  box-shadow: var(--lx-clay-shadow-surface);
  transition: transform 190ms cubic-bezier(0.22, 1, 0.36, 1), box-shadow 190ms ease;
}

.skill-card:hover {
  transform: translateY(-4px);
  box-shadow: var(--lx-clay-shadow-raised);
}

.skill-card__link {
  min-height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  color: inherit;
  text-decoration: none;
}

.skill-card__topline,
.skill-card__meta {
  display: flex;
  align-items: center;
}

.skill-card__topline {
  min-height: 34px;
  gap: 9px;
}

.skill-card__mark {
  width: 34px;
  height: 34px;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 11px;
  background: var(--lx-clay-accent-soft);
  color: var(--lx-clay-accent-deep);
}

.skill-card__badge,
.skill-card__category,
.skill-card__tags li {
  font-size: 12px;
  font-weight: 750;
}

.skill-card__badge {
  border-radius: 999px;
  padding: 5px 9px;
  background: var(--lx-clay-accent);
  color: var(--lx-clay-on-accent);
}

.skill-card__category {
  min-width: 0;
  overflow: hidden;
  color: var(--lx-clay-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-card__body {
  flex: 1;
  margin-top: 22px;
}

.skill-card__body h3 {
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 22px;
  font-weight: 950;
  letter-spacing: -0.025em;
  line-height: 1.18;
}

.skill-card__body p {
  display: -webkit-box;
  overflow: hidden;
  margin: 11px 0 0;
  color: var(--lx-clay-text-secondary);
  font-size: 14px;
  line-height: 1.7;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.skill-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  margin: 20px 0 0;
  padding: 0;
  list-style: none;
}

.skill-card__tags li {
  border-radius: 999px;
  padding: 5px 9px;
  background: var(--lx-clay-recessed);
  color: var(--lx-clay-text-muted);
}

.skill-card__meta {
  gap: 8px;
  margin-top: 22px;
  color: var(--lx-clay-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.skill-card__meta span + span::before {
  margin-right: 8px;
  content: '·';
}

.skill-card__meta svg {
  margin-left: auto;
  color: var(--lx-clay-accent-deep);
  transition: transform 180ms ease;
}

.skill-card:hover .skill-card__meta svg {
  transform: translateX(3px);
}

.skill-card--featured .skill-card__link {
  padding: 28px;
}

.skill-card--featured .skill-card__body h3 {
  font-size: 26px;
}

@media (prefers-reduced-motion: reduce) {
  .skill-card,
  .skill-card__meta svg {
    transition-duration: 0.01ms;
  }
}
</style>
