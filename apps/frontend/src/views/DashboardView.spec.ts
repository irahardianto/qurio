import { mount } from "@vue/test-utils";
import { createTestingPinia } from "@pinia/testing";
import { describe, it, expect, vi } from "vitest";
import DashboardView from "./DashboardView.vue";
import { useStatsStore } from "../features/stats/stats.store";
import { useSourceStore } from "../features/sources/source.store";

// Mock vue-router
vi.mock("vue-router", () => ({
  useRoute: () => ({ path: "/" }),
  useRouter: () => ({ push: vi.fn() }),
}));

const globalStubs = {
  Card: { template: "<div><slot /></div>" },
  CardHeader: { template: "<div><slot /></div>" },
  CardTitle: { template: "<div><slot /></div>" },
  CardContent: { template: "<div><slot /></div>" },
  SourceList: { template: "<div>SourceList</div>" },
  Database: { template: "<svg></svg>" },
  FileText: { template: "<svg></svg>" },
  AlertTriangle: { template: "<svg></svg>" },
  Activity: { template: "<svg></svg>" },
};

describe("DashboardView", () => {
  it("fetches stats and sources on mount", () => {
    mount(DashboardView, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });
    const statsStore = useStatsStore();
    const sourceStore = useSourceStore();

    expect(statsStore.fetchStats).toHaveBeenCalled();
    expect(sourceStore.fetchSources).toHaveBeenCalled();
  });

  it("displays all three stat cards", () => {
    const wrapper = mount(DashboardView, {
      global: {
        plugins: [
          createTestingPinia({
            createSpy: vi.fn,
            initialState: {
              stats: {
                stats: { sources: 12, documents: 340, failed_jobs: 5 },
              },
            },
          }),
        ],
        stubs: globalStubs,
      },
    });

    expect(wrapper.text()).toContain("Total Sources");
    expect(wrapper.text()).toContain("12");
    expect(wrapper.text()).toContain("Indexed Documents");
    expect(wrapper.text()).toContain("340");
    expect(wrapper.text()).toContain("Failed Jobs");
    expect(wrapper.text()).toContain("5");
  });

  it("renders the page header", () => {
    const wrapper = mount(DashboardView, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });

    expect(wrapper.text()).toContain("Dashboard");
    expect(wrapper.text()).toContain("System Overview");
  });

  it("renders the Recent Sources section with SourceList", () => {
    const wrapper = mount(DashboardView, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });

    expect(wrapper.text()).toContain("Recent Sources");
    expect(wrapper.text()).toContain("SourceList");
  });

  it("shows zero stats by default", () => {
    const wrapper = mount(DashboardView, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });

    // Default stats store has all zeros
    const text = wrapper.text();
    expect(text).toContain("0");
  });
});
