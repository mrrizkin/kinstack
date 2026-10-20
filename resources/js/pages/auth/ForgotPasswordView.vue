<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";
import type { AuthFormField, FormSubmitEvent } from "@nuxt/ui";
import { computed } from "vue";

import AuthLayout from "@/layouts/AuthLayout.vue";

defineProps<{
  notice?: string;
}>();

const form = useForm({ email: "" });

const fields = computed<AuthFormField[]>(() => [
  {
    name: "email",
    type: "email",
    label: "Email",
    placeholder: "name@example.com",
    icon: "i-material-symbols:mail-outline-rounded",
    autocomplete: "email",
    required: true,
    size: "lg",
    error: form.errors.email,
  },
]);

function submit(payload: FormSubmitEvent<Record<string, unknown>>) {
  form.email = String(payload.data.email ?? "");
  form.post("/login/forgot");
}
</script>

<template>
  <Head title="Forgot password" />
  <AuthLayout title="Forgot password" subtitle="We will send a reset link to your email.">
    <div class="space-y-4">
      <UAlert
        v-if="notice"
        color="success"
        variant="subtle"
        icon="i-material-symbols:mark-email-read-outline-rounded"
        :title="notice" />

      <UAuthForm
        :fields="fields"
        :submit="{
          label: form.processing ? 'Sending…' : 'Send reset link',
          color: 'primary',
          block: true,
          size: 'lg',
          loading: form.processing,
        }"
        @submit="submit" />

      <UButton to="/login" label="Back to sign in" color="neutral" variant="ghost" size="lg" block />
    </div>
  </AuthLayout>
</template>
