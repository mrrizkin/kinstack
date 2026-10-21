<script setup lang="ts">
/**
 * Filter trigger with a responsive panel: popover from `md` up, bottom drawer
 * below it. The page owns field state, validation, and navigation through
 * `apply`/`reset`, and calls `close()` after validation passes so errors stay
 * visible while the user fixes them.
 */
import { ref } from "vue";

import { icons } from "@/lib/icons";

interface Props {
  /** Number of filters currently applied; shown as a badge and a stronger trigger. */
  activeCount?: number;
  /** A filter request is in flight. */
  applying?: boolean;
  disabled?: boolean;
  label?: string;
  resetLabel?: string;
  applyLabel?: string;
}

const props = withDefaults(defineProps<Props>(), {
  activeCount: 0,
  applying: false,
  disabled: false,
  label: "Filter",
  resetLabel: "Reset",
  applyLabel: "Apply",
});

const emit = defineEmits<{ apply: []; reset: [] }>();

const popoverOpen = ref(false);
const drawerOpen = ref(false);

function close() {
  popoverOpen.value = false;
  drawerOpen.value = false;
}

defineExpose({ close });
</script>

<template>
  <!-- An open panel stays open across the breakpoint until dismissed: hiding
       a modal overlay with CSS would leave the body scroll locked. -->
  <UPopover
    v-model:open="popoverOpen"
    class="hidden md:inline-flex"
    :content="{ align: 'end', sideOffset: 8, collisionPadding: 12 }"
    :ui="{ content: 'p-0' }">
    <UButton
      type="button"
      color="neutral"
      :variant="props.activeCount > 0 ? 'soft' : 'outline'"
      :icon="icons.filter"
      :label="props.label"
      :disabled="props.disabled">
      <template v-if="props.activeCount > 0" #trailing>
        <UBadge color="primary" variant="solid" size="sm" :label="String(props.activeCount)" />
      </template>
    </UButton>

    <template #content="{ close: closePopover }">
      <form class="w-[26rem] max-w-[calc(100vw-2rem)] space-y-4 p-4" @submit.prevent="emit('apply')">
        <div class="flex items-center justify-between gap-2">
          <p class="text-highlighted text-sm font-semibold">{{ props.label }}</p>
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            size="xs"
            square
            :icon="icons.close"
            aria-label="Close filters"
            @click="closePopover" />
        </div>

        <div class="grid max-h-[min(60vh,24rem)] grid-cols-2 gap-3 overflow-y-auto pr-1">
          <slot />
        </div>

        <div class="border-default flex items-center justify-end gap-2 border-t pt-3">
          <UButton
            type="button"
            color="neutral"
            variant="outline"
            :icon="icons.reset"
            :label="props.resetLabel"
            :disabled="props.applying"
            @click="emit('reset')" />
          <UButton
            type="submit"
            color="primary"
            :icon="icons.filter"
            :label="props.applyLabel"
            :loading="props.applying" />
        </div>
      </form>
    </template>
  </UPopover>

  <UDrawer
    v-model:open="drawerOpen"
    class="md:hidden"
    direction="bottom"
    :title="props.label"
    close
    :ui="{ content: 'md:mx-auto md:max-w-md', body: 'p-4', footer: 'p-4' }">
    <UButton
      type="button"
      color="neutral"
      :variant="props.activeCount > 0 ? 'soft' : 'outline'"
      :icon="icons.filter"
      :label="props.label"
      :disabled="props.disabled">
      <template v-if="props.activeCount > 0" #trailing>
        <UBadge color="primary" variant="solid" size="sm" :label="String(props.activeCount)" />
      </template>
    </UButton>

    <template #body>
      <form id="app-filter-form" class="grid grid-cols-1 gap-3" @submit.prevent="emit('apply')">
        <slot />
      </form>
    </template>

    <template #footer>
      <div class="flex w-full items-center gap-2">
        <UButton
          type="button"
          color="neutral"
          variant="outline"
          :icon="icons.reset"
          :label="props.resetLabel"
          class="flex-1 justify-center"
          :disabled="props.applying"
          @click="emit('reset')" />
        <UButton
          type="submit"
          form="app-filter-form"
          color="primary"
          :label="props.applyLabel"
          :loading="props.applying"
          class="flex-1 justify-center" />
      </div>
    </template>
  </UDrawer>
</template>
