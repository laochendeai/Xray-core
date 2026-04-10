<template>
  <n-layout has-sider class="panel-shell">
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
      class="panel-sider"
    >
      <div class="panel-brand">
        <span class="panel-brand-mark">X</span>
        <div v-if="!appStore.sidebarCollapsed" class="panel-brand-copy">
          <span class="panel-brand-title">Xray Panel</span>
          <span class="panel-brand-subtitle">{{ currentRouteLabel }}</span>
        </div>
      </div>
      <n-menu
        :collapsed="appStore.sidebarCollapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
        :value="currentRoute"
        @update:value="handleMenuClick"
        class="panel-menu"
      />
    </n-layout-sider>
    <n-layout class="panel-main">
      <n-layout-header bordered class="panel-topbar">
        <div class="panel-topbar-left">
          <n-button quaternary circle size="small" @click="appStore.toggleSidebar" class="mobile-menu">
            <template #icon><span class="panel-menu-icon">&#9776;</span></template>
          </n-button>
          <div class="panel-route">
            <span class="panel-route-title">{{ currentRouteLabel }}</span>
            <span class="panel-route-path">{{ route.path }}</span>
          </div>
        </div>
        <div class="panel-topbar-actions">
          <n-select
            v-model:value="currentLocale"
            :options="localeOptions"
            size="small"
            class="panel-locale-select"
            @update:value="handleLocaleChange"
          />
          <n-button quaternary circle size="small" @click="appStore.toggleTheme">
            <template #icon><span>{{ appStore.isDark ? '&#9728;' : '&#9790;' }}</span></template>
          </n-button>
          <n-dropdown :options="userOptions" @select="handleUserAction">
            <n-button quaternary size="small" class="panel-user-button">{{ authStore.username }}</n-button>
          </n-dropdown>
        </div>
      </n-layout-header>
      <n-layout-content class="panel-content" :native-scrollbar="false">
        <div class="panel-content-wrap">
          <router-view />
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
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
  { label: t('nav.readiness'), key: 'readiness' },
  { label: t('nav.inbounds'), key: 'inbounds' },
  { label: t('nav.outbounds'), key: 'outbounds' },
  { label: t('nav.users'), key: 'users' },
  { label: t('nav.subscriptions'), key: 'subscriptions' },
  { label: t('nav.nodePool'), key: 'node-pool' },
  { label: t('nav.routing'), key: 'routing' },
  { label: t('nav.dns'), key: 'dns' },
  { label: t('nav.monitor'), key: 'monitor' },
  { label: t('nav.settings'), key: 'settings' },
  { label: t('nav.config'), key: 'config' },
  { label: t('nav.support'), key: 'support' }
])

const localeOptions = [
  { label: '中文', value: 'zh-CN' },
  { label: 'English', value: 'en' }
]

const userOptions = [
  { label: t('auth.logout'), key: 'logout' }
]

const currentRouteLabel = computed(() => {
  const matched = menuOptions.value.find(option => option.key === currentRoute.value)
  return typeof matched?.label === 'string' ? matched.label : 'Xray Panel'
})

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

<style scoped>
.panel-shell {
  height: 100vh;
  background: transparent;
}

.panel-sider {
  height: 100vh;
  background: var(--panel-surface);
  border-right: 1px solid var(--panel-border);
}

.panel-brand {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 20px 18px 16px;
}

.panel-brand-mark {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border-radius: 14px;
  background: var(--panel-primary);
  color: #fff;
  font-size: 18px;
  font-weight: 700;
}

.panel-brand-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.panel-brand-title {
  color: var(--panel-text);
  font-size: 16px;
  font-weight: 700;
}

.panel-brand-subtitle {
  overflow: hidden;
  color: var(--panel-text-3);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.panel-menu {
  padding: 4px 10px 18px;
}

.panel-main {
  background: transparent;
}

.panel-topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  height: 72px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 clamp(18px, 2.4vw, 34px);
  background: var(--panel-surface);
  border-bottom: 1px solid var(--panel-border);
}

.panel-topbar-left,
.panel-topbar-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.panel-route {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.panel-route-title {
  font-size: 17px;
  font-weight: 700;
  letter-spacing: 0.01em;
}

.panel-route-path {
  color: var(--panel-text-3);
  font-size: 12px;
}

.panel-menu-icon {
  font-size: 18px;
}

.panel-locale-select {
  width: 112px;
}

.panel-user-button {
  border-radius: 999px;
}

.panel-content {
  background: transparent;
}

.panel-content-wrap {
  padding: clamp(18px, 2.4vw, 32px);
}

.mobile-menu {
  display: none;
}

:deep(.n-layout-toggle-bar) {
  border-radius: 999px 0 0 999px;
}

:deep(.panel-menu .n-menu-item-content),
:deep(.panel-menu .n-submenu-children .n-menu-item-content) {
  border-radius: 14px;
  transition: background-color 0.2s ease, transform 0.2s ease;
}

:deep(.panel-menu .n-menu-item-content:hover),
:deep(.panel-menu .n-submenu-children .n-menu-item-content:hover) {
  transform: translateX(2px);
}

@media (max-width: 768px) {
  .mobile-menu {
    display: inline-flex;
  }

  .panel-topbar {
    height: 64px;
    padding-inline: 16px;
  }

  .panel-route-path {
    display: none;
  }

  .panel-content-wrap {
    padding: 16px;
  }
}

@media (max-width: 640px) {
  .panel-topbar-actions {
    gap: 8px;
  }

  .panel-locale-select {
    width: 92px;
  }
}
</style>
