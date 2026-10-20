import { router, useForm } from "@inertiajs/vue3";
import QRCode from "qrcode";
import { onMounted, ref, watch } from "vue";

import { parseThemePreference } from "@/lib/contracts/preferences";
import { useServerErrors } from "@/lib/errors";
import { useNotificationSound } from "@/lib/sound";
import { useTheme } from "@/lib/theme";

import type { ActiveSession, Preferences, Profile, Session, TwoFactor, ThemeOption } from "../types";

interface ProfileProps {
  profile: Profile;
  preferences: Preferences;
  session: Session;
  sessions: ActiveSession[];
  two_factor: TwoFactor;
}

export function useProfileForms(props: ProfileProps) {
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
    theme: parseThemePreference(props.preferences.theme),
    sound_enabled: props.preferences.sound_enabled,
  });

  const themeOptions: ThemeOption[] = [
    { label: "Light", value: "light", icon: "i-bi-sun" },
    { label: "Dark", value: "dark", icon: "i-bi-moon" },
    { label: "Follow system", value: "system", icon: "i-bi-display" },
  ];

  // Apply the stored preferences on visit so another device picks them up.
  onMounted(() => {
    if (preference.value !== props.preferences.theme) {
      setPreference(parseThemePreference(props.preferences.theme));
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
  const revokingSession = ref(false);
  const pendingRevokeOthers = ref(false);
  const revokingOthers = ref(false);
  const beginningTwoFactor = ref(false);
  const pendingDisableTwoFactor = ref(false);
  const disablingTwoFactor = ref(false);

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
    if (beginningTwoFactor.value) return;
    router.post(`/profile/two-factor/begin`, undefined, {
      preserveScroll: true,
      onStart: () => (beginningTwoFactor.value = true),
      onFinish: () => (beginningTwoFactor.value = false),
    });
  }

  function confirmTwoFactor() {
    twoFactorForm.post(`/profile/two-factor/confirm`, {
      preserveScroll: true,
      onSuccess: () => twoFactorForm.reset(),
    });
  }

  function disableTwoFactor() {
    pendingDisableTwoFactor.value = true;
  }

  function confirmDisableTwoFactor() {
    if (disablingTwoFactor.value) return;
    twoFactorPasswordForm.post(`/profile/two-factor/disable`, {
      preserveScroll: true,
      onStart: () => (disablingTwoFactor.value = true),
      onFinish: () => {
        disablingTwoFactor.value = false;
        pendingDisableTwoFactor.value = false;
      },
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
    if (!target || revokingSession.value) return;
    router.post(`/profile/sessions/${target.public_id}/revoke`, undefined, {
      preserveScroll: true,
      onStart: () => (revokingSession.value = true),
      onFinish: () => {
        revokingSession.value = false;
        pendingSession.value = null;
      },
    });
  }

  function revokeOtherSessions() {
    pendingRevokeOthers.value = true;
  }

  function confirmRevokeOtherSessions() {
    if (revokingOthers.value) return;
    router.post(`/profile/sessions/revoke-others`, undefined, {
      preserveScroll: true,
      onStart: () => (revokingOthers.value = true),
      onFinish: () => {
        revokingOthers.value = false;
        pendingRevokeOthers.value = false;
      },
    });
  }

  return {
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
  };
}
