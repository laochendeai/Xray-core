<template>
  <n-space vertical :size="16">
    <n-space justify="space-between" align="center">
      <h2>{{ t('outbounds.title') }}</h2>
      <n-button type="primary" @click="openAddDialog">{{ t('outbounds.addOutbound') }}</n-button>
    </n-space>

    <n-data-table :columns="columns" :data="outbounds" :loading="loading" :pagination="{ pageSize: 20 }" />

    <n-modal v-model:show="showDialog" preset="dialog" :title="dialogTitle" style="width: 760px">
      <n-form :model="form" label-placement="left" label-width="auto">
        <n-form-item :label="t('outbounds.tag')">
          <n-input v-model:value="form.tag" placeholder="my-outbound" :disabled="mode === 'edit'" />
        </n-form-item>
        <n-form-item :label="t('outbounds.configJson')">
          <n-input
            v-model:value="form.configJson"
            type="textarea"
            :rows="14"
            :placeholder="outboundConfigPlaceholder"
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

const outbounds = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const mode = ref<'add' | 'edit' | 'clone'>('add')
const editingTag = ref('')

const form = ref({
  tag: '',
  configJson: '{}'
})

const outboundConfigPlaceholder = `{
  "tag": "my-outbound",
  "senderType": "xray.app.proxyman.SenderConfig",
  "senderSettings": {},
  "proxyType": "xray.proxy.freedom.Config",
  "proxySettings": {}
}`

const dialogTitle = computed(() => {
  if (mode.value === 'edit') return t('outbounds.editOutbound')
  if (mode.value === 'clone') return t('outbounds.cloneOutbound')
  return t('outbounds.addOutbound')
})

const columns: DataTableColumns = [
  { title: t('outbounds.tag'), key: 'tag' },
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
            default: () => t('outbounds.deleteConfirm')
          })
        ]
      })
    }
  }
]

function toPrettyJSON(value: any) {
  return JSON.stringify(value, null, 2)
}

function normalizeOutboundDraft(rawOutbound: any, tag: string) {
  const outbound = rawOutbound && typeof rawOutbound === 'object' ? { ...rawOutbound } : {}
  outbound.tag = tag
  return outbound
}

async function fetchOutbounds() {
  loading.value = true
  try {
    const data = await handlerAPI.listOutbounds()
    outbounds.value = data.outbounds || []
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
    const data = await handlerAPI.getOutbound(tag)
    const outbound = data?.outbound ?? data
    mode.value = 'edit'
    editingTag.value = tag
    form.value = {
      tag,
      configJson: toPrettyJSON(normalizeOutboundDraft(outbound, tag))
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
    const data = await handlerAPI.getOutbound(tag)
    const outbound = data?.outbound ?? data
    mode.value = 'clone'
    editingTag.value = ''
    form.value = {
      tag: cloneTag,
      configJson: toPrettyJSON(normalizeOutboundDraft(outbound, cloneTag))
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
      message.error(t('outbounds.tagRequired'))
      return
    }

    const outbound = JSON.parse(form.value.configJson || '{}')
    outbound.tag = targetTag

    if (mode.value === 'edit') {
      await handlerAPI.updateOutbound(editingTag.value, outbound)
    } else {
      await handlerAPI.addOutbound(outbound)
    }

    message.success(t('common.success'))
    showDialog.value = false
    await fetchOutbounds()
  } catch (err: any) {
    message.error(err?.error || t('outbounds.invalidConfig'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(tag: string) {
  try {
    await handlerAPI.removeOutbound(tag)
    message.success(t('common.success'))
    await fetchOutbounds()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

onMounted(fetchOutbounds)
</script>
