<script setup lang="ts">
import AppCellBadge from "@/components/cells/AppCellBadge.vue";

withDefaults(defineProps<{ active: boolean; activeLabel?: string; inactiveLabel?: string }>(), {
  activeLabel: "Active",
  inactiveLabel: "Inactive",
});
</script>

<template>
  <AppCellBadge :color="active ? 'success' : 'error'" :label="active ? activeLabel : inactiveLabel" />
</template>
