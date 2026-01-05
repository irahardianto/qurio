import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import { describe, it, expect, vi } from 'vitest'
import Settings from './Settings.vue'
import { useSettingsStore } from './settings.store'

describe('Settings.vue', () => {
  it('fetches settings on mount', () => {
    mount(Settings, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
      },
    })
    const store = useSettingsStore()
    expect(store.fetchSettings).toHaveBeenCalled()
  })

  it('calls updateSettings when save button is clicked', async () => {
    const wrapper = mount(Settings, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
      },
    })
    const store = useSettingsStore()
    
    // Find the save button (last button or by text)
    const buttons = wrapper.findAll('button')
    const saveBtn = buttons.find(b => b.text().includes('Save Configuration'))
    await saveBtn?.trigger('click')
    expect(store.updateSettings).toHaveBeenCalled()
  })

  it('shows api key input only when reranker provider is selected', async () => {
    const wrapper = mount(Settings, {
      global: {
        plugins: [createTestingPinia({ stubActions: false, createSpy: vi.fn })],
      },
    })
    const store = useSettingsStore()
    
    // Initial state: none
    store.rerankProvider = 'none'
    await wrapper.vm.$nextTick()
    expect(wrapper.find('#apiKey').exists()).toBe(false)
    
    // Change to jina
    store.rerankProvider = 'jina'
    await wrapper.vm.$nextTick()
    expect(wrapper.find('#apiKey').exists()).toBe(true)
  })
})
