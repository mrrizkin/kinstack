<script setup lang="ts">
import { Head, usePage } from "@inertiajs/vue3";
import { computed, ref } from "vue";

import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import { formatDateTime } from "@/lib/formatters";
import { createPasskey, supportsWebAuthn } from "@/lib/webauthn";

interface Passkey {
  id: string;
  name: string;
  created_at: string;
  last_used_at?: string;
}

const props = defineProps<{ passkeys: Passkey[] }>();

const page = usePage();
const passkeys = ref([...props.passkeys]);
const name = ref("This device");
const saving = ref(false);
const revoking = ref("");
const message = ref("");
const error = ref("");
const csrfToken = computed(() => String(page.props.csrf_token ?? ""));
const supported = supportsWebAuthn();

async function addPasskey() {
  if (saving.value || !name.value.trim()) return;
  saving.value = true;
  message.value = "";
  error.value = "";
  try {
    const optionsResponse = await fetch("/passkeys/register/options", {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json", "X-XSRF-TOKEN": csrfToken.value },
    });
    if (!optionsResponse.ok) throw new Error("Unable to start passkey registration.");
    const credential = await createPasskey(await optionsResponse.json());
    credential.name = name.value.trim();
    const response = await fetch("/passkeys/register", {
      method: "POST",
      credentials: "same-origin",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
        "X-XSRF-TOKEN": csrfToken.value,
      },
      body: JSON.stringify(credential),
    });
    if (!response.ok) {
      throw new Error(response.status === 409 ? "This passkey is already registered." : "Passkey could not be saved.");
    }
    passkeys.value.push(await response.json());
    message.value = "Passkey added. You can now sign in with this device.";
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : "Passkey registration was cancelled or failed.";
  } finally {
    saving.value = false;
  }
}

async function revokePasskey(passkey: Passkey) {
  if (revoking.value) return;
  revoking.value = passkey.id;
  error.value = "";
  try {
    const response = await fetch(`/passkeys/${encodeURIComponent(passkey.id)}/delete`, {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json", "X-XSRF-TOKEN": csrfToken.value },
    });
    if (!response.ok) throw new Error("Passkey could not be revoked.");
    passkeys.value = passkeys.value.filter((item) => item.id !== passkey.id);
    message.value = "Passkey revoked.";
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : "Failed to revoke passkey.";
  } finally {
    revoking.value = "";
  }
}
</script>

<template>
  <Head title="Passkeys" />
  <AppLayout>
    <UDashboardPanel id="passkeys">
      <template #header>
        <UDashboardNavbar title="Passkeys">
          <template #leading>
            <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
          </template>
          <span class="text-muted hidden text-sm lg:block">
            Sign in without a password using your fingerprint, face, or device PIN.
          </span>
          <template #right>
            <AppNotificationBell />
          </template>
        </UDashboardNavbar>
      </template>

      <template #body>
        <div class="space-y-6">
          <UAlert v-if="message" color="success" variant="subtle" icon="i-bi-check-circle" :title="message" />
          <UAlert v-if="error" color="error" variant="subtle" icon="i-bi-exclamation-circle" :title="error" />

          <UCard>
            <div v-if="passkeys.length === 0" class="text-muted py-8 text-center text-sm">
              No passkeys registered yet.
            </div>
            <ul v-else class="divide-default divide-y">
              <li
                v-for="passkey in passkeys"
                :key="passkey.id"
                class="flex flex-col gap-2 py-4 sm:flex-row sm:items-center sm:justify-between">
                <div class="min-w-0">
                  <p class="text-highlighted truncate font-medium">{{ passkey.name }}</p>
                  <p class="text-muted mt-1 text-xs">
                    Added {{ formatDateTime(passkey.created_at) }}
                    <span v-if="passkey.last_used_at"> · Last used {{ formatDateTime(passkey.last_used_at) }}</span>
                  </p>
                </div>
                <UButton
                  color="error"
                  variant="outline"
                  size="xs"
                  icon="i-bi-trash"
                  :loading="revoking === passkey.id"
                  @click="revokePasskey(passkey)">
                  Revoke
                </UButton>
              </li>
            </ul>
          </UCard>

          <UCard>
            <div class="space-y-4">
              <UFormField label="Passkey name">
                <UInput v-model="name" maxlength="80" placeholder="e.g. Work laptop" class="w-full" />
              </UFormField>
              <UButton
                color="primary"
                icon="i-bi-fingerprint"
                :loading="saving"
                :disabled="!supported || saving"
                @click="addPasskey">
                Add passkey
              </UButton>
              <p v-if="!supported" class="text-muted text-xs">This browser does not support WebAuthn.</p>
            </div>
          </UCard>
        </div>
      </template>
    </UDashboardPanel>
  </AppLayout>
</template>
