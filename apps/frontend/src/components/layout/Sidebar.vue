<script setup lang="ts">
import { RouterLink, useRoute } from "vue-router";
import {
  Database,
  Settings,
  LayoutDashboard,
  AlertTriangle,
} from "lucide-vue-next";

const route = useRoute();

const navigation = [
  { name: "Dashboard", href: "/", icon: LayoutDashboard },
  { name: "Sources", href: "/sources", icon: Database },
  { name: "Failed Jobs", href: "/jobs", icon: AlertTriangle },
  { name: "Settings", href: "/settings", icon: Settings },
];

const isActive = (path: string) => route.path === path;
</script>

<template>
  <aside
    class="w-64 flex-shrink-0 border-r border-border bg-card/30 backdrop-blur-sm flex flex-col z-20"
  >
    <div class="h-16 flex items-center px-6 border-b border-border">
      <!-- Logo Area -->
      <span
        class="font-mono font-bold text-xl tracking-tight text-foreground flex items-center gap-2"
      >
        <img
          src="/qurio.png"
          alt="Qurio icon"
          class="icon"
        >
        <span><span class="text-primary">&lt;</span>Qurio<span class="text-primary">/&gt;</span></span>
      </span>
    </div>

    <nav class="flex-1 px-4 py-6 space-y-1">
      <router-link
        v-for="item in navigation"
        :key="item.name"
        :to="item.href"
        class="group flex items-center px-3 py-2 text-sm font-medium rounded-md transition-all duration-200"
        :class="[
          isActive(item.href)
            ? 'bg-primary/10 text-primary shadow-[0_0_10px_rgba(59,130,246,0.15)] border-l-2 border-primary'
            : 'text-muted-foreground hover:bg-secondary/50 hover:text-foreground border-l-2 border-transparent',
        ]"
      >
        <component
          :is="item.icon"
          class="mr-3 h-5 w-5 flex-shrink-0"
        />
        {{ item.name }}
      </router-link>
    </nav>

    <div class="p-4 border-t border-border">
      <div class="flex items-center gap-2">
        <div class="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
        <span class="text-xs font-mono text-muted-foreground">System Online</span>
      </div>
      <div class="mt-1 text-xs font-mono text-muted-foreground/50">
        v0.8.2
      </div>
    </div>
  </aside>
</template>
