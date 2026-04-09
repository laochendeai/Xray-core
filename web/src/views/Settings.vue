<template>
  <n-space vertical :size="16">
    <h2>{{ t('settings.title') }}</h2>

    <n-tabs v-model:value="activeTab" type="line">
      <n-tab-pane name="tun" :tab="t('settings.tun')">
        <n-space vertical :size="12">
          <n-alert type="info" :title="t('settings.tunSafetyTitle')">
            {{ t('settings.tunSafetyDesc') }}
          </n-alert>

          <n-card size="small">
            <n-space vertical :size="16">
              <div style="display: flex; justify-content: space-between; align-items: center; gap: 16px; flex-wrap: wrap">
                <n-space align="center" :size="12">
                  <strong>{{ t('settings.transparentMode') }}</strong>
                  <n-tag :type="tunStatusType" size="small">
                    {{ tunStatusText }}
                  </n-tag>
                </n-space>
                <n-space :size="12" align="center">
                  <n-button type="warning" secondary :loading="installingTunBootstrap" @click="handleInstallTunBootstrap">
                    {{ t('settings.installTunPrivilege') }}
                  </n-button>
                  <n-button @click="goToNodePool">
                    {{ t('settings.openNodePool') }}
                  </n-button>
                  <n-button @click="fetchTunStatus" :loading="loadingTun">
                    {{ t('common.refresh') }}
                  </n-button>
                </n-space>
              </div>

              <n-alert type="info" :title="t('settings.primaryControlsTitle')">
                {{ t('settings.primaryControlsDesc') }}
              </n-alert>

              <n-alert v-if="tunStatus.message" :type="tunStatus.running ? 'success' : (tunStatus.available ? 'warning' : 'error')">
                {{ tunStatus.message }}
              </n-alert>

              <n-alert v-if="tunStatus.useSudo && !tunStatus.elevationReady" type="warning" :title="t('settings.tunPrivilegeTitle')">
                {{ t('settings.tunPrivilegeDesc') }}
              </n-alert>

              <n-alert v-if="!tunStatus.helperExists" type="error">
                {{ t('settings.tunHelperMissing') }}
              </n-alert>

              <n-alert v-if="tunRepairRecommended" type="warning" :title="t('settings.tunRepairTitle')">
                {{ t('settings.tunRepairDesc') }}
              </n-alert>

              <n-card v-if="tunBootstrapNeeded" size="small" embedded>
                <n-space justify="space-between" align="center" wrap>
                  <div>{{ t('settings.tunInstallDesc') }}</div>
                  <n-button type="warning" secondary :loading="installingTunBootstrap" @click="handleInstallTunBootstrap">
                    {{ t('settings.installTunPrivilege') }}
                  </n-button>
                </n-space>
              </n-card>

              <n-descriptions bordered :column="1" size="small">
                <n-descriptions-item :label="t('settings.machineState')">{{ machineStateText }}</n-descriptions-item>
                <n-descriptions-item :label="t('settings.lastFallbackReason')">{{ machineReasonText }}</n-descriptions-item>
                <n-descriptions-item :label="t('settings.transparentMode')">{{ tunStatus.running ? t('common.enabled') : t('common.disabled') }}</n-descriptions-item>
                <n-descriptions-item :label="t('settings.localOnly')">{{ tunStatus.allowRemote ? t('common.disabled') : t('common.enabled') }}</n-descriptions-item>
                <n-descriptions-item :label="t('settings.tunInterface')">{{ tunStatus.interfaceName || '-' }}</n-descriptions-item>
                <n-descriptions-item label="MTU">{{ tunStatus.mtu || '-' }}</n-descriptions-item>
                <n-descriptions-item :label="t('settings.remoteDns')">{{ (tunStatus.remoteDns || []).join(', ') || '-' }}</n-descriptions-item>
                <n-descriptions-item :label="t('settings.privilegeMode')">{{ tunStatus.useSudo ? 'sudo -n' : 'direct' }}</n-descriptions-item>
                <n-descriptions-item :label="t('settings.helperPath')">{{ tunStatus.helperPath || '-' }}</n-descriptions-item>
                <n-descriptions-item :label="t('settings.runtimeConfigPath')">{{ tunStatus.runtimeConfigPath || '-' }}</n-descriptions-item>
              </n-descriptions>

              <n-card size="small" embedded :title="t('settings.egressObservations')">
                <n-descriptions bordered :column="1" size="small">
                  <n-descriptions-item :label="t('settings.directEgress')">{{ tunEgressHeadline(tunStatus.directEgress) }}</n-descriptions-item>
                  <n-descriptions-item :label="t('settings.proxyEgress')">{{ tunEgressHeadline(tunStatus.proxyEgress) }}</n-descriptions-item>
                </n-descriptions>
                <div v-if="tunEgressMeta(tunStatus.directEgress)" class="settings-tun-meta">
                  <strong>{{ t('settings.directEgress') }}:</strong> {{ tunEgressMeta(tunStatus.directEgress) }}
                </div>
                <div v-if="tunEgressMeta(tunStatus.proxyEgress)" class="settings-tun-meta">
                  <strong>{{ t('settings.proxyEgress') }}:</strong> {{ tunEgressMeta(tunStatus.proxyEgress) }}
                </div>
              </n-card>

              <n-card size="small" embedded :title="t('settings.routingDecisions')">
                <div v-if="routingDiagnostics.length" class="settings-routing-list">
                  <div v-for="item in routingDiagnostics" :key="item.category" class="settings-routing-item">
                    <div class="settings-routing-title">{{ routingDiagnosticTitle(item) }}</div>
                    <div class="settings-tun-meta">{{ t('settings.dnsPath') }}: {{ item.dnsPath || '-' }}</div>
                    <div class="settings-tun-meta">{{ t('settings.resolver') }}: {{ item.resolver || '-' }}</div>
                    <div class="settings-tun-meta">{{ t('settings.routeLabel') }}: {{ item.route || '-' }}</div>
                    <div class="settings-tun-meta">{{ t('settings.decisionReason') }}: {{ item.reason || '-' }}</div>
                    <div v-if="item.domains?.length" class="settings-tun-meta">{{ item.domains.join(', ') }}</div>
                  </div>
                </div>
                <n-empty v-else :description="t('settings.noRoutingDiagnostics')" />
              </n-card>

              <n-alert v-if="tunDiagnostics.length" type="warning" :title="t('settings.tunDiagnostics')">
                <div v-for="item in tunDiagnostics" :key="item">{{ item }}</div>
              </n-alert>

              <n-card v-if="tunStatus.lastOutput" size="small" embedded>
                <pre style="margin: 0; white-space: pre-wrap; word-break: break-word">{{ tunStatus.lastOutput }}</pre>
              </n-card>
            </n-space>
          </n-card>
        </n-space>
      </n-tab-pane>

      <!-- Log -->
      <n-tab-pane name="log" :tab="t('settings.log')">
        <n-card size="small">
          <n-space vertical :size="12">
            <n-descriptions bordered :column="1">
              <n-descriptions-item v-if="logConfig.access" label="Access Log">{{ logConfig.access }}</n-descriptions-item>
              <n-descriptions-item v-if="logConfig.error" label="Error Log">{{ logConfig.error }}</n-descriptions-item>
              <n-descriptions-item :label="t('settings.logLevel')">{{ logConfig.loglevel || 'warning' }}</n-descriptions-item>
              <n-descriptions-item label="DNS Log">{{ logConfig.dnsLog ? 'Enabled' : 'Disabled' }}</n-descriptions-item>
            </n-descriptions>
            <n-button type="primary" @click="handleRestartLogger" :loading="restarting">
              {{ t('settings.restartLogger') }}
            </n-button>
          </n-space>
        </n-card>
      </n-tab-pane>

      <!-- Policy -->
      <n-tab-pane name="policy" :tab="t('settings.policy')">
        <n-card size="small">
          <template v-if="policyConfig">
            <pre style="font-size: 13px">{{ JSON.stringify(policyConfig, null, 2) }}</pre>
          </template>
          <n-empty v-else description="No policy configured" />
        </n-card>
      </n-tab-pane>

      <!-- Observatory -->
      <n-tab-pane name="observatory" :tab="t('settings.observatory')">
        <n-card size="small">
          <n-space vertical :size="12">
            <n-button @click="fetchObservatory" :loading="loadingObs">{{ t('common.refresh') }}</n-button>
            <n-data-table :columns="obsColumns" :data="obsStatus" size="small" />
          </n-space>
        </n-card>
      </n-tab-pane>

      <!-- API -->
      <n-tab-pane name="api" :tab="t('settings.api')">
        <n-card size="small">
          <template v-if="apiConfig">
            <n-descriptions bordered :column="1">
              <n-descriptions-item label="Tag">{{ apiConfig.tag }}</n-descriptions-item>
              <n-descriptions-item label="Listen">{{ apiConfig.listen || '-' }}</n-descriptions-item>
              <n-descriptions-item label="Services">{{ apiConfig.services?.join(', ') || '-' }}</n-descriptions-item>
            </n-descriptions>
          </template>
          <n-empty v-else description="No API configured" />
        </n-card>
      </n-tab-pane>
    </n-tabs>
  </n-space>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NSpace, NTabs, NTabPane, NCard, NDescriptions, NDescriptionsItem, NButton, NDataTable, NEmpty, NAlert, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { configAPI, loggerAPI, observatoryAPI, tunAPI } from '@/api/client'
