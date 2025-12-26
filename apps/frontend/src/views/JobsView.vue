<script setup lang="ts">
import { onMounted } from 'vue'
import { useJobStore } from '../features/jobs/job.store'
import { RefreshCw, CheckCircle, AlertOctagon } from 'lucide-vue-next'

const jobStore = useJobStore()

onMounted(() => {
  jobStore.fetchFailedJobs()
})

const formatDate = (date: string) => {
  return new Date(date).toLocaleString()
}
</script>

<template>
  <div class="jobs-view">
    <div class="header">
      <div class="header-content">
        <h1 class="title">Failed Jobs</h1>
        <p class="subtitle">Manage ingestion failures and retries</p>
      </div>
      <button class="refresh-btn" @click="jobStore.fetchFailedJobs" :disabled="jobStore.isLoading">
        <RefreshCw :size="16" :class="{ 'spin': jobStore.isLoading }" />
        Refresh
      </button>
    </div>

    <div v-if="jobStore.jobs.length === 0 && !jobStore.isLoading" class="empty-state">
      <CheckCircle :size="48" class="empty-icon" />
      <h3>No Failed Jobs</h3>
      <p>All ingestion tasks are running smoothly.</p>
    </div>

    <div v-else class="jobs-list">
      <div v-for="job in jobStore.jobs" :key="job.id" class="job-card">
        <div class="job-header">
          <div class="job-meta">
            <span class="source-id">Source: {{ job.source_id }}</span>
            <span class="date">{{ formatDate(job.created_at) }}</span>
          </div>
          <span class="badge retry-count">Retries: {{ job.retries }}</span>
        </div>
        
        <div class="job-body">
          <div class="error-container">
            <AlertOctagon :size="16" class="error-icon" />
            <code class="error-msg">{{ job.error }}</code>
          </div>
          
          <!-- Debug Payload -->
          <details class="payload-details">
            <summary>View Payload</summary>
            <pre>{{ JSON.stringify(job.payload, null, 2) }}</pre>
          </details>
        </div>

        <div class="job-footer">
          <button class="retry-btn" @click="jobStore.retryJob(job.id)" :disabled="jobStore.isLoading">
            <RefreshCw :size="14" />
            Retry Job
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.jobs-view {
  max-width: 900px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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

.refresh-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-main);
  cursor: pointer;
  font-weight: 500;
  transition: all 0.2s;
}

.refresh-btn:hover:not(:disabled) {
  background-color: var(--color-border);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.empty-state {
  text-align: center;
  padding: 4rem 2rem;
  background-color: var(--color-surface);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
}

.empty-icon {
  color: var(--color-success);
  margin-bottom: 1rem;
}

.jobs-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.job-card {
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
}

.job-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.job-meta {
  display: flex;
  gap: 1rem;
  font-size: 0.875rem;
  color: var(--color-text-muted);
}

.source-id {
  font-family: var(--font-mono);
  color: var(--color-primary);
}

.badge {
  font-size: 0.75rem;
  padding: 0.125rem 0.5rem;
  border-radius: 999px;
  background-color: var(--color-border);
  color: var(--color-text-muted);
}

.job-body {
  margin-bottom: 1rem;
}

.error-container {
  display: flex;
  gap: 0.75rem;
  padding: 0.75rem;
  background-color: rgba(239, 68, 68, 0.1);
  border-radius: var(--radius-md);
  color: #ef4444;
  margin-bottom: 0.75rem;
  align-items: flex-start;
}

.error-msg {
  font-family: var(--font-mono);
  font-size: 0.875rem;
  word-break: break-all;
}

.payload-details {
  font-size: 0.875rem;
  color: var(--color-text-muted);
}

.payload-details summary {
  cursor: pointer;
  margin-bottom: 0.5rem;
}

.payload-details pre {
  background-color: var(--color-background);
  padding: 0.75rem;
  border-radius: var(--radius-md);
  overflow-x: auto;
  font-family: var(--font-mono);
  font-size: 0.75rem;
}

.job-footer {
  display: flex;
  justify-content: flex-end;
}

.retry-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background-color: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.2s;
}

.retry-btn:hover:not(:disabled) {
  opacity: 0.9;
}
</style>
