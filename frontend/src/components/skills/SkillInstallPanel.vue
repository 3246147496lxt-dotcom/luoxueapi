<template>
  <aside class="skill-install" :aria-labelledby="`${skill.slug}-install-title`">
    <div class="skill-install__heading">
      <span aria-hidden="true"><Icon name="download" size="md" :stroke-width="1.8" /></span>
      <div>
        <h2 :id="`${skill.slug}-install-title`">{{ t('skills.install.title') }}</h2>
        <p>{{ t('skills.install.localOnly') }}</p>
      </div>
    </div>

    <ol class="skill-install__steps">
      <li>
        <div class="skill-install__step-heading">
          <span>1</span>
          <div>
            <h3>{{ t('skills.install.scopeTitle') }}</h3>
            <p>{{ t('skills.install.scopeDescription') }}</p>
          </div>
        </div>

        <div class="skill-install__scope" role="radiogroup" :aria-label="t('skills.install.scopeTitle')">
          <button
            type="button"
            role="radio"
            :aria-checked="scope === 'personal'"
            :class="{ 'is-active': scope === 'personal' }"
            @click="scope = 'personal'"
          >
            <strong>{{ t('skills.install.personal') }}</strong>
            <span>{{ t('skills.install.personalHint') }}</span>
          </button>
          <button
            type="button"
            role="radio"
            :aria-checked="scope === 'project'"
            :class="{ 'is-active': scope === 'project' }"
            @click="scope = 'project'"
          >
            <strong>{{ t('skills.install.project') }}</strong>
            <span>{{ t('skills.install.projectHint') }}</span>
          </button>
        </div>

        <code class="skill-install__path">{{ installPath }}</code>
      </li>

      <li>
        <div class="skill-install__step-heading">
          <span>2</span>
          <div>
            <h3>{{ t('skills.install.actionTitle') }}</h3>
            <p>{{ t('skills.install.actionDescription') }}</p>
          </div>
        </div>

        <a class="skill-install__review-link" href="#skill-risk-title">
          <Icon name="shield" size="xs" aria-hidden="true" />
          {{ t('skills.install.reviewFirst') }}
        </a>

        <button
          type="button"
          class="skill-install__primary"
          :disabled="!activeVersion"
          @click="copyCodexPrompt"
        >
          <Icon :name="promptCopied ? 'check' : 'copy'" size="sm" aria-hidden="true" />
          {{ t(promptCopied ? 'skills.install.promptCopied' : 'skills.install.copyForCodex') }}
        </button>
        <a
          v-if="activeVersion"
          class="skill-install__secondary"
          :href="downloadURL"
          :download="`${skill.slug}-${activeVersion}.zip`"
          @click="markDownloadStarted"
        >
          <Icon name="download" size="sm" aria-hidden="true" />
          {{ t(downloadStarted ? 'skills.install.downloadStarted' : 'skills.install.downloadZip') }}
        </a>
        <button v-else type="button" class="skill-install__secondary skill-install__secondary--disabled" disabled>
          <Icon name="download" size="sm" aria-hidden="true" />
          {{ t('skills.install.versionUnavailable') }}
        </button>
        <p class="skill-install__disclosure">
          <Icon name="shield" size="sm" aria-hidden="true" />
          {{ t('skills.install.noAutoRun') }}
        </p>
      </li>

      <li>
        <div class="skill-install__step-heading">
          <span>3</span>
          <div>
            <h3>{{ t('skills.install.verifyTitle') }}</h3>
            <p>{{ t('skills.install.verifyDescription') }}</p>
          </div>
        </div>

        <div class="skill-install__verify">
          <code>${{ skill.slug }}</code>
          <span>{{ t('skills.install.verifyOr') }}</span>
          <code>/skills</code>
        </div>
        <p class="skill-install__restart">{{ t('skills.install.restartHint') }}</p>
      </li>
    </ol>
  </aside>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { getSkillVersionDownloadURL, type PublicSkill } from '@/api/skills'

