<script setup lang="ts">
/**
 * Select cell with an inline busy indicator.
 *
 * Used for role assignment in the users and access slices. Emits the plain
 * string value so callers do not repeat the `String(value)` narrowing.
 */
import { icons } from "@/lib/icons";

withDefaults(
  defineProps<{
    modelValue: string;
    items: { label: string; value: string }[];
    busy?: boolean;
    disabled?: boolean;
    size?: "xs" | "sm";
  }>(),
  { busy: false, disabled: false, size: "xs" },
);

const emit = defineEmits<{ "update:modelValue": [value: string] }>();
</script>

<template>
  <div class="flex items-center gap-2">
    <USelect
      :model-value="modelValue"
      :items="items"
      :size="size"
      :class="size === 'sm' ? 'w-44' : 'w-40'"
      :disabled="busy || disabled"
      @update:model-value="(value: unknown) => emit('update:modelValue', String(value))" />
    <UIcon v-if="busy" :name="icons.loading" class="text-muted size-3.5 animate-spin" />
  </div>
</template>
