<template>
  <AppLayout>
    <div class="mx-auto max-w-[950px] space-y-6" data-testid="desktop-devices-shell">
      <AdminPageHeader
        :title="t('desktopDevices.title')"
        :description="t('desktopDevices.description')"
      >
        <template #secondary-actions>
          <button
            type="button"
            class="btn btn-secondary min-h-10"
            :disabled="loading"
            @click="loadDevices"
          >
            <Icon name="refresh" size="sm" :class="{ 'motion-safe:animate-spin': loading }" />
            {{ t('desktopDevices.refresh') }}
          </button>
        </template>
      </AdminPageHeader>

      <div
        v-if="errorMessage"
        class="desktop-devices-alert"
        role="alert"
      >
        <Icon name="exclamationTriangle" size="sm" aria-hidden="true" />
        <span>{{ errorMessage }}</span>
      </div>

      <div
        v-if="loading && devices.length === 0"
        class="desktop-devices-loading"
        role="status"
        aria-live="polite"
      >
        <Icon name="refresh" size="md" class="motion-safe:animate-spin" />
        <span>{{ t('desktopDevices.loading') }}</span>
      </div>

      <div v-else-if="devices.length === 0" class="desktop-devices-empty">
        <span class="desktop-devices-empty__icon" aria-hidden="true">
          <Icon name="cpu" size="lg" />
        </span>
        <div>
          <h2>{{ t('desktopDevices.emptyTitle') }}</h2>
          <p>{{ t('desktopDevices.emptyDescription') }}</p>
        </div>
      </div>

      <section v-else class="desktop-device-list" :aria-label="t('desktopDevices.title')">
        <article
          v-for="device in devices"
          :key="device.id"
          class="desktop-device-card"
          :data-device-id="device.id"
        >
          <div class="desktop-device-card__main">
            <span class="desktop-device-card__icon" aria-hidden="true">
              <Icon name="cpu" size="md" />
            </span>

            <div class="min-w-0 flex-1">
              <form
                v-if="editingId === device.id"
                class="desktop-device-rename"
                @submit.prevent="saveRename(device)"
              >
                <label :for="`desktop-device-name-${device.id}`" class="sr-only">
                  {{ t('desktopDevices.nameLabel') }}
                </label>
                <input
                  :id="`desktop-device-name-${device.id}`"
                  v-model="editingName"
                  class="input min-h-10 min-w-0 flex-1"
                  maxlength="80"
                  autocomplete="off"
                  @keydown.esc="cancelRename"
                />
                <button type="submit" class="btn btn-primary min-h-10" :disabled="busyId === device.id">
                  {{ t('desktopDevices.save') }}
                </button>
                <button type="button" class="btn btn-secondary min-h-10" :disabled="busyId === device.id" @click="cancelRename">
                  {{ t('desktopDevices.cancel') }}
                </button>
              </form>

              <div v-else class="desktop-device-card__heading">
                <h2>{{ device.name }}</h2>
                <span class="desktop-device-status" :class="`desktop-device-status--${device.status}`">
                  {{ statusLabel(device.status) }}
                </span>
              </div>

              <p class="desktop-device-card__system">{{ systemLabel(device) }}</p>
              <div class="desktop-device-card__meta">
                <span>{{ activityLabel(device) }}</span>
                <span v-if="device.approved_at">{{ t('desktopDevices.approvedAt', { time: formatDate(device.approved_at) }) }}</span>
                <span v-if="device.app_version">{{ t('desktopDevices.appVersion', { version: device.app_version }) }}</span>
              </div>
            </div>
          </div>

          <div v-if="device.status !== 'revoked' && editingId !== device.id" class="desktop-device-card__actions">
            <button
              type="button"
              class="btn btn-secondary min-h-10"
              :disabled="busyId === device.id"
              @click="beginRename(device)"
            >
              <Icon name="edit" size="sm" aria-hidden="true" />
              {{ t('desktopDevices.rename') }}
            </button>
            <button
              type="button"
              class="desktop-device-revoke"
              :disabled="busyId === device.id"
              @click="askRevoke(device)"
            >
              <Icon name="trash" size="sm" aria-hidden="true" />
              {{ t('desktopDevices.revoke') }}
            </button>
          </div>
        </article>
      </section>
    </div>

    <ConfirmDialog
      :show="revokeTarget !== null"
      :title="t('desktopDevices.revokeTitle')"
      :message="t('desktopDevices.revokeMessage', { name: revokeTarget?.name || '' })"
      :confirm-text="t('desktopDevices.revokeConfirm')"
      :cancel-text="t('desktopDevices.cancel')"
      danger
      @confirm="confirmRevoke"
      @cancel="revokeTarget = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { desktopAPI, type DesktopDevice } from '@/api/desktop'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'

