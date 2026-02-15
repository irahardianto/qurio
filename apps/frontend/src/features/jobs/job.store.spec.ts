import { setActivePinia, createPinia } from "pinia";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { useJobStore } from "./job.store";

describe("Job Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    global.fetch = vi.fn();
  });

  it("initializes with correct default state", () => {
    const store = useJobStore();
    expect(store.jobs).toEqual([]);
    expect(store.isLoading).toBe(false);
    expect(store.error).toBe(null);
  });

  it("fetchFailedJobs populates state on success", async () => {
    const store = useJobStore();
    const mockJobs = [
      {
        id: "1",
        handler: "test",
        error: "fail",
        source_id: "s1",
        retries: 0,
        payload: {},
        created_at: "now",
      },
    ];

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: mockJobs }),
    });

    const promise = store.fetchFailedJobs();
    expect(store.isLoading).toBe(true);
    await promise;

    expect(store.jobs).toEqual(mockJobs);
    expect(store.isLoading).toBe(false);
    expect(store.error).toBe(null);
  });

  it("fetchFailedJobs handles error", async () => {
    const store = useJobStore();

    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: "Internal Server Error",
    });

    await store.fetchFailedJobs();

    expect(store.jobs).toEqual([]);
    expect(store.isLoading).toBe(false);
    expect(store.error).toContain("Failed to fetch jobs");
  });

  it("retryJob removes job from list on success", async () => {
    const store = useJobStore();
    store.jobs = [
      {
        id: "1",
        handler: "test",
        error: "fail",
        source_id: "s1",
        retries: 0,
        payload: {},
        created_at: "",
      },
    ];

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
    });

    const promise = store.retryJob("1");
    expect(store.isLoading).toBe(true);
    await promise;

    expect(global.fetch).toHaveBeenCalledWith(
      "/api/jobs/1/retry",
      expect.objectContaining({
        method: "POST",
      }),
    );
    expect(store.jobs).toHaveLength(0);
    expect(store.isLoading).toBe(false);
  });

  it("retryJob handles API error", async () => {
    const store = useJobStore();
    store.jobs = [
      {
        id: "1",
        handler: "test",
        error: "fail",
        source_id: "s1",
        retries: 0,
        payload: {},
        created_at: "",
      },
    ];

    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: "Server Error",
    });

    await store.retryJob("1");

    expect(store.error).toContain("Failed to retry job");
    expect(store.jobs).toHaveLength(1); // Job NOT removed on failure
    expect(store.isLoading).toBe(false);
  });

  it("retryJob handles non-Error exception", async () => {
    const store = useJobStore();
    store.jobs = [
      {
        id: "1",
        handler: "test",
        error: "fail",
        source_id: "s1",
        retries: 0,
        payload: {},
        created_at: "",
      },
    ];

    global.fetch = vi.fn().mockRejectedValue("string error");

    await store.retryJob("1");

    expect(store.error).toBe("Unknown error");
    expect(store.isLoading).toBe(false);
  });

  it("fetchFailedJobs handles network rejection", async () => {
    const store = useJobStore();

    global.fetch = vi.fn().mockRejectedValue(new Error("Network Error"));

    await store.fetchFailedJobs();

    expect(store.error).toBe("Network Error");
    expect(store.isLoading).toBe(false);
  });

  it("fetchFailedJobs handles non-Error exception", async () => {
    const store = useJobStore();

    global.fetch = vi.fn().mockRejectedValue("string throw");

    await store.fetchFailedJobs();

    expect(store.error).toBe("Unknown error");
    expect(store.isLoading).toBe(false);
  });
});
