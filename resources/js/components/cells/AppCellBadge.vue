<script setup lang="ts">
/**
 * Badge cell with one shared visual recipe.
 *
 * The wrapper exists so every table and mobile list renders the same variant and
 * size, and so badges can be created from `h()` in a `.ts` column definition.
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
