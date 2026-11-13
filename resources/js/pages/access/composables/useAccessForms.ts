import { router } from "@inertiajs/vue3";
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, ref } from "vue";

import { dialogHooks, formView, openModel, subjectModel, useConfirmation, useDialog } from "@/lib/dialog";
import { useServerErrors } from "@/lib/errors";
import { icons } from "@/lib/icons";
import { queryString } from "@/lib/query";
import { useRoleChange } from "@/lib/role-change";

import type { Override, OverrideForm, Permission, PermissionCategory, Role, UserAccess } from "../types";

interface AccessProps {
  roles: Role[];
  role_options: Role[];
  permissions: Permission[];
  categories: string[];
  users: UserAccess[];
  page?: { page: number };
  roles_page?: { page: number };
}

interface RoleCreateValues {
  code: string;
  name: string;
  description: string;
  rank: number;
}

interface RoleEditValues {
  name: string;
  description: string;
  rank: number;
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

  const permissionsByCategory = computed<PermissionCategory[]>(() =>
    props.categories
      .map((category) => ({
        category,
        items: props.permissions.filter((permission) => permission.category === category),
      }))
      .filter((group) => group.items.length > 0),
  );

  const permissionOptions = computed(() =>
    props.permissions.map((permission) => ({
      label: permission.label,
      value: permission.code,
    })),
  );

  const roleOptions = computed(() => props.role_options.map((role) => ({ label: role.name, value: role.code })));

  const permissions = useDialog<Role, { permissions: string[] }>({
    label: "role permissions",
    initialValues: (role) => ({ permissions: role ? [...role.permissions] : [] }),
    idOf: (role) => role.code,
  });

  const editingRole = subjectModel(permissions);
  const selectedPermissions = computed(() => permissions.values.permissions);
  const savingPermissions = permissions.isSaving;

  function openRolePermissions(role: Role) {
    if (permissions.isOpen.value) {
      return;
    }
    permissions.open(role);
  }

  function togglePermission(code: string, checked: boolean) {
    const current = permissions.values.permissions;
    permissions.values.permissions = checked
      ? [...new Set([...current, code])]
      : current.filter((item) => item !== code);
  }

  function saveRolePermissions() {
    const role = permissions.subject.value;
    if (!role || permissions.isSaving.value) {
      return;
    }
    router.post(
      accessURL(`/roles/${role.code}/permissions${rolesPageQuery.value}`),
      { permissions: permissions.values.permissions },
      { preserveScroll: true, ...dialogHooks(permissions) },
    );
  }

  const createRole = useDialog<null, RoleCreateValues>({
    label: "create role",
    initialValues: () => ({ code: "", name: "", description: "", rank: 20 }),
  });

  const showCreateRole = openModel(createRole);
  const createRoleForm = formView(createRole);

  function submitCreateRole() {
    const { code, name, description, rank } = createRole.values;
    router.post(
      accessURL(`/roles${rolesPageQuery.value}`),
      { code, name, description, rank },
      { preserveScroll: true, ...dialogHooks(createRole) },
    );
  }

  const deleteRole = useConfirmation<Role>({ label: "delete role", idOf: (role) => role.code });
  const pendingDeleteRole = subjectModel(deleteRole);
  const deletingRole = deleteRole.isRunning;

  function requestDeleteRole(role: Role) {
    if (deleteRole.isOpen.value) {
      return;
    }
    deleteRole.open(role);
  }

  function confirmDeleteRole() {
    const role = deleteRole.subject.value;
    if (!role || deleteRole.isRunning.value) {
      return;
    }
    deleteRole.confirm();
    router.post(
      accessURL(`/roles/${role.code}/delete${rolesPageQuery.value}`),
      {},
      {
        preserveScroll: true,
        onFinish: () => deleteRole.done(),
      },
    );
  }

  const editRole = useDialog<Role, RoleEditValues>({
    label: "edit role",
    initialValues: (role) => ({
      name: role?.name ?? "",
      description: role?.description ?? "",
      rank: role?.rank ?? 20,
    }),
    idOf: (role) => role.code,
  });

  const editingRoleMeta = subjectModel(editRole);
  const editRoleForm = formView(editRole);

  function openEditRole(role: Role) {
    if (editRole.isOpen.value) {
      return;
    }
    editRole.open(role);
  }