import type { MachineState, MachineStateReason, TunEgressObservation, TunRoutingDiagnostic, TunStatusResponse } from '@/api/types'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()

const activeTab = ref('tun')
const logConfig = ref<any>({})
const policyConfig = ref<any>(null)
const apiConfig = ref<any>(null)
const obsStatus = ref<any[]>([])
const loadingObs = ref(false)
const restarting = ref(false)
const loadingTun = ref(false)
const installingTunBootstrap = ref(false)
const tunStatus = ref<TunStatusResponse>({
  status: 'unknown',
  running: false,
  available: false,
  allowRemote: false,
  useSudo: true,
  helperExists: false,
  elevationReady: false,
  helperCurrent: true,
  binaryCurrent: true,
  privilegeInstallRecommended: false,
  binaryPath: '',
  helperPath: '',
  stateDir: '',
  runtimeConfigPath: '',
  interfaceName: '',
  mtu: 0,
  remoteDns: [],
  configPath: '',
  xrayBinary: '',
  message: '',
  lastOutput: '',
  diagnostics: [],
  routingDiagnostics: []
})

const obsColumns: DataTableColumns = [
  { title: 'Outbound', key: 'outboundTag' },
  { title: 'Alive', key: 'alive', render: (row: any) => row.alive ? 'Yes' : 'No' },
  { title: 'Delay (ms)', key: 'delay' },
  { title: 'Error', key: 'lastErrorReason', ellipsis: { tooltip: true } }
]

