<script setup lang="ts">
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";

import AppMobileList from "@/components/AppMobileList.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import AppCellStack from "@/components/cells/AppCellStack.vue";
import type { TableView } from "@/lib/table-view";

import type { Role } from "../types";

defineProps<{
  roles: Role[];
  view: TableView;
  menuItems: (role: Role) => DropdownMenuItem[];
  emptyActions: { label: string; icon: string; onClick: () => void }[];
}>();
</script>

<template>
  <div class="space-y-3">
    <AppTableCard :view="view" class="hidden md:block">
      <UTable :data="roles" :columns="view.columns" class="w-full">
        <template #empty>
          <UEmpty
            icon="i-bi-shield-lock"
            title="No roles yet."
            description="Create a role to group permissions for your team."
            variant="naked"
            :actions="emptyActions"
            class="py-6" />
        </template>
      </UTable>
    </AppTableCard>

    <div class="md:hidden">
      <AppMobileList v-if="roles.length > 0">
        <article v-for="role in roles" :key="role.code" class="space-y-3 p-4">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <AppCellStack
                :primary="role.name"
                :secondary="role.code"
                :tertiary="role.description || '—'"
                :badges="role.is_system ? [{ label: 'Built-in' }] : []"
                mono="secondary" />
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
      <UEmpty
        v-else
        icon="i-bi-shield-lock"
        title="No roles yet."
        description="Create a role to group permissions for your team."
        variant="naked"
        :actions="emptyActions"
        class="py-10" />
    </div>
  </div>
</template>