const { t, locale } = useI18n()
const devices = ref<DesktopDevice[]>([])
const loading = ref(false)
const busyId = ref('')
const errorMessage = ref('')
const editingId = ref('')
const editingName = ref('')
const revokeTarget = ref<DesktopDevice | null>(null)

function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function statusLabel(status: string): string {
  const key = `desktopDevices.${status}`
  const label = t(key)
  return label === key ? status : label
}

function systemLabel(device: DesktopDevice): string {
  return [device.platform, device.os_version, device.architecture]
    .filter(Boolean)
    .join(' · ')
}

function activityLabel(device: DesktopDevice): string {
  if (!device.last_seen_at) return t('desktopDevices.neverSeen')
  return t('desktopDevices.lastSeen', { time: formatDate(device.last_seen_at) })
}

async function loadDevices() {
  loading.value = true
  errorMessage.value = ''
  try {
    devices.value = await desktopAPI.listDevices()
  } catch {
    errorMessage.value = t('desktopDevices.loadError')
  } finally {
    loading.value = false
  }
}

function beginRename(device: DesktopDevice) {
  errorMessage.value = ''
  editingId.value = device.id
  editingName.value = device.name
}

function cancelRename() {
  editingId.value = ''
  editingName.value = ''
}

async function saveRename(device: DesktopDevice) {
  const name = editingName.value.trim()
  if (name.length === 0 || Array.from(name).length > 80) {
    errorMessage.value = t('desktopDevices.invalidName')
    return
  }
  busyId.value = device.id
  errorMessage.value = ''
  try {
    const updated = await desktopAPI.renameDevice(device.id, name)
    devices.value = devices.value.map((candidate) => candidate.id === updated.id ? updated : candidate)
    cancelRename()
  } catch {
    errorMessage.value = t('desktopDevices.renameError')
  } finally {
    busyId.value = ''
  }
}

function askRevoke(device: DesktopDevice) {
  errorMessage.value = ''
  revokeTarget.value = device
}

async function confirmRevoke() {
  const target = revokeTarget.value
  if (!target) return
  revokeTarget.value = null
  busyId.value = target.id
  errorMessage.value = ''
  try {
    const updated = await desktopAPI.revokeDevice(target.id)
    devices.value = devices.value.map((candidate) => candidate.id === updated.id ? updated : candidate)
    if (editingId.value === updated.id) cancelRename()
  } catch {
    errorMessage.value = t('desktopDevices.revokeError')
  } finally {
    busyId.value = ''
  }
}

onMounted(loadDevices)
</script>

<style scoped>
.desktop-devices-alert {
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

.desktop-devices-loading,
.desktop-devices-empty {
  display: flex;
  min-height: 220px;
  align-items: center;
  justify-content: center;
  gap: 14px;
  border-block: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
}

.desktop-devices-empty {
  text-align: left;
}

.desktop-devices-empty__icon,
.desktop-device-card__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-recessed);
}

.desktop-devices-empty__icon {
  width: 52px;
  height: 52px;
  flex: 0 0 52px;
  border-radius: 8px;
}