const tunStatusText = computed(() => {
  if (tunStatus.value.running) return t('common.enabled')
  if (tunStatus.value.status === 'error' || tunStatus.value.status === 'unavailable') return t('common.error')
  return t('common.disabled')
})

const tunStatusType = computed(() => {
  if (tunStatus.value.running) return 'success'
  if (tunStatus.value.status === 'blocked') return 'warning'
  if (tunStatus.value.status === 'error' || tunStatus.value.status === 'unavailable') return 'error'
  return 'warning'
})

const tunDiagnostics = computed(() => Array.isArray(tunStatus.value.diagnostics) ? tunStatus.value.diagnostics : [])
const routingDiagnostics = computed<TunRoutingDiagnostic[]>(() => Array.isArray(tunStatus.value.routingDiagnostics) ? tunStatus.value.routingDiagnostics : [])
const tunBootstrapNeeded = computed(() => Boolean(tunStatus.value.privilegeInstallRecommended))
const tunRepairRecommended = computed(() => tunStatus.value.helperCurrent === false || tunStatus.value.binaryCurrent === false)
const machineStateText = computed(() => translateCode('nodePool.machineStateLabel', (tunStatus.value.machineState || 'clean') as MachineState))
const machineReasonText = computed(() => translateCode('nodePool.reason', (tunStatus.value.lastStateReason || 'startup_default_clean') as MachineStateReason))

