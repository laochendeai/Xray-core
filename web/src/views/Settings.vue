<template>
  <n-space vertical :size="16">
    <h2>{{ t('settings.title') }}</h2>

    <n-tabs v-model:value="activeTab" type="line">
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
import { ref, onMounted } from 'vue'
import { NSpace, NTabs, NTabPane, NCard, NDescriptions, NDescriptionsItem, NButton, NDataTable, NEmpty, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { configAPI, loggerAPI, observatoryAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const activeTab = ref('log')
const logConfig = ref<any>({})
const policyConfig = ref<any>(null)
const apiConfig = ref<any>(null)
const obsStatus = ref<any[]>([])
const loadingObs = ref(false)
const restarting = ref(false)

const obsColumns: DataTableColumns = [
  { title: 'Outbound', key: 'outboundTag' },
  { title: 'Alive', key: 'alive', render: (row: any) => row.alive ? 'Yes' : 'No' },
  { title: 'Delay (ms)', key: 'delay' },
  { title: 'Error', key: 'lastErrorReason', ellipsis: { tooltip: true } }
]

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

onMounted(() => {
  loadConfig()
  fetchObservatory()
})
</script>
