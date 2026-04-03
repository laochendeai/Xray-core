<template>
  <n-space vertical :size="16">
    <n-space justify="space-between" align="center">
      <h2>{{ t('subscriptions.title') }}</h2>
      <n-button type="primary" @click="openAddDialog">{{ t('subscriptions.addSubscription') }}</n-button>
    </n-space>

    <n-data-table :columns="columns" :data="subscriptions" :loading="loading" :pagination="{ pageSize: 20 }" />

    <n-modal
      :show="showDialog"
      preset="dialog"
      :title="dialogTitle"
      style="width: 640px"
      @update:show="handleDialogVisibilityChange"
    >
      <n-form :model="form" label-placement="left" label-width="auto">
        <n-form-item :label="t('subscriptions.sourceType')">
          <n-select
            v-model:value="form.sourceType"
            :options="sourceTypeOptions"
            :disabled="isEditing"
            @update:value="handleSourceTypeChange"
          />
        </n-form-item>

        <n-form-item v-if="form.sourceType === 'url'" :label="t('subscriptions.url')">
          <n-input v-model:value="form.url" placeholder="https://..." />
        </n-form-item>

        <n-form-item v-else-if="!isEditing && form.sourceType === 'manual'" :label="t('subscriptions.manualContent')">
          <n-input
            v-model:value="form.content"
            type="textarea"
            :autosize="{ minRows: 8, maxRows: 14 }"
            :placeholder="t('subscriptions.manualPlaceholder')"
          />
        </n-form-item>

        <n-form-item v-else-if="!isEditing && form.sourceType === 'file'" :label="t('subscriptions.localFile')">
          <n-space vertical :size="8" class="file-import-block">
            <input ref="fileInputRef" class="hidden-file-input" type="file" :accept="fileAccept" @change="handleFilePicked" />
            <n-space align="center" wrap>
              <n-button secondary @click="openFilePicker">{{ t('subscriptions.selectFile') }}</n-button>
              <span class="file-import-meta">
                {{ form.sourceName || t('subscriptions.noFileSelected') }}
              </span>
            </n-space>
            <div class="file-import-meta">{{ t('subscriptions.fileImportHint') }}</div>
          </n-space>
        </n-form-item>

        <n-form-item v-else-if="isEditing" :label="t('subscriptions.source')">
          <n-space vertical :size="8" class="file-import-block">
            <n-input :value="editSourceSummary" readonly />
            <div v-if="editingSubscription?.sourceType !== 'url'" class="file-import-meta">
              {{ t('subscriptions.nonUrlEditHint') }}
            </div>
          </n-space>
        </n-form-item>

        <n-form-item :label="t('subscriptions.remark')">
          <n-input v-model:value="form.remark" />
        </n-form-item>

        <n-form-item v-if="form.sourceType === 'url'" :label="t('subscriptions.autoRefresh')">
          <n-switch v-model:value="form.autoRefresh" />
        </n-form-item>

        <n-form-item v-if="form.sourceType === 'url'" :label="t('subscriptions.refreshInterval')">
          <n-input-number v-model:value="form.refreshIntervalMin" :min="5" :max="1440" />
        </n-form-item>
      </n-form>

      <template #action>
        <n-button @click="handleDialogVisibilityChange(false)">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="handleSubmit">{{ submitLabel }}</n-button>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { computed, h, ref } from 'vue'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NPopconfirm,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  useMessage,
  type DataTableColumns
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { subscriptionAPI } from '@/api/client'
import type { SubscriptionRecord, SubscriptionSourceType, SubscriptionUpdateRequest } from '@/api/types'

type DialogMode = 'add' | 'edit'

const { t } = useI18n()
const message = useMessage()

const subscriptions = ref<SubscriptionRecord[]>([])
const loading = ref(false)
const showDialog = ref(false)
const dialogMode = ref<DialogMode>('add')
const saving = ref(false)
const refreshingId = ref<string | null>(null)
const toggleLoadingId = ref<string | null>(null)
const editingSubscription = ref<SubscriptionRecord | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

const fileAccept = '.txt,.sub,.conf,.json,.yaml,.yml'

function createDefaultForm() {
  return {
    sourceType: 'url' as SubscriptionSourceType,
    url: '',
    content: '',
    sourceName: '',
    remark: '',
    autoRefresh: true,
    refreshIntervalMin: 60
  }
}

const form = ref(createDefaultForm())

