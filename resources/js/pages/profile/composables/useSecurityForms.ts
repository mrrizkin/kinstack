import { router, useForm } from "@inertiajs/vue3";
import QRCode from "qrcode";
import { ref, watch } from "vue";

import { useServerErrors } from "@/lib/errors";

import type { ActiveSession, Profile, TwoFactor } from "../types";

interface SecurityFormsProps {
  profile: Profile;
  sessions: ActiveSession[];
  two_factor: TwoFactor;
}

export function useSecurityForms(props: SecurityFormsProps) {
  const passwordForm = useForm({
    current_password: "",
    new_password: "",
    confirm_password: "",
  });

  function submitPassword() {
    passwordForm.post("/profile/password", {
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
    router.post("/profile/two-factor/begin", undefined, {
      preserveScroll: true,
      onStart: () => (beginningTwoFactor.value = true),
      onFinish: () => (beginningTwoFactor.value = false),
    });
  }

  function confirmTwoFactor() {
    twoFactorForm.post("/profile/two-factor/confirm", {
      preserveScroll: true,
      onSuccess: () => twoFactorForm.reset(),
    });
  }

  function disableTwoFactor() {
    pendingDisableTwoFactor.value = true;
  }

  function confirmDisableTwoFactor() {
    if (disablingTwoFactor.value) return;
    twoFactorPasswordForm.post("/profile/two-factor/disable", {
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
    twoFactorPasswordForm.post("/profile/two-factor/recovery-codes", {
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
    router.post("/profile/sessions/revoke-others", undefined, {
      preserveScroll: true,
      onStart: () => (revokingOthers.value = true),
      onFinish: () => {
        revokingOthers.value = false;
        pendingRevokeOthers.value = false;
      },
    });
  }

  return {
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
  };
}
