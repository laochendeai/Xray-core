<template>
  <n-space vertical :size="16">
    <div>
      <h2 class="page-title">{{ t('readiness.title') }}</h2>
      <div class="page-subtitle">{{ t('readiness.subtitle') }}</div>
    </div>

    <n-card size="small">
      <template #header>
        <n-space align="center" :size="12">
          <span>{{ t('readiness.summaryTitle') }}</span>
          <n-tag :type="overview.type">{{ overview.badgeLabel }}</n-tag>
        </n-space>
      </template>
      <template #header-extra>
        <n-button size="small" @click="fetchReadiness" :loading="loading">
          {{ t('common.refresh') }}
        </n-button>
      </template>

      <n-space vertical :size="12">
        <n-alert :type="overview.type" :title="overview.title">
          {{ overview.description }}
        </n-alert>

        <n-alert v-if="loadError" type="error">
          {{ loadError }}
        </n-alert>

        <n-grid :cols="3" :x-gap="12" responsive="screen" item-responsive>
          <n-gi span="3 m:1">
            <n-statistic :label="t('readiness.cards.blocking')">
              <template #default>{{ readiness?.blockingCount ?? 0 }}</template>
            </n-statistic>
          </n-gi>
          <n-gi span="3 m:1">
            <n-statistic :label="t('readiness.cards.warning')">
              <template #default>{{ readiness?.warningCount ?? 0 }}</template>
            </n-statistic>
          </n-gi>
          <n-gi span="3 m:1">
            <n-statistic :label="t('readiness.cards.checks')">
              <template #default>{{ readiness?.checks.length ?? 0 }}</template>
            </n-statistic>
          </n-gi>
        </n-grid>

        <div class="page-subtitle">
          {{ t('readiness.lastUpdated') }}: {{ formatDateTime(readiness?.updatedAt) }}
        </div>
      </n-space>
    </n-card>

    <n-spin :show="loading">
      <n-space vertical :size="12">
        <n-card
          v-for="item in describedChecks"
          :key="item.check.key"
          size="small"
          embedded
        >
          <template #header>
            <n-space align="center" :size="8">
              <n-tag :type="readinessSeverityType(item.check.severity)">
                {{ readinessSeverityLabel(t, item.check.severity) }}
              </n-tag>
              <strong>{{ item.description.title }}</strong>
            </n-space>
          </template>
          <template #header-extra>
            <n-space align="center" :size="8">
              <n-tag size="small" :bordered="false">
                {{ readinessAreaLabel(t, item.check.area) }}
              </n-tag>
              <n-button
                v-if="item.check.actionRoute"
                text
                type="primary"
                @click="openAction(item.check.actionRoute)"
              >
                {{ t('readiness.goToArea') }}
              </n-button>
            </n-space>
          </template>

          <n-space vertical :size="8">
            <div>{{ item.description.summary }}</div>
            <div
              v-for="detail in item.description.details"
              :key="detail"
              class="check-detail"
            >
              {{ detail }}
            </div>
          </n-space>
        </n-card>

        <n-empty v-if="!describedChecks.length && !loading" :description="t('readiness.noChecks')" />
      </n-space>
    </n-spin>
  </n-space>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NAlert,
  NButton,
  NCard,
  NEmpty,
  NGi,
  NGrid,
  NSpace,
  NSpin,
  NStatistic,
  NTag
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { readinessAPI } from '@/api/client'
import type { ReadinessResponse } from '@/api/types'
import {
  describeReadinessCheck,
  describeReadinessOverview,
  readinessAreaLabel,
  readinessSeverityLabel,
  readinessSeverityType
} from '@/utils/readiness'

const router = useRouter()
const { t } = useI18n()

const readiness = ref<ReadinessResponse | null>(null)
const loading = ref(false)
const loadError = ref('')

const overview = computed(() => describeReadinessOverview(t, readiness.value))
const describedChecks = computed(() =>
  (readiness.value?.checks ?? []).map((check) => ({
    check,
    description: describeReadinessCheck(t, check)
  }))
)

async function fetchReadiness() {
  loading.value = true
  loadError.value = ''
  try {
    readiness.value = await readinessAPI.get()
  } catch (err: any) {
    loadError.value = err?.error || t('common.error')
  } finally {
    loading.value = false
  }
}

function openAction(route: string) {
  router.push(route)
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

onMounted(() => {
  fetchReadiness()
})
</script>

<style scoped>
.page-title {
  margin: 0;
}

.page-subtitle {
  margin-top: 6px;
  color: var(--n-text-color-3);
}

.check-detail {
  color: var(--n-text-color-2);
  font-size: 13px;
}
</style>
