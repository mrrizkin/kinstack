<script setup lang="ts">
import { Head, router, useForm, usePage } from "@inertiajs/vue3";
import type { DropdownMenuItem, TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, h, ref, resolveComponent } from "vue";

import AppMobileList from "@/components/AppMobileList.vue";
import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import { useServerErrors } from "@/lib/errors";
import { formatDateTime } from "@/lib/formatters";
import { PERMISSIONS, usePermissions } from "@/lib/permissions";
import { useTableView } from "@/lib/table-view";

interface User {
  public_id: string;
  username: string;
  email?: string;
  display_name: string;
  role: string;
  is_active: boolean;
  invited: boolean;
  last_login_at?: string;
  created_at: string;
}

interface Role {
  code: string;
  name: string;
  description: string;
  is_system: boolean;
}

const props = defineProps<{ users: User[]; roles: Role[] }>();

const roleOptions = computed(() => props.roles.map((role) => ({ label: role.name, value: role.code })));

const usersURL = (suffix = "") => `/users${suffix}`;

const page = usePage();
const { formError } = useServerErrors();
const { can } = usePermissions();

const showCreate = ref(false);
const createForm = useForm({
  username: "",
  email: "",
  display_name: "",
  password: "",
  role: props.roles[0]?.code ?? "",
});

function submitCreate() {
  createForm.post(usersURL(), {
    onSuccess: () => {
      createForm.reset();
      showCreate.value = false;
    },
  });
}

const showInvite = ref(false);

const headerActions = computed<DropdownMenuItem[]>(() => {
  const actions: DropdownMenuItem[] = [];
  if (can(PERMISSIONS.accessManage)) {
    actions.push({
      label: "Access",
      icon: "i-bi-shield-lock",
      to: "/access",
    });
  }
  if (can(PERMISSIONS.usersManage)) {
    actions.push({
      label: "Invite",
      icon: "i-bi-envelope-plus",
      onSelect: () => {
        showInvite.value = !showInvite.value;
      },
    });
    actions.push({
      label: showCreate.value ? "Close" : "Add user",
      icon: showCreate.value ? "i-bi-x-lg" : "i-bi-person-plus",
      onSelect: () => {
        showCreate.value = !showCreate.value;
      },
    });
  }
  return actions;
});

const inviteForm = useForm({
  email: "",
  username: "",
  display_name: "",
  role: props.roles[0]?.code ?? "",
});

function submitInvite() {
  inviteForm.post(usersURL("/invite"), {
    onSuccess: () => {
      inviteForm.reset();
      showInvite.value = false;
    },
  });
}

function toggleActive(user: User) {
  router.post(usersURL(`/${user.public_id}/active`), { is_active: !user.is_active });
}

function changeRole(user: User, role: string) {
  router.put(usersURL(`/${user.public_id}`), {
    email: user.email ?? "",
    display_name: user.display_name,
    role,
  });
}

function userMenuItems(user: User): DropdownMenuItem[] {
  return [
    {
      label: "Reset password",
      icon: "i-bi-key",
      onSelect: () => resetPassword(user),
    },
    {
      label: user.is_active ? "Deactivate" : "Activate",
      icon: user.is_active ? "i-bi-person-x" : "i-bi-person-check",
      color: user.is_active ? "error" : "success",
      onSelect: () => toggleActive(user),
    },
  ];
}

const passwordTarget = ref<User | null>(null);
const passwordForm = useForm({ password: "" });

function resetPassword(user: User) {
  passwordTarget.value = user;
  passwordForm.reset();
  passwordForm.clearErrors();
}

function confirmResetPassword() {
  const user = passwordTarget.value;
  if (!user) return;
  passwordForm.post(usersURL(`/${user.public_id}/password`), {
    onSuccess: () => {
      passwordTarget.value = null;
      passwordForm.reset();
    },
  });
}

const UBadge = resolveComponent("UBadge");
const USelect = resolveComponent("USelect");
const UButton = resolveComponent("UButton");

const userColumns: TableColumn<User>[] = [
  {
    accessorKey: "display_name",
    header: "User",
    cell: ({ row }) =>
      h("div", undefined, [
        h("div", { class: "text-highlighted font-medium" }, row.original.display_name),
        h("div", { class: "text-muted text-xs" }, `@${row.original.username}`),
        row.original.email ? h("div", { class: "text-muted text-xs" }, row.original.email) : null,
        row.original.invited
          ? h(UBadge, { color: "info", variant: "subtle", size: "sm", class: "mt-1" }, () => "Invited")
          : null,
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
        "onUpdate:modelValue": (value: unknown) => changeRole(row.original, String(value)),
      }),
  },
  {
    accessorKey: "is_active",
    header: "Status",
    cell: ({ row }) =>
      h(
        UBadge,
        {
          color: row.original.is_active ? "success" : "error",
          variant: "subtle",
          size: "sm",
        },
        () => (row.original.is_active ? "Active" : "Inactive"),
      ),
  },
  {
    accessorKey: "last_login_at",
    header: "Last login",
    cell: ({ row }) => formatDateTime(row.original.last_login_at),
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
          variant: "outline",
          label: "Reset password",
          onClick: () => resetPassword(row.original),
        }),
        h(UButton, {
          size: "xs",
          color: row.original.is_active ? "error" : "success",
          variant: "link",
          label: row.original.is_active ? "Deactivate" : "Activate",
          onClick: () => toggleActive(row.original),
        }),
      ]),
  },
];
const view = useTableView("users", userColumns, { paged: false });
</script>

