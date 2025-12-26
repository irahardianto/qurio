import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface Stats {
  sources: number
  documents: number
  failed_jobs: number
}

export const useStatsStore = defineStore('stats', () => {
  const stats = ref<Stats>({ sources: 0, documents: 0, failed_jobs: 0 })
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchStats() {
    isLoading.value = true
    error.value = null
    try {
      const res = await fetch('/api/stats')
      if (!res.ok) throw new Error(`Failed to fetch stats: ${res.statusText}`)
      const json = await res.json()
      stats.value = json
    } catch (e: any) {
      error.value = e.message || 'Unknown error'
    } finally {
      isLoading.value = false
    }
  }

  return { stats, isLoading, error, fetchStats }
})
