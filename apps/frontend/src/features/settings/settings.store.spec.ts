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
});
