<script setup lang="ts">
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";

import AppMobileList from "@/components/AppMobileList.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import type { TableView } from "@/lib/table-view";

import type { Role } from "../types";

defineProps<{
  roles: Role[];
  view: TableView;
  menuItems: (role: Role) => DropdownMenuItem[];
}>();
</script>

<template>
  <div class="space-y-3">
    <AppTableCard :view="view" class="hidden md:block">
      <UTable :data="roles" :columns="view.columns" class="w-full" />
    </AppTableCard>

    <div class="md:hidden">
      <AppMobileList v-if="roles.length > 0">
        <article v-for="role in roles" :key="role.code" class="space-y-3 p-4">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-highlighted font-medium">{{ role.name }}</p>
                <UBadge v-if="role.is_system" color="neutral" variant="subtle" size="sm">Built-in</UBadge>
              </div>
              <p class="text-muted mt-1 font-mono text-xs">{{ role.code }}</p>
              <p class="text-muted mt-1 text-xs">{{ role.description || "—" }}</p>
            </div>
            <UDropdownMenu :items="[menuItems(role)]" :content="{ align: 'end', collisionPadding: 12 }">
              <UButton
                icon="i-material-symbols:more-vert"
                color="neutral"
                variant="ghost"
                size="xs"
                square
                :aria-label="`Actions for role ${role.name}`" />
            </UDropdownMenu>
          </div>
          <div class="grid grid-cols-2 gap-2 text-xs">
            <div class="flex items-center justify-between gap-2">
              <span class="text-muted">Priority</span>
              <span class="text-toned">{{ role.rank }}</span>
            </div>
            <div class="flex items-center justify-between gap-2">
              <span class="text-muted">Permissions</span>
              <span class="text-toned">{{ role.permissions.length }} permissions</span>
            </div>
          </div>
        </article>
      </AppMobileList>
      <UEmpty v-else icon="i-bi-shield-lock" title="No roles yet." variant="naked" class="py-10" />
    </div>
  </div>
</template>
