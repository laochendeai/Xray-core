<template>
  <n-space vertical :size="16" class="node-pool-page">
    <n-space justify="space-between" align="center" wrap>
      <div>
        <h2>{{ t('nodePool.title') }}</h2>
        <p class="node-pool-subtitle">{{ t('nodePool.subtitle') }}</p>
      </div>
      <n-button :loading="loading" @click="refreshAll">{{ t('common.refresh') }}</n-button>
    </n-space>

    <n-alert :type="machineBannerType" :title="machineBannerTitle">
      {{ machineBannerBody }}
    </n-alert>

    <n-card size="small" class="machine-strip">
      <n-space justify="space-between" align="start" wrap>
        <n-space vertical :size="10">
          <n-space align="center" :size="12" wrap>
            <strong>{{ t('nodePool.machineState') }}</strong>
            <n-tag :type="machineStateTagType" size="small">
              {{ machineStateLabel }}
            </n-tag>
            <n-tag :type="summary.healthy ? 'success' : 'warning'" size="small">
              {{ t('nodePool.activePoolCount', { active: summary.activeNodes, minimum: summary.minActiveNodes }) }}
            </n-tag>
          </n-space>

          <div class="node-pool-meta">
            {{ machineReasonLabel }}
            <span v-if="formattedMachineChangedAt">· {{ formattedMachineChangedAt }}</span>
          </div>

          <div class="node-pool-meta" v-if="tunStatus.message">
            {{ tunStatus.message }}
          </div>
        </n-space>

        <n-space :size="12" wrap class="machine-actions">
          <n-button type="primary" :loading="tunUpdating" @click="handleEnableTransparent">
            {{ t('nodePool.enableTransparent') }}
          </n-button>
          <n-button :loading="tunUpdating" @click="handleRestoreClean">
            {{ t('nodePool.restoreClean') }}
          </n-button>
          <n-button type="warning" secondary :loading="installingTunBootstrap" @click="handleInstallTunBootstrap">
            {{ t('settings.installTunPrivilege') }}
          </n-button>
        </n-space>
      </n-space>
    </n-card>

    <n-alert v-if="tunRepairRecommended" type="warning" :title="t('settings.tunRepairTitle')">
      {{ t('settings.tunRepairDesc') }}
    </n-alert>

    <n-alert v-if="tunBootstrapNeeded" type="warning" :title="t('settings.tunPrivilegeTitle')">
      <n-space justify="space-between" align="center" wrap>
        <span>{{ t('settings.tunInstallDesc') }}</span>
        <n-button type="warning" secondary :loading="installingTunBootstrap" @click="handleInstallTunBootstrap">
          {{ t('settings.installTunPrivilege') }}
        </n-button>
      </n-space>
    </n-alert>

    <div class="summary-strip">
      <div class="summary-item">
        <div class="summary-label">{{ t('nodePool.status.active') }}</div>
        <div class="summary-value">{{ summary.activeCount }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">{{ t('nodePool.status.staging') }}</div>
        <div class="summary-value">{{ summary.stagingCount }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">{{ t('nodePool.status.quarantine') }}</div>
        <div class="summary-value">{{ summary.quarantineCount }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">{{ t('nodePool.cleanliness.unknown') }}</div>
        <div class="summary-value">{{ summary.unknownCleanCount }}</div>
      </div>
    </div>

    <section class="section-block">
      <n-space justify="space-between" align="center" wrap>
        <h3>{{ t('nodePool.recentEvents') }}</h3>
        <span class="node-pool-meta" v-if="formattedSummaryEvaluatedAt">
          {{ t('nodePool.lastEvaluatedAt') }}: {{ formattedSummaryEvaluatedAt }}
        </span>
      </n-space>

      <n-empty v-if="!recentEvents.length" :description="t('nodePool.emptyEvents')" />
      <n-list v-else bordered>
        <n-list-item v-for="event in recentEvents" :key="`${event.nodeId}-${event.at}-${event.reason}`">
          <div class="event-row">
            <div class="event-main">
              <n-space align="center" :size="8" wrap>
                <strong>{{ event.remark || event.nodeAddress || event.nodeId }}</strong>
                <n-tag size="small" :type="statusTagType(event.status)">
                  {{ statusLabel(event.status) }}
                </n-tag>
                <n-tag size="small">
                  {{ reasonLabel(event.reason) }}
                </n-tag>
              </n-space>
              <div class="node-pool-meta">{{ event.details || event.nodeAddress || event.nodeId }}</div>
            </div>
            <div class="event-time">{{ formatDateTime(event.at) }}</div>
          </div>
        </n-list-item>
      </n-list>
    </section>

    <section class="section-block">
      <n-space justify="space-between" align="center" wrap>
        <h3>{{ t('nodePool.status.active') }} ({{ activeNodes.length }})</h3>
      </n-space>
      <template v-if="activeNodes.length">
        <n-data-table
          v-if="!isCompact"
          :columns="activeColumns"
          :data="activeNodes"
          :loading="loading"
          :pagination="{ pageSize: 10 }"
        />
        <div v-else class="node-card-list">
          <div v-for="node in activeNodes" :key="node.id" class="node-card">
            <div class="node-card-header">
              <strong>{{ node.remark || node.address }}</strong>
              <n-tag size="small" :type="statusTagType(node.status)">
                {{ statusLabel(node.status) }}
              </n-tag>
            </div>
            <div class="node-card-meta">{{ node.address }}:{{ node.port }}</div>
            <div class="node-card-meta">{{ reasonLabel(node.statusReason) }}</div>
            <n-space :size="8" wrap>
              <n-tag size="small" :type="cleanlinessTagType(node.cleanliness)">
                {{ cleanlinessLabel(node.cleanliness) }}
              </n-tag>
              <n-tag size="small">{{ delayLabel(node) }}</n-tag>
              <n-tag size="small">{{ failRateLabel(node) }}</n-tag>
            </n-space>
            <n-space :size="8" class="node-card-actions">
              <n-button size="small" type="warning" @click="handleQuarantine(node.id)">
                {{ t('nodePool.quarantine') }}
              </n-button>
              <n-button size="small" type="error" @click="handleRemove(node.id)">
                {{ t('nodePool.remove') }}
              </n-button>
            </n-space>
          </div>
        </div>
      </template>
      <n-empty v-else :description="t('nodePool.emptyActive')" />
    </section>

    <section class="section-block">
      <n-space justify="space-between" align="center" wrap>
        <h3>{{ t('nodePool.status.staging') }} ({{ stagingNodes.length }})</h3>
      </n-space>
      <template v-if="stagingNodes.length">
        <n-data-table
          v-if="!isCompact"
          :columns="stagingColumns"
          :data="stagingNodes"
          :loading="loading"
          :pagination="{ pageSize: 10 }"
        />
        <div v-else class="node-card-list">
          <div v-for="node in stagingNodes" :key="node.id" class="node-card">
            <div class="node-card-header">
              <strong>{{ node.remark || node.address }}</strong>
              <n-tag size="small" :type="statusTagType(node.status)">
                {{ statusLabel(node.status) }}
              </n-tag>
            </div>
            <div class="node-card-meta">{{ node.address }}:{{ node.port }}</div>
            <div class="node-card-meta">{{ reasonLabel(node.statusReason) }}</div>
            <n-space :size="8" wrap>
              <n-tag size="small" :type="cleanlinessTagType(node.cleanliness)">
                {{ cleanlinessLabel(node.cleanliness) }}
              </n-tag>
              <n-tag size="small">{{ delayLabel(node) }}</n-tag>
              <n-tag size="small">{{ failRateLabel(node) }}</n-tag>
            </n-space>
            <n-space :size="8" class="node-card-actions">
              <n-button size="small" type="success" @click="handlePromote(node.id)">
                {{ t('nodePool.promote') }}
              </n-button>
              <n-button size="small" type="error" @click="handleRemove(node.id)">
                {{ t('nodePool.remove') }}
              </n-button>
            </n-space>
          </div>
        </div>
      </template>
      <n-empty v-else :description="t('nodePool.emptyStaging')" />
    </section>

    <section class="section-block">
      <n-space justify="space-between" align="center" wrap>
        <h3>{{ t('nodePool.status.quarantine') }} ({{ quarantineNodes.length }})</h3>
        <n-button
          v-if="quarantineNodes.length"
          size="small"
          type="warning"
          :loading="bulkRemoving"
          @click="handleBulkRemoveQuarantine"
        >
          {{ t('nodePool.bulkRemoveUnstable') }}
        </n-button>
      </n-space>
      <template v-if="quarantineNodes.length">
        <n-data-table
          v-if="!isCompact"
          :columns="quarantineColumns"
          :data="quarantineNodes"
          :loading="loading"
          :pagination="{ pageSize: 10 }"
        />
        <div v-else class="node-card-list">
          <div v-for="node in quarantineNodes" :key="node.id" class="node-card">
            <div class="node-card-header">
              <strong>{{ node.remark || node.address }}</strong>
              <n-tag size="small" :type="statusTagType(node.status)">
                {{ statusLabel(node.status) }}
              </n-tag>
            </div>
            <div class="node-card-meta">{{ node.address }}:{{ node.port }}</div>
            <div class="node-card-meta">{{ reasonLabel(node.statusReason) }}</div>
            <n-space :size="8" wrap>
              <n-tag size="small" :type="cleanlinessTagType(node.cleanliness)">
                {{ cleanlinessLabel(node.cleanliness) }}
              </n-tag>
              <n-tag size="small">{{ delayLabel(node) }}</n-tag>
              <n-tag size="small">{{ failRateLabel(node) }}</n-tag>
            </n-space>
            <n-space :size="8" class="node-card-actions">
              <n-button size="small" type="success" @click="handlePromote(node.id)">
                {{ t('nodePool.promote') }}
              </n-button>
              <n-button size="small" type="error" @click="handleRemove(node.id)">
                {{ t('nodePool.remove') }}
              </n-button>
            </n-space>
          </div>
        </div>
      </template>
      <n-empty v-else :description="t('nodePool.emptyQuarantine')" />
    </section>

    <section class="section-block">
      <n-space justify="space-between" align="center" wrap>
        <h3>{{ t('nodePool.status.candidate') }} ({{ candidateNodes.length }})</h3>
      </n-space>
      <template v-if="candidateNodes.length">
        <n-data-table
          v-if="!isCompact"
          :columns="candidateColumns"
          :data="candidateNodes"
          :loading="loading"
          :pagination="{ pageSize: 10 }"
        />
        <div v-else class="node-card-list">
          <div v-for="node in candidateNodes" :key="node.id" class="node-card">
            <div class="node-card-header">
              <strong>{{ node.remark || node.address }}</strong>
              <n-tag size="small" :type="statusTagType(node.status)">
                {{ statusLabel(node.status) }}
              </n-tag>
            </div>
            <div class="node-card-meta">{{ node.address }}:{{ node.port }}</div>
            <div class="node-card-meta">{{ reasonLabel(node.statusReason) }}</div>
            <n-space :size="8" wrap>
              <n-tag size="small" :type="cleanlinessTagType(node.cleanliness)">
                {{ cleanlinessLabel(node.cleanliness) }}
              </n-tag>
              <n-tag size="small">{{ failRateLabel(node) }}</n-tag>
            </n-space>
            <n-space :size="8" class="node-card-actions">
              <n-button size="small" type="error" @click="handleRemove(node.id)">
                {{ t('nodePool.remove') }}
              </n-button>
            </n-space>
          </div>
        </div>
      </template>
      <n-empty v-else :description="t('nodePool.emptyCandidate')" />
    </section>

    <n-collapse>
      <n-collapse-item :title="`${t('nodePool.status.removed')} (${removedNodes.length})`">
        <n-empty v-if="!removedNodes.length" :description="t('nodePool.emptyRemoved')" />
        <n-data-table
          v-else-if="!isCompact"
          :columns="removedColumns"
          :data="removedNodes"
          :loading="loading"
          :pagination="{ pageSize: 10 }"
        />
        <div v-else class="node-card-list">
          <div v-for="node in removedNodes" :key="node.id" class="node-card">
            <div class="node-card-header">
              <strong>{{ node.remark || node.address }}</strong>
              <n-tag size="small" :type="statusTagType(node.status)">
                {{ statusLabel(node.status) }}
              </n-tag>
            </div>
            <div class="node-card-meta">{{ node.address }}:{{ node.port }}</div>
            <div class="node-card-meta">{{ reasonLabel(node.statusReason) }}</div>
          </div>
        </div>
      </n-collapse-item>
    </n-collapse>

    <n-collapse>
      <n-collapse-item :title="t('nodePool.validationConfig')">
        <n-form :model="configForm" label-placement="left" label-width="220px">
          <n-form-item :label="t('nodePool.minActiveNodes')">
            <n-input-number v-model:value="configForm.minActiveNodes" :min="1" :max="20" />
          </n-form-item>
          <n-form-item :label="t('nodePool.minSamples')">
            <n-input-number v-model:value="configForm.minSamples" :min="1" :max="100" />
          </n-form-item>
          <n-form-item :label="t('nodePool.maxFailRate')">
            <n-input-number v-model:value="configForm.maxFailRate" :min="0" :max="1" :step="0.05" />
          </n-form-item>
          <n-form-item :label="t('nodePool.maxAvgDelay')">
            <n-input-number v-model:value="configForm.maxAvgDelayMs" :min="100" :max="10000" :step="100" />
          </n-form-item>
          <n-form-item :label="t('nodePool.demoteAfterFails')">
            <n-input-number v-model:value="configForm.demoteAfterFails" :min="1" :max="50" />
          </n-form-item>
          <n-form-item :label="t('nodePool.probeInterval')">
            <n-input-number v-model:value="configForm.probeIntervalSec" :min="10" :max="3600" />
          </n-form-item>
          <n-form-item :label="t('nodePool.probeUrl')">
            <n-input v-model:value="configForm.probeUrl" />
          </n-form-item>
          <n-form-item :label="t('nodePool.minBandwidthKbps')">
            <n-input-number v-model:value="configForm.minBandwidthKbps" :min="0" :max="1000000" :step="1000" />
          </n-form-item>
          <n-form-item :label="t('nodePool.autoRemoveDemoted')">
            <n-switch v-model:value="configForm.autoRemoveDemoted" />
          </n-form-item>
          <n-form-item>
            <n-button type="primary" :loading="savingConfig" @click="handleSaveConfig">
              {{ t('nodePool.saveConfig') }}
            </n-button>
          </n-form-item>
        </n-form>
      </n-collapse-item>
    </n-collapse>
  </n-space>
</template>

<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NCollapse,
  NCollapseItem,
  NDataTable,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NList,
  NListItem,
  NPopconfirm,
  NSpace,
  NSwitch,
  NTag,
  useMessage,
  type DataTableColumns
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { nodePoolAPI, tunAPI } from '@/api/client'
import type {
  CleanlinessStatus,
  MachineState,
  MachineStateReason,
  NodeEvent,
  NodePoolDashboardResponse,
  NodePoolSummary,
  NodeRecord,
  NodeStatus,
  TransitionReason,
  TunStatusResponse,
  ValidationConfig
} from '@/api/types'

const { t, te } = useI18n()
const message = useMessage()

const loading = ref(false)
const tunUpdating = ref(false)
const installingTunBootstrap = ref(false)
const savingConfig = ref(false)
const bulkRemoving = ref(false)
const isCompact = ref(typeof window !== 'undefined' ? window.innerWidth < 768 : false)

const dashboard = ref<NodePoolDashboardResponse>({
  nodes: [],
  summary: {
    candidateCount: 0,
    stagingCount: 0,
    activeCount: 0,
    quarantineCount: 0,
    removedCount: 0,
    trustedCount: 0,
    suspiciousCount: 0,
    unknownCleanCount: 0,
    activeNodes: 0,
    minActiveNodes: 0,
    healthy: false,
    lastEvaluatedAt: ''
  },
  recentEvents: []
})

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
  message: ''
})

const configForm = ref<ValidationConfig>({
  minSamples: 10,
  maxFailRate: 0.3,
  maxAvgDelayMs: 1000,
  probeIntervalSec: 60,
  probeUrl: 'https://www.gstatic.com/generate_204',
  demoteAfterFails: 5,
  autoRemoveDemoted: false,
  minActiveNodes: 3,
  minBandwidthKbps: 0
})

const nodes = computed(() => dashboard.value.nodes || [])
const summary = computed<NodePoolSummary>(() => dashboard.value.summary)
const recentEvents = computed<NodeEvent[]>(() => dashboard.value.recentEvents || [])
const activeNodes = computed(() => nodes.value.filter((node) => node.status === 'active'))
const stagingNodes = computed(() => nodes.value.filter((node) => node.status === 'staging'))
const quarantineNodes = computed(() => nodes.value.filter((node) => node.status === 'quarantine'))
const candidateNodes = computed(() => nodes.value.filter((node) => node.status === 'candidate'))
const removedNodes = computed(() => nodes.value.filter((node) => node.status === 'removed'))

const machineState = computed<MachineState>(() => tunStatus.value.machineState || 'clean')
const tunBootstrapNeeded = computed(() => Boolean(tunStatus.value.privilegeInstallRecommended))
const tunRepairRecommended = computed(() => tunStatus.value.helperCurrent === false || tunStatus.value.binaryCurrent === false)
const machineStateLabel = computed(() => translateCode('nodePool.machineStateLabel', machineState.value))
const machineReasonLabel = computed(() => translateCode('nodePool.machineReason', tunStatus.value.lastStateReason || 'startup_default_clean'))
const formattedMachineChangedAt = computed(() => formatDateTime(tunStatus.value.lastStateChangedAt))
const formattedSummaryEvaluatedAt = computed(() => formatDateTime(summary.value.lastEvaluatedAt))

const machineStateTagType = computed(() => {
  switch (machineState.value) {
    case 'proxied':
      return 'success'
    case 'degraded':
      return 'error'
    case 'recovering':
      return 'warning'
    default:
      return 'default'
  }
})

const machineBannerType = computed(() => {
  if (machineState.value === 'degraded') return 'error'
  if (!summary.value.healthy || tunStatus.value.status === 'blocked') return 'warning'
  if (machineState.value === 'proxied') return 'success'
  return 'info'
})

const machineBannerTitle = computed(() => {
  if (machineState.value === 'degraded') return t('nodePool.banner.degradedTitle')
  if (!summary.value.healthy || tunStatus.value.status === 'blocked') return t('nodePool.banner.poolWarningTitle')
  if (machineState.value === 'proxied') return t('nodePool.banner.proxiedTitle')
  return t('nodePool.banner.cleanTitle')
})

const machineBannerBody = computed(() => {
  if (machineState.value === 'degraded') {
    return `${machineReasonLabel.value}. ${tunStatus.value.message || ''}`.trim()
  }
  if (!summary.value.healthy || tunStatus.value.status === 'blocked') {
    return `${t('nodePool.activePoolCount', { active: summary.value.activeNodes, minimum: summary.value.minActiveNodes })}. ${machineReasonLabel.value}`
  }
  return tunStatus.value.message || machineReasonLabel.value
})

function translateCode(prefix: string, code: string): string {
  const key = `${prefix}.${code}`
  return te(key) ? t(key) : code
}

function statusLabel(status: NodeStatus) {
  return translateCode('nodePool.status', status)
}

function reasonLabel(reason: TransitionReason | MachineStateReason) {
  return translateCode('nodePool.reason', reason)
}

function cleanlinessLabel(cleanliness: CleanlinessStatus) {
  return translateCode('nodePool.cleanliness', cleanliness)
}

function statusTagType(status: NodeStatus) {
  switch (status) {
    case 'active':
      return 'success'
    case 'quarantine':
      return 'error'
    case 'candidate':
      return 'warning'
    case 'removed':
      return 'default'
    default:
      return 'info'
  }
}

function cleanlinessTagType(cleanliness: CleanlinessStatus) {
  switch (cleanliness) {
    case 'trusted':
      return 'success'
    case 'suspicious':
      return 'error'
    default:
      return 'warning'
  }
}

function failRateLabel(node: NodeRecord) {
  if (!node.totalPings) return t('nodePool.failRateUnknown')
  return `${((node.failedPings / node.totalPings) * 100).toFixed(1)}%`
}

function delayLabel(node: NodeRecord) {
  return node.avgDelayMs > 0 ? `${node.avgDelayMs}ms` : t('nodePool.delayUnknown')
}

function formatDateTime(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString()
}

function syncViewport() {
  isCompact.value = window.innerWidth < 768
}

function createColumns(group: 'candidate' | 'staging' | 'active' | 'quarantine' | 'removed'): DataTableColumns<NodeRecord> {
  const columns: DataTableColumns<NodeRecord> = [
    {
      title: () => t('subscriptions.remark'),
      key: 'remark',
      width: 160,
      ellipsis: { tooltip: true }
    },
    {
      title: () => t('common.status'),
      key: 'status',
      width: 120,
      render: (row) => h(NTag, { size: 'small', type: statusTagType(row.status) }, { default: () => statusLabel(row.status) })
    },
    {
      title: () => t('nodePool.cleanlinessLabel'),
      key: 'cleanliness',
      width: 120,
      render: (row) => h(NTag, { size: 'small', type: cleanlinessTagType(row.cleanliness) }, { default: () => cleanlinessLabel(row.cleanliness) })
    },
    {
      title: () => t('nodePool.address'),
      key: 'address',
      width: 200,
      render: (row) => `${row.address}:${row.port}`
    },
    {
      title: () => t('nodePool.reasonLabel'),
      key: 'statusReason',
      width: 200,
      ellipsis: { tooltip: true },
      render: (row) => reasonLabel(row.statusReason)
    },
    {
      title: () => t('nodePool.avgDelay'),
      key: 'avgDelayMs',
      width: 110,
      render: (row) => delayLabel(row)
    },
    {
      title: () => t('nodePool.failRate'),
      key: 'failRate',
      width: 110,
      render: (row) => failRateLabel(row)
    },
    {
      title: () => t('nodePool.lastCheckedAt'),
      key: 'lastCheckedAt',
      width: 180,
      render: (row) => formatDateTime(row.lastCheckedAt || row.statusUpdatedAt || row.addedAt) || '-'
    }
  ]

  if (group === 'removed') {
    return columns
  }

  columns.push({
    title: () => t('common.actions'),
    key: 'actions',
    width: group === 'candidate' ? 110 : 220,
    render: (row) => {
      const actions: any[] = []

      if (group === 'staging' || group === 'quarantine') {
        actions.push(
          h(
            NPopconfirm,
            { onPositiveClick: () => handlePromote(row.id) },
            {
              trigger: () => h(NButton, { size: 'small', type: 'success' }, { default: () => t('nodePool.promote') }),
              default: () => t('nodePool.promoteConfirm')
            }
          )
        )
      }

      if (group === 'active') {
        actions.push(
          h(
            NPopconfirm,
            { onPositiveClick: () => handleQuarantine(row.id) },
            {
              trigger: () => h(NButton, { size: 'small', type: 'warning' }, { default: () => t('nodePool.quarantine') }),
              default: () => t('nodePool.quarantineConfirm')
            }
          )
        )
      }

      actions.push(
        h(
          NPopconfirm,
          { onPositiveClick: () => handleRemove(row.id) },
          {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('nodePool.remove') }),
            default: () => t('nodePool.removeConfirm')
          }
        )
      )

      return h(NSpace, { size: 'small' }, { default: () => actions })
    }
  })

  return columns
}

