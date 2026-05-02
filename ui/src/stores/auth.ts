import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const apiKey = ref<string | null>(null)
  const isLoading = ref(false)

  const loadApiKey = () => {
    const stored = localStorage.getItem('management_api_key') || localStorage.getItem('api_key')
    if (stored) {
      apiKey.value = stored
    }
  }

  const setApiKey = (key: string) => {
    apiKey.value = key
    localStorage.setItem('management_api_key', key)
  }

  const clearApiKey = () => {
    apiKey.value = null
    localStorage.removeItem('management_api_key')
    localStorage.removeItem('api_key')
  }

  return {
    apiKey,
    isLoading,
    loadApiKey,
    setApiKey,
    clearApiKey,
  }
})
