import { ref, onMounted, onUnmounted } from 'vue'

export function usePolling(fn: () => Promise<void>, interval: number = 5000) {
  const isPolling = ref(false)
  let timer: ReturnType<typeof setInterval> | null = null

  function start() {
    if (isPolling.value) return
    isPolling.value = true
    fn()
    timer = setInterval(fn, interval)
  }

  function stop() {
    isPolling.value = false
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  onMounted(start)
  onUnmounted(stop)

  return { isPolling, start, stop }
}
