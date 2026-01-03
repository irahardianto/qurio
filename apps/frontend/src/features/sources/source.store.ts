import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface Chunk {
  content: string
  vector?: number[]
  source_url: string
  source_id: string
  source_name?: string
  chunk_index: number
  type: string
  language: string
  title: string
}

export interface Source {
  id: string
  name: string
  type?: string
  url?: string
  status?: string
  lastSyncedAt?: string
  max_depth?: number
  exclusions?: string[]
  chunks?: Chunk[]
  total_chunks?: number
}

export interface SourcePage {
  id: string
  source_id: string
  url: string
  status: string
  depth: number
  error?: string
  created_at: string
  updated_at: string
}

export const useSourceStore = defineStore('sources', () => {
  const sources = ref<Source[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  let pollingInterval: any = null // eslint-disable-line @typescript-eslint/no-explicit-any

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
    } catch (e: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
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
        s.status === 'processing' || s.status === 'pending' || s.status === 'in_progress'
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
    } catch (e: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
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
    } catch (e: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
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
    } catch (e: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
      error.value = e.message || 'Unknown error'
    } finally {
      isLoading.value = false
    }
  }

  async function uploadSource(file: File) {
    isLoading.value = true
    error.value = null
    try {
      const formData = new FormData()
      formData.append('file', file)

      const res = await fetch('/api/sources/upload', {
        method: 'POST',
        body: formData,
      })
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}))
        throw new Error(errorData.error?.message || `Failed to upload source: ${res.statusText}`)
      }
      const json = await res.json()
      sources.value.push(json.data)
      return json.data
    } catch (e: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
      error.value = e.message || 'Unknown error'
      console.error('Failed to upload source', e)
      throw e
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
      return json.data as Source
    } catch (e: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
      error.value = e.message || 'Unknown error'
      return null
    } finally {
      isLoading.value = false
    }
  }

  async function getSourcePages(id: string) {
    try {
      const res = await fetch(`/api/sources/${id}/pages`)
      if (!res.ok) throw new Error(`Failed to fetch source pages: ${res.statusText}`)
      const json = await res.json()
      return json.data as SourcePage[]
    } catch (e: any) { // eslint-disable-line @typescript-eslint/no-explicit-any
      console.error('Failed to fetch source pages', e)
      return []
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
    uploadSource,
    getSource,
    getSourcePages,
    startPolling,
    stopPolling
  }
})
