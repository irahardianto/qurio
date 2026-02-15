import { mount, flushPromises } from "@vue/test-utils";
import { createTestingPinia } from "@pinia/testing";
import {
  describe,
  it,
  expect,
  vi,
  beforeEach,
  afterEach,
  type Mock,
} from "vitest";
import SourceDetailView from "./SourceDetailView.vue";
import { useSourceStore } from "../features/sources/source.store";

// Mock vue-router
const routeParams = { id: "src-1" };
const backMock = vi.fn();
vi.mock("vue-router", () => ({
  useRoute: () => ({ params: routeParams }),
  useRouter: () => ({ back: backMock }),
}));

// Global stubs for UI components and icons
const globalStubs = {
  Button: { template: "<button @click=\"$emit('click')\"><slot /></button>" },
  Card: { template: "<div><slot /></div>" },
  CardHeader: { template: "<div><slot /></div>" },
  CardTitle: { template: "<div><slot /></div>" },
  CardContent: { template: "<div><slot /></div>" },
  Badge: { template: "<span><slot /></span>" },
  StatusBadge: { template: "<div>StatusBadge</div>", props: ["status"] },
  SourceProgress: {
    template: "<div>SourceProgress</div>",
    props: ["pages"],
  },
  ArrowLeft: { template: "<svg></svg>" },
  Database: { template: "<svg></svg>" },
  FileText: { template: "<svg></svg>" },
  Layers: { template: "<svg></svg>" },
  Hash: { template: "<svg></svg>" },
  Braces: { template: "<svg></svg>" },
  ExternalLink: { template: "<svg></svg>" },
  Copy: { template: "<svg></svg>" },
};

