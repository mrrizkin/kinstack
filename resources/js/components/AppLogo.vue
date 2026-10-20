<script setup lang="ts">
withDefaults(defineProps<{ compact?: boolean }>(), { compact: false });
</script>

<template>
  <div class="flex items-center gap-2">
    <span
      class="bg-primary text-inverted grid size-9 place-items-center rounded-md text-sm font-bold"
      aria-hidden="true">
      K
    </span>
    <span v-if="!compact" class="text-highlighted text-lg font-semibold tracking-tight"> KinStack </span>
  </div>
</template>
