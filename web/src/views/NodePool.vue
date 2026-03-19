<template>
  <n-space vertical :size="16">
    <n-space justify="space-between" align="center">
      <h2>{{ t('nodePool.title') }}</h2>
      <n-button @click="fetchNodes">{{ t('common.refresh') }}</n-button>
    </n-space>

    <n-tabs v-model:value="activeTab" type="line" @update:value="handleTabChange">
      <n-tab-pane name="staging" :tab="t('nodePool.staging') + ` (${stagingCount})`">
        <n-data-table :columns="stagingColumns" :data="nodes" :loading="loading" :pagination="{ pageSize: 20 }" />
      </n-tab-pane>
      <n-tab-pane name="active" :tab="t('nodePool.active') + ` (${activeCount})`">
        <n-data-table :columns="activeColumns" :data="nodes" :loading="loading" :pagination="{ pageSize: 20 }" />
      </n-tab-pane>
      <n-tab-pane name="demoted" :tab="t('nodePool.demoted') + ` (${demotedCount})`">
        <n-data-table :columns="demotedColumns" :data="nodes" :loading="loading" :pagination="{ pageSize: 20 }" />
      </n-tab-pane>
    </n-tabs>

    <!-- Validation Config Panel -->
    <n-collapse>
      <n-collapse-item :title="t('nodePool.validationConfig')">
        <n-form :model="configForm" label-placement="left" label-width="200px">
          <n-form-item :label="t('nodePool.minSamples')">
            <n-input-number v-model:value="configForm.minSamples" :min="1" :max="100" />
          </n-form-item>
          <n-form-item :label="t('nodePool.maxFailRate')">
            <n-input-number v-model:value="configForm.maxFailRate" :min="0" :max="1" :step="0.05" />
          </n-form-item>
          <n-form-item :label="t('nodePool.maxAvgDelay')">
            <n-input-number v-model:value="configForm.maxAvgDelayMs" :min="100" :max="10000" :step="100" />
          </n-form-item>
          <n-form-item :label="t('nodePool.probeInterval')">
            <n-input-number v-model:value="configForm.probeIntervalSec" :min="10" :max="3600" />
          </n-form-item>
          <n-form-item :label="t('nodePool.probeUrl')">
            <n-input v-model:value="configForm.probeUrl" />
          </n-form-item>
          <n-form-item :label="t('nodePool.demoteAfterFails')">
            <n-input-number v-model:value="configForm.demoteAfterFails" :min="1" :max="50" />
          </n-form-item>
          <n-form-item :label="t('nodePool.autoRemoveDemoted')">
            <n-switch v-model:value="configForm.autoRemoveDemoted" />
          </n-form-item>
          <n-form-item>
            <n-button type="primary" :loading="savingConfig" @click="handleSaveConfig">{{ t('nodePool.saveConfig') }}</n-button>
          </n-form-item>
        </n-form>
      </n-collapse-item>
    </n-collapse>
  </n-space>
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import {
  NSpace, NButton, NDataTable, NTabs, NTabPane, NCollapse, NCollapseItem,
  NForm, NFormItem, NInput, NInputNumber, NSwitch, NPopconfirm, NTag,
  useMessage, type DataTableColumns
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { nodePoolAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const activeTab = ref('staging')
const nodes = ref<any[]>([])
const allNodes = ref<any[]>([])
const loading = ref(false)
const savingConfig = ref(false)

const configForm = ref({
  minSamples: 10,
  maxFailRate: 0.3,
  maxAvgDelayMs: 1000,
  probeIntervalSec: 60,
  probeUrl: 'https://www.gstatic.com/generate_204',
  demoteAfterFails: 5,
  autoRemoveDemoted: false
})

const stagingCount = computed(() => allNodes.value.filter(n => n.status === 'staging').length)
const activeCount = computed(() => allNodes.value.filter(n => n.status === 'active').length)
const demotedCount = computed(() => allNodes.value.filter(n => n.status === 'demoted').length)

function formatFailRate(row: any): string {
  if (row.totalPings === 0) return '-'
  return (row.failedPings / row.totalPings * 100).toFixed(1) + '%'
}

function delayTag(row: any) {
  if (row.avgDelayMs <= 0) return h(NTag, { size: 'small' }, { default: () => '-' })
  const type = row.avgDelayMs < 300 ? 'success' : row.avgDelayMs < 800 ? 'warning' : 'error'
  return h(NTag, { size: 'small', type }, { default: () => row.avgDelayMs + 'ms' })
}

const baseColumns: DataTableColumns = [
  { title: () => t('subscriptions.remark'), key: 'remark', width: 150, ellipsis: { tooltip: true } },
  { title: () => t('nodePool.protocol'), key: 'protocol', width: 100 },
  {
    title: () => t('nodePool.address'),
    key: 'address',
    width: 200,
    render(row: any) { return `${row.address}:${row.port}` }
  },
  {
    title: () => t('nodePool.avgDelay'),
    key: 'avgDelayMs',
    width: 120,
    render: delayTag
  },
  {
    title: () => t('nodePool.failRate'),
    key: 'failRate',
    width: 100,
    render: (row: any) => formatFailRate(row)
  },
  { title: () => t('nodePool.totalPings'), key: 'totalPings', width: 100 }
]

const stagingColumns = computed<DataTableColumns>(() => [
  ...baseColumns,
  {
    title: () => t('common.actions'),
    key: 'actions',
    width: 200,
    render(row: any) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(NPopconfirm, { onPositiveClick: () => handlePromote(row.id) }, {
            trigger: () => h(NButton, { size: 'small', type: 'success' }, { default: () => t('nodePool.promote') }),
            default: () => t('nodePool.promoteConfirm')
          }),
          h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
            default: () => t('nodePool.deleteConfirm')
          })
        ]
      })
    }
  }
])

