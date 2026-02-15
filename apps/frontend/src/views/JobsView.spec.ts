import { mount } from "@vue/test-utils";
import { createTestingPinia } from "@pinia/testing";
import { describe, it, expect, vi, type Mock } from "vitest";
import JobsView from "./JobsView.vue";
import { useJobStore } from "../features/jobs/job.store";

// Mock vue-router (not used directly, but lucide icons import might reference it)
vi.mock("vue-router", () => ({
  useRoute: () => ({ path: "/jobs" }),
  useRouter: () => ({ push: vi.fn() }),
}));

const globalStubs = {
  Button: {
    template:
      '<button @click="$emit(\'click\')" :disabled="disabled"><slot /></button>',
    props: ["disabled"],
  },
  Badge: { template: "<span><slot /></span>", props: ["variant"] },
  RefreshCw: { template: "<svg></svg>" },
  CheckCircle: { template: "<svg></svg>" },
  AlertOctagon: { template: "<svg></svg>" },
  Terminal: { template: "<svg></svg>" },
};

describe("JobsView", () => {
  it("fetches failed jobs on mount", () => {
    mount(JobsView, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });
    const store = useJobStore();
    expect(store.fetchFailedJobs).toHaveBeenCalled();
  });

  it("shows empty state when no failed jobs", () => {
    const wrapper = mount(JobsView, {
      global: {
        plugins: [
          createTestingPinia({
            createSpy: vi.fn,
            initialState: { jobs: { jobs: [], isLoading: false } },
          }),
        ],
        stubs: globalStubs,
      },
    });

    expect(wrapper.text()).toContain("All Systems Operational");
    expect(wrapper.text()).toContain("No ingestion failures");
  });

  it("renders job table with data", () => {
    const jobs = [
      {
        id: "abcdef12-3456-7890-abcd-ef1234567890",
        source_id: "s-1",
        handler: "web",
        payload: {},
        error: "Connection timeout",
        retries: 3,
        created_at: "2026-01-15T10:30:00Z",
      },
    ];
    const wrapper = mount(JobsView, {
      global: {
        plugins: [
          createTestingPinia({
            createSpy: vi.fn,
            initialState: { jobs: { jobs, isLoading: false } },
          }),
        ],
        stubs: globalStubs,
      },
    });

    expect(wrapper.text()).toContain("abcdef12");
    expect(wrapper.text()).toContain("s-1");
    expect(wrapper.text()).toContain("Connection timeout");
    expect(wrapper.text()).toContain("Failed (3)");
  });

  it("refresh button calls fetchFailedJobs", async () => {
    const wrapper = mount(JobsView, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });
    const store = useJobStore();

    // Reset call count from mount
    (store.fetchFailedJobs as Mock).mockClear();

    const refreshBtn = wrapper
      .findAll("button")
      .find((b) => b.text().includes("Refresh Logs"));
    await refreshBtn?.trigger("click");

    expect(store.fetchFailedJobs).toHaveBeenCalled();
  });

  it("retry button calls retryJob with correct ID", async () => {
    const jobs = [
      {
        id: "job-123",
        source_id: "s-1",
        handler: "web",
        payload: {},
        error: "fail",
        retries: 1,
        created_at: "2026-01-15T10:30:00Z",
      },
    ];
    const wrapper = mount(JobsView, {
      global: {
        plugins: [
          createTestingPinia({
            createSpy: vi.fn,
            initialState: { jobs: { jobs, isLoading: false } },
          }),
        ],
        stubs: globalStubs,
      },
    });
    const store = useJobStore();

    // Find retry button (the one with sr-only "Retry" text)
    const retryBtn = wrapper
      .findAll("button")
      .find((b) => b.text().includes("Retry"));
    await retryBtn?.trigger("click");

    expect(store.retryJob).toHaveBeenCalledWith("job-123");
  });

  it("loading state disables buttons", () => {
    const wrapper = mount(JobsView, {
      global: {
        plugins: [
          createTestingPinia({
            createSpy: vi.fn,
            initialState: { jobs: { jobs: [], isLoading: true } },
          }),
        ],
        stubs: globalStubs,
      },
    });

    const refreshBtn = wrapper
      .findAll("button")
      .find((b) => b.text().includes("Refresh Logs"));
    expect(refreshBtn?.attributes("disabled")).toBeDefined();
  });

  it("formatDate formats dates correctly", () => {
    const jobs = [
      {
        id: "job-1",
        source_id: "s1",
        handler: "web",
        payload: {},
        error: "err",
        retries: 0,
        created_at: "2026-06-15T14:30:00Z",
      },
    ];
    const wrapper = mount(JobsView, {
      global: {
        plugins: [
          createTestingPinia({
            createSpy: vi.fn,
            initialState: { jobs: { jobs, isLoading: false } },
          }),
        ],
        stubs: globalStubs,
      },
    });

    // The formatted date should contain "Jun" and "15"
    const text = wrapper.text();
    expect(text).toContain("Jun");
    expect(text).toContain("15");
  });

  it("renders page header and subtitle", () => {
    const wrapper = mount(JobsView, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });

    expect(wrapper.text()).toContain("System Monitor");
    expect(wrapper.text()).toContain("Ingestion Failure Logs");
  });
});
