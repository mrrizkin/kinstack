<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import AppPageHeader from "@/components/AppPageHeader.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import SettingsLayout from "@/layouts/SettingsLayout.vue";
import { icons } from "@/lib/icons";

import SecurityPanel from "./components/SecurityPanel.vue";
import { useSecurityForms } from "./composables/useSecurityForms";
import type { ActiveSession, Profile, TwoFactor } from "./types";

defineOptions({ layout: [BaseLayout, SettingsLayout] });

const props = defineProps<{
  profile: Profile;
  sessions: ActiveSession[];
  two_factor: TwoFactor;
}>();

const {
  passwordForm,
  fieldError,
  formError,
  pendingSession,
  revokingSession,
  pendingRevokeOthers,
  revokingOthers,
  beginningTwoFactor,
  pendingDisableTwoFactor,
  disablingTwoFactor,
  twoFactorForm,
  twoFactorPasswordForm,
  qrDataUrl,
  submitPassword,
  beginTwoFactor,
  confirmTwoFactor,
  disableTwoFactor,
  confirmDisableTwoFactor,
  regenerateRecoveryCodes,
  confirmRevokeSession,
  revokeOtherSessions,
  confirmRevokeOtherSessions,
} = useSecurityForms(props);
</script>

<template>
  <Head title="Security" />
  <UDashboardPanel id="settings-security">
    <template #header>
      <AppPageHeader title="Security" description="Password, two-step verification, and active sessions." />
    </template>

    <template #body>
      <UAlert v-if="formError" color="error" variant="subtle" :icon="icons.error" :title="formError" class="mb-4" />

      <SecurityPanel
        v-model:pending-session="pendingSession"
        :profile="profile"
        :password-form="passwordForm"
        :two-factor-form="twoFactorForm"
        :two-factor-password-form="twoFactorPasswordForm"
        :two-factor="two_factor"
        :sessions="sessions"
        :qr-data-url="qrDataUrl"
        :field-error="fieldError"
        :revoking-session="revokingSession"
        v-model:pending-revoke-others="pendingRevokeOthers"
        :revoking-others="revokingOthers"
        :beginning-two-factor="beginningTwoFactor"
        v-model:pending-disable-two-factor="pendingDisableTwoFactor"
        :disabling-two-factor="disablingTwoFactor"
        :submit-password="submitPassword"
        :begin-two-factor="beginTwoFactor"
        :confirm-two-factor="confirmTwoFactor"
        :disable-two-factor="disableTwoFactor"
        :confirm-disable-two-factor="confirmDisableTwoFactor"
        :regenerate-recovery-codes="regenerateRecoveryCodes"
        :revoke-other-sessions="revokeOtherSessions"
        :confirm-revoke-session="confirmRevokeSession"
        :confirm-revoke-other-sessions="confirmRevokeOtherSessions" />
    </template>
  </UDashboardPanel>
</template>
