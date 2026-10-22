<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { computed } from "vue";

import AppConfirmDialog from "@/components/AppConfirmDialog.vue";
import AppPageHeader from "@/components/AppPageHeader.vue";
import AppPagination, { type PageInfo } from "@/components/AppPagination.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { icons } from "@/lib/icons";
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
  page: PageInfo;
  roles_page: PageInfo;
}>();

const {
  tab,
  actionError,
  editingRole,
  selectedPermissions,
  savingPermissions,
  permissionsByCategory,
  openRolePermissions,
  togglePermission,
  saveRolePermissions,
  showCreateRole,
  createRoleForm,
  submitCreateRole,
  pendingDeleteRole,
  deletingRole,
  requestDeleteRole,
  confirmDeleteRole,
  editingRoleMeta,
  editRoleForm,
  openEditRole,
  submitEditRole,
  editingUser,
  overrideForm,
  submittingOverride,
  overrideFieldError,
  permissionOptions,
  roleOptions,
  openUserOverrides,
  closeUserOverrides,
  submitOverride,
  pendingRemoveOverride,
  removingOverride,
  requestRemoveOverride,
  confirmRemoveOverride,
  assignRole,
  assigningRoleId,
  roleMenuItems,
  accessUserMenuItems,
} = useAccessForms(props);

const roleView = useTableView(
  "access.roles",
  createRoleColumns({ openEditRole, openRolePermissions, requestDeleteRole }),
  { paged: false },
);
const userView = useTableView(
  "access.users",
  createUserColumns({
    roleOptions: () => roleOptions.value,
    assignRole,
    openUserOverrides,
    assigningRoleId: () => assigningRoleId.value,
  }),
  { paged: false },
);

const emptyRoleActions = computed(() => [
  { label: "Create a role", icon: icons.add, onClick: () => (showCreateRole.value = true) },
]);

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
      <AppPageHeader title="Access" description="Manage roles, permissions, and custom user permissions">
        <template #actions>
          <UButton :icon="icons.add" label="New role" color="primary" size="sm" @click="showCreateRole = true" />
        </template>
      </AppPageHeader>
    </template>

    <template #body>
      <UAlert
        v-if="actionError"
        color="error"
        variant="subtle"
        :icon="icons.warning"
        :description="actionError"
        class="mb-4" />

      <UAlert
        color="neutral"
        variant="subtle"
        :icon="icons.info"
        title="Roles are managed from this page"
        description="Each role has permissions you can configure. Create custom roles or add custom permissions for a single user."
        class="mb-4" />

      <UTabs
        v-model="tab"
        :items="[
          { label: 'Roles & Permissions', value: 'roles', icon: icons.shieldCheck },
          { label: 'User Permissions', value: 'users', icon: icons.userPermissions },
        ]"
        class="w-full">
        <template #content="{ item }">
          <RolePanel
            v-if="item.value === 'roles'"
            :roles="roles"
            :view="roleView"
            :menu-items="roleMenuItems"
            :empty-actions="emptyRoleActions" />
          <UserAccessPanel
            v-else
            :users="users"
            :view="userView"
            :role-options="roleOptions"
            :menu-items="accessUserMenuItems"
            :assign-role="assignRole"
            :assigning-role-id="assigningRoleId" />
          <!-- Each tab pages independently; paging one drops the other's key. -->
          <AppPagination v-if="item.value === 'users'" :page="page" url="/access" />
          <AppPagination v-else :page="props.roles_page" url="/access" page-param="roles_page" />
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
    :saving-permissions="savingPermissions"
    :submit-create="submitCreateRole"
    :submit-edit="submitEditRole"
    :toggle-permission="togglePermission"
    :save-role-permissions="saveRolePermissions" />

  <OverrideDialog
    v-model:open="overrideOpen"
    :user="editingUser"
    :override-form="overrideForm"
    :permission-options="permissionOptions"
    :submitting="submittingOverride"
    :field-error="overrideFieldError"
    :submit-override="submitOverride"
    :request-remove-override="requestRemoveOverride" />

  <AppConfirmDialog
    :open="pendingDeleteRole !== null"
    title="Delete role"
    :description="
      pendingDeleteRole ? `Delete the ${pendingDeleteRole.name} role? Users must be moved to another role first.` : ''
    "
    confirm-label="Delete role"
    :loading="deletingRole"
    @update:open="
      (value: boolean) => {
        if (!value) pendingDeleteRole = null;
      }
    "
    @confirm="confirmDeleteRole" />

  <AppConfirmDialog
    :open="pendingRemoveOverride !== null"
    title="Delete custom permission"
    :description="
      pendingRemoveOverride
        ? `Remove ${pendingRemoveOverride.override.label} from ${pendingRemoveOverride.user.display_name}?`
        : ''
    "
    confirm-label="Delete permission"
    :loading="removingOverride"
    @update:open="
      (value: boolean) => {
        if (!value) pendingRemoveOverride = null;
      }
    "
    @confirm="confirmRemoveOverride" />
</template>