const isEditing = computed(() => dialogMode.value === 'edit')
const dialogTitle = computed(() =>
  isEditing.value ? t('subscriptions.editSubscription') : t('subscriptions.addSubscription')
)
const submitLabel = computed(() => (isEditing.value ? t('common.save') : t('common.confirm')))
const editSourceSummary = computed(() => {
  if (!editingSubscription.value) {
    return '-'
  }
  return sourceDisplay(editingSubscription.value)
})

const sourceTypeOptions = computed(() => [
  { label: t('subscriptions.sourceTypeOptions.url'), value: 'url' as SubscriptionSourceType },
  { label: t('subscriptions.sourceTypeOptions.manual'), value: 'manual' as SubscriptionSourceType },
  { label: t('subscriptions.sourceTypeOptions.file'), value: 'file' as SubscriptionSourceType }
])

const columns: DataTableColumns<SubscriptionRecord> = [
  { title: () => t('subscriptions.remark'), key: 'remark', width: 150 },
  {
    title: () => t('subscriptions.sourceType'),
    key: 'sourceType',
    width: 130,
    render(row) {
      return h(
        NTag,
        { size: 'small', type: sourceTypeTagType(row.sourceType) },
        { default: () => sourceTypeLabel(row.sourceType) }
      )
    }
  },
  {
    title: () => t('subscriptions.source'),
    key: 'source',
    ellipsis: { tooltip: true },
    width: 320,
    render(row) {
      return sourceDisplay(row)
    }
  },
  {
    title: () => t('subscriptions.autoRefresh'),
    key: 'autoRefresh',
    width: 110,
    render(row) {
      return h(
        NTag,
        { type: row.autoRefresh ? 'success' : 'default', size: 'small' },
        { default: () => (row.autoRefresh ? t('common.enabled') : t('common.disabled')) }
      )
    }
  },
  { title: () => t('subscriptions.refreshInterval'), key: 'refreshIntervalMin', width: 140 },
  {
    title: () => t('subscriptions.lastRefresh'),
    key: 'lastRefresh',
    width: 180,
    render(row) {
      return row.lastRefresh ? new Date(row.lastRefresh).toLocaleString() : t('subscriptions.never')
    }
  },
  { title: () => t('subscriptions.nodeCount'), key: 'nodeCount', width: 80 },
  {
    title: () => t('common.actions'),
    key: 'actions',
    width: 320,
    render(row) {
      const actions = [
        h(
          NButton,
          {
            size: 'small',
            onClick: () => openEditDialog(row)
          },
          { default: () => t('common.edit') }
        )
      ]

      if (row.sourceType === 'url') {
        actions.push(
          h(
            NButton,
            {
              size: 'small',
              type: row.autoRefresh ? 'warning' : 'success',
              loading: toggleLoadingId.value === row.id,
              onClick: () => handleToggleAutoRefresh(row)
            },
            { default: () => (row.autoRefresh ? t('subscriptions.pause') : t('subscriptions.resume')) }
          )
        )
      }

      actions.push(
        h(
          NButton,
          {
            size: 'small',
            onClick: () => handleRefresh(row.id),
            loading: refreshingId.value === row.id
          },
          { default: () => t('subscriptions.refreshNow') }
        )
      )
      actions.push(
        h(
          NPopconfirm,
          { onPositiveClick: () => handleDelete(row.id) },
          {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
            default: () => t('subscriptions.deleteConfirm')
          }
        )
      )

      return h(
        NSpace,
        { size: 'small', wrap: true },
        {
          default: () => actions
        }
      )
    }
  }
]

function sourceTypeLabel(sourceType: SubscriptionSourceType) {
  return t(`subscriptions.sourceTypeOptions.${sourceType}`)
}

function sourceTypeTagType(sourceType: SubscriptionSourceType) {
  switch (sourceType) {
    case 'manual':
      return 'warning'
    case 'file':
      return 'info'
    default:
      return 'success'
  }
}

function sourceDisplay(row: SubscriptionRecord) {
  if (row.sourceType === 'file') {
    return row.sourceName || t('subscriptions.fileContent')
  }
  if (row.sourceType === 'manual') {
    return row.sourceName || t('subscriptions.manualContentSummary')
  }
  return row.url || '-'
}

function resetDialogState() {
  dialogMode.value = 'add'
  editingSubscription.value = null
  form.value = createDefaultForm()
  if (fileInputRef.value) {
    fileInputRef.value.value = ''
  }
}

function openAddDialog() {
  resetDialogState()
  showDialog.value = true
}

