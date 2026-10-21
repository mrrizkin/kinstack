<script setup lang="ts">
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";

import AppMobileList from "@/components/AppMobileList.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import AppCellSelect from "@/components/cells/AppCellSelect.vue";
import AppCellTime from "@/components/cells/AppCellTime.vue";
import AppStatusBadgeCell from "@/components/cells/AppStatusBadgeCell.vue";
import AppUserIdentityCell from "@/components/cells/AppUserIdentityCell.vue";
import type { TableView } from "@/lib/table-view";

import type { User } from "../columns";

defineProps<{
  users: User[];
  view: TableView;
  roleOptions: { label: string; value: string }[];
  menuItems: (user: User) => DropdownMenuItem[];
  changeRole: (user: User, role: string) => void;
  busyUserId: string | null;
  emptyActions: { label: string; icon: string; onClick: () => void }[];
}>();
</script>

<template>
  <AppTableCard :view="view" class="hidden md:block">
    <UTable :data="users" :columns="view.columns" class="w-full">
      <template #empty>
        <UEmpty
          icon="i-bi-people"
          title="No users yet."
          description="Add a user or send an invitation to get started."
          variant="naked"
          :actions="emptyActions"
          class="py-6" />
      </template>
    </UTable>
  </AppTableCard>

  <div class="md:hidden">
    <AppMobileList v-if="users.length > 0">
      <article v-for="user in users" :key="user.public_id" class="space-y-3 p-4">
        <div class="flex flex-wrap items-start justify-between gap-2">
          <div class="min-w-0">
            <AppUserIdentityCell
              :name="user.display_name"
              :username="user.username"
              :email="user.email"
              :badges="user.invited ? [{ label: 'Invited', color: 'info' }] : []" />
          </div>
          <div class="flex items-center gap-1">
            <AppStatusBadgeCell :active="user.is_active" />
            <UDropdownMenu :items="[menuItems(user)]" :content="{ align: 'end', collisionPadding: 12 }">
              <UButton
                icon="i-material-symbols:more-vert"
                color="neutral"
                variant="ghost"
                size="xs"
                square
                :aria-label="`Actions for ${user.display_name}`" />
            </UDropdownMenu>
          </div>
        </div>

        <div class="space-y-2">
          <div class="flex items-center justify-between gap-3">
            <span class="text-muted text-xs">Role</span>
            <div class="flex items-center gap-2">
              <AppCellSelect
                :model-value="user.role"
                :items="roleOptions"
                size="sm"
                :busy="busyUserId === user.public_id"
                @update:model-value="(role: string) => changeRole(user, role)" />
            </div>
          </div>
          <div class="flex items-center justify-between gap-3">
            <span class="text-muted text-xs">Last login</span>
            <AppCellTime :value="user.last_login_at" align="end" />
          </div>
        </div>
      </article>
    </AppMobileList>
    <UEmpty
      v-else
      icon="i-bi-people"
      title="No users yet."
      description="Add a user or send an invitation to get started."
      variant="naked"
      :actions="emptyActions"
      class="py-10" />
  </div>
</template>
