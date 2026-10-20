<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { computed } from "vue";

import AppPageHeader from "@/components/AppPageHeader.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { useServerErrors } from "@/lib/errors";
import { useTableView } from "@/lib/table-view";

import { createRoleColumns, createUserColumns } from "./columns";
import OverrideDialog from "./components/OverrideDialog.vue";
import RoleDialog from "./components/RoleDialog.vue";
import RolePanel from "./components/RolePanel.vue";
import UserAccessPanel from "./components/UserAccessPanel.vue";
import { useAccessForms } from "./composables/useAccessForms";
import type { Permission, Role, UserAccess } from "./types";

defineOptions({ layout: [BaseLayout, AppLayout] });

const props = defineProps<{
  roles: Role[];
  permissions: Permission[];
  categories: string[];
  users: UserAccess[];
}>();

const { formError } = useServerErrors();

const {
  tab,
  editingRole,
  selectedPermissions,
  permissionsByCategory,
  openRolePermissions,
  togglePermission,
  saveRolePermissions,
  showCreateRole,
  createRoleForm,
  submitCreateRole,
  deleteRole,
  editingRoleMeta,
  editRoleForm,
  openEditRole,
  submitEditRole,
  editingUser,
  overrideForm,
  overrideErrors,
  permissionOptions,
  roleOptions,
  openUserOverrides,
  closeUserOverrides,
  submitOverride,
  removeOverride,
  assignRole,
  roleMenuItems,
  accessUserMenuItems,
} = useAccessForms(props);

const roleView = useTableView("access.roles", createRoleColumns({ openEditRole, openRolePermissions, deleteRole }), {
  paged: false,
});
const userView = useTableView(
  "access.users",
  createUserColumns({ roleOptions: () => roleOptions.value, assignRole, openUserOverrides }),
  { paged: false },
);

const overrideOpen = computed({
  get: () => editingUser.value !== null,
  set: (value: boolean) => {
    if (!value) closeUserOverrides();
  },
});
</script>

<template>
  <Head title="Access" />
  <UDashboardPanel id="access">
    <template #header>
      <AppPageHeader title="Access" description="Manage roles, permissions, and per-user overrides">
        <template #actions>
          <UButton icon="i-bi-plus" label="New role" color="primary" size="sm" @click="showCreateRole = true" />
        </template>
      </AppPageHeader>
    </template>

    <template #body>
      <UAlert
        v-if="formError"
        color="error"
        variant="subtle"
        icon="i-bi-exclamation-triangle"
        :description="formError"
        class="mb-4" />

      <UAlert
        color="neutral"
        variant="subtle"
        icon="i-bi-info-circle"
        title="Roles are managed from this page"
        description="Each role has permissions you can configure. Use this page to create custom roles or per-user permission overrides."
        class="mb-4" />

      <UTabs
        v-model="tab"
        :items="[
          { label: 'Roles & Permissions', value: 'roles', icon: 'i-bi-shield-check' },
          { label: 'User Permissions', value: 'users', icon: 'i-bi-person-lock' },
        ]"
        class="w-full">
        <template #content="{ item }">
          <RolePanel v-if="item.value === 'roles'" :roles="roles" :view="roleView" :menu-items="roleMenuItems" />
          <UserAccessPanel
            v-else
            :users="users"
            :view="userView"
            :role-options="roleOptions"
            :menu-items="accessUserMenuItems"
            :assign-role="assignRole" />
        </template>
      </UTabs>
    </template>
  </UDashboardPanel>

  <RoleDialog
    v-model:create-open="showCreateRole"
    v-model:edit-target="editingRoleMeta"
    v-model:editing-role="editingRole"
    :create-form="createRoleForm"
    :edit-form="editRoleForm"
    :permissions-by-category="permissionsByCategory"
    :selected-permissions="selectedPermissions"
    :submit-create="submitCreateRole"
    :submit-edit="submitEditRole"
    :toggle-permission="togglePermission"
    :save-role-permissions="saveRolePermissions" />

  <OverrideDialog
    v-model:open="overrideOpen"
    :user="editingUser"
    :override-form="overrideForm"
    :permission-options="permissionOptions"
    :override-errors="overrideErrors"
    :submit-override="submitOverride"
    :remove-override="removeOverride" />
</template>
