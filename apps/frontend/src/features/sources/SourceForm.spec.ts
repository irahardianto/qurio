import { mount } from "@vue/test-utils";
import { describe, it, expect, vi, type Mock } from "vitest";
import { createTestingPinia } from "@pinia/testing";
import SourceForm from "./SourceForm.vue";
import { useSourceStore } from "./source.store";
import { useSettingsStore } from "../settings/settings.store";
import { nextTick } from "vue";

// Mock vue-router
const pushMock = vi.fn();
vi.mock("vue-router", () => ({
  useRouter: () => ({
    push: pushMock,
  }),
}));

// Global Stubs
const globalStubs = {
  Button: { template: "<button><slot /></button>" },
  // Use real input to ensure v-model works correctly in tests or stick to simple stub if v-model binding is compatible
  // Vitest/Vue Test Utils handle v-model on simple elements well.
  // But if Input is a component wrapping input, we need to be careful.
  // The code imports Input from '@/components/ui/input'. This is a component.
  // If we stub it as '<input />', v-model on the component needs to bind to 'modelValue' prop and emit 'update:modelValue'.
  // Simple '<input />' stub might not forward v-model correctly if the test sets value on the stub root.
  // Better to use a functional stub that emits input events.
  Input: {
    template:
      '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" :type="type" :placeholder="placeholder" :disabled="disabled" :min="min" :max="max" />',
    props: ["modelValue", "type", "placeholder", "disabled", "min", "max"],
  },
  Textarea: {
    template:
      '<textarea :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ["modelValue", "placeholder"],
  },
  Globe: { template: "<svg></svg>" },
  FileUp: { template: "<svg></svg>" },
  Loader2: { template: "<svg></svg>" },
  Plus: { template: "<svg></svg>" },
  Settings2: { template: "<svg></svg>" },
  ChevronDown: { template: "<svg></svg>" },
  ChevronUp: { template: "<svg></svg>" },
  UploadCloud: { template: "<svg></svg>" },
  Dialog: { template: "<div><slot /></div>" },
  DialogContent: { template: "<div><slot /></div>" },
  DialogHeader: { template: "<div><slot /></div>" },
  DialogTitle: { template: "<div><slot /></div>" },
  DialogDescription: { template: "<div><slot /></div>" },
  DialogFooter: { template: "<div><slot /></div>" },
};

