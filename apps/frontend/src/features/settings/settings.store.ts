import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useSettingsStore = defineStore('settings', () => {
  const rerankProvider = ref('none')
  const rerankApiKey = ref('')
  const geminiApiKey = ref('')
  const searchAlpha = ref(0.5)
  const searchTopK = ref(20)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const successMessage = ref<string | null>(null)

  async function fetchSettings() {
    isLoading.value = true
    error.value = null
    try {
      const res = await fetch('/api/settings')
      if (!res.ok) throw new Error('Failed to fetch settings')
      const json = await res.json()
      const data = json.data || {}
      rerankProvider.value = data.rerank_provider || 'none'
      rerankApiKey.value = data.rerank_api_key || ''
      geminiApiKey.value = data.gemini_api_key || ''
      searchAlpha.value = data.search_alpha ?? 0.5
      searchTopK.value = data.search_top_k ?? 20
    } catch (e: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
      error.value = e.message
    } finally {
      isLoading.value = false
    }
  }

  async function updateSettings() {
    isLoading.value = true
    error.value = null
    successMessage.value = null
    try {
      const res = await fetch('/api/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          rerank_provider: rerankProvider.value,
          rerank_api_key: rerankApiKey.value,
          gemini_api_key: geminiApiKey.value,
          search_alpha: searchAlpha.value,
          search_top_k: searchTopK.value,
        }),
      })
      if (!res.ok) throw new Error('Failed to update settings')
      successMessage.value = 'Settings saved successfully'
      setTimeout(() => successMessage.value = null, 3000)
    } catch (e: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
      error.value = e.message
    } finally {
      isLoading.value = false
    }
  }

  return {
    rerankProvider,
    rerankApiKey,
    geminiApiKey,
    searchAlpha,
    searchTopK,
    isLoading,
    error,
    successMessage,
    fetchSettings,
    updateSettings,
  }
})
