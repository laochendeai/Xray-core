import { defineStore } from 'pinia'
import { ref } from 'vue'
import { statsAPI } from '@/api/client'

export interface SysStats {
  numGoroutine: number
  numGC: number
  alloc: number
  totalAlloc: number
  sys: number
  mallocs: number
  frees: number
  liveObjects: number
  pauseTotalNs: number
  uptime: number
}

export interface StatItem {
  name: string
  value: number
}

export const useStatsStore = defineStore('stats', () => {
  const sysStats = ref<SysStats | null>(null)
  const queryStats = ref<StatItem[]>([])
  const onlineUsers = ref<string[]>([])
  const onlineCount = ref(0)

  async function fetchSysStats() {
    const data = await statsAPI.getSysStats()
    sysStats.value = data
  }

  async function fetchQueryStats(pattern?: string) {
    const data = await statsAPI.queryStats(pattern)
    queryStats.value = data.stats
  }

  async function fetchOnlineUsers() {
    const data = await statsAPI.getOnlineUsers()
    onlineUsers.value = data.users || []
    onlineCount.value = data.count || 0
  }

  return { sysStats, queryStats, onlineUsers, onlineCount, fetchSysStats, fetchQueryStats, fetchOnlineUsers }
})
