import { describe, it, expect, vi, beforeEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useSettingsStore } from "./settings.store";

// Mock global fetch
const fetchMock = vi.fn();
global.fetch = fetchMock;

describe("Settings Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    fetchMock.mockReset();
  });

  it("fetchSettings - success", async () => {
    const store = useSettingsStore();

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: {
          rerank_provider: "cohere",
          search_alpha: 0.8,
        },
      }),
    });

    await store.fetchSettings();

    expect(store.rerankProvider).toBe("cohere");
    expect(store.searchAlpha).toBe(0.8);
    expect(store.isLoading).toBe(false);
    expect(store.error).toBeNull();
  });

  it("fetchSettings - error", async () => {
    const store = useSettingsStore();

    fetchMock.mockResolvedValueOnce({
      ok: false,
    });

    await store.fetchSettings();

    expect(store.error).toBe("Failed to fetch settings");
    expect(store.isLoading).toBe(false);
  });

  it("updateSettings - success", async () => {
    const store = useSettingsStore();
    store.rerankProvider = "jina";

    fetchMock.mockResolvedValueOnce({
      ok: true,
    });

    await store.updateSettings();

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/settings",
      expect.objectContaining({
        method: "PUT",
        body: expect.stringContaining('"rerank_provider":"jina"'),
      }),
    );
    expect(store.successMessage).toBe("Settings saved successfully");
  });

  it("updateSettings - error", async () => {
    const store = useSettingsStore();

    fetchMock.mockResolvedValueOnce({
      ok: false,
    });

    await store.updateSettings();

    expect(store.error).toBe("Failed to update settings");
    expect(store.isLoading).toBe(false);
    expect(store.successMessage).toBeNull();
  });

  it("fetchSettings - non-Error exception", async () => {
    const store = useSettingsStore();

    fetchMock.mockRejectedValueOnce("string error");

    await store.fetchSettings();

    expect(store.error).toBe("Unknown error");
    expect(store.isLoading).toBe(false);
  });

  it("updateSettings - non-Error exception", async () => {
    const store = useSettingsStore();

    fetchMock.mockRejectedValueOnce(42);

    await store.updateSettings();

    expect(store.error).toBe("Unknown error");
    expect(store.isLoading).toBe(false);
  });

  it("fetchSettings populates all fields", async () => {
    const store = useSettingsStore();

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: {
          rerank_provider: "jina",
          rerank_api_key: "rk-123",
          gemini_api_key: "gk-456",
          search_alpha: 0.7,
          search_top_k: 30,
        },
      }),
    });

    await store.fetchSettings();

    expect(store.rerankProvider).toBe("jina");
    expect(store.rerankApiKey).toBe("rk-123");
    expect(store.geminiApiKey).toBe("gk-456");
    expect(store.searchAlpha).toBe(0.7);
    expect(store.searchTopK).toBe(30);
  });

  it("updateSettings success message clears after timeout", async () => {
    vi.useFakeTimers();
    const store = useSettingsStore();

    fetchMock.mockResolvedValueOnce({ ok: true });

    await store.updateSettings();

    expect(store.successMessage).toBe("Settings saved successfully");

    vi.advanceTimersByTime(3000);

    expect(store.successMessage).toBeNull();
    vi.useRealTimers();
  });
});
