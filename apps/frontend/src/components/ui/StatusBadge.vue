<script setup lang="ts">
import { computed } from "vue";
import { Badge } from "@/components/ui/badge";

const props = defineProps<{
  status: string;
}>();

const variant = computed(() => {
  switch (props.status.toLowerCase()) {
    case "indexed":
    case "completed":
      return "default";
    case "processing":
    case "pending":
    case "in_progress":
      return "secondary";
    case "failed":
      return "destructive";
    default:
      return "outline";
  }
});
</script>

<template>
  <Badge
    :variant="variant"
    class="capitalize"
  >
    {{ status }}
  </Badge>
</template>
