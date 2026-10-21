<script setup lang="ts">
import AppCellActions from "@/components/cells/AppCellActions.vue";

const emit = defineEmits<{ manage: [] }>();
</script>

<template>
  <AppCellActions>
    <UButton size="xs" color="neutral" variant="outline" label="Manage permissions" @click="emit('manage')" />
  </AppCellActions>
</template>
