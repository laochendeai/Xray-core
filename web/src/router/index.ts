import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/components/layout/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/Dashboard.vue') },
      { path: 'inbounds', name: 'Inbounds', component: () => import('@/views/Inbounds.vue') },
      { path: 'outbounds', name: 'Outbounds', component: () => import('@/views/Outbounds.vue') },
      { path: 'users', name: 'Users', component: () => import('@/views/Users.vue') },
      { path: 'routing', name: 'Routing', component: () => import('@/views/Routing.vue') },
      { path: 'dns', name: 'DNS', component: () => import('@/views/DNS.vue') },
      { path: 'monitor', name: 'Monitor', component: () => import('@/views/Monitor.vue') },
      { path: 'settings', name: 'Settings', component: () => import('@/views/Settings.vue') },
      { path: 'config', name: 'Config', component: () => import('@/views/Config.vue') },
      { path: 'subscriptions', name: 'Subscriptions', component: () => import('@/views/Subscriptions.vue') },
      { path: 'node-pool', name: 'NodePool', component: () => import('@/views/NodePool.vue') }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  if (to.meta.requiresAuth !== false && !authStore.isAuthenticated) {
    next('/login')
  } else {
    next()
  }
})

export default router