function tunEgressHeadline(observation?: TunEgressObservation) {
  if (!observation) return '-'
  const parts = [observation.status, observation.route]
  if (observation.ip) {
    parts.push(observation.ip)
  }
  return parts.filter(Boolean).join(' · ')
}

function tunEgressMeta(observation?: TunEgressObservation) {
  if (!observation) return ''
  const parts = [
    observation.source ? `${t('settings.source')}: ${observation.source}` : '',
    observation.checkedAt ? `${t('settings.checkedAt')}: ${new Date(observation.checkedAt).toLocaleString()}` : '',
    observation.note ? `${t('settings.note')}: ${observation.note}` : '',
    observation.error ? `${t('settings.lastError')}: ${observation.error}` : ''
  ]
  return parts.filter(Boolean).join(' · ')
}

function routingDiagnosticTitle(item: TunRoutingDiagnostic) {
  return `${t('settings.diagnosticCategory')}: ${item.category}`
}

async function loadConfig() {
  try {
    const data = await configAPI.get()
    if (data.config) {
      const config = typeof data.config === 'string' ? JSON.parse(data.config) : data.config
      logConfig.value = config.log || {}
      policyConfig.value = config.policy || null
      apiConfig.value = config.api || null
    }
  } catch { /* ignore */ }
}

async function fetchObservatory() {
  loadingObs.value = true
  try {
    const data = await observatoryAPI.getStatus()
    obsStatus.value = data.status || []
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    loadingObs.value = false
  }
}

async function handleRestartLogger() {
  restarting.value = true
  try {
    await loggerAPI.restart()
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    restarting.value = false
  }
}

function applyTunStatus(data: Partial<TunStatusResponse>) {
  tunStatus.value = {
    ...tunStatus.value,
    ...data,
    diagnostics: Array.isArray(data?.diagnostics) ? data.diagnostics : [],
    routingDiagnostics: Array.isArray(data?.routingDiagnostics) ? data.routingDiagnostics : []
  }
}

async function fetchTunStatus() {
  loadingTun.value = true
  try {
    const data = await tunAPI.status()
    applyTunStatus(data)
  } catch (err: any) {
    applyTunStatus({
      status: 'error',
      running: false,
      available: false,
      message: err?.message || err?.error || t('common.error'),
      lastOutput: err?.lastOutput || '',
      diagnostics: err?.diagnostics || []
    })
  } finally {
    loadingTun.value = false
  }
}

async function handleInstallTunBootstrap() {
  installingTunBootstrap.value = true
  try {
    const data = await tunAPI.installPrivilege()
    applyTunStatus(data)
    message.success(data.message || t('common.success'))
  } catch (err: any) {
    if (err?.status) {
      applyTunStatus(err)
    }
    message.error(err?.message || err?.error || t('common.error'))
    await fetchTunStatus()
  } finally {
    installingTunBootstrap.value = false
  }
}

function translateCode(prefix: string, code: string) {
  return t(`${prefix}.${code}`)
}

function goToNodePool() {
  router.push('/node-pool')
}

onMounted(() => {
  loadConfig()
  fetchObservatory()
  fetchTunStatus()
})
</script>

<style scoped>
.settings-routing-list {
  display: grid;
  gap: 12px;
}

.settings-routing-item {
  border: 1px solid rgba(128, 128, 128, 0.2);
  border-radius: 8px;
  padding: 12px;
}

.settings-routing-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.settings-tun-meta {
  font-size: 13px;
  color: var(--n-text-color-2);
  margin-top: 8px;
  word-break: break-word;
}
</style>
