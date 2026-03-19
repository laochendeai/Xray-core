<template>
  <n-space vertical :size="16">
    <h2>{{ t('routing.title') }}</h2>

    <n-tabs v-model:value="activeTab" type="line">
      <!-- Rules Tab -->
      <n-tab-pane name="rules" :tab="t('routing.rules')">
        <n-space vertical :size="12">
          <n-space justify="end">
            <n-button type="primary" size="small" @click="showAddRule = true">{{ t('routing.addRule') }}</n-button>
            <n-button size="small" @click="fetchRules">{{ t('common.refresh') }}</n-button>
          </n-space>
          <n-data-table :columns="ruleColumns" :data="rules" :loading="loadingRules" size="small" />
        </n-space>
      </n-tab-pane>

      <!-- Balancers Tab -->
      <n-tab-pane name="balancers" :tab="t('routing.balancers')">
        <n-space vertical :size="12">
          <n-form inline>
            <n-form-item label="Balancer Tag">
              <n-input v-model:value="balancerTag" placeholder="balancer-tag" />
            </n-form-item>
            <n-form-item>
              <n-button type="primary" @click="fetchBalancer">{{ t('common.search') }}</n-button>
            </n-form-item>
          </n-form>
          <n-card v-if="balancerInfo" size="small">
            <pre>{{ JSON.stringify(balancerInfo, null, 2) }}</pre>
          </n-card>
        </n-space>
      </n-tab-pane>

      <!-- Route Test Tab -->
      <n-tab-pane name="test" :tab="t('routing.test')">
        <n-form :model="testForm" label-placement="left" label-width="100px">
          <n-grid :cols="2" :x-gap="12">
            <n-gi>
              <n-form-item :label="t('routing.domain')">
                <n-input v-model:value="testForm.domain" placeholder="example.com" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('routing.ip')">
                <n-input v-model:value="testForm.ip" placeholder="1.1.1.1" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('routing.port')">
                <n-input-number v-model:value="testForm.port" :min="1" :max="65535" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('routing.network')">
                <n-select v-model:value="testForm.network" :options="[{label:'TCP',value:'tcp'},{label:'UDP',value:'udp'}]" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="Source IP">
                <n-input v-model:value="testForm.sourceIP" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item label="Inbound Tag">
                <n-input v-model:value="testForm.inboundTag" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-button type="primary" @click="handleTestRoute" :loading="testing">
            {{ t('routing.testRoute') }}
          </n-button>
        </n-form>
        <n-card v-if="testResult" size="small" style="margin-top: 16px">
          <pre>{{ JSON.stringify(testResult, null, 2) }}</pre>
        </n-card>
      </n-tab-pane>
    </n-tabs>

    <!-- Add Rule Dialog -->
    <n-modal v-model:show="showAddRule" preset="dialog" :title="t('routing.addRule')" style="width: 500px">
      <n-form-item label="Rule (JSON)">
        <n-input v-model:value="newRule" type="textarea" :rows="8" placeholder='{"ruleTag":"my-rule", ...}' />
      </n-form-item>
      <template #action>
        <n-button @click="showAddRule = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" @click="handleAddRule">{{ t('common.confirm') }}</n-button>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NSpace, NTabs, NTabPane, NDataTable, NButton, NModal, NForm, NFormItem, NInput, NInputNumber, NSelect, NCard, NGrid, NGi, NPopconfirm, useMessage, type DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { routingAPI } from '@/api/client'

const { t } = useI18n()
const message = useMessage()

const activeTab = ref('rules')
const rules = ref<any[]>([])
const loadingRules = ref(false)
const showAddRule = ref(false)
const newRule = ref('{}')
const balancerTag = ref('')
const balancerInfo = ref<any>(null)
const testing = ref(false)
const testResult = ref<any>(null)

const testForm = ref({
  domain: '',
  ip: '',
  port: 443,
  network: 'tcp',
  sourceIP: '',
  inboundTag: ''
})

const ruleColumns: DataTableColumns = [
  { title: 'Tag', key: 'tag' },
  { title: t('routing.ruleTag'), key: 'ruleTag' },
  {
    title: t('common.actions'), key: 'actions',
    render(row: any) {
      return h(NPopconfirm, {
        onPositiveClick: () => handleDeleteRule(row.ruleTag || row.tag)
      }, {
        trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
        default: () => t('common.confirm') + '?'
      })
    }
  }
]

async function fetchRules() {
  loadingRules.value = true
  try {
    const data = await routingAPI.listRules()
    rules.value = data.rules || []
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    loadingRules.value = false
  }
}

async function handleAddRule() {
  try {
    await routingAPI.addRule(JSON.parse(newRule.value))
    message.success(t('common.success'))
    showAddRule.value = false
    fetchRules()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

async function handleDeleteRule(tag: string) {
  try {
    await routingAPI.removeRule(tag)
    message.success(t('common.success'))
    fetchRules()
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

async function fetchBalancer() {
  if (!balancerTag.value) return
  try {
    const data = await routingAPI.getBalancer(balancerTag.value)
    balancerInfo.value = data.balancer
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  }
}

async function handleTestRoute() {
  testing.value = true
  try {
    const data = await routingAPI.testRoute(testForm.value)
    testResult.value = data.result
  } catch (err: any) {
    message.error(err?.error || t('common.error'))
  } finally {
    testing.value = false
  }
}

onMounted(fetchRules)
</script>
