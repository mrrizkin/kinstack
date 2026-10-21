<script setup lang="ts">
import AppCellBadge from "@/components/cells/AppCellBadge.vue";

import type { Override } from "../types";

defineProps<{ overrides: Override[] }>();
</script>

<template>
  <span v-if="overrides.length === 0" class="text-muted text-xs">None</span>
  <div v-else class="flex flex-wrap gap-1">
    <AppCellBadge
      v-for="override in overrides"
      :key="override.permission"
      :color="override.expired ? 'neutral' : override.effect === 'allow' ? 'success' : 'error'"
      :label="`${override.effect === 'allow' ? '+' : '−'} ${override.label}`" />
  </div>
</template>
