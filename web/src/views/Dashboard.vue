<template>
  <n-space vertical :size="24">
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

    <n-card :title="t('dashboard.readinessTitle')" size="small">
      <template #header-extra>
        <n-space align="center" :size="12">
          <n-tag :type="readinessOverview.type" size="small">
            {{ readinessOverview.badgeLabel }}
          </n-tag>
          <n-button size="small" tertiary @click="router.push('/readiness')">
            {{ t('dashboard.openReadiness') }}
          </n-button>
        </n-space>
      </template>

      <n-space vertical :size="12">
        <n-alert :type="readinessOverview.type">
          {{ readinessOverview.description }}
        </n-alert>

        <n-grid :cols="3" :x-gap="12" responsive="screen" item-responsive>
          <n-gi span="3 m:1">
            <n-statistic :label="t('readiness.cards.blocking')">
              <template #default>{{ readiness?.blockingCount ?? '-' }}</template>
            </n-statistic>
          </n-gi>
          <n-gi span="3 m:1">
            <n-statistic :label="t('readiness.cards.warning')">
              <template #default>{{ readiness?.warningCount ?? '-' }}</template>
            </n-statistic>
          </n-gi>
          <n-gi span="3 m:1">
            <n-statistic :label="t('readiness.cards.checks')">
              <template #default>{{ readiness?.checks.length ?? '-' }}</template>
            </n-statistic>
          </n-gi>
        </n-grid>
      </n-space>
    </n-card>

    <n-card :title="t('dashboard.updateStatusTitle')" size="small">
      <template #header-extra>
        <n-space align="center" :size="12">
          <n-tag :type="updateBadge.type" size="small">
            {{ updateBadge.label }}
          </n-tag>
          <n-button size="small" @click="fetchUpdateStatus(true)" :loading="loadingUpdate">
            {{ t('dashboard.checkUpdates') }}
          </n-button>
        </n-space>
      </template>

      <n-space vertical :size="12">
        <n-alert type="info">
          {{ t('dashboard.updateHint') }}
        </n-alert>

        <n-descriptions bordered :column="1" size="small">
          <n-descriptions-item :label="t('dashboard.currentVersion')">
            {{ updateStatus?.currentVersion || '-' }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('dashboard.latestVersion')">
            {{ updateStatus?.latestVersion || '-' }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('dashboard.releaseSource')">
            {{ updateStatus?.source || '-' }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('dashboard.releaseDate')">
            {{ formatDateTime(updateStatus?.latestPublishedAt) }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('dashboard.lastChecked')">
            {{ formatDateTime(updateStatus?.checkedAt) }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('dashboard.releasePage')">
            <a
              v-if="updateStatus?.latestReleaseUrl"
              :href="updateStatus.latestReleaseUrl"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ t('dashboard.openReleasePage') }}
            </a>
            <span v-else>-</span>
          </n-descriptions-item>
        </n-descriptions>

        <n-alert v-if="updateStatus?.message" :type="updateStatus.status === 'error' ? 'error' : 'warning'">
          {{ updateStatus.message }}
        </n-alert>
      </n-space>
    </n-card>

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
import { useRouter } from 'vue-router'
import {
  NSpace,
  NGrid,
  NGi,
  NCard,
  NStatistic,
  NDataTable,
  NAlert,
  NButton,
  NDescriptions,
  NDescriptionsItem,
  NTag,
  type DataTableColumns
} from 'naive-ui'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { useI18n } from 'vue-i18n'
import { readinessAPI, statsAPI } from '@/api/client'
import type { ReadinessResponse, UpdateStatusResponse } from '@/api/types'
import { useStatsStore } from '@/stores/stats'
import { describeReadinessOverview } from '@/utils/readiness'
import { formatBytes, formatUptime } from '@/utils/format'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const router = useRouter()
const { t } = useI18n()
const statsStore = useStatsStore()

const sysStats = computed(() => statsStore.sysStats)

const totalUplink = ref(0)
const totalDownlink = ref(0)
const trafficHistory = ref<{ time: string; up: number; down: number }[]>([])
const topUsers = ref<any[]>([])
const updateStatus = ref<UpdateStatusResponse | null>(null)
const readiness = ref<ReadinessResponse | null>(null)
const loadingUpdate = ref(false)

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

async function fetchReadiness() {
  try {
    readiness.value = await readinessAPI.get()
  } catch {
    readiness.value = null
  }
}

async function fetchUpdateStatus(force = false) {
  loadingUpdate.value = true
  try {
    updateStatus.value = await statsAPI.getUpdateStatus(force)
  } catch (err: any) {
    updateStatus.value = {
      currentVersion: updateStatus.value?.currentVersion || '-',
      latestVersion: updateStatus.value?.latestVersion,
      releaseTitle: updateStatus.value?.releaseTitle,
      latestReleaseUrl: updateStatus.value?.latestReleaseUrl,
      latestPublishedAt: updateStatus.value?.latestPublishedAt,
      checkedAt: updateStatus.value?.checkedAt,
      source: updateStatus.value?.source || 'XTLS/Xray-core',
      status: 'error',
      message: err?.error || t('dashboard.updateCheckUnavailable'),
      updateAvailable: false,
      stale: Boolean(updateStatus.value)
    }
  } finally {
    loadingUpdate.value = false
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

const updateBadge = computed(() => {
  if (loadingUpdate.value) {
    return { type: 'info' as const, label: t('common.loading') }
  }
  if (!updateStatus.value) {
    return { type: 'default' as const, label: '-' }
  }
  if (updateStatus.value.status === 'error') {
    return { type: 'error' as const, label: t('dashboard.updateCheckUnavailable') }
  }
  if (updateStatus.value.status === 'stale') {
    return { type: 'warning' as const, label: t('dashboard.updateStatusStale') }
  }
  if (updateStatus.value.updateAvailable) {
    return { type: 'warning' as const, label: t('dashboard.updateAvailable') }
  }
  return { type: 'success' as const, label: t('dashboard.upToDate') }
})

const readinessOverview = computed(() => describeReadinessOverview(t, readiness.value))

function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

onMounted(() => {
  fetchData()
  fetchReadiness()
  fetchUpdateStatus()
  pollTimer = setInterval(fetchData, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>
