<script setup lang="ts">
import { Head, router, useForm } from "@inertiajs/vue3";
import { computed, ref } from "vue";

import AppConfirmDialog from "@/components/AppConfirmDialog.vue";
import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { formatDateTime } from "@/lib/formatters";
import { usePageProps } from "@/lib/page-props";
import { permissionLabel } from "@/lib/permissions";

defineOptions({ layout: [BaseLayout, AppLayout] });

interface APIToken {
  public_id: string;
  name: string;
  prefix: string;
  scopes: string[];
  expires_at?: string;
  last_used_at?: string;
  created_at: string;
}

const props = defineProps<{ tokens: APIToken[]; created_token?: string }>();

const { permissions } = usePageProps();

const showCreate = ref(false);
const copied = ref(false);
const copyError = ref("");
const revokeTarget = ref<APIToken | null>(null);
const revoking = ref(false);
const createForm = useForm({ name: "", expires_in_days: 0, scopes: [] as string[] });

function toggleScope(permission: string, checked: boolean) {
  if (checked) {
    if (!createForm.scopes.includes(permission)) createForm.scopes.push(permission);
    return;
  }
  createForm.scopes = createForm.scopes.filter((scope) => scope !== permission);
}

const expiryOptions = [
  { label: "No expiration", value: 0 },
  { label: "30 days", value: 30 },
  { label: "90 days", value: 90 },
  { label: "1 year", value: 365 },
];

const hasCreated = computed(() => (props.created_token ?? "").length > 0);

function submitCreate() {
  createForm.post("/api-tokens", {
    onSuccess: () => {
      createForm.reset();
      showCreate.value = false;
    },
  });
}

function revoke(token: APIToken) {
  revokeTarget.value = token;
}

function confirmRevoke() {
  const token = revokeTarget.value;
  if (!token || revoking.value) return;
  router.post(`/api-tokens/${token.public_id}/revoke`, undefined, {
    preserveScroll: true,
    onStart: () => (revoking.value = true),
    onFinish: () => {
      revoking.value = false;
      revokeTarget.value = null;
    },
  });
}

async function copyToken() {
  if (!props.created_token) return;
  copyError.value = "";
  try {
    await navigator.clipboard.writeText(props.created_token);
    copied.value = true;
    setTimeout(() => (copied.value = false), 2000);
  } catch {
    copied.value = false;
    copyError.value = "Could not copy automatically. Select the token and copy it manually.";
  }
}
</script>

<template>
  <Head title="API tokens" />
  <UDashboardPanel id="api-tokens">
    <template #header>
      <UDashboardNavbar title="API tokens">
        <template #leading>
          <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
        </template>
        <span class="text-muted hidden text-sm lg:block">
          Tokens for connecting integrations without a browser session.
        </span>
        <template #right>
          <AppNotificationBell />
          <UButton color="primary" icon="i-bi-plus" @click="showCreate = !showCreate"> New token </UButton>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="space-y-6">
        <UAlert v-if="hasCreated" color="success" variant="subtle" icon="i-bi-key" title="Token created">
          <template #description>
            <p class="mb-2">Copy it now. This token is only shown once.</p>
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <UInput :model-value="props.created_token" readonly class="w-full font-mono text-xs" />
                <UButton
                  :icon="copied ? 'i-bi-check' : 'i-bi-clipboard'"
                  color="neutral"
                  variant="outline"
                  @click="copyToken">
                  {{ copied ? "Copied" : "Copy" }}
                </UButton>
              </div>
              <p v-if="copyError" class="text-error text-xs">{{ copyError }}</p>
            </div>
          </template>
        </UAlert>

        <UCard>
          <div v-if="props.tokens.length === 0" class="flex flex-col items-center gap-3 py-10 text-center">
            <UIcon name="i-material-symbols:key-outline-rounded" class="text-dimmed size-8" />
            <div>
              <p class="text-highlighted text-sm font-medium">No API tokens yet.</p>
              <p class="text-muted mt-1 text-xs">
                Create a token for integrations that need to access the API without a browser session.
              </p>
            </div>
            <UButton color="primary" size="sm" icon="i-material-symbols:add-rounded" @click="showCreate = true">
              New token
            </UButton>
          </div>
          <ul v-else class="divide-default divide-y">
            <li
              v-for="token in props.tokens"
              :key="token.public_id"
              class="flex flex-col gap-2 py-4 sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <p class="text-highlighted truncate font-medium">{{ token.name }}</p>
                <p class="text-muted font-mono text-xs">{{ token.prefix }}…</p>
                <p class="text-muted mt-1 text-xs">
                  Created {{ formatDateTime(token.created_at) }}
                  <span v-if="token.last_used_at"> · Last used {{ formatDateTime(token.last_used_at) }}</span>
                  <span v-else> · Never used</span>
                  <span v-if="token.expires_at"> · Expires {{ formatDateTime(token.expires_at) }}</span>
                </p>
                <p v-if="token.scopes?.length" class="text-muted mt-1 text-xs">
                  Permissions: {{ token.scopes.map(permissionLabel).join(", ") }}
                </p>
                <p v-else class="text-muted mt-1 text-xs">No permissions · no access</p>
              </div>
              <UButton
                color="error"
                variant="outline"
                size="xs"
                icon="i-bi-trash"
                :loading="revoking && revokeTarget?.public_id === token.public_id"
                @click="revoke(token)">
                Revoke
              </UButton>
            </li>
          </ul>
        </UCard>
      </div>

      <USlideover
        v-model:open="showCreate"
        title="New token"
        description="Give it a name so it is easy to recognize."
        :ui="{ content: 'sm:max-w-md' }">
        <template #body>
          <form class="space-y-4" @submit.prevent="submitCreate">
            <UFormField label="Name" :error="createForm.errors.name">
              <UInput v-model="createForm.name" class="w-full" placeholder="e.g. CI pipeline" />
            </UFormField>
            <UFormField label="Expiry" :error="createForm.errors.expires_in_days">
              <USelect v-model="createForm.expires_in_days" :items="expiryOptions" class="w-full" />
            </UFormField>
            <UFormField label="Permissions" :error="createForm.errors.scopes">
              <div class="space-y-2">
                <UCheckbox
                  v-for="permission in permissions"
                  :key="permission"
                  :model-value="createForm.scopes.includes(permission)"
                  :label="permissionLabel(permission)"
                  @update:model-value="
                    (checked: boolean | 'indeterminate') => toggleScope(permission, checked === true)
                  " />
                <p v-if="permissions.length === 0" class="text-muted text-xs">
                  You do not have any permissions yet, so this token will not be able to access anything.
                </p>
              </div>
            </UFormField>
            <p class="text-muted text-xs">
              The token can only use permissions you already have. Without selected permissions, the token has no
              access.
            </p>
            <div class="flex justify-end gap-2">
              <UButton color="neutral" variant="ghost" @click="showCreate = false">Cancel</UButton>
              <UButton type="submit" color="primary" :loading="createForm.processing">Create token</UButton>
            </div>
          </form>
        </template>
      </USlideover>
    </template>
  </UDashboardPanel>

  <AppConfirmDialog
    :open="revokeTarget !== null"
    title="Revoke API token"
    :description="revokeTarget ? `${revokeTarget.name} will stop working immediately. This cannot be undone.` : ''"
    confirm-label="Revoke token"
    :loading="revoking"
    @update:open="
      (value: boolean) => {
        if (!value) revokeTarget = null;
      }
    "
    @confirm="confirmRevoke" />
</template>
