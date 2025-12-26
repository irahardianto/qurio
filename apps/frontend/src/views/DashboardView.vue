<script setup lang="ts">
import { onMounted } from 'vue'
import { useStatsStore } from '../features/stats/stats.store'
import { useSourceStore } from '../features/sources/source.store'
import SourceList from '../features/sources/SourceList.vue'
import { Database, FileText, AlertTriangle } from 'lucide-vue-next'

const statsStore = useStatsStore()
const sourceStore = useSourceStore()

onMounted(() => {
  statsStore.fetchStats()
  sourceStore.fetchSources()
})
</script>

<template>
  <div class="dashboard">
    <div class="header">
      <h1 class="title">Dashboard</h1>
      <p class="subtitle">System Overview</p>
    </div>

    <!-- Stats Grid -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon-wrapper blue">
          <Database :size="24" />
        </div>
        <div class="stat-content">
          <span class="stat-label">Sources</span>
          <span class="stat-value">{{ statsStore.stats.sources }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-wrapper green">
          <FileText :size="24" />
        </div>
        <div class="stat-content">
          <span class="stat-label">Documents</span>
          <span class="stat-value">{{ statsStore.stats.documents }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-wrapper red">
          <AlertTriangle :size="24" />
        </div>
        <div class="stat-content">
          <span class="stat-label">Failed Jobs</span>
          <span class="stat-value">{{ statsStore.stats.failed_jobs }}</span>
        </div>
      </div>
    </div>

    <!-- Recent Sources -->
    <div class="recent-sources">
      <h2 class="section-title">Recent Sources</h2>
      <SourceList />
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 1200px;
  margin: 0 auto;
}

.header {
  margin-bottom: 2rem;
}

.title {
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--color-text-main);
  margin-bottom: 0.5rem;
}

.subtitle {
  color: var(--color-text-muted);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1.5rem;
  margin-bottom: 3rem;
}

.stat-card {
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.stat-icon-wrapper {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-icon-wrapper.blue {
  background-color: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.stat-icon-wrapper.green {
  background-color: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.stat-icon-wrapper.red {
  background-color: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.stat-content {
  display: flex;
  flex-direction: column;
}

.stat-label {
  font-size: 0.875rem;
  color: var(--color-text-muted);
  margin-bottom: 0.25rem;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--color-text-main);
}

.section-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text-main);
  margin-bottom: 1rem;
}
</style>
