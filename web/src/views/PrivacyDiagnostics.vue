<template>
  <n-space vertical :size="16" class="privacy-page">
    <div>
      <h2 class="page-title">{{ t("privacy.title") }}</h2>
      <div class="page-subtitle">{{ t("privacy.subtitle") }}</div>
    </div>

    <n-alert type="info" :title="t('privacy.ippureGateTitle')">
      {{ t("privacy.ippureGateBody") }}
    </n-alert>

    <n-card size="small" :title="t('privacy.hardeningCenterTitle')">
      <n-space vertical :size="12">
        <n-alert type="info" :title="t('privacy.hardeningCenterIntroTitle')">
          {{ t("privacy.hardeningCenterIntroBody") }}
        </n-alert>
        <n-alert v-if="hardeningError" type="error">
          {{ hardeningError }}
        </n-alert>
        <n-alert v-if="hardeningActionMessage" :type="hardeningActionType">
          {{ hardeningActionMessage }}
        </n-alert>
        <n-descriptions bordered :column="1" size="small">
          <n-descriptions-item :label="t('privacy.hardeningPlatform')">
            {{ hardening?.platform || "-" }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('privacy.browserPolicyStatus')">
            {{ browserPolicySummary }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('privacy.controlledBrowserStatus')">
            {{ controlledBrowserSummary }}
          </n-descriptions-item>
          <n-descriptions-item :label="t('privacy.dailyBrowserFingerprintStatus')">
            {{ dailyBrowserFingerprintSummary }}
          </n-descriptions-item>
        </n-descriptions>
        <n-space>
          <n-button :loading="loadingHardening" @click="loadHardeningStatus">
            {{ t("privacy.refreshHardeningStatus") }}
          </n-button>
          <n-button
            type="primary"
            :loading="installingBrowserPolicy"
            :disabled="!hardening?.browserPolicy.canInstall"
            @click="installBrowserPolicy"
          >
            {{ t("privacy.installBrowserPolicy") }}
          </n-button>
          <n-button
            type="primary"
            secondary
            :loading="openingControlledBrowser"
            :disabled="!hardening?.controlledBrowser.available"
            @click="openControlledBrowser"
          >
            {{ t("privacy.openControlledBrowser") }}
          </n-button>
        </n-space>
        <n-alert v-if="browserPolicyHint" type="warning" :title="t('privacy.browserPolicyHintTitle')">
          {{ browserPolicyHint }}
        </n-alert>
        <n-alert v-if="controlledBrowserHint" type="warning" :title="t('privacy.controlledBrowserHintTitle')">
          {{ controlledBrowserHint }}
        </n-alert>
        <n-list v-if="hardening?.browserPolicy.targets.length" bordered>
          <n-list-item v-for="target in hardening.browserPolicy.targets" :key="target.browser">
            {{ target.browser }} ·
            {{ target.detected ? t("privacy.browserDetected") : t("privacy.browserNotDetected") }} ·
            {{ target.configured ? t("privacy.policyConfigured") : t("privacy.policyMissing") }}
          </n-list-item>
        </n-list>
      </n-space>
    </n-card>

    <n-space>
      <n-button type="primary" :loading="runningAll" @click="runAllChecks">
        {{ t("privacy.runAllChecks") }}
      </n-button>
      <n-button :loading="loadingContext" @click="loadAllContext">
        {{ t("privacy.refreshContext") }}
      </n-button>
    </n-space>

    <n-alert v-if="loadError" type="error">{{ loadError }}</n-alert>
    <n-alert v-else-if="context && !context.supported" type="warning">
      {{ context.unsupportedReason || t("privacy.unsupported") }}
    </n-alert>

    <n-card size="small" :title="t('privacy.runtimeContextTitle')">
      <n-descriptions bordered :column="1">
        <n-descriptions-item :label="t('privacy.tunRunning')">
          {{ context?.tunStatus?.running ? t("common.enabled") : t("common.disabled") }}
        </n-descriptions-item>
        <n-descriptions-item :label="t('privacy.routeMode')">
          {{ routeModeLabel(context?.tunSettings?.routeMode || context?.tunStatus?.status || "") }}
        </n-descriptions-item>
        <n-descriptions-item :label="t('privacy.remoteDns')">
          {{ (context?.tunSettings?.remoteDns || context?.tunStatus?.remoteDns || []).join(", ") || "-" }}
        </n-descriptions-item>
        <n-descriptions-item :label="t('privacy.directEgress')">
          {{ context?.tunStatus?.directEgress?.ip || "-" }}
        </n-descriptions-item>
        <n-descriptions-item :label="t('privacy.proxyEgress')">
          {{ context?.tunStatus?.proxyEgress?.ip || "-" }}
        </n-descriptions-item>
      </n-descriptions>
    </n-card>

    <n-grid :cols="2" :x-gap="16" :y-gap="16" responsive="screen">
      <n-gi>
        <n-card size="small">
          <template #header>
            <div class="card-header">
              <span>{{ t("privacy.ipTitle") }}</span>
              <n-tag :type="riskTagType(ipExposure.leakRisk)">
                {{ riskLabel(ipExposure.leakRisk) }}
              </n-tag>
            </div>
          </template>
          <n-space vertical :size="10">
            <n-button size="small" :loading="runningIp" @click="runIpCheck">
              {{ t("privacy.runIpCheck") }}
            </n-button>
            <n-alert :type="riskAlertType(ipExposure.leakRisk)" :title="ipExposureTitle">
              {{ ipExposureSummary }}
            </n-alert>
            <n-alert type="info" :title="t('privacy.ipProtectionTitle')">
              {{ t("privacy.ipProtectionBody") }}
            </n-alert>
            <n-descriptions bordered :column="1" size="small">
              <n-descriptions-item :label="t('privacy.browserEgress')">
                {{ ipExposure.browserIp || "-" }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('privacy.directEgress')">
                {{ ipExposure.directIp || "-" }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('privacy.proxyEgress')">
                {{ ipExposure.proxyIp || "-" }}
              </n-descriptions-item>
            </n-descriptions>
          </n-space>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card size="small">
          <template #header>
            <div class="card-header">
              <span>{{ t("privacy.dnsTitle") }}</span>
              <n-tag :type="riskTagType(dnsResult.leakRisk)">
                {{ riskLabel(dnsResult.leakRisk) }}
              </n-tag>
            </div>
          </template>
          <n-space vertical :size="10">
            <n-alert :type="riskAlertType(dnsResult.leakRisk)" :title="dnsAlertTitle">
              {{ dnsSummary }}
            </n-alert>
            <n-alert type="info" :title="t('privacy.dnsProtectionTitle')">
              {{ t("privacy.dnsProtectionBody") }}
            </n-alert>
            <n-list bordered>
              <n-list-item v-for="note in dnsResult.notes" :key="note">
                {{ note }}
              </n-list-item>
            </n-list>
            <n-list v-if="routingDiagnostics.length" bordered>
              <n-list-item
                v-for="diagnostic in routingDiagnostics"
                :key="`${diagnostic.category}-${diagnostic.dnsPath}`"
              >
                {{ routingDiagnosticLabel(diagnostic.category) }} ·
                {{ dnsPathLabel(diagnostic.dnsPath) }} ·
                {{ routeLabel(diagnostic.route) }} ·
                {{ diagnostic.resolver || "-" }}
              </n-list-item>
            </n-list>
          </n-space>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card size="small">
          <template #header>
            <div class="card-header">
              <span>{{ t("privacy.webrtcTitle") }}</span>
              <n-tag :type="riskTagType(webrtcResult.leakRisk)">
                {{ riskLabel(webrtcResult.leakRisk) }}
              </n-tag>
            </div>
          </template>
          <n-space vertical :size="10">
            <n-button size="small" :loading="runningWebRTC" @click="runWebRTCCheck">
              {{ t("privacy.runWebRtcCheck") }}
            </n-button>
            <n-alert :type="riskAlertType(webrtcResult.leakRisk)" :title="webrtcAlertTitle">
              {{ webrtcSummary }}
            </n-alert>
            <n-alert type="info" :title="t('privacy.webrtcProtectionTitle')">
              {{ t("privacy.webrtcProtectionBody") }}
            </n-alert>
            <n-list v-if="webrtcResult.candidates.length" bordered>
              <n-list-item v-for="candidate in webrtcResult.candidates" :key="candidate.candidate">
                {{ candidateTypeLabel(candidate.type) }} ·
                {{ protocolLabel(candidate.protocol) }} ·
                {{ candidate.address || "-" }}<span v-if="candidate.port">:{{ candidate.port }}</span>
              </n-list-item>
            </n-list>
          </n-space>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card size="small">
          <template #header>
            <div class="card-header">
              <span>{{ t("privacy.fingerprintTitle") }}</span>
              <n-tag :type="riskTagType(fingerprintRisk.leakRisk)">
                {{ fingerprintRiskLabel(fingerprintRisk.leakRisk) }}
              </n-tag>
            </div>
          </template>
          <n-space vertical :size="10">
            <n-button size="small" :loading="runningFingerprint" @click="runFingerprintCheck">
              {{ t("privacy.runFingerprintCheck") }}
            </n-button>
            <n-alert :type="riskAlertType(fingerprintRisk.leakRisk)" :title="fingerprintAlertTitle">
              {{ fingerprintSummary }}
            </n-alert>
            <n-alert type="info" :title="t('privacy.fingerprintProtectionTitle')">
              {{ t("privacy.fingerprintProtectionBody") }}
            </n-alert>
            <n-descriptions v-if="fingerprintSnapshot" bordered :column="1" size="small">
              <n-descriptions-item :label="t('privacy.fingerprintTimezone')">
                {{ fingerprintSnapshot.timezone || "-" }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('privacy.fingerprintLanguages')">
                {{ fingerprintSnapshot.languages.join(", ") || "-" }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('privacy.fingerprintScreen')">
                {{ fingerprintSnapshot.screen.width }}x{{ fingerprintSnapshot.screen.height }}
                @ {{ fingerprintSnapshot.screen.devicePixelRatio }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('privacy.fingerprintWebGL')">
                {{ fingerprintSnapshot.webglVendor || "-" }} · {{ fingerprintSnapshot.webglRenderer || "-" }}
              </n-descriptions-item>
            </n-descriptions>
          </n-space>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card size="small">
          <template #header>
            <div class="card-header">
              <span>{{ t("privacy.cleanlinessTitle") }}</span>
              <n-tag :type="riskTagType(cleanlinessRisk)">
                {{ riskLabel(cleanlinessRisk) }}
              </n-tag>
            </div>
          </template>
          <n-space vertical :size="10">
            <n-alert :type="riskAlertType(cleanlinessRisk)" :title="cleanlinessAlertTitle">
              {{ cleanlinessSummaryText }}
            </n-alert>
            <n-descriptions bordered :column="1" size="small">
              <n-descriptions-item :label="t('nodePool.status.active')">
                {{ activeNodes.length }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('privacy.localIntelligenceChecked')">
                {{ activeIntelligenceCheckedCount }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('nodePool.cleanliness.trusted')">
                {{ intelligenceSummary.trustedCount }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('nodePool.cleanliness.suspicious')">
                {{ intelligenceSummary.suspiciousCount }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('nodePool.cleanliness.unknown')">
                {{ intelligenceSummary.unknownCleanCount }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('nodePool.networkType.residential_likely')">
                {{ intelligenceSummary.residentialCount }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('nodePool.networkType.isp_likely')">
                {{ intelligenceSummary.ispLikeCount }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('nodePool.networkType.datacenter_likely')">
                {{ intelligenceSummary.datacenterCount }}
              </n-descriptions-item>
              <n-descriptions-item :label="t('nodePool.networkType.unknown')">
                {{ intelligenceSummary.unknownNetworkCount }}
              </n-descriptions-item>
            </n-descriptions>
          </n-space>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card size="small">
          <template #header>
            <div class="card-header">
              <span>{{ t("privacy.dedupeTitle") }}</span>
              <n-tag :type="riskTagType(dedupeRisk)">
                {{ riskLabel(dedupeRisk) }}
              </n-tag>
            </div>
          </template>
          <n-alert :type="riskAlertType(dedupeRisk)" :title="dedupeAlertTitle">
            {{ dedupeSummary }}
          </n-alert>
        </n-card>
      </n-gi>
    </n-grid>
  </n-space>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  NAlert,
  NButton,
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NGi,
  NGrid,
  NList,
  NListItem,
  NSpace,
  NTag,
  useMessage,
} from "naive-ui";
import { useI18n } from "vue-i18n";
import { nodePoolAPI, privacyAPI } from "@/api/client";
import type {
  NodeRecord,
  PrivacyDiagnosticsContextResponse,
  PrivacyDnsResult,
  PrivacyFingerprintSnapshot,
  PrivacyHardeningStatusResponse,
  PrivacyWebRTCCandidate,
  PrivacyWebRTCResult,
  TunRoutingDiagnostic,
} from "@/api/types";
import { summarizeNodeIntelligence } from "@/utils/nodePool";
import {
  classifyFingerprintRisk,
  classifyIpExposure,
  classifyRuntimeDnsRisk,
  classifyWebRTCRisk,
  collectFingerprintSnapshot,
  parseIceCandidate,
  type PrivacyRiskLevel,
} from "@/utils/privacyDiagnostics";

const { t } = useI18n();
const message = useMessage();

const context = ref<PrivacyDiagnosticsContextResponse | null>(null);
const hardening = ref<PrivacyHardeningStatusResponse | null>(null);
const loadError = ref("");
const hardeningError = ref("");
const hardeningActionMessage = ref("");
const hardeningActionType = ref<"success" | "info" | "warning" | "error">("info");
const loadingContext = ref(false);
const loadingHardening = ref(false);
const installingBrowserPolicy = ref(false);
const openingControlledBrowser = ref(false);
const runningAll = ref(false);
const runningIp = ref(false);
const runningWebRTC = ref(false);
const runningFingerprint = ref(false);
const browserPublicIp = ref("");
const browserPublicIpError = ref("");
const activeNodes = ref<NodeRecord[]>([]);
const allNodes = ref<NodeRecord[]>([]);
const fingerprintSnapshot = ref<PrivacyFingerprintSnapshot | null>(null);

const webrtcResult = ref<PrivacyWebRTCResult>({
  supported: false,
  gathered: false,
  leakRisk: "unknown",
  exposedPrivateAddress: false,
  exposedPublicAddress: false,
  candidates: [],
});

async function loadContext() {
  loadError.value = "";
  loadingContext.value = true;
  try {
    context.value = await privacyAPI.getContext();
  } catch (err: any) {
    loadError.value = err?.error || err?.message || t("common.error");
  } finally {
    loadingContext.value = false;
  }
}

async function loadNodePool() {
  try {
    const response = await nodePoolAPI.list();
    allNodes.value = response.nodes;
    activeNodes.value = response.nodes.filter((node) => node.status === "active");
  } catch {
    allNodes.value = [];
    activeNodes.value = [];
  }
}

async function loadHardeningStatus() {
  hardeningError.value = "";
  loadingHardening.value = true;
  try {
    hardening.value = await privacyAPI.getHardeningStatus();
  } catch (err: any) {
    hardeningError.value = err?.error || err?.message || t("common.error");
  } finally {
    loadingHardening.value = false;
  }
}

async function installBrowserPolicy() {
  hardeningActionMessage.value = "";
  installingBrowserPolicy.value = true;
  try {
    const response = await privacyAPI.installBrowserPolicy();
    if (response.status) hardening.value = response.status;
    hardeningActionType.value = response.ok ? "success" : "warning";
    hardeningActionMessage.value = response.message;
    message.success(response.message);
  } catch (err: any) {
    hardeningActionType.value = "error";
    hardeningActionMessage.value = err?.error || err?.message || t("common.error");
    message.error(hardeningActionMessage.value);
    await loadHardeningStatus();
  } finally {
    installingBrowserPolicy.value = false;
  }
}

async function openControlledBrowser() {
  hardeningActionMessage.value = "";
  openingControlledBrowser.value = true;
  try {
    const response = await privacyAPI.openControlledBrowser();
    if (response.status) hardening.value = response.status;
    hardeningActionType.value = response.ok ? "success" : "warning";
    hardeningActionMessage.value = response.pid
      ? t("privacy.controlledBrowserStarted", { pid: response.pid, log: response.logFile || "-" })
      : response.message;
    message.success(hardeningActionMessage.value);
  } catch (err: any) {
    hardeningActionType.value = "error";
    hardeningActionMessage.value = err?.error || err?.message || t("common.error");
    message.error(hardeningActionMessage.value);
    await loadHardeningStatus();
  } finally {
    openingControlledBrowser.value = false;
  }
}

async function loadAllContext() {
  await Promise.all([loadContext(), loadNodePool(), loadHardeningStatus()]);
}

async function fetchJsonWithTimeout(url: string, timeoutMs: number) {
  const controller = new AbortController();
  const timer = window.setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(url, {
      cache: "no-store",
      signal: controller.signal,
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    return await response.json();
  } finally {
    window.clearTimeout(timer);
  }
}

async function runIpCheck() {
  runningIp.value = true;
  browserPublicIpError.value = "";
  const endpoints = [
    "https://api64.ipify.org?format=json",
    "https://api.ipify.org?format=json",
  ];
  try {
    for (const endpoint of endpoints) {
      try {
        const result = await fetchJsonWithTimeout(endpoint, 5000);
        browserPublicIp.value = String(result.ip || "").trim();
        if (browserPublicIp.value) return;
      } catch (err: any) {
        browserPublicIpError.value = err?.message || t("common.error");
      }
    }
    throw new Error(browserPublicIpError.value || t("common.error"));
  } catch (err: any) {
    browserPublicIpError.value = err?.message || t("common.error");
    message.error(browserPublicIpError.value);
  } finally {
    runningIp.value = false;
  }
}

async function runWebRTCCheck() {
  runningWebRTC.value = true;
  try {
    const RTCPeer = window.RTCPeerConnection || (window as any).webkitRTCPeerConnection;
    if (!RTCPeer) {
      webrtcResult.value = {
        supported: false,
        gathered: false,
        leakRisk: "unknown",
        exposedPrivateAddress: false,
        exposedPublicAddress: false,
        candidates: [],
        error: t("privacy.webrtcUnsupported"),
      };
      return;
    }

    const candidates: PrivacyWebRTCCandidate[] = [];
    const seen = new Set<string>();
    const pc = new RTCPeer({ iceServers: [{ urls: ["stun:stun.l.google.com:19302"] }] });
    pc.createDataChannel("probe");

    pc.onicecandidate = (event: RTCPeerConnectionIceEvent) => {
      const raw = event.candidate?.candidate;
      if (!raw || seen.has(raw)) return;
      seen.add(raw);
      candidates.push(parseIceCandidate(raw));
    };

    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    await new Promise((resolve) => window.setTimeout(resolve, 2500));
    pc.close();

    const risk = classifyWebRTCRisk(candidates);
    webrtcResult.value = {
      supported: true,
      gathered: true,
      ...risk,
      candidates,
    };
  } catch (err: any) {
    webrtcResult.value = {
      supported: true,
      gathered: false,
      leakRisk: "unknown",
      exposedPrivateAddress: false,
      exposedPublicAddress: false,
      candidates: [],
      error: err?.message || t("common.error"),
    };
    message.error(err?.message || t("common.error"));
  } finally {
    runningWebRTC.value = false;
  }
}

async function runFingerprintCheck() {
  runningFingerprint.value = true;
  try {
    fingerprintSnapshot.value = await collectFingerprintSnapshot();
  } catch (err: any) {
    message.error(err?.message || t("common.error"));
  } finally {
    runningFingerprint.value = false;
  }
}

async function runAllChecks() {
  runningAll.value = true;
  try {
    await loadAllContext();
    await Promise.all([runIpCheck(), runWebRTCCheck(), runFingerprintCheck()]);
  } finally {
    runningAll.value = false;
  }
}

const routingDiagnostics = computed<TunRoutingDiagnostic[]>(() => context.value?.tunStatus?.routingDiagnostics || []);
const runtimeDnsRisk = computed(() => classifyRuntimeDnsRisk(context.value));
const ipExposure = computed(() => classifyIpExposure(browserPublicIp.value, context.value));
const fingerprintRisk = computed(() => classifyFingerprintRisk(fingerprintSnapshot.value));
const intelligenceSummary = computed(() => summarizeNodeIntelligence(activeNodes.value));
const activeIntelligenceCheckedCount = computed(
  () => activeNodes.value.filter((node) => !!node.intelligenceCheckedAt).length,
);

const browserPolicySummary = computed(() => {
  const policy = hardening.value?.browserPolicy;
  if (!policy) return t("privacy.hardeningStatusUnknown");
  if (!policy.supported) return policy.unsupportedReason || t("privacy.browserPolicyUnsupported");
  if (policy.installed) {
    if (policy.detectedBrowsers === 0) {
      return t("privacy.browserPolicyConfiguredNoDetectedSummary");
    }
    return t("privacy.browserPolicyInstalledSummary", {
      configured: policy.configuredBrowsers,
      detected: policy.detectedBrowsers,
    });
  }
  if (policy.configured) {
    return t("privacy.browserPolicyPartialSummary", {
      configured: policy.configuredBrowsers,
      detected: policy.detectedBrowsers,
    });
  }
  return t("privacy.browserPolicyMissingSummary");
});

const browserPolicyHint = computed(() => {
  const policy = hardening.value?.browserPolicy;
  if (!policy) return "";
  if (!policy.supported) return policy.unsupportedReason || t("privacy.browserPolicyUnsupported");
  if (!policy.canInstall) return policy.installUnavailable || t("privacy.browserPolicyInstallUnavailable");
  if (policy.restartRequired) return t("privacy.browserPolicyRestartRequired");
  return "";
});

const controlledBrowserSummary = computed(() => {
  const controlled = hardening.value?.controlledBrowser;
  if (!controlled) return t("privacy.hardeningStatusUnknown");
  if (controlled.available) {
    return t("privacy.controlledBrowserAvailableSummary", {
      output: controlled.outputDir,
    });
  }
  return controlled.unsupportedReason || t("privacy.controlledBrowserUnavailableSummary");
});

const controlledBrowserHint = computed(() => {
  const controlled = hardening.value?.controlledBrowser;
  if (!controlled || controlled.available) return "";
  return controlled.unsupportedReason || t("privacy.controlledBrowserUnavailableSummary");
});

const dailyBrowserFingerprintSummary = computed(() => {
  const hardeningInfo = hardening.value?.dailyBrowserFingerprint;
  if (!hardeningInfo) return t("privacy.hardeningStatusUnknown");
  return hardeningInfo.reason || t("privacy.dailyBrowserFingerprintBoundary");
});

const dnsResult = computed<PrivacyDnsResult>(() => ({
  leakRisk: runtimeDnsRisk.value.leakRisk,
  expectedRemoteDns: context.value?.tunSettings?.remoteDns || context.value?.tunStatus?.remoteDns || [],
  tunRunning: runtimeDnsRisk.value.tunRunning,
  routeMode: context.value?.tunSettings?.routeMode || "",
  hasRemoteDnsRoute: runtimeDnsRisk.value.hasRemoteDnsRoute,
  hasDirectDnsRoute: runtimeDnsRisk.value.hasDirectDnsRoute,
  hasRemoteResolvers: runtimeDnsRisk.value.hasRemoteResolvers,
  notes: dnsNotes.value,
}));

const duplicateNodeIds = computed(() => {
  const seen = new Set<string>();
  const duplicates = new Set<string>();
  for (const node of allNodes.value) {
    if (seen.has(node.id)) {
      duplicates.add(node.id);
    }
    seen.add(node.id);
  }
  return Array.from(duplicates);
});

const dedupeRisk = computed<PrivacyRiskLevel>(() => (duplicateNodeIds.value.length > 0 ? "high" : "low"));
const cleanlinessRisk = computed<PrivacyRiskLevel>(() => {
  if (!activeNodes.value.length) return "unknown";
  if (intelligenceSummary.value.suspiciousCount > 0 || intelligenceSummary.value.datacenterCount > 0) {
    return "warning";
  }
  if (intelligenceSummary.value.trustedCount > 0 || intelligenceSummary.value.residentialCount > 0) {
    return "low";
  }
  return "unknown";
});

const ipExposureTitle = computed(() => {
  if (browserPublicIpError.value) return t("privacy.ipUnknownTitle");
  if (ipExposure.value.browserMatchesDirect) return t("privacy.ipHighRiskTitle");
  if (ipExposure.value.browserMatchesProxy) return t("privacy.ipLowRiskTitle");
  if (ipExposure.value.leakRisk === "warning") return t("privacy.ipWarningTitle");
  return t("privacy.ipUnknownTitle");
});

const ipExposureSummary = computed(() => {
  if (browserPublicIpError.value) return browserPublicIpError.value;
  if (!ipExposure.value.browserIp) return t("privacy.ipSummaryIdle");
  if (ipExposure.value.browserMatchesDirect) return t("privacy.ipDirectExposed");
  if (ipExposure.value.browserMatchesProxy) return t("privacy.ipProxyMatched");
  return t("privacy.ipMismatch", { ip: ipExposure.value.browserIp });
});

const dnsNotes = computed(() => {
  const notes = [];
  notes.push(runtimeDnsRisk.value.tunRunning ? t("privacy.dnsTunRunning") : t("privacy.dnsTunStopped"));
  if (runtimeDnsRisk.value.hasRemoteResolvers) {
    notes.push(t("privacy.dnsExpectedResolvers", { resolvers: dnsResolversDisplay.value }));
  } else {
    notes.push(t("privacy.dnsNoRemoteResolvers"));
  }
  notes.push(
    runtimeDnsRisk.value.hasRemoteDnsRoute
      ? t("privacy.dnsRemoteRoutePresent")
      : t("privacy.dnsRemoteRouteMissing"),
  );
  if (runtimeDnsRisk.value.hasDirectDnsRoute) {
    notes.push(t("privacy.dnsUnexpectedDirectRoute"));
  }
  return notes;
});

const dnsResolversDisplay = computed(() => {
  return (context.value?.tunSettings?.remoteDns || context.value?.tunStatus?.remoteDns || []).join(", ") || "-";
});

const dnsAlertTitle = computed(() =>
  dnsResult.value.leakRisk === "low"
    ? t("privacy.dnsLowRiskTitle")
    : dnsResult.value.leakRisk === "warning"
      ? t("privacy.dnsWarningTitle")
      : t("privacy.dnsUnknownTitle"),
);

const dnsSummary = computed(() => {
  if (!dnsResult.value.notes.length) return t("privacy.dnsSummaryIdle");
  return dnsResult.value.notes.join(" · ");
});

const webrtcAlertTitle = computed(() =>
  webrtcResult.value.leakRisk === "high"
    ? t("privacy.webrtcHighRiskTitle")
    : webrtcResult.value.leakRisk === "warning"
      ? t("privacy.webrtcWarningTitle")
      : webrtcResult.value.leakRisk === "low"
        ? t("privacy.webrtcLowRiskTitle")
        : t("privacy.webrtcUnknownTitle"),
);

const webrtcSummary = computed(() => {
  if (webrtcResult.value.error) return webrtcResult.value.error;
  if (!webrtcResult.value.gathered) return t("privacy.webrtcSummaryIdle");
  const notes = [];
  notes.push(t("privacy.webrtcCandidateCount", { count: webrtcResult.value.candidates.length }));
  if (webrtcResult.value.exposedPrivateAddress) notes.push(t("privacy.webrtcPrivateExposed"));
  if (webrtcResult.value.exposedPublicAddress) notes.push(t("privacy.webrtcPublicExposed"));
  if (!webrtcResult.value.exposedPrivateAddress && !webrtcResult.value.exposedPublicAddress) {
    notes.push(t("privacy.webrtcNoExposure"));
  }
  return notes.join(" · ");
});

const fingerprintAlertTitle = computed(() =>
  fingerprintRisk.value.leakRisk === "high"
    ? t("privacy.fingerprintHighRiskTitle")
    : fingerprintRisk.value.leakRisk === "warning"
      ? t("privacy.fingerprintWarningTitle")
      : fingerprintRisk.value.leakRisk === "low"
        ? t("privacy.fingerprintLowRiskTitle")
        : t("privacy.fingerprintUnknownTitle"),
);

const fingerprintSummary = computed(() => {
  if (!fingerprintSnapshot.value) return t("privacy.fingerprintSummaryIdle");
  const summary = t("privacy.fingerprintSummary", {
    count: fingerprintRisk.value.highEntropySurfaceCount,
  });
  if (fingerprintRisk.value.leakRisk === "high" || fingerprintRisk.value.leakRisk === "warning") {
    return `${summary} ${t("privacy.fingerprintUseControlledBrowser")}`;
  }
  return summary;
});

const cleanlinessAlertTitle = computed(() =>
  cleanlinessRisk.value === "warning"
    ? t("privacy.cleanlinessWarningTitle")
    : cleanlinessRisk.value === "low"
      ? t("privacy.cleanlinessLowRiskTitle")
      : t("privacy.cleanlinessUnknownTitle"),
);

const cleanlinessSummaryText = computed(() => {
  if (!activeNodes.value.length) return t("privacy.cleanlinessNoActiveNodes");
  return t("privacy.cleanlinessSummary", {
    active: activeNodes.value.length,
    checked: activeIntelligenceCheckedCount.value,
    trusted: intelligenceSummary.value.trustedCount,
    suspicious: intelligenceSummary.value.suspiciousCount,
    unknownClean: intelligenceSummary.value.unknownCleanCount,
    residential: intelligenceSummary.value.residentialCount,
    isp: intelligenceSummary.value.ispLikeCount,
    datacenter: intelligenceSummary.value.datacenterCount,
    unknownNetwork: intelligenceSummary.value.unknownNetworkCount,
  });
});

const dedupeAlertTitle = computed(() =>
  duplicateNodeIds.value.length ? t("privacy.dedupeWarningTitle") : t("privacy.dedupeLowRiskTitle"),
);

const dedupeSummary = computed(() => {
  if (!allNodes.value.length) return t("privacy.dedupeNoNodes");
  if (duplicateNodeIds.value.length) {
    return t("privacy.dedupeDuplicateIds", { ids: duplicateNodeIds.value.join(", ") });
  }
  return t("privacy.dedupeNoDuplicates", { count: allNodes.value.length });
});

function riskTagType(risk: PrivacyRiskLevel) {
  if (risk === "high") return "error";
  if (risk === "warning") return "warning";
  if (risk === "low") return "success";
  return "default";
}

function riskAlertType(risk: PrivacyRiskLevel) {
  if (risk === "high") return "error";
  if (risk === "warning") return "warning";
  if (risk === "low") return "success";
  return "info";
}

function riskLabel(risk: PrivacyRiskLevel) {
  return t(`privacy.risk.${risk}`);
}

function fingerprintRiskLabel(risk: PrivacyRiskLevel) {
  return t(`privacy.fingerprintRisk.${risk}`);
}

function routeModeLabel(value: string) {
  if (value === "strict_proxy") return t("privacy.routeModeStrictProxy");
  if (value === "auto_tested") return t("privacy.routeModeAutoTested");
  return value ? t("privacy.routeModeOther") : "-";
}

function dnsPathLabel(value: string) {
  if (value === "dns-remote") return t("privacy.dnsPathRemote");
  if (value === "dns-cn") return t("privacy.dnsPathChina");
  if (value === "dns-direct-local") return t("privacy.dnsPathLocal");
  return t("privacy.dnsPathOther");
}

function routeLabel(value: string) {
  const normalized = value.toLowerCase();
  if (normalized.includes("proxy")) return t("privacy.routeProxy");
  if (normalized === "direct") return t("privacy.routeDirect");
  return t("privacy.routeOther");
}

function routingDiagnosticLabel(value: string) {
  if (value === "default_proxy_domains") return t("privacy.routingDefaultProxy");
  if (value === "cn_direct_domains") return t("privacy.routingChinaDirect");
  if (value === "protected_direct_domains") return t("privacy.routingProtectedDirect");
  return t("privacy.routingOther");
}

function candidateTypeLabel(value: string) {
  if (value === "host") return t("privacy.webrtcCandidateHost");
  if (value === "srflx") return t("privacy.webrtcCandidateServerReflexive");
  if (value === "relay") return t("privacy.webrtcCandidateRelay");
  if (value === "prflx") return t("privacy.webrtcCandidatePeerReflexive");
  return t("privacy.webrtcCandidateUnknown");
}

function protocolLabel(value: string) {
  if (value === "udp") return t("privacy.protocolUdp");
  if (value === "tcp") return t("privacy.protocolTcp");
  return t("privacy.protocolUnknown");
}

onMounted(async () => {
  await loadAllContext();
  await runFingerprintCheck();
});
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

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
</style>
