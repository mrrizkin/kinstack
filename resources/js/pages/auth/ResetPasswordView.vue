<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";
import type { AuthFormField, FormSubmitEvent } from "@nuxt/ui";
import { computed } from "vue";

import AuthLayout from "@/layouts/AuthLayout.vue";

const props = defineProps<{
  token?: string;
  done?: boolean;
}>();

const form = useForm({
  new_password: "",
  confirm_password: "",
});

const fields = computed<AuthFormField[]>(() => [
  {
    name: "new_password",
    type: "password",
    label: "New password",
    placeholder: "At least 8 characters",
    autocomplete: "new-password",
    required: true,
    size: "lg",
    error: form.errors.new_password,
  },
  {
    name: "confirm_password",
    type: "password",
    label: "Confirm new password",
    placeholder: "Confirm new password",
    autocomplete: "new-password",
    required: true,
    size: "lg",
    error: form.errors.confirm_password,
  },
]);

function submit(payload: FormSubmitEvent<Record<string, unknown>>) {
  form.new_password = String(payload.data.new_password ?? "");
  form.confirm_password = String(payload.data.confirm_password ?? "");
  form.post(`/login/reset/${props.token ?? ""}`);
}
</script>

<template>
  <Head title="Reset password" />
  <AuthLayout title="Reset password" subtitle="Set a new password for your account.">
    <div class="space-y-4">
      <template v-if="props.done">
        <UAlert
          color="success"
          variant="subtle"
          icon="i-material-symbols:verified-user-outline-rounded"
          title="Password changed successfully"
          description="You can now sign in with your new password." />
        <UButton to="/login" color="primary" size="lg" block label="Sign in" />
      </template>

      <template v-else>
        <UAuthForm
          :fields="fields"
          :submit="{
            label: form.processing ? 'Saving…' : 'Save password',
            color: 'primary',
            block: true,
            size: 'lg',
            loading: form.processing,
          }"
          @submit="submit" />

        <UButton to="/login" label="Back to sign in" color="neutral" variant="ghost" size="lg" block />
      </template>
    </div>
  </AuthLayout>
</template>
