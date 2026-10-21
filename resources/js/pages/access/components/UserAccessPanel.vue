<script setup lang="ts">
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";

import AppMobileList from "@/components/AppMobileList.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import AppCellSelect from "@/components/cells/AppCellSelect.vue";
import AppStatusBadgeCell from "@/components/cells/AppStatusBadgeCell.vue";
import AppUserIdentityCell from "@/components/cells/AppUserIdentityCell.vue";
import { icons } from "@/lib/icons";
import type { TableView } from "@/lib/table-view";

import type { UserAccess } from "../types";
import UserOverridesCell from "./UserOverridesCell.vue";

defineProps<{
  users: UserAccess[];
  view: TableView;
  roleOptions: { label: string; value: string }[];
  menuItems: (user: UserAccess) => DropdownMenuItem[];
  assignRole: (user: UserAccess, role: string) => void;
  assigningRoleId: string | null;
}>();
</script>

<template>
  <div class="space-y-4">
    <UAlert
      color="neutral"
      variant="subtle"
      :icon="icons.info"
      description="Custom permissions add (allow) or remove (deny) a single permission for a specific user. Deny always takes precedence over roles." />
    <AppTableCard :view="view" class="hidden md:block">
      <UTable :data="users" :columns="view.columns" class="w-full">
        <template #empty>
          <UEmpty
            :icon="icons.users"
            title="No users yet."
            description="Create a user account before adding custom permissions."
            variant="naked"
            :actions="[{ label: 'Open users', to: '/users', icon: icons.users }]"
            class="py-6" />
        </template>
      </UTable>
    </AppTableCard>

    <div class="md:hidden">
      <AppMobileList v-if="users.length > 0">
        <article v-for="user in users" :key="user.public_id" class="space-y-3 p-4">
          <div class="flex flex-wrap items-start justify-between gap-2">
            <div class="min-w-0">
              <AppUserIdentityCell :name="user.display_name" :username="user.username" />
            </div>
            <div class="flex items-center gap-1">
              <AppStatusBadgeCell :active="user.is_active" />
              <UDropdownMenu :items="[menuItems(user)]" :content="{ align: 'end', collisionPadding: 12 }">
                <UButton
                  :icon="icons.more"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  square
                  :aria-label="`Actions for ${user.display_name}`" />
              </UDropdownMenu>
            </div>
          </div>
          <div class="flex items-center justify-between gap-3">
            <span class="text-muted text-xs">Role</span>
            <div class="flex items-center gap-2">
              <AppCellSelect
                :model-value="user.role"
                :items="roleOptions"
                size="sm"
                :busy="assigningRoleId === user.public_id"
                @update:model-value="(role: string) => assignRole(user, role)" />
            </div>
          </div>
          <div class="space-y-1.5">
            <p class="text-muted text-xs">Custom permissions</p>
            <UserOverridesCell :overrides="user.overrides" />
          </div>
        </article>
      </AppMobileList>
      <UEmpty
        v-else
        :icon="icons.users"
        title="No users yet."
        description="Create a user account before adding custom permissions."
        variant="naked"
        :actions="[{ label: 'Open users', to: '/users', icon: icons.users }]"
        class="py-10" />
    </div>
  </div>
</template>
