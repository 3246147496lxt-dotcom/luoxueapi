<template>
  <section class="skill-report" aria-labelledby="skill-report-title">
    <div class="skill-report__heading">
      <div>
        <h3 id="skill-report-title">{{ t('admin.skills.editor.report.title') }}</h3>
        <p v-if="version">
          {{ t('admin.skills.editor.report.files', { count: version.file_count }) }}
          <span aria-hidden="true"> · </span>
          {{ t('admin.skills.editor.report.packageSize', { size: formatBytes(version.byte_size) }) }}
          <span aria-hidden="true"> · </span>
          {{ t('admin.skills.editor.report.unpackedSize', { size: formatBytes(version.unpacked_size) }) }}
        </p>
      </div>
      <SkillStatusBadge v-if="version" :status="version.status" />
    </div>

    <div v-if="!effectiveReport" class="skill-report__empty" data-testid="skill-report-empty">
      <Icon name="shield" size="lg" aria-hidden="true" />
      <p>{{ t('admin.skills.editor.report.noReport') }}</p>
    </div>

    <template v-else>
      <div
        class="skill-report__summary"
        :class="effectiveReport.valid ? 'skill-report__summary--passed' : 'skill-report__summary--failed'"
        role="status"
        data-testid="skill-report-summary"
      >
        <Icon
          :name="effectiveReport.valid ? 'checkCircle' : 'exclamationTriangle'"
          size="md"
          aria-hidden="true"
        />
        <div>
          <p class="skill-report__summary-title">
            {{ effectiveReport.valid
              ? t('admin.skills.editor.report.passed')
              : t('admin.skills.editor.report.failed') }}
          </p>
          <p class="skill-report__summary-counts">
            {{ t('admin.skills.editor.report.errors', { count: errors.length }) }}
            <span aria-hidden="true"> · </span>
            {{ t('admin.skills.editor.report.warnings', { count: warnings.length }) }}
          </p>
        </div>
      </div>

      <div v-if="issues.length" class="skill-report__issues">
        <div
          v-for="issue in issues"
          :key="`${issue.severity}:${issue.code}:${issue.path || ''}:${issue.message}`"
          class="skill-report__issue"
          :class="`skill-report__issue--${issue.severity}`"
        >
          <Icon
            :name="issue.severity === 'error' ? 'xCircle' : 'exclamationCircle'"
            size="sm"
            aria-hidden="true"
          />
          <div>
            <p>{{ issue.message }}</p>
            <code v-if="issue.path">{{ issue.path }}</code>
          </div>
        </div>
      </div>
      <p v-else class="skill-report__clean">
        <Icon name="check" size="sm" aria-hidden="true" />
        {{ t('admin.skills.editor.report.noIssues') }}
      </p>

      <dl v-if="version" class="skill-report__facts">
        <div>
          <dt>{{ t('admin.skills.editor.report.checksum') }}</dt>
          <dd><code :title="version.sha256">{{ compactHash(version.sha256) }}</code></dd>
        </div>
        <div>
          <dt>SKILL.md</dt>
          <dd>{{ version.manifest_name || '—' }}</dd>
        </div>
      </dl>

      <details v-if="version?.file_manifest.length" class="skill-report__manifest">
        <summary>
          <span>{{ t('admin.skills.editor.report.manifest') }}</span>
          <span>{{ t('admin.skills.editor.report.files', { count: version.file_manifest.length }) }}</span>
        </summary>
        <ul>
          <li v-for="file in version.file_manifest" :key="file.path">
            <code>{{ file.path }}</code>
            <span>{{ formatBytes(file.byte_size) }}</span>
          </li>
        </ul>
      </details>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  AdminSkillVersion,
  SkillValidationIssue,
  SkillValidationReport,
} from '@/api/admin/skills'
import Icon from '@/components/icons/Icon.vue'
import SkillStatusBadge from './SkillStatusBadge.vue'

const props = defineProps<{
  version: AdminSkillVersion | null
  report?: SkillValidationReport | null
}>()

const { t } = useI18n()

const effectiveReport = computed(() => props.report ?? props.version?.validation_report ?? null)
const errors = computed(() => effectiveReport.value?.errors ?? [])
const warnings = computed(() => effectiveReport.value?.warnings ?? [])
const issues = computed(() => [
  ...errors.value.map((issue) => ({ ...issue, severity: 'error' as const })),
  ...warnings.value.map((issue) => ({ ...issue, severity: 'warning' as const })),
] satisfies Array<SkillValidationIssue & { severity: 'error' | 'warning' }>)

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / Math.pow(1024, unitIndex)
  return `${value >= 10 || unitIndex === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unitIndex]}`
}

