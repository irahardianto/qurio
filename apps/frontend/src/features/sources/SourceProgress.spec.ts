import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import SourceProgress from "./SourceProgress.vue";
import type { SourcePage } from "./source.store";

// Stub UI components to avoid parsing issues
const globalStubs = {
  Card: { template: "<div><slot /></div>" },
  CardHeader: { template: "<div><slot /></div>" },
  CardTitle: { template: "<div><slot /></div>" },
  CardContent: { template: "<div><slot /></div>" },
  Badge: { template: "<span><slot /></span>" },
  Activity: { template: "<svg></svg>" },
  CheckCircle: { template: "<svg></svg>" },
  Clock: { template: "<svg></svg>" },
  AlertCircle: { template: "<svg></svg>" },
};

// Helper to generate valid SourcePage
const createMockPage = (overrides: Partial<SourcePage> = {}): SourcePage => ({
  id: "1",
  source_id: "s1",
  url: "http://example.com",
  status: "pending",
  depth: 0,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  ...overrides,
});

describe("SourceProgress.vue", () => {
  it("renders correct progress stats", () => {
    const pages: SourcePage[] = [
      createMockPage({ id: "1", url: "http://a.com", status: "completed" }),
      createMockPage({ id: "2", url: "http://b.com", status: "pending" }),
      createMockPage({ id: "3", url: "http://c.com", status: "failed" }),
      createMockPage({ id: "4", url: "http://d.com", status: "processing" }),
    ];

    const wrapper = mount(SourceProgress, {
      props: { pages },
      global: { stubs: globalStubs },
    });

    expect(wrapper.text()).toContain("25% (1/4)");
    expect(wrapper.text()).toContain("Completed");
    // Check specific counts (implementation detail: usually finding by text or specific element)
    // Here we trust the computed logic which drives the text we just checked
  });

  it("handles empty pages gracefully", () => {
    const wrapper = mount(SourceProgress, {
      props: { pages: [] },
      global: { stubs: globalStubs },
    });

    expect(wrapper.text()).toContain("0% (0/0)");
    expect(wrapper.text()).toContain("No pages found yet");
  });

  it("renders list of active crawls", () => {
    const pages: SourcePage[] = [
      createMockPage({
        id: "1",
        url: "http://example.com/page1",
        status: "processing",
      }),
    ];

    const wrapper = mount(SourceProgress, {
      props: { pages },
      global: { stubs: globalStubs },
    });

    expect(wrapper.text()).toContain("http://example.com/page1");
    expect(wrapper.text()).toContain("processing");
  });
});