const activeColumns = computed<DataTableColumns>(() => [
  ...baseColumns,
  { title: () => t('nodePool.consecutiveFails'), key: 'consecutiveFails', width: 120 },
  {
    title: () => t('common.actions'),
    key: 'actions',
    width: 200,
    render(row: any) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(NPopconfirm, { onPositiveClick: () => handleDemote(row.id) }, {
            trigger: () => h(NButton, { size: 'small', type: 'warning' }, { default: () => t('nodePool.demote') }),
            default: () => t('nodePool.demoteConfirm')
          }),
          h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
            default: () => t('nodePool.deleteConfirm')
          })
        ]
      })
    }
  }
])

const demotedColumns = computed<DataTableColumns>(() => [
  ...baseColumns,
  {
    title: () => t('common.actions'),
    key: 'actions',
    width: 100,
    render(row: any) {
      return h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
        trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
        default: () => t('nodePool.deleteConfirm')
      })
    }
  }
])

async function fetchNodes() {
  loading.value = true
  try {
    const data = await nodePoolAPI.list(activeTab.value)
    nodes.value = data.nodes || []
    // Also fetch all for counts
    const allData = await nodePoolAPI.list()
    allNodes.value = allData.nodes || []
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    loading.value = false
  }
}

async function fetchConfig() {
  try {
    const data = await nodePoolAPI.getConfig()
    configForm.value = { ...configForm.value, ...data }
  } catch (err: any) {
    // use defaults
  }
}

function handleTabChange() {
  fetchNodes()
}

async function handlePromote(id: string) {
  try {
    await nodePoolAPI.promote(id)
    message.success(t('common.success'))
    fetchNodes()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

async function handleDemote(id: string) {
  try {
    await nodePoolAPI.demote(id)
    message.success(t('common.success'))
    fetchNodes()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

async function handleDelete(id: string) {
  try {
    await nodePoolAPI.delete(id)
    message.success(t('common.success'))
    fetchNodes()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

async function handleSaveConfig() {
  savingConfig.value = true
  try {
    await nodePoolAPI.updateConfig(configForm.value)
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    savingConfig.value = false
  }
}

onMounted(() => {
  fetchNodes()
  fetchConfig()
})
</script>
