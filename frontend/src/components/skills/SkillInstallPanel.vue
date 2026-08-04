<template>
  <section class="skill-install" :aria-labelledby="`${skill.slug}-install-title`">
    <header class="skill-install__heading">
      <h2 :id="`${skill.slug}-install-title`">{{ t('skills.install.title') }}</h2>
      <div class="skill-install__scope">
        <span>{{ t('skills.install.scopeLabel') }}</span>
        <SettingsChoiceMenu
          class="skill-install__scope-menu"
          :model-value="scope"
          :options="scopeOptions"
          :ariaLabel="t('skills.install.scopeLabel')"
          @update:model-value="updateScope"
        />
      </div>
    </header>

    <div class="skill-install__card">
      <div class="skill-install__modes" role="tablist" :aria-label="t('skills.install.methodLabel')">
        <button
          id="skill-install-command-tab"
          type="button"
          role="tab"
          :aria-selected="mode === 'command'"
          :aria-controls="`${skill.slug}-install-payload`"
          :class="{ 'is-active': mode === 'command' }"
          @click="mode = 'command'"
        >
          {{ t('skills.install.command') }}
        </button>
        <button
          id="skill-install-codex-tab"
          type="button"
          role="tab"
          :aria-selected="mode === 'codex'"
          :aria-controls="`${skill.slug}-install-payload`"
          :class="{ 'is-active': mode === 'codex' }"
          @click="mode = 'codex'"
        >
          {{ t('skills.install.copyForCodex') }}
        </button>
      </div>

      <div
        :id="`${skill.slug}-install-payload`"
        class="skill-install__payload"
        role="tabpanel"
        :aria-labelledby="mode === 'command' ? 'skill-install-command-tab' : 'skill-install-codex-tab'"
      >
        <code>{{ activeVersion ? activePayload : t('skills.install.versionUnavailable') }}</code>
        <button
          type="button"
          :disabled="!activeVersion"
          :aria-label="t(mode === 'command' ? 'skills.install.copyCommandAria' : 'skills.install.copyPromptAria')"
          @click="copyActivePayload"
        >
          <Icon :name="copiedMode === mode ? 'check' : 'copy'" size="sm" aria-hidden="true" />
        </button>
      </div>

      <p class="skill-install__hint">
        <Icon name="infoCircle" size="xs" aria-hidden="true" />
        <span v-if="mode === 'command'">
          {{ t('skills.install.commandHint') }}
          <code>${{ skill.slug }}</code>
          {{ t('skills.install.verifyOr') }}
          <code>/skills</code>
          {{ t('skills.install.verifySuffix') }}
        </span>
        <span v-else>{{ t('skills.install.codexHint') }}</span>
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import SettingsChoiceMenu from '@/components/settings/SettingsChoiceMenu.vue'
import { useClipboard } from '@/composables/useClipboard'
import { getSkillVersionDownloadURL, type PublicSkill } from '@/api/skills'

type InstallMode = 'command' | 'codex'
type InstallScope = 'global' | 'project'

