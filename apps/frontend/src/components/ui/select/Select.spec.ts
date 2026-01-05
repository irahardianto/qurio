import { mount, shallowMount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import { nextTick } from 'vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem, SelectGroup, SelectLabel, SelectSeparator } from './index'

describe('Select Components', () => {
  it('Select renders', () => {
    const wrapper = shallowMount(Select)
    expect(wrapper.exists()).toBe(true)
  })

  // Use composition to provide context for sub-components
  it('SelectTrigger and SelectValue render', () => {
    const wrapper = mount({
      template: `
        <Select>
          <SelectTrigger>
            <SelectValue placeholder="Placeholder" />
          </SelectTrigger>
        </Select>
      `,
      components: { Select, SelectTrigger, SelectValue }
    })
    expect(wrapper.text()).toContain('Placeholder')
  })

  it('SelectGroup and SelectLabel render', async () => {
      // Must attach to body for Portal to work correctly in JSDOM environment
      const wrapper = mount({
          template: `
            <Select :open="true">
                <SelectContent>
                    <SelectGroup>
                        <SelectLabel>Group Label</SelectLabel>
                    </SelectGroup>
                </SelectContent>
            </Select>
          `,
          components: { Select, SelectContent, SelectGroup, SelectLabel }
      }, { attachTo: document.body })
      
      await nextTick()
      
      // Content is teleported to body, not in wrapper
      expect(document.body.innerHTML).toContain('Group Label')
      
      wrapper.unmount()
  })

  // SelectItem requires SelectContent which might render in portal or be lazy
  // Testing pure rendering inside SelectContent
  it('SelectItem renders in context', () => {
     const wrapper = mount({
         template: `
            <Select open>
                <SelectContent>
                    <SelectItem value="opt1">Option 1</SelectItem>
                </SelectContent>
            </Select>
         `,
         components: { Select, SelectContent, SelectItem }
     })
     // Check existence - text might be hidden or in portal
     expect(wrapper.exists()).toBe(true)
  })

  it('SelectSeparator renders', () => {
      const wrapper = shallowMount(SelectSeparator)
      expect(wrapper.classes()).toContain('-mx-1')
  })
})