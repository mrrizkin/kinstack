<script setup lang="ts">
import { Head, router } from "@inertiajs/vue3";
import { computed } from "vue";

import AppPageHeader from "@/components/AppPageHeader.vue";
import AppPagination, { type PageInfo } from "@/components/AppPagination.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { icons } from "@/lib/icons";
import { PERMISSIONS, usePermissions } from "@/lib/permissions";
import { useTableView } from "@/lib/table-view";

import { createUserColumns, type Role, type User } from "./columns";
import UserDialogs from "./components/UserDialogs.vue";
import UserTable from "./components/UserTable.vue";
import { useUserForms } from "./composables/useUserForms";

defineOptions({ layout: [BaseLayout, AppLayout] });

const props = defineProps<{ users: User[]; roles: Role[]; page: PageInfo }>();

const { can } = usePermissions();

const {
  roleOptions,
  formError,
  actionError,
  showCreate,
  createForm,
  submitCreate,
  showInvite,
  headerActions,
  inviteForm,
  submitInvite,
  busyUserId,
  pendingToggle,
  togglingActive,
  toggleActive,
  confirmToggleActive,
  changeRole,
  userMenuItems,
  passwordTarget,
  passwordForm,
  resetPassword,
  confirmResetPassword,
} = useUserForms(props);

const view = useTableView(
  "users",
  createUserColumns({
    roleOptions: () => roleOptions.value,
    changeRole,
    resetPassword,
    toggleActive,
    isBusy: (user) => busyUserId.value === user.public_id,
  }),
  { paged: false },
);

const emptyActions = computed(() => {
  const actions: { label: string; icon: string; onClick: () => void }[] = [];
  if (can(PERMISSIONS.usersManage)) {
    actions.push({ label: "Add user", icon: icons.userAdd, onClick: () => (showCreate.value = true) });
    actions.push({ label: "Invite user", icon: icons.invite, onClick: () => (showInvite.value = true) });
  } else if (can(PERMISSIONS.accessManage)) {
    actions.push({ label: "Open access", icon: icons.access, onClick: () => router.visit("/access") });
  }
  if (actions.length === 0) {
    actions.push({ label: "Back to dashboard", icon: icons.arrowLeft, onClick: () => router.visit("/dashboard") });
  }
  return actions;
});
</script>

<template>
  <Head title="Users" />
  <UDashboardPanel id="users">
    <template #header>
      <AppPageHeader title="Users">
        <span class="text-muted hidden text-sm lg:block">Manage user accounts and roles.</span>
        <template #actions>
          <div class="hidden items-center gap-2 md:flex">
            <UButton
              v-if="can(PERMISSIONS.accessManage)"
              :to="`/access`"
              color="neutral"
              variant="outline"
              :icon="icons.access"
              size="sm"
              label="Access" />
            <UButton
              v-if="can(PERMISSIONS.usersManage)"
              @click="showInvite = !showInvite"
              color="neutral"
              variant="outline"
              :icon="icons.invite"
              size="sm">
              Invite
            </UButton>
            <UButton
              v-if="can(PERMISSIONS.usersManage)"
              @click="showCreate = !showCreate"
              color="primary"
              size="sm"
              :label="showCreate ? 'Close' : 'Add user'" />
          </div>

          <UDropdownMenu
            v-if="headerActions.length > 1"
            class="md:hidden"
            :items="[headerActions]"
            :content="{ align: 'end', collisionPadding: 12 }">
            <UButton :icon="icons.more" color="neutral" variant="outline" size="sm" square aria-label="User actions" />
          </UDropdownMenu>
          <UButton
            v-else-if="headerActions.length === 1"
            class="md:hidden"
            :to="headerActions[0]?.to"
            :label="headerActions[0]?.label"
            :icon="headerActions[0]?.icon"
            color="primary"
            size="sm"
            @click="headerActions[0]?.onSelect?.($event)" />
        </template>
      </AppPageHeader>
    </template>

    <template #body>
      <div class="space-y-6">
        <UAlert
          v-if="formError || actionError"
          color="error"
          variant="subtle"
          :title="formError || actionError"
          class="mb-4" />

        <UserDialogs
          v-model:show-create="showCreate"
          v-model:show-invite="showInvite"
          v-model:password-target="passwordTarget"
          v-model:pending-toggle="pendingToggle"
          :role-options="roleOptions"
          :create-form="createForm"
          :invite-form="inviteForm"
          :password-form="passwordForm"
          :toggling-active="togglingActive"
          :submit-create="submitCreate"
          :submit-invite="submitInvite"
          :confirm-reset-password="confirmResetPassword"
          :confirm-toggle-active="confirmToggleActive" />

        <UserTable
          :users="props.users"
          :view="view"
          :role-options="roleOptions"
          :menu-items="userMenuItems"
          :change-role="changeRole"
          :busy-user-id="busyUserId"
          :empty-actions="emptyActions" />

        <AppPagination :page="props.page" url="/users" />
      </div>
    </template>
  </UDashboardPanel>
</template>
