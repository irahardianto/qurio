<script setup lang="ts">
import { ref } from 'vue'
import { useSourceStore } from './source.store'
import { Plus, Loader2, ChevronDown, ChevronUp } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

const store = useSourceStore()
const url = ref('')
const maxDepth = ref(0)
const exclusions = ref('')
const showAdvanced = ref(false)

const emit = defineEmits(['submit'])

async function submit() {
  if (!url.value) return
  
  // Basic URL validation
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
}
</script>

<template>
  <form @submit.prevent="submit" class="w-full space-y-4 p-4 border rounded-lg bg-card text-card-foreground shadow-sm">
    <div class="flex flex-col space-y-2">
      <div class="flex w-full items-center space-x-2">
         <Input 
           v-model="url" 
           type="text" 
           placeholder="https://docs.example.com" 
           :disabled="store.isLoading" 
           class="flex-1"
         />
         <Button type="submit" :disabled="store.isLoading">
           <Loader2 v-if="store.isLoading" class="mr-2 h-4 w-4 animate-spin" />
           <Plus v-else class="mr-2 h-4 w-4" />
           Ingest
         </Button>
      </div>
      
      <div class="flex items-center">
        <button type="button" @click="showAdvanced = !showAdvanced" class="text-sm text-muted-foreground hover:text-foreground flex items-center transition-colors">
          <ChevronDown v-if="!showAdvanced" class="h-4 w-4 mr-1" />
          <ChevronUp v-else class="h-4 w-4 mr-1" />
          Advanced Configuration
        </button>
      </div>

      <div v-if="showAdvanced" class="space-y-4 pt-4 border-t animate-in slide-in-from-top-2 fade-in duration-200">
        <div class="grid w-full max-w-sm items-center gap-1.5">
           <label class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">Crawl Depth</label>
           <Input v-model.number="maxDepth" type="number" min="0" max="5" placeholder="0" />
           <p class="text-[0.8rem] text-muted-foreground">0 = Single page, 1 = Direct links, etc.</p>
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

    <div v-if="store.error" class="text-destructive text-sm font-mono flex items-center gap-2">
      <span>!</span>
      {{ store.error }}
    </div>
  </form>
</template>