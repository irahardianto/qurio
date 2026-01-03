<script setup lang="ts">
import { useSourceStore } from './source.store'
import { onMounted, onUnmounted } from 'vue'
import { RefreshCw, Trash2, ExternalLink, FileText } from 'lucide-vue-next'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'

const store = useSourceStore()

const handleDelete = async (id: string) => {
  if (confirm('Are you sure you want to delete this source?')) {
    await store.deleteSource(id)
  }
}

const handleResync = async (id: string) => {
  await store.resyncSource(id)
}

onMounted(() => {
  store.fetchSources()
  store.startPolling()
})

onUnmounted(() => {
  store.stopPolling()
})
</script>

<template>
  <div class="space-y-4">
    <div
      v-if="store.isLoading && store.sources.length === 0"
      class="text-center p-8 text-muted-foreground border border-dashed rounded-lg bg-muted/10"
    >
      <div class="animate-spin h-6 w-6 border-2 border-primary border-t-transparent rounded-full mx-auto mb-4" />
      <span>Retrieving knowledge sources...</span>
    </div>
    
    <div
      v-else-if="store.sources.length === 0"
      class="text-center p-8 text-muted-foreground border border-dashed rounded-lg bg-muted/10"
    >
      No sources configured. Ingest documentation to begin.
    </div>

    <div
      v-else
      class="grid gap-4 md:grid-cols-2 lg:grid-cols-3"
    >
      <Card
        v-for="source in store.sources"
        :key="source.id"
        class="bg-card"
      >
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle
            class="text-sm font-medium truncate pr-4 flex-1"
            :title="source.url"
          >
            {{ source.type === 'file' && source.url ? (source.url?.split('/').pop() ?? '').split('_').slice(1).join('_') : source.url }}
          </CardTitle>
          <a
            v-if="source.type !== 'file'"
            :href="source.url"
            target="_blank"
            class="text-muted-foreground hover:text-primary transition-colors"
          >
            <ExternalLink :size="14" />
          </a>
          <span
            v-else
            class="text-muted-foreground"
          >
            <FileText :size="14" />
          </span>
        </CardHeader>
        <CardContent>
          <div class="text-xs text-muted-foreground font-mono mb-4">
            ID: {{ source.id.substring(0, 8) }}
          </div>
          <StatusBadge :status="source.status || 'pending'" />
        </CardContent>
        <CardFooter class="flex justify-end gap-2">
          <Button
            variant="ghost"
            size="sm"
            class="text-xs"
            @click="$router.push(`/sources/${source.id}`)"
          >
            View Details
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title="Re-sync"
            @click="handleResync(source.id)"
          >
            <RefreshCw :size="16" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            class="text-destructive hover:text-destructive hover:bg-destructive/10"
            title="Delete"
            @click="handleDelete(source.id)"
          >
            <Trash2 :size="16" />
          </Button>
        </CardFooter>
      </Card>
    </div>
  </div>
</template>