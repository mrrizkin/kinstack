<script setup lang="ts">
import { Head, router, useForm } from "@inertiajs/vue3";
import QRCode from "qrcode";
import { computed, onMounted, ref, watch } from "vue";

import AppConfirmDialog from "@/components/AppConfirmDialog.vue";
import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import { useServerErrors } from "@/lib/errors";
import { formatDateTime } from "@/lib/formatters";
import { useNotificationSound } from "@/lib/sound";
import { useTheme, type ThemePreference } from "@/lib/theme";

interface Profile {
  public_id: string;
  username: string;
  display_name: string;
  role: string;
  invited: boolean;
  last_login_at: string;
  created_at: string;
}

interface Preferences {
  theme: string;
  sound_enabled: boolean;
}

interface Session {
  ip: string;
  user_agent: string;
  login_at: string;
}

interface ActiveSession {
  public_id: string;
  ip: string;
  user_agent: string;
  login_at: string;
  last_seen_at: string;
  current: boolean;
}

interface TwoFactor {
  enabled: boolean;
  pending_secret: string;
  uri: string;
  recovery_codes: string[];
}

const props = defineProps<{
  profile: Profile;
  preferences: Preferences;
  session: Session;
  sessions: ActiveSession[];
  two_factor: TwoFactor;
}>();

const { preference, setPreference } = useTheme();
const { enabled: soundEnabled } = useNotificationSound();

const activeTab = ref("profile");

const profileForm = useForm({ display_name: props.profile.display_name });
const passwordForm = useForm({
  current_password: "",
  new_password: "",
  confirm_password: "",
});
const preferencesForm = useForm({
  theme: props.preferences.theme as ThemePreference,
  sound_enabled: props.preferences.sound_enabled,
});

const themeOptions: { label: string; value: ThemePreference; icon: string }[] = [
  { label: "Light", value: "light", icon: "i-bi-sun" },
  { label: "Dark", value: "dark", icon: "i-bi-moon" },
  { label: "Follow system", value: "system", icon: "i-bi-display" },
];

const roleLabel = computed(() => {
  const code = props.profile.role;
  if (!code) return "No role";
  return code.charAt(0).toUpperCase() + code.slice(1);
});
const roleColor = "neutral" as const;

// Apply the stored preferences on visit so another device picks them up.
onMounted(() => {
  if (preference.value !== props.preferences.theme) {
    setPreference(props.preferences.theme as ThemePreference);
  }
  if (soundEnabled.value !== props.preferences.sound_enabled) {
    soundEnabled.value = props.preferences.sound_enabled;
  }
});

watch(
  () => props.profile.display_name,
  (value) => {
    profileForm.display_name = value;
  },
);

function submitProfile() {
  profileForm.post(`/profile`, { preserveScroll: true });
}

function submitPreferences() {
  // Apply immediately so the change is visible before the server round-trip.
  setPreference(preferencesForm.theme);
  soundEnabled.value = preferencesForm.sound_enabled;
  preferencesForm.post(`/profile/preferences`, { preserveScroll: true });
}

function submitPassword() {
  passwordForm.post(`/profile/password`, {
    preserveScroll: true,
    onSuccess: () => passwordForm.reset(),
  });
}

const { message: fieldError, formError } = useServerErrors();

const pendingSession = ref<ActiveSession | null>(null);

const twoFactorForm = useForm({ code: "" });
const twoFactorPasswordForm = useForm({ password: "" });
const qrDataUrl = ref("");

watch(
  () => props.two_factor.uri,
  async (uri) => {
    qrDataUrl.value = uri ? await QRCode.toDataURL(uri, { margin: 1, width: 220 }) : "";
  },
  { immediate: true },
);

function beginTwoFactor() {
  router.post(`/profile/two-factor/begin`, undefined, { preserveScroll: true });
}

function confirmTwoFactor() {
  twoFactorForm.post(`/profile/two-factor/confirm`, {
    preserveScroll: true,
    onSuccess: () => twoFactorForm.reset(),
  });
}

function disableTwoFactor() {
  twoFactorPasswordForm.post(`/profile/two-factor/disable`, {
    preserveScroll: true,
    onSuccess: () => twoFactorPasswordForm.reset(),
  });
}

function regenerateRecoveryCodes() {
  twoFactorPasswordForm.post(`/profile/two-factor/recovery-codes`, {
    preserveScroll: true,
    onSuccess: () => twoFactorPasswordForm.reset(),
  });
}

function confirmRevokeSession() {
  const target = pendingSession.value;
  if (!target) return;
  router.post(`/profile/sessions/${target.public_id}/revoke`, undefined, {
    preserveScroll: true,
    onFinish: () => {
      pendingSession.value = null;
    },
  });
}

function revokeOtherSessions() {
  router.post(`/profile/sessions/revoke-others`, undefined, { preserveScroll: true });
}
</script>

