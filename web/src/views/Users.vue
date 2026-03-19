<template>
  <n-space vertical :size="16">
    <n-space justify="space-between" align="center">
      <h2>{{ t('users.title') }}</h2>
      <n-button type="primary" @click="showAdd = true">{{ t('users.addUser') }}</n-button>
    </n-space>

    <n-data-table :columns="columns" :data="users" :loading="loading" :pagination="{ pageSize: 20 }" />

    <!-- Add User Dialog -->
    <n-modal v-model:show="showAdd" preset="dialog" :title="t('users.addUser')" style="width: 500px">
      <n-form :model="addForm" label-placement="left" label-width="auto">
        <n-form-item :label="t('users.inboundTag')">
          <n-input v-model:value="addForm.inboundTag" placeholder="inbound-tag" />
        </n-form-item>
        <n-form-item :label="t('users.email')">
          <n-input v-model:value="addForm.email" placeholder="user@example.com" />
        </n-form-item>
        <n-form-item :label="t('users.level')">
          <n-input-number v-model:value="addForm.level" :min="0" />
        </n-form-item>
        <n-form-item label="Account Type">
          <n-input v-model:value="addForm.accountType" placeholder="xray.proxy.vless.Account" />
        </n-form-item>
        <n-form-item label="Account (JSON)">
          <n-input v-model:value="addForm.account" type="textarea" :rows="4" placeholder='{"id": "uuid"}' />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="showAdd = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="handleAddUser">{{ t('common.confirm') }}</n-button>
      </template>
    </n-modal>

    <!-- Share Link Dialog -->
    <n-modal v-model:show="showShare" preset="dialog" :title="t('users.shareLink')" style="width: 500px">
      <n-form :model="shareForm" label-placement="left" label-width="auto">
        <n-form-item :label="t('outbounds.protocol')">
          <n-select v-model:value="shareForm.protocol" :options="shareProtocols" />
        </n-form-item>
        <n-form-item label="Address">
          <n-input v-model:value="shareForm.address" placeholder="server.example.com" />
        </n-form-item>
        <n-form-item :label="t('outbounds.port')">
          <n-input-number v-model:value="shareForm.port" :min="1" :max="65535" />
        </n-form-item>
        <n-form-item label="UUID/Password">
          <n-input v-model:value="shareForm.uuid" />
        </n-form-item>
        <n-form-item label="Transport">
          <n-select v-model:value="shareForm.type" :options="transportOptions" />
        </n-form-item>
        <n-form-item label="TLS">
          <n-select v-model:value="shareForm.tls" :options="tlsOptions" />
        </n-form-item>
        <n-form-item label="SNI">
          <n-input v-model:value="shareForm.sni" />
        </n-form-item>
        <n-form-item label="Path">
          <n-input v-model:value="shareForm.path" />
        </n-form-item>
        <n-button type="primary" @click="generateLink" :loading="generating" block>
          {{ t('users.generateLink') }}
        </n-button>
        <template v-if="generatedLink">
          <n-divider />
          <n-input :value="generatedLink" type="textarea" readonly :rows="3" />
          <n-space style="margin-top: 8px">
            <n-button size="small" @click="copyLink">{{ t('common.copy') }}</n-button>
          </n-space>
        </template>
      </n-form>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NSpace, NButton, NDataTable, NModal, NForm, NFormItem, NInput, NInputNumber, NSelect, NPopconfirm, NDivider, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { usersAPI, handlerAPI, shareAPI } from '@/api/client'
import { formatBytes, copyToClipboard } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()

const users = ref<any[]>([])
const loading = ref(false)
const showAdd = ref(false)
const showShare = ref(false)
const saving = ref(false)
const generating = ref(false)
const generatedLink = ref('')

const addForm = ref({
  inboundTag: '',
  email: '',
  level: 0,
  accountType: '',
  account: '{}'
})

const shareForm = ref({
  protocol: 'vless',
  address: '',
  port: 443,
  uuid: '',
  type: 'tcp',
  tls: 'tls',
  sni: '',
  path: ''
})

const shareProtocols = [
  { label: 'VLESS', value: 'vless' },
  { label: 'VMess', value: 'vmess' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'Shadowsocks', value: 'ss' }
]

const transportOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'WebSocket', value: 'ws' },
  { label: 'gRPC', value: 'grpc' },
  { label: 'HTTP/2', value: 'h2' },
  { label: 'QUIC', value: 'quic' },
  { label: 'KCP', value: 'kcp' },
  { label: 'HTTPUpgrade', value: 'httpupgrade' },
  { label: 'SplitHTTP', value: 'splithttp' }
]

const tlsOptions = [
  { label: 'TLS', value: 'tls' },
  { label: 'REALITY', value: 'reality' },
  { label: 'None', value: 'none' }
]

const columns: DataTableColumns = [
  { title: t('users.email'), key: 'email' },
  { title: t('users.inboundTag'), key: 'inboundTag' },
  { title: t('users.level'), key: 'level' },
  {
    title: t('users.onlineStatus'), key: 'online',
    render: (row: any) => row.online ? t('common.online') : t('common.offline')
  },
  {
    title: t('users.uplink'), key: 'uplink',
    render: (row: any) => formatBytes(row.uplink || 0)
  },
  {
    title: t('users.downlink'), key: 'downlink',
    render: (row: any) => formatBytes(row.downlink || 0)
  },
  {
    title: t('common.actions'), key: 'actions',
    render(row: any) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => openShareDialog(row) }, { default: () => t('users.shareLink') }),
          h(NPopconfirm, {
            onPositiveClick: () => handleDelete(row)
          }, {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
            default: () => t('users.deleteConfirm')
          })
        ]
      })
    }
  }
]

async function fetchUsers() {
  loading.value = true
  try {
    const data = await usersAPI.listAll()
    users.value = data.users || []
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    loading.value = false
  }
}

async function handleAddUser() {
  saving.value = true
  try {
    let account = undefined
    if (addForm.value.account && addForm.value.account !== '{}') {
      account = JSON.parse(addForm.value.account)
    }
    await handlerAPI.addInboundUser(addForm.value.inboundTag, {
      email: addForm.value.email,
      level: addForm.value.level,
      accountType: addForm.value.accountType,
      account
    })
    message.success(t('common.success'))
    showAdd.value = false
    fetchUsers()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(row: any) {
  try {
    if (row.inboundTag) {
      await handlerAPI.removeInboundUser(row.inboundTag, row.email)
    } else {
      await usersAPI.deleteUser(row.email)
    }
    message.success(t('common.success'))
    fetchUsers()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

function openShareDialog(row: any) {
  shareForm.value.protocol = 'vless'
  shareForm.value.uuid = ''
  generatedLink.value = ''
  showShare.value = true
}

async function generateLink() {
  generating.value = true
  try {
    const resp = await shareAPI.generate({
      protocol: shareForm.value.protocol,
      address: shareForm.value.address,
      port: shareForm.value.port,
      uuid: shareForm.value.uuid,
      type: shareForm.value.type,
      tls: shareForm.value.tls,
      sni: shareForm.value.sni,
      path: shareForm.value.path
    })
    generatedLink.value = resp.link
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    generating.value = false
  }
}

async function copyLink() {
  try {
    await copyToClipboard(generatedLink.value)
    message.success(t('common.copied'))
  } catch {
    message.error(t('common.error'))
  }
}

onMounted(fetchUsers)
</script>
