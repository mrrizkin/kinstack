<script setup lang="ts">
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";

import AppMobileList from "@/components/AppMobileList.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import type { TableView } from "@/lib/table-view";

import type { UserAccess } from "../types";

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
      icon="i-bi-info-circle"
      description="Custom permissions add (allow) or remove (deny) a single permission for a specific user. Deny always takes precedence over roles." />
    <AppTableCard :view="view" class="hidden md:block">
      <UTable :data="users" :columns="view.columns" class="w-full">
        <template #empty>
          <UEmpty
            icon="i-bi-people"
            title="No users yet."
            description="Create a user account before adding custom permissions."
            variant="naked"
            :actions="[{ label: 'Open users', to: '/users', icon: 'i-bi-people' }]"
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
            </div>
            <div class="flex items-center gap-1">
              <UBadge :color="user.is_active ? 'success' : 'error'" variant="subtle" size="sm">
                {{ user.is_active ? "Active" : "Inactive" }}
              </UBadge>
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
          <div class="flex items-center justify-between gap-3">
            <span class="text-muted text-xs">Role</span>
            <div class="flex items-center gap-2">
              <USelect
                :model-value="user.role"
                :items="roleOptions"
                size="sm"
                class="w-44"
                :disabled="assigningRoleId === user.public_id"
                @update:model-value="(value: unknown) => assignRole(user, String(value))" />
              <UIcon
                v-if="assigningRoleId === user.public_id"
                name="i-bi-arrow-repeat"
                class="text-muted size-3.5 animate-spin" />
            </div>
          </div>
          <div class="space-y-1.5">
            <p class="text-muted text-xs">Custom permissions</p>
            <div v-if="user.overrides.length > 0" class="flex flex-wrap gap-1">
              <UBadge
                v-for="override in user.overrides"
                :key="override.permission"
                :color="override.expired ? 'neutral' : override.effect === 'allow' ? 'success' : 'error'"
                variant="subtle"
                size="sm">
                {{ override.effect === "allow" ? "+" : "−" }} {{ override.label }}
              </UBadge>
            </div>
            <p v-else class="text-muted text-xs">None</p>
          </div>
        </article>
      </AppMobileList>
      <UEmpty
        v-else
        icon="i-bi-people"
        title="No users yet."
        description="Create a user account before adding custom permissions."
        variant="naked"
        :actions="[{ label: 'Open users', to: '/users', icon: 'i-bi-people' }]"
        class="py-10" />
    </div>
  </div>
</template>
