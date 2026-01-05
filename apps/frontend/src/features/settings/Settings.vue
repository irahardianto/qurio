<script setup lang="ts">
import { onMounted } from 'vue'
import { useSettingsStore } from './settings.store'
import { Save, Loader2, HelpCircle } from 'lucide-vue-next'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'


const store = useSettingsStore()

const handleUpdateSettings = async () => {
  try {
    await store.updateSettings()
  } catch (error) {
    // Error is handled in store state usually, but catching here prevents unhandled rejection
    console.error('Update failed', error)
  }
}

onMounted(async () => {
  try {
    await store.fetchSettings()
  } catch (error) {
    console.error('Fetch failed', error)
  }
})
</script>

<template>
  <div class="space-y-6 max-w-lg">
    <div
      v-if="store.error"
      class="p-3 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-md"
    >
      {{ store.error }}
    </div>
    <div
      v-if="store.successMessage"
      class="p-3 text-sm text-emerald-500 bg-emerald-500/10 border border-emerald-500/20 rounded-md"
    >
      {{ store.successMessage }}
    </div>

    <div class="space-y-2">
      <label for="geminiKey" class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">Gemini API Key</label>
      <Input
        id="geminiKey"
        v-model="store.geminiApiKey"
        type="password"
        placeholder="Enter Gemini API Key"
        class="font-mono"
      />
      <p class="text-[0.8rem] text-muted-foreground">
        Required for generating embeddings via Google AI Studio.
      </p>
    </div>

    <div class="space-y-4">
      <div class="space-y-2">
        <div class="flex items-center gap-2">
           <label for="searchAlpha" class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
            Search Balance: <span class="font-mono text-primary">{{ store.searchAlpha }}</span>
          </label>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger as-child>
                <HelpCircle class="h-4 w-4 text-muted-foreground cursor-help hover:text-foreground transition-colors" />
              </TooltipTrigger>
              <TooltipContent>
                <p class="max-w-xs">Adjusts importance of Keyword vs Vector search.<br>0.0 = Exact Match<br>1.0 = Conceptual Match</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
        <input 
          id="searchAlpha" 
          v-model.number="store.searchAlpha" 
          type="range" 
          min="0" 
          max="1" 
          step="0.1"
          class="w-full h-2 bg-secondary rounded-lg appearance-none cursor-pointer accent-primary"
        >
        <div class="flex justify-between text-xs text-muted-foreground font-mono">
          <span>Exact (0.0)</span>
          <span>Conceptual (1.0)</span>
        </div>
      </div>

      <div class="space-y-2">
        <div class="flex items-center gap-2">
          <label for="searchTopK" class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">Max Results</label>
           <TooltipProvider>
            <Tooltip>
              <TooltipTrigger as-child>
                <HelpCircle class="h-4 w-4 text-muted-foreground cursor-help hover:text-foreground transition-colors" />
              </TooltipTrigger>
              <TooltipContent>
                <p>Maximum number of document chunks to retrieve per search.</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
        <Input
          id="searchTopK"
          v-model.number="store.searchTopK"
          type="number"
          class="font-mono"
        />
      </div>
    </div>

    <div class="space-y-2">
      <label for="provider" class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">Rerank Provider</label>
      <Select v-model="store.rerankProvider">
        <SelectTrigger class="w-full">
          <SelectValue placeholder="Select a provider" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="none">None</SelectItem>
          <SelectItem value="jina">Jina AI</SelectItem>
          <SelectItem value="cohere">Cohere</SelectItem>
        </SelectContent>
      </Select>
      <p class="text-[0.8rem] text-muted-foreground">
        Select an external provider to re-rank search results for better accuracy.
      </p>
    </div>

    <div
      v-if="store.rerankProvider !== 'none'"
      class="space-y-2 animate-in slide-in-from-top-2 fade-in duration-200"
    >
      <label for="apiKey" class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">Rerank API Key</label>
      <Input
        id="apiKey"
        v-model="store.rerankApiKey"
        type="password"
        placeholder="Enter Provider API Key"
        class="font-mono"
      />
    </div>

    <Button 
      :disabled="store.isLoading" 
      @click="handleUpdateSettings"
      class="w-full sm:w-auto"
    >
      <Loader2
        v-if="store.isLoading"
        class="mr-2 h-4 w-4 animate-spin"
      />
      <Save
        v-else
        class="mr-2 h-4 w-4"
      />
      <span>{{ store.isLoading ? 'Saving...' : 'Save Configuration' }}</span>
    </Button>
  </div>
</template>