<script setup lang="ts">
import Sidebar from './Sidebar.vue'
</script>

<template>
  <div class="flex h-screen w-full bg-background overflow-hidden font-sans text-foreground">
    <Sidebar />
    <main class="flex-1 flex flex-col min-w-0 overflow-hidden relative bg-background">
      <!-- Optional: Grid overlay for "technical" texture -->
      <div class="absolute inset-0 bg-[url('/grid-pattern.svg')] opacity-[0.02] pointer-events-none"></div>
      
      <div class="flex-1 overflow-y-auto p-6 md:p-8 scroll-smooth">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>