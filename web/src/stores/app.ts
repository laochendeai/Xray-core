import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAppStore = defineStore('app', () => {
  const isDark = ref(localStorage.getItem('theme') === 'dark')
  const locale = ref(localStorage.getItem('locale') || 'zh-CN')
  const sidebarCollapsed = ref(false)

  function toggleTheme() {
    isDark.value = !isDark.value
    localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
  }

  function setLocale(lang: string) {
    locale.value = lang
    localStorage.setItem('locale', lang)
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  const themeLabel = computed(() => isDark.value ? 'Dark' : 'Light')

  return { isDark, locale, sidebarCollapsed, toggleTheme, setLocale, toggleSidebar, themeLabel }
})
