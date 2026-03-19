<template>
  <n-space vertical :size="16">
    <h2>{{ t('dns.title') }}</h2>
    <n-alert type="info">
      DNS configuration is managed through the config file. Edit the "dns" section in the Config page.
    </n-alert>

    <n-card :title="t('dns.title')" size="small">
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
      <n-empty v-else description="DNS config not loaded. Check Config page." />
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NSpace, NCard, NAlert, NDescriptions, NDescriptionsItem, NDivider, NDataTable, NEmpty, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { configAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const dnsConfig = ref<any>(null)

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

const serverColumns: DataTableColumns = [
  { title: '#', key: 'index', width: 40 },
  { title: 'Address', key: 'address' },
  { title: 'Domains', key: 'domains', render: (row: any) => row.domains ? row.domains.join(', ') : '-' },
  { title: t('dns.queryStrategy'), key: 'queryStrategy', render: (row: any) => row.queryStrategy || '-' }
]

const hostColumns: DataTableColumns = [
  { title: 'Domain', key: 'domain' },
  { title: 'Target', key: 'target' }
]

async function loadDNSConfig() {
  try {
    const data = await configAPI.get()
    if (data.config) {
      const config = typeof data.config === 'string' ? JSON.parse(data.config) : data.config
      dnsConfig.value = config.dns || null
    }
  } catch (err: any) {
    message.error('Failed to load DNS config')
  }
}

onMounted(loadDNSConfig)
</script>
