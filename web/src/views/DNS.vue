<template>
  <n-space vertical :size="16" class="dns-page">
    <h2>{{ t('dns.title') }}</h2>
    <n-alert type="info" :title="t('dns.runtimeSplitTitle')">
      {{ t('dns.runtimeSplitBody') }}
    </n-alert>
    <n-alert type="info">
      {{ t('dns.remoteDnsControlHint') }}
    </n-alert>

    <n-card :title="t('dns.runtimeSplitTitle')" size="small">
      <div class="dns-flow-grid">
        <div class="dns-flow-item">
          <strong>{{ t('dns.flowCnTitle') }}</strong>
          <div class="dns-flow-desc">{{ t('dns.flowCnBody') }}</div>
          <div class="dns-flow-meta">{{ t('dns.resolversLabel') }}: {{ chinaResolversDisplay }}</div>
        </div>
        <div class="dns-flow-item">
          <strong>{{ t('dns.flowRemoteTitle') }}</strong>
          <div class="dns-flow-desc">{{ t('dns.flowRemoteBody') }}</div>
          <div class="dns-flow-meta">{{ t('dns.resolversLabel') }}: {{ remoteResolversDisplay }}</div>
        </div>
        <div class="dns-flow-item">
          <strong>{{ t('dns.flowBaseConfigTitle') }}</strong>
          <div class="dns-flow-desc">{{ t('dns.flowBaseConfigBody') }}</div>
          <div class="dns-flow-meta">
            {{ tunStatus?.running ? t('dns.runtimeStatusRunning') : t('dns.runtimeStatusStopped') }}
          </div>
        </div>
      </div>
    </n-card>

    <n-card :title="t('dns.baseConfigTitle')" size="small">
      <n-alert type="warning" class="dns-config-alert">
        {{ t('dns.baseConfigHint') }}
      </n-alert>
      <template v-if="dnsConfig">
        <n-descriptions bordered :column="1">
          <n-descriptions-item v-if="dnsConfig.clientIp" :label="t('dns.clientIp')">
            {{ dnsConfig.clientIp }}
          </n-descriptions-item>
          <n-descriptions-item v-if="dnsConfig.queryStrategy" :label="t('dns.queryStrategy')">
            {{ dnsConfig.queryStrategy }}
          </n-descriptions-item>
          <n-descriptions-item v-if="dnsConfig.tag" label="Tag">
            {{ dnsConfig.tag }}
          </n-descriptions-item>
        </n-descriptions>

        <n-divider>{{ t('dns.servers') }}</n-divider>
        <n-data-table :columns="serverColumns" :data="dnsServers" size="small" />

        <template v-if="dnsHosts && Object.keys(dnsHosts).length > 0">
          <n-divider>{{ t('dns.hosts') }}</n-divider>
          <n-data-table :columns="hostColumns" :data="hostEntries" size="small" />
        </template>
      </template>
      <n-empty v-else :description="t('dns.notLoaded')" />
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NSpace, NCard, NAlert, NDescriptions, NDescriptionsItem, NDivider, NDataTable, NEmpty, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { configAPI, tunAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const dnsConfig = ref<any>(null)
const tunStatus = ref<any>(null)
const defaultChinaResolvers = ['223.5.5.5', '119.29.29.29']
const defaultRemoteResolvers = ['1.1.1.1', '8.8.8.8']

const dnsServers = computed(() => {
  if (!dnsConfig.value?.servers) return []
  return dnsConfig.value.servers.map((s: any, i: number) => {
    if (typeof s === 'string') return { index: i, address: s }
    return { index: i, ...s }
  })
})

const dnsHosts = computed(() => dnsConfig.value?.hosts || {})

const hostEntries = computed(() => {
  return Object.entries(dnsHosts.value).map(([domain, value]) => ({
    domain,
    target: Array.isArray(value) ? value.join(', ') : String(value)
  }))
})

const chinaResolversDisplay = computed(() => defaultChinaResolvers.join(', '))
const remoteResolversDisplay = computed(() => {
  const resolvers = Array.isArray(tunStatus.value?.remoteDns) && tunStatus.value.remoteDns.length
    ? tunStatus.value.remoteDns
    : defaultRemoteResolvers
  return resolvers.join(', ')
})

const serverColumns: DataTableColumns = [
  { title: '#', key: 'index', width: 40 },
  { title: t('dns.address'), key: 'address' },
  { title: t('dns.domains'), key: 'domains', render: (row: any) => row.domains ? row.domains.join(', ') : '-' },
  { title: t('dns.queryStrategy'), key: 'queryStrategy', render: (row: any) => row.queryStrategy || '-' }
]

const hostColumns: DataTableColumns = [
  { title: t('dns.domain'), key: 'domain' },
  { title: t('dns.target'), key: 'target' }
]

async function loadDNSConfig() {
  try {
    const data = await configAPI.get()
    if (data.config) {
      const config = typeof data.config === 'string' ? JSON.parse(data.config) : data.config
      dnsConfig.value = config.dns || null
    }
  } catch (err: any) {
    message.error(t('dns.loadFailed'))
  }
}

async function loadTunStatus() {
  try {
    tunStatus.value = await tunAPI.status()
  } catch {
    tunStatus.value = null
  }
}

onMounted(() => {
  loadDNSConfig()
  loadTunStatus()
})
</script>

<style scoped>
.dns-page h2 {
  margin: 0;
}

.dns-flow-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.dns-flow-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border: 1px solid var(--n-border-color);
  border-radius: 10px;
  background: var(--n-color);
}

.dns-flow-desc {
  color: var(--n-text-color-2);
  line-height: 1.6;
}

.dns-flow-meta {
  color: var(--n-text-color-3);
  font-size: 13px;
}

.dns-config-alert {
  margin-bottom: 12px;
}

@media (max-width: 900px) {
  .dns-flow-grid {
    grid-template-columns: 1fr;
  }
}
</style>
