<script setup lang="ts">
import { computed } from "vue";

import AppConfirmDialog from "@/components/AppConfirmDialog.vue";
import AppPagination from "@/components/AppPagination.vue";
import AppCellTime from "@/components/cells/AppCellTime.vue";
import type { PageInfo } from "@/lib/contracts/pages";
import { icons } from "@/lib/icons";

import type { TwoFactorAction } from "../composables/useSecurityForms";
import type {
  ActiveSession,
  PasswordForm,
  Profile,
  TwoFactor,
  TwoFactorCodeForm,
  TwoFactorPasswordForm,
} from "../types";
import PasswordConfirmDialog from "./PasswordConfirmDialog.vue";
import PasswordDialog from "./PasswordDialog.vue";
import TwoFactorEnrollDialog from "./TwoFactorEnrollDialog.vue";

const passwordDialogOpen = defineModel<boolean>("passwordDialogOpen", { required: true });
const enrollDialogOpen = defineModel<boolean>("enrollDialogOpen", { required: true });
const twoFactorAction = defineModel<TwoFactorAction | null>("twoFactorAction", { required: true });
const pendingSession = defineModel<ActiveSession | null>("pendingSession", { required: true });
const pendingRevokeOthers = defineModel<boolean>("pendingRevokeOthers", { required: true });

const props = defineProps<{
  profile: Profile;
  passwordForm: PasswordForm;
  twoFactorForm: TwoFactorCodeForm;
  twoFactorPasswordForm: TwoFactorPasswordForm;
  twoFactor: TwoFactor;
  sessions: ActiveSession[];
  sessionsPage: PageInfo;
  qrDataUrl: string;
  revokingSession: boolean;
  revokingOthers: boolean;
  beginningTwoFactor: boolean;
  fieldError: (field: string, local?: Record<string, unknown>) => string | undefined;
  openPasswordDialog: () => void;
  submitPassword: () => void;
  openEnrollDialog: () => void;
  closeEnrollDialog: (open: boolean) => void;
  openTwoFactorAction: (action: TwoFactorAction) => void;
  closeTwoFactorAction: (open: boolean) => void;
  confirmTwoFactor: () => void;
  confirmTwoFactorAction: () => void;
  confirmRevokeSession: () => void;
  revokeOtherSessions: () => void;
  confirmRevokeOtherSessions: () => void;
}>();

function closePendingSession(value: boolean) {
  if (!value) pendingSession.value = null;
}

function closeRevokeOthers(value: boolean) {
  if (!value) pendingRevokeOthers.value = false;
}

const twoFactorActionDialog = computed(() => {
  if (twoFactorAction.value === "disable") {
    return {
      title: "Disable two-step verification",
      description:
        "Your account will be protected by your password only until you enable 2FA again. Confirm your password to continue.",
      confirmLabel: "Disable 2FA",
      color: "error" as const,
    };
  }
  return {
    title: "Regenerate recovery codes",
    description: "New codes replace every existing recovery code immediately. Confirm your password to continue.",
    confirmLabel: "Regenerate codes",
    color: "primary" as const,
  };
});

const enrollLabel = computed(() => (props.twoFactor.pending_secret ? "Continue setup" : "Enable 2FA"));
</script>

