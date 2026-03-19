<template>
  <n-space vertical :size="16">
    <h2>{{ t('config.title') }}</h2>

    <n-tabs v-model:value="activeTab" type="line">
      <!-- Raw Editor -->
      <n-tab-pane name="editor" :tab="t('config.editor')">
        <n-card size="small">
          <n-space vertical :size="12">
            <n-space>
              <n-button type="primary" @click="handleSave" :loading="saving">{{ t('config.save') }}</n-button>
              <n-button @click="handleValidate" :loading="validating">{{ t('config.validate') }}</n-button>
              <n-button type="warning" @click="handleReload" :loading="reloading">{{ t('config.reload') }}</n-button>
              <n-button @click="loadConfig">{{ t('common.refresh') }}</n-button>
            </n-space>
            <n-input
              v-model:value="configText"
              type="textarea"
              :rows="30"
              :loading="loadingConfig"
              style="font-family: monospace; font-size: 13px"
              placeholder="Loading..."
            />
          </n-space>
        </n-card>
      </n-tab-pane>

      <!-- Import/Export -->
      <n-tab-pane name="importExport" :tab="t('config.importExport')">
        <n-card size="small">
          <n-space :size="12">
            <n-button @click="handleExport">{{ t('config.exportConfig') }}</n-button>
            <n-upload
              :custom-request="handleImport"
              :show-file-list="false"
              accept=".json"
            >
              <n-button>{{ t('config.importConfig') }}</n-button>
            </n-upload>
          </n-space>
        </n-card>
      </n-tab-pane>

      <!-- Backup/Restore -->
      <n-tab-pane name="backup" :tab="t('config.backup')">
        <n-card size="small">
          <n-space vertical :size="12">
            <n-space>
              <n-button type="primary" @click="handleCreateBackup">{{ t('config.createBackup') }}</n-button>
              <n-button @click="fetchBackups">{{ t('common.refresh') }}</n-button>
            </n-space>
            <n-data-table :columns="backupColumns" :data="backups" :loading="loadingBackups" size="small" />
          </n-space>
        </n-card>
      </n-tab-pane>
    </n-tabs>
  </n-space>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NSpace, NTabs, NTabPane, NCard, NButton, NInput, NDataTable, NUpload, NPopconfirm, useMessage, type DataTableColumns, type UploadCustomRequestOptions } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { configAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const activeTab = ref('editor')
const configText = ref('')
const loadingConfig = ref(false)
const saving = ref(false)
const validating = ref(false)
const reloading = ref(false)
const backups = ref<any[]>([])
const loadingBackups = ref(false)

const backupColumns: DataTableColumns = [
  { title: 'Name', key: 'name' },
  { title: 'Size', key: 'size', render: (row: any) => `${(row.size / 1024).toFixed(1)} KB` },
  { title: 'Modified', key: 'modified' },
  {
    title: t('common.actions'), key: 'actions',
    render(row: any) {
      return h(NPopconfirm, {
        onPositiveClick: () => handleRestoreBackup(row.name)
      }, {
        trigger: () => h(NButton, { size: 'small' }, { default: () => t('config.restoreBackup') }),
        default: () => `Restore ${row.name}?`
      })
    }
  }
]

async function loadConfig() {
  loadingConfig.value = true
  try {
    const data = await configAPI.get()
    if (data.config) {
      const config = typeof data.config === 'string' ? data.config : JSON.stringify(data.config, null, 2)
      configText.value = typeof config === 'string' ? config : JSON.stringify(config, null, 2)
    }
  } catch (err: any) {
    message.error('Failed to load config')
  } finally {
    loadingConfig.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    const config = JSON.parse(configText.value)
    await configAPI.save(config)
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(err?.error || 'Invalid JSON or save failed')
  } finally {
    saving.value = false
  }
}

async function handleValidate() {
  validating.value = true
  try {
    const config = JSON.parse(configText.value)
    const data = await configAPI.validate(config)
    if (data.valid) {
      message.success('Config is valid')
    } else {
      message.warning('Config validation: ' + data.error)
    }
  } catch (err: any) {
    message.error(err?.error || 'Invalid JSON')
  } finally {
    validating.value = false
  }
}

async function handleReload() {
  reloading.value = true
  try {
    await configAPI.reload()
    message.success('Reload requested')
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    reloading.value = false
  }
}

function handleExport() {
  const blob = new Blob([configText.value], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'xray-config.json'
  a.click()
  URL.revokeObjectURL(url)
}

function handleImport({ file }: UploadCustomRequestOptions) {
  if (!file.file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    try {
      const text = e.target?.result as string
      JSON.parse(text) // validate JSON
      configText.value = text
      message.success('Config imported. Click Save to apply.')
    } catch {
      message.error('Invalid JSON file')
    }
  }
  reader.readAsText(file.file)
}

async function fetchBackups() {
  loadingBackups.value = true
  try {
    const data = await configAPI.listBackups()
    backups.value = data.backups || []
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    loadingBackups.value = false
  }
}

async function handleCreateBackup() {
  try {
    await configAPI.createBackup()
    message.success(t('common.success'))
    fetchBackups()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

async function handleRestoreBackup(name: string) {
  try {
    await configAPI.restoreBackup(name)
    message.success(t('common.success'))
    loadConfig()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

onMounted(() => {
  loadConfig()
  fetchBackups()
})
</script>
