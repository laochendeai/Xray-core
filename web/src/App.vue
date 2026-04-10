<template>
  <n-config-provider :theme="theme" :locale="naiveLocale" :date-locale="naiveDateLocale">
    <n-loading-bar-provider>
      <n-message-provider>
        <n-dialog-provider>
          <div class="app-frame" :class="{ 'app-frame-dark': appStore.isDark }">
            <router-view />
          </div>
        </n-dialog-provider>
      </n-message-provider>
    </n-loading-bar-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NConfigProvider, NLoadingBarProvider, NMessageProvider, NDialogProvider, darkTheme, zhCN, dateZhCN, enUS, dateEnUS } from 'naive-ui'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const theme = computed(() => appStore.isDark ? darkTheme : null)

const naiveLocale = computed(() => appStore.locale === 'zh-CN' ? zhCN : enUS)
const naiveDateLocale = computed(() => appStore.locale === 'zh-CN' ? dateZhCN : dateEnUS)
</script>

<style>
:root {
  --panel-max-width: 1500px;
  --panel-edge-gap: clamp(16px, 2vw, 28px);
  --panel-radius-xl: 30px;
  --panel-radius-lg: 24px;
  --panel-radius-md: 18px;
  --panel-shadow: 0 24px 60px rgba(15, 23, 42, 0.08);
}

html,
body,
#app {
  min-height: 100%;
}

body {
  margin: 0;
  font-family: "Avenir Next", "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Noto Sans SC", "Microsoft YaHei", sans-serif;
  background: #fff;
  color: #111827;
}

* {
  box-sizing: border-box;
}

.page-shell {
  width: min(100%, var(--panel-max-width));
  margin: 0 auto;
}

.app-frame {
  min-height: 100vh;
  background: #f5f7fb;
  color: #1f2937;
  --panel-page-bg: #f5f7fb;
  --panel-surface: #ffffff;
  --panel-surface-soft: #f7f9fc;
  --panel-border: rgba(15, 23, 42, 0.08);
  --panel-text: #1f2937;
  --panel-text-2: #4b5563;
  --panel-text-3: #6b7280;
  --panel-primary: #2080f0;
}

.app-frame-dark {
  background: #101014;
  color: #f3f4f6;
  --panel-page-bg: #101014;
  --panel-surface: #1a1d24;
  --panel-surface-soft: #14171d;
  --panel-border: rgba(255, 255, 255, 0.08);
  --panel-text: #f3f4f6;
  --panel-text-2: #d1d5db;
  --panel-text-3: #9ca3af;
  --panel-primary: #63e2b7;
}

.app-frame a {
  color: var(--panel-primary);
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
