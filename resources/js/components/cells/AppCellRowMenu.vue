<script setup lang="ts">
/**
 * Row action menu, shared by desktop tables and mobile cards.
 */
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";

import { icons } from "@/lib/icons";

defineProps<{ items: DropdownMenuItem[]; label: string }>();
</script>

<template>
  <div class="flex justify-end">
    <UDropdownMenu :items="[items]" :content="{ align: 'end', collisionPadding: 12 }">
      <UButton :icon="icons.more" color="neutral" variant="ghost" size="xs" square :aria-label="label" />
    </UDropdownMenu>
  </div>
</template>
