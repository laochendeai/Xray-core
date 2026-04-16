<template>
  <n-space vertical :size="16" class="privacy-page">
    <div>
      <h2 class="page-title">{{ t('privacy.title') }}</h2>
      <div class="page-subtitle">{{ t('privacy.subtitle') }}</div>
    </div>

    <n-alert type="info" :title="t('privacy.firstPhaseTitle')">
      {{ t('privacy.firstPhaseBody') }}
    </n-alert>

    <n-card size="small" :title="t('privacy.runtimeContextTitle')">
      <n-space vertical :size="12">
        <n-alert v-if="loadError" type="error">{{ loadError }}</n-alert>
        <n-alert v-else-if="context && !context.supported" type="warning">
          {{ context.unsupportedReason || t('privacy.unsupported') }}
        </n-alert>
        <n-descriptions v-else bordered :column="1">
          <n-descriptions-item :label="t('privacy.tunRunning')">
            {{ context?.tunStatus?.running ? t('common.enabled') : t('common.disabled') }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('privacy.routeMode')">
            {{ context?.tunSettings?.routeMode || '-' }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('privacy.remoteDns')">
            {{ (context?.tunSettings?.remoteDns || []).join(', ') || '-' }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('privacy.directEgress')">
            {{ context?.tunStatus?.directEgress?.ip || '-' }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('privacy.proxyEgress')">
            {{ context?.tunStatus?.proxyEgress?.ip || '-' }}
          </n-descriptions-item>
        </n-descriptions>
      </n-space>
    </n-card>

    <n-card size="small" :title="t('privacy.dnsTitle')">
      <n-space vertical :size="12">
        <n-button @click="runDnsCheck" :loading="runningDns">{{ t('privacy.runDnsCheck') }}</n-button>
        <n-alert :type="dnsAlertType" :title="dnsAlertTitle">
          {{ dnsSummary }}
        </n-alert>
        <n-list bordered>
          <n-list-item v-for="note in dnsResult.notes" :key="note">{{ note }}</n-list-item>
        </n-list>
      </n-space>
    </n-card>

    <n-card size="small" :title="t('privacy.webrtcTitle')">
      <n-space vertical :size="12">
        <n-button @click="runWebRTCCheck" :loading="runningWebRTC">{{ t('privacy.runWebRtcCheck') }}</n-button>
        <n-alert :type="webrtcAlertType" :title="webrtcAlertTitle">
          {{ webrtcSummary }}
        </n-alert>
        <n-list bordered>
          <n-list-item v-for="candidate in webrtcResult.candidates" :key="candidate.candidate">
            {{ candidate.type }} · {{ candidate.protocol }} · {{ candidate.address }}<span v-if="candidate.port">:{{ candidate.port }}</span>
          </n-list-item>
        </n-list>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NList,
  NListItem,
  NSpace,
  useMessage
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { privacyAPI } from '@/api/client'
import type {
  PrivacyDiagnosticsContextResponse,
  PrivacyDnsResult,
  PrivacyWebRTCCandidate,
  PrivacyWebRTCResult
} from '@/api/types'

const { t } = useI18n()
const message = useMessage()

const context = ref<PrivacyDiagnosticsContextResponse | null>(null)
const loadError = ref('')
const runningDns = ref(false)
const runningWebRTC = ref(false)

const dnsResult = ref<PrivacyDnsResult>({
  leakRisk: 'unknown',
  expectedRemoteDns: [],
  tunRunning: false,
  routeMode: 'strict_proxy',
  notes: []
})

const webrtcResult = ref<PrivacyWebRTCResult>({
  supported: false,
  gathered: false,
  leakRisk: 'unknown',
  exposedPrivateAddress: false,
  exposedPublicAddress: false,
  candidates: []
})

async function loadContext() {
  loadError.value = ''
  try {
    context.value = await privacyAPI.getContext()
  } catch (err: any) {
    loadError.value = err?.error || err?.message || t('common.error')
  }
}

async function runDnsCheck() {
  runningDns.value = true
  try {
    const tunRunning = !!context.value?.tunStatus?.running
    const routeMode = context.value?.tunSettings?.routeMode || 'strict_proxy'
    const expectedRemoteDns = context.value?.tunSettings?.remoteDns || []
    const notes: string[] = []
    notes.push(tunRunning ? t('privacy.dnsTunRunning') : t('privacy.dnsTunStopped'))
    if (expectedRemoteDns.length) {
      notes.push(t('privacy.dnsExpectedResolvers', { resolvers: expectedRemoteDns.join(', ') }))
    }
    if (routeMode === 'auto_tested') {
      notes.push(t('privacy.dnsAutoTestedHint'))
    } else {
      notes.push(t('privacy.dnsStrictProxyHint'))
    }
    dnsResult.value = {
      leakRisk: tunRunning ? 'warning' : 'unknown',
      expectedRemoteDns,
      tunRunning,
      routeMode,
      notes
    }
  } finally {
    runningDns.value = false
  }
}

function extractAddress(candidate: string) {
  const parts = candidate.trim().split(/\s+/)
  return parts[4] || ''
}

function extractPort(candidate: string) {
  const parts = candidate.trim().split(/\s+/)
  const raw = parts[5]
  const value = Number(raw)
  return Number.isFinite(value) ? value : null
}

function isPrivateAddress(address: string) {
  return /^(10\.|192\.168\.|172\.(1[6-9]|2\d|3[01])\.|127\.|169\.254\.|fc|fd|fe80:|::1)/i.test(address)
}

async function runWebRTCCheck() {
  runningWebRTC.value = true
  try {
    const RTCPeer = (window as any).RTCPeerConnection || (window as any).webkitRTCPeerConnection
    if (!RTCPeer) {
      webrtcResult.value = {
        supported: false,
        gathered: false,
        leakRisk: 'unknown',
        exposedPrivateAddress: false,
        exposedPublicAddress: false,
        candidates: [],
        error: t('privacy.webrtcUnsupported')
      }
      return
    }

    const candidates: PrivacyWebRTCCandidate[] = []
    const seen = new Set<string>()
    const pc = new RTCPeer({ iceServers: [{ urls: ['stun:stun.l.google.com:19302'] }] })
    pc.createDataChannel('probe')

    pc.onicecandidate = (event: RTCPeerConnectionIceEvent) => {
      const raw = event.candidate?.candidate
      if (!raw || seen.has(raw)) return
      seen.add(raw)
      const address = extractAddress(raw)
      const protocol = raw.includes(' udp ') ? 'udp' : raw.includes(' tcp ') ? 'tcp' : 'unknown'
      const typeMatch = raw.match(/ typ (host|srflx|relay|prflx)/)
      candidates.push({
        candidate: raw,
        type: typeMatch?.[1] || 'unknown',
        protocol,
        address,
        port: extractPort(raw),
        isPrivateAddress: isPrivateAddress(address)
      })
    }

    const offer = await pc.createOffer()
    await pc.setLocalDescription(offer)
    await new Promise((resolve) => setTimeout(resolve, 2500))
    pc.close()

    const exposedPrivateAddress = candidates.some((candidate) => candidate.isPrivateAddress)
    const exposedPublicAddress = candidates.some((candidate) => candidate.type === 'srflx')
    const leakRisk = exposedPrivateAddress || exposedPublicAddress ? 'warning' : candidates.length ? 'low' : 'unknown'

    webrtcResult.value = {
      supported: true,
      gathered: true,
      leakRisk,
      exposedPrivateAddress,
      exposedPublicAddress,
      candidates
    }
  } catch (err: any) {
    webrtcResult.value = {
      supported: true,
      gathered: false,
      leakRisk: 'unknown',
      exposedPrivateAddress: false,
      exposedPublicAddress: false,
      candidates: [],
      error: err?.message || t('common.error')
    }
    message.error(err?.message || t('common.error'))
  } finally {
    runningWebRTC.value = false
  }
}

const dnsAlertType = computed(() => dnsResult.value.leakRisk === 'warning' ? 'warning' : dnsResult.value.leakRisk === 'low' ? 'success' : 'info')
const dnsAlertTitle = computed(() => dnsResult.value.leakRisk === 'warning' ? t('privacy.dnsWarningTitle') : dnsResult.value.leakRisk === 'low' ? t('privacy.dnsLowRiskTitle') : t('privacy.dnsUnknownTitle'))
const dnsSummary = computed(() => {
  if (!dnsResult.value.notes.length) return t('privacy.dnsSummaryIdle')
  return dnsResult.value.notes.join(' · ')
})

const webrtcAlertType = computed(() => webrtcResult.value.leakRisk === 'warning' ? 'warning' : webrtcResult.value.leakRisk === 'low' ? 'success' : 'info')
const webrtcAlertTitle = computed(() => webrtcResult.value.leakRisk === 'warning' ? t('privacy.webrtcWarningTitle') : webrtcResult.value.leakRisk === 'low' ? t('privacy.webrtcLowRiskTitle') : t('privacy.webrtcUnknownTitle'))
const webrtcSummary = computed(() => {
  if (webrtcResult.value.error) return webrtcResult.value.error
  if (!webrtcResult.value.gathered) return t('privacy.webrtcSummaryIdle')
  const notes = []
  notes.push(t('privacy.webrtcCandidateCount', { count: webrtcResult.value.candidates.length }))
  if (webrtcResult.value.exposedPrivateAddress) notes.push(t('privacy.webrtcPrivateExposed'))
  if (webrtcResult.value.exposedPublicAddress) notes.push(t('privacy.webrtcPublicExposed'))
  if (!webrtcResult.value.exposedPrivateAddress && !webrtcResult.value.exposedPublicAddress) notes.push(t('privacy.webrtcNoExposure'))
  return notes.join(' · ')
})

onMounted(async () => {
  await loadContext()
})
</script>

<style scoped>
.privacy-page {
  padding-bottom: 24px;
}
.page-title {
  margin: 0;
}
.page-subtitle {
  margin-top: 6px;
  color: var(--n-text-color-3);
}
</style>
