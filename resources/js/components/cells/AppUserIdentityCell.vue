<script setup lang="ts">
/**
 * Identity cell: a display name, optional handle and email, and optional badges.
 *
 * Shared by the users and access slices. It takes flat props rather than a `User`
 * type so `resources/js/components` never depends on `resources/js/pages`.
 */
import AppCellBadge from "@/components/cells/AppCellBadge.vue";
import type { CellBadgeSpec } from "@/components/cells/types";

withDefaults(
  defineProps<{
    name: string;
    username?: string;
    email?: string;
    badges?: CellBadgeSpec[];
    truncate?: boolean;
  }>(),
  { badges: () => [], truncate: false },
);
</script>

<template>
  <div>
    <div class="text-highlighted font-medium" :class="{ truncate }">{{ name }}</div>
    <div v-if="username" class="text-muted text-xs">@{{ username }}</div>
    <div v-if="email" class="text-muted truncate text-xs">{{ email }}</div>
    <div v-if="badges.length > 0" class="mt-1 flex flex-wrap gap-1">
      <AppCellBadge v-for="badge in badges" :key="badge.label" :label="badge.label" :color="badge.color" />
    </div>
  </div>
</template>
