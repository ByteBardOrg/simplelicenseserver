import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const apiKey = ref<string | null>(null)

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
    setApiKey,
    clearApiKey,
  }
})
