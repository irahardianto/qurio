<script setup lang="ts">
import { onMounted } from "vue";
import { useStatsStore } from "../features/stats/stats.store";
import { useSourceStore } from "../features/sources/source.store";
import SourceList from "../features/sources/SourceList.vue";
import { Database, FileText, AlertTriangle, Activity } from "lucide-vue-next";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";

const statsStore = useStatsStore();
const sourceStore = useSourceStore();

onMounted(() => {
  statsStore.fetchStats();
  sourceStore.fetchSources();
});
</script>

<template>
  <div class="space-y-8 w-full p-6 lg:p-10 animate-in fade-in duration-500">
    <div class="flex items-center justify-between pb-2">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">
          Dashboard
        </h1>
        <p class="text-muted-foreground mt-2 flex items-center gap-2 text-lg">
          <Activity class="w-5 h-5" />
          System Overview
        </p>
      </div>
    </div>

    <!-- Stats Grid -->
    <div class="grid gap-4 md:grid-cols-3">
      <Card
        class="bg-card/50 backdrop-blur-sm border-border shadow-sm hover:shadow-md transition-shadow"
      >
        <CardHeader
          class="flex flex-row items-center justify-between space-y-0 pb-2"
        >
          <CardTitle class="text-sm font-medium text-muted-foreground">
            Total Sources
          </CardTitle>
          <Database class="h-4 w-4 text-primary" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-foreground">
            {{ statsStore.stats.sources }}
          </div>
          <p class="text-xs text-muted-foreground mt-1">
            Active ingestion targets
          </p>
        </CardContent>
      </Card>

      <Card
        class="bg-card/50 backdrop-blur-sm border-border shadow-sm hover:shadow-md transition-shadow"
      >
        <CardHeader
          class="flex flex-row items-center justify-between space-y-0 pb-2"
        >
          <CardTitle class="text-sm font-medium text-muted-foreground">
            Indexed Documents
          </CardTitle>
          <FileText class="h-4 w-4 text-emerald-500" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-foreground">
            {{ statsStore.stats.documents }}
          </div>
          <p class="text-xs text-muted-foreground mt-1">
            Chunks in vector store
          </p>
        </CardContent>
      </Card>

      <Card
        class="bg-card/50 backdrop-blur-sm border-border shadow-sm hover:shadow-md transition-shadow"
      >
        <CardHeader
          class="flex flex-row items-center justify-between space-y-0 pb-2"
        >
          <CardTitle class="text-sm font-medium text-muted-foreground">
            Failed Jobs
          </CardTitle>
          <AlertTriangle class="h-4 w-4 text-destructive" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-foreground">
            {{ statsStore.stats.failed_jobs }}
          </div>
          <p class="text-xs text-muted-foreground mt-1">
            Requires attention
          </p>
        </CardContent>
      </Card>
    </div>

    <!-- Recent Sources -->
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-xl font-semibold tracking-tight">
          Recent Sources
        </h2>
      </div>
      <div
        class="rounded-xl border border-border bg-card/30 backdrop-blur-sm p-6 shadow-sm"
      >
        <SourceList />
      </div>
    </div>
  </div>
</template>
