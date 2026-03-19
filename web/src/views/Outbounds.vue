<template>
  <n-space vertical :size="16">
    <n-space justify="space-between" align="center">
      <h2>{{ t('outbounds.title') }}</h2>
      <n-button type="primary" @click="showAdd = true">{{ t('outbounds.addOutbound') }}</n-button>
    </n-space>

    <n-data-table :columns="columns" :data="outbounds" :loading="loading" :pagination="{ pageSize: 20 }" />

    <!-- Add Dialog -->
    <n-modal v-model:show="showAdd" preset="dialog" :title="t('outbounds.addOutbound')" style="width: 600px">
      <n-form :model="form" label-placement="left" label-width="auto">
        <n-form-item :label="t('outbounds.tag')">
          <n-input v-model:value="form.tag" placeholder="my-outbound" />
        </n-form-item>
        <n-form-item :label="t('outbounds.protocol')">
          <n-select v-model:value="form.protocol" :options="protocolOptions" />
        </n-form-item>
        <n-form-item label="Settings (JSON)">
          <n-input v-model:value="form.settings" type="textarea" :rows="6" placeholder='{}' />
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
import { NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput, NSelect, NPopconfirm, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { handlerAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const outbounds = ref<any[]>([])
const loading = ref(false)
const showAdd = ref(false)
const saving = ref(false)

const form = ref({
  tag: '',
  protocol: 'freedom',
  settings: '{}'
})

const protocolOptions = [
  { label: 'Freedom', value: 'freedom' },
  { label: 'Blackhole', value: 'blackhole' },
  { label: 'VLESS', value: 'vless' },
  { label: 'VMess', value: 'vmess' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'Shadowsocks', value: 'shadowsocks' },
  { label: 'SOCKS', value: 'socks' },
  { label: 'HTTP', value: 'http' },
  { label: 'DNS', value: 'dns' },
  { label: 'Loopback', value: 'loopback' },
  { label: 'WireGuard', value: 'wireguard' },
  { label: 'Hysteria', value: 'hysteria' }
]

const columns: DataTableColumns = [
  { title: t('outbounds.tag'), key: 'tag' },
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
            default: () => t('outbounds.deleteConfirm')
          })
        ]
      })
    }
  }
]

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

async function handleAdd() {
  saving.value = true
  try {
    await handlerAPI.addOutbound({
      tag: form.value.tag,
      protocol: form.value.protocol,
      settings: JSON.parse(form.value.settings)
    })
    message.success(t('common.success'))
    showAdd.value = false
    fetchOutbounds()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(tag: string) {
  try {
    await handlerAPI.removeOutbound(tag)
    message.success(t('common.success'))
    fetchOutbounds()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

onMounted(fetchOutbounds)
</script>
