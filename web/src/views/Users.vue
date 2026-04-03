<template>
  <n-space vertical :size="16">
    <n-space justify="space-between" align="center">
      <h2>{{ t('users.title') }}</h2>
      <n-button type="primary" @click="openAddDialog">{{ t('users.addUser') }}</n-button>
    </n-space>

    <n-data-table :columns="columns" :data="users" :loading="loading" :pagination="{ pageSize: 20 }" />

    <n-modal v-model:show="showEditor" preset="dialog" :title="editorTitle" style="width: 560px">
      <n-form :model="userForm" label-placement="left" label-width="auto">
        <n-form-item :label="t('users.inboundTag')">
          <n-input v-model:value="userForm.inboundTag" placeholder="inbound-tag" :disabled="mode === 'edit'" />
        </n-form-item>
        <n-form-item :label="t('users.email')">
          <n-input v-model:value="userForm.email" placeholder="user@example.com" />
        </n-form-item>
        <n-form-item :label="t('users.level')">
          <n-input-number v-model:value="userForm.level" :min="0" />
        </n-form-item>
        <n-form-item :label="t('users.accountType')">
          <n-input v-model:value="userForm.accountType" placeholder="xray.proxy.vless.Account" />
        </n-form-item>
        <n-form-item :label="t('users.accountJson')">
          <n-input v-model:value="userForm.account" type="textarea" :rows="6" placeholder='{"id": "uuid"}' />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="showEditor = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="saving" @click="handleSubmit">{{ t('common.confirm') }}</n-button>
      </template>
    </n-modal>

    <n-modal v-model:show="showShare" preset="dialog" :title="t('users.shareLink')" style="width: 560px">
      <n-form :model="shareForm" label-placement="left" label-width="auto">
        <n-form-item :label="t('users.protocol')">
          <n-select v-model:value="shareForm.protocol" :options="shareProtocols" />
        </n-form-item>
        <n-form-item :label="t('users.address')">
          <n-input v-model:value="shareForm.address" placeholder="server.example.com" />
        </n-form-item>
        <n-form-item :label="t('outbounds.port')">
          <n-input-number v-model:value="shareForm.port" :min="1" :max="65535" />
        </n-form-item>
        <n-form-item :label="t('users.credential')">
          <n-input v-model:value="shareForm.credential" />
        </n-form-item>
        <n-form-item :label="t('users.transport')">
          <n-select v-model:value="shareForm.type" :options="transportOptions" />
        </n-form-item>
        <n-form-item :label="t('users.tls')">
          <n-select v-model:value="shareForm.tls" :options="tlsOptions" />
        </n-form-item>
        <n-form-item :label="t('users.sni')">
          <n-input v-model:value="shareForm.sni" />
        </n-form-item>
        <n-form-item :label="t('users.path')">
          <n-input v-model:value="shareForm.path" />
        </n-form-item>
        <n-button type="primary" :loading="generating" block @click="generateLink">
          {{ t('users.generateLink') }}
        </n-button>
        <template v-if="generatedLink">
          <n-divider />
          <n-input :value="generatedLink" type="textarea" readonly :rows="3" />
          <n-space justify="space-between" align="center" style="margin-top: 12px">
            <n-button size="small" @click="copyLink">{{ t('common.copy') }}</n-button>
            <QrcodeVue :value="generatedLink" :size="160" level="M" />
          </n-space>
        </template>
      </n-form>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import {
  NSpace,
  NButton,
  NDataTable,
  NModal,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NPopconfirm,
  NDivider,
  useMessage,
  type DataTableColumns
} from 'naive-ui'
import QrcodeVue from 'qrcode.vue'
import { useI18n } from 'vue-i18n'
import { usersAPI, handlerAPI, shareAPI } from '@/api/client'
import { formatBytes, copyToClipboard } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()

const users = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const generating = ref(false)
const showEditor = ref(false)
const showShare = ref(false)
const mode = ref<'add' | 'edit'>('add')
const editingContext = ref({ inboundTag: '', email: '' })
const generatedLink = ref('')

