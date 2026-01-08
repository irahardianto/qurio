### Task 1: Scaffold Dialog Component

**Files:**
- Create: `apps/frontend/src/components/ui/dialog/Dialog.vue`
- Create: `apps/frontend/src/components/ui/dialog/DialogContent.vue`
- Create: `apps/frontend/src/components/ui/dialog/DialogHeader.vue`
- Create: `apps/frontend/src/components/ui/dialog/DialogTitle.vue`
- Create: `apps/frontend/src/components/ui/dialog/DialogDescription.vue`
- Create: `apps/frontend/src/components/ui/dialog/DialogFooter.vue`
- Create: `apps/frontend/src/components/ui/dialog/index.ts`
- Test: `apps/frontend/src/components/ui/dialog/Dialog.spec.ts`

**Requirements:**
- **Acceptance Criteria**
  1. `Dialog` component renders correctly.
  2. `DialogContent` renders with correct styling (modal overlay, centered content).
  3. `Dialog` can be controlled via `v-model:open`.

- **Functional Requirements**
  1. Provide a reusable Modal/Dialog component using `reka-ui` primitives.

- **Non-Functional Requirements**
  1. Match existing Shadcn/UI style (Tailwind classes).

- **Test Coverage**
  - [Unit] `Dialog.spec.ts` - renders content when open.

**Step 1: Write failing test**
```typescript
// apps/frontend/src/components/ui/dialog/Dialog.spec.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { Dialog, DialogContent, DialogTrigger } from './index'

describe('Dialog', () => {
  it('renders content when open', async () => {
    const wrapper = mount({
      components: { Dialog, DialogContent, DialogTrigger },
      template: `
        <Dialog defaultOpen>
          <DialogTrigger>Open</DialogTrigger>
          <DialogContent>Test Content</DialogContent>
        </Dialog>
      `
    })
    expect(document.body.innerHTML).toContain('Test Content')
  })
})
```

**Step 2: Verify test fails**
Run: `npm run test apps/frontend/src/components/ui/dialog/Dialog.spec.ts`
Expected: FAIL (Module not found)

**Step 3: Write minimal implementation**
```typescript
// apps/frontend/src/components/ui/dialog/index.ts
export { default as Dialog } from './Dialog.vue'
export { default as DialogContent } from './DialogContent.vue'
export { default as DialogHeader } from './DialogHeader.vue'
export { default as DialogTitle } from './DialogTitle.vue'
export { default as DialogDescription } from './DialogDescription.vue'
export { default as DialogFooter } from './DialogFooter.vue'
export { default as DialogTrigger } from './DialogTrigger.vue'
```

```vue
// apps/frontend/src/components/ui/dialog/Dialog.vue
<script setup lang="ts">
import { DialogRoot, type DialogRootEmits, type DialogRootProps, useForwardPropsEmits } from 'reka-ui'

const props = defineProps<DialogRootProps>()
const emits = defineEmits<DialogRootEmits>()
const forwarded = useForwardPropsEmits(props, emits)
</script>

<template>
  <DialogRoot v-bind="forwarded">
    <slot />
  </DialogRoot>
</template>
```

```vue
// apps/frontend/src/components/ui/dialog/DialogTrigger.vue
<script setup lang="ts">
import { DialogTrigger, type DialogTriggerProps } from 'reka-ui'

const props = defineProps<DialogTriggerProps>()
</script>

<template>
  <DialogTrigger v-bind="props">
    <slot />
  </DialogTrigger>
</template>
```

```vue
// apps/frontend/src/components/ui/dialog/DialogContent.vue
<script setup lang="ts">
import { computed } from 'vue'
import { DialogContent, type DialogContentEmits, type DialogContentProps, DialogOverlay, DialogPortal, useForwardPropsEmits, DialogClose } from 'reka-ui'
import { X } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = defineProps<DialogContentProps & { class?: string }>()
const emits = defineEmits<DialogContentEmits>()

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props
  return delegated
})

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <DialogPortal>
    <DialogOverlay
      class="fixed inset-0 z-50 bg-black/80  data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
    />
    <DialogContent
      v-bind="forwarded"
      :class="cn(
        'fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%] sm:rounded-lg',
        props.class,
      )"
    >
      <slot />
      <DialogClose
        class="absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none data-[state=open]:bg-accent data-[state=open]:text-muted-foreground"
      >
        <X class="h-4 w-4" />
        <span class="sr-only">Close</span>
      </DialogClose>
    </DialogContent>
  </DialogPortal>
</template>
```

```vue
// apps/frontend/src/components/ui/dialog/DialogHeader.vue
<script setup lang="ts">
import { cn } from '@/lib/utils'
import type { HTMLAttributes } from 'vue'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('flex flex-col space-y-1.5 text-center sm:text-left', props.class)">
    <slot />
  </div>
</template>
```

