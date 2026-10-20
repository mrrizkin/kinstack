<script setup lang="ts">
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";

import AppMobileList from "@/components/AppMobileList.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import { formatDateTime } from "@/lib/formatters";
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
            <p class="text-highlighted font-medium">{{ user.display_name }}</p>
            <p class="text-muted text-xs">@{{ user.username }}</p>
            <p v-if="user.email" class="text-muted truncate text-xs">{{ user.email }}</p>
          </div>
          <div class="flex items-center gap-1">
            <UBadge :color="user.is_active ? 'success' : 'error'" variant="subtle" size="sm">
              {{ user.is_active ? "Active" : "Inactive" }}
            </UBadge>
            <UBadge v-if="user.invited" color="info" variant="subtle" size="sm">Invited</UBadge>
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
              <USelect
                :model-value="user.role"
                :items="roleOptions"
                size="sm"
                class="w-44"
                :disabled="busyUserId === user.public_id"
                @update:model-value="(value: unknown) => changeRole(user, String(value))" />
              <UIcon
                v-if="busyUserId === user.public_id"
                name="i-bi-arrow-repeat"
                class="text-muted size-3.5 animate-spin" />
            </div>
          </div>
          <div class="flex items-center justify-between gap-3">
            <span class="text-muted text-xs">Last login</span>
            <span class="text-toned text-right text-xs">{{ formatDateTime(user.last_login_at) }}</span>
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
