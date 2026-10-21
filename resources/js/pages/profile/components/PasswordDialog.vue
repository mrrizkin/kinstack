<script setup lang="ts">
import { icons } from "@/lib/icons";

import type { PasswordForm, Profile } from "../types";

const open = defineModel<boolean>("open", { required: true });

const props = defineProps<{
  profile: Profile;
  passwordForm: PasswordForm;
  fieldError: (field: string, local?: Record<string, unknown>) => string | undefined;
}>();

const emit = defineEmits<{ submit: [] }>();
</script>

<template>
  <UModal
    v-model:open="open"
    :title="props.profile.invited ? 'Set your password' : 'Change password'"
    :description="
      props.profile.invited
        ? 'Choose a password with at least 8 characters to activate your account.'
        : 'Confirm your current password, then choose a new one. Every other device is signed out.'
    "
    :ui="{ content: 'sm:max-w-md' }">
    <template #body>
      <form id="change-password" class="space-y-4" @submit.prevent="emit('submit')">
        <UFormField
          v-if="!props.profile.invited"
          label="Current password"
          :error="props.fieldError('current_password', props.passwordForm.errors)">
          <UInput
            v-model="props.passwordForm.current_password"
            type="password"
            placeholder="••••••••"
            autocomplete="current-password"
            class="w-full" />
        </UFormField>
        <UFormField
          label="New password"
          hint="At least 8 characters"
          :error="props.fieldError('new_password', props.passwordForm.errors)">
          <UInput
            v-model="props.passwordForm.new_password"
            type="password"
            placeholder="••••••••"
            autocomplete="new-password"
            class="w-full" />
        </UFormField>
        <UFormField
          label="Confirm new password"
          :error="props.fieldError('confirm_password', props.passwordForm.errors)">
          <UInput
            v-model="props.passwordForm.confirm_password"
            type="password"
            placeholder="••••••••"
            autocomplete="new-password"
            class="w-full" />
        </UFormField>
      </form>
    </template>

    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" label="Cancel" @click="open = false" />
        <UButton
          type="submit"
          form="change-password"
          color="primary"
          :icon="icons.lock"
          :label="props.profile.invited ? 'Set password' : 'Change password'"
          :loading="props.passwordForm.processing" />
      </div>
    </template>
  </UModal>
</template>