const activeColumns = computed(() => createColumns('active'))
const stagingColumns = computed(() => createColumns('staging'))
const quarantineColumns = computed(() => createColumns('quarantine'))
const candidateColumns = computed(() => createColumns('candidate'))
const removedColumns = computed(() => createColumns('removed'))

async function fetchDashboard() {
  const data = await nodePoolAPI.list()
  dashboard.value = data
}

async function fetchConfig() {
  const data = await nodePoolAPI.getConfig()
  configForm.value = { ...configForm.value, ...data }
}

async function fetchTunStatus() {
  applyTunStatus(await tunAPI.status())
}

function applyTunStatus(status: TunStatusResponse) {
  tunStatus.value = {
    ...tunStatus.value,
    ...status,
    diagnostics: Array.isArray(status?.diagnostics) ? status.diagnostics : []
  }
}

async function refreshAll() {
  loading.value = true
  try {
    await Promise.all([fetchDashboard(), fetchConfig(), fetchTunStatus()])
  } catch (err: any) {
    message.error(err?.message || err?.error || t('common.error'))
  } finally {
    loading.value = false
  }
}

async function handleEnableTransparent() {
  tunUpdating.value = true
  try {
    applyTunStatus(await tunAPI.start())
    message.success(tunStatus.value.message || t('common.success'))
  } catch (err: any) {
    if (err?.status) {
      applyTunStatus(err)
    }
    message.error(err?.message || err?.error || t('common.error'))
  } finally {
    tunUpdating.value = false
    await refreshAll()
  }
}