.desktop-devices-empty h2 {
  color: var(--lx-clay-text);
  font-size: var(--workspace-type-brand-size);
  font-weight: var(--workspace-type-brand-weight);
}

.desktop-devices-empty p {
  max-width: 460px;
  margin-top: 4px;
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.6;
}

.desktop-device-list {
  display: grid;
  gap: 12px;
}

.desktop-device-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  padding: 18px;
  background: var(--lx-clay-surface);
}

.desktop-device-card__main {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: flex-start;
  gap: 14px;
}

.desktop-device-card__icon {
  width: 42px;
  height: 42px;
  flex: 0 0 42px;
  border-radius: 8px;
}

.desktop-device-card__heading,
.desktop-device-card__meta,
.desktop-device-card__actions,
.desktop-device-rename {
  display: flex;
  align-items: center;
}

.desktop-device-card__heading {
  min-width: 0;
  flex-wrap: wrap;
  gap: 8px;
}

.desktop-device-card__heading h2 {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--lx-clay-text);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.desktop-device-status {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  border-radius: 999px;
  padding: 2px 8px;
  color: rgb(55 65 81);
  background: rgb(243 244 246);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.desktop-device-status--active {
  color: rgb(4 120 87);
  background: rgb(209 250 229);
}

.desktop-device-status--pending {
  color: rgb(161 98 7);
  background: rgb(254 249 195);
}

.desktop-device-status--revoked {
  color: rgb(185 28 28);
  background: rgb(254 226 226);
}

.desktop-device-card__system {
  margin-top: 5px;
  color: var(--lx-clay-text-secondary);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.desktop-device-card__meta {
  flex-wrap: wrap;
  gap: 4px 12px;
  margin-top: 7px;
  color: var(--lx-clay-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.desktop-device-card__actions,
.desktop-device-rename {
  gap: 8px;
}

.desktop-device-rename {
  width: 100%;
  flex-wrap: wrap;
}

.desktop-device-revoke {
  display: inline-flex;
  min-height: 40px;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border-radius: 6px;
  padding: 0 12px;
  color: rgb(185 28 28);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.desktop-device-revoke:hover {
  background: rgb(254 242 242);
}

.desktop-device-revoke:focus-visible {
  outline: 2px solid rgb(239 68 68);
  outline-offset: 2px;
}

.desktop-device-revoke:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

:global(html.dark) .desktop-devices-alert {
  border-color: rgb(127 29 29 / 0.7);
  color: rgb(252 165 165);
  background: rgb(69 10 10 / 0.35);
}

:global(html.dark) .desktop-device-status {
  color: rgb(209 213 219);
  background: rgb(55 65 81);
}

:global(html.dark) .desktop-device-status--active {
  color: rgb(110 231 183);
  background: rgb(6 78 59 / 0.55);
}

:global(html.dark) .desktop-device-status--pending {
  color: rgb(253 224 71);
  background: rgb(113 63 18 / 0.55);
}

:global(html.dark) .desktop-device-status--revoked {
  color: rgb(252 165 165);
  background: rgb(127 29 29 / 0.5);
}

:global(html.dark) .desktop-device-revoke:hover {
  background: rgb(69 10 10 / 0.4);
}

@media (max-width: 720px) {
  .desktop-device-card {
    align-items: stretch;
    flex-direction: column;
  }

  .desktop-device-card__actions {
    justify-content: flex-end;
    padding-left: 56px;
  }
}

@media (max-width: 480px) {
  .desktop-devices-empty {
    align-items: flex-start;
    flex-direction: column;
    padding-block: 28px;
  }

  .desktop-device-card {
    padding: 14px;
  }

  .desktop-device-card__icon {
    width: 36px;
    height: 36px;
    flex-basis: 36px;
  }

  .desktop-device-card__actions {
    justify-content: stretch;
    padding-left: 50px;
  }

  .desktop-device-card__actions > * {
    flex: 1;
  }

  .desktop-device-rename > .input {
    flex-basis: 100%;
  }
}
</style>
