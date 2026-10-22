<script setup lang="ts">
/** Presentational: the page owns the presets, navigation, and active state. */
interface Item {
  key: string;
  label: string;
  active: boolean;
  icon?: string;
}

const props = withDefaults(defineProps<{ items: Item[]; disabled?: boolean }>(), { disabled: false });

const emit = defineEmits<{ select: [key: string] }>();
</script>

<template>
  <div v-if="props.items.length > 0" class="-mx-1 flex min-w-0 gap-1.5 overflow-x-auto px-1 py-0.5">
    <UButton
      v-for="item in props.items"
      :key="item.key"
      type="button"
      size="sm"
      class="shrink-0"
      :color="item.active ? 'primary' : 'neutral'"
      :variant="item.active ? 'soft' : 'outline'"
      :aria-pressed="item.active"
      :disabled="props.disabled"
      :icon="item.icon"
      :label="item.label"
      @click="emit('select', item.key)" />
  </div>
</template>
