<script setup lang="ts">
import AppCellActions from "@/components/cells/AppCellActions.vue";

defineProps<{ active: boolean; busy: boolean }>();

const emit = defineEmits<{ reset: []; toggle: [] }>();
</script>

<template>
  <AppCellActions>
    <UButton size="xs" color="neutral" variant="outline" label="Reset password" @click="emit('reset')" />
    <UButton
      size="xs"
      :color="active ? 'error' : 'success'"
      variant="link"
      :label="active ? 'Deactivate' : 'Activate'"
      :loading="busy"
      @click="emit('toggle')" />
  </AppCellActions>
</template>
