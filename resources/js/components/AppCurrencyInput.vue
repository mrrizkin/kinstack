<script setup lang="ts">
import { computed } from "vue";

interface Props {
  id?: string;
  name?: string;
  min?: number;
  max?: number;
  step?: number;
  size?: "xs" | "sm" | "md" | "lg" | "xl";
  placeholder?: string;
  disabled?: boolean;
  readonly?: boolean;
  required?: boolean;
  prefix?: string;
  locale?: string;
  currency?: string;
  align?: "left" | "right";
}

const props = withDefaults(defineProps<Props>(), {
  min: 0,
  size: "md",
  placeholder: "0",
  disabled: false,
  readonly: false,
  required: false,
  prefix: "IDR",
  locale: "en-US",
  currency: "IDR",
  align: "right",
});

const model = defineModel<number | undefined>({ default: 0 });

// Parent forms always receive a finite number so totals never become NaN.
const safeModel = computed<number>({
  get: () => model.value ?? 0,
  set: (value) => {
    model.value = Number.isFinite(value) ? value : 0;
  },
});

const badgeSize = computed(() => (props.size === "xl" ? "lg" : props.size === "md" ? "md" : props.size));

// Grouping only (`15.000`): the currency symbol comes from the prefix badge.
const formatOptions = computed<Intl.NumberFormatOptions>(() => ({
  style: "decimal",
  useGrouping: true,
  maximumFractionDigits: 0,
}));
</script>

<template>
  <UFieldGroup :size="size" class="w-full">
    <UBadge :size="badgeSize" color="neutral" variant="outline" :label="prefix" class="rounded-e-none font-medium" />

    <UInputNumber
      :id="id"
      v-model="safeModel"
      class="min-w-0 flex-1"
      :name="name"
      :min="min"
      :max="max"
      :step="step"
      :size="size"
      :locale="locale"
      :format-options="formatOptions"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :required="required"
      :increment="false"
      :decrement="false"
      :disable-wheel-change="true"
      :ui="{ base: align === 'right' ? 'text-right' : '' }" />
  </UFieldGroup>
</template>
