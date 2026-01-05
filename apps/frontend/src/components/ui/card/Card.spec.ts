import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './index'

describe('Card Components', () => {
  it('Card renders slots and classes', () => {
    const wrapper = mount(Card, {
      slots: { default: 'Card Body' },
      props: { class: 'custom-class' }
    })
    expect(wrapper.text()).toBe('Card Body')
    expect(wrapper.classes()).toContain('custom-class')
    expect(wrapper.classes()).toContain('rounded-xl')
  })

  it('CardHeader renders slots', () => {
    const wrapper = mount(CardHeader, {
      slots: { default: 'Header' }
    })
    expect(wrapper.text()).toBe('Header')
    expect(wrapper.classes()).toContain('flex')
    expect(wrapper.classes()).toContain('flex-col')
  })

  it('CardTitle renders slots', () => {
    const wrapper = mount(CardTitle, {
      slots: { default: 'Title' }
    })
    expect(wrapper.text()).toBe('Title')
    expect(wrapper.classes()).toContain('font-semibold')
  })

  it('CardDescription renders slots', () => {
    const wrapper = mount(CardDescription, {
      slots: { default: 'Desc' }
    })
    expect(wrapper.text()).toBe('Desc')
    expect(wrapper.classes()).toContain('text-muted-foreground')
  })

  it('CardContent renders slots', () => {
    const wrapper = mount(CardContent, {
      slots: { default: 'Content' }
    })
    expect(wrapper.text()).toBe('Content')
    expect(wrapper.classes()).toContain('p-6')
  })

  it('CardFooter renders slots', () => {
    const wrapper = mount(CardFooter, {
      slots: { default: 'Footer' }
    })
    expect(wrapper.text()).toBe('Footer')
    expect(wrapper.classes()).toContain('flex')
    expect(wrapper.classes()).toContain('items-center')
  })
})
