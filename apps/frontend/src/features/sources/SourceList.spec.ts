import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import { createTestingPinia } from '@pinia/testing'
import SourceList from './SourceList.vue'
import { useSourceStore } from './source.store'

describe('SourceList', () => {
  it('displays sources', async () => {
    const wrapper = mount(SourceList, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
      },
    })
    const store = useSourceStore()
    store.sources = [{ id: '1', url: 'https://example.com', status: 'pending' }]
    
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('https://example.com')
  })

  it('calls deleteSource when delete button clicked', async () => {
    window.confirm = vi.fn(() => true)
    
    const wrapper = mount(SourceList, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
      },
    })
    const store = useSourceStore()
    store.sources = [{ id: '1', url: 'https://example.com', name: 'Test', status: 'indexed' }]
    
    await wrapper.vm.$nextTick()
    
    const deleteBtn = wrapper.find('button[title="Delete"]')
    await deleteBtn.trigger('click')
    
    expect(window.confirm).toHaveBeenCalled()
    expect(store.deleteSource).toHaveBeenCalledWith('1')
  })
})
