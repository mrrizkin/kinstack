<script setup lang="ts">
/**
 * One badge recipe, so every table and mobile list renders the same variant.
 */
import type { CellBadgeColor, CellBadgeSize, CellBadgeVariant } from "./types";

withDefaults(
  defineProps<{
    label?: string;
    color?: CellBadgeColor;
    variant?: CellBadgeVariant;
    size?: CellBadgeSize;
    align?: "start" | "end";
  }>(),
  { color: "neutral", variant: "subtle", size: "sm", align: "start" },
);
</script>

<template>
  <div :class="align === 'end' ? 'flex justify-end' : 'inline-flex'">
    <UBadge :color="color" :variant="variant" :size="size">
      <slot>{{ label }}</slot>
    </UBadge>
  </div>
</template>
