<script setup lang="ts">
import { ref } from 'vue'
import { useSourceStore } from './source.store'
import { Plus, Loader2, ChevronDown, ChevronUp, Globe, FileUp, Settings2, UploadCloud } from 'lucide-vue-next'
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
const isDragging = ref(false)

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
    file.value = target.files[0] || null
  }
}

function onDrop(e: DragEvent) {
  isDragging.value = false
  const droppedFiles = e.dataTransfer?.files
  if (droppedFiles && droppedFiles.length > 0) {
    file.value = droppedFiles[0] || null
  }
}
</script>

<template>
  <div class="w-full bg-card border border-border rounded-xl shadow-sm overflow-hidden transition-all duration-300 hover:shadow-[0_0_20px_rgba(59,130,246,0.05)]">
    <!-- Tab Navigation -->
    <div class="flex border-b border-border">
      <button 
        class="flex-1 py-4 text-sm font-medium flex items-center justify-center gap-2 transition-all duration-200"
        :class="activeTab === 'web' 
          ? 'bg-background text-primary border-b-2 border-primary' 
          : 'bg-muted/30 text-muted-foreground hover:bg-muted/50 hover:text-foreground'"
        @click="activeTab = 'web'"
      >
        <Globe class="h-4 w-4" /> 
        <span>Web Crawler</span>
      </button>
      <button 
        class="flex-1 py-4 text-sm font-medium flex items-center justify-center gap-2 transition-all duration-200"
        :class="activeTab === 'file' 
          ? 'bg-background text-primary border-b-2 border-primary' 
          : 'bg-muted/30 text-muted-foreground hover:bg-muted/50 hover:text-foreground'"
        @click="activeTab = 'file'"
      >
        <FileUp class="h-4 w-4" /> 
        <span>File Upload</span>
      </button>
    </div>

    <form class="p-6 md:p-8 space-y-6" @submit.prevent="submit">
      <!-- Web Form -->
      <div v-if="activeTab === 'web'" class="space-y-6">
        <div class="flex flex-col space-y-4">
          <div class="relative group">
            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
              <Globe class="h-5 w-5 text-muted-foreground group-focus-within:text-primary transition-colors" />
            </div>
            <Input 
              v-model="url" 
              type="text" 
              placeholder="https://docs.example.com" 
              :disabled="store.isLoading" 
              class="pl-12 h-14 text-lg font-mono bg-background/50 focus:bg-background transition-all shadow-sm border-muted-foreground/20 focus:border-primary"
            />
          </div>
          
          <Button
            type="submit"
            :disabled="store.isLoading"
            class="w-full h-12 text-base font-semibold shadow-md hover:shadow-lg transition-all"
            size="lg"
          >
            <Loader2 v-if="store.isLoading" class="mr-2 h-5 w-5 animate-spin" />
            <Plus v-else class="mr-2 h-5 w-5" />
            Ingest Knowledge Source
          </Button>

          <!-- Advanced Toggle -->
          <div class="pt-2">
            <button
              type="button"
              class="text-sm text-muted-foreground hover:text-primary flex items-center gap-2 transition-colors mx-auto"
              @click="showAdvanced = !showAdvanced"
            >
              <Settings2 class="h-4 w-4" />
              <span>{{ showAdvanced ? 'Hide Configuration' : 'Advanced Configuration' }}</span>
              <ChevronDown v-if="!showAdvanced" class="h-3 w-3" />
              <ChevronUp v-else class="h-3 w-3" />
            </button>
          </div>

          <!-- Advanced Settings -->
          <div
            v-if="showAdvanced"
            class="grid md:grid-cols-2 gap-6 p-6 bg-muted/20 rounded-lg border border-border/50 animate-in slide-in-from-top-2 fade-in duration-200"
          >
            <div class="space-y-2">
              <label class="text-sm font-medium leading-none text-foreground">Crawl Depth</label>
              <Input
                v-model.number="maxDepth"
                type="number"
                min="0"
                max="5"
                class="font-mono"
              />
              <p class="text-xs text-muted-foreground">
                0 = Single page only<br>
                1 = Follow direct links (Recommended)
              </p>
            </div>
            
            <div class="space-y-2">
              <label class="text-sm font-medium leading-none text-foreground">Exclusions (Regex)</label>
              <Textarea 
                v-model="exclusions" 
                placeholder="/login&#10;/private"
                class="font-mono min-h-[80px]"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- File Form -->
      <div v-else class="space-y-6">
        <div 
          class="border-2 border-dashed rounded-xl p-8 transition-all text-center relative"
          :class="[
            isDragging ? 'border-primary bg-primary/5 scale-[1.02]' : 'border-muted-foreground/25 hover:bg-muted/10 hover:border-primary/50'
          ]"
          @dragover.prevent="isDragging = true"
          @dragleave.prevent="isDragging = false"
          @drop.prevent="onDrop"
        >
          <input
            id="file-upload"
            type="file"
            accept=".pdf,.md,.txt,.html"
            class="hidden"
            @change="onFileChange"
          />
          <label for="file-upload" class="cursor-pointer flex flex-col items-center gap-4 w-full h-full">
            <div 
              class="h-20 w-20 rounded-full flex items-center justify-center transition-colors mb-2"
              :class="isDragging ? 'bg-primary/20 text-primary' : 'bg-primary/10 text-primary'"
            >
              <UploadCloud class="h-10 w-10" :class="{ 'animate-bounce': isDragging }" />
            </div>
            <div class="space-y-1">
              <p class="text-lg font-medium text-foreground">
                <span v-if="file" class="text-primary font-bold">{{ file.name }}</span>
                <span v-else>
                  <span class="text-primary hover:underline">Click to upload</span> or drag and drop
                </span>
              </p>
              <p class="text-sm text-muted-foreground">
                PDF, Markdown, Text, HTML (Max 50MB)
              </p>
            </div>
          </label>
        </div>

        <Button
          type="submit"
          :disabled="store.isLoading || !file"
          class="w-full h-12 text-base font-semibold"
          size="lg"
        >
          <Loader2 v-if="store.isLoading" class="mr-2 h-5 w-5 animate-spin" />
          <Plus v-else class="mr-2 h-5 w-5" />
          Upload & Ingest
        </Button>
      </div>

      <div v-if="store.error" class="bg-destructive/10 border border-destructive/20 rounded-md p-3 flex items-start gap-3">
        <div class="bg-destructive text-destructive-foreground rounded-full p-0.5 mt-0.5">
          <Plus class="h-3 w-3 rotate-45" />
        </div>
        <p class="text-sm text-destructive font-medium">{{ store.error }}</p>
      </div>
    </form>
  </div>
</template>
