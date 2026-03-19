<template>
  <n-space vertical :size="16">
    <h2>{{ t('monitor.title') }}</h2>

    <n-tabs v-model:value="activeTab" type="line">
      <!-- Traffic Tab -->
      <n-tab-pane name="traffic" :tab="t('monitor.traffic')">
        <n-card size="small">
          <v-chart :option="trafficChartOption" autoresize style="height: 300px" />
        </n-card>
      </n-tab-pane>

      <!-- Connections Tab -->
      <n-tab-pane name="connections" :tab="t('monitor.connections')">
        <n-card size="small">
          <n-space justify="end" style="margin-bottom: 12px">
            <n-badge :value="isWsConnected ? 'Connected' : 'Disconnected'" :type="isWsConnected ? 'success' : 'error'" />
          </n-space>
          <n-data-table :columns="connectionColumns" :data="connections" size="small" :max-height="400" virtual-scroll />
        </n-card>
      </n-tab-pane>

      <!-- System Tab -->
      <n-tab-pane name="system" :tab="t('monitor.system')">
        <n-grid :cols="2" :x-gap="16" :y-gap="16">
          <n-gi>
            <n-card title="Goroutines" size="small">
              <v-chart :option="goroutineChartOption" autoresize style="height: 200px" />
            </n-card>
          </n-gi>
          <n-gi>
            <n-card :title="t('dashboard.memory')" size="small">
              <v-chart :option="memoryChartOption" autoresize style="height: 200px" />
            </n-card>
          </n-gi>
        </n-grid>
      </n-tab-pane>
    </n-tabs>
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { NSpace, NTabs, NTabPane, NCard, NGrid, NGi, NDataTable, NBadge, type DataTableColumns } from 'naive-ui'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { useI18n } from 'vue-i18n'
import { useWebSocket } from '@/composables/useWebSocket'
import { useStatsStore } from '@/stores/stats'
import { formatBytes } from '@/utils/format'

use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent])

const { t } = useI18n()
const statsStore = useStatsStore()

const activeTab = ref('traffic')

// WebSocket for traffic
const { data: trafficData, isConnected: isWsConnected } = useWebSocket('/api/v1/ws/traffic')

// WebSocket for connections
const { data: routingData } = useWebSocket('/api/v1/ws/routing-stats')

const connections = ref<any[]>([])
const trafficHistory = ref<{ time: string; up: number; down: number }[]>([])
const sysHistory = ref<{ time: string; goroutines: number; memory: number }[]>([])

let sysTimer: ReturnType<typeof setInterval> | null = null

watch(trafficData, (val) => {
  if (!val?.stats) return
  let up = 0, down = 0
  for (const [name, stat] of Object.entries(val.stats) as any[]) {
    if (name.includes('uplink')) up += stat.value
    if (name.includes('downlink')) down += stat.value
  }
  const time = new Date().toLocaleTimeString()
  trafficHistory.value.push({ time, up, down })
  if (trafficHistory.value.length > 60) {
    trafficHistory.value = trafficHistory.value.slice(-60)
  }
})

watch(routingData, (val) => {
  if (!val) return
  connections.value.unshift({
    ...val,
    time: new Date().toLocaleTimeString()
  })
  if (connections.value.length > 200) {
    connections.value = connections.value.slice(0, 200)
  }
})

async function fetchSysStats() {
  try {
    await statsStore.fetchSysStats()
    if (statsStore.sysStats) {
      const time = new Date().toLocaleTimeString()
      sysHistory.value.push({
        time,
        goroutines: statsStore.sysStats.numGoroutine,
        memory: statsStore.sysStats.alloc
      })
      if (sysHistory.value.length > 60) {
        sysHistory.value = sysHistory.value.slice(-60)
      }
    }
  } catch { /* ignore */ }
}

const trafficChartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['Upload', 'Download'] },
  grid: { left: '3%', right: '3%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: trafficHistory.value.map(h => h.time) },
  yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatBytes(v) } },
  series: [
    { name: 'Upload', type: 'line', smooth: true, data: trafficHistory.value.map(h => h.up), areaStyle: {} },
    { name: 'Download', type: 'line', smooth: true, data: trafficHistory.value.map(h => h.down), areaStyle: {} }
  ]
}))

const goroutineChartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '3%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: sysHistory.value.map(h => h.time) },
  yAxis: { type: 'value' },
  series: [{ name: 'Goroutines', type: 'line', smooth: true, data: sysHistory.value.map(h => h.goroutines) }]
}))

const memoryChartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '3%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: sysHistory.value.map(h => h.time) },
  yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatBytes(v) } },
  series: [{ name: 'Memory', type: 'line', smooth: true, data: sysHistory.value.map(h => h.memory), areaStyle: {} }]
}))

const connectionColumns: DataTableColumns = [
  { title: 'Time', key: 'time', width: 100 },
  { title: 'Inbound', key: 'inboundTag', width: 120 },
  { title: 'Outbound', key: 'outboundTag', width: 120 },
  { title: t('routing.network'), key: 'network', width: 80 },
  { title: 'Target', key: 'targetDomain', width: 200 },
  { title: 'User', key: 'user', width: 150 }
]

onMounted(() => {
  fetchSysStats()
  sysTimer = setInterval(fetchSysStats, 2000)
})

onUnmounted(() => {
  if (sysTimer) clearInterval(sysTimer)
})
</script>
