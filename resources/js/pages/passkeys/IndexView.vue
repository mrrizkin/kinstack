<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { z } from "zod";

import AppConfirmDialog from "@/components/AppConfirmDialog.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import SettingsLayout from "@/layouts/SettingsLayout.vue";
import { passkeyOptionsSchema, passkeySummarySchema } from "@/lib/contracts/passkeys";
import { formatDateTime } from "@/lib/formatters";
import { requestJson } from "@/lib/http";
import { icons } from "@/lib/icons";
import { useNotify } from "@/lib/notify";
import { usePageProps } from "@/lib/page-props";
import { createPasskey, supportsWebAuthn } from "@/lib/webauthn";

defineOptions({ layout: [BaseLayout, AppLayout, SettingsLayout] });

interface Passkey {
  id: string;
  name: string;
  created_at: string;
  last_used_at?: string;
}

const props = defineProps<{ passkeys: Passkey[] }>();

const { csrfToken } = usePageProps();
const { t } = useI18n();
const notify = useNotify();
const passkeys = ref([...props.passkeys]);
const name = ref("This device");
const nameError = ref("");
const saving = ref(false);
const revoking = ref("");
const revokeTarget = ref<Passkey | null>(null);
const addDialogOpen = ref(false);
const supported = supportsWebAuthn();

// A dismissed WebAuthn prompt is a decision, not a failure.
function isCancellation(caught: unknown): boolean {
  return caught instanceof DOMException && (caught.name === "NotAllowedError" || caught.name === "AbortError");
}

function failureMessage(caught: unknown, fallback: string): string {
  return caught instanceof Error && caught.message ? caught.message : fallback;
}

function openAddDialog() {
  if (!supported) return;
  nameError.value = "";
  addDialogOpen.value = true;
}

async function confirmAddPasskey() {
  if (saving.value) return;
  nameError.value = "";
  if (!name.value.trim()) {
    nameError.value = "Enter a name so you can recognize this passkey later.";
    return;
  }
  saving.value = true;
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
    notify("success", "Passkey added. You can now sign in with this device.");
    name.value = "This device";
    addDialogOpen.value = false;
  } catch (caught) {
    if (isCancellation(caught)) {
      notify("info", "Passkey setup was cancelled.");
    } else {
      notify("error", failureMessage(caught, "Passkey registration failed."));
    }
  } finally {
    saving.value = false;
  }
}

function revokePasskey(passkey: Passkey) {
  if (revoking.value) return;
  revokeTarget.value = passkey;
}

async function confirmRevokePasskey() {
  const passkey = revokeTarget.value;
  if (!passkey || revoking.value) return;
  revoking.value = passkey.id;
  try {
    await requestJson(`/passkeys/${encodeURIComponent(passkey.id)}/delete`, z.unknown(), {
      method: "POST",
      csrfToken: csrfToken.value,
    });
    passkeys.value = passkeys.value.filter((item) => item.id !== passkey.id);
    notify("success", "Passkey revoked.");
  } catch (caught) {
    notify("error", failureMessage(caught, "Failed to revoke passkey."));
  } finally {
    revoking.value = "";
    revokeTarget.value = null;
  }
}

const emptyActions = computed(() =>
  supported
    ? [{ label: "Add your first passkey", icon: icons.passkey, onClick: openAddDialog }]
    : [{ label: "Back to profile", icon: icons.arrowLeft, to: "/profile" }],
);
</script>

<template>
  <Head :title="t('shell.settings.sections.passkeys.title')" />

  <div class="space-y-6">
    <div class="flex justify-end">
      <UButton color="primary" :icon="icons.add" label="Add passkey" :disabled="!supported" @click="openAddDialog" />
    </div>

    <UCard>
      <UEmpty
        v-if="passkeys.length === 0"
        :icon="icons.passkey"
        title="No passkeys yet."
        description="Add a passkey to sign in with your fingerprint, face, or device PIN."
        variant="naked"
        :actions="emptyActions"
        class="py-8" />
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
            :icon="icons.delete"
            :loading="revoking === passkey.id"
            @click="revokePasskey(passkey)">
            Revoke
          </UButton>
        </li>
      </ul>
    </UCard>

    <UAlert
      v-if="!supported"
      color="warning"
      :icon="icons.warning"
      title="Passkeys are unavailable in this browser"
      description="Use a browser that supports WebAuthn to add a passkey to this account." />
  </div>

  <UModal
    v-model:open="addDialogOpen"
    title="Add a passkey"
    description="Name this passkey so you can recognize it later. Your browser then asks for your fingerprint, face, or device PIN."
    :ui="{ content: 'sm:max-w-md' }">
    <template #body>
      <form id="add-passkey" class="space-y-4" @submit.prevent="confirmAddPasskey">
        <UFormField label="Passkey name" hint="Up to 80 characters" :error="nameError || undefined">
          <UInput v-model="name" maxlength="80" placeholder="e.g. Work laptop" class="w-full" />
        </UFormField>
        <p class="text-muted text-xs">The passkey is created on this device only and never leaves it.</p>
      </form>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" label="Cancel" @click="addDialogOpen = false" />
        <UButton
          type="submit"
          form="add-passkey"
          color="primary"
          :icon="icons.passkey"
          label="Continue"
          :loading="saving"
          :disabled="saving" />
      </div>
    </template>
  </UModal>

  <AppConfirmDialog
    :open="revokeTarget !== null"
    title="Revoke passkey"
    :description="revokeTarget ? `${revokeTarget.name} can no longer be used to sign in.` : ''"
    confirm-label="Revoke passkey"
    :loading="Boolean(revoking)"
    @update:open="
      (value: boolean) => {
        if (!value) revokeTarget = null;
      }
    "
    @confirm="confirmRevokePasskey" />
</template>
