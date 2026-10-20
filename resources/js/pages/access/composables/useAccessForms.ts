import { router, useForm } from "@inertiajs/vue3";
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, ref } from "vue";

import type { Override, OverrideForm, Permission, PermissionCategory, Role, UserAccess } from "../types";

interface AccessProps {
  roles: Role[];
  permissions: Permission[];
  categories: string[];
  users: UserAccess[];
}

export function useAccessForms(props: AccessProps) {
  const accessURL = (suffix = "") => `/access${suffix}`;
  const tab = ref("roles");

  // ---- Role permission matrix ----
  const editingRole = ref<Role | null>(null);
  const selectedPermissions = ref<string[]>([]);

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
    if (!role) return;
    router.post(
      accessURL(`/roles/${role.code}/permissions`),
      { permissions: selectedPermissions.value },
      { preserveScroll: true, onSuccess: () => (editingRole.value = null) },
    );
  }

  // ---- Create role ----
  const showCreateRole = ref(false);
  const createRoleForm = useForm({ code: "", name: "", description: "", rank: 20 });

  function submitCreateRole() {
    createRoleForm.post(accessURL("/roles"), {
      preserveScroll: true,
      onSuccess: () => {
        createRoleForm.reset();
        showCreateRole.value = false;
      },
    });
  }

  function deleteRole(role: Role) {
    router.post(accessURL(`/roles/${role.code}/delete`), {}, { preserveScroll: true });
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
    editRoleForm.put(accessURL(`/roles/${role.code}`), {
      preserveScroll: true,
      onSuccess: () => (editingRoleMeta.value = null),
    });
  }

  // ---- User overrides ----
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
  const overrideErrors = ref("");

  const permissionOptions = computed(() =>
    props.permissions.map((permission) => ({
      label: `${permission.label} · ${permission.code}`,
      value: permission.code,
    })),
  );

  const roleOptions = computed(() => props.roles.map((role) => ({ label: role.name, value: role.code })));

  function openUserOverrides(user: UserAccess) {
    editingUserId.value = user.public_id;
    overrideErrors.value = "";
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
    if (!user) return;
    overrideErrors.value = "";
    if (!overrideForm.value.permission) {
      overrideErrors.value = "Select a permission first.";
      return;
    }
    router.post(
      accessURL(`/users/${user.public_id}/permissions`),
      {
        permission: overrideForm.value.permission,
        effect: overrideForm.value.effect,
        reason: overrideForm.value.reason,
        expires_at: overrideForm.value.expires,
      },
      { preserveScroll: true },
    );
  }

  function removeOverride(user: UserAccess, override: Override) {
    router.post(
      accessURL(`/users/${user.public_id}/permissions/delete`),
      { permission: override.permission },
      { preserveScroll: true },
    );
  }

  function assignRole(user: UserAccess, role: string) {
    router.post(accessURL(`/users/${user.public_id}/role`), { role }, { preserveScroll: true });
  }

  function roleMenuItems(role: Role): DropdownMenuItem[] {
    const actions: DropdownMenuItem[] = [
      {
        label: "Edit",
        icon: "i-bi-pencil-square",
        onSelect: () => openEditRole(role),
      },
      {
        label: "Manage permissions",
        icon: "i-bi-sliders",
        onSelect: () => openRolePermissions(role),
      },
    ];
    if (!role.is_system) {
      actions.push({
        label: "Delete",
        icon: "i-bi-trash",
        color: "error",
        onSelect: () => deleteRole(role),
      });
    }
    return actions;
  }

  function accessUserMenuItems(user: UserAccess): DropdownMenuItem[] {
    return [
      {
        label: "Manage permissions",
        icon: "i-bi-sliders",
        onSelect: () => openUserOverrides(user),
      },
    ];
  }

  return {
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
  };
}