function compactHash(hash: string): string {
  if (!hash) return '—'
  return hash.length > 20 ? `${hash.slice(0, 12)}…${hash.slice(-8)}` : hash
}
</script>

<style scoped>
.skill-report {
  display: grid;
  min-width: 0;
  gap: 14px;
}

.skill-report__heading {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.skill-report__heading h3 {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 0.9375rem;
  font-weight: 800;
}

.skill-report__heading p {
  margin: 4px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  line-height: 1.5;
}

.skill-report__empty {
  display: flex;
  min-height: 126px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 24px;
  border: 1px dashed var(--lx-clay-border-strong);
  border-radius: var(--lx-clay-radius-form);
  color: var(--lx-clay-text-muted);
  background: var(--lx-clay-surface-soft);
  text-align: center;
}

.skill-report__empty p {
  max-width: 50ch;
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.6;
}

.skill-report__summary {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  padding: 13px 14px;
  border: 1px solid transparent;
  border-radius: var(--lx-clay-radius-control);
}

.skill-report__summary--passed {
  border-color: color-mix(in srgb, var(--lx-clay-success) 24%, transparent);
  color: var(--lx-clay-success-text);
  background: var(--lx-clay-success-soft);
}

.skill-report__summary--failed {
  border-color: color-mix(in srgb, var(--lx-clay-danger) 24%, transparent);
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.skill-report__summary-title,
.skill-report__summary-counts,
.skill-report__issue p,
.skill-report__clean {
  margin: 0;
}

.skill-report__summary-title {
  font-size: 0.8125rem;
  font-weight: 800;
  line-height: 1.45;
}

.skill-report__summary-counts {
  margin-top: 2px;
  font-size: 0.75rem;
  opacity: 0.82;
}

.skill-report__issues {
  display: grid;
  gap: 8px;
}

.skill-report__issue {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  padding: 10px 12px;
  border-radius: var(--lx-clay-radius-control);
  background: var(--lx-clay-surface-soft);
  font-size: 0.8125rem;
  line-height: 1.5;
}

.skill-report__issue--error {
  color: var(--lx-clay-danger);
}

.skill-report__issue--warning {
  color: var(--lx-clay-warning);
}

.skill-report__issue code {
  display: block;
  margin-top: 2px;
  color: currentColor;
  font-family: var(--lx-clay-font-mono);
  font-size: 0.6875rem;
  opacity: 0.8;
}

.skill-report__clean {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--lx-clay-success-text);
  font-size: 0.8125rem;
  font-weight: 700;
}

.skill-report__facts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
  border-top: 1px solid var(--lx-clay-border);
  border-bottom: 1px solid var(--lx-clay-border);
}

.skill-report__facts > div {
  min-width: 0;
  padding: 11px 0;
}

.skill-report__facts > div + div {
  padding-left: 14px;
  border-left: 1px solid var(--lx-clay-border);
}

.skill-report__facts dt {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 700;
}

.skill-report__facts dd {
  overflow: hidden;
  margin: 4px 0 0;
  color: var(--lx-clay-text);
  font-size: 0.75rem;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-report__facts code {
  font-family: var(--lx-clay-font-mono);
}

.skill-report__manifest {
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  background: var(--lx-clay-surface-soft);
}

.skill-report__manifest summary {
  display: flex;
  min-height: 42px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 9px 12px;
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
  font-weight: 750;
  cursor: pointer;
}

.skill-report__manifest ul {
  max-height: 220px;
  overflow: auto;
  margin: 0;
  padding: 0 12px 10px;
  list-style: none;
}

.skill-report__manifest li {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 7px 0;
  border-top: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
}

.skill-report__manifest code {
  overflow: hidden;
  color: var(--lx-clay-text-secondary);
  font-family: var(--lx-clay-font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 640px) {
  .skill-report__facts {
    grid-template-columns: minmax(0, 1fr);
  }

  .skill-report__facts > div + div {
    padding-left: 0;
    border-top: 1px solid var(--lx-clay-border);
    border-left: 0;
  }
}
</style>
