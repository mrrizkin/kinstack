<script setup lang="ts">
import AppConfirmDialog from "@/components/AppConfirmDialog.vue";

import type { CreateUserForm, InviteUserForm, PasswordForm, User } from "../columns";

const showCreate = defineModel<boolean>("showCreate", { required: true });
const showInvite = defineModel<boolean>("showInvite", { required: true });
const passwordTarget = defineModel<User | null>("passwordTarget", { required: true });
const pendingToggle = defineModel<User | null>("pendingToggle", { required: true });

defineProps<{
  roleOptions: { label: string; value: string }[];
  createForm: CreateUserForm;
  inviteForm: InviteUserForm;
  passwordForm: PasswordForm;
  togglingActive: boolean;
  submitCreate: () => void;
  submitInvite: () => void;
  confirmResetPassword: () => void;
  confirmToggleActive: () => void;
}>();
</script>

<template>
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

  <UModal
    :open="passwordTarget !== null"
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

  <AppConfirmDialog
    :open="pendingToggle !== null"
    title="Deactivate user"
    :description="
      pendingToggle ? `${pendingToggle.display_name} will be signed out and cannot sign in until reactivated.` : ''
    "
    confirm-label="Deactivate user"
    :loading="togglingActive"
    @update:open="
      (value: boolean) => {
        if (!value) pendingToggle = null;
      }
    "
    @confirm="confirmToggleActive" />
</template>
