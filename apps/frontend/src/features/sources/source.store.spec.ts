import { setActivePinia, createPinia } from 'pinia'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useSourceStore } from './source.store'

describe('Source Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    global.fetch = vi.fn()
  })

  it('initializes with correct default state', () => {
    const store = useSourceStore()
    expect(store.sources).toEqual([])
    expect(store.isLoading).toBe(false)
    expect(store.error).toBe(null)
  })

  it('fetchSources populates state on success', async () => {
    const store = useSourceStore()
    const mockSources = [{ id: '1', name: 'Test' }]
    
    // Mock successful response
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: mockSources })
    })

    const promise = store.fetchSources()
    expect(store.isLoading).toBe(true)
    await promise
    
    expect(store.sources).toEqual(mockSources)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBe(null)
  })

  it('fetchSources handles error', async () => {
    const store = useSourceStore()
    
    // Mock error response
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: 'Internal Server Error'
    })

    await store.fetchSources()
    
    expect(store.sources).toEqual([])
    expect(store.isLoading).toBe(false)
    expect(store.error).toContain('Failed to fetch sources')
  })

  it('addSource posts to API and updates state on success', async () => {
    const store = useSourceStore()
    const newSourceInput = { 
      name: 'New Source', 
      url: 'http://example.com',
      max_depth: 2,
      exclusions: ['/admin']
    }
    const createdSource = { id: '2', ...newSourceInput }
    
    global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ data: createdSource })
    })

    const promise = store.addSource(newSourceInput)
    expect(store.isLoading).toBe(true)
    await promise

    expect(global.fetch).toHaveBeenCalledWith('/api/sources', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(newSourceInput)
    }))
    expect(store.sources).toContainEqual(createdSource)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBe(null)
  })

  it('addSource handles error', async () => {
    const store = useSourceStore()
    const newSourceInput = { name: 'New Source', url: 'http://example.com' }
    
    global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        statusText: 'Bad Request'
    })

    await store.addSource(newSourceInput)

    expect(store.sources).toHaveLength(0)
    expect(store.isLoading).toBe(false)
    expect(store.error).toContain('Failed to add source')
  })
})