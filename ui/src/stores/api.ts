import axios from 'axios'

const managementApi = axios.create({
  baseURL: '/management',
  headers: {
    'Content-Type': 'application/json',
  },
})

const rootApi = axios.create({
  baseURL: '/',
  headers: {
    'Content-Type': 'application/json',
  },
})

const storedManagementKey = () => localStorage.getItem('management_api_key') || localStorage.getItem('api_key')

const managementAuthHeaders = () => {
  const apiKey = storedManagementKey()
  return apiKey ? { Authorization: `Bearer ${apiKey}` } : {}
}

managementApi.interceptors.request.use((config) => {
  const apiKey = storedManagementKey()
  if (apiKey) {
    config.headers.Authorization = `Bearer ${apiKey}`
  }
  return config
})

export const slugAPI = {
  list: (params: { include_archived?: boolean } = {}) => managementApi.get('/slugs', { params }),
  get: (name: string) => managementApi.get(`/slugs/${encodeURIComponent(name)}`),
  create: (data: {
    name: string
    max_activations: number
    expiration_type: string
    expiration_days?: number
    fixed_expires_at?: string
    offline_enabled?: boolean
    offline_token_lifetime_hours?: number
    attributes?: Record<string, any>
  }) => managementApi.post('/slugs', data),
  update: (
    name: string,
    data: Partial<{
      name: string
      max_activations: number
      expiration_type: string
      expiration_days: number
      fixed_expires_at: string
      offline_enabled: boolean
      offline_token_lifetime_hours: number
      attributes: Record<string, any>
    }>,
  ) => managementApi.patch(`/slugs/${encodeURIComponent(name)}`, data),
  delete: (name: string) => managementApi.delete(`/slugs/${encodeURIComponent(name)}`),
}

export const apiKeysAPI = {
  list: () => managementApi.get('/api-keys'),
  create: (name: string) => managementApi.post('/api-keys', { name }),
  revoke: (id: number) => managementApi.post(`/api-keys/${id}/revoke`),
}

export const licensesAPI = {
  list: (params: { page?: number; page_size?: number; q?: string; status?: string } = {}) => managementApi.get('/licenses', { params }),
  // Admin routes take the license id, never the key, so keys stay out of URLs and access logs.
  get: (id: string) => managementApi.get(`/licenses/${encodeURIComponent(id)}`),
  update: (id: string, data: Partial<{ max_activations: number; expires_at: string | null; revoked: boolean }>) =>
    managementApi.patch(`/licenses/${encodeURIComponent(id)}`, data),
  removeActivation: (id: string, activationId: number) =>
    managementApi.delete(`/licenses/${encodeURIComponent(id)}/activations/${activationId}`),
  generate: (data: { slug: string; metadata?: Record<string, any> }) => rootApi.post('/api/v1/generate', data, { headers: managementAuthHeaders() }),
}

export const webhooksAPI = {
  list: () => managementApi.get('/webhooks'),
  deliveries: (limit = 25) => managementApi.get('/webhooks/deliveries', { params: { limit } }),
  create: (data: { name: string; url: string; events: string[]; enabled: boolean }) => managementApi.post('/webhooks', data),
  update: (
    id: number,
    data: Partial<{ name: string; url: string; events: string[]; enabled: boolean }>,
  ) => managementApi.patch(`/webhooks/${id}`, data),
  delete: (id: number) => managementApi.delete(`/webhooks/${id}`),
  rotateSecret: (id: number) => managementApi.post(`/webhooks/${id}/rotate-secret`),
}

export const offlineAPI = {
  signingKeys: () => managementApi.get('/offline/signing-keys'),
  publicKeys: () => managementApi.get('/offline/public-keys'),
  createSigningKey: (name: string) => managementApi.post('/offline/signing-keys', { name }),
  activateSigningKey: (id: number) => managementApi.post(`/offline/signing-keys/${id}/activate`),
  retireSigningKey: (id: number) => managementApi.post(`/offline/signing-keys/${id}/retire`),
}

// Public runtime endpoints. They take no credentials; the license key is the secret.
// The admin UI uses them to bind a device by hand, e.g. to hand out an offline JWT for an on-prem install.
export const runtimeAPI = {
  activate: (data: { license_key: string; fingerprint: string; metadata?: Record<string, any> }) => rootApi.post('/api/v1/activate', data),
  validate: (data: { license_key: string; fingerprint: string }) => rootApi.post('/api/v1/validate', data),
}