<template>
  <Head title="My profile" />
  <AppLayout>
    <UDashboardPanel id="profile">
      <template #header>
        <UDashboardNavbar :title="profile.display_name">
          <template #leading>
            <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
          </template>
          <span class="text-muted hidden text-sm lg:block">Your profile and account preferences.</span>
          <template #right>
            <AppNotificationBell />
          </template>
        </UDashboardNavbar>
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

        <!-- Profile -->
        <div v-if="activeTab === 'profile'" class="mt-4 grid gap-4 lg:grid-cols-3">
          <UCard class="lg:col-span-2">
            <h2 class="text-highlighted text-sm font-semibold">Account information</h2>
            <form class="mt-4 space-y-4" @submit.prevent="submitProfile">
              <UFormField label="Username" description="Username cannot be changed.">
                <UInput :model-value="profile.username" disabled class="w-full" />
              </UFormField>
              <UFormField
                label="Display name"
                hint="3-120 characters"
                :error="fieldError('display_name', profileForm.errors)">
                <UInput v-model="profileForm.display_name" class="w-full" />
              </UFormField>
              <div class="flex justify-end">
                <UButton
                  type="submit"
                  color="primary"
                  label="Save profile"
                  icon="i-bi-check2"
                  :loading="profileForm.processing" />
              </div>
            </form>
          </UCard>

          <UCard>
            <h2 class="text-highlighted text-sm font-semibold">Overview</h2>
            <dl class="mt-4 space-y-3 text-sm">
              <div class="flex items-center justify-between gap-2">
                <dt class="text-muted">Role</dt>
                <dd>
                  <UBadge :color="roleColor" variant="subtle" size="sm">{{ roleLabel }}</UBadge>
                </dd>
              </div>
              <div class="flex items-center justify-between gap-2">
                <dt class="text-muted">Last login</dt>
                <dd class="text-highlighted">{{ formatDateTime(profile.last_login_at) }}</dd>
              </div>
              <div class="flex items-center justify-between gap-2">
                <dt class="text-muted">Account created</dt>
                <dd class="text-highlighted">{{ formatDateTime(profile.created_at) }}</dd>
              </div>
            </dl>
          </UCard>
        </div>

        <!-- Security -->
        <div v-else-if="activeTab === 'security'" class="mt-4 grid gap-4 lg:grid-cols-3">
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
                  <UInput
                    v-model="passwordForm.confirm_password"
                    type="password"
                    placeholder="••••••••"
                    class="w-full" />
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
                  <UBadge :color="two_factor.enabled ? 'success' : 'neutral'" variant="subtle" size="sm">
                    {{ two_factor.enabled ? "Active" : "Inactive" }}
                  </UBadge>
                </h2>
                <p class="text-muted mt-1 text-xs">
                  Protect your account with a code from your authenticator app when signing in from a new device.
                </p>
              </div>
              <UButton
                v-if="!two_factor.enabled && !two_factor.pending_secret"
                type="button"
                color="primary"
                size="xs"
                icon="i-bi-qr-code"
                label="Enable 2FA"
                @click="beginTwoFactor" />
            </div>

            <div v-if="two_factor.recovery_codes.length" class="mt-4 space-y-3">
              <UAlert
                color="warning"
                variant="subtle"
                icon="i-bi-exclamation-triangle"
                title="Save these recovery codes now"
                description="Each code can only be used once and will not be shown again." />
              <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
                <code
                  v-for="code in two_factor.recovery_codes"
                  :key="code"
                  class="border-default bg-elevated rounded border px-2 py-1 text-center font-mono text-xs">
                  {{ code }}
                </code>
              </div>
            </div>

            <div
              v-if="!two_factor.enabled && two_factor.pending_secret"
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
                  <code class="text-highlighted font-mono break-all">{{ two_factor.pending_secret }}</code>
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

            <div v-if="two_factor.enabled" class="mt-4 grid gap-4 sm:grid-cols-2">
              <form class="space-y-3" @submit.prevent="regenerateRecoveryCodes">
                <p class="text-highlighted text-xs font-semibold">Regenerate recovery codes</p>
                <UFormField label="Password" :error="fieldError('two_factor_password', twoFactorPasswordForm.errors)">
                  <UInput
                    v-model="twoFactorPasswordForm.password"
                    type="password"
                    placeholder="••••••••"
                    class="w-full" />
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
                  <UInput
                    v-model="twoFactorPasswordForm.password"
                    type="password"
                    placeholder="••••••••"
                    class="w-full" />
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

        <!-- Preferences -->
        <div v-else class="mt-4 grid gap-4 lg:grid-cols-3">
          <UCard class="lg:col-span-2">
            <h2 class="text-highlighted text-sm font-semibold">Personalization</h2>
            <form class="mt-4 space-y-4" @submit.prevent="submitPreferences">
              <UFormField
                label="Display theme"
                description="Applies across all devices after saving."
                :error="fieldError('theme', preferencesForm.errors)">
                <USelect v-model="preferencesForm.theme" :items="themeOptions" class="w-full" />
              </UFormField>

              <UFormField label="Notification sound">
                <USwitch
                  v-model="preferencesForm.sound_enabled"
                  label="Play a chime when a notification appears"
                  description="A preview plays when enabled." />
              </UFormField>

              <div class="flex justify-end">
                <UButton
                  type="submit"
                  color="primary"
                  label="Save preferences"
                  icon="i-bi-check2"
                  :loading="preferencesForm.processing" />
              </div>
            </form>
          </UCard>
        </div>
      </template>
    </UDashboardPanel>
  </AppLayout>
</template>
