<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSourceStore, type SourcePage } from '../features/sources/source.store'
import { ArrowLeft, Database, FileText, Layers, Hash, Braces, ExternalLink, Copy } from 'lucide-vue-next'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import SourceProgress from '../features/sources/SourceProgress.vue'

const route = useRoute()
const router = useRouter()
const store = useSourceStore()

const source = ref<any>(null)
const pages = ref<SourcePage[]>([])
const isLoading = ref(true)
const selectedChunk = ref<any>(null)
let pollingInterval: any = null

const fetchPages = async () => {
  const id = route.params.id as string
  if (id) {
    pages.value = await store.getSourcePages(id)
  }
}

onMounted(async () => {
  const id = route.params.id as string
  if (id) {
    source.value = await store.getSource(id)
    await fetchPages()
    isLoading.value = false
    
    // Select first chunk by default if available
    if (source.value?.chunks && source.value.chunks.length > 0) {
      selectedChunk.value = source.value.chunks[0]
    }

    // Poll if active
    if (source.value?.status === 'in_progress' || source.value?.status === 'pending' || source.value?.status === 'processing') {
      pollingInterval = setInterval(async () => {
        source.value = await store.getSource(id) // Update status
        await fetchPages()
        
        // Stop polling if done
        if (source.value?.status === 'completed' || source.value?.status === 'failed') {
          clearInterval(pollingInterval)
        }
      }, 2000)
    }
  }
})

// Watch for chunk updates to preserve selection or auto-select
watch(() => source.value?.chunks, (newChunks) => {
  if (newChunks && newChunks.length > 0 && !selectedChunk.value) {
    selectedChunk.value = newChunks[0]
  }
})

onUnmounted(() => {
  if (pollingInterval) clearInterval(pollingInterval)
})

const copyToClipboard = (text: string) => {
  navigator.clipboard.writeText(text)
  // Toast notification could be added here
}
</script>

