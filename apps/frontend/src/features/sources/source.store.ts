import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface Source {
  id: string
  name: string
  url?: string
  status?: string
  lastSyncedAt?: string
  max_depth?: number
  exclusions?: string[]
}

export const useSourceStore = defineStore('sources', () => {
  const sources = ref<Source[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  let pollingInterval: any = null

  async function fetchSources(background = false) {
    if (!background) isLoading.value = true
    error.value = null
    try {
      const res = await fetch('/api/sources')
      if (!res.ok) {
        throw new Error(`Failed to fetch sources: ${res.statusText}`)
      }
      const json = await res.json()
      sources.value = json.data || []
    } catch (e: any) {
      error.value = e.message || 'Unknown error'
      console.error('Failed to fetch sources', e)
    } finally {
      if (!background) isLoading.value = false
    }
  }

  function startPolling() {
    if (pollingInterval) return
    pollingInterval = setInterval(() => {
      const hasActiveSources = sources.value.some(s => 
        s.status === 'processing' || s.status === 'pending'
      )
      if (hasActiveSources) {
        fetchSources(true)
      }
    }, 2000)
  }

  function stopPolling() {
    if (pollingInterval) {
      clearInterval(pollingInterval)
      pollingInterval = null
    }
  }

  async function addSource(source: Omit<Source, 'id'>) {
    isLoading.value = true
    error.value = null
    try {
      const res = await fetch('/api/sources', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(source),
      })
      if (!res.ok) {
        throw new Error(`Failed to add source: ${res.statusText}`)
      }
      const json = await res.json()
      sources.value.push(json.data)
    } catch (e: any) {
      error.value = e.message || 'Unknown error'
      console.error('Failed to add source', e)
    } finally {
      isLoading.value = false
    }
  }

  async function deleteSource(id: string) {
    isLoading.value = true
    error.value = null
    try {
      const res = await fetch(`/api/sources/${id}`, { method: 'DELETE' })
      if (!res.ok) throw new Error(`Failed to delete source: ${res.statusText}`)
      sources.value = sources.value.filter(s => s.id !== id)
    } catch (e: any) {
      error.value = e.message || 'Unknown error'
    } finally {
      isLoading.value = false
    }
  }

  async function resyncSource(id: string) {
    isLoading.value = true
    error.value = null
    try {
      const res = await fetch(`/api/sources/${id}/resync`, { method: 'POST' })
      if (!res.ok) throw new Error(`Failed to resync source: ${res.statusText}`)
    } catch (e: any) {
      error.value = e.message || 'Unknown error'
    } finally {
      isLoading.value = false
    }
  }

  async function getSource(id: string) {
    isLoading.value = true
    error.value = null
    try {
      const res = await fetch(`/api/sources/${id}`)
      if (!res.ok) throw new Error(`Failed to fetch source details: ${res.statusText}`)
      const json = await res.json()
      return json.data
    } catch (e: any) {
      error.value = e.message || 'Unknown error'
      return null
    } finally {
      isLoading.value = false
    }
  }

  return { 
    sources, 
    isLoading, 
    error, 
    fetchSources, 
    addSource,
    deleteSource,
    resyncSource,
    getSource,
    startPolling,
    stopPolling
  }
})