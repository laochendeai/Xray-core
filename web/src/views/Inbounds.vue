<template>
  <n-space vertical :size="16">
    <n-space justify="space-between" align="center">
      <h2>{{ t('inbounds.title') }}</h2>
      <n-button type="primary" @click="showAdd = true">{{ t('inbounds.addInbound') }}</n-button>
    </n-space>

    <n-data-table :columns="columns" :data="inbounds" :loading="loading" :pagination="{ pageSize: 20 }" />

    <!-- Add/Edit Dialog -->
    <n-modal v-model:show="showAdd" preset="dialog" :title="t('inbounds.addInbound')" style="width: 600px">
      <n-form :model="form" label-placement="left" label-width="auto">
        <n-form-item :label="t('inbounds.tag')">
          <n-input v-model:value="form.tag" placeholder="my-inbound" />
        </n-form-item>
        <n-form-item :label="t('inbounds.protocol')">
          <n-select v-model:value="form.protocol" :options="protocolOptions" />
        </n-form-item>
        <n-form-item :label="t('inbounds.listen')">
          <n-input v-model:value="form.listen" placeholder="0.0.0.0" />
        </n-form-item>
        <n-form-item :label="t('inbounds.port')">
          <n-input-number v-model:value="form.port" :min="1" :max="65535" placeholder="443" />
        </n-form-item>
        <n-form-item label="Settings (JSON)">
          <n-input v-model:value="form.settings" type="textarea" :rows="6" placeholder='{"clients": []}' />
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
import { NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput, NInputNumber, NSelect, NPopconfirm, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { handlerAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const inbounds = ref<any[]>([])
const loading = ref(false)
const showAdd = ref(false)
const saving = ref(false)

const form = ref({
  tag: '',
  protocol: 'vless',
  listen: '0.0.0.0',
  port: 443,
  settings: '{}'
})

const protocolOptions = [
  { label: 'VLESS', value: 'vless' },
  { label: 'VMess', value: 'vmess' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'Shadowsocks', value: 'shadowsocks' },
  { label: 'SOCKS', value: 'socks' },
  { label: 'HTTP', value: 'http' },
  { label: 'Dokodemo-door', value: 'dokodemo-door' },
  { label: 'Hysteria', value: 'hysteria' },
  { label: 'WireGuard', value: 'wireguard' }
]

const columns: DataTableColumns = [
  { title: t('inbounds.tag'), key: 'tag' },
  {
    title: t('common.actions'),
    key: 'actions',
    render(row: any) {
      return h(NSpace, {}, {
        default: () => [
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

async function handleAdd() {
  saving.value = true
  try {
    await handlerAPI.addInbound({
      tag: form.value.tag,
      protocol: form.value.protocol,
      listen: form.value.listen,
      port: form.value.port,
      settings: JSON.parse(form.value.settings)
    })
    message.success(t('common.success'))
    showAdd.value = false
    fetchInbounds()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(tag: string) {
  try {
    await handlerAPI.removeInbound(tag)
    message.success(t('common.success'))
    fetchInbounds()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

onMounted(fetchInbounds)
</script>