function openEditDialog(row: SubscriptionRecord) {
  dialogMode.value = 'edit'
  editingSubscription.value = row
  form.value = {
    sourceType: row.sourceType,
    url: row.url || '',
    content: '',
    sourceName: row.sourceName || '',
    remark: row.remark,
    autoRefresh: row.autoRefresh,
    refreshIntervalMin: row.refreshIntervalMin || 60
  }
  if (fileInputRef.value) {
    fileInputRef.value.value = ''
  }
  showDialog.value = true
}

function handleDialogVisibilityChange(value: boolean) {
  showDialog.value = value
  if (!value) {
    resetDialogState()
  }
}

function handleSourceTypeChange(value: SubscriptionSourceType) {
  if (isEditing.value) {
    return
  }
  form.value.sourceType = value
  form.value.url = ''
  form.value.content = ''
  form.value.sourceName = ''
  form.value.autoRefresh = value === 'url'
  form.value.refreshIntervalMin = 60
  if (fileInputRef.value) {
    fileInputRef.value.value = ''
  }
}

function openFilePicker() {
  fileInputRef.value?.click()
}

async function handleFilePicked(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) {
    form.value.sourceName = ''
    form.value.content = ''
    return
  }

  try {
    form.value.sourceName = file.name
    form.value.content = await file.text()
  } catch (err: any) {
    form.value.sourceName = ''
    form.value.content = ''
    message.error(err?.message || t('common.error'))
  }
}

async function fetchSubscriptions() {
  loading.value = true
  try {
    const data = await subscriptionAPI.list()
    subscriptions.value = data.subscriptions || []
  } catch (err: any) {
    message.error(err?.error || err?.message || t('common.error'))
  } finally {
    loading.value = false
  }
}

function buildUpdatePayload(): SubscriptionUpdateRequest {
  const payload: SubscriptionUpdateRequest = {
    sourceType: form.value.sourceType,
    remark: form.value.remark
  }

  if (form.value.sourceType === 'url') {
    payload.url = form.value.url.trim()
    payload.autoRefresh = form.value.autoRefresh
    payload.refreshIntervalMin = form.value.refreshIntervalMin
  }

  return payload
}

async function handleSubmit() {
  if (form.value.sourceType === 'url' && !form.value.url.trim()) {
    message.warning(t('subscriptions.urlRequired'))
    return
  }
  if (!isEditing.value && form.value.sourceType !== 'url' && !form.value.content.trim()) {
    message.warning(t('subscriptions.contentRequired'))
    return
  }

  saving.value = true
  try {
    if (isEditing.value && editingSubscription.value) {
      await subscriptionAPI.update(editingSubscription.value.id, buildUpdatePayload())
    } else {
      await subscriptionAPI.add({
        sourceType: form.value.sourceType,
        url: form.value.sourceType === 'url' ? form.value.url.trim() : undefined,
        content: form.value.sourceType === 'url' ? undefined : form.value.content,
        sourceName: form.value.sourceType === 'file' ? form.value.sourceName : undefined,
        remark: form.value.remark,
        autoRefresh: form.value.sourceType === 'url' ? form.value.autoRefresh : false,
        refreshIntervalMin: form.value.sourceType === 'url' ? form.value.refreshIntervalMin : 0
      })
    }
    message.success(t('common.success'))
    handleDialogVisibilityChange(false)
    await fetchSubscriptions()
  } catch (err: any) {
    message.error(err?.error || err?.message || t('common.error'))
  } finally {
    saving.value = false
  }
}

async function handleRefresh(id: string) {
  refreshingId.value = id
  try {
    await subscriptionAPI.refresh(id)
    message.success(t('common.success'))
    await fetchSubscriptions()
  } catch (err: any) {
    message.error(err?.error || err?.message || t('common.error'))
  } finally {
    refreshingId.value = null
  }
}

async function handleToggleAutoRefresh(row: SubscriptionRecord) {
  toggleLoadingId.value = row.id
  try {
    await subscriptionAPI.update(row.id, {
      sourceType: row.sourceType,
      autoRefresh: !row.autoRefresh
    })
    message.success(t('common.success'))
    await fetchSubscriptions()
  } catch (err: any) {
    message.error(err?.error || err?.message || t('common.error'))
  } finally {
    toggleLoadingId.value = null
  }
}

async function handleDelete(id: string) {
  try {
    await subscriptionAPI.delete(id)
    message.success(t('common.success'))
    await fetchSubscriptions()
  } catch (err: any) {
    message.error(err?.error || err?.message || t('common.error'))
  }
}

fetchSubscriptions()
</script>

<style scoped>
h2 {
  margin: 0;
}

.hidden-file-input {
  display: none;
}

.file-import-block {
  width: 100%;
}

.file-import-meta {
  color: var(--n-text-color-3);
  font-size: 13px;
}
</style>