const props = defineProps<{
  skill: PublicSkill
  version?: string
  sha256?: string
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const scope = ref<'personal' | 'project'>('personal')
const promptCopied = ref(false)
const downloadStarted = ref(false)
let promptTimer: ReturnType<typeof setTimeout> | null = null
let downloadTimer: ReturnType<typeof setTimeout> | null = null

const activeVersion = computed(() => props.version || props.skill.current_version?.version || '')
const installPath = computed(() => scope.value === 'personal'
  ? `$HOME/.agents/skills/${props.skill.slug}`
  : `<repo>/.agents/skills/${props.skill.slug}`)
const installParent = computed(() => scope.value === 'personal'
  ? '$HOME/.agents/skills'
  : '<repo>/.agents/skills')
const downloadURL = computed(() => activeVersion.value
  ? getSkillVersionDownloadURL(props.skill.slug, activeVersion.value)
  : '')
const absoluteDownloadURL = computed(() => {
  if (typeof window === 'undefined') return downloadURL.value
  return new URL(downloadURL.value, window.location.origin).href
})

const codexPrompt = computed(() => t('skills.install.codexPrompt', {
  name: props.skill.display_name,
  slug: props.skill.slug,
  version: activeVersion.value,
  url: absoluteDownloadURL.value,
  sha256: props.sha256 || t('skills.install.shaUnavailable'),
  path: installPath.value,
  parent: installParent.value,
}))

async function copyCodexPrompt() {
  if (!activeVersion.value) return
  const copied = await copyToClipboard(codexPrompt.value, t('skills.install.copySuccess'))
  if (!copied) return
  promptCopied.value = true
  if (promptTimer) clearTimeout(promptTimer)
  promptTimer = setTimeout(() => {
    promptCopied.value = false
  }, 2200)
}

function markDownloadStarted() {
  downloadStarted.value = true
  if (downloadTimer) clearTimeout(downloadTimer)
  downloadTimer = setTimeout(() => {
    downloadStarted.value = false
  }, 2600)
}

onBeforeUnmount(() => {
  if (promptTimer) clearTimeout(promptTimer)
  if (downloadTimer) clearTimeout(downloadTimer)
})
</script>

<style scoped>
.skill-install {
  border-radius: 16px;
  padding: 24px;
  background: var(--lx-clay-surface-elevated);
  box-shadow: var(--lx-clay-shadow-overview);
}

.skill-install__heading,
.skill-install__step-heading {
  display: flex;
  align-items: flex-start;
}

.skill-install__heading {
  gap: 13px;
}

.skill-install__heading > span {
  width: 40px;
  height: 40px;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 13px;
  background: var(--lx-clay-accent-soft);
  color: var(--lx-clay-accent-deep);
}

.skill-install h2,
.skill-install h3,
.skill-install p {
  margin: 0;
}

.skill-install h2,
.skill-install h3 {
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-weight: 900;
}

.skill-install h2 {
  font-size: 20px;
  line-height: 1.2;
}

.skill-install__heading p {
  margin-top: 4px;
  color: var(--lx-clay-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.skill-install__steps {
  display: grid;
  gap: 0;
  margin: 24px 0 0;
  padding: 0;
  list-style: none;
}

.skill-install__steps > li {
  padding: 22px 0;
  border-top: 1px solid var(--lx-clay-border);
}

.skill-install__steps > li:last-child {
  padding-bottom: 0;
}

.skill-install__step-heading {
  gap: 11px;
}

.skill-install__step-heading > span {
  width: 26px;
  height: 26px;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  background: var(--lx-clay-accent);
  color: var(--lx-clay-on-accent);
  font-size: 12px;
  font-weight: 900;
}

.skill-install__step-heading h3 {
  font-size: 15px;
}

.skill-install__step-heading p {
  margin-top: 4px;
  color: var(--lx-clay-text-muted);
  font-size: 12px;
  line-height: 1.55;
}

.skill-install__scope {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-top: 14px;
}

.skill-install__scope button {
  min-width: 0;
  border: 1px solid transparent;
  border-radius: 12px;
  padding: 11px 10px;
  background: var(--lx-clay-recessed);
  color: var(--lx-clay-text-secondary);
  text-align: left;
  cursor: pointer;
}

.skill-install__scope button.is-active {
  border-color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
  color: var(--lx-clay-accent-deep);
}

.skill-install__scope strong,
.skill-install__scope span {
  display: block;
}

.skill-install__scope strong {
  font-size: 13px;
}

.skill-install__scope span {
  margin-top: 3px;
  font-size: 11px;
}

.skill-install__path,
.skill-install__verify code {
  font-family: var(--lx-clay-font-mono);
}

.skill-install__path {
  display: block;
  overflow-x: auto;
  margin-top: 10px;
  border-radius: 10px;
  padding: 10px 11px;
  background: var(--lx-clay-code-canvas);
  color: #f8f5fc;
  font-size: 11px;
  white-space: nowrap;
}

.skill-install__primary,
.skill-install__secondary {
  min-height: 46px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border-radius: 13px;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 850;
  text-decoration: none;
}

.skill-install__review-link {
  min-height: 36px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-top: 14px;
  color: var(--lx-clay-accent-deep);
  font-size: 11px;
  font-weight: 800;
  text-decoration: none;
}

.skill-install__review-link:hover {
  text-decoration: underline;
  text-underline-offset: 3px;
}

.skill-install__primary {
  width: 100%;
  margin-top: 7px;
  border: 0;
  background: linear-gradient(145deg, var(--lx-clay-light-accent), var(--lx-clay-accent-deepest));
  box-shadow: var(--lx-clay-shadow-primary);
  color: #fff;
  cursor: pointer;
}

.skill-install__primary:disabled {
  cursor: not-allowed;
  opacity: 0.52;
}

.skill-install__secondary {
  width: 100%;
  border: 0;
  margin-top: 9px;
  background: var(--lx-clay-recessed);
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-ui);
  cursor: pointer;
}

.skill-install__secondary--disabled {
  cursor: not-allowed;
  opacity: 0.52;
}

.skill-install__disclosure {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin-top: 12px !important;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  line-height: 1.55;
}

.skill-install__disclosure svg {
  flex: 0 0 auto;
  margin-top: 1px;
  color: var(--lx-clay-success-text);
}

.skill-install__verify {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 7px;
  margin-top: 14px;
}

.skill-install__verify code {
  border-radius: 9px;
  padding: 7px 9px;
  background: var(--lx-clay-recessed);
  color: var(--lx-clay-accent-deep);
  font-size: 12px;
  font-weight: 800;
}

.skill-install__verify span,
.skill-install__restart {
  color: var(--lx-clay-text-muted);
  font-size: 11px;
}

.skill-install__restart {
  margin-top: 10px !important;
  line-height: 1.55;
}

@media (max-width: 430px) {
  .skill-install {
    padding: 20px;
  }

  .skill-install__scope {
    grid-template-columns: 1fr;
  }
}
</style>
