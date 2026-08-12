<template>
  <form class="skill-import-editor-form" @submit.prevent="emit('save')">
    <header class="skill-import-editor-form__header">
      <div>
        <h2>{{ model.id
          ? t('admin.skills.imports.sourceEditor.editTitle')
          : t('admin.skills.imports.sourceEditor.createTitle') }}</h2>
        <p>{{ t('admin.skills.imports.sourceEditor.description') }}</p>
      </div>
      <div class="skill-import-editor-form__actions">
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

    <section class="skill-import-editor-form__section" aria-labelledby="source-identity-title">
      <div class="skill-import-editor-form__section-heading">
        <h3 id="source-identity-title">{{ t('admin.skills.imports.sourceEditor.identity') }}</h3>
        <p>{{ t('admin.skills.imports.sourceEditor.identityHint') }}</p>
      </div>
      <div class="skill-import-form-grid">
        <label class="skill-import-field skill-import-field--wide">
          <span>{{ t('admin.skills.imports.fields.sourceName') }}</span>
          <input v-model.trim="model.name" class="input" maxlength="120" required />
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.adapter') }}</span>
          <select
            v-model="model.adapter"
            class="input"
            :disabled="Boolean(model.id)"
            @change="applyAdapterDefaults"
          >
            <option value="skills_sh">{{ t('admin.skills.imports.adapter.skills_sh') }}</option>
            <option value="github">{{ t('admin.skills.imports.adapter.github') }}</option>
            <option value="well_known">{{ t('admin.skills.imports.adapter.well_known') }}</option>
            <option value="manifest">{{ t('admin.skills.imports.adapter.manifest') }}</option>
          </select>
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.namespace') }}</span>
          <input v-model.trim="model.namespace" class="input" maxlength="160" required :disabled="Boolean(model.id)" />
          <small>{{ t('admin.skills.imports.sourceEditor.namespaceHint') }}</small>
        </label>
        <label class="skill-import-field skill-import-field--wide">
          <span>{{ t('admin.skills.imports.fields.baseUrl') }}</span>
          <input v-model.trim="model.baseUrl" class="input" type="url" maxlength="2048" />
        </label>
        <label class="skill-import-field">
          <span>{{ t('admin.skills.imports.fields.priority') }}</span>
          <input v-model.number="model.catalogPriority" class="input" type="number" min="0" max="1000000" required />
          <small>{{ t('admin.skills.imports.sourceEditor.priorityHint') }}</small>
        </label>
        <div class="skill-import-switch-field">
          <div>
            <span>{{ t('admin.skills.imports.fields.sourceEnabled') }}</span>
            <small>{{ t('admin.skills.imports.sourceEditor.disableHint') }}</small>
          </div>
          <Toggle
            v-model="model.enabled"
            :aria-label="t('admin.skills.imports.fields.sourceEnabled')"
          />
        </div>
      </div>
    </section>

    <section class="skill-import-editor-form__section" aria-labelledby="source-config-title">
      <div class="skill-import-editor-form__section-heading">
        <h3 id="source-config-title">{{ t('admin.skills.imports.sourceEditor.config') }}</h3>
        <p>{{ t('admin.skills.imports.sourceEditor.configHint') }}</p>
      </div>
      <label class="skill-import-field">
        <span class="sr-only">{{ t('admin.skills.imports.sourceEditor.config') }}</span>
        <textarea
          v-model="model.sourceConfig"
          class="input skill-import-json-editor"
          rows="12"
          spellcheck="false"
          aria-describedby="source-config-safety"
        ></textarea>
      </label>
      <p id="source-config-safety" class="skill-import-inline-note">
        <Icon name="shield" size="sm" aria-hidden="true" />
        {{ t('admin.skills.imports.sourceEditor.safetyNote') }}
      </p>
    </section>
  </form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SkillImportAdapter } from '@/api/admin/skillImport'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'

export interface SkillImportSourceDraft {
  id: number | null
  name: string
  adapter: SkillImportAdapter
  namespace: string
  baseUrl: string
  sourceConfig: string
  catalogPriority: number
  enabled: boolean
}

const model = defineModel<SkillImportSourceDraft>({ required: true })
defineProps<{ saving: boolean }>()
const emit = defineEmits<{
  (event: 'save'): void
  (event: 'cancel'): void
}>()
const { t } = useI18n()

function validBaseURL(value: string): boolean {
  if (!value.trim()) return true
  try {
    const url = new URL(value.trim())
    return url.protocol === 'https:' && Boolean(url.hostname) && !url.username && !url.password
  } catch {
    return false
  }
}

function applyAdapterDefaults(): void {
  const config = model.value.sourceConfig.trim()
  if (model.value.adapter === 'manifest') {
    model.value.baseUrl = ''
    if (!config || config === '{}') model.value.sourceConfig = '{\n  "upload_only": true\n}'
    return
  }
  if (/^\{\s*"upload_only"\s*:\s*true\s*\}$/.test(config)) model.value.sourceConfig = '{}'
}

const isValid = computed(() => Boolean(
  model.value.name.trim()
  && model.value.namespace.trim()
  && validBaseURL(model.value.baseUrl)
  && model.value.catalogPriority >= 0,
))
</script>
