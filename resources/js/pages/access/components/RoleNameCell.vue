<script setup lang="ts">
import AppCellStack from "@/components/cells/AppCellStack.vue";

import type { Role } from "../types";

defineProps<{ role: Role }>();
</script>

<template>
  <AppCellStack
    :primary="role.name"
    :secondary="role.code"
    :tertiary="role.description || '—'"
    :badges="role.is_system ? [{ label: 'Built-in' }] : []"
    mono="secondary" />
</template>
