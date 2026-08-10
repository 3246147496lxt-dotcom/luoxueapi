<template>
  <AppLayout>
    <div class="mx-auto max-w-[950px] space-y-6" data-testid="quota-viewer-devices-shell">
      <AdminPageHeader
        :title="t('quotaViewerDevices.title')"
        :description="t('quotaViewerDevices.description')"
      >
        <template #secondary-actions>
          <button
            type="button"
            class="btn btn-secondary min-h-11"
            :disabled="loading"
            @click="loadDevices"
          >
            <Icon name="refresh" size="sm" :class="{ 'motion-safe:animate-spin': loading }" />
            {{ t('quotaViewerDevices.refresh') }}
          </button>
        </template>
      </AdminPageHeader>

      <div v-if="errorMessage" class="quota-viewer-devices-alert" role="alert">
        <Icon name="exclamationTriangle" size="sm" aria-hidden="true" />
        <span>{{ errorMessage }}</span>
      </div>

      <div
        v-if="loading && devices.length === 0"
        class="quota-viewer-devices-state"
        role="status"
        aria-live="polite"
      >
        <Icon name="refresh" size="md" class="motion-safe:animate-spin" />
        <span>{{ t('quotaViewerDevices.loading') }}</span>
      </div>

      <div v-else-if="devices.length === 0" class="quota-viewer-devices-state">
        <span class="quota-viewer-device-icon quota-viewer-device-icon--empty" aria-hidden="true">
          <Icon name="shield" size="lg" />
        </span>
        <div>
          <h2>{{ t('quotaViewerDevices.emptyTitle') }}</h2>
        <p>{{ t('quotaViewerDevices.emptyDescription') }}</p>
        <RouterLink to="/quota-viewer" class="btn btn-secondary mt-2">
          <Icon name="download" size="sm" aria-hidden="true" />
          {{ t('quotaViewerDevices.getViewer') }}
        </RouterLink>
      </div>
      </div>

      <section v-else class="quota-viewer-device-list" :aria-label="t('quotaViewerDevices.title')">
        <article
          v-for="device in devices"
          :key="device.id"
          class="quota-viewer-device"
          :data-device-id="device.id"
        >
          <div class="quota-viewer-device__main">
            <span class="quota-viewer-device-icon" aria-hidden="true">
              <Icon name="shield" size="md" />
            </span>

            <div class="min-w-0 flex-1">
              <div class="quota-viewer-device__heading">
                <h2>{{ device.name }}</h2>
                <span
                  class="quota-viewer-device-status"
                  :class="`quota-viewer-device-status--${device.status}`"
                >
                  {{ statusLabel(device.status) }}
                </span>
              </div>
              <p class="quota-viewer-device__system">{{ systemLabel(device) }}</p>
              <div class="quota-viewer-device__meta">
                <span>{{ activityLabel(device) }}</span>
                <span v-if="device.approved_at">
                  {{ t('quotaViewerDevices.approvedAt', { time: formatDate(device.approved_at) }) }}
                </span>
                <span v-if="device.app_version">
                  {{ t('quotaViewerDevices.appVersion', { version: device.app_version }) }}
                </span>
              </div>
            </div>
          </div>

          <button
            v-if="device.status !== 'revoked'"
            type="button"
            class="quota-viewer-device__revoke"
            :disabled="busyId === device.id"
            @click="askRevoke(device)"
          >
            <Icon
              :name="busyId === device.id ? 'refresh' : 'trash'"
              size="sm"
              aria-hidden="true"
              :class="{ 'motion-safe:animate-spin': busyId === device.id }"
            />
            {{ t('quotaViewerDevices.revoke') }}
          </button>
        </article>
      </section>
    </div>

    <ConfirmDialog
      :show="revokeTarget !== null"
      :title="t('quotaViewerDevices.revokeTitle')"
      :message="t('quotaViewerDevices.revokeMessage', { name: revokeTarget?.name || '' })"
      :confirm-text="t('quotaViewerDevices.revokeConfirm')"
      :cancel-text="t('quotaViewerDevices.cancel')"
      danger
      @confirm="confirmRevoke"
      @cancel="revokeTarget = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  quotaViewerAPI,
  type QuotaViewerDevice,
} from '@/api/quotaViewer'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'

const { t, locale } = useI18n()
const devices = ref<QuotaViewerDevice[]>([])
const loading = ref(false)
const busyId = ref('')
const errorMessage = ref('')
const revokeTarget = ref<QuotaViewerDevice | null>(null)

