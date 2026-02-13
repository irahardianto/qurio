<script setup lang="ts">
import Settings from "../features/settings/Settings.vue";
import { Settings as SettingsIcon } from "lucide-vue-next";
</script>

<template>
  <div class="space-y-8 w-full p-6 lg:p-10 animate-in fade-in duration-500">
    <div class="flex items-center justify-between pb-2">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">
          Settings
        </h1>
        <p class="text-muted-foreground mt-2 flex items-center gap-2 text-lg">
          <SettingsIcon class="w-5 h-5" />
          Configure AI providers and system preferences.
        </p>
      </div>
    </div>

    <div
      class="rounded-xl border border-border bg-card/50 backdrop-blur-sm p-6 shadow-sm"
    >
      <Settings />
    </div>
  </div>
</template>
