<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";
import type { AuthFormField, FormSubmitEvent } from "@nuxt/ui";
import { computed, ref } from "vue";

import AuthLayout from "@/layouts/AuthLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { passkeyCredentialSchema, passkeyOptionsSchema } from "@/lib/contracts/passkeys";
import { requestJson } from "@/lib/http";
import { usePageProps } from "@/lib/page-props";
import { getPasskey, supportsWebAuthn } from "@/lib/webauthn";

defineOptions({
  layout: [
    BaseLayout,
    [AuthLayout, { title: "Sign in to your account", subtitle: "Use the credentials provided by your administrator." }],
  ],
});

const props = defineProps<{
  notice?: string;
  oauth_enabled?: boolean;
  oauth_providers?: { name: string; label: string }[];
  oauth_error?: string;
  magic_link_enabled?: boolean;
  passkey_enabled?: boolean;
  password_reset_enabled?: boolean;
  saml_enabled?: boolean;
}>();

const { csrfToken } = usePageProps();
const passkeySupported = supportsWebAuthn();
const hasAlternatives = computed(() =>
  Boolean(
    props.magic_link_enabled ||
    props.oauth_enabled ||
    props.saml_enabled ||
    (props.passkey_enabled && passkeySupported),
  ),
);
const passkeyLoading = ref(false);
const passkeyError = ref("");

const form = useForm({
  username: "",
  password: "",
});

const credentialsError = computed(() => {
  const errors = form.errors as Record<string, string | undefined>;
  return errors.auth ?? "";
});

const fields = computed<AuthFormField[]>(() => [
  {
    name: "username",
    type: "text",
    label: "Username",
    placeholder: "Enter username",
    icon: "i-material-symbols:person-outline-rounded",
    autocomplete: "username",
    required: true,
    size: "lg",
    error: form.errors.username,
  },
  {
    name: "password",
    type: "password",
    label: "Password",
    placeholder: "••••••••",
    autocomplete: "current-password",
    required: true,
    size: "lg",
    error: form.errors.password,
  },
]);

function submit(payload: FormSubmitEvent<Record<string, unknown>>) {
  form.clearErrors();
  form.username = String(payload.data.username ?? "").trim();
  form.password = String(payload.data.password ?? "");

  const validation: { username?: string; password?: string } = {};
  if (!form.username) validation.username = "Username is required.";
  if (!form.password) validation.password = "Password is required.";
  if (Object.keys(validation).length > 0) {
    form.setError(validation);
    return;
  }

  form.post("/login", {
    onFinish: () => form.reset("password"),
  });
}

async function submitPasskey() {
  if (passkeyLoading.value) return;
  passkeyLoading.value = true;
  passkeyError.value = "";
  try {
    const options = await requestJson("/login/passkey/options", passkeyOptionsSchema, {
      method: "POST",
      csrfToken: csrfToken.value,
    });
    const credential = await getPasskey(options);
    await requestJson("/login/passkey", passkeyCredentialSchema, {
      method: "POST",
      csrfToken: csrfToken.value,
      body: credential,
    });
    window.location.assign("/");
  } catch (caught) {
    passkeyError.value = caught instanceof Error ? caught.message : "Passkey sign-in failed.";
  } finally {
    passkeyLoading.value = false;
  }
}

const magicForm = useForm({
  email: "",
});

function submitMagicLink() {
  magicForm.post("/login/magic-link", {
    onSuccess: () => magicForm.reset("email"),
  });
}
</script>

<template>
  <Head title="Sign in" />
  <div class="space-y-4">
    <UAlert
      v-if="notice"
      color="success"
      variant="subtle"
      icon="i-material-symbols:mark-email-read-outline-rounded"
      :title="notice" />
    <UAlert
      v-if="oauth_error"
      color="error"
      variant="subtle"
      icon="i-material-symbols:cancel-outline-rounded"
      :title="oauth_error" />
    <UAlert
      v-if="credentialsError"
      color="error"
      variant="subtle"
      icon="i-material-symbols:cancel-outline-rounded"
      :title="credentialsError" />

    <UAuthForm
      :fields="fields"
      :submit="{
        label: form.processing ? 'Signing in…' : 'Sign in',
        color: 'primary',
        block: true,
        size: 'lg',
        loading: form.processing,
      }"
      @submit="submit">
      <template v-if="password_reset_enabled" #password-hint>
        <ULink
          to="/login/forgot"
          class="text-primary text-xs font-medium underline-offset-4 hover:underline"
          tabindex="-1">
          Forgot password?
        </ULink>
      </template>

      <template v-if="hasAlternatives" #footer>
        <div class="space-y-2.5">
          <USeparator label="or" class="mb-1" />
          <UButton
            v-if="passkey_enabled && passkeySupported"
            label="Sign in with passkey"
            icon="i-material-symbols:passkey-rounded"
            color="neutral"
            variant="outline"
            size="lg"
            block
            :loading="passkeyLoading"
            :disabled="passkeyLoading"
            @click="submitPasskey" />

          <UButton
            v-for="provider in oauth_providers ?? []"
            :key="provider.name"
            :label="`Sign in with ${provider.label}`"
            icon="i-material-symbols:login-rounded"
            color="neutral"
            variant="outline"
            size="lg"
            block
            :to="`/login/oauth/start?provider=${provider.name}`" />

          <UButton
            v-if="saml_enabled"
            label="Sign in with SAML"
            icon="i-material-symbols:domain-rounded"
            color="neutral"
            variant="outline"
            size="lg"
            block
            to="/login/saml/start" />

          <form
            v-if="magic_link_enabled"
            class="flex flex-col gap-2 pt-2 sm:flex-row"
            @submit.prevent="submitMagicLink">
            <UFormField name="email" :error="magicForm.errors.email" class="flex-1">
              <UInput
                v-model="magicForm.email"
                type="email"
                icon="i-material-symbols:mail-outline-rounded"
                placeholder="name@example.com"
                autocomplete="email"
                size="lg"
                class="w-full" />
            </UFormField>
            <UButton
              type="submit"
              label="Send link"
              color="neutral"
              variant="outline"
              size="lg"
              :loading="magicForm.processing"
              :disabled="magicForm.processing" />
          </form>

          <UAlert
            v-if="passkeyError"
            color="error"
            variant="subtle"
            icon="i-material-symbols:cancel-outline-rounded"
            :title="passkeyError" />
        </div>
      </template>
    </UAuthForm>

    <p class="text-muted text-center text-xs leading-5">
      Don't have access yet? Contact your administrator to request an invitation.
    </p>
  </div>
</template>
