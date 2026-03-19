import { ref, onMounted, onUnmounted } from 'vue'

export function useWebSocket(path: string) {
  const data = ref<any>(null)
  const isConnected = ref(false)
  const error = ref<string | null>(null)
  let ws: WebSocket | null = null

  function connect() {
    const token = localStorage.getItem('token') || ''
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${window.location.host}${path}?token=${token}`

    ws = new WebSocket(url)

    ws.onopen = () => {
      isConnected.value = true
      error.value = null
    }

    ws.onmessage = (event) => {
      try {
        data.value = JSON.parse(event.data)
      } catch {
        data.value = event.data
      }
    }

    ws.onerror = () => {
      error.value = 'WebSocket error'
    }

    ws.onclose = () => {
      isConnected.value = false
    }
  }

  function disconnect() {
    if (ws) {
      ws.close()
      ws = null
    }
  }

  onMounted(connect)
  onUnmounted(disconnect)

  return { data, isConnected, error, connect, disconnect }
}
