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
  passwordDialogOpen,
  fieldError,
  formError,
  pendingSession,
  revokingSession,
  pendingRevokeOthers,
  revokingOthers,
  beginningTwoFactor,
  enrollDialogOpen,
  twoFactorForm,
  twoFactorPasswordForm,
  twoFactorAction,
  qrDataUrl,
  openPasswordDialog,
  submitPassword,
  openEnrollDialog,
  closeEnrollDialog,
  openTwoFactorAction,
  closeTwoFactorAction,
  confirmTwoFactor,
  confirmTwoFactorAction,
  confirmRevokeSession,
  revokeOtherSessions,
  confirmRevokeOtherSessions,
} = useSecurityForms(props);
</script>

<template>
  <Head :title="t('shell.settings.sections.security.title')" />

  <UAlert v-if="formError" color="error" variant="subtle" :icon="icons.error" :title="formError" class="mb-4" />

  <SecurityPanel
    v-model:password-dialog-open="passwordDialogOpen"
    v-model:enroll-dialog-open="enrollDialogOpen"
    v-model:two-factor-action="twoFactorAction"
    v-model:pending-session="pendingSession"
    v-model:pending-revoke-others="pendingRevokeOthers"
    :profile="profile"
    :password-form="passwordForm"
    :two-factor-form="twoFactorForm"
    :two-factor-password-form="twoFactorPasswordForm"
    :two-factor="two_factor"
    :sessions="sessions"
    :qr-data-url="qrDataUrl"
    :field-error="fieldError"
    :revoking-session="revokingSession"
    :revoking-others="revokingOthers"
    :beginning-two-factor="beginningTwoFactor"
    :open-password-dialog="openPasswordDialog"
    :submit-password="submitPassword"
    :open-enroll-dialog="openEnrollDialog"
    :close-enroll-dialog="closeEnrollDialog"
    :open-two-factor-action="openTwoFactorAction"
    :close-two-factor-action="closeTwoFactorAction"
    :confirm-two-factor="confirmTwoFactor"
    :confirm-two-factor-action="confirmTwoFactorAction"
    :confirm-revoke-session="confirmRevokeSession"
    :revoke-other-sessions="revokeOtherSessions"
    :confirm-revoke-other-sessions="confirmRevokeOtherSessions" />
</template>