<template>
  <Head title="Users" />
  <AppLayout>
    <UDashboardPanel id="users">
      <template #header>
        <UDashboardNavbar title="Users">
          <template #leading>
            <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
          </template>
          <span class="text-muted hidden text-sm lg:block">Manage user accounts and roles.</span>
          <template #right>
            <AppNotificationBell />
            <div class="hidden items-center gap-2 md:flex">
              <UButton
                v-if="can(PERMISSIONS.accessManage)"
                :to="`/access`"
                color="neutral"
                variant="outline"
                icon="i-bi-shield-lock"
                size="sm"
                label="Access" />
              <UButton
                v-if="can(PERMISSIONS.usersManage)"
                @click="showInvite = !showInvite"
                color="neutral"
                variant="outline"
                icon="i-bi-envelope-plus"
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
              <UButton
                icon="i-material-symbols:more-vert"
                color="neutral"
                variant="outline"
                size="sm"
                square
                aria-label="User actions" />
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
        </UDashboardNavbar>
      </template>

      <template #body>
        <div class="space-y-6">
          <UAlert v-if="formError" color="error" variant="subtle" :title="formError" class="mb-4" />

          <USlideover
            v-model:open="showCreate"
            title="Add user"
            description="Create a new user account."
            :ui="{ content: 'sm:max-w-md' }">
            <template #body>
              <form class="space-y-4" @submit.prevent="submitCreate">
                <UFormField label="Username" :error="createForm.errors.username">
                  <UInput v-model="createForm.username" class="w-full" placeholder="e.g. alex" />
                </UFormField>
                <UFormField label="Email" hint="Optional, for magic links" :error="createForm.errors.email">
                  <UInput v-model="createForm.email" type="email" class="w-full" placeholder="e.g. alex@example.com" />
                </UFormField>
                <UFormField label="Display name" :error="createForm.errors.display_name">
                  <UInput v-model="createForm.display_name" class="w-full" placeholder="e.g. Alex Johnson" />
                </UFormField>
                <UFormField label="Password" hint="At least 8 characters" :error="createForm.errors.password">
                  <UInput v-model="createForm.password" type="password" class="w-full" />
                </UFormField>
                <UFormField label="Role" :error="createForm.errors.role">
                  <USelect v-model="createForm.role" :items="roleOptions" class="w-full" />
                </UFormField>
                <div class="flex justify-end">
                  <UButton type="submit" color="primary" label="Save" :loading="createForm.processing" />
                </div>
              </form>
            </template>
          </USlideover>

          <USlideover
            v-model:open="showInvite"
            title="Invite user"
            description="Create an account and send an invitation link by email."
            :ui="{ content: 'sm:max-w-md' }">
            <template #body>
              <form class="space-y-4" @submit.prevent="submitInvite">
                <UFormField label="Email" hint="Invitation recipient" :error="inviteForm.errors.email">
                  <UInput v-model="inviteForm.email" type="email" class="w-full" placeholder="e.g. alex@example.com" />
                </UFormField>
                <UFormField
                  label="Username"
                  hint="Optional, defaults to the email if empty"
                  :error="inviteForm.errors.username">
                  <UInput v-model="inviteForm.username" class="w-full" placeholder="e.g. alex" />
                </UFormField>
                <UFormField label="Display name" :error="inviteForm.errors.display_name">
                  <UInput v-model="inviteForm.display_name" class="w-full" placeholder="e.g. Alex Johnson" />
                </UFormField>
                <UFormField label="Role" :error="inviteForm.errors.role">
                  <USelect v-model="inviteForm.role" :items="roleOptions" class="w-full" />
                </UFormField>
                <div class="flex justify-end">
                  <UButton type="submit" color="primary" label="Send invitation" :loading="inviteForm.processing" />
                </div>
              </form>
            </template>
          </USlideover>

          <AppTableCard :view="view" class="hidden md:block">
            <UTable :data="props.users" :columns="view.columns" class="w-full">
              <template #empty>
                <UEmpty icon="i-bi-people" title="No users yet." variant="naked" class="py-6" />
              </template>
            </UTable>
          </AppTableCard>

          <div class="md:hidden">
            <AppMobileList v-if="props.users.length > 0">
              <article v-for="user in props.users" :key="user.public_id" class="space-y-3 p-4">
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
                    <UDropdownMenu :items="[userMenuItems(user)]" :content="{ align: 'end', collisionPadding: 12 }">
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
                    <USelect
                      :model-value="user.role"
                      :items="roleOptions"
                      size="sm"
                      class="w-44"
                      @update:model-value="(value: unknown) => changeRole(user, String(value))" />
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
              class="py-10" />
          </div>
        </div>
        <UModal
          :open="Boolean(passwordTarget)"
          title="Reset password"
          :description="passwordTarget ? `New password for ${passwordTarget.username}.` : ''"
          :ui="{ content: 'sm:max-w-md' }"
          @update:open="
            (value: boolean) => {
              if (!value) passwordTarget = null;
            }
          ">
          <template #body>
            <UFormField label="New password" hint="At least 8 characters" :error="passwordForm.errors.password">
              <UInput v-model="passwordForm.password" type="password" placeholder="••••••••" class="w-full" />
            </UFormField>
          </template>

          <template #footer>
            <div class="flex w-full justify-end gap-2">
              <UButton color="neutral" variant="outline" label="Cancel" @click="passwordTarget = null" />
              <UButton
                color="primary"
                label="Save password"
                :loading="passwordForm.processing"
                @click="confirmResetPassword" />
            </div>
          </template>
        </UModal>
      </template>
    </UDashboardPanel>
  </AppLayout>
</template>
