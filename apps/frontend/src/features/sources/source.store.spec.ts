import { describe, it, expect, vi, beforeEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useSourceStore } from "./source.store";

// Mock global fetch
const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

describe("Source Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fetchMock.mockReset();
  });

  it("fetchSources updates state on success", async () => {
    const store = useSourceStore();
    const mockData = [{ id: "1", name: "Test Source" }];

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: mockData }),
    });

    await store.fetchSources();

    expect(store.sources).toEqual(mockData);
    expect(store.isLoading).toBe(false);
    expect(store.error).toBeNull();
  });

  it("fetchSources handles error", async () => {
    const store = useSourceStore();

    fetchMock.mockResolvedValueOnce({
      ok: false,
      statusText: "Internal Server Error",
    });

    await store.fetchSources();

    expect(store.error).toBe("Failed to fetch sources: Internal Server Error");
    expect(store.isLoading).toBe(false);
  });

  it("addSource updates state on success", async () => {
    const store = useSourceStore();
    const newSource = { name: "New Source", url: "http://test.com" };
    const mockResponse = { id: "2", ...newSource };

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: mockResponse }),
    });

    await store.addSource(newSource);

    expect(store.sources).toContainEqual(mockResponse);
    expect(store.isLoading).toBe(false);
  });

  it("addSource handles error", async () => {
    const store = useSourceStore();
    const newSource = { name: "New Source" };

    fetchMock.mockResolvedValueOnce({
      ok: false,
      statusText: "Bad Request",
    });

    await store.addSource(newSource);

    expect(store.error).toBe("Failed to add source: Bad Request");
  });

  it("deleteSource removes source from state", async () => {
    const store = useSourceStore();
    store.sources = [{ id: "1", name: "Delete Me" }];

    fetchMock.mockResolvedValueOnce({
      ok: true,
    });

    await store.deleteSource("1");

    expect(store.sources).toHaveLength(0);
  });

  it("resyncSource calls API", async () => {
    const store = useSourceStore();

    fetchMock.mockResolvedValueOnce({
      ok: true,
    });

    await store.resyncSource("1");

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/sources/1/resync",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("uploadSource uploads file and updates state", async () => {
    const store = useSourceStore();
    const file = new File(["content"], "test.pdf", { type: "application/pdf" });
    const mockResponse = { id: "3", name: "test.pdf" };

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: mockResponse }),
    });

    await store.uploadSource(file, "test.pdf");

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/sources/upload",
      expect.objectContaining({
        method: "POST",
        body: expect.any(FormData),
      }),
    );
    expect(store.sources).toContainEqual(mockResponse);
  });

  it("uploadSource handles error", async () => {
    const store = useSourceStore();
    const file = new File(["content"], "test.pdf", { type: "application/pdf" });

    fetchMock.mockResolvedValueOnce({
      ok: false,
      statusText: "Payload Too Large",
      json: async () => ({ error: { message: "File too large" } }),
    });

    await expect(store.uploadSource(file, "test.pdf")).rejects.toThrow(
      "File too large",
    );
    expect(store.error).toBe("File too large");
  });

  it("getSource fetches source details", async () => {
    const store = useSourceStore();
    const mockData = { id: "1", name: "Detail Source" };

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: mockData }),
    });

    const result = await store.getSource("1");
    expect(result).toEqual(mockData);
  });

  it("getSource handles error gracefully", async () => {
    const store = useSourceStore();

    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: async () => ({}),
    });

    const result = await store.getSource("missing");
    expect(result).toBeNull();
    expect(store.error).toContain("Failed to fetch source");
  });

  it("getSourcePages fetches pages", async () => {
    const store = useSourceStore();
    const mockPages = [{ id: "p1", url: "http://u.rl" }];

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: mockPages }),
    });

    const result = await store.getSourcePages("1");
    expect(result).toEqual(mockPages);
  });

  it("polling fetches sources when active", async () => {
    const store = useSourceStore();
    vi.useFakeTimers();

    // Initial state with active source
    store.sources = [{ id: "1", name: "Test", status: "processing" }];

    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ data: [{ id: "1", status: "completed" }] }),
    });

    store.startPolling();
    vi.advanceTimersByTime(2000);

    expect(fetchMock).toHaveBeenCalled();

    store.stopPolling();
    vi.useRealTimers();
  });

  it("getSource sends pagination params", async () => {
    const store = useSourceStore();

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: {} }),
    });

    await store.getSource("1", { limit: 20, offset: 10, exclude_chunks: true });

    const url = fetchMock.mock.calls[0][0];
    expect(url).toContain("limit=20");
    expect(url).toContain("offset=10");
    expect(url).toContain("exclude_chunks=true");
  });

  it("fetchChunks fetches only chunks", async () => {
    const store = useSourceStore();
    const mockChunks = [{ content: "c1" }];

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { chunks: mockChunks } }),
    });

    const chunks = await store.fetchChunks("1", 50, 20);

    const url = fetchMock.mock.calls[0][0];
    expect(url).toContain("limit=20");
    expect(url).toContain("offset=50");
    expect(chunks).toEqual(mockChunks);
  });

  it("pollSourceStatus fetches with exclude_chunks", async () => {
    const store = useSourceStore();

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { id: "1", status: "completed" } }),
    });

    await store.pollSourceStatus("1");

    const url = fetchMock.mock.calls[0][0];
    expect(url).toContain("exclude_chunks=true");
  });

  it("deleteSource handles API error", async () => {
    const store = useSourceStore();
    store.sources = [{ id: "1", name: "Keep Me" }];

    fetchMock.mockResolvedValueOnce({
      ok: false,
      statusText: "Server Error",
    });

    await store.deleteSource("1");

    expect(store.error).toBe("Failed to delete source: Server Error");
    expect(store.sources).toHaveLength(1); // Source NOT removed on error
    expect(store.isLoading).toBe(false);
  });

  it("fetchChunks handles error and returns empty array", async () => {
    const store = useSourceStore();

    fetchMock.mockResolvedValueOnce({
      ok: false,
      statusText: "Not Found",
    });

    const chunks = await store.fetchChunks("1", 0, 100);

    expect(chunks).toEqual([]);
  });

  it("getSourcePages handles error and returns empty array", async () => {
    const store = useSourceStore();

    fetchMock.mockResolvedValueOnce({
      ok: false,
      statusText: "Server Error",
    });

    const pages = await store.getSourcePages("1");

    expect(pages).toEqual([]);
  });

  it("fetchSources with background=true skips loading state", async () => {
    const store = useSourceStore();
    const mockData = [{ id: "1", name: "Bg Source" }];

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: mockData }),
    });

    const promise = store.fetchSources(true);
    // isLoading should NOT be set to true for background fetches
    expect(store.isLoading).toBe(false);
    await promise;

    expect(store.sources).toEqual(mockData);
    expect(store.isLoading).toBe(false);
  });

  it("startPolling is idempotent", () => {
    const store = useSourceStore();
    vi.useFakeTimers();

    store.sources = [{ id: "1", name: "Test", status: "processing" }];

    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ data: [] }),
    });

    store.startPolling();
    store.startPolling(); // Should not create duplicate

    vi.advanceTimersByTime(2000);

    // Should only be called once (one interval, not two)
    expect(fetchMock).toHaveBeenCalledTimes(1);

    store.stopPolling();
    vi.useRealTimers();
  });

  it("fetchSources handles network exception", async () => {
    const store = useSourceStore();

    fetchMock.mockRejectedValueOnce(new Error("Network Error"));

    await store.fetchSources();

    expect(store.error).toBe("Network Error");
    expect(store.isLoading).toBe(false);
  });

  it("resyncSource handles API error", async () => {
    const store = useSourceStore();

    fetchMock.mockResolvedValueOnce({
      ok: false,
      statusText: "Bad Gateway",
    });

    await store.resyncSource("1");

    expect(store.error).toBe("Failed to resync source: Bad Gateway");
    expect(store.isLoading).toBe(false);
  });

  it("uploadSource handles error when json parse fails", async () => {
    const store = useSourceStore();
    const file = new File(["content"], "test.pdf", { type: "application/pdf" });

    fetchMock.mockResolvedValueOnce({
      ok: false,
      statusText: "Payload Too Large",
      json: async () => {
        throw new Error("parse error");
      },
    });

    await expect(store.uploadSource(file, "test.pdf")).rejects.toThrow(
      "Failed to upload source: Payload Too Large",
    );
    expect(store.error).toBe("Failed to upload source: Payload Too Large");
  });
});
