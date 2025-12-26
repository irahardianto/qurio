<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { Database, Settings, Box, LayoutDashboard, AlertTriangle } from 'lucide-vue-next'

const route = useRoute()

const isActive = (path: string) => route.path === path
</script>

<template>
  <aside class="sidebar">
    <div class="logo-container">
      <div class="logo-icon">
        <Box class="icon" />
      </div>
      <span class="logo-text">QURIO</span>
    </div>

    <nav class="nav">
      <RouterLink
        to="/"
        class="nav-item"
        :class="{ active: isActive('/') }"
      >
        <LayoutDashboard
          class="nav-icon"
          :size="20"
        />
        <span>Dashboard</span>
      </RouterLink>

      <RouterLink
        to="/sources"
        class="nav-item"
        :class="{ active: isActive('/sources') }"
      >
        <Database
          class="nav-icon"
          :size="20"
        />
        <span>Sources</span>
      </RouterLink>
      
      <RouterLink
        to="/jobs"
        class="nav-item"
        :class="{ active: isActive('/jobs') }"
      >
        <AlertTriangle
          class="nav-icon"
          :size="20"
        />
        <span>Failed Jobs</span>
      </RouterLink>

      <RouterLink
        to="/settings"
        class="nav-item"
        :class="{ active: isActive('/settings') }"
      >
        <Settings
          class="nav-icon"
          :size="20"
        />
        <span>Settings</span>
      </RouterLink>
    </nav>

    <div class="footer">
      <div class="status">
        <div class="status-dot" />
        <span>System Online</span>
      </div>
      <span class="version">v0.2.0-MVP</span>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 260px;
  background-color: var(--color-void);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  padding: 1.5rem;
  height: 100vh;
  position: fixed;
  left: 0;
  top: 0;
}

.logo-container {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 3rem;
  padding-left: 0.5rem;
}

.logo-icon {
  color: var(--color-primary);
}

.logo-text {
  font-family: var(--font-sans);
  font-weight: 800;
  font-size: 1.5rem;
  letter-spacing: -0.05em;
  color: var(--color-text-main);
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-md);
  color: var(--color-text-muted);
  text-decoration: none;
  font-weight: 500;
  transition: all 0.2s ease;
}

.nav-item:hover {
  background-color: rgba(255, 255, 255, 0.03);
  color: var(--color-text-main);
}

.nav-item.active {
  background-color: rgba(59, 130, 246, 0.1); /* Primary with opacity */
  color: var(--color-primary);
}

.footer {
  border-top: 1px solid var(--color-border);
  padding-top: 1.5rem;
}

.status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--color-success);
  margin-bottom: 0.25rem;
}

.status-dot {
  width: 6px;
  height: 6px;
  background-color: var(--color-success);
  border-radius: 50%;
  box-shadow: 0 0 8px var(--color-success);
}

.version {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--color-text-muted);
}
</style>
