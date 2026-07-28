<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, CheckCircle2, Clipboard, Code2, FileCog, Monitor, Power, TerminalSquare } from 'lucide-vue-next'
import type { DesktopSnapshot } from '@/types'

const props = defineProps<{ snapshot: DesktopSnapshot; busy: string }>()
defineEmits<{ setTakeover: [enabled: boolean]; openCodex: []; copyCommand: [] }>()
const { t } = useI18n()

const configLabel = computed(() => t(`codex.${props.snapshot.configStatus}`))
const canToggle = computed(() => props.snapshot.configStatus === 'clean' || props.snapshot.configStatus === 'managed')
const canEnable = computed(() => canToggle.value
  && props.snapshot.routes.length > 0
  && Boolean(props.snapshot.selectedGroupId && props.snapshot.selectedModel)
  && props.snapshot.today.balance > 0)
const takeoverBlockedLabel = computed(() => props.snapshot.routes.length === 0 || !props.snapshot.selectedGroupId
  ? t('codex.routeRequired')
  : t('codex.balanceRequired'))
</script>

<template>
  <div class="view codex-view">
    <section class="takeover-banner" :class="{ 'takeover-banner--active': snapshot.takeoverEnabled }">
      <div class="takeover-banner__copy">
        <span class="takeover-banner__icon"><Code2 :size="24" /></span>
        <div>
          <h2>{{ t('codex.title') }}</h2>
          <p>{{ configLabel }}</p>
        </div>
      </div>
      <button
        class="toggle"
        type="button"
        role="switch"
        :aria-checked="snapshot.takeoverEnabled"
        :disabled="busy === 'takeover' || (snapshot.takeoverEnabled ? !canToggle : !canEnable)"
        @click="$emit('setTakeover', !snapshot.takeoverEnabled)"
      >
        <span />
      </button>
    </section>

    <div v-if="!canToggle" class="conflict-banner" role="alert">
      <AlertTriangle :size="19" />
      <div>
        <strong>{{ configLabel }}</strong>
        <p>{{ snapshot.configMessage }}</p>
      </div>
    </div>
    <div v-else-if="!snapshot.takeoverEnabled && !canEnable" class="conflict-banner" role="status">
      <AlertTriangle :size="19" />
      <div><strong>{{ takeoverBlockedLabel }}</strong></div>
    </div>

    <section class="installations-section">
      <div class="section-heading"><h2>{{ t('codex.applications') }}</h2></div>
      <div class="installation-list">
        <article v-for="installation in snapshot.installations" :key="installation.kind" class="installation-row">
          <span class="app-icon">
            <TerminalSquare v-if="installation.kind === 'cli'" :size="21" />
            <Monitor v-else :size="21" />
          </span>
          <div class="installation-row__identity">
            <strong>{{ installation.kind === 'cli' ? t('codex.cli') : t('codex.desktop') }}</strong>
            <code>{{ installation.path }}</code>
          </div>
          <span class="installation-version">{{ installation.version || '—' }}</span>
          <span class="status-label" :class="installation.installed ? 'status-label--success' : 'status-label--muted'">
            <CheckCircle2 v-if="installation.installed" :size="14" />
            {{ installation.installed ? t('codex.installed') : t('codex.notFound') }}
          </span>
          <button v-if="installation.kind === 'desktop'" class="secondary-button" type="button" @click="$emit('openCodex')">
            {{ t('common.openCodex') }}
          </button>
          <button v-else class="icon-button" type="button" :title="t('common.copyCommand')" @click="$emit('copyCommand')">
            <Clipboard :size="17" />
          </button>
        </article>
      </div>
    </section>

    <section class="config-section">
      <div class="section-heading"><h2>{{ t('codex.config') }}</h2></div>
      <dl class="config-list">
        <div>
          <dt><FileCog :size="17" />config.toml</dt>
          <dd>{{ snapshot.configStatus === 'managed' ? t('codex.managed') : t('codex.clean') }}</dd>
        </div>
        <div>
          <dt><Power :size="17" />{{ t('codex.localAddress') }}</dt>
          <dd><code>http://127.0.0.1:{{ snapshot.gatewayPort }}</code></dd>
        </div>
        <div>
          <dt><CheckCircle2 :size="17" />{{ t('codex.localToken') }}</dt>
          <dd>{{ t('codex.hidden') }}</dd>
        </div>
      </dl>
      <button v-if="snapshot.takeoverEnabled" class="danger-text-button" type="button" :disabled="busy === 'takeover'" @click="$emit('setTakeover', false)">
        {{ t('codex.restore') }}
      </button>
    </section>
  </div>
</template>
