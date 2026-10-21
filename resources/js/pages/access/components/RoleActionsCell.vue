<script setup lang="ts">
import AppCellActions from "@/components/cells/AppCellActions.vue";

import type { Role } from "../types";

defineProps<{ role: Role }>();

const emit = defineEmits<{ edit: []; permissions: []; remove: [] }>();
</script>

<template>
  <AppCellActions>
    <UButton size="xs" color="neutral" variant="outline" label="Edit" @click="emit('edit')" />
    <UButton size="xs" color="neutral" variant="ghost" label="Manage permissions" @click="emit('permissions')" />
    <UButton v-if="!role.is_system" size="xs" color="error" variant="link" label="Delete" @click="emit('remove')" />
  </AppCellActions>
</template>
