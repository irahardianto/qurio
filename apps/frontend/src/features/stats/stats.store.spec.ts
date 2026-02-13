import { setActivePinia, createPinia } from "pinia";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { useStatsStore } from "./stats.store";

describe("Stats Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    global.fetch = vi.fn();
  });

  it("initializes with correct default state", () => {
    const store = useStatsStore();
    expect(store.stats).toEqual({ sources: 0, documents: 0, failed_jobs: 0 });
    expect(store.isLoading).toBe(false);
    expect(store.error).toBe(null);
  });

  it("fetchStats populates state on success", async () => {
    const store = useStatsStore();
    const mockStats = { sources: 5, documents: 100, failed_jobs: 2 };

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: mockStats }),
    });

    const promise = store.fetchStats();
    expect(store.isLoading).toBe(true);
    await promise;

    expect(store.stats).toEqual(mockStats);
    expect(store.isLoading).toBe(false);
    expect(store.error).toBe(null);
  });

  it("fetchStats handles error", async () => {
    const store = useStatsStore();

    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: "Internal Server Error",
    });

    await store.fetchStats();

    expect(store.stats).toEqual({ sources: 0, documents: 0, failed_jobs: 0 });
    expect(store.isLoading).toBe(false);
    expect(store.error).toContain("Failed to fetch stats");
  });
});
