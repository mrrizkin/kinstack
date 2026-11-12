import { router } from "@inertiajs/vue3";
import QRCode from "qrcode";
import { computed, ref, watch } from "vue";

import { dialogHooks, formView, openModel, subjectModel, useAction, useConfirmation, useDialog } from "@/lib/dialog";
import { useServerErrors } from "@/lib/errors";
import { queryString } from "@/lib/query";

import type { ActiveSession, Profile, TwoFactor } from "../types";

interface SecurityFormsProps {
  profile: Profile;
  sessions: ActiveSession[];
  sessions_page?: { page: number };
  two_factor: TwoFactor;
}

/** A two-factor action that requires the account password first. */
export type TwoFactorAction = "disable" | "regenerate";

interface PasswordValues {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

export function useSecurityForms(props: SecurityFormsProps) {
  const { message: fieldError, formError } = useServerErrors();
  const pageQuery = () =>
    queryString({ page: (props.sessions_page?.page ?? 1) > 1 ? props.sessions_page?.page : undefined });

  // ---- Change password ----
  const password = useDialog<null, PasswordValues>({
    label: "password change",
    initialValues: () => ({ current_password: "", new_password: "", confirm_password: "" }),
  });
  const passwordForm = formView(password);
  const passwordDialogOpen = openModel(password);

  function openPasswordDialog() {
    passwordDialogOpen.value = true;
  }

  function submitPassword() {
    if (password.isSaving.value) return;
    const { current_password, new_password, confirm_password } = password.values;
    router.post(
      "/profile/password",
      { current_password, new_password, confirm_password },
      { preserveScroll: true, ...dialogHooks(password) },
    );
  }

  // ---- Sessions ----
  const session = useConfirmation<ActiveSession>({
    label: "revoke session",
    idOf: (target) => target.public_id,
  });
  const pendingSession = subjectModel(session);
  const revokingSession = session.isRunning;

  function confirmRevokeSession() {
    const target = session.subject.value;
    if (!target || session.isRunning.value) return;
    session.confirm();
    router.post(`/profile/sessions/${target.public_id}/revoke${pageQuery()}`, undefined, {
      preserveScroll: true,
      onFinish: () => session.done(),
    });
  }

  // A confirmation without a subject, so the shared machine still owns the
  // open/busy pair.
  const revokeOthers = useDialog<null, Record<string, never>>({
    label: "revoke other sessions",
    initialValues: () => ({}),
  });
  const pendingRevokeOthers = openModel(revokeOthers);
  const revokingOthers = revokeOthers.isSaving;

  function revokeOtherSessions() {
    pendingRevokeOthers.value = true;
  }

  function confirmRevokeOtherSessions() {
    if (revokeOthers.isSaving.value) return;
    router.post(`/profile/sessions/revoke-others${pageQuery()}`, undefined, {
      preserveScroll: true,
      ...dialogHooks(revokeOthers),
    });
  }

  // ---- Two-factor enrolment ----
  const begin = useAction();
  const beginningTwoFactor = begin.isRunning;
  const enroll = useDialog<null, { code: string }>({
    label: "two-factor enrolment",
    initialValues: () => ({ code: "" }),
  });
  const twoFactorForm = formView(enroll);

  const enrollDialogOpen = computed({
    get: () => enroll.isOpen.value,
    set: (open: boolean) => {
      if (!open) {
        enroll.close();
        return;
      }
      if (enroll.isOpen.value) return;
      enroll.open(null);
      // A pending secret is reused so the dialog shows the existing QR code.
      if (!props.two_factor.enabled && !props.two_factor.pending_secret) {
        beginTwoFactor();
      }
    },
  });

  function beginTwoFactor() {
    if (!begin.start()) return;
    router.post("/profile/two-factor/begin", undefined, {
      preserveScroll: true,
      onFinish: () => begin.settled(),
    });
  }

  function openEnrollDialog() {
    enrollDialogOpen.value = true;
  }

  function closeEnrollDialog(open: boolean) {
    if (!open) enrollDialogOpen.value = false;
  }

  function confirmTwoFactor() {
    if (enroll.isSaving.value) return;
    const { code } = enroll.values;
    router.post("/profile/two-factor/confirm", { code }, { preserveScroll: true, ...dialogHooks(enroll) });
  }

  // ---- Two-factor actions that need the password ----
  const action = useDialog<TwoFactorAction, { password: string }>({
    label: "two-factor action",
    initialValues: () => ({ password: "" }),
    idOf: (kind) => kind,
  });
  const twoFactorAction = subjectModel(action);
  const twoFactorPasswordForm = formView(action);
  const qrDataUrl = ref("");

  watch(
    () => props.two_factor.uri,
    async (uri) => {
      qrDataUrl.value = uri ? await QRCode.toDataURL(uri, { margin: 1, width: 220 }) : "";
    },
    { immediate: true },
  );

  function openTwoFactorAction(kind: TwoFactorAction) {
    if (action.isOpen.value) return;
    action.open(kind);
  }

  function closeTwoFactorAction(open: boolean) {
    if (open) return;
    action.close();
  }

  function confirmTwoFactorAction() {
    const kind = action.subject.value;
    if (!kind || action.isSaving.value) return;
    const url = kind === "disable" ? "/profile/two-factor/disable" : "/profile/two-factor/recovery-codes";
    router.post(url, { password: action.values.password }, { preserveScroll: true, ...dialogHooks(action) });
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
