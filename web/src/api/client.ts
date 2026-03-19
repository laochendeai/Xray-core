import axios from 'axios'

const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' }
})

// JWT interceptor
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error.response?.data || error)
  }
)

export const authAPI = {
  login: (username: string, password: string) =>
    apiClient.post('/auth/login', { username, password }) as Promise<{ token: string }>
}

export const statsAPI = {
  getSysStats: () => apiClient.get('/sys/stats') as Promise<any>,
  queryStats: (pattern?: string) =>
    apiClient.get('/stats/query', { params: { pattern } }) as Promise<any>,
  getOnlineUsers: () => apiClient.get('/stats/online-users') as Promise<any>,
  getOnlineIPs: (email: string) =>
    apiClient.get('/stats/online-ips', { params: { email } }) as Promise<any>
}

export const handlerAPI = {
  listInbounds: () => apiClient.get('/inbounds') as Promise<any>,
  addInbound: (inbound: any) => apiClient.post('/inbounds', { inbound }) as Promise<any>,
  removeInbound: (tag: string) => apiClient.delete(`/inbounds/${tag}`) as Promise<any>,
  getInboundUsers: (tag: string) => apiClient.get(`/inbounds/${tag}/users`) as Promise<any>,
  addInboundUser: (tag: string, user: any) =>
    apiClient.post(`/inbounds/${tag}/users`, user) as Promise<any>,
  removeInboundUser: (tag: string, email: string) =>
    apiClient.delete(`/inbounds/${tag}/users/${email}`) as Promise<any>,
  listOutbounds: () => apiClient.get('/outbounds') as Promise<any>,
  addOutbound: (outbound: any) => apiClient.post('/outbounds', { outbound }) as Promise<any>,
  removeOutbound: (tag: string) => apiClient.delete(`/outbounds/${tag}`) as Promise<any>
}

export const usersAPI = {
  listAll: () => apiClient.get('/users/') as Promise<any>,
  deleteUser: (email: string) => apiClient.delete(`/users/${email}`) as Promise<any>
}

export const routingAPI = {
  listRules: () => apiClient.get('/routing/rules') as Promise<any>,
  addRule: (rule: any) => apiClient.post('/routing/rules', { rule }) as Promise<any>,
  removeRule: (tag: string) => apiClient.delete(`/routing/rules/${tag}`) as Promise<any>,
  testRoute: (params: any) => apiClient.post('/routing/test', params) as Promise<any>,
  getBalancer: (tag: string) => apiClient.get(`/routing/balancers/${tag}`) as Promise<any>,
  overrideBalancer: (tag: string, target: string) =>
    apiClient.put(`/routing/balancers/${tag}`, { target }) as Promise<any>
}

export const observatoryAPI = {
  getStatus: () => apiClient.get('/observatory/status') as Promise<any>
}

export const loggerAPI = {
  restart: () => apiClient.post('/logger/restart') as Promise<any>
}

export const configAPI = {
  get: () => apiClient.get('/config') as Promise<any>,
  save: (config: any) => apiClient.put('/config', { config }) as Promise<any>,
  reload: () => apiClient.post('/config/reload') as Promise<any>,
  validate: (config: any) => apiClient.post('/config/validate', { config }) as Promise<any>,
  listBackups: () => apiClient.get('/config/backups') as Promise<any>,
  createBackup: () => apiClient.post('/config/backups', { action: 'create' }) as Promise<any>,
  restoreBackup: (name: string) =>
    apiClient.post('/config/backups', { action: 'restore', name }) as Promise<any>
}

export const shareAPI = {
  generate: (params: any) => apiClient.post('/share/generate', params) as Promise<any>
}

export const subscriptionAPI = {
  list: () => apiClient.get('/subscriptions') as Promise<any>,
  add: (data: { url: string; remark: string; autoRefresh: boolean; refreshIntervalMin: number }) =>
    apiClient.post('/subscriptions', data) as Promise<any>,
  delete: (id: string) => apiClient.delete(`/subscriptions/${id}`) as Promise<any>,
  refresh: (id: string) => apiClient.post(`/subscriptions/${id}/refresh`) as Promise<any>
}

export const nodePoolAPI = {
  list: (status?: string) =>
    apiClient.get('/node-pool', { params: status ? { status } : {} }) as Promise<any>,
  promote: (id: string) => apiClient.post(`/node-pool/${id}/promote`) as Promise<any>,
  demote: (id: string) => apiClient.post(`/node-pool/${id}/demote`) as Promise<any>,
  delete: (id: string) => apiClient.delete(`/node-pool/${id}`) as Promise<any>,
  getConfig: () => apiClient.get('/node-pool/config') as Promise<any>,
  updateConfig: (config: any) => apiClient.put('/node-pool/config', config) as Promise<any>
}

export default apiClient