function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function statusLabel(status: string): string {
  const key = `quotaViewerDevices.${status}`
  const label = t(key)
  return label === key ? status : label
}

function systemLabel(device: QuotaViewerDevice): string {
  return [device.platform, device.os_version, device.architecture]
    .filter(Boolean)
    .join(' · ')
}

function activityLabel(device: QuotaViewerDevice): string {
  if (!device.last_seen_at) return t('quotaViewerDevices.neverSeen')
  return t('quotaViewerDevices.lastSeen', { time: formatDate(device.last_seen_at) })
}

async function loadDevices() {
  if (loading.value) return
  loading.value = true
  errorMessage.value = ''
  try {
    devices.value = await quotaViewerAPI.listDevices()
  } catch {
    errorMessage.value = t('quotaViewerDevices.loadError')
  } finally {
    loading.value = false
  }
}

function askRevoke(device: QuotaViewerDevice) {
  errorMessage.value = ''
  revokeTarget.value = device
}

async function confirmRevoke() {
  const target = revokeTarget.value
  if (!target || busyId.value) return
  revokeTarget.value = null
  busyId.value = target.id
  errorMessage.value = ''
  try {
    const updated = await quotaViewerAPI.revokeDevice(target.id)
    devices.value = devices.value.map((device) => device.id === updated.id ? updated : device)
  } catch {
    errorMessage.value = t('quotaViewerDevices.revokeError')
  } finally {
    busyId.value = ''
  }
}

onMounted(loadDevices)
</script>

<style scoped>
.quota-viewer-devices-alert {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  border: 1px solid rgb(254 202 202);
  border-radius: 8px;
  padding: 12px 14px;
  color: rgb(185 28 28);
  background: rgb(254 242 242);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
}

.quota-viewer-devices-state {
  display: flex;
  min-height: 220px;
  align-items: center;
  justify-content: center;
  gap: 14px;
  border-block: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
}

.quota-viewer-devices-state h2 {
  color: var(--lx-clay-text);
  font-size: var(--workspace-type-brand-size);
  font-weight: var(--workspace-type-brand-weight);
}

.quota-viewer-devices-state p {
  max-width: 480px;
  margin-top: 4px;
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.6;
}

.quota-viewer-device-list {
  display: grid;
  gap: 12px;
}

.quota-viewer-device {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  padding: 18px;
  background: var(--lx-clay-surface);
}

.quota-viewer-device__main {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: flex-start;
  gap: 14px;
}

.quota-viewer-device-icon {
  display: inline-flex;
  width: 42px;
  height: 42px;
  flex: 0 0 42px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-recessed);
}

.quota-viewer-device-icon--empty {
  width: 52px;
  height: 52px;
  flex-basis: 52px;
}

.quota-viewer-device__heading,
.quota-viewer-device__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}

.quota-viewer-device__heading {
  min-width: 0;
  gap: 8px;
}

.quota-viewer-device__heading h2 {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--lx-clay-text);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.quota-viewer-device-status {
  display: inline-flex;
  min-height: 22px;
  align-items: center;
  border-radius: 999px;
  padding: 2px 8px;
  color: rgb(55 65 81);
  background: rgb(243 244 246);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.quota-viewer-device-status--active {
  color: rgb(4 120 87);
  background: rgb(209 250 229);
}

.quota-viewer-device-status--pending {
  color: rgb(161 98 7);
  background: rgb(254 249 195);
}

.quota-viewer-device-status--revoked {
  color: rgb(185 28 28);
  background: rgb(254 226 226);
}

.quota-viewer-device__system {
  margin-top: 5px;
  color: var(--lx-clay-text-secondary);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.quota-viewer-device__meta {
  gap: 4px 12px;
  margin-top: 7px;
  color: var(--lx-clay-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.quota-viewer-device__revoke {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border-radius: 7px;
  padding: 0 14px;
  color: rgb(185 28 28);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  transition:
    background-color 160ms ease,
    opacity 160ms ease;
}

.quota-viewer-device__revoke:hover:not(:disabled) {
  background: rgb(254 242 242);
}

.quota-viewer-device__revoke:focus-visible {
  outline: 2px solid rgb(220 38 38);
  outline-offset: 2px;
}

.quota-viewer-device__revoke:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

@media (max-width: 640px) {
  .quota-viewer-device {
    align-items: stretch;
    flex-direction: column;
  }

  .quota-viewer-device__revoke {
    width: 100%;
  }
}
</style>
