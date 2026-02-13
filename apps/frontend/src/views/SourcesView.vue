<script setup lang="ts">
import SourceForm from "../features/sources/SourceForm.vue";
import SourceList from "../features/sources/SourceList.vue";
import { Database } from "lucide-vue-next";
</script>

<template>
  <div class="space-y-8 w-full p-6 lg:p-10">
    <!-- Header -->
    <div class="flex items-center justify-between pb-2">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">
          Knowledge Base
        </h1>
        <p class="text-muted-foreground mt-2 flex items-center gap-2 text-lg">
          <Database class="w-5 h-5" />
          Manage data ingestion sources and verify indexed content.
        </p>
      </div>
    </div>

    <!-- Hero Input Section -->
    <div class="w-full">
      <SourceForm />
    </div>

    <!-- Sources List Section -->
    <div class="space-y-6 pt-6">
      <div class="flex flex-col space-y-1.5 border-b border-border pb-4">
        <h3
          class="text-xl font-semibold leading-none tracking-tight text-foreground"
        >
          Active Sources
        </h3>
        <p class="text-sm text-muted-foreground">
          Currently tracked data sources, their indexing status, and chunk
          counts.
        </p>
      </div>

      <div
        class="rounded-xl border border-border bg-card/30 backdrop-blur-sm p-6 shadow-sm"
      >
        <SourceList />
      </div>
    </div>
  </div>
</template>
