<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider
      bordered
      :collapsed="appStore.sidebarCollapsed"
      collapse-mode="width"
      :collapsed-width="64"
      :width="220"
      show-trigger
      @collapse="appStore.sidebarCollapsed = true"
      @expand="appStore.sidebarCollapsed = false"
      :native-scrollbar="false"
      style="height: 100vh"
    >
      <div style="padding: 16px; text-align: center; font-weight: bold; font-size: 18px">
        <span v-if="!appStore.sidebarCollapsed">Xray Panel</span>
        <span v-else>X</span>
      </div>
      <n-menu
        :collapsed="appStore.sidebarCollapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
        :value="currentRoute"
        @update:value="handleMenuClick"
      />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered style="height: 56px; padding: 0 24px; display: flex; align-items: center; justify-content: space-between">
        <div style="display: flex; align-items: center; gap: 12px">
          <n-button quaternary circle size="small" @click="appStore.toggleSidebar" class="mobile-menu">
            <template #icon><span style="font-size: 18px">&#9776;</span></template>
          </n-button>
        </div>
        <div style="display: flex; align-items: center; gap: 12px">
          <n-select
            v-model:value="currentLocale"
            :options="localeOptions"
            size="small"
            style="width: 100px"
            @update:value="handleLocaleChange"
          />
          <n-button quaternary circle size="small" @click="appStore.toggleTheme">
            <template #icon><span>{{ appStore.isDark ? '&#9728;' : '&#9790;' }}</span></template>
          </n-button>
          <n-dropdown :options="userOptions" @select="handleUserAction">
            <n-button quaternary size="small">{{ authStore.username }}</n-button>
          </n-dropdown>
        </div>
      </n-layout-header>
      <n-layout-content content-style="padding: 24px" :native-scrollbar="false">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, ref, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NLayout, NLayoutSider, NLayoutHeader, NLayoutContent, NMenu, NButton, NSelect, NDropdown, type MenuOption } from 'naive-ui'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const { t } = useI18n()

const currentLocale = ref(appStore.locale)

const currentRoute = computed(() => {
  const path = route.path.split('/')[1] || 'dashboard'
  return path
})

const menuOptions = computed<MenuOption[]>(() => [
  { label: t('nav.dashboard'), key: 'dashboard' },
  { label: t('nav.inbounds'), key: 'inbounds' },
  { label: t('nav.outbounds'), key: 'outbounds' },
  { label: t('nav.users'), key: 'users' },
  { label: t('nav.subscriptions'), key: 'subscriptions' },
  { label: t('nav.nodePool'), key: 'node-pool' },
  { label: t('nav.routing'), key: 'routing' },
  { label: t('nav.dns'), key: 'dns' },
  { label: t('nav.monitor'), key: 'monitor' },
  { label: t('nav.settings'), key: 'settings' },
  { label: t('nav.config'), key: 'config' }
])

const localeOptions = [
  { label: '中文', value: 'zh-CN' },
  { label: 'English', value: 'en' }
]

const userOptions = [
  { label: t('auth.logout'), key: 'logout' }
]

function handleMenuClick(key: string) {
  router.push('/' + key)
}

function handleLocaleChange(val: string) {
  appStore.setLocale(val)
  window.location.reload()
}

function handleUserAction(key: string) {
  if (key === 'logout') {
    authStore.logout()
  }
}
</script>

<style>
.mobile-menu {
  display: none;
}

@media (max-width: 768px) {
  .mobile-menu {
    display: inline-flex;
  }
}
</style>
