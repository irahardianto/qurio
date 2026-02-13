<script setup lang="ts">
import { DialogTitle, type DialogTitleProps } from "reka-ui";
import { cn } from "@/lib/utils";
import { type HTMLAttributes } from "vue";
import { reactiveOmit } from "@vueuse/core";

const props = withDefaults(
  defineProps<DialogTitleProps & { class?: HTMLAttributes["class"] }>(),
  {
    class: undefined,
  },
);

const delegatedProps = reactiveOmit(props, "class");
</script>

<template>
  <DialogTitle
    v-bind="delegatedProps"
    :class="
      cn(
        'text-lg font-semibold leading-none tracking-tight text-foreground',
        props.class,
      )
    "
  >
    <slot />
  </DialogTitle>
</template>
