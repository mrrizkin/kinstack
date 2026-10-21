<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { useI18n } from "vue-i18n";

import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import SettingsLayout from "@/layouts/SettingsLayout.vue";
import { icons } from "@/lib/icons";

import SecurityPanel from "./components/SecurityPanel.vue";
import { useSecurityForms } from "./composables/useSecurityForms";
import type { ActiveSession, Profile, TwoFactor } from "./types";

defineOptions({ layout: [BaseLayout, AppLayout, SettingsLayout] });

const props = defineProps<{
  profile: Profile;
  sessions: ActiveSession[];
  two_factor: TwoFactor;
}>();

const { t } = useI18n();

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
  <Head :title="t('shell.settings.sections.security.title')" />

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