const props = defineProps<{
  skill: PublicSkill
  version?: string
  sha256?: string
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const scope = ref<InstallScope>('global')
const mode = ref<InstallMode>('command')
const copiedMode = ref<InstallMode | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | null = null

const scopeOptions = computed(() => [
  { value: 'global', label: t('skills.install.globalOption') },
  { value: 'project', label: t('skills.install.projectOption') },
])
const activeVersion = computed(() => props.version || props.skill.current_version?.version || '')
const installPath = computed(() => scope.value === 'global'
  ? `$HOME/.agents/skills/${props.skill.slug}`
  : `<repo>/.agents/skills/${props.skill.slug}`)
const installParent = computed(() => scope.value === 'global'
  ? '$HOME/.agents/skills'
  : '<repo>/.agents/skills')
const downloadURL = computed(() => activeVersion.value
  ? getSkillVersionDownloadURL(props.skill.slug, activeVersion.value)
  : '')
const absoluteDownloadURL = computed(() => {
  if (!downloadURL.value || typeof window === 'undefined') return downloadURL.value
  return new URL(downloadURL.value, window.location.origin).href
})

function shellSingleQuote(value: string) {
  return `'${value.replace(/'/g, `'"'"'`)}'`
}

const shellInstallRoot = computed(() => scope.value === 'global'
  ? 'SKILLS_DIR="$HOME/.agents/skills"'
  : 'REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)" && SKILLS_DIR="$REPO_ROOT/.agents/skills"')

const installCommand = computed(() => {
  if (!activeVersion.value) return ''
  const checksum = props.sha256?.trim()
  const steps = [
    'set -eu',
    `SKILL_SLUG=${shellSingleQuote(props.skill.slug)}`,
    shellInstallRoot.value,
    'mkdir -p "$SKILLS_DIR"',
    'TMP_DIR="$(mktemp -d)"',
    'STAGE_ROOT=""',
    'TARGET_DIR=""',
    'BACKUP_DIR=""',
    'cleanup_install() { status=$?; trap - EXIT HUP INT TERM; if [ -n "$BACKUP_DIR" ] && { [ -e "$BACKUP_DIR" ] || [ -L "$BACKUP_DIR" ]; } && [ -n "$TARGET_DIR" ] && ! { [ -e "$TARGET_DIR" ] || [ -L "$TARGET_DIR" ]; }; then mv "$BACKUP_DIR" "$TARGET_DIR"; fi; rm -rf -- "$TMP_DIR"; if [ -n "$STAGE_ROOT" ]; then rm -rf -- "$STAGE_ROOT"; fi; exit "$status"; }',
    'trap cleanup_install EXIT',
    "trap 'exit 130' HUP INT TERM",
    `curl -fsSL ${shellSingleQuote(absoluteDownloadURL.value)} -o "$TMP_DIR/skill.zip"`,
  ]

  if (checksum) {
    steps.push(
      `EXPECTED_SHA256=${shellSingleQuote(checksum)}`,
      'if command -v shasum >/dev/null 2>&1; then (cd "$TMP_DIR" && printf \'%s  skill.zip\\n\' "$EXPECTED_SHA256" | shasum -a 256 -c -); elif command -v sha256sum >/dev/null 2>&1; then (cd "$TMP_DIR" && printf \'%s  skill.zip\\n\' "$EXPECTED_SHA256" | sha256sum -c -); else printf \'A SHA-256 utility is required.\\n\' >&2; false; fi',
    )
  }

  steps.push(
    'STAGE_ROOT="$(mktemp -d "$SKILLS_DIR/.${SKILL_SLUG}.install.XXXXXX")"',
    'TARGET_DIR="$SKILLS_DIR/$SKILL_SLUG"',
    'BACKUP_DIR="$STAGE_ROOT/.previous"',
    'unzip -q "$TMP_DIR/skill.zip" -d "$STAGE_ROOT"',
    'test -f "$STAGE_ROOT/$SKILL_SLUG/SKILL.md"',
    'if [ -e "$TARGET_DIR" ] || [ -L "$TARGET_DIR" ]; then mv "$TARGET_DIR" "$BACKUP_DIR"; fi',
    'mv "$STAGE_ROOT/$SKILL_SLUG" "$TARGET_DIR"',
    'rm -rf -- "$BACKUP_DIR"',
    'trap - EXIT HUP INT TERM',
    'rm -rf -- "$TMP_DIR" "$STAGE_ROOT"',
  )

  return steps.join('; ')
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

const activePayload = computed(() => mode.value === 'command'
  ? installCommand.value
  : codexPrompt.value)

function updateScope(value: string) {
  if (value === 'global' || value === 'project') scope.value = value
}

async function copyActivePayload() {
  if (!activeVersion.value || !activePayload.value) return
  const successMessage = mode.value === 'command'
    ? t('skills.install.commandCopySuccess')
    : t('skills.install.copySuccess')
  const copied = await copyToClipboard(activePayload.value, successMessage)
  if (!copied) return
  copiedMode.value = mode.value
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => {
    copiedMode.value = null
  }, 2200)
}

onBeforeUnmount(() => {
  if (copiedTimer) clearTimeout(copiedTimer)
})
</script>

<style scoped>
.skill-install {
  min-width: 0;
}

.skill-install__heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 24px;
}

.skill-install__heading h2 {
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: clamp(27px, 3vw, 34px);
  font-weight: 950;
  letter-spacing: -0.03em;
  line-height: 1.15;
}

.skill-install__scope {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--lx-clay-text-muted);
  font-size: 12px;
  font-weight: 720;
}