async function handleRestoreClean() {
  tunUpdating.value = true
  try {
    applyTunStatus(await tunAPI.restoreClean())
    message.success(tunStatus.value.message || t('common.success'))
  } catch (err: any) {
    if (err?.status) {
      applyTunStatus(err)
    }
    message.error(err?.message || err?.error || t('common.error'))
  } finally {
    tunUpdating.value = false
    await refreshAll()
  }
}

async function handleInstallTunBootstrap() {
  installingTunBootstrap.value = true
  try {
    applyTunStatus(await tunAPI.installPrivilege())
    message.success(tunStatus.value.message || t('common.success'))
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

async function handlePromote(id: string) {
  try {
    await nodePoolAPI.promote(id)
    message.success(t('common.success'))
    await refreshAll()
  } catch (err: any) {
    message.error(err?.message || err?.error || t('common.error'))
  }
}

async function handleQuarantine(id: string) {
  try {
    await nodePoolAPI.quarantine(id)
    message.success(t('common.success'))
    await refreshAll()
  } catch (err: any) {
    message.error(err?.message || err?.error || t('common.error'))
  }
}

async function handleRemove(id: string) {
  try {
    await nodePoolAPI.remove(id)
    message.success(t('common.success'))
    await refreshAll()
  } catch (err: any) {
    message.error(err?.message || err?.error || t('common.error'))
  }
}

async function handleBulkRemoveQuarantine() {
  bulkRemoving.value = true
  try {
    await nodePoolAPI.bulkRemove({ statuses: ['quarantine'], onlyUnstable: true })
    message.success(t('common.success'))
    await refreshAll()
  } catch (err: any) {
    message.error(err?.message || err?.error || t('common.error'))
  } finally {
    bulkRemoving.value = false
  }
}

async function handleSaveConfig() {
  savingConfig.value = true
  try {
    await nodePoolAPI.updateConfig(configForm.value)
    message.success(t('common.success'))
    await refreshAll()
  } catch (err: any) {
    message.error(err?.message || err?.error || t('common.error'))
  } finally {
    savingConfig.value = false
  }
}

onMounted(() => {
  syncViewport()
  window.addEventListener('resize', syncViewport)
  refreshAll()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncViewport)
})
</script>

<style scoped>
.node-pool-page h2,
.node-pool-page h3 {
  margin: 0;
}

.node-pool-subtitle {
  margin: 6px 0 0;
  color: var(--n-text-color-3);
}

.machine-strip {
  border-left: 4px solid var(--n-color-target, #18a058);
}

.machine-actions {
  justify-content: flex-end;
}

.node-pool-meta {
  color: var(--n-text-color-3);
  font-size: 13px;
}

.summary-strip {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.summary-item {
  padding: 14px 16px;
  border: 1px solid var(--n-border-color);
  border-radius: 10px;
  background: var(--n-color);
}

.summary-label {
  color: var(--n-text-color-3);
  font-size: 12px;
}

.summary-value {
  margin-top: 6px;
  font-size: 24px;
  font-weight: 600;
}

.section-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.event-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.event-main {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.event-time {
  color: var(--n-text-color-3);
  font-size: 12px;
  white-space: nowrap;
}

.node-card-list {
  display: grid;
  gap: 12px;
}

.node-card {
  border: 1px solid var(--n-border-color);
  border-radius: 10px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.node-card-header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.node-card-meta {
  color: var(--n-text-color-3);
  font-size: 13px;
}

.node-card-actions {
  margin-top: 4px;
}

@media (max-width: 1199px) {
  .summary-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .summary-strip {
    grid-template-columns: 1fr;
  }

  .machine-actions {
    width: 100%;
  }

  .machine-actions :deep(button) {
    width: 100%;
  }

  .event-row {
    flex-direction: column;
  }
}
</style>