```vue
// apps/frontend/src/components/ui/dialog/DialogTitle.vue
<script setup lang="ts">
import { DialogTitle, type DialogTitleProps } from 'reka-ui'
import { cn } from '@/lib/utils'
import { computed, type HTMLAttributes } from 'vue'

const props = defineProps<DialogTitleProps & { class?: HTMLAttributes['class'] }>()

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props
  return delegated
})
</script>

<template>
  <DialogTitle
    v-bind="delegatedProps"
    :class="cn('text-lg font-semibold leading-none tracking-tight', props.class)"
  >
    <slot />
  </DialogTitle>
</template>
```

```vue
// apps/frontend/src/components/ui/dialog/DialogDescription.vue
<script setup lang="ts">
import { DialogDescription, type DialogDescriptionProps } from 'reka-ui'
import { cn } from '@/lib/utils'
import { computed, type HTMLAttributes } from 'vue'

const props = defineProps<DialogDescriptionProps & { class?: HTMLAttributes['class'] }>()

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props
  return delegated
})
</script>

<template>
  <DialogDescription
    v-bind="delegatedProps"
    :class="cn('text-sm text-muted-foreground', props.class)"
  >
    <slot />
  </DialogDescription>
</template>
```

```vue
// apps/frontend/src/components/ui/dialog/DialogFooter.vue
<script setup lang="ts">
import { cn } from '@/lib/utils'
import type { HTMLAttributes } from 'vue'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2', props.class)">
    <slot />
  </div>
</template>
```

**Step 4: Verify test passes**
Run: `npm run test apps/frontend/src/components/ui/dialog/Dialog.spec.ts`
Expected: PASS

### Task 2: Implement Gemini API Key Check

**Files:**
- Modify: `apps/frontend/src/features/sources/SourceForm.vue`

**Requirements:**
- **Acceptance Criteria**
  1. Clicking "Ingest" (Web) without Gemini API Key shows Dialog.
  2. Clicking "Upload" (File) without Gemini API Key shows Dialog.
  3. Dialog contains link to Settings.
  4. Ingestion DOES NOT start if Key is missing.
  5. If Key is present, ingestion starts as normal.

- **Functional Requirements**
  1. Check `settingsStore.geminiApiKey` before submission.
  2. Prevent default submission if key is missing.
  3. Show `Dialog`.

- **Non-Functional Requirements**
  1. Use existing `Dialog` component.

- **Test Coverage**
  - [Unit] `SourceForm.spec.ts` - test `submit` with and without key.

**Step 1: Write failing test**
```typescript
// apps/frontend/src/features/sources/SourceForm.spec.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import SourceForm from './SourceForm.vue'
import { useSettingsStore } from '@/features/settings/settings.store'
import { useSourceStore } from './source.store'

describe('SourceForm', () => {
  it('shows alert dialog when gemini api key is missing on web ingest', async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: { Dialog: true, DialogContent: true, DialogTrigger: true }
      }
    })
    
    const settingsStore = useSettingsStore()
    const sourceStore = useSourceStore()
    settingsStore.geminiApiKey = ''
    
    // Set URL
    await wrapper.find('input[type="text"]').setValue('https://example.com')
    
    // Click submit
    await wrapper.find('form').trigger('submit')
    
    // Expect sourceStore.addSource NOT to be called
    expect(sourceStore.addSource).not.toHaveBeenCalled()
    
    // Expect Dialog state to be open (check for text or state)
    // Since we mocked Dialog, we check if the boolean ref changed.
    // We need to access the component instance or check if the Dialog receives the 'open' prop
    // Ideally we inspect the vm
    expect(wrapper.vm.showApiKeyAlert).toBe(true) 
  })
})
```

**Step 2: Verify test fails**
Run: `npm run test apps/frontend/src/features/sources/SourceForm.spec.ts`
Expected: FAIL (showApiKeyAlert undefined or false)

**Step 3: Write minimal implementation**
```vue
// apps/frontend/src/features/sources/SourceForm.vue
// ... imports
import { useSettingsStore } from '@/features/settings/settings.store'
import { 
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle 
} from '@/components/ui/dialog'
import { useRouter } from 'vue-router'

// ... inside script setup
const settingsStore = useSettingsStore()
const router = useRouter()
const showApiKeyAlert = ref(false)

async function submit() {
  // Check Gemini API Key
  if (!settingsStore.geminiApiKey) {
    showApiKeyAlert.value = true
    return
  }
  
  // ... rest of existing submit logic
}

function goToSettings() {
  showApiKeyAlert.value = false
  router.push('/settings')
}

// ... inside template (add Dialog at the end)
<Dialog v-model:open="showApiKeyAlert">
  <DialogContent>
    <DialogHeader>
      <DialogTitle>Gemini API Key Required</DialogTitle>
      <DialogDescription>
        You need to configure your Gemini API Key in the settings before you can ingest content. This key is required for parsing and embedding the data.
      </DialogDescription>
    </DialogHeader>
    <DialogFooter>
      <Button variant="outline" @click="showApiKeyAlert = false">
        Cancel
      </Button>
      <Button @click="goToSettings">
        Go to Settings
      </Button>
    </DialogFooter>
  </DialogContent>
</Dialog>
```