<template>
  <div class="space-y-6 w-full p-6 lg:p-10 animate-in fade-in duration-500 h-[calc(100vh-4rem)] flex flex-col">
    <!-- Header -->
    <div class="flex items-center space-x-4 border-b border-border pb-4 flex-shrink-0">
      <Button
        variant="ghost"
        size="icon"
        @click="router.back()"
        class="h-10 w-10 rounded-full hover:bg-muted"
      >
        <ArrowLeft class="h-5 w-5" />
      </Button>
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-foreground font-mono">
          Source Details
        </h1>
        <p
          v-if="source"
          class="text-muted-foreground font-mono mt-1 text-sm"
        >
          ID: {{ source.id }}
        </p>
      </div>
    </div>

    <div
      v-if="isLoading"
      class="flex flex-col items-center justify-center p-24 space-y-4"
    >
      <div class="animate-spin h-10 w-10 border-2 border-primary border-t-transparent rounded-full" />
      <span class="text-muted-foreground font-mono text-sm animate-pulse">Retrieving Metadata...</span>
    </div>

    <div
      v-else-if="source"
      class="grid gap-6 md:grid-cols-3 flex-1 min-h-0"
    >
      <!-- Metadata Column (Scrollable) -->
      <div class="md:col-span-1 flex flex-col gap-6 overflow-y-auto pr-2">
        <Card class="bg-card/50 backdrop-blur-sm border-border shadow-sm flex-shrink-0">
          <CardHeader class="pb-3 border-b border-border/50">
            <CardTitle class="flex items-center gap-2 text-base font-semibold">
              <Database class="h-4 w-4 text-primary" />
              Metadata
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-4 pt-4">
            <div class="space-y-1">
              <span class="text-xs text-muted-foreground uppercase tracking-wider font-semibold">Source URL</span>
              <div class="flex items-center gap-2">
                <a 
                  v-if="source.type === 'web'"
                  :href="source.url" 
                  target="_blank" 
                  class="text-sm font-medium hover:underline text-primary truncate block"
                  :title="source.url"
                >
                  {{ source.url }} <ExternalLink class="inline h-3 w-3 ml-0.5 mb-0.5" />
                </a>
                 <span v-else class="text-sm font-medium truncate block" :title="source.url">
                  {{ (source.url?.split('/').pop() ?? '').split('_').slice(1).join('_') }}
                </span>
              </div>
            </div>
            
            <div class="grid grid-cols-2 gap-4">
               <div class="space-y-1">
                <span class="text-xs text-muted-foreground uppercase tracking-wider font-semibold">Status</span>
                <div><StatusBadge :status="source.status" /></div>
              </div>
              <div class="space-y-1">
                <span class="text-xs text-muted-foreground uppercase tracking-wider font-semibold">Chunks</span>
                <div class="font-mono text-sm">{{ source.total_chunks }}</div>
              </div>
            </div>
            
            <div
              v-if="source.max_depth > 0 || (source.exclusions && source.exclusions.length > 0)"
              class="pt-4 border-t border-border/50 space-y-3"
            >
              <div v-if="source.max_depth > 0" class="flex justify-between items-center">
                <span class="text-xs text-muted-foreground uppercase tracking-wider font-semibold">Crawl Depth</span>
                <span class="text-sm font-mono bg-secondary px-2 py-0.5 rounded">{{ source.max_depth }}</span>
              </div>
              <div
                v-if="source.exclusions?.length"
                class="space-y-2"
              >
                <span class="text-xs text-muted-foreground uppercase tracking-wider font-semibold block">Exclusions</span>
                <div class="flex flex-wrap gap-1.5">
                  <Badge
                    v-for="ex in source.exclusions"
                    :key="ex"
                    variant="outline"
                    class="text-[10px] font-mono bg-background/50"
                  >
                    {{ ex }}
                  </Badge>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Progress Section -->
        <div 
          v-if="source.type === 'web' || pages.length > 0" 
          class="flex-shrink-0"
        >
          <SourceProgress :pages="pages" />
        </div>
      </div>

      <!-- Master-Detail Chunks View -->
      <div class="md:col-span-2 flex flex-col h-full min-h-[500px] border border-border rounded-xl bg-card/30 backdrop-blur-sm shadow-sm overflow-hidden">
        <!-- Header -->
        <div class="flex items-center justify-between px-4 py-3 border-b border-border bg-secondary/10">
          <h3 class="text-sm font-semibold flex items-center gap-2">
            <Layers class="h-4 w-4 text-primary" />
            Ingested Chunks
            <Badge variant="secondary" class="ml-2 text-xs">{{ source.chunks?.length || 0 }}</Badge>
          </h3>
        </div>

        <div class="flex flex-1 min-h-0">
           <!-- Master List (Left Column) -->
           <div class="w-1/3 border-r border-border overflow-y-auto flex flex-col bg-background/20">
             <div v-if="!source.chunks || source.chunks.length === 0" class="p-8 text-center text-muted-foreground text-sm">
               No chunks indexed.
             </div>
             <button
               v-for="(chunk, i) in source.chunks"
               :key="i"
               @click="selectedChunk = chunk"
               class="flex flex-col gap-1 p-3 text-left border-b border-border/50 hover:bg-muted/50 transition-colors focus:outline-none"
               :class="{ 'bg-primary/10 border-l-2 border-l-primary': selectedChunk === chunk, 'border-l-2 border-l-transparent': selectedChunk !== chunk }"
             >
               <div class="flex items-center justify-between mb-1">
                 <span class="font-mono text-xs text-muted-foreground flex items-center gap-1">
                   <Hash class="h-3 w-3" /> {{ chunk.chunk_index }}
                 </span>
                 <div class="flex items-center gap-1">
                    <Badge v-if="chunk.type" variant="outline" class="text-[9px] px-1 py-0 h-4 uppercase">{{ chunk.type }}</Badge>
                    <span class="text-[10px] text-muted-foreground bg-secondary px-1.5 py-0.5 rounded">
                      {{ chunk.content?.length || 0 }}
                    </span>
                 </div>
               </div>
               <div v-if="chunk.title" class="text-xs font-semibold truncate text-foreground/80 mb-0.5">
                 {{ chunk.title }}
               </div>
               <div class="text-xs font-medium truncate opacity-70">
                 {{ chunk.content?.substring(0, 50) }}...
               </div>
             </button>
           </div>

           <!-- Detail View (Right Column) -->
           <div class="flex-1 overflow-y-auto bg-card/10 p-6">
             <div v-if="selectedChunk" class="space-y-6">
               <div class="flex items-start justify-between border-b border-border/50 pb-4">
                 <div class="space-y-1">
                   <div class="flex items-center gap-2">
                     <h4 class="text-lg font-bold font-mono text-primary flex items-center gap-2">
                       Chunk #{{ selectedChunk.chunk_index }}
                     </h4>
                     <Badge v-if="selectedChunk.type" variant="secondary" class="font-mono text-xs uppercase">
                        {{ selectedChunk.type }}
                     </Badge>
                     <Badge v-if="selectedChunk.language" variant="outline" class="font-mono text-xs">
                        {{ selectedChunk.language }}
                     </Badge>
                   </div>
                   
                   <div v-if="selectedChunk.title" class="text-base font-semibold text-foreground/90">
                      {{ selectedChunk.title }}
                   </div>

                   <div class="flex items-center gap-2 text-sm text-muted-foreground">
                     <FileText class="h-4 w-4" />
                     <span class="truncate max-w-[300px]" :title="selectedChunk.source_url">
                       {{ selectedChunk.source_url }}
                     </span>
                   </div>
                 </div>
                 <Button variant="ghost" size="sm" @click="copyToClipboard(selectedChunk.content)" title="Copy Content">
                   <Copy class="h-4 w-4" />
                 </Button>
               </div>

               <div class="space-y-2">
                 <label class="text-xs uppercase tracking-wider font-semibold text-muted-foreground flex items-center gap-2">
                   <Braces class="h-3 w-3" /> Content
                 </label>
                 <div class="relative group">
                   <div class="absolute right-2 top-2 opacity-0 group-hover:opacity-100 transition-opacity">
                     <!-- Floating actions could go here -->
                   </div>
                   <pre class="whitespace-pre-wrap font-mono text-sm leading-relaxed bg-background/50 p-4 rounded-lg border border-border text-foreground/90 overflow-x-auto">{{ selectedChunk.content }}</pre>
                 </div>
               </div>
             </div>
             
             <div v-else class="h-full flex flex-col items-center justify-center text-muted-foreground">
               <Layers class="h-12 w-12 opacity-20 mb-4" />
               <p>Select a chunk from the list to view details.</p>
             </div>
           </div>
        </div>
      </div>
    </div>
  </div>
</template>