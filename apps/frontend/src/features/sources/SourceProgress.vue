<script setup lang="ts">
import { computed } from "vue";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Activity, CheckCircle, Clock, AlertCircle } from "lucide-vue-next";
import type { SourcePage } from "./source.store";

const props = defineProps<{
  pages: SourcePage[];
}>();

const stats = computed(() => {
  const total = props.pages.length;
  const completed = props.pages.filter((p) => p.status === "completed").length;
  const processing = props.pages.filter(
    (p) => p.status === "processing",
  ).length;
  const pending = props.pages.filter((p) => p.status === "pending").length;
  const failed = props.pages.filter((p) => p.status === "failed").length;

  const progress = total > 0 ? Math.round((completed / total) * 100) : 0;

  return { total, completed, processing, pending, failed, progress };
});
</script>

<template>
  <Card class="h-full">
    <CardHeader class="pb-2">
      <CardTitle class="flex items-center gap-2 text-lg">
        <Activity class="h-5 w-5 text-primary" />
        Crawl Progress
      </CardTitle>
    </CardHeader>
    <CardContent class="space-y-6">
      <!-- Progress Bar -->
      <div class="space-y-2">
        <div class="flex justify-between text-sm">
          <span class="font-medium">Overall Progress</span>
          <span class="text-muted-foreground">{{ stats.progress }}% ({{ stats.completed }}/{{
            stats.total
          }})</span>
        </div>
        <div class="h-2 w-full bg-secondary rounded-full overflow-hidden">
          <div
            class="h-full bg-primary transition-all duration-500 ease-in-out"
            :style="{ width: `${stats.progress}%` }"
          />
        </div>
      </div>

      <!-- Stats Grid -->
      <div class="grid grid-cols-2 gap-4">
        <div class="flex items-center gap-2 p-3 bg-muted/20 rounded-lg border">
          <Clock class="h-4 w-4 text-blue-500" />
          <div class="flex flex-col">
            <span class="text-xs text-muted-foreground">Pending</span>
            <span class="font-bold">{{ stats.pending }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2 p-3 bg-muted/20 rounded-lg border">
          <Activity class="h-4 w-4 text-yellow-500" />
          <div class="flex flex-col">
            <span class="text-xs text-muted-foreground">Processing</span>
            <span class="font-bold">{{ stats.processing }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2 p-3 bg-muted/20 rounded-lg border">
          <CheckCircle class="h-4 w-4 text-green-500" />
          <div class="flex flex-col">
            <span class="text-xs text-muted-foreground">Completed</span>
            <span class="font-bold">{{ stats.completed }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2 p-3 bg-muted/20 rounded-lg border">
          <AlertCircle class="h-4 w-4 text-red-500" />
          <div class="flex flex-col">
            <span class="text-xs text-muted-foreground">Failed</span>
            <span class="font-bold">{{ stats.failed }}</span>
          </div>
        </div>
      </div>

      <!-- Active Pages List -->
      <div class="space-y-3 pt-4 border-t">
        <h4 class="text-sm font-medium">
          Active Crawls
        </h4>
        <div class="max-h-[300px] overflow-y-auto space-y-2 pr-2">
          <div
            v-for="page in pages"
            :key="page.id"
            class="flex items-center justify-between p-2 rounded border bg-background text-sm"
          >
            <span
              class="truncate max-w-[70%] text-muted-foreground"
              :title="page.url"
            >
              {{ page.url }}
            </span>
            <Badge
              :variant="
                page.status === 'completed'
                  ? 'default'
                  : page.status === 'failed'
                    ? 'destructive'
                    : 'secondary'
              "
              class="text-[10px] capitalize"
            >
              {{ page.status }}
            </Badge>
          </div>
          <div
            v-if="pages.length === 0"
            class="text-xs text-center text-muted-foreground py-4"
          >
            No pages found yet.
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
