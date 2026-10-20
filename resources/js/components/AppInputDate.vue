<script setup lang="ts">
import type { DateValue } from "@internationalized/date";
import { computed, useTemplateRef } from "vue";

import type { DateRangeInput } from "@/lib/date";
import { fromCalendarDate, toCalendarDate } from "@/lib/date";

interface Props {
  /** When `true`, the model is a `{ start, end }` range instead of a single date. */
  range?: boolean;
  id?: string;
  name?: string;
  size?: "xs" | "sm" | "md" | "lg" | "xl";
  color?: "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral";
  variant?: "outline" | "soft" | "subtle" | "ghost" | "none";
  /** Sample date (`YYYY-MM-DD`) used to render the placeholder segments. */
  placeholder?: string;
  disabled?: boolean;
  readonly?: boolean;
  required?: boolean;
  /** Minimum selectable date as `YYYY-MM-DD`. */
  min?: string;
  /** Maximum selectable date as `YYYY-MM-DD`. */
  max?: string;
  /** Icon shown on the calendar picker button. */
  icon?: string;
  locale?: string;
}

const props = withDefaults(defineProps<Props>(), {
  range: false,
  size: "md",
  disabled: false,
  readonly: false,
  required: false,
  icon: "i-bi-calendar",
  locale: "en-US",
});

/** Single date: `YYYY-MM-DD`. Range: `{ start, end }` as `YYYY-MM-DD`. */
const model = defineModel<string | DateRangeInput>({ default: "" });

const inputDate = useTemplateRef<{
  $el?: HTMLElement;
  inputsRef?: { $el?: HTMLElement }[];
}>("inputDate");

// Prefer the first date segment (as documented by Nuxt UI) and fall back to the field root.
const pickerReference = computed(() => inputDate.value?.inputsRef?.[0]?.$el ?? inputDate.value?.$el);

type CalendarRange = { start: DateValue; end: DateValue };

const calendarValue = computed<DateValue | CalendarRange | null>(() => {
  if (props.range) {
    const value = (model.value ?? {}) as Partial<DateRangeInput>;
    const start = toCalendarDate(value.start);
    const end = toCalendarDate(value.end);
    const from = start ?? end;
    if (!from) return null;
    // Keep a partially selected range incomplete so UCalendar keeps its anchor.
    return (end ? { start: from, end } : { start: from }) as CalendarRange;
  }
  return toCalendarDate(model.value as string) ?? null;
});

function onUpdate(value: unknown) {
  if (props.range) {
    const next = (value ?? {}) as { start?: unknown; end?: unknown };
    model.value = { start: fromCalendarDate(next.start), end: fromCalendarDate(next.end) };
    return;
  }
  model.value = fromCalendarDate(value);
}

const minValue = computed(() => toCalendarDate(props.min));
const maxValue = computed(() => toCalendarDate(props.max));
</script>

<template>
  <UInputDate
    ref="inputDate"
    :model-value="calendarValue"
    :range="range"
    :id="id"
    :name="name"
    :size="size"
    :color="color"
    :variant="variant"
    :placeholder="toCalendarDate(placeholder)"
    :disabled="disabled"
    :readonly="readonly"
    :required="required"
    :min-value="minValue"
    :max-value="maxValue"
    :locale="locale"
    @update:model-value="onUpdate">
    <template #trailing>
      <UPopover :reference="pickerReference">
        <UButton
          color="neutral"
          variant="link"
          :size="size === 'xl' ? 'md' : size"
          :icon="icon"
          :disabled="disabled || readonly"
          :aria-label="range ? 'Select date range' : 'Select date'"
          class="px-0" />

        <template #content>
          <UCalendar
            :model-value="calendarValue"
            :range="range"
            :number-of-months="range ? 2 : 1"
            :min-value="minValue"
            :max-value="maxValue"
            class="p-2"
            @update:model-value="onUpdate" />
        </template>
      </UPopover>
    </template>
  </UInputDate>
</template>
