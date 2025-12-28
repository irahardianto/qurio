<script setup lang="ts">
import { onMounted } from 'vue'
import { useJobStore } from '../features/jobs/job.store'
import { RefreshCw, CheckCircle, AlertOctagon, Terminal } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

const jobStore = useJobStore()

onMounted(() => {
  jobStore.fetchFailedJobs()
})

const formatDate = (date: string) => {
  return new Date(date).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}
</script>

<template>
  <div class="space-y-6 w-full p-6 lg:p-10">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">System Monitor</h1>
        <p class="text-muted-foreground mt-1 flex items-center gap-2">
          <Terminal class="w-4 h-4" />
          Ingestion Failure Logs
        </p>
      </div>
      <Button variant="outline" @click="jobStore.fetchFailedJobs" :disabled="jobStore.isLoading">
        <RefreshCw class="mr-2 h-4 w-4" :class="{ 'animate-spin': jobStore.isLoading }" />
        Refresh Logs
      </Button>
    </div>

    <div v-if="jobStore.jobs.length === 0 && !jobStore.isLoading" class="flex flex-col items-center justify-center p-12 bg-card/50 border border-border rounded-lg border-dashed">
      <CheckCircle class="h-12 w-12 text-emerald-500 mb-4" />
      <h3 class="text-lg font-medium text-foreground">All Systems Operational</h3>
      <p class="text-muted-foreground">No ingestion failures detected in the logs.</p>
    </div>

    <div v-else class="rounded-md border border-border overflow-hidden bg-card/50 backdrop-blur-sm">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead class="text-xs text-muted-foreground uppercase bg-secondary/50 border-b border-border">
            <tr>
              <th class="px-6 py-3 font-mono">Job ID / Source</th>
              <th class="px-6 py-3 font-mono">Timestamp</th>
              <th class="px-6 py-3 font-mono">Error Log</th>
              <th class="px-6 py-3 font-mono">Status</th>
              <th class="px-6 py-3 font-mono text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="job in jobStore.jobs" :key="job.id" class="hover:bg-muted/50 transition-colors">
              <td class="px-6 py-4 font-mono">
                <div class="flex flex-col">
                  <span class="text-primary font-medium">{{ job.id.substring(0, 8) }}...</span>
                  <span class="text-xs text-muted-foreground">Source: {{ job.source_id }}</span>
                </div>
              </td>
              <td class="px-6 py-4 font-mono text-muted-foreground whitespace-nowrap">
                {{ formatDate(job.created_at) }}
              </td>
              <td class="px-6 py-4">
                <div class="flex items-start gap-2 max-w-md">
                  <AlertOctagon class="w-4 h-4 text-destructive flex-shrink-0 mt-0.5" />
                  <code class="text-xs text-destructive/90 font-mono bg-destructive/10 px-1 py-0.5 rounded break-all line-clamp-2 hover:line-clamp-none transition-all">
                    {{ job.error }}
                  </code>
                </div>
              </td>
              <td class="px-6 py-4">
                <Badge variant="destructive" class="font-mono text-xs uppercase">
                  Failed ({{ job.retries }})
                </Badge>
              </td>
              <td class="px-6 py-4 text-right">
                <Button size="sm" variant="ghost" class="h-8 px-2" @click="jobStore.retryJob(job.id)" :disabled="jobStore.isLoading">
                  <RefreshCw class="w-4 h-4 text-primary hover:text-primary/80" />
                  <span class="sr-only">Retry</span>
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>