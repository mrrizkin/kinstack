<script setup lang="ts">
import { Head, router, useForm } from "@inertiajs/vue3";
import type { DropdownMenuItem, TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, h, reactive, ref, resolveComponent } from "vue";

import AppInputDate from "@/components/AppInputDate.vue";
import AppMobileList from "@/components/AppMobileList.vue";
import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { useServerErrors } from "@/lib/errors";
import { useTableView } from "@/lib/table-view";

defineOptions({ layout: [BaseLayout, AppLayout] });

interface Role {
  code: string;
  name: string;
  description: string;
  rank: number;
  is_system: boolean;
  permissions: string[];
}

interface Permission {
  code: string;
  category: string;
  label: string;
  description: string;
}

interface Override {
  permission: string;
  label: string;
  effect: "allow" | "deny";
  reason: string;
  expires_at?: string;
  expired: boolean;
}

interface UserAccess {
  public_id: string;
  username: string;
  display_name: string;
  role: string;
  role_name: string;
  is_active: boolean;
  overrides: Override[];
}

const props = defineProps<{
  roles: Role[];
  permissions: Permission[];
  categories: string[];
  users: UserAccess[];
}>();

const accessURL = (suffix = "") => `/access${suffix}`;

const { formError } = useServerErrors();
const tab = ref("roles");

// ---- Role permission matrix ----
const editingRole = ref<Role | null>(null);
const selectedPermissions = ref<string[]>([]);

const permissionsByCategory = computed(() =>
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
const overrideForm = reactive({
  permission: "",
  effect: "allow" as "allow" | "deny",
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
  overrideForm.permission = permissionOptions.value[0]?.value ?? "";
  overrideForm.effect = "allow";
  overrideForm.reason = "";
  overrideForm.expires = "";
}

function submitOverride() {
  const user = editingUser.value;
  if (!user) return;
  overrideErrors.value = "";
  const expiry = overrideForm.expires;
  if (!overrideForm.permission) {
    overrideErrors.value = "Select a permission first.";
    return;
  }
  router.post(
    accessURL(`/users/${user.public_id}/permissions`),
    {
      permission: overrideForm.permission,
      effect: overrideForm.effect,
      reason: overrideForm.reason,
      expires_at: expiry,
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

// ---- Tables ----
const UBadge = resolveComponent("UBadge");
const UButton = resolveComponent("UButton");
const USelect = resolveComponent("USelect");

const roleColumns: TableColumn<Role>[] = [
  {
    accessorKey: "name",
    header: "Role",
    cell: ({ row }) =>
      h("div", undefined, [
        h("div", { class: "text-highlighted flex items-center gap-2 font-medium" }, [
          row.original.name,
          row.original.is_system
            ? h(UBadge, { color: "neutral", variant: "subtle", size: "sm" }, () => "Built-in")
            : null,
        ]),
        h("div", { class: "text-muted text-xs" }, `${row.original.code} · ${row.original.description || "—"}`),
      ]),
  },
  {
    accessorKey: "rank",
    header: "Priority",
    cell: ({ row }) => String(row.original.rank),
  },
  {
    id: "permissions",
    header: "Permissions",
    cell: ({ row }) => `${row.original.permissions.length} permissions`,
  },
  {
    id: "actions",
    header: "Actions",
    meta: { class: { th: "text-right", td: "text-right" } },
    cell: ({ row }) =>
      h("div", { class: "flex justify-end gap-1" }, [
        h(UButton, {
          size: "xs",
          color: "neutral",
          variant: "ghost",
          label: "Edit",
          onClick: () => openEditRole(row.original),
        }),
        h(UButton, {
          size: "xs",
          color: "neutral",
          variant: "outline",
          label: "Manage permissions",
          onClick: () => openRolePermissions(row.original),
        }),
        !row.original.is_system
          ? h(UButton, {
              size: "xs",
              color: "error",
              variant: "link",
              label: "Delete",
              onClick: () => deleteRole(row.original),
            })
          : null,
      ]),
  },
];

const userColumns: TableColumn<UserAccess>[] = [
  {
    accessorKey: "display_name",
    header: "User",
    cell: ({ row }) =>
      h("div", undefined, [
        h("div", { class: "text-highlighted font-medium" }, row.original.display_name),
        h("div", { class: "text-muted text-xs" }, `@${row.original.username}`),
      ]),
  },
  {
    accessorKey: "role",
    header: "Role",
    cell: ({ row }) =>
      h(USelect, {
        modelValue: row.original.role,
        items: roleOptions.value,
        size: "xs",
        class: "w-40",
        "onUpdate:modelValue": (value: unknown) => assignRole(row.original, String(value)),
      }),
  },
  {
    id: "overrides",
    header: "Custom permissions",
    cell: ({ row }) =>
      row.original.overrides.length === 0
        ? h("span", { class: "text-muted text-xs" }, "None")
        : h(
            "div",
            { class: "flex flex-wrap gap-1" },
            row.original.overrides.map((override) =>
              h(
                UBadge,
                {
                  color: override.expired ? "neutral" : override.effect === "allow" ? "success" : "error",
                  variant: "subtle",
                  size: "sm",
                },
                () => `${override.effect === "allow" ? "+" : "−"} ${override.label}`,
              ),
            ),
          ),
  },
  {
    id: "actions",
    header: "Actions",
    meta: { class: { th: "text-right", td: "text-right" } },
    cell: ({ row }) =>
      h(UButton, {
        size: "xs",
        color: "neutral",
        variant: "outline",
        label: "Manage permissions",
        onClick: () => openUserOverrides(row.original),
      }),
  },
];
const roleView = useTableView("access.roles", roleColumns, { paged: false });
const userView = useTableView("access.users", userColumns, { paged: false });
</script>

<template>
  <Head title="Access" />
  <UDashboardPanel id="access">
    <template #header>
      <UDashboardNavbar title="Access" description="Manage roles, permissions, and per-user overrides">
        <template #leading>
          <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
        </template>
        <template #right>
          <AppNotificationBell />
          <UButton icon="i-bi-plus" label="New role" color="primary" size="sm" @click="showCreateRole = true" />
        </template>
      </UDashboardNavbar>
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
          <div v-if="item.value === 'roles'" class="space-y-3">
            <AppTableCard :view="roleView" class="hidden md:block">
              <UTable :data="roles" :columns="roleView.columns" class="w-full" />
            </AppTableCard>

            <div class="md:hidden">
              <AppMobileList v-if="roles.length > 0">
                <article v-for="role in roles" :key="role.code" class="space-y-3 p-4">
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0">
                      <div class="flex flex-wrap items-center gap-2">
                        <p class="text-highlighted font-medium">{{ role.name }}</p>
                        <UBadge v-if="role.is_system" color="neutral" variant="subtle" size="sm">Built-in</UBadge>
                      </div>
                      <p class="text-muted mt-1 font-mono text-xs">{{ role.code }}</p>
                      <p class="text-muted mt-1 text-xs">{{ role.description || "—" }}</p>
                    </div>
                    <UDropdownMenu :items="[roleMenuItems(role)]" :content="{ align: 'end', collisionPadding: 12 }">
                      <UButton
                        icon="i-material-symbols:more-vert"
                        color="neutral"
                        variant="ghost"
                        size="xs"
                        square
                        :aria-label="`Actions for role ${role.name}`" />
                    </UDropdownMenu>
                  </div>
                  <div class="grid grid-cols-2 gap-2 text-xs">
                    <div class="flex items-center justify-between gap-2">
                      <span class="text-muted">Priority</span>
                      <span class="text-toned">{{ role.rank }}</span>
                    </div>
                    <div class="flex items-center justify-between gap-2">
                      <span class="text-muted">Permissions</span>
                      <span class="text-toned">{{ role.permissions.length }} permissions</span>
                    </div>
                  </div>
                </article>
              </AppMobileList>
              <UEmpty v-else icon="i-bi-shield-lock" title="No roles yet." variant="naked" class="py-10" />
            </div>
          </div>

          <div v-else class="space-y-4">
            <UAlert
              color="neutral"
              variant="subtle"
              icon="i-bi-info-circle"
              description="Custom permissions add (allow) or remove (deny) a single permission for a specific user. Deny always takes precedence over roles." />
            <AppTableCard :view="userView" class="hidden md:block">
              <UTable :data="users" :columns="userView.columns" class="w-full" />
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
                      <UDropdownMenu
                        :items="[accessUserMenuItems(user)]"
                        :content="{ align: 'end', collisionPadding: 12 }">
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
                    <USelect
                      :model-value="user.role"
                      :items="roleOptions"
                      size="sm"
                      class="w-44"
                      @update:model-value="(value: unknown) => assignRole(user, String(value))" />
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
              <UEmpty v-else icon="i-bi-people" title="No users yet." variant="naked" class="py-10" />
            </div>
          </div>
        </template>
      </UTabs>
    </template>
  </UDashboardPanel>

  <!-- Create role -->
  <UModal v-model:open="showCreateRole" title="New role" description="A custom role with configurable permissions.">
    <template #body>
      <div class="space-y-4">
        <UFormField label="Code" :error="createRoleForm.errors.code">
          <UInput v-model="createRoleForm.code" placeholder="e.g. supervisor" class="w-full" />
        </UFormField>
        <UFormField label="Name" :error="createRoleForm.errors.name">
          <UInput v-model="createRoleForm.name" placeholder="e.g. Supervisor" class="w-full" />
        </UFormField>
        <UFormField label="Description" :error="createRoleForm.errors.description">
          <UInput v-model="createRoleForm.description" class="w-full" />
        </UFormField>
        <UFormField label="Priority" :error="createRoleForm.errors.rank">
          <UInput v-model.number="createRoleForm.rank" type="number" min="0" max="1000" class="w-full" />
        </UFormField>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton color="neutral" variant="ghost" label="Cancel" @click="showCreateRole = false" />
        <UButton label="Save" :loading="createRoleForm.processing" @click="submitCreateRole" />
      </div>
    </template>
  </UModal>

  <!-- Edit role metadata -->
  <UModal
    :open="editingRoleMeta !== null"
    title="Edit role"
    description="Role name, description, and priority. The role code is fixed."
    @update:open="(value: boolean) => (!value ? (editingRoleMeta = null) : null)">
    <template #body>
      <div class="space-y-4">
        <UFormField label="Code (fixed)">
          <UInput :model-value="editingRoleMeta?.code ?? ''" disabled class="w-full" />
        </UFormField>
        <UFormField label="Name" :error="editRoleForm.errors.name">
          <UInput v-model="editRoleForm.name" class="w-full" />
        </UFormField>
        <UFormField label="Description" :error="editRoleForm.errors.description">
          <UInput v-model="editRoleForm.description" class="w-full" />
        </UFormField>
        <UFormField label="Priority" :error="editRoleForm.errors.rank">
          <UInput v-model.number="editRoleForm.rank" type="number" min="0" max="1000" class="w-full" />
        </UFormField>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton color="neutral" variant="ghost" label="Cancel" @click="editingRoleMeta = null" />
        <UButton label="Save" :loading="editRoleForm.processing" @click="submitEditRole" />
      </div>
    </template>
  </UModal>

  <!-- Role permissions -->
  <USlideover
    :open="editingRole !== null"
    :title="editingRole ? `Role permissions · ${editingRole.name}` : ''"
    @update:open="(value: boolean) => (!value ? (editingRole = null) : null)">
    <template #body>
      <div class="space-y-5">
        <div v-for="group in permissionsByCategory" :key="group.category" class="space-y-2">
          <p class="text-muted text-xs font-semibold tracking-wide uppercase">{{ group.category }}</p>
          <UCheckbox
            v-for="permission in group.items"
            :key="permission.code"
            :model-value="selectedPermissions.includes(permission.code)"
            :label="permission.label"
            :description="permission.code"
            @update:model-value="
              (value: boolean | 'indeterminate') => togglePermission(permission.code, value === true)
            " />
        </div>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton color="neutral" variant="ghost" label="Cancel" @click="editingRole = null" />
        <UButton label="Save permissions" @click="saveRolePermissions" />
      </div>
    </template>
  </USlideover>

  <!-- User overrides -->
  <USlideover
    :open="editingUser !== null"
    :title="editingUser ? `Custom permissions · ${editingUser.display_name}` : ''"
    @update:open="(value: boolean) => (!value ? (editingUserId = null) : null)">
    <template #body>
      <div v-if="editingUser" class="space-y-5">
        <div class="space-y-2">
          <p class="text-muted text-xs font-semibold tracking-wide uppercase">Active permissions</p>
          <UEmpty
            v-if="editingUser.overrides.length === 0"
            icon="i-bi-slash-circle"
            title="No custom permissions yet"
            description="This user only uses permissions from their role." />
          <div v-else class="space-y-2">
            <div
              v-for="override in editingUser.overrides"
              :key="override.permission"
              class="border-default flex items-start justify-between gap-3 rounded border p-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <UBadge
                    :color="override.expired ? 'neutral' : override.effect === 'allow' ? 'success' : 'error'"
                    variant="subtle"
                    size="sm">
                    {{ override.effect === "allow" ? "Allow" : "Deny" }}
                  </UBadge>
                  <span class="text-highlighted truncate text-sm font-medium">{{ override.label }}</span>
                </div>
                <p class="text-muted mt-1 text-xs">
                  {{ override.reason || "No reason" }}
                  <span v-if="override.expired"> · expired</span>
                  <span v-else-if="override.expires_at"> · until {{ override.expires_at.slice(0, 10) }}</span>
                </p>
              </div>
              <UButton
                size="xs"
                color="error"
                variant="link"
                label="Delete"
                @click="removeOverride(editingUser!, override)" />
            </div>
          </div>
        </div>

        <USeparator />

        <div class="space-y-4">
          <p class="text-muted text-xs font-semibold tracking-wide uppercase">Add / edit permission</p>
          <UAlert v-if="overrideErrors" color="error" variant="subtle" :description="overrideErrors" />
          <UFormField label="Permission">
            <USelect v-model="overrideForm.permission" :items="permissionOptions" class="w-full" />
          </UFormField>
          <UFormField label="Effect">
            <USelect
              v-model="overrideForm.effect"
              :items="[
                { label: 'Allow — add permission', value: 'allow' },
                { label: 'Deny — remove permission', value: 'deny' },
              ]"
              class="w-full" />
          </UFormField>
          <UFormField label="Reason">
            <UInput v-model="overrideForm.reason" placeholder="e.g. covering the night shift" class="w-full" />
          </UFormField>
          <UFormField label="Valid until (optional)">
            <AppInputDate v-model="overrideForm.expires" placeholder="2026-01-01" class="w-full" />
          </UFormField>
          <UButton label="Save custom permission" block @click="submitOverride" />
        </div>
      </div>
    </template>
  </USlideover>
</template>
