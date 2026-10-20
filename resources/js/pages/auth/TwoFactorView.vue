<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";
import type { AuthFormField, FormSubmitEvent } from "@nuxt/ui";
import { computed } from "vue";

import AuthLayout from "@/layouts/AuthLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";

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

const fields = computed<AuthFormField[]>(() => [
  {
    name: "code",
    type: "text",
    label: "Verification code",
    placeholder: "123456 or recovery code",
    icon: "i-material-symbols:shield-outline-rounded",
    autocomplete: "one-time-code",
    required: true,
    size: "lg",
    error: form.errors.code,
  },
]);

function submit(payload: FormSubmitEvent<Record<string, unknown>>) {
  form.code = String(payload.data.code ?? "");
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
      :fields="fields"
      :submit="{
        label: form.processing ? 'Verifying…' : 'Verify',
        color: 'primary',
        block: true,
        size: 'lg',
        loading: form.processing,
      }"
      @submit="submit" />

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
