<template>
  <n-space vertical :size="16">
    <n-space justify="space-between" align="center">
      <h2>{{ t('subscriptions.title') }}</h2>
      <n-button type="primary" @click="showAdd = true">{{ t('subscriptions.addSubscription') }}</n-button>
    </n-space>

    <n-data-table :columns="columns" :data="subscriptions" :loading="loading" :pagination="{ pageSize: 20 }" />

    <!-- Add Dialog -->
    <n-modal v-model:show="showAdd" preset="dialog" :title="t('subscriptions.addSubscription')" style="width: 600px">
      <n-form :model="form" label-placement="left" label-width="auto">
        <n-form-item :label="t('subscriptions.url')">
          <n-input v-model:value="form.url" placeholder="https://..." />
        </n-form-item>
        <n-form-item :label="t('subscriptions.remark')">
          <n-input v-model:value="form.remark" placeholder="" />
        </n-form-item>
        <n-form-item :label="t('subscriptions.autoRefresh')">
          <n-switch v-model:value="form.autoRefresh" />
        </n-form-item>
        <n-form-item :label="t('subscriptions.refreshInterval')" v-if="form.autoRefresh">
          <n-input-number v-model:value="form.refreshIntervalMin" :min="5" :max="1440" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="showAdd = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="handleAdd">{{ t('common.confirm') }}</n-button>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput, NInputNumber, NSwitch, NPopconfirm, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { subscriptionAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const subscriptions = ref<any[]>([])
const loading = ref(false)
const showAdd = ref(false)
const saving = ref(false)

const form = ref({
  url: '',
  remark: '',
  autoRefresh: true,
  refreshIntervalMin: 60
})

const columns: DataTableColumns = [
  { title: () => t('subscriptions.remark'), key: 'remark', width: 150 },
  {
    title: () => t('subscriptions.url'),
    key: 'url',
    ellipsis: { tooltip: true },
    width: 300
  },
  {
    title: () => t('subscriptions.autoRefresh'),
    key: 'autoRefresh',
    width: 100,
    render(row: any) {
      return h(NTag, { type: row.autoRefresh ? 'success' : 'default', size: 'small' }, {
        default: () => row.autoRefresh ? t('common.enabled') : t('common.disabled')
      })
    }
  },
  { title: () => t('subscriptions.refreshInterval'), key: 'refreshIntervalMin', width: 120 },
  {
    title: () => t('subscriptions.lastRefresh'),
    key: 'lastRefresh',
    width: 180,
    render(row: any) {
      return row.lastRefresh ? new Date(row.lastRefresh).toLocaleString() : t('subscriptions.never')
    }
  },
  { title: () => t('subscriptions.nodeCount'), key: 'nodeCount', width: 80 },
  {
    title: () => t('common.actions'),
    key: 'actions',
    width: 200,
    render(row: any) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(NButton, {
            size: 'small',
            onClick: () => handleRefresh(row.id),
            loading: refreshingId.value === row.id
          }, { default: () => t('subscriptions.refreshNow') }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDelete(row.id)
          }, {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
            default: () => t('subscriptions.deleteConfirm')
          })
        ]
      })
    }
  }
]

const refreshingId = ref<string | null>(null)

async function fetchSubscriptions() {
  loading.value = true
  try {
    const data = await subscriptionAPI.list()
    subscriptions.value = data.subscriptions || []
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    loading.value = false
  }
}

async function handleAdd() {
  if (!form.value.url) {
    message.warning(t('subscriptions.url') + ' required')
    return
  }
  saving.value = true
  try {
    await subscriptionAPI.add(form.value)
    message.success(t('common.success'))
    showAdd.value = false
    form.value = { url: '', remark: '', autoRefresh: true, refreshIntervalMin: 60 }
    fetchSubscriptions()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    saving.value = false
  }
}

async function handleRefresh(id: string) {
  refreshingId.value = id
  try {
    await subscriptionAPI.refresh(id)
    message.success(t('common.success'))
    fetchSubscriptions()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    refreshingId.value = null
  }
}

async function handleDelete(id: string) {
  try {
    await subscriptionAPI.delete(id)
    message.success(t('common.success'))
    fetchSubscriptions()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

onMounted(fetchSubscriptions)
</script>
