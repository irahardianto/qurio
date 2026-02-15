import { mount } from "@vue/test-utils";
import { describe, it, expect, vi } from "vitest";
import Sidebar from "./Sidebar.vue";

// Mock vue-router
const routePath = { path: "/" };
vi.mock("vue-router", () => ({
  useRoute: () => routePath,
  RouterLink: {
    template: '<a :href="to" :class="$attrs.class"><slot /></a>',
    props: ["to"],
  },
}));

const globalStubs = {
  LayoutDashboard: { template: "<svg></svg>" },
  Database: { template: "<svg></svg>" },
  Settings: { template: "<svg></svg>" },
  AlertTriangle: { template: "<svg></svg>" },
};

describe("Sidebar", () => {
  it("renders all navigation links", () => {
    const wrapper = mount(Sidebar, {
      global: { stubs: globalStubs },
    });

    expect(wrapper.text()).toContain("Dashboard");
    expect(wrapper.text()).toContain("Sources");
    expect(wrapper.text()).toContain("Failed Jobs");
    expect(wrapper.text()).toContain("Settings");
  });

  it("renders logo area", () => {
    const wrapper = mount(Sidebar, {
      global: { stubs: globalStubs },
    });

    expect(wrapper.text()).toContain("Qurio");
  });

  it("shows system online status", () => {
    const wrapper = mount(Sidebar, {
      global: { stubs: globalStubs },
    });

    expect(wrapper.text()).toContain("System Online");
  });

  it("renders correct navigation hrefs", () => {
    const wrapper = mount(Sidebar, {
      global: { stubs: globalStubs },
    });

    const links = wrapper.findAll("a");
    const hrefs = links.map((l) => l.attributes("href"));

    expect(hrefs).toContain("/");
    expect(hrefs).toContain("/sources");
    expect(hrefs).toContain("/jobs");
    expect(hrefs).toContain("/settings");
  });

  it("highlights active route", () => {
    routePath.path = "/sources";
    const wrapper = mount(Sidebar, {
      global: { stubs: globalStubs },
    });

    const sourcesLink = wrapper
      .findAll("a")
      .find((l) => l.attributes("href") === "/sources");
    expect(sourcesLink?.classes()).toContain("text-primary");
  });
});