const userForm = ref({
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
  credential: '',
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

const editorTitle = computed(() => mode.value === 'edit' ? t('users.editUser') : t('users.addUser'))

const columns: DataTableColumns = [
  { title: t('users.email'), key: 'email' },
  { title: t('users.inboundTag'), key: 'inboundTag' },
  { title: t('users.protocol'), key: 'protocol', render: (row: any) => row.protocol || '-' },
  { title: t('users.level'), key: 'level' },
  {
    title: t('users.onlineStatus'),
    key: 'online',
    render: (row: any) => row.online ? t('common.online') : t('common.offline')
  },
  {
    title: t('users.uplink'),
    key: 'uplink',
    render: (row: any) => formatBytes(row.uplink || 0)
  },
  {
    title: t('users.downlink'),
    key: 'downlink',
    render: (row: any) => formatBytes(row.downlink || 0)
  },
  {
    title: t('common.actions'),
    key: 'actions',
    render(row: any) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => openEditDialog(row) }, { default: () => t('common.edit') }),
          h(NPopconfirm, {
            onPositiveClick: () => handleResetTraffic(row)
          }, {
            trigger: () => h(NButton, { size: 'small' }, { default: () => t('users.resetTraffic') }),
            default: () => t('users.resetTrafficConfirm')
          }),
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

function resetUserForm() {
  userForm.value = {
    inboundTag: '',
    email: '',
    level: 0,
    accountType: '',
    account: '{}'
  }
}

function formatJson(value: any) {
  return JSON.stringify(value ?? {}, null, 2)
}

function normalizedProtocol(protocol?: string) {
  if (protocol === 'shadowsocks' || protocol === 'shadowsocks_2022') {
    return 'ss'
  }
  if (protocol === 'vless' || protocol === 'vmess' || protocol === 'trojan' || protocol === 'ss') {
    return protocol
  }
  return 'vless'
}

function extractCredential(row: any) {
  const account = row?.account ?? {}
  return account.id || account.password || ''
}

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

function openAddDialog() {
  mode.value = 'add'
  editingContext.value = { inboundTag: '', email: '' }
  resetUserForm()
  showEditor.value = true
}

function openEditDialog(row: any) {
  mode.value = 'edit'
  editingContext.value = {
    inboundTag: row.inboundTag,
    email: row.email
  }
  userForm.value = {
    inboundTag: row.inboundTag,
    email: row.email,
    level: row.level ?? 0,
    accountType: row.accountType || '',
    account: formatJson(row.account)
  }
  showEditor.value = true
}

async function handleSubmit() {
  saving.value = true
  try {
    const account = JSON.parse(userForm.value.account || '{}')
    const payload = {
      email: userForm.value.email.trim(),
      level: userForm.value.level || 0,
      accountType: userForm.value.accountType.trim(),
      account
    }

    if (mode.value === 'edit') {
      await handlerAPI.updateInboundUser(
        editingContext.value.inboundTag,
        editingContext.value.email,
        payload
      )
    } else {
      await handlerAPI.addInboundUser(userForm.value.inboundTag.trim(), payload)
    }

    message.success(t('common.success'))
    showEditor.value = false
    await fetchUsers()
  } catch (err: any) {
    message.error(err?.error || t('users.invalidAccount'))
  } finally {
    saving.value = false
  }
}

async function handleResetTraffic(row: any) {
  try {
    await usersAPI.resetTraffic(row.email)
    message.success(t('common.success'))
    await fetchUsers()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
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
    await fetchUsers()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

function openShareDialog(row: any) {
  generatedLink.value = ''
  shareForm.value = {
    protocol: normalizedProtocol(row.protocol),
    address: '',
    port: 443,
    credential: extractCredential(row),
    type: 'tcp',
    tls: 'tls',
    sni: '',
    path: ''
  }
  showShare.value = true
}

async function generateLink() {
  generating.value = true
  try {
    const resp = await shareAPI.generate({
      protocol: shareForm.value.protocol,
      address: shareForm.value.address,
      port: shareForm.value.port,
      uuid: shareForm.value.credential,
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
