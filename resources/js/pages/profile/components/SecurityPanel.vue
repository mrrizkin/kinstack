<script setup lang="ts">
import AppConfirmDialog from "@/components/AppConfirmDialog.vue";
import { formatDateTime } from "@/lib/formatters";

import type {
  ActiveSession,
  PasswordForm,
  Profile,
  TwoFactor,
  TwoFactorCodeForm,
  TwoFactorPasswordForm,
} from "../types";

const pendingSession = defineModel<ActiveSession | null>("pendingSession", { required: true });

defineProps<{
  profile: Profile;
  passwordForm: PasswordForm;
  twoFactorForm: TwoFactorCodeForm;
  twoFactorPasswordForm: TwoFactorPasswordForm;
  twoFactor: TwoFactor;
  sessions: ActiveSession[];
  qrDataUrl: string;
  fieldError: (field: string, local?: Record<string, unknown>) => string;
  submitPassword: () => void;
  beginTwoFactor: () => void;
  confirmTwoFactor: () => void;
  disableTwoFactor: () => void;
  regenerateRecoveryCodes: () => void;
  revokeOtherSessions: () => void;
  confirmRevokeSession: () => void;
}>();
</script>

<template>
  <div class="mt-4 grid gap-4 lg:grid-cols-3">
    <UCard class="lg:col-span-2">
      <h2 class="text-highlighted text-sm font-semibold">Change password</h2>
      <p class="text-muted mt-1 text-xs">
        {{
          profile.invited
            ? "This account is still pending invitation. Set your first password with at least 8 characters."
            : "Enter your current password, then a new password with at least 8 characters."
        }}
      </p>
      <UAlert
        v-if="profile.invited"
        class="mt-4"
        color="info"
        variant="subtle"
        icon="i-bi-envelope-check"
        title="Invited account"
        description="You have been invited to join. Set a password to fully activate your account." />
      <form class="mt-4 space-y-4" @submit.prevent="submitPassword">
        <UFormField
          v-if="!profile.invited"
          label="Current password"
          :error="fieldError('current_password', passwordForm.errors)">
          <UInput v-model="passwordForm.current_password" type="password" placeholder="••••••••" class="w-full" />
        </UFormField>
        <div class="grid gap-4 sm:grid-cols-2">
          <UFormField label="New password" :error="fieldError('new_password', passwordForm.errors)">
            <UInput v-model="passwordForm.new_password" type="password" placeholder="••••••••" class="w-full" />
          </UFormField>
          <UFormField label="Confirm new password" :error="fieldError('confirm_password', passwordForm.errors)">
            <UInput v-model="passwordForm.confirm_password" type="password" placeholder="••••••••" class="w-full" />
          </UFormField>
        </div>
        <div class="flex justify-end">
          <UButton
            type="submit"
            color="primary"
            label="Save password"
            icon="i-bi-shield-lock"
            :loading="passwordForm.processing" />
        </div>
      </form>
    </UCard>

    <UCard class="lg:col-span-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 class="text-highlighted flex items-center gap-2 text-sm font-semibold">
            <UIcon name="i-bi-shield-check" />
            Two-step verification (2FA)
            <UBadge :color="twoFactor.enabled ? 'success' : 'neutral'" variant="subtle" size="sm">
              {{ twoFactor.enabled ? "Active" : "Inactive" }}
            </UBadge>
          </h2>
          <p class="text-muted mt-1 text-xs">
            Protect your account with a code from your authenticator app when signing in from a new device.
          </p>
        </div>
        <UButton
          v-if="!twoFactor.enabled && !twoFactor.pending_secret"
          type="button"
          color="primary"
          size="xs"
          icon="i-bi-qr-code"
          label="Enable 2FA"
          @click="beginTwoFactor" />
      </div>

      <div v-if="twoFactor.recovery_codes.length" class="mt-4 space-y-3">
        <UAlert
          color="warning"
          variant="subtle"
          icon="i-bi-exclamation-triangle"
          title="Save these recovery codes now"
          description="Each code can only be used once and will not be shown again." />
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
          <code
            v-for="code in twoFactor.recovery_codes"
            :key="code"
            class="border-default bg-elevated rounded border px-2 py-1 text-center font-mono text-xs">
            {{ code }}
          </code>
        </div>
      </div>

      <div
        v-if="!twoFactor.enabled && twoFactor.pending_secret"
        class="mt-4 flex flex-col gap-4 sm:flex-row sm:items-start">
        <div class="border-default bg-default w-fit rounded-lg border p-3">
          <img v-if="qrDataUrl" :src="qrDataUrl" alt="2FA QR code" class="size-44" />
          <p v-else class="text-muted w-44 text-center text-xs">QR code unavailable</p>
        </div>
        <form class="flex-1 space-y-3" @submit.prevent="confirmTwoFactor">
          <p class="text-muted text-xs">
            Scan the QR code with Google Authenticator or Authy, then enter the 6-digit code.
          </p>
          <p class="text-muted text-xs">
            Manual key:
            <code class="text-highlighted font-mono break-all">{{ twoFactor.pending_secret }}</code>
          </p>
          <UFormField label="Verification code" :error="fieldError('code', twoFactorForm.errors)">
            <UInput
              v-model="twoFactorForm.code"
              placeholder="123456"
              autocomplete="one-time-code"
              class="w-full sm:w-48" />
          </UFormField>
          <UButton
            type="submit"
            color="primary"
            size="sm"
            icon="i-bi-check2"
            label="Confirm"
            :loading="twoFactorForm.processing" />
        </form>
      </div>

      <div v-if="twoFactor.enabled" class="mt-4 grid gap-4 sm:grid-cols-2">
        <form class="space-y-3" @submit.prevent="regenerateRecoveryCodes">
          <p class="text-highlighted text-xs font-semibold">Regenerate recovery codes</p>
          <UFormField label="Password" :error="fieldError('two_factor_password', twoFactorPasswordForm.errors)">
            <UInput v-model="twoFactorPasswordForm.password" type="password" placeholder="••••••••" class="w-full" />
          </UFormField>
          <UButton
            type="submit"
            color="neutral"
            variant="outline"
            size="sm"
            icon="i-bi-arrow-repeat"
            label="Regenerate"
            :loading="twoFactorPasswordForm.processing" />
        </form>
        <form class="space-y-3" @submit.prevent="disableTwoFactor">
          <p class="text-highlighted text-xs font-semibold">Disable two-step verification</p>
          <UFormField label="Password" :error="fieldError('two_factor_password', twoFactorPasswordForm.errors)">
            <UInput v-model="twoFactorPasswordForm.password" type="password" placeholder="••••••••" class="w-full" />
          </UFormField>
          <UButton
            type="submit"
            color="error"
            variant="outline"
            size="sm"
            icon="i-bi-shield-slash"
            label="Disable"
            :loading="twoFactorPasswordForm.processing" />
        </form>
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
          icon="i-bi-box-arrow-right"
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
              <UIcon name="i-bi-display" class="text-muted shrink-0" />
              <span class="truncate">{{ item.user_agent || "Unknown device" }}</span>
              <UBadge v-if="item.current" color="primary" variant="subtle" size="sm">This session</UBadge>
            </p>
            <p class="text-muted mt-1 text-xs">
              {{ item.ip || "-" }} · last active {{ formatDateTime(item.last_seen_at) }} · started
              {{ formatDateTime(item.login_at) }}
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
        <li v-if="sessions.length === 0" class="text-muted py-3 text-sm">No active sessions.</li>
      </ul>
    </UCard>

    <AppConfirmDialog
      :open="Boolean(pendingSession)"
      title="Revoke session"
      :description="
        pendingSession ? `Device ${pendingSession.user_agent || 'this one'} will be signed out immediately.` : ''
      "
      confirm-label="Revoke session"
      @update:open="
        (value: boolean) => {
          if (!value) pendingSession = null;
        }
      "
      @confirm="confirmRevokeSession" />
  </div>
</template>
