<script setup lang="ts">
import { ref } from 'vue'
import { useSourceStore } from './source.store'
import { Plus, Loader2, ChevronDown, ChevronUp, Globe, FileUp } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

const store = useSourceStore()
const url = ref('')
const maxDepth = ref(0)
const exclusions = ref('')
const showAdvanced = ref(false)
const activeTab = ref<'web' | 'file'>('web')
const file = ref<File | null>(null)

const emit = defineEmits(['submit'])

async function submit() {
  if (activeTab.value === 'web') {
    if (!url.value) return
    
    try {
      new URL(url.value)
    } catch {
      alert('Please enter a valid URL (e.g., https://docs.example.com)')
      return
    }

    const exclusionsList = exclusions.value
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)

    await store.addSource({
      name: url.value, 
      url: url.value,
      max_depth: maxDepth.value,
      exclusions: exclusionsList
    })

    if (!store.error) {
      url.value = ''
      maxDepth.value = 0
      exclusions.value = ''
      showAdvanced.value = false
      emit('submit')
    }
  } else {
    if (file.value) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const filePayload = file.value as any
      await store.uploadSource(filePayload)
      if (!store.error) {
        file.value = null
        emit('submit')
      }
    }
  }
}

function onFileChange(e: Event) {
  const target = e.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    file.value = target.files[0]
  }
}
</script>

<template>
  <div class="w-full border rounded-lg bg-card text-card-foreground shadow-sm overflow-hidden">
    <div class="flex border-b bg-muted/30">
      <button 
        class="flex-1 p-3 text-sm font-medium flex items-center justify-center gap-2 transition-colors hover:bg-muted/50"
        :class="activeTab === 'web' ? 'bg-background text-foreground border-b-2 border-primary shadow-sm' : 'text-muted-foreground'"
        @click="activeTab = 'web'"
      >
        <Globe class="h-4 w-4" /> Web Source
      </button>
      <button 
        class="flex-1 p-3 text-sm font-medium flex items-center justify-center gap-2 transition-colors hover:bg-muted/50"
        :class="activeTab === 'file' ? 'bg-background text-foreground border-b-2 border-primary shadow-sm' : 'text-muted-foreground'"
        @click="activeTab = 'file'"
      >
        <FileUp class="h-4 w-4" /> File Upload
      </button>
    </div>

    <form
      class="p-4 space-y-4"
      @submit.prevent="submit"
    >
      <!-- Web Form -->
      <div
        v-if="activeTab === 'web'"
        class="space-y-4"
      >
        <div class="flex flex-col space-y-2">
          <div class="flex w-full items-center space-x-2">
            <Input 
              v-model="url" 
              type="text" 
              placeholder="https://docs.example.com" 
              :disabled="store.isLoading" 
              class="flex-1"
            />
            <Button
              type="submit"
              :disabled="store.isLoading"
            >
              <Loader2
                v-if="store.isLoading"
                class="mr-2 h-4 w-4 animate-spin"
              />
              <Plus
                v-else
                class="mr-2 h-4 w-4"
              />
              Ingest
            </Button>
          </div>
          
          <div class="flex items-center">
            <button
              type="button"
              class="text-sm text-muted-foreground hover:text-foreground flex items-center transition-colors"
              @click="showAdvanced = !showAdvanced"
            >
              <ChevronDown
                v-if="!showAdvanced"
                class="h-4 w-4 mr-1"
              />
              <ChevronUp
                v-else
                class="h-4 w-4 mr-1"
              />
              Advanced Configuration
            </button>
          </div>

          <div
            v-if="showAdvanced"
            class="space-y-4 pt-4 border-t animate-in slide-in-from-top-2 fade-in duration-200"
          >
            <div class="grid w-full max-w-sm items-center gap-1.5">
              <label class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">Crawl Depth</label>
              <Input
                v-model.number="maxDepth"
                type="number"
                min="0"
                max="5"
                placeholder="0"
              />
              <p class="text-[0.8rem] text-muted-foreground">
                0 = Single page, 1 = Direct links, etc.
              </p>
            </div>
            
            <div class="grid w-full gap-1.5">
              <label class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">Exclusions (Regex per line)</label>
              <Textarea 
                v-model="exclusions" 
                placeholder="/login&#10;/private"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- File Form -->
      <div
        v-else
        class="space-y-4"
      >
        <div class="grid w-full gap-1.5">
          <label class="text-sm font-medium">Select Document</label>
          <Input
            type="file"
            accept=".pdf,.md,.txt,.html"
            class="cursor-pointer"
            @change="onFileChange"
          />
          <p class="text-[0.8rem] text-muted-foreground">
            Supported: PDF, Markdown, Text, HTML (Max 50MB)
          </p>
        </div>
        <Button
          type="submit"
          :disabled="store.isLoading || !file"
          class="w-full sm:w-auto"
        >
          <Loader2
            v-if="store.isLoading"
            class="mr-2 h-4 w-4 animate-spin"
          />
          <Plus
            v-else
            class="mr-2 h-4 w-4"
          />
          Upload & Ingest
        </Button>
      </div>

      <div
        v-if="store.error"
        class="text-destructive text-sm font-mono flex items-center gap-2 pt-2 border-t"
      >
        <span>!</span>
        {{ store.error }}
      </div>
    </form>
  </div>
</template>
