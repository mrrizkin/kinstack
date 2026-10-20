<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import AppPageHeader from "@/components/AppPageHeader.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";

import AccountPanel from "./components/AccountPanel.vue";
import SecurityPanel from "./components/SecurityPanel.vue";
import { useProfileForms } from "./composables/useProfileForms";
import type { ActiveSession, Preferences, Profile, Session, TwoFactor } from "./types";

defineOptions({ layout: [BaseLayout, AppLayout] });

const props = defineProps<{
  profile: Profile;
  preferences: Preferences;
  session: Session;
  sessions: ActiveSession[];
  two_factor: TwoFactor;
}>();

const {
  activeTab,
  profileForm,
  passwordForm,
  preferencesForm,
  themeOptions,
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
  submitProfile,
  submitPreferences,
  submitPassword,
  beginTwoFactor,
  confirmTwoFactor,
  disableTwoFactor,
  confirmDisableTwoFactor,
  regenerateRecoveryCodes,
  confirmRevokeSession,
  revokeOtherSessions,
  confirmRevokeOtherSessions,
} = useProfileForms(props);
</script>

<template>
  <Head title="My profile" />
  <UDashboardPanel id="profile">
    <template #header>
      <AppPageHeader :title="profile.display_name">
        <span class="text-muted hidden text-sm lg:block">Your profile and account preferences.</span>
      </AppPageHeader>
    </template>

    <template #body>
      <UAlert v-if="formError" color="error" variant="subtle" icon="i-bi-x-circle" :title="formError" class="mb-4" />

      <UTabs
        v-model="activeTab"
        :items="[
          { label: 'Profile', value: 'profile', icon: 'i-bi-person-badge' },
          { label: 'Security', value: 'security', icon: 'i-bi-shield-lock' },
          { label: 'Preferences', value: 'preferences', icon: 'i-bi-sliders' },
        ]"
        :content="false" />

      <SecurityPanel
        v-if="activeTab === 'security'"
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
      <AccountPanel
        v-else
        :tab="activeTab"
        :profile="profile"
        :profile-form="profileForm"
        :preferences-form="preferencesForm"
        :theme-options="themeOptions"
        :field-error="fieldError"
        :submit-profile="submitProfile"
        :submit-preferences="submitPreferences" />
    </template>
  </UDashboardPanel>
</template>
