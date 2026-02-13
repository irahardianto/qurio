import { createRouter, createWebHistory } from "vue-router";
import DashboardView from "../views/DashboardView.vue";
import SourcesView from "../views/SourcesView.vue";
import SettingsView from "../views/SettingsView.vue";
import SourceDetailView from "../views/SourceDetailView.vue";
import JobsView from "../views/JobsView.vue";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      name: "dashboard",
      component: DashboardView,
    },
    {
      path: "/sources",
      name: "sources",
      component: SourcesView,
    },
    {
      path: "/sources/:id",
      name: "source-detail",
      component: SourceDetailView,
    },
    {
      path: "/jobs",
      name: "jobs",
      component: JobsView,
    },
    {
      path: "/settings",
      name: "settings",
      component: SettingsView,
    },
  ],
});

export default router;
