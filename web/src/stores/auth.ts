import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authAPI } from '@/api/client'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref(localStorage.getItem('username') || '')

  const isAuthenticated = computed(() => !!token.value)

  async function login(user: string, pass: string) {
    const resp = await authAPI.login(user, pass)
    token.value = resp.token
    username.value = user
    localStorage.setItem('token', resp.token)
    localStorage.setItem('username', user)
    router.push('/dashboard')
  }

  function logout() {
    token.value = ''
    username.value = ''
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    router.push('/login')
  }

  return { token, username, isAuthenticated, login, logout }
})