describe("SourceDetailView", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    backMock.mockReset();
    routeParams.id = "src-1";
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function mountView(
    getSourceReturn: unknown = null,
    getSourcePagesReturn: unknown[] = [],
  ) {
    const pinia = createTestingPinia({ createSpy: vi.fn, stubActions: false });
    const store = useSourceStore(pinia);

    // Mock store methods
    vi.spyOn(store, "getSource").mockResolvedValue(getSourceReturn as any); // eslint-disable-line @typescript-eslint/no-explicit-any

    vi.spyOn(store, "getSourcePages").mockResolvedValue(
      getSourcePagesReturn as any, // eslint-disable-line @typescript-eslint/no-explicit-any
    );
    vi.spyOn(store, "fetchChunks").mockResolvedValue([]);
    vi.spyOn(store, "pollSourceStatus").mockResolvedValue(null);

    const wrapper = mount(SourceDetailView, {
      global: { plugins: [pinia], stubs: globalStubs },
    });
    return { wrapper, store };
  }

  it("shows loading spinner initially", () => {
    const { wrapper } = mountView();
    expect(wrapper.text()).toContain("Retrieving Metadata...");
  });

  it("fetches source and pages on mount", async () => {
    const source = {
      id: "src-1",
      name: "Test",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 0,
      chunks: [],
    };
    const { store } = mountView(source, []);

    await flushPromises();

    expect(store.getSource).toHaveBeenCalledWith("src-1");
    expect(store.getSourcePages).toHaveBeenCalledWith("src-1");
  });

  it("displays source metadata after loading", async () => {
    const source = {
      id: "src-1",
      name: "My Source",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 42,
      max_depth: 2,
      chunks: [],
    };
    const { wrapper } = mountView(source);
    await flushPromises();

    expect(wrapper.text()).toContain("Source Details");
    expect(wrapper.text()).toContain("src-1");
    expect(wrapper.text()).toContain("http://example.com");
    expect(wrapper.text()).toContain("42");
  });

  it("selects first chunk by default when chunks available", async () => {
    const chunk = {
      content: "Hello world",
      chunk_index: 0,
      source_url: "http://example.com",
      source_id: "src-1",
      type: "text",
      language: "en",
      title: "First Chunk",
    };
    const source = {
      id: "src-1",
      name: "Test",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 1,
      chunks: [chunk],
    };
    const { wrapper } = mountView(source);
    await flushPromises();

    expect(wrapper.text()).toContain("First Chunk");
    expect(wrapper.text()).toContain("Hello world");
  });

  it("shows empty chunk message when no chunks", async () => {
    const source = {
      id: "src-1",
      name: "Test",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 0,
      chunks: [],
    };
    const { wrapper } = mountView(source);
    await flushPromises();

    expect(wrapper.text()).toContain("No chunks indexed");
  });

  it("starts polling for in_progress sources", async () => {
    const source = {
      id: "src-1",
      name: "Test",
      status: "in_progress",
      url: "http://example.com",
      type: "web",
      total_chunks: 0,
      chunks: [],
    };
    const { store } = mountView(source, []);
    await flushPromises();

    // Advance timer to trigger poll
    vi.advanceTimersByTime(2000);
    await flushPromises();

    expect(store.pollSourceStatus).toHaveBeenCalledWith("src-1");
  });

  it("stops polling when source completes", async () => {
    const source = {
      id: "src-1",
      name: "Test",
      status: "processing",
      url: "http://example.com",
      type: "web",
      total_chunks: 0,
      chunks: [],
    };
    const { store } = mountView(source, []);
    await flushPromises();

    // After first poll, return completed status
    vi.spyOn(store, "pollSourceStatus").mockResolvedValue({
      id: "src-1",
      name: "Test",
      status: "completed",

      total_chunks: 5,
    } as any); // eslint-disable-line @typescript-eslint/no-explicit-any

    vi.advanceTimersByTime(2000);
    await flushPromises();

    // Reset mock call count
    (store.pollSourceStatus as Mock).mockClear();

    // Advance again — should NOT poll
    vi.advanceTimersByTime(2000);
    await flushPromises();

    expect(store.pollSourceStatus).not.toHaveBeenCalled();
  });

  it("clears polling interval on unmount", async () => {
    const source = {
      id: "src-1",
      name: "Test",
      status: "pending",
      url: "http://example.com",
      type: "web",
      total_chunks: 0,
      chunks: [],
    };
    const { wrapper, store } = mountView(source);
    await flushPromises();

    wrapper.unmount();

    // Advance timer — poll should NOT fire
    (store.pollSourceStatus as Mock).mockClear();
    vi.advanceTimersByTime(4000);
    await flushPromises();

    expect(store.pollSourceStatus).not.toHaveBeenCalled();
  });

  it("shows load more button when more chunks available", async () => {
    const chunks = [
      {
        content: "c1",
        chunk_index: 0,
        source_url: "u",
        source_id: "src-1",
        type: "t",
        language: "en",
        title: "Chunk 0",
      },
    ];
    const source = {
      id: "src-1",
      name: "Test",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 10,
      chunks,
    };
    const { wrapper } = mountView(source);
    await flushPromises();

    expect(wrapper.text()).toContain("Load More");
    expect(wrapper.text()).toContain("1");
    expect(wrapper.text()).toContain("10");
  });

  it("loadMoreChunks appends new chunks", async () => {
    const existingChunks = [
      {
        content: "c1",
        chunk_index: 0,
        source_url: "u",
        source_id: "src-1",
        type: "t",
        language: "en",
        title: "Chunk 0",
      },
    ];
    const source = {
      id: "src-1",
      name: "Test",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 10,
      chunks: existingChunks,
    };
    const { wrapper, store } = mountView(source);
    await flushPromises();

    const newChunk = {
      content: "c2",
      chunk_index: 1,
      source_url: "u",
      source_id: "src-1",
      type: "t",
      language: "en",
      title: "Chunk 1",
    };
    vi.spyOn(store, "fetchChunks").mockResolvedValue([newChunk]);

    // Click Load More
    const loadMoreBtn = wrapper
      .findAll("button")
      .find((b) => b.text().includes("Load More"));
    await loadMoreBtn?.trigger("click");
    await flushPromises();

    expect(store.fetchChunks).toHaveBeenCalledWith("src-1", 1, 100);
    expect(wrapper.text()).toContain("Chunk 1");
  });

  it("copyToClipboard calls navigator.clipboard.writeText", async () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: { writeText: writeTextMock },
    });

    const chunk = {
      content: "Copy this text",
      chunk_index: 0,
      source_url: "http://example.com",
      source_id: "src-1",
      type: "text",
      language: "en",
      title: "To Copy",
    };
    const source = {
      id: "src-1",
      name: "Test",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 1,
      chunks: [chunk],
    };
    const { wrapper } = mountView(source);
    await flushPromises();

    // Find the copy button (button with Copy icon)
    const copyBtn = wrapper
      .findAll("button")
      .find((b) => b.attributes("title") === "Copy Content");
    await copyBtn?.trigger("click");

    expect(writeTextMock).toHaveBeenCalledWith("Copy this text");
  });

  it("renders web URL as external link", async () => {
    const source = {
      id: "src-1",
      name: "Web",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 0,
      chunks: [],
    };
    const { wrapper } = mountView(source);
    await flushPromises();

    const link = wrapper.find('a[href="http://example.com"]');
    expect(link.exists()).toBe(true);
    expect(link.attributes("target")).toBe("_blank");
  });

  it("displays exclusions badges when present", async () => {
    const source = {
      id: "src-1",
      name: "Test",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 0,
      max_depth: 1,
      exclusions: ["/login", "/admin"],
      chunks: [],
    };
    const { wrapper } = mountView(source);
    await flushPromises();

    expect(wrapper.text()).toContain("/login");
    expect(wrapper.text()).toContain("/admin");
  });

  it("navigates back when back button is clicked", async () => {
    const source = {
      id: "src-1",
      name: "Test",
      status: "completed",
      url: "http://example.com",
      type: "web",
      total_chunks: 0,
      chunks: [],
    };
    const { wrapper } = mountView(source);
    await flushPromises();

    // The first button should be the back button
    const buttons = wrapper.findAll("button");
    await buttons[0].trigger("click");

    expect(backMock).toHaveBeenCalled();
  });
});