  function submitEditRole() {
    const role = editRole.subject.value;
    if (!role || editRole.isSaving.value) {
      return;
    }
    const { name, description, rank } = editRole.values;
    router.put(
      accessURL(`/roles/${role.code}${rolesPageQuery.value}`),
      { name, description, rank },
      { preserveScroll: true, ...dialogHooks(editRole) },
    );
  }

  // The dialog tracks the user's id, so the panel always reflects the latest props.
  const overrides = useDialog<{ id: string }, OverrideForm>({
    label: "user overrides",
    initialValues: () => ({
      permission: permissionOptions.value[0]?.value ?? "",
      effect: "allow",
      reason: "",
      expires: "",
    }),
    idOf: (subject) => subject.id,
  });

  const editingUser = computed(
    () => props.users.find((user) => user.public_id === overrides.subject.value?.id) ?? null,
  );
  const overrideForm = formView(overrides);
  const submittingOverride = overrides.isSaving;

  function overrideFieldError(field: string): string | undefined {
    return overrides.errors.value[field] || message(field);
  }

  function openUserOverrides(user: UserAccess) {
    if (overrides.isOpen.value) {
      return;
    }
    overrides.open({ id: user.public_id });
  }

  function closeUserOverrides() {
    overrides.close();
  }

  function submitOverride() {
    const user = editingUser.value;
    if (!user || overrides.isSaving.value) {
      return;
    }
    const { permission, effect, reason, expires } = overrides.values;
    if (!permission) {
      overrides.invalid({ permission: "Select a permission first." });
      return;
    }
    // A successful save keeps the panel open so the new override is visible.
    router.post(
      accessURL(`/users/${user.public_id}/permissions${pageQuery.value}`),
      { permission, effect, reason, expires_at: expires },
      { preserveScroll: true, ...dialogHooks(overrides, { keepOpen: true }) },
    );
  }

  const removeOverride = useConfirmation<{ user: UserAccess; override: Override }>({
    label: "remove custom permission",
    idOf: (pending) => `${pending.user.public_id}:${pending.override.permission}`,
  });

  const pendingRemoveOverride = subjectModel(removeOverride);
  const removingOverride = removeOverride.isRunning;

  function requestRemoveOverride(user: UserAccess, override: Override) {
    if (removeOverride.isOpen.value) {
      return;
    }
    removeOverride.open({ user, override });
  }

  function confirmRemoveOverride() {
    const pending = removeOverride.subject.value;
    if (!pending || removeOverride.isRunning.value) {
      return;
    }
    removeOverride.confirm();
    router.post(
      accessURL(`/users/${pending.user.public_id}/permissions/delete${pageQuery.value}`),
      { permission: pending.override.permission },
      { preserveScroll: true, onFinish: () => removeOverride.done() },
    );
  }

  const roleChange = useRoleChange();
  const roleTarget = roleChange.target;
  const roleOpen = roleChange.isOpen;
  const roleSaving = roleChange.isSaving;
  const roleValue = roleChange.role;

  function requestRoleChange(user: UserAccess) {
    if (roleSaving.value) {
      return;
    }
    roleChange.open({ id: user.public_id, name: user.display_name, currentRole: user.role });
  }

  function closeRoleChange() {
    roleChange.close();
  }

  function submitRoleChange() {
    const target = roleTarget.value;
    if (!target) {
      return;
    }
    const user = props.users.find((candidate) => candidate.public_id === target.id);
    if (!user) {
      return;
    }
    if (roleValue.value === user.role) {
      roleChange.close();
      return;
    }
    roleChange.save();
    router.post(
      accessURL(`/users/${user.public_id}/role${pageQuery.value}`),
      { role: roleValue.value },
      {
        preserveScroll: true,
        onSuccess: () => roleChange.committed(),
        onError: () => roleChange.failed(),
        onFinish: () => roleChange.failed(),
      },
    );
  }

  function roleLabel(code: string): string {
    return roleOptions.value.find((option) => option.value === code)?.label ?? code;
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
    const busy = roleChange.busyId.value === user.public_id;
    return [
      {
        label: "Change role",
        icon: icons.access,
        disabled: busy,
        onSelect: () => requestRoleChange(user),
      },
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
    roleLabel,
    roleTarget,
    roleOpen,
    roleSaving,
    roleValue,
    closeRoleChange,
    submitRoleChange,
    roleMenuItems,
    accessUserMenuItems,
  };
}
