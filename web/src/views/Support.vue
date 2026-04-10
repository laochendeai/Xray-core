<template>
  <div class="page-shell support-page">
    <section class="support-hero">
      <div class="support-hero-copy">
        <p class="support-eyebrow">{{ t('support.platformName') }}</p>
        <h1 class="support-title">{{ t('nav.support') }}</h1>
        <p class="support-summary">{{ t('support.subtitle') }}</p>
        <n-space class="support-actions" :size="12" wrap>
          <n-button
            type="primary"
            tag="a"
            :href="supportUrl"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ t('support.openPage') }}
          </n-button>
          <n-button secondary @click="handleCopyLink">
            {{ t('support.copyLink') }}
          </n-button>
        </n-space>
      </div>

      <div class="support-metrics">
        <div class="support-metric-card">
          <span class="support-metric-label">{{ t('support.platformLabel') }}</span>
          <strong class="support-metric-value">{{ t('support.platformName') }}</strong>
        </div>
        <div class="support-metric-card">
          <span class="support-metric-label">{{ t('support.routeLabel') }}</span>
          <strong class="support-metric-value">{{ supportRoute }}</strong>
        </div>
        <div class="support-metric-card support-metric-card-wide">
          <span class="support-metric-label">{{ t('support.addressLabel') }}</span>
          <strong class="support-metric-value support-link">{{ supportUrl }}</strong>
        </div>
      </div>
    </section>

    <n-grid :cols="2" :x-gap="18" :y-gap="18" responsive="screen" item-responsive>
      <n-gi span="2 m:1">
        <n-card :title="t('support.channelTitle')" size="small" :bordered="false" class="support-panel">
          <n-space vertical :size="16">
            <n-alert type="info">
              {{ t('support.channelBody') }}
            </n-alert>
            <div class="support-link-row">
              <span class="support-link-label">{{ t('support.addressLabel') }}</span>
              <a :href="supportUrl" target="_blank" rel="noopener noreferrer" class="support-link-anchor">
                {{ supportUrl }}
              </a>
            </div>
            <p class="support-note">{{ t('support.noPressure') }}</p>
          </n-space>
        </n-card>
      </n-gi>

      <n-gi span="2 m:1">
        <n-card :title="t('support.qrcodeTitle')" size="small" :bordered="false" class="support-panel">
          <div class="support-qrcode-block">
            <QrcodeVue :value="supportUrl" :size="184" level="M" />
            <p class="support-qrcode-hint">{{ t('support.qrcodeHint') }}</p>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card :title="t('support.fundsTitle')" size="small" :bordered="false" class="support-panel">
      <ul class="support-checklist">
        <li>{{ t('support.funds.runtime') }}</li>
        <li>{{ t('support.funds.ui') }}</li>
        <li>{{ t('support.funds.docs') }}</li>
      </ul>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NAlert, NButton, NCard, NGi, NGrid, NSpace, useMessage } from 'naive-ui'
import QrcodeVue from 'qrcode.vue'
import { useI18n } from 'vue-i18n'
import { copyToClipboard } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()

const supportUrl = 'https://ifdian.net/a/abc678'
const supportRoute = '/support'

async function handleCopyLink() {
  try {
    await copyToClipboard(supportUrl)
    message.success(t('common.copied'))
  } catch {
    message.error(t('common.error'))
  }
}
</script>

<style scoped>
.support-page {
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.support-hero {
  display: grid;
  gap: 24px;
  padding: clamp(22px, 3vw, 36px);
  border: 1px solid var(--panel-border);
  border-radius: var(--panel-radius-xl);
  background: var(--panel-surface);
  box-shadow: var(--panel-shadow);
  grid-template-columns: minmax(0, 1.15fr) minmax(320px, 0.95fr);
}

.support-hero-copy {
  display: flex;
  flex-direction: column;
}

.support-eyebrow {
  margin: 0 0 12px;
  color: var(--panel-text-3);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.24em;
  text-transform: uppercase;
}

.support-title {
  margin: 0;
  font-size: clamp(2rem, 4vw, 3rem);
  line-height: 1.05;
  letter-spacing: -0.03em;
}

.support-summary {
  max-width: 44rem;
  margin: 14px 0 0;
  color: var(--panel-text-2);
  font-size: 15px;
  line-height: 1.7;
}

.support-actions {
  margin-top: 22px;
}

.support-metrics {
  display: grid;
  align-self: end;
  gap: 14px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.support-metric-card {
  display: flex;
  min-height: 118px;
  flex-direction: column;
  justify-content: space-between;
  padding: 18px 20px;
  border: 1px solid var(--panel-border);
  border-radius: 24px;
  background: var(--panel-surface-soft);
}

.support-metric-card-wide {
  grid-column: 1 / -1;
}

.support-metric-label {
  color: var(--panel-text-3);
  font-size: 12px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.support-metric-value {
  font-size: clamp(1.05rem, 1.8vw, 1.4rem);
  font-weight: 700;
  line-height: 1.3;
}

.support-link {
  word-break: break-all;
}

.support-panel {
  border: 1px solid var(--panel-border);
  border-radius: var(--panel-radius-xl);
  background: var(--panel-surface);
  box-shadow: var(--panel-shadow);
}

.support-link-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.support-link-label {
  color: var(--panel-text-3);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.support-link-anchor {
  word-break: break-all;
}

.support-note {
  margin: 0;
  color: var(--panel-text-2);
  line-height: 1.7;
}

.support-qrcode-block {
  display: flex;
  min-height: 100%;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  text-align: center;
}

.support-qrcode-hint {
  max-width: 22rem;
  margin: 0;
  color: var(--panel-text-2);
  line-height: 1.6;
}

.support-checklist {
  display: grid;
  gap: 12px;
  margin: 0;
  padding-left: 18px;
  color: var(--panel-text-2);
  line-height: 1.7;
}

:deep(.support-panel > .n-card-header) {
  padding: 22px 24px 0;
}

:deep(.support-panel > .n-card__content) {
  padding: 18px 24px 24px;
}

@media (max-width: 1080px) {
  .support-hero {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .support-hero {
    padding: 20px;
    border-radius: 26px;
  }

  .support-metrics {
    grid-template-columns: 1fr;
  }

  :deep(.support-panel > .n-card-header) {
    padding: 18px 18px 0;
  }

  :deep(.support-panel > .n-card__content) {
    padding: 16px 18px 18px;
  }
}
</style>
