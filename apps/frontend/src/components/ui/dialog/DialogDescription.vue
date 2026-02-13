<script setup lang="ts">
import { DialogDescription, type DialogDescriptionProps } from "reka-ui";
import { cn } from "@/lib/utils";
import { type HTMLAttributes } from "vue";
import { reactiveOmit } from "@vueuse/core";

const props = withDefaults(
  defineProps<DialogDescriptionProps & { class?: HTMLAttributes["class"] }>(),
  {
    class: undefined,
  },
);

const delegatedProps = reactiveOmit(props, "class");
</script>

<template>
  <DialogDescription
    v-bind="delegatedProps"
    :class="cn('text-sm text-muted-foreground', props.class)"
  >
    <slot />
  </DialogDescription>
</template>
