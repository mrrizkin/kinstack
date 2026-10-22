import { router, useForm } from "@inertiajs/vue3";
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, ref } from "vue";

import { useServerErrors } from "@/lib/errors";
import { icons } from "@/lib/icons";
import { queryString } from "@/lib/query";

import type { Override, OverrideForm, Permission, PermissionCategory, Role, UserAccess } from "../types";

interface AccessProps {
  roles: Role[];
  permissions: Permission[];
  categories: string[];
  users: UserAccess[];
  page?: { page: number };
  roles_page?: { page: number };
}

export function useAccessForms(props: AccessProps) {
  const accessURL = (suffix = "") => `/access${suffix}`;
  // User-table mutations return to the page the user was viewing. Role
  // mutations live on the roles tab and do not need it.
  const pageQuery = computed(() => {
    const current = props.page?.page ?? 1;
    return queryString({ page: current > 1 ? current : undefined });
  });
  // Role mutations stay on the roles tab, so they preserve that tab's page.
  const rolesPageQuery = computed(() => {
    const current = props.roles_page?.page ?? 1;
    return queryString({ roles_page: current > 1 ? current : undefined });
  });
  const tab = ref("roles");
  const { message } = useServerErrors();
  const actionError = computed(() => message("role") || message("permission") || message("form"));

  // ---- Role permission matrix ----
  const editingRole = ref<Role | null>(null);
  const selectedPermissions = ref<string[]>([]);
  const savingPermissions = ref(false);

  const permissionsByCategory = computed<PermissionCategory[]>(() =>
    props.categories
      .map((category) => ({
        category,
        items: props.permissions.filter((permission) => permission.category === category),
      }))
      .filter((group) => group.items.length > 0),
  );

  function openRolePermissions(role: Role) {
    editingRole.value = role;
    selectedPermissions.value = [...role.permissions];
  }

  function togglePermission(code: string, checked: boolean) {
    const set = new Set(selectedPermissions.value);
    if (checked) set.add(code);
    else set.delete(code);
    selectedPermissions.value = [...set];
  }

  function saveRolePermissions() {
    const role = editingRole.value;
    if (!role || savingPermissions.value) return;
    router.post(
      accessURL(`/roles/${role.code}/permissions${rolesPageQuery.value}`),
      { permissions: selectedPermissions.value },
      {
        preserveScroll: true,
        onStart: () => (savingPermissions.value = true),
        onFinish: () => (savingPermissions.value = false),
        onSuccess: () => (editingRole.value = null),
      },
    );
  }

  // ---- Create role ----
  const showCreateRole = ref(false);
  const createRoleForm = useForm({ code: "", name: "", description: "", rank: 20 });

  function submitCreateRole() {
    createRoleForm.post(accessURL(`/roles${rolesPageQuery.value}`), {
      preserveScroll: true,
      onSuccess: () => {
        createRoleForm.reset();
        showCreateRole.value = false;
      },
    });
  }

  // ---- Delete role (confirmed) ----
  const pendingDeleteRole = ref<Role | null>(null);
  const deletingRole = ref(false);

  function requestDeleteRole(role: Role) {
    pendingDeleteRole.value = role;
  }

  function confirmDeleteRole() {
    const role = pendingDeleteRole.value;
    if (!role || deletingRole.value) return;
    router.post(
      accessURL(`/roles/${role.code}/delete${rolesPageQuery.value}`),
      {},
      {
        preserveScroll: true,
        onStart: () => (deletingRole.value = true),
        onFinish: () => {
          deletingRole.value = false;
          pendingDeleteRole.value = null;
        },
      },
    );
  }

  // ---- Edit role metadata (name/description/rank are data, not code) ----
  const editingRoleMeta = ref<Role | null>(null);
  const editRoleForm = useForm({ name: "", description: "", rank: 20 });

  function openEditRole(role: Role) {
    editingRoleMeta.value = role;
    editRoleForm.clearErrors();
    editRoleForm.name = role.name;
    editRoleForm.description = role.description;
    editRoleForm.rank = role.rank;
  }

  function submitEditRole() {
    const role = editingRoleMeta.value;
    if (!role) return;
    editRoleForm.put(accessURL(`/roles/${role.code}${rolesPageQuery.value}`), {
      preserveScroll: true,
      onSuccess: () => (editingRoleMeta.value = null),
    });
  }

  // ---- User custom permissions ----
  // Track only the selected user's id so the panel always reflects the latest
  // props after an Inertia reload (a stored object snapshot would go stale).
  const editingUserId = ref<string | null>(null);
  const editingUser = computed(() => props.users.find((user) => user.public_id === editingUserId.value) ?? null);
  const overrideForm = ref<OverrideForm>({
    permission: "",
    effect: "allow",
    reason: "",
    expires: "",
  });
  const overrideLocalErrors = ref<Record<string, string>>({});
  const submittingOverride = ref(false);

  const permissionOptions = computed(() =>
    props.permissions.map((permission) => ({
      label: permission.label,
      value: permission.code,
    })),
  );

  const roleOptions = computed(() => props.roles.map((role) => ({ label: role.name, value: role.code })));

  function overrideFieldError(field: string): string | undefined {
    return overrideLocalErrors.value[field] || message(field);
  }

  function openUserOverrides(user: UserAccess) {
    editingUserId.value = user.public_id;
    overrideLocalErrors.value = {};
    overrideForm.value.permission = permissionOptions.value[0]?.value ?? "";
    overrideForm.value.effect = "allow";
    overrideForm.value.reason = "";
    overrideForm.value.expires = "";
  }

  function closeUserOverrides() {
    editingUserId.value = null;
  }

  function submitOverride() {
    const user = editingUser.value;
    if (!user || submittingOverride.value) return;
    overrideLocalErrors.value = {};
    if (!overrideForm.value.permission) {
      overrideLocalErrors.value.permission = "Select a permission first.";
      return;
    }
    router.post(
      accessURL(`/users/${user.public_id}/permissions${pageQuery.value}`),
      {
        permission: overrideForm.value.permission,
        effect: overrideForm.value.effect,
        reason: overrideForm.value.reason,
        expires_at: overrideForm.value.expires,
      },
      {
        preserveScroll: true,
        onStart: () => (submittingOverride.value = true),
        onFinish: () => (submittingOverride.value = false),
      },
    );
  }

  // ---- Delete custom permission (confirmed) ----
  const pendingRemoveOverride = ref<{ user: UserAccess; override: Override } | null>(null);
  const removingOverride = ref(false);

  function requestRemoveOverride(user: UserAccess, override: Override) {
    pendingRemoveOverride.value = { user, override };
  }

  function confirmRemoveOverride() {
    const pending = pendingRemoveOverride.value;
    if (!pending || removingOverride.value) return;
    router.post(
      accessURL(`/users/${pending.user.public_id}/permissions/delete${pageQuery.value}`),
      { permission: pending.override.permission },
      {
        preserveScroll: true,
        onStart: () => (removingOverride.value = true),
        onFinish: () => {
          removingOverride.value = false;
          pendingRemoveOverride.value = null;
        },
      },
    );
  }

  const assigningRoleId = ref<string | null>(null);

  function assignRole(user: UserAccess, role: string) {
    if (assigningRoleId.value) return;
    router.post(
      accessURL(`/users/${user.public_id}/role${pageQuery.value}`),
      { role },
      {
        preserveScroll: true,
        onStart: () => (assigningRoleId.value = user.public_id),
        onFinish: () => (assigningRoleId.value = null),
      },
    );
  }

  function roleMenuItems(role: Role): DropdownMenuItem[] {
    if (role.is_system) {
      // Built-in roles are immutable: their metadata and permission set are
      // owned by migrations, so the UI only reports that they cannot change.
      return [
        {
          label: "Built-in roles are read-only",
          icon: icons.lock,
          disabled: true,
        },
      ];
    }
    return [
      {
        label: "Edit",
        icon: icons.edit,
        onSelect: () => openEditRole(role),
      },
      {
        label: "Manage permissions",
        icon: icons.sliders,
        onSelect: () => openRolePermissions(role),
      },
      {
        label: "Delete",
        icon: icons.delete,
        color: "error",
        onSelect: () => requestDeleteRole(role),
      },
    ];
  }

  function accessUserMenuItems(user: UserAccess): DropdownMenuItem[] {
    return [
      {
        label: "Manage permissions",
        icon: icons.sliders,
        onSelect: () => openUserOverrides(user),
      },
    ];
  }

  return {
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
  };
}