**Step 4: Verify test passes**
Run: `npm run test apps/frontend/src/features/sources/SourceForm.spec.ts`
Expected: PASS

### Task 3: Update Crawl Depth Hints

**Files:**
- Modify: `apps/frontend/src/features/sources/SourceForm.vue`

**Requirements:**
- **Acceptance Criteria**
  1. Crawl depth hint shows option for 2+.
  2. Hint text suggests "Use with caution" for depth 2+.

- **Functional Requirements**
  1. Update static text in `SourceForm.vue`.

- **Non-Functional Requirements**
  1. Keep UI clean, avoid clutter.

- **Test Coverage**
  - [Unit] `SourceForm.spec.ts` - Verify hint text content.

**Step 1: Write failing test**
```typescript
// apps/frontend/src/features/sources/SourceForm.spec.ts
// Add to existing describe block
it('displays correct crawl depth hints', () => {
  const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: { Dialog: true, DialogContent: true, DialogTrigger: true }
      }
    })
  // Open advanced settings
  wrapper.vm.showAdvanced = true
  await wrapper.nextTick()
  
  const hintText = wrapper.find('.text-xs.text-muted-foreground').text()
  expect(hintText).toContain('2+ = Deep recursive (Use with caution)')
})
```

**Step 2: Verify test fails**
Run: `npm run test apps/frontend/src/features/sources/SourceForm.spec.ts`
Expected: FAIL (Text not found)

**Step 3: Write minimal implementation**
```vue
// apps/frontend/src/features/sources/SourceForm.vue
// Replace existing hint paragraph
<p class="text-xs text-muted-foreground">
  0 = Single page only<br>
  1 = Direct links (Recommended)<br>
  2+ = Deep recursive (Use with caution)
</p>
```

**Step 4: Verify test passes**
Run: `npm run test apps/frontend/src/features/sources/SourceForm.spec.ts`
Expected: PASS

### Task 4: Add Info Box to Gemini API Key Setting

**Files:**
- Modify: `apps/frontend/src/features/settings/Settings.vue`

**Requirements:**
- **Acceptance Criteria**
  1. Tooltip (HelpCircle icon) appears next to "Gemini API Key" label.
  2. Hovering shows "Key can be updated dynamically" message.

- **Functional Requirements**
  1. Use `Tooltip` components from `@/components/ui/tooltip`.

- **Non-Functional Requirements**
  1. Consistent with "Search Balance" tooltip style.

- **Test Coverage**
  - [Unit] `Settings.spec.ts` - Verify Tooltip exists and contains text.

**Step 1: Write failing test**
```typescript
// apps/frontend/src/features/settings/Settings.spec.ts
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import Settings from './Settings.vue'

describe('Settings', () => {
  it('shows info tooltip for gemini api key', async () => {
    const wrapper = mount(Settings, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: { 
          Tooltip: true, 
          TooltipTrigger: true, 
          TooltipContent: true,
          TooltipProvider: true
        } 
      }
    })
    
    // Check if HelpCircle icon exists in the Gemini Key section
    // We might need a more specific selector if there are multiple HelpCircles
    // Assuming it's the first one or we can find it by label parent
    const labels = wrapper.findAll('label')
    const geminiLabel = labels.find(l => l.text().includes('Gemini API Key'))
    expect(geminiLabel).toBeDefined()
    
    // Verify tooltip logic (simplified for stubbed components)
    // We verify the structure exists
    expect(wrapper.html()).toContain('The key can be updated dynamically') 
  })
})
```

**Step 2: Verify test fails**
Run: `npm run test apps/frontend/src/features/settings/Settings.spec.ts`
Expected: FAIL (Text not found)

**Step 3: Write minimal implementation**
```vue
// apps/frontend/src/features/settings/Settings.vue
// Wrap label in div and add Tooltip
<div class="flex items-center gap-2">
  <label for="geminiKey" class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">Gemini API Key</label>
  <TooltipProvider>
    <Tooltip>
      <TooltipTrigger as-child>
        <HelpCircle class="h-4 w-4 text-muted-foreground cursor-help hover:text-foreground transition-colors" />
      </TooltipTrigger>
      <TooltipContent>
        <p class="max-w-xs">The key can be updated dynamically without restarting.</p>
      </TooltipContent>
    </Tooltip>
  </TooltipProvider>
</div>
```

**Step 4: Verify test passes**
Run: `npm run test apps/frontend/src/features/settings/Settings.spec.ts`
Expected: PASS

