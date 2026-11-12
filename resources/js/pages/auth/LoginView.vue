<script setup lang="ts">
import { Head, useForm } from "@inertiajs/vue3";
import type { AuthFormField, FormSubmitEvent } from "@nuxt/ui";
import { computed, ref } from "vue";

import AuthLayout from "@/layouts/AuthLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { loginPageSchema } from "@/lib/contracts/pages";
import { passkeyCredentialSchema, passkeyOptionsSchema } from "@/lib/contracts/passkeys";
import { useAction } from "@/lib/dialog";
import { requestJson } from "@/lib/http";
import { icons } from "@/lib/icons";
import { usePageSchema } from "@/lib/page-props";
import { usePageProps } from "@/lib/page-props";
import { getPasskey, supportsWebAuthn } from "@/lib/webauthn";

defineOptions({
  layout: [
    BaseLayout,
    [AuthLayout, { title: "Sign in to your account", subtitle: "Use the credentials provided by your administrator." }],
  ],
});

const { notice, oauth_enabled, oauth_providers, oauth_error, passkey_enabled, password_reset_enabled, saml_enabled } =
  usePageSchema(loginPageSchema);

const { csrfToken } = usePageProps();
const passkeySupported = supportsWebAuthn();
const hasAlternatives = computed(() => Boolean(oauth_enabled || saml_enabled || (passkey_enabled && passkeySupported)));
const passkeyRequest = useAction();
const passkeyLoading = passkeyRequest.isRunning;
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
    icon: icons.account,
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
  if (!passkeyRequest.start()) return;
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
    passkeyRequest.settled();
  }
}
</script>

<template>
  <Head title="Sign in" />
  <div class="space-y-4">
    <UAlert v-if="notice" color="success" variant="subtle" :icon="icons.markRead" :title="notice" />
    <UAlert v-if="oauth_error" color="error" variant="subtle" :icon="icons.error" :title="oauth_error" />
    <UAlert v-if="credentialsError" color="error" variant="subtle" :icon="icons.error" :title="credentialsError" />

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
            :icon="icons.passkey"
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
            :icon="icons.login"
            color="neutral"
            variant="outline"
            size="lg"
            block
            :to="`/login/oauth/start?provider=${provider.name}`" />

          <UButton
            v-if="saml_enabled"
            label="Sign in with SAML"
            :icon="icons.domain"
            color="neutral"
            variant="outline"
            size="lg"
            block
            to="/login/saml/start" />

          <UAlert v-if="passkeyError" color="error" variant="subtle" :icon="icons.error" :title="passkeyError" />
        </div>
      </template>
    </UAuthForm>

    <p class="text-muted text-center text-xs leading-5">
      Don't have access yet? Contact your administrator to request an invitation.
    </p>
  </div>
</template>