.skill-install__scope-menu {
  width: 194px;
  flex: 0 0 194px;
}

.skill-install__scope-menu :deep(.settings-choice-menu__trigger) {
  width: 100%;
  max-width: none;
  height: 42px;
  justify-content: space-between;
  border-color: color-mix(in srgb, var(--lx-clay-border) 74%, transparent);
  border-radius: 12px;
  padding-inline: 13px 11px;
  background: var(--lx-clay-recessed);
  color: var(--lx-clay-text);
  font-size: 12px;
  font-weight: 790;
}

.skill-install__scope-menu :deep(.settings-choice-menu__trigger:hover),
.skill-install__scope-menu :deep(.settings-choice-menu__trigger--open) {
  border-color: var(--lx-clay-border-strong);
  background: var(--lx-clay-surface-soft);
}

.skill-install__card {
  min-width: 0;
  border: 1px solid transparent;
  border-radius: 17px;
  padding: 26px;
  background: var(--lx-clay-surface-elevated);
  box-shadow: var(--lx-clay-shadow-form);
}

.skill-install__modes {
  width: fit-content;
  display: inline-grid;
  grid-template-columns: 1fr 1fr;
  border-radius: 12px;
  padding: 4px;
  background: var(--lx-clay-recessed);
}

.skill-install__modes button {
  min-height: 42px;
  border: 0;
  border-radius: 9px;
  padding: 0 17px;
  background: transparent;
  color: var(--lx-clay-text-secondary);
  font-family: var(--lx-clay-font-ui);
  font-size: 12px;
  font-weight: 820;
  cursor: pointer;
}

.skill-install__modes button.is-active {
  background: var(--lx-clay-accent-soft);
  color: var(--lx-clay-accent-deep);
  box-shadow: 0 1px 2px color-mix(in srgb, var(--lx-clay-accent) 12%, transparent);
}

.skill-install__payload {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 40px;
  align-items: center;
  gap: 8px;
  margin-top: 24px;
  border-radius: 12px;
  padding: 9px 9px 9px 16px;
  background: var(--lx-clay-code-canvas);
  color: #f8f5fc;
}

.skill-install__payload > code {
  min-width: 0;
  overflow: auto hidden;
  padding: 7px 0;
  font-family: var(--lx-clay-font-mono);
  font-size: 11px;
  line-height: 1.55;
  scrollbar-width: none;
  white-space: nowrap;
}

.skill-install__payload > code::-webkit-scrollbar {
  display: none;
}

.skill-install__payload > button {
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.09);
  color: #fff;
  cursor: pointer;
}

.skill-install__payload > button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.skill-install__hint {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 12px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  line-height: 1.6;
}

.skill-install__hint > svg {
  flex: 0 0 auto;
  margin-top: 2px;
}

.skill-install__hint code {
  border-radius: 6px;
  padding: 1px 5px;
  background: var(--lx-clay-recessed);
  color: var(--lx-clay-accent-deep);
  font-family: var(--lx-clay-font-mono);
  font-size: 10px;
}

.skill-install button:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

@media (max-width: 640px) {
  .skill-install__heading {
    align-items: stretch;
    flex-direction: column;
    gap: 14px;
  }

  .skill-install__scope {
    align-items: stretch;
    flex-direction: column;
    gap: 7px;
  }

  .skill-install__scope-menu {
    width: 100%;
    flex-basis: auto;
  }

  .skill-install__card {
    padding: 24px;
  }

  .skill-install__modes {
    width: 100%;
  }

  .skill-install__modes button {
    padding-inline: 10px;
  }
}

@media (max-width: 390px) {
  .skill-install__card {
    padding: 20px;
  }
}
</style>
