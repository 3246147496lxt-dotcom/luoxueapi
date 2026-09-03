<template>
  <form class="skill-import-editor-form" @submit.prevent="emit('save')">
    <header class="skill-import-editor-form__header">
      <div>
        <h2>{{ model.id
          ? t('admin.skills.imports.scheduleEditor.editTitle')
          : t('admin.skills.imports.scheduleEditor.createTitle') }}</h2>
        <p>{{ t('admin.skills.imports.scheduleEditor.description') }}</p>
      </div>
      <div class="skill-import-editor-form__actions">
        <button
          v-if="model.id"
          type="button"
          class="btn btn-secondary btn-sm"
          :disabled="saving || !selectedSource?.enabled"
          @click="emit('run')"
        >
          <Icon name="arrowRight" size="sm" aria-hidden="true" />
          <span class="ml-1.5">{{ t('admin.skills.imports.actions.runNow') }}</span>
        </button>
        <button type="button" class="btn btn-ghost btn-sm" :disabled="saving" @click="emit('cancel')">
          {{ t('common.cancel') }}
        </button>
        <button type="submit" class="btn btn-primary btn-sm" :disabled="saving || !isValid">
          <Icon name="check" size="sm" aria-hidden="true" />
          <span class="ml-1.5">{{ saving
            ? t('common.saving')
            : t('admin.skills.imports.actions.save') }}</span>
        </button>
      </div>
    </header>

    <section class="skill-import-editor-form__section" aria-labelledby="schedule-basics-title">
      <div class="skill-import-editor-form__section-heading">
        <h3 id="schedule-basics-title">{{ t('admin.skills.imports.scheduleEditor.basics') }}</h3>
        <p>{{ t('admin.skills.imports.scheduleEditor.basicsHint') }}</p>
      </div>
      <div class="skill-import-form-grid">
        <label class="skill-import-field skill-import-field--wide">
          <span>{{ t('admin.skills.imports.fields.scheduleName') }}</span>
          <input v-model.trim="model.name" class="input" maxlength="120" required />
        </label>
        <label class="skill-import-field skill-import-field--wide">
          <span>{{ t('admin.skills.imports.fields.source') }}</span>
          <select v-model.number="model.sourceId" class="input" required :disabled="Boolean(model.id)">
            <option :value="0" disabled>{{ t('admin.skills.imports.fields.selectSource') }}</option>
            <option
              v-for="source in sources"
              :key="source.id"
              :value="source.id"
              :disabled="!source.enabled && source.id !== model.sourceId"
            >
              {{ source.name }}{{ source.enabled ? '' : ` · ${t('admin.skills.imports.state.disabled')}` }}
            </option>
          </select>
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.cron') }}</span>
          <input v-model.trim="model.cronExpression" class="input skill-import-mono-input" required />
          <small>{{ t('admin.skills.imports.scheduleEditor.cronHint') }}</small>
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.timezone') }}</span>
          <input v-model.trim="model.timezone" class="input" required />
        </label>
        <div class="skill-import-switch-field skill-import-field--wide">
          <div>
            <span>{{ t('admin.skills.imports.fields.scheduleEnabled') }}</span>
            <small>{{ t('admin.skills.imports.scheduleEditor.disabledDefault') }}</small>
          </div>
          <Toggle
            v-model="model.enabled"
            :aria-label="t('admin.skills.imports.fields.scheduleEnabled')"
          />
        </div>
      </div>
    </section>

    <section class="skill-import-editor-form__section" aria-labelledby="schedule-scope-title">
      <div class="skill-import-editor-form__section-heading">
        <h3 id="schedule-scope-title">{{ t('admin.skills.imports.scheduleEditor.scope') }}</h3>
        <p>{{ t('admin.skills.imports.scheduleEditor.scopeHint') }}</p>
      </div>
      <div class="skill-import-form-grid skill-import-form-grid--four">
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.startRank') }}</span>
          <input v-model.number="model.startRank" class="input" type="number" min="1" required />
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.limit') }}</span>
          <input v-model.number="model.limit" class="input" type="number" min="1" max="5000" required />
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.concurrency') }}</span>
          <input v-model.number="model.concurrency" class="input" type="number" min="1" max="16" required />
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.publishPolicy') }}</span>
          <select v-model="model.publishPolicy" class="input">
            <option value="auto_publish">{{ t('admin.skills.imports.mode.autoPublish') }}</option>
            <option value="review">{{ t('admin.skills.imports.mode.review') }}</option>
          </select>
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.metadataPolicy') }}</span>
          <select v-model="model.metadataPolicy" class="input">
            <option value="refresh">{{ t('admin.skills.imports.metadataPolicy.refresh') }}</option>
            <option value="create_only">{{ t('admin.skills.imports.metadataPolicy.createOnly') }}</option>
          </select>
        </label>
      </div>
    </section>

    <section class="skill-import-editor-form__section skill-import-gate-section" aria-labelledby="schedule-gate-title">
      <div class="skill-import-editor-form__section-heading">
        <h3 id="schedule-gate-title">{{ t('admin.skills.imports.scheduleEditor.gate') }}</h3>
        <p>{{ t('admin.skills.imports.scheduleEditor.gateHint') }}</p>
      </div>
      <div class="skill-import-gate-panel">
        <div class="skill-import-switch-field skill-import-field--wide">
          <div>
            <span>{{ t('admin.skills.imports.fields.safeGate') }}</span>
            <small>{{ t('admin.skills.imports.scheduleEditor.safeGateHint') }}</small>
          </div>
          <Toggle
            v-model="model.safeGate"
            :disabled="model.publishPolicy !== 'review'"
            :aria-label="t('admin.skills.imports.fields.safeGate')"
          />
        </div>
        <div class="skill-import-switch-field">
          <div>
            <span>{{ t('admin.skills.imports.fields.requireAllValid') }}</span>
            <small>{{ t('admin.skills.imports.scheduleEditor.validHint') }}</small>
          </div>
          <Toggle
            v-model="model.requireAllValid"
            :disabled="!model.safeGate"
            :aria-label="t('admin.skills.imports.fields.requireAllValid')"
          />
        </div>
        <div class="skill-import-switch-field">
          <div>
            <span>{{ t('admin.skills.imports.fields.allowLicenseUnverified') }}</span>
            <small>{{ t('admin.skills.imports.scheduleEditor.licenseHint') }}</small>
          </div>
          <Toggle
            v-model="model.allowLicenseUnverified"
            :disabled="!model.safeGate"
            :aria-label="t('admin.skills.imports.fields.allowLicenseUnverified')"
          />
        </div>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.maxBlocked') }}</span>
          <input v-model.number="model.maxBlockedItems" class="input" type="number" min="0" required :disabled="!model.safeGate" />
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.maxFailed') }}</span>
          <input v-model.number="model.maxFailedItems" class="input" type="number" min="0" required :disabled="!model.safeGate" />
        </label>
      </div>
      <p class="skill-import-inline-note">
        <Icon name="shield" size="sm" aria-hidden="true" />
        {{ t('admin.skills.imports.scheduleEditor.transactionNote') }}
      </p>
    </section>
  </form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  SkillImportMetadataPolicy,
  SkillImportPublishPolicy,
  SkillImportSource,
} from '@/api/admin/skillImport'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'

export interface SkillImportScheduleDraft {
  id: number | null
  sourceId: number
  name: string
  enabled: boolean
  cronExpression: string
  timezone: string
  startRank: number
  limit: number
  concurrency: number
  safeGate: boolean
  publishPolicy: SkillImportPublishPolicy
  metadataPolicy: SkillImportMetadataPolicy
  requireAllValid: boolean
  allowLicenseUnverified: boolean
  maxBlockedItems: number
  maxFailedItems: number
}

const props = defineProps<{
  sources: SkillImportSource[]
  saving: boolean
}>()
const model = defineModel<SkillImportScheduleDraft>({ required: true })
const emit = defineEmits<{
  (event: 'save'): void
  (event: 'cancel'): void
  (event: 'run'): void
}>()
const { t } = useI18n()

const selectedSource = computed(() => props.sources.find((source) => source.id === model.value.sourceId))

const isValid = computed(() => Boolean(
  model.value.name.trim()
  && model.value.sourceId > 0
  && selectedSource.value
  && (!model.value.enabled || selectedSource.value.enabled)
  && model.value.cronExpression.trim()
  && model.value.timezone.trim()
  && model.value.startRank > 0
  && model.value.limit > 0
  && model.value.concurrency > 0,
))
</script>
