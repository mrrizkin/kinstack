<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";
import type { AuthFormField, FormSubmitEvent } from "@nuxt/ui";
import { computed, ref } from "vue";

import AuthLayout from "@/layouts/AuthLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { icons } from "@/lib/icons";

defineOptions({
  layout: [
    BaseLayout,
    [
      AuthLayout,
      {
        title: "Two-step verification",
        subtitle: "Enter the 6-digit code from your authenticator app, or one of your recovery codes.",
      },
    ],
  ],
});

const form = useForm({
  code: "",
});

// Authenticator codes are six digits, so they get one box per digit. Recovery
// codes are longer and alphanumeric, so they fall back to a regular input.
const useRecoveryCode = ref(false);

const fields = computed<AuthFormField[]>(() => [
  {
    name: "code",
    type: "otp",
    label: "Verification code",
    length: 6,
    otp: true,
    separator: 3,
    size: "lg",
    autofocus: true,
    required: true,
    error: form.errors.code,
  },
]);

function submit(payload: FormSubmitEvent<Record<string, unknown>>) {
  const value = payload.data.code;
  form.code = Array.isArray(value) ? value.join("") : String(value ?? "");
  form.post("/2fa", {
    onFinish: () => form.reset("code"),
  });
}

function toggleRecoveryCode() {
  useRecoveryCode.value = !useRecoveryCode.value;
  form.reset("code");
  form.clearErrors();
}

function submitRecoveryCode() {
  if (!form.code.trim() || form.processing) return;
  form.post("/2fa", {
    onFinish: () => form.reset("code"),
  });
}

function cancel() {
  form.post("/2fa/cancel");
}
</script>

<template>
  <Head title="Two-step verification" />
  <div class="space-y-4">
    <UAuthForm
      v-if="!useRecoveryCode"
      :fields="fields"
      :submit="{
        label: form.processing ? 'Verifying…' : 'Verify',
        color: 'primary',
        block: true,
        size: 'lg',
        loading: form.processing,
      }"
      @submit="submit" />

    <form v-else class="space-y-4" @submit.prevent="submitRecoveryCode">
      <UFormField label="Recovery code" :error="form.errors.code">
        <UInput
          v-model="form.code"
          size="lg"
          placeholder="xxxxx-xxxxx"
          autocomplete="one-time-code"
          class="w-full font-mono" />
      </UFormField>
      <UButton
        type="submit"
        color="primary"
        size="lg"
        block
        :label="form.processing ? 'Verifying…' : 'Verify'"
        :loading="form.processing"
        :disabled="!form.code.trim()" />
    </form>

    <UButton
      :label="useRecoveryCode ? 'Use authenticator code' : 'Use a recovery code'"
      :icon="useRecoveryCode ? icons.qrCode : icons.key"
      color="neutral"
      variant="ghost"
      size="lg"
      block
      :disabled="form.processing"
      @click="toggleRecoveryCode" />

    <UButton
      label="Back to sign in"
      color="neutral"
      variant="ghost"
      size="lg"
      block
      :disabled="form.processing"
      @click="cancel" />
  </div>
</template>
