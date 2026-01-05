import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import Textarea from './Textarea.vue'

describe('Textarea', () => {
  it('renders correctly', () => {
    const wrapper = mount(Textarea, {
      props: {
        placeholder: 'Enter text'
      }
    })
    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('textarea').attributes('placeholder')).toBe('Enter text')
  })

  it('updates v-model', async () => {
    const wrapper = mount(Textarea, {
      props: {
        modelValue: 'initial'
      }
    })
    const textarea = wrapper.find('textarea')
    expect(textarea.element.value).toBe('initial')

    await textarea.setValue('updated')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['updated'])
  })

  it('handles disabled state', () => {
    const wrapper = mount(Textarea, {
      props: {
        disabled: true
      }
    })
    expect(wrapper.find('textarea').element.disabled).toBe(true)
  })
})
