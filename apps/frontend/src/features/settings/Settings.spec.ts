import { mount, flushPromises } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import { describe, it, expect, vi } from 'vitest'
import Settings from './Settings.vue'
import { useSettingsStore } from './settings.store'

const globalStubs = {
  Card: { template: '<div><slot /></div>' },
  CardHeader: { template: '<div><slot /></div>' },
  CardTitle: { template: '<div><slot /></div>' },
  CardDescription: { template: '<div><slot /></div>' },
  CardContent: { template: '<div><slot /></div>' },
  CardFooter: { template: '<div><slot /></div>' },
  Button: { template: '<button><slot /></button>' },
  Input: { template: '<input />' },
  Select: { template: '<div><slot /></div>' },
  SelectTrigger: { template: '<div><slot /></div>' },
  SelectValue: { template: '<span></span>' },
  SelectContent: { template: '<div><slot /></div>' },
  SelectItem: { template: '<div><slot /></div>' },
}

describe('Settings.vue', () => {
  it('fetches settings on mount', () => {
    const wrapper = mount(Settings, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs
      }
    })
    const store = useSettingsStore()
    expect(store.fetchSettings).toHaveBeenCalled()
  })

  it('calls updateSettings when save button is clicked', async () => {
    const wrapper = mount(Settings, {
      global: {
        plugins: [createTestingPinia({
            initialState: {
                settings: { 
                    rerank_provider: 'cohere',
                    search_top_k: 5
                } 
            },
            createSpy: vi.fn 
        })],
        stubs: globalStubs
      }
    })
    
    const store = useSettingsStore()
    
    // Find the save button (it's the button in the footer usually)
    const btn = wrapper.findAll('button').find(b => b.text() === 'Save Configuration')
    await btn?.trigger('click')
    
    expect(store.updateSettings).toHaveBeenCalled()
  })

  it('shows success message on save', async () => {
    const wrapper = mount(Settings, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs
      }
    })
    
    const store = useSettingsStore()
    store.updateSettings.mockResolvedValue()
    
    const btn = wrapper.findAll('button').find(b => b.text() === 'Save Configuration')
    await btn?.trigger('click')
    
    expect(store.updateSettings).toHaveBeenCalled()
  })

  it('shows error message on failure', async () => {
    const wrapper = mount(Settings, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs
      }
    })

    const store = useSettingsStore()
    store.updateSettings.mockRejectedValueOnce(new Error('Failed'))
    
    const btn = wrapper.findAll('button').find(b => b.text() === 'Save Configuration')
    await btn?.trigger('click')
    
    await flushPromises()
    
    expect(store.updateSettings).toHaveBeenCalled()
  })

  it('shows loading state', async () => {
    const wrapper = mount(Settings, {
      global: {
        plugins: [createTestingPinia({
            initialState: {
                settings: { isLoading: true }
            },
            createSpy: vi.fn 
        })],
        stubs: globalStubs
      }
    })
    
    const btn = wrapper.findAll('button').find(b => b.text() === 'Saving...')
    expect(btn?.attributes('disabled')).toBeDefined()
  })
})