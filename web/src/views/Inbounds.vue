<template>
  <n-space vertical :size="16">
    <n-space justify="space-between" align="center">
      <h2>{{ t('inbounds.title') }}</h2>
      <n-button type="primary" @click="openAddDialog">{{ t('inbounds.addInbound') }}</n-button>
    </n-space>

    <n-data-table :columns="columns" :data="inbounds" :loading="loading" :pagination="{ pageSize: 20 }" />

    <n-modal v-model:show="showDialog" preset="dialog" :title="dialogTitle" style="width: 760px">
      <n-form :model="form" label-placement="left" label-width="auto">
        <n-form-item :label="t('inbounds.tag')">
          <n-input v-model:value="form.tag" placeholder="my-inbound" :disabled="mode === 'edit'" />
        </n-form-item>
        <n-form-item :label="t('inbounds.configJson')">
          <n-input
            v-model:value="form.configJson"
            type="textarea"
            :rows="14"
            :placeholder="inboundConfigPlaceholder"
          />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="showDialog = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="handleSubmit">{{ t('common.confirm') }}</n-button>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput, NPopconfirm, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { handlerAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const inbounds = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const mode = ref<'add' | 'edit' | 'clone'>('add')
const editingTag = ref('')

const form = ref({
  tag: '',
  configJson: '{}'
})

const inboundConfigPlaceholder = `{
  "tag": "my-inbound",
  "receiverType": "xray.app.proxyman.ReceiverConfig",
  "receiverSettings": {},
  "proxyType": "xray.proxy.vless.inbound.Config",
  "proxySettings": {}
}`

const dialogTitle = computed(() => {
  if (mode.value === 'edit') return t('inbounds.editInbound')
  if (mode.value === 'clone') return t('inbounds.cloneInbound')
  return t('inbounds.addInbound')
})

const columns: DataTableColumns = [
  { title: t('inbounds.tag'), key: 'tag' },
  {
    title: t('common.actions'),
    key: 'actions',
    render(row: any) {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => openEditDialog(row.tag) }, { default: () => t('common.edit') }),
          h(NButton, { size: 'small', onClick: () => openCloneDialog(row.tag) }, { default: () => t('common.clone') }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDelete(row.tag)
          }, {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
            default: () => t('inbounds.deleteConfirm')
          })
        ]
      })
    }
  }
]

function toPrettyJSON(value: any) {
  return JSON.stringify(value, null, 2)
}

function normalizeInboundDraft(rawInbound: any, tag: string) {
  const inbound = rawInbound && typeof rawInbound === 'object' ? { ...rawInbound } : {}
  inbound.tag = tag
  return inbound
}

async function fetchInbounds() {
  loading.value = true
  try {
    const data = await handlerAPI.listInbounds()
    inbounds.value = data.inbounds || []
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    loading.value = false
  }
}

function openAddDialog() {
  mode.value = 'add'
  editingTag.value = ''
  form.value = {
    tag: '',
    configJson: '{}'
  }
  showDialog.value = true
}

async function openEditDialog(tag: string) {
  loading.value = true
  try {
    const data = await handlerAPI.getInbound(tag)
    const inbound = data?.inbound ?? data
    mode.value = 'edit'
    editingTag.value = tag
    form.value = {
      tag,
      configJson: toPrettyJSON(normalizeInboundDraft(inbound, tag))
    }
    showDialog.value = true
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    loading.value = false
  }
}

async function openCloneDialog(tag: string) {
  loading.value = true
  try {
    const cloneTag = `${tag}-copy`
    const data = await handlerAPI.getInbound(tag)
    const inbound = data?.inbound ?? data
    mode.value = 'clone'
    editingTag.value = ''
    form.value = {
      tag: cloneTag,
      configJson: toPrettyJSON(normalizeInboundDraft(inbound, cloneTag))
    }
    showDialog.value = true
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    loading.value = false
  }
}

async function handleSubmit() {
  saving.value = true
  try {
    const targetTag = mode.value === 'edit' ? editingTag.value : form.value.tag.trim()
    if (!targetTag) {
      message.error(t('inbounds.tagRequired'))
      return
    }

    const inbound = JSON.parse(form.value.configJson || '{}')
    inbound.tag = targetTag

    if (mode.value === 'edit') {
      await handlerAPI.updateInbound(editingTag.value, inbound)
    } else {
      await handlerAPI.addInbound(inbound)
    }

    message.success(t('common.success'))
    showDialog.value = false
    await fetchInbounds()
  } catch (err: any) {
    message.error(err?.error || t('inbounds.invalidConfig'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(tag: string) {
  try {
    await handlerAPI.removeInbound(tag)
    message.success(t('common.success'))
    await fetchInbounds()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

onMounted(fetchInbounds)
</script>
