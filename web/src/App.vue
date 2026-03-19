<template>
  <n-config-provider :theme="theme" :locale="naiveLocale" :date-locale="naiveDateLocale">
    <n-loading-bar-provider>
      <n-message-provider>
        <n-dialog-provider>
          <router-view />
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
