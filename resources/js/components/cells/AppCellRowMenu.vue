<script setup lang="ts">
/**
 * Right-aligned row action menu. Desktop tables and mobile cards share one
 * ellipsis trigger so the same item list drives both layouts.
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
