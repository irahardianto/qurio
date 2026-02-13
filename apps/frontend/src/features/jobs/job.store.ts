import { defineStore } from "pinia";
import { ref } from "vue";

export interface Job {
  id: string;
  source_id: string;
  handler: string;
  payload: unknown;
  error: string;
  retries: number;
  created_at: string;
}

export const useJobStore = defineStore("jobs", () => {
  const jobs = ref<Job[]>([]);
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  async function fetchFailedJobs() {
    isLoading.value = true;
    error.value = null;
    try {
      const res = await fetch("/api/jobs/failed");
      if (!res.ok) throw new Error(`Failed to fetch jobs: ${res.statusText}`);
      const json = await res.json();
      jobs.value = json.data;
    } catch (e: unknown) {
      if (e instanceof Error) {
        error.value = e.message;
      } else {
        error.value = "Unknown error";
      }
    } finally {
      isLoading.value = false;
    }
  }

  async function retryJob(id: string) {
    isLoading.value = true;
    error.value = null;
    try {
      const res = await fetch(`/api/jobs/${id}/retry`, { method: "POST" });
      if (!res.ok) throw new Error(`Failed to retry job: ${res.statusText}`);
      // Remove from list
      jobs.value = jobs.value.filter((j) => j.id !== id);
    } catch (e: unknown) {
      if (e instanceof Error) {
        error.value = e.message;
      } else {
        error.value = "Unknown error";
      }
    } finally {
      isLoading.value = false;
    }
  }

  return { jobs, isLoading, error, fetchFailedJobs, retryJob };
});
