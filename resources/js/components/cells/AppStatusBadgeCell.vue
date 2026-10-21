<script setup lang="ts">
/** Active/inactive badge shared by tables and mobile lists. */
import AppCellBadge from "@/components/cells/AppCellBadge.vue";

withDefaults(defineProps<{ active: boolean; activeLabel?: string; inactiveLabel?: string }>(), {
  activeLabel: "Active",
  inactiveLabel: "Inactive",
});
</script>

<template>
  <AppCellBadge :color="active ? 'success' : 'error'" :label="active ? activeLabel : inactiveLabel" />
</template>
