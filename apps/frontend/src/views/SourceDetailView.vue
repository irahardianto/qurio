<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSourceStore } from '../features/sources/source.store'
import { ArrowLeft, Database, FileText, Layers } from 'lucide-vue-next'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

const route = useRoute()
const router = useRouter()
const store = useSourceStore()

const source = ref<any>(null)
const isLoading = ref(true)

onMounted(async () => {
  const id = route.params.id as string
  if (id) {
    source.value = await store.getSource(id)
    isLoading.value = false
  }
})
</script>

<template>
  <div class="space-y-6 animate-in fade-in duration-500">
    <!-- Header -->
    <div class="flex items-center space-x-4">
      <Button variant="ghost" size="icon" @click="router.back()">
        <ArrowLeft class="h-4 w-4" />
      </Button>
      <div>
        <h1 class="text-2xl font-bold tracking-tight">Source Details</h1>
        <p class="text-muted-foreground" v-if="source">
          {{ source.type === 'file' && source.url ? (source.url.split('/').pop() ?? '').split('_').slice(1).join('_') : source.url }}
        </p>
      </div>
    </div>

    <div v-if="isLoading" class="flex justify-center p-12">
      <div class="animate-spin h-8 w-8 border-2 border-primary border-t-transparent rounded-full"></div>
    </div>

    <div v-else-if="source" class="grid gap-6 md:grid-cols-3">
      <!-- Metadata Card -->
      <Card class="md:col-span-1 h-fit">
        <CardHeader>
          <CardTitle class="flex items-center gap-2">
            <Database class="h-4 w-4" />
            Metadata
          </CardTitle>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="flex justify-between items-center">
            <span class="text-sm text-muted-foreground">Status</span>
            <StatusBadge :status="source.status" />
          </div>
          <div class="flex justify-between items-center">
            <span class="text-sm text-muted-foreground">ID</span>
            <span class="text-sm font-mono">{{ source.id.substring(0, 8) }}...</span>
          </div>
           <div class="flex justify-between items-center">
            <span class="text-sm text-muted-foreground">Chunks</span>
            <Badge variant="secondary">{{ source.total_chunks }}</Badge>
          </div>
          
           <div v-if="source.max_depth > 0" class="pt-4 border-t space-y-2">
             <div class="flex justify-between items-center">
                <span class="text-sm text-muted-foreground">Max Depth</span>
                <span class="text-sm font-medium">{{ source.max_depth }}</span>
             </div>
             <div v-if="source.exclusions?.length" class="space-y-1">
                <span class="text-sm text-muted-foreground block">Exclusions</span>
                <div class="flex flex-wrap gap-1">
                   <Badge v-for="ex in source.exclusions" :key="ex" variant="outline" class="text-xs">{{ ex }}</Badge>
                </div>
             </div>
           </div>
        </CardContent>
      </Card>

      <!-- Chunks List -->
      <Card class="md:col-span-2">
        <CardHeader>
          <CardTitle class="flex items-center gap-2">
            <Layers class="h-4 w-4" />
            Ingested Chunks
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div v-if="source.chunks && source.chunks.length > 0" class="space-y-4">
             <div v-for="(chunk, i) in source.chunks" :key="i" class="p-4 rounded-lg border bg-muted/5 space-y-2">
                <div class="flex justify-between items-start">
                   <div class="flex items-center gap-2">
                      <FileText class="h-3 w-3 text-muted-foreground" />
                      <a v-if="source.type !== 'file'" :href="chunk.SourceURL" target="_blank" class="text-xs font-medium hover:underline text-primary truncate max-w-[300px]">
                         {{ chunk.SourceURL }}
                      </a>
                      <span v-else class="text-xs font-medium text-muted-foreground truncate max-w-[300px]">
                         {{ (chunk.SourceURL.split('/').pop() ?? '').split('_').slice(1).join('_') }}
                      </span>
                   </div>
                   <Badge variant="outline" class="text-[10px] font-mono">Index: {{ chunk.ChunkIndex }}</Badge>
                </div>
                <p class="text-sm text-muted-foreground line-clamp-3 font-mono bg-background p-2 rounded border">
                   {{ chunk.Content }}
                </p>
             </div>
          </div>
          <div v-else class="text-center p-8 text-muted-foreground">
             No chunks found. Is the ingestion complete?
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