describe("SourceForm", () => {
  it("calls addSource on submit with advanced config", async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });
    const store = useSourceStore();
    const settingsStore = useSettingsStore();
    settingsStore.geminiApiKey = "test-key";

    const nameInput = wrapper.find(
      'input[placeholder="e.g., Company Documentation"]',
    );
    await nameInput.setValue("https://example.com");
    const urlInput = wrapper.find(
      'input[placeholder="https://docs.example.com"]',
    );
    await urlInput.setValue("https://example.com");

    // Toggle Advanced
    // const toggle = wrapper.find('button[type="button"]'); // First button is toggle in this context if we are careful, but tabs are buttons too.
    // The tabs are buttons. The advanced toggle is a button.
    // Tabs are first.
    const buttons = wrapper.findAll("button");
    const advancedToggle = buttons.find((b) =>
      b.text().includes("Configuration"),
    );
    await advancedToggle?.trigger("click");

    const depthInput = wrapper.find('input[type="number"]');
    await depthInput.setValue(2);

    const textarea = wrapper.find("textarea");
    await textarea.setValue("/login\n/admin");

    await wrapper.find("form").trigger("submit");

    expect(store.addSource).toHaveBeenCalledWith({
      name: "https://example.com",
      url: "https://example.com",
      max_depth: 2,
      exclusions: ["/login", "/admin"],
    });
  });

  it("validates URL format", async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });
    const store = useSourceStore();
    const settingsStore = useSettingsStore();
    settingsStore.geminiApiKey = "test-key";
    const alertMock = vi.spyOn(window, "alert").mockImplementation(() => { });

    const nameInput = wrapper.find(
      'input[placeholder="e.g., Company Documentation"]',
    );
    await nameInput.setValue("Invalid URL Test");
    const urlInput = wrapper.find(
      'input[placeholder="https://docs.example.com"]',
    );
    await urlInput.setValue("invalid-url");

    await wrapper.find("form").trigger("submit");

    expect(alertMock).toHaveBeenCalled();
    expect(store.addSource).not.toHaveBeenCalled();
  });

  it("handles file upload", async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });
    const store = useSourceStore();
    const settingsStore = useSettingsStore();
    settingsStore.geminiApiKey = "test-key";

    // Switch to File Tab
    const buttons = wrapper.findAll("button");
    const fileTab = buttons.find((b) => b.text().includes("File Upload"));
    await fileTab?.trigger("click");

    // Set Name
    const nameInput = wrapper.find(
      'input[placeholder="e.g., Quarterly Report 2024"]',
    );
    await nameInput.setValue("Test File");

    // Trigger file change
    const fileInput = wrapper.find('input[type="file"]');
    const file = new File(["content"], "test.pdf", { type: "application/pdf" });

    // Simulate file selection
    Object.defineProperty(fileInput.element, "files", { value: [file] });
    await fileInput.trigger("change");

    await wrapper.find("form").trigger("submit");

    expect(store.uploadSource).toHaveBeenCalled();
  });

  it("shows error message from store", async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [
          createTestingPinia({
            createSpy: vi.fn,
            initialState: { sources: { error: "Something went wrong" } },
          }),
        ],
        stubs: globalStubs,
      },
    });

    expect(wrapper.text()).toContain("Something went wrong");
  });

  it("validates empty url", async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });

    const settingsStore = useSettingsStore();
    settingsStore.geminiApiKey = "test-key";

    // Default tab is Web, so just submit empty
    await wrapper.find("form").trigger("submit");

    // Check that store was NOT called
    const store = useSourceStore();
    expect(store.addSource).not.toHaveBeenCalled();
  });

  it("resets form after success", async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });
    const store = useSourceStore();
    const settingsStore = useSettingsStore();
    settingsStore.geminiApiKey = "test-key";
    (store.addSource as Mock).mockResolvedValue(undefined);

    await wrapper
      .find('input[placeholder="e.g., Company Documentation"]')
      .setValue("Test Source");
    await wrapper
      .find('input[placeholder="https://docs.example.com"]')
      .setValue("http://example.com");
    await wrapper.find("form").trigger("submit");

    expect(store.addSource).toHaveBeenCalled();
    // Verify input cleared
    expect(
      (
        wrapper.find('input[placeholder="e.g., Company Documentation"]')
          .element as HTMLInputElement
      ).value,
    ).toBe("");
    expect(
      (
        wrapper.find('input[placeholder="https://docs.example.com"]')
          .element as HTMLInputElement
      ).value,
    ).toBe("");
  });

  it("shows alert dialog when gemini api key is missing on web ingest", async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });

    const settingsStore = useSettingsStore();
    const sourceStore = useSourceStore();
    settingsStore.geminiApiKey = "";

    // Set URL
    await wrapper
      .find('input[placeholder="e.g., Company Documentation"]')
      .setValue("Test Source");
    await wrapper
      .find('input[placeholder="https://docs.example.com"]')
      .setValue("https://example.com");

    // Click submit
    await wrapper.find("form").trigger("submit");

    // Expect sourceStore.addSource NOT to be called
    expect(sourceStore.addSource).not.toHaveBeenCalled();

    // Check if showApiKeyAlert became true.
    // Since we mocked Dialog, we can check the vm state directly
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    expect((wrapper.vm as any).showApiKeyAlert).toBe(true);
  });

  it("displays correct crawl depth hints", async () => {
    const wrapper = mount(SourceForm, {
      global: {
        plugins: [createTestingPinia({ createSpy: vi.fn })],
        stubs: globalStubs,
      },
    });
    // Open advanced settings
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (wrapper.vm as any).showAdvanced = true;
    await nextTick();

    // Check for spans in the flex container
    const hints = wrapper.findAll(".flex.flex-row.gap-4.text-xs span");
    expect(hints.length).toBe(3);
    expect(hints[2].text()).toContain("2+ = Deep recursive (Caution)");
  });
});
