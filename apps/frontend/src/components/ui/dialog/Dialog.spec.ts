import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { Dialog, DialogContent, DialogTrigger } from './index'

describe('Dialog', () => {
  it('renders content when open', async () => {
    // We need to register the components locally or globaly. 
    // Since index export them, we can use them in the template.
    // However, reka-ui components often need a provider or specific structure.
    // Let's try mounting a simple usage.
    
    const wrapper = mount({
      components: { Dialog, DialogContent, DialogTrigger },
      template: `
        <Dialog>
          <DialogTrigger>Open</DialogTrigger>
          <DialogContent>Test Content</DialogContent>
        </Dialog>
      `
    }, {
        attachTo: document.body
    })
    
    await wrapper.find('button').trigger('click')
    
    // Wait for animation/tick
    await new Promise(resolve => setTimeout(resolve, 100))
    
    expect(document.body.innerHTML).toContain('Test Content')
    wrapper.unmount()
  })
})
