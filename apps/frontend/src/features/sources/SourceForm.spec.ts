import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import { createTestingPinia } from '@pinia/testing'
import SourceForm from './SourceForm.vue'
import { useSourceStore } from './source.store'

// Global Stubs
const globalStubs = {
  Button: { template: '<button><slot /></button>' },
  // Use real input to ensure v-model works correctly in tests or stick to simple stub if v-model binding is compatible
  // Vitest/Vue Test Utils handle v-model on simple elements well.
  // But if Input is a component wrapping input, we need to be careful.
  // The code imports Input from '@/components/ui/input'. This is a component.
  // If we stub it as '<input />', v-model on the component needs to bind to 'modelValue' prop and emit 'update:modelValue'.
  // Simple '<input />' stub might not forward v-model correctly if the test sets value on the stub root.
  // Better to use a functional stub that emits input events.
  Input: {
    template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" :type="type" :placeholder="placeholder" :disabled="disabled" :min="min" :max="max" />',
    props: ['modelValue', 'type', 'placeholder', 'disabled', 'min', 'max']
  },
  Textarea: {
    template: '<textarea :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ['modelValue', 'placeholder']
  },
  Globe: { template: '<svg></svg>' },
  FileUp: { template: '<svg></svg>' },
  Loader2: { template: '<svg></svg>' },
  Plus: { template: '<svg></svg>' },
  Settings2: { template: '<svg></svg>' },
  ChevronDown: { template: '<svg></svg>' },
  ChevronUp: { template: '<svg></svg>' },
  UploadCloud: { template: '<svg></svg>' },
}

describe('SourceForm', () => {
  it('calls addSource on submit with advanced config', async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs
      },
    })
    const store = useSourceStore()
    
    const input = wrapper.find('input[type="text"]')
    await input.setValue('https://example.com')

    // Toggle Advanced
    const toggle = wrapper.find('button[type="button"]') // First button is toggle in this context if we are careful, but tabs are buttons too.
    // The tabs are buttons. The advanced toggle is a button.
    // Tabs are first.
    const buttons = wrapper.findAll('button')
    const advancedToggle = buttons.find(b => b.text().includes('Configuration'))
    await advancedToggle?.trigger('click')
    
    const depthInput = wrapper.find('input[type="number"]')
    await depthInput.setValue(2)

    const textarea = wrapper.find('textarea')
    await textarea.setValue('/login\n/admin')

    await wrapper.find('form').trigger('submit')
    
    expect(store.addSource).toHaveBeenCalledWith({ 
      name: 'https://example.com', 
      url: 'https://example.com',
      max_depth: 2,
      exclusions: ['/login', '/admin']
    })
  })

  it('validates URL format', async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs
      },
    })
    const store = useSourceStore()
    const alertMock = vi.spyOn(window, 'alert').mockImplementation(() => {})

    const input = wrapper.find('input[type="text"]')
    await input.setValue('invalid-url')

    await wrapper.find('form').trigger('submit')

    expect(alertMock).toHaveBeenCalled()
    expect(store.addSource).not.toHaveBeenCalled()
  })

  it('handles file upload', async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs
      },
    })
    const store = useSourceStore()

    // Switch to File Tab
    const buttons = wrapper.findAll('button')
    const fileTab = buttons.find(b => b.text().includes('File Upload'))
    await fileTab?.trigger('click')

    // Trigger file change
    const fileInput = wrapper.find('input[type="file"]')
    const file = new File(['content'], 'test.pdf', { type: 'application/pdf' })
    
    // Simulate file selection
    Object.defineProperty(fileInput.element, 'files', { value: [file] })
    await fileInput.trigger('change')

    await wrapper.find('form').trigger('submit')

    expect(store.uploadSource).toHaveBeenCalled()
  })

  it('shows error message from store', async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn, initialState: { sources: { error: 'Something went wrong' } } })],
        stubs: globalStubs
      },
    })

    expect(wrapper.text()).toContain('Something went wrong')
  })
})
