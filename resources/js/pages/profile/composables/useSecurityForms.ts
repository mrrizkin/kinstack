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

/** A two-factor action that requires the account password first. */
export type TwoFactorAction = "disable" | "regenerate";

export function useSecurityForms(props: SecurityFormsProps) {
  const { message: fieldError, formError } = useServerErrors();

  const passwordForm = useForm({
    current_password: "",
    new_password: "",
    confirm_password: "",
  });
  const passwordDialogOpen = ref(false);

  function openPasswordDialog() {
    passwordForm.reset();
    passwordForm.clearErrors();
    passwordDialogOpen.value = true;
  }

  function submitPassword() {
    passwordForm.post("/profile/password", {
      preserveScroll: true,
      onSuccess: () => {
        passwordForm.reset();
        passwordDialogOpen.value = false;
      },
    });
  }

  const pendingSession = ref<ActiveSession | null>(null);
  const revokingSession = ref(false);
  const pendingRevokeOthers = ref(false);
  const revokingOthers = ref(false);

  const beginningTwoFactor = ref(false);
  const enrollDialogOpen = ref(false);
  const twoFactorForm = useForm({ code: "" });
  const twoFactorPasswordForm = useForm({ password: "" });
  const twoFactorAction = ref<TwoFactorAction | null>(null);
  const qrDataUrl = ref("");

  watch(
    () => props.two_factor.uri,
    async (uri) => {
      qrDataUrl.value = uri ? await QRCode.toDataURL(uri, { margin: 1, width: 220 }) : "";
    },
    { immediate: true },
  );

  function openEnrollDialog() {
    twoFactorForm.reset();
    twoFactorForm.clearErrors();
    enrollDialogOpen.value = true;
    // A pending secret is reused so the dialog shows the existing QR code.
    if (!props.two_factor.enabled && !props.two_factor.pending_secret) {
      beginTwoFactor();
    }
  }

  function closeEnrollDialog(open: boolean) {
    if (!open) enrollDialogOpen.value = false;
  }

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
      onSuccess: () => {
        twoFactorForm.reset();
        enrollDialogOpen.value = false;
      },
    });
  }

  function openTwoFactorAction(action: TwoFactorAction) {
    twoFactorPasswordForm.reset();
    twoFactorPasswordForm.clearErrors();
    twoFactorAction.value = action;
  }

  function closeTwoFactorAction(open: boolean) {
    if (open) return;
    twoFactorAction.value = null;
    twoFactorPasswordForm.reset();
  }

  function confirmTwoFactorAction() {
    const action = twoFactorAction.value;
    if (!action || twoFactorPasswordForm.processing) return;
    const url = action === "disable" ? "/profile/two-factor/disable" : "/profile/two-factor/recovery-codes";
    twoFactorPasswordForm.post(url, {
      preserveScroll: true,
      onSuccess: () => {
        twoFactorAction.value = null;
        twoFactorPasswordForm.reset();
      },
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
    passwordDialogOpen,
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
    fieldError,
    formError,
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
  };
}
