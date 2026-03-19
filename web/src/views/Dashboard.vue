<template>
  <n-space vertical :size="24">
    <!-- System Info Cards -->
    <n-grid :cols="4" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <n-gi span="4 m:1">
        <n-card size="small">
          <n-statistic :label="t('dashboard.uptime')">
            <template #default>{{ sysStats ? formatUptime(sysStats.uptime) : '-' }}</template>
          </n-statistic>
        </n-card>
      </n-gi>
      <n-gi span="4 m:1">
        <n-card size="small">
          <n-statistic :label="t('dashboard.goroutines')">
            <template #default>{{ sysStats?.numGoroutine ?? '-' }}</template>
          </n-statistic>
        </n-card>
      </n-gi>
      <n-gi span="4 m:1">
        <n-card size="small">
          <n-statistic :label="t('dashboard.memory')">
            <template #default>{{ sysStats ? formatBytes(sysStats.alloc) : '-' }}</template>
          </n-statistic>
        </n-card>
      </n-gi>
      <n-gi span="4 m:1">
        <n-card size="small">
          <n-statistic :label="t('dashboard.onlineUsers')">
            <template #default>{{ statsStore.onlineCount }}</template>
          </n-statistic>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- Traffic Overview -->
    <n-grid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <n-gi span="2 m:1">
        <n-card :title="t('dashboard.trafficOverview')" size="small">
          <n-space vertical>
            <div>{{ t('dashboard.totalUpload') }}: <strong>{{ formatBytes(totalUplink) }}</strong></div>
            <div>{{ t('dashboard.totalDownload') }}: <strong>{{ formatBytes(totalDownlink) }}</strong></div>
          </n-space>
        </n-card>
      </n-gi>
      <n-gi span="2 m:1">
        <n-card :title="t('dashboard.realtimeTraffic')" size="small">
          <v-chart :option="trafficChartOption" autoresize style="height: 200px" />
        </n-card>
      </n-gi>
    </n-grid>

    <!-- Top Users -->
    <n-card :title="t('dashboard.topUsers')" size="small">
      <n-data-table :columns="topUserColumns" :data="topUsers" :pagination="false" size="small" />
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { NSpace, NGrid, NGi, NCard, NStatistic, NDataTable, type DataTableColumns } from 'naive-ui'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { useI18n } from 'vue-i18n'
import { useStatsStore } from '@/stores/stats'
import { formatBytes, formatUptime } from '@/utils/format'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const { t } = useI18n()
const statsStore = useStatsStore()

const sysStats = computed(() => statsStore.sysStats)

const totalUplink = ref(0)
const totalDownlink = ref(0)
const trafficHistory = ref<{ time: string; up: number; down: number }[]>([])
const topUsers = ref<any[]>([])

let pollTimer: ReturnType<typeof setInterval> | null = null

async function fetchData() {
  try {
    await statsStore.fetchSysStats()
    await statsStore.fetchOnlineUsers()
    await statsStore.fetchQueryStats()

    let up = 0, down = 0
    const userMap: Record<string, { email: string; uplink: number; downlink: number }> = {}

    for (const stat of statsStore.queryStats) {
      const parts = stat.name.split('>>>')
      if (parts.length >= 4 && parts[3] === 'uplink') {
        up += stat.value
        if (parts[0] === 'user') {
          const email = parts[1]
          if (!userMap[email]) userMap[email] = { email, uplink: 0, downlink: 0 }
          userMap[email].uplink += stat.value
        }
      }
      if (parts.length >= 4 && parts[3] === 'downlink') {
        down += stat.value
        if (parts[0] === 'user') {
          const email = parts[1]
          if (!userMap[email]) userMap[email] = { email, uplink: 0, downlink: 0 }
          userMap[email].downlink += stat.value
        }
      }
    }

    totalUplink.value = up
    totalDownlink.value = down

    // Update traffic history
    const now = new Date().toLocaleTimeString()
    trafficHistory.value.push({ time: now, up, down })
    if (trafficHistory.value.length > 30) {
      trafficHistory.value = trafficHistory.value.slice(-30)
    }

    // Top 10 users by total traffic
    topUsers.value = Object.values(userMap)
      .sort((a, b) => (b.uplink + b.downlink) - (a.uplink + a.downlink))
      .slice(0, 10)
  } catch {
    // Ignore polling errors
  }
}

const trafficChartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: [t('dashboard.totalUpload'), t('dashboard.totalDownload')] },
  grid: { left: '3%', right: '3%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: trafficHistory.value.map(h => h.time) },
  yAxis: {
    type: 'value',
    axisLabel: { formatter: (v: number) => formatBytes(v) }
  },
  series: [
    { name: t('dashboard.totalUpload'), type: 'line', smooth: true, data: trafficHistory.value.map(h => h.up) },
    { name: t('dashboard.totalDownload'), type: 'line', smooth: true, data: trafficHistory.value.map(h => h.down) }
  ]
}))

const topUserColumns: DataTableColumns = [
  { title: 'Email', key: 'email' },
  { title: t('dashboard.totalUpload'), key: 'uplink', render: (row: any) => formatBytes(row.uplink) },
  { title: t('dashboard.totalDownload'), key: 'downlink', render: (row: any) => formatBytes(row.downlink) }
]

onMounted(() => {
  fetchData()
  pollTimer = setInterval(fetchData, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>
