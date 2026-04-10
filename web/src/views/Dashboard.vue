<template>
  <div class="page-shell dashboard-page">
    <section class="dashboard-hero">
      <div class="dashboard-hero-copy">
        <p class="dashboard-eyebrow">Xray Panel</p>
        <div class="dashboard-headline">
          <h1 class="dashboard-title">{{ t('nav.dashboard') }}</h1>
          <n-tag :type="readinessOverview.type" round size="small">
            {{ readinessOverview.badgeLabel }}
          </n-tag>
        </div>
        <p class="dashboard-summary">{{ readinessOverview.description }}</p>
        <n-space class="dashboard-actions" :size="12" wrap>
          <n-button secondary @click="router.push('/readiness')">
            {{ t('dashboard.openReadiness') }}
          </n-button>
          <n-button secondary @click="router.push('/support')">
            {{ t('dashboard.supportAuthor') }}
          </n-button>
          <n-button @click="fetchUpdateStatus(true)" :loading="loadingUpdate">
            {{ t('dashboard.checkUpdates') }}
          </n-button>
        </n-space>
      </div>

      <div class="dashboard-metrics">
        <div class="dashboard-metric-card">
          <span class="dashboard-metric-label">{{ t('dashboard.uptime') }}</span>
          <strong class="dashboard-metric-value">{{ sysStats ? formatUptime(sysStats.uptime) : '-' }}</strong>
        </div>
        <div class="dashboard-metric-card">
          <span class="dashboard-metric-label">{{ t('dashboard.goroutines') }}</span>
          <strong class="dashboard-metric-value">{{ sysStats?.numGoroutine ?? '-' }}</strong>
        </div>
        <div class="dashboard-metric-card">
          <span class="dashboard-metric-label">{{ t('dashboard.memory') }}</span>
          <strong class="dashboard-metric-value">{{ sysStats ? formatBytes(sysStats.alloc) : '-' }}</strong>
        </div>
        <div class="dashboard-metric-card">
          <span class="dashboard-metric-label">{{ t('dashboard.onlineUsers') }}</span>
          <strong class="dashboard-metric-value">{{ statsStore.onlineCount }}</strong>
        </div>
      </div>
    </section>

    <n-grid :cols="2" :x-gap="18" :y-gap="18" responsive="screen" item-responsive>
      <n-gi span="2 m:1">
        <n-card :title="t('dashboard.readinessTitle')" size="small" :bordered="false" class="dashboard-panel">
          <template #header-extra>
            <n-tag :type="readinessOverview.type" size="small">
              {{ readinessOverview.badgeLabel }}
            </n-tag>
          </template>

          <n-space vertical :size="14">
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
      </n-gi>

      <n-gi span="2 m:1">
        <n-card :title="t('dashboard.updateStatusTitle')" size="small" :bordered="false" class="dashboard-panel">
          <template #header-extra>
            <n-tag :type="updateBadge.type" size="small">
              {{ updateBadge.label }}
            </n-tag>
          </template>

          <n-space vertical :size="14">
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
      </n-gi>
    </n-grid>

    <n-grid :cols="2" :x-gap="18" :y-gap="18" responsive="screen" item-responsive>
      <n-gi span="2 m:1">
        <n-card :title="t('dashboard.trafficOverview')" size="small" :bordered="false" class="dashboard-panel">
          <n-space vertical :size="12">
            <div>{{ t('dashboard.totalUpload') }}: <strong>{{ formatBytes(totalUplink) }}</strong></div>
            <div>{{ t('dashboard.totalDownload') }}: <strong>{{ formatBytes(totalDownlink) }}</strong></div>
          </n-space>
        </n-card>
      </n-gi>
      <n-gi span="2 m:1">
        <n-card :title="t('dashboard.realtimeTraffic')" size="small" :bordered="false" class="dashboard-panel">
          <v-chart :option="trafficChartOption" autoresize class="dashboard-chart" />
        </n-card>
      </n-gi>
    </n-grid>

    <n-card :title="t('dashboard.topUsers')" size="small" :bordered="false" class="dashboard-panel dashboard-table">
      <n-data-table :columns="topUserColumns" :data="topUsers" :pagination="false" size="small" />
    </n-card>
  </div>
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

<style scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.dashboard-hero {
  position: relative;
  display: grid;
  overflow: hidden;
  gap: 24px;
  padding: clamp(22px, 3vw, 36px);
  border: 1px solid var(--panel-border);
  border-radius: var(--panel-radius-xl);
  background: var(--panel-surface);
  box-shadow: var(--panel-shadow);
  color: var(--panel-text);
  grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.95fr);
}

.dashboard-hero::after {
  display: none;
  content: "";
}

.dashboard-hero-copy {
  position: relative;
  z-index: 1;
}

.dashboard-eyebrow {
  margin: 0 0 12px;
  color: var(--panel-text-3);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.24em;
  text-transform: uppercase;
}

.dashboard-headline {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.dashboard-title {
  margin: 0;
  font-size: clamp(2rem, 4vw, 3rem);
  line-height: 1.05;
  letter-spacing: -0.03em;
}

.dashboard-summary {
  max-width: 44rem;
  margin: 14px 0 0;
  color: var(--panel-text-2);
  font-size: 15px;
  line-height: 1.7;
}

.dashboard-actions {
  margin-top: 22px;
}

.dashboard-metrics {
  position: relative;
  z-index: 1;
  display: grid;
  align-self: end;
  gap: 14px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.dashboard-metric-card {
  display: flex;
  min-height: 118px;
  flex-direction: column;
  justify-content: space-between;
  padding: 18px 20px;
  border: 1px solid var(--panel-border);
  border-radius: 24px;
  background: var(--panel-surface-soft);
}

.dashboard-metric-label {
  color: var(--panel-text-3);
  font-size: 12px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.dashboard-metric-value {
  font-size: clamp(1.25rem, 2vw, 1.9rem);
  font-weight: 700;
  line-height: 1.15;
}

.dashboard-panel {
  border: 1px solid var(--panel-border);
  border-radius: var(--panel-radius-xl);
  background: var(--panel-surface);
  box-shadow: var(--panel-shadow);
}

.dashboard-chart {
  height: 240px;
}

:deep(.dashboard-panel > .n-card-header) {
  padding: 22px 24px 0;
}

:deep(.dashboard-panel > .n-card__content) {
  padding: 18px 24px 24px;
}

:deep(.dashboard-panel .n-alert) {
  border-radius: 16px;
}

:deep(.dashboard-table .n-data-table-wrapper) {
  overflow: hidden;
  border-radius: 18px;
}

@media (max-width: 1080px) {
  .dashboard-hero {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .dashboard-hero {
    padding: 20px;
    border-radius: 26px;
  }

  .dashboard-metrics {
    grid-template-columns: 1fr;
  }

  :deep(.dashboard-panel > .n-card-header) {
    padding: 18px 18px 0;
  }

  :deep(.dashboard-panel > .n-card__content) {
    padding: 16px 18px 18px;
  }
}
</style>
