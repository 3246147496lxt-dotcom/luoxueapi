<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle } from 'lucide-vue-next'
import { desktopApi } from '@/api'
import AppSidebar from '@/components/AppSidebar.vue'
import OverviewView from '@/views/OverviewView.vue'
import CodexView from '@/views/CodexView.vue'
import RequestsView from '@/views/RequestsView.vue'
import SettingsView from '@/views/SettingsView.vue'
import PairingView from '@/views/PairingView.vue'
import { useDesktop } from '@/composables/useDesktop'
import type { ViewName } from '@/types'

const { t } = useI18n()
const {
  snapshot,
  requests,
  pairing,
  state,
  isRunning,
  refresh,
  loadRequests,
  startPairing,
  pollPairing,
  restartGateway,
  setTakeover,
  selectRoute,
  updateSettings,
  checkForUpdates,
  installUpdate
} = useDesktop()
const activeView = ref<ViewName>('overview')

const title = computed(() => t(`nav.${activeView.value}`))

async function logout() {
  await desktopApi.logout()
  await refresh()
}
</script>

<template>
  <div v-if="state.loading" class="app-loading" role="status" aria-live="polite">
    <div class="brand-mark brand-mark--large"><img class="brand-mark__image" src="/logo.png" alt="" /></div>
    <span>{{ t('common.loading') }}</span>
  </div>

  <PairingView
    v-else-if="!snapshot?.paired"
    :pairing="pairing"
    :busy="state.busy === 'pairing'"
    :error="state.error"
    @start="startPairing"
    @poll="pollPairing"
    @open="desktopApi.openPairingPage"
  />

  <div v-else-if="snapshot" class="app-shell">
    <AppSidebar
      :active-view="activeView"
      :snapshot="snapshot"
      @navigate="activeView = $event"
    />

    <main class="workspace">
      <header class="window-header" data-tauri-drag-region>
        <div>
          <h1>{{ title }}</h1>
          <p v-if="activeView === 'overview'" class="window-header__subtitle">
            {{ snapshot.deviceName }}
          </p>
        </div>
        <div class="gateway-chip" :class="`gateway-chip--${snapshot.gatewayStatus}`">
          <span class="status-dot" aria-hidden="true" />
          <span>{{ isRunning ? t('common.running') : t('common.stopped') }}</span>
          <code v-if="snapshot.gatewayPort">:{{ snapshot.gatewayPort }}</code>
        </div>
      </header>

      <div v-if="state.error" class="inline-alert" role="alert">
        <AlertTriangle :size="18" />
        <span>{{ state.error }}</span>
        <button class="text-button" type="button" @click="refresh">{{ t('common.retry') }}</button>
      </div>

      <OverviewView
        v-if="activeView === 'overview'"
        :snapshot="snapshot"
        :requests="requests"
        :busy="state.busy"
        @restart="restartGateway"
        @select-route="selectRoute"
        @load-requests="loadRequests"
        @open-codex="desktopApi.openCodex"
        @copy-command="desktopApi.copyCliCommand"
      />
      <CodexView
        v-else-if="activeView === 'codex'"
        :snapshot="snapshot"
        :busy="state.busy"
        @set-takeover="setTakeover"
        @open-codex="desktopApi.openCodex"
        @copy-command="desktopApi.copyCliCommand"
      />
      <RequestsView
        v-else-if="activeView === 'requests'"
        :requests="requests"
        :loading="state.busy === 'requests'"
        @load="loadRequests"
      />
      <SettingsView
        v-else
        :snapshot="snapshot"
        :busy="state.busy"
        @save="updateSettings"
        @check-update="checkForUpdates"
        @install-update="installUpdate"
        @prepare-uninstall="desktopApi.prepareUninstall"
        @logout="logout"
      />
    </main>
  </div>
</template>
