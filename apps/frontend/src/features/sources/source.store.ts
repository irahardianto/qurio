import { defineStore } from "pinia";
import { ref } from "vue";

export interface Chunk {
  content: string;
  vector?: number[];
  source_url: string;
  source_id: string;
  source_name?: string;
  chunk_index: number;
  type: string;
  language: string;
  title: string;
}

export interface Source {
  id: string;
  name: string;
  type?: string;
  url?: string;
  status?: string;
  updated_at?: string;
  max_depth?: number;
  exclusions?: string[];
  chunks?: Chunk[];
  total_chunks?: number;
}

export interface SourcePage {
  id: string;
  source_id: string;
  url: string;
  status: string;
  depth: number;
  error?: string;
  created_at: string;
  updated_at: string;
}

export const useSourceStore = defineStore("sources", () => {
  const sources = ref<Source[]>([]);
  const isLoading = ref(false);
  const error = ref<string | null>(null);
  let pollingInterval: ReturnType<typeof setInterval> | null = null;

  async function fetchSources(background = false) {
    if (!background) isLoading.value = true;
    error.value = null;
    try {
      const res = await fetch("/api/sources");
      if (!res.ok) {
        throw new Error(`Failed to fetch sources: ${res.statusText}`);
      }
      const json = await res.json();
      sources.value = json.data || [];
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "Unknown error";
      error.value = message;
      console.error("Failed to fetch sources", e);
    } finally {
      if (!background) isLoading.value = false;
    }
  }

  function startPolling() {
    if (pollingInterval) return;
    pollingInterval = setInterval(() => {
      const hasActiveSources = sources.value.some(
        (s) =>
          s.status === "processing" ||
          s.status === "pending" ||
          s.status === "in_progress",
      );
      if (hasActiveSources) {
        fetchSources(true);
      }
    }, 2000);
  }

  function stopPolling() {
    if (pollingInterval) {
      clearInterval(pollingInterval);
      pollingInterval = null;
    }
  }

  async function addSource(source: Omit<Source, "id">) {
    isLoading.value = true;
    error.value = null;
    try {
      const res = await fetch("/api/sources", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(source),
      });
      if (!res.ok) {
        throw new Error(`Failed to add source: ${res.statusText}`);
      }
      const json = await res.json();
      sources.value.push(json.data);
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "Unknown error";
      error.value = message;
      console.error("Failed to add source", e);
    } finally {
      isLoading.value = false;
    }
  }

  async function deleteSource(id: string) {
    isLoading.value = true;
    error.value = null;
    try {
      const res = await fetch(`/api/sources/${id}`, { method: "DELETE" });
      if (!res.ok)
        throw new Error(`Failed to delete source: ${res.statusText}`);
      sources.value = sources.value.filter((s) => s.id !== id);
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "Unknown error";
      error.value = message;
    } finally {
      isLoading.value = false;
    }
  }

  async function resyncSource(id: string) {
    isLoading.value = true;
    error.value = null;
    try {
      const res = await fetch(`/api/sources/${id}/resync`, { method: "POST" });
      if (!res.ok)
        throw new Error(`Failed to resync source: ${res.statusText}`);
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "Unknown error";
      error.value = message;
    } finally {
      isLoading.value = false;
    }
  }

  async function uploadSource(file: File, name: string) {
    isLoading.value = true;
    error.value = null;
    try {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("name", name);

      const res = await fetch("/api/sources/upload", {
        method: "POST",
        body: formData,
      });
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        throw new Error(
          errorData.error?.message ||
            `Failed to upload source: ${res.statusText}`,
        );
      }
      const json = await res.json();
      sources.value.push(json.data);
      return json.data;
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "Unknown error";
      error.value = message;
      console.error("Failed to upload source", e);
      throw e;
    } finally {
      isLoading.value = false;
    }
  }

  async function getSource(
    id: string,
    params: { limit?: number; offset?: number; exclude_chunks?: boolean } = {},
    background = false,
  ) {
    if (!background) isLoading.value = true;
    error.value = null;
    try {
      const query = new URLSearchParams();
      if (params.limit !== undefined)
        query.append("limit", params.limit.toString());
      if (params.offset !== undefined)
        query.append("offset", params.offset.toString());
      if (params.exclude_chunks) query.append("exclude_chunks", "true");

      const queryString = query.toString() ? `?${query.toString()}` : "";
      const res = await fetch(`/api/sources/${id}${queryString}`);

      if (!res.ok)
        throw new Error(`Failed to fetch source details: ${res.statusText}`);
      const json = await res.json();
      return json.data as Source;
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "Unknown error";
      error.value = message;
      return null;
    } finally {
      if (!background) isLoading.value = false;
    }
  }

  async function fetchChunks(id: string, offset: number, limit = 100) {
    try {
      const query = new URLSearchParams();
      query.append("limit", limit.toString());
      query.append("offset", offset.toString());
      const res = await fetch(`/api/sources/${id}?${query.toString()}`);
      if (!res.ok) throw new Error(`Failed to fetch chunks`);
      const json = await res.json();
      return (json.data as Source).chunks || [];
    } catch (e) {
      console.error(e);
      return [];
    }
  }

  async function pollSourceStatus(id: string) {
    return getSource(id, { exclude_chunks: true }, true);
  }

  async function getSourcePages(id: string) {
    try {
      const res = await fetch(`/api/sources/${id}/pages`);
      if (!res.ok)
        throw new Error(`Failed to fetch source pages: ${res.statusText}`);
      const json = await res.json();
      return json.data as SourcePage[];
    } catch (e: unknown) {
      console.error("Failed to fetch source pages", e);
      return [];
    }
  }

  return {
    sources,
    isLoading,
    error,
    fetchSources,
    addSource,
    deleteSource,
    resyncSource,
    uploadSource,
    getSource,
    getSourcePages,
    startPolling,
    stopPolling,
    fetchChunks,
    pollSourceStatus,
  };
});