<template>
  <div class="grid gap-4 lg:grid-cols-3">
    <UCard class="lg:col-span-2">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 class="text-highlighted flex items-center gap-2 text-sm font-semibold">
            <UIcon :name="icons.lock" class="text-muted" />
            Password
          </h2>
          <p class="text-muted mt-1 text-xs">
            {{
              profile.invited
                ? "This account is still pending invitation. Set your first password with at least 8 characters."
                : "Your sign-in password. Changing it signs out every other device."
            }}
          </p>
        </div>
        <UButton
          type="button"
          color="primary"
          size="xs"
          :icon="profile.invited ? icons.invite : icons.lock"
          :label="profile.invited ? 'Set password' : 'Change password'"
          @click="openPasswordDialog" />
      </div>

      <UAlert
        v-if="profile.invited"
        class="mt-4"
        color="info"
        :icon="icons.invite"
        title="Invited account"
        description="You have been invited to join. Set a password to fully activate your account." />

      <dl class="mt-4 flex flex-wrap gap-x-8 gap-y-2 text-xs">
        <div>
          <dt class="text-muted">Minimum length</dt>
          <dd class="text-highlighted">8 characters</dd>
        </div>
        <div>
          <dt class="text-muted">Verification</dt>
          <dd class="text-highlighted">{{ profile.invited ? "Not required yet" : "Current password required" }}</dd>
        </div>
      </dl>
    </UCard>

    <UCard>
      <div class="flex items-start justify-between gap-2">
        <h2 class="text-highlighted flex items-center gap-2 text-sm font-semibold">
          <UIcon :name="icons.shieldCheck" class="text-muted" />
          Two-step verification
        </h2>
        <UBadge :color="twoFactor.enabled ? 'success' : 'neutral'" variant="subtle" size="sm">
          {{ twoFactor.enabled ? "Active" : "Inactive" }}
        </UBadge>
      </div>
      <p class="text-muted mt-1 text-xs">
        {{
          twoFactor.enabled
            ? "Your account asks for a code from your authenticator app when signing in from a new device."
            : "Protect your account with a code from your authenticator app when signing in from a new device."
        }}
      </p>

      <div class="mt-4 flex flex-col gap-2">
        <template v-if="twoFactor.enabled">
          <UButton
            type="button"
            color="neutral"
            variant="outline"
            :icon="icons.reload"
            label="Regenerate recovery codes"
            block
            @click="openTwoFactorAction('regenerate')" />
          <UButton
            type="button"
            color="error"
            variant="outline"
            :icon="icons.shieldOff"
            label="Disable 2FA"
            block
            @click="openTwoFactorAction('disable')" />
        </template>
        <UButton
          v-else
          type="button"
          color="primary"
          :icon="twoFactor.pending_secret ? icons.arrowRight : icons.qrCode"
          :label="enrollLabel"
          :loading="beginningTwoFactor"
          block
          @click="openEnrollDialog" />
      </div>
    </UCard>

    <UCard v-if="twoFactor.recovery_codes.length" class="lg:col-span-3">
      <UAlert
        color="warning"
        :icon="icons.warning"
        title="Save these recovery codes now"
        description="Each code can only be used once and will not be shown again." />
      <div class="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
        <code
          v-for="code in twoFactor.recovery_codes"
          :key="code"
          class="border-default bg-elevated rounded border px-2 py-1 text-center font-mono text-xs">
          {{ code }}
        </code>
      </div>
    </UCard>

    <UCard class="lg:col-span-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 class="text-highlighted text-sm font-semibold">Active sessions</h2>
          <p class="text-muted mt-1 text-xs">All devices currently signed in to this account.</p>
        </div>
        <UButton
          v-if="sessions.length > 1"
          type="button"
          size="xs"
          color="error"
          variant="outline"
          :icon="icons.logout"
          label="Sign out other sessions"
          @click="revokeOtherSessions" />
      </div>
      <ul class="divide-default mt-4 divide-y">
        <li
          v-for="item in sessions"
          :key="item.public_id"
          class="flex flex-wrap items-center justify-between gap-3 py-3">
          <div class="min-w-0">
            <p class="text-highlighted flex items-center gap-2 text-sm font-medium">
              <UIcon :name="icons.system" class="text-muted shrink-0" />
              <span class="truncate">{{ item.user_agent || "Unknown device" }}</span>
              <UBadge v-if="item.current" color="primary" variant="subtle" size="sm">This session</UBadge>
            </p>
            <p class="text-muted mt-1 text-xs">
              {{ item.ip || "-" }} · last active <AppCellTime :value="item.last_seen_at" inline /> · started
              <AppCellTime :value="item.login_at" inline />
            </p>
          </div>
          <UButton
            v-if="!item.current"
            type="button"
            size="xs"
            color="error"
            variant="link"
            label="Revoke"
            @click="pendingSession = item" />
        </li>
        <li
          v-if="sessions.length === 0"
          class="text-muted flex flex-wrap items-center justify-between gap-2 py-3 text-sm">
          <span>No active sessions.</span>
          <UButton to="/help" size="xs" color="neutral" variant="link" label="View help" />
        </li>
      </ul>
      <AppPagination :page="sessionsPage" url="/profile/security" />
    </UCard>

    <PasswordDialog
      v-model:open="passwordDialogOpen"
      :profile="profile"
      :password-form="passwordForm"
      :field-error="fieldError"
      @submit="submitPassword" />

    <TwoFactorEnrollDialog
      :open="enrollDialogOpen"
      :two-factor="twoFactor"
      :two-factor-form="twoFactorForm"
      :qr-data-url="qrDataUrl"
      :beginning-two-factor="beginningTwoFactor"
      :field-error="fieldError"
      @update:open="closeEnrollDialog"
      @submit="confirmTwoFactor" />

    <PasswordConfirmDialog
      :open="twoFactorAction !== null"
      :title="twoFactorActionDialog.title"
      :description="twoFactorActionDialog.description"
      :confirm-label="twoFactorActionDialog.confirmLabel"
      :color="twoFactorActionDialog.color"
      :password-form="twoFactorPasswordForm"
      :field-error="fieldError"
      @update:open="closeTwoFactorAction"
      @confirm="confirmTwoFactorAction" />

    <AppConfirmDialog
      :open="Boolean(pendingSession)"
      title="Revoke session"
      :description="
        pendingSession ? `Device ${pendingSession.user_agent || 'this one'} will be signed out immediately.` : ''
      "
      confirm-label="Revoke session"
      :loading="revokingSession"
      @update:open="closePendingSession"
      @confirm="confirmRevokeSession" />

    <AppConfirmDialog
      :open="pendingRevokeOthers"
      title="Sign out other sessions"
      description="Every other device will be signed out immediately. This session stays active."
      confirm-label="Sign out other sessions"
      :loading="revokingOthers"
      @update:open="closeRevokeOthers"
      @confirm="confirmRevokeOtherSessions" />
  </div>
</template>
