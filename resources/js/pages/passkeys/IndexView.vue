<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { ref } from "vue";
import { z } from "zod";

import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { passkeyOptionsSchema, passkeySummarySchema } from "@/lib/contracts/passkeys";
import { formatDateTime } from "@/lib/formatters";
import { requestJson } from "@/lib/http";
import { usePageProps } from "@/lib/page-props";
import { createPasskey, supportsWebAuthn } from "@/lib/webauthn";

defineOptions({ layout: [BaseLayout, AppLayout] });

interface Passkey {
  id: string;
  name: string;
  created_at: string;
  last_used_at?: string;
}

const props = defineProps<{ passkeys: Passkey[] }>();

const { csrfToken } = usePageProps();
const passkeys = ref([...props.passkeys]);
const name = ref("This device");
const saving = ref(false);
const revoking = ref("");
const message = ref("");
const error = ref("");
const supported = supportsWebAuthn();

async function addPasskey() {
  if (saving.value || !name.value.trim()) return;
  saving.value = true;
  message.value = "";
  error.value = "";
  try {
    const options = await requestJson("/passkeys/register/options", passkeyOptionsSchema, {
      method: "POST",
      csrfToken: csrfToken.value,
    });
    const credential = await createPasskey(options);
    credential.name = name.value.trim();
    const saved = await requestJson("/passkeys/register", passkeySummarySchema, {
      method: "POST",
      csrfToken: csrfToken.value,
      body: credential,
    });
    passkeys.value.push(saved);
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
    await requestJson(`/passkeys/${encodeURIComponent(passkey.id)}/delete`, z.unknown(), {
      method: "POST",
      csrfToken: csrfToken.value,
    });
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
</template>
