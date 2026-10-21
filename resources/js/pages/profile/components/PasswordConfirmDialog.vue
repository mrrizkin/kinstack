<script setup lang="ts">
import AppConfirmDialog from "@/components/AppConfirmDialog.vue";

import type { TwoFactorPasswordForm } from "../types";

const open = defineModel<boolean>("open", { required: true });

const props = withDefaults(
  defineProps<{
    title: string;
    description: string;
    confirmLabel: string;
    color?: "error" | "primary" | "warning";
    passwordForm: TwoFactorPasswordForm;
    fieldError: (field: string, local?: Record<string, unknown>) => string | undefined;
  }>(),
  { color: "primary" },
);

const emit = defineEmits<{ confirm: [] }>();
</script>

<template>
  <AppConfirmDialog
    :open="open"
    :title="props.title"
    :description="props.description"
    :confirm-label="props.confirmLabel"
    :color="props.color"
    :loading="props.passwordForm.processing"
    @update:open="(value: boolean) => (open = value)"
    @confirm="emit('confirm')">
    <form id="confirm-security-action" class="space-y-3" @submit.prevent="emit('confirm')">
      <UFormField
        label="Account password"
        description="Verified before the change is applied."
        :error="props.fieldError('two_factor_password', props.passwordForm.errors)">
        <UInput
          v-model="props.passwordForm.password"
          type="password"
          placeholder="••••••••"
          autocomplete="current-password"
          class="w-full" />
      </UFormField>
    </form>
  </AppConfirmDialog>
</template>
