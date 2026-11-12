import { router } from "@inertiajs/vue3";
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed } from "vue";

import { dialogHooks, formView, openModel, subjectModel, useConfirmation, useDialog } from "@/lib/dialog";
import { useServerErrors } from "@/lib/errors";
import { icons } from "@/lib/icons";
import { PERMISSIONS, usePermissions } from "@/lib/permissions";
import { useRoleChange } from "@/lib/role-change";

import type { Role, User } from "../columns";

interface UsersProps {
  users: User[];
  roles: Role[];
}

interface CreateValues {
  username: string;
  email: string;
  display_name: string;
  password: string;
  role: string;
}

interface InviteValues {
  email: string;
  username: string;
  display_name: string;
  role: string;
}

export function useUserForms(props: UsersProps) {
  const roleOptions = computed(() => props.roles.map((role) => ({ label: role.name, value: role.code })));

  function roleLabel(code: string): string {
    return roleOptions.value.find((option) => option.value === code)?.label ?? code;
  }
  const usersURL = (suffix = "") => `/users${suffix}`;

  const { formError, message } = useServerErrors();
  const actionError = computed(() => message("is_active") || message("role") || message("form"));
  const { can } = usePermissions();

  // ---- Create user ----
  const createUser = useDialog<null, CreateValues>({
    label: "create user",
    initialValues: () => ({
      username: "",
      email: "",
      display_name: "",
      password: "",
      role: props.roles[0]?.code ?? "",
    }),
  });

  const showCreate = openModel(createUser);
  const createForm = formView(createUser);

  function submitCreate() {
    const { username, email, display_name, password, role } = createUser.values;
    router.post(usersURL(), { username, email, display_name, password, role }, dialogHooks(createUser));
  }

  // ---- Invite user ----
  const invite = useDialog<null, InviteValues>({
    label: "invite user",
    initialValues: () => ({
      email: "",
      username: "",
      display_name: "",
      role: props.roles[0]?.code ?? "",
    }),
  });

  const showInvite = openModel(invite);
  const inviteForm = formView(invite);

  function submitInvite() {
    const { email, username, display_name, role } = invite.values;
    router.post(usersURL("/invite"), { email, username, display_name, role }, dialogHooks(invite));
  }

  const headerActions = computed<DropdownMenuItem[]>(() => {
    const actions: DropdownMenuItem[] = [];
    if (can(PERMISSIONS.accessManage)) {
      actions.push({
        label: "Access",
        icon: icons.access,
        to: "/access",
      });
    }
    if (can(PERMISSIONS.usersManage)) {
      actions.push({
        label: "Invite",
        icon: icons.invite,
        onSelect: () => {
          showInvite.value = !showInvite.value;
        },
      });
      actions.push({
        label: showCreate.value ? "Close" : "Add user",
        icon: showCreate.value ? icons.close : icons.userAdd,
        onSelect: () => {
          showCreate.value = !showCreate.value;
        },
      });
    }
    return actions;
  });

  // ---- Activate / deactivate ----
  // Activation runs immediately; deactivation asks first. Both are one state:
  // idle, asking, or running.
  const activity = useConfirmation<User>({ label: "user activity", idOf: (user) => user.public_id });
  const pendingToggle = subjectModel(activity);
  const togglingActive = activity.isRunning;

  function applyActivation(user: User, isActive: boolean) {
    router.post(
      usersURL(`/${user.public_id}/active`),
      { is_active: isActive },
      {
        preserveScroll: true,
        onFinish: () => activity.done(),
      },
    );
  }

  function toggleActive(user: User) {
    if (activity.isRunning.value) {
      return;
    }
    if (!user.is_active) {
      activity.start(user);
      applyActivation(user, true);
      return;
    }
    activity.open(user);
  }

  function confirmToggleActive() {
    const user = activity.subject.value;
    if (!user || activity.isRunning.value) {
      return;
    }
    activity.confirm();
    applyActivation(user, false);
  }

  // ---- Change role (confirmed through a modal, never inline) ----
  const roleChange = useRoleChange();
  const roleTarget = roleChange.target;
  const roleOpen = roleChange.isOpen;
  const roleSaving = roleChange.isSaving;
  const roleValue = roleChange.role;
  const busyUserId = computed(() => roleChange.busyId.value ?? activity.busyId.value);

  function requestRoleChange(user: User) {
    if (busyUserId.value) {
      return;
    }
    roleChange.open({ id: user.public_id, name: user.display_name, currentRole: user.role });
  }

  function closeRoleChange() {
    roleChange.close();
  }

  function submitRoleChange() {
    const target = roleTarget.value;
    if (!target) {
      return;
    }
    const user = props.users.find((candidate) => candidate.public_id === target.id);
    if (!user) {
      return;
    }
    if (roleValue.value === user.role) {
      roleChange.close();
      return;
    }
    roleChange.save();
    router.put(
      usersURL(`/${user.public_id}`),
      {
        email: user.email ?? "",
        display_name: user.display_name,
        role: roleValue.value,
      },
      {
        preserveScroll: true,
        onSuccess: () => roleChange.committed(),
        onError: () => roleChange.failed(),
        onFinish: () => roleChange.failed(),
      },
    );
  }

  function userMenuItems(user: User): DropdownMenuItem[] {
    const busy = busyUserId.value === user.public_id;
    return [
      {
        label: "Change role",
        icon: icons.access,
        disabled: busy,
        onSelect: () => requestRoleChange(user),
      },
      {
        label: "Reset password",
        icon: icons.key,
        disabled: busy,
        onSelect: () => resetPassword(user),
      },
      {
        label: user.is_active ? "Deactivate" : "Activate",
        icon: user.is_active ? icons.userDeactivate : icons.userActivate,
        color: user.is_active ? "error" : "success",
        loading: busy,
        disabled: busy,
        onSelect: () => toggleActive(user),
      },
    ];
  }

  // ---- Reset password ----
  const password = useDialog<User, { password: string }>({
    label: "reset password",
    initialValues: () => ({ password: "" }),
    idOf: (user) => user.public_id,
  });

  const passwordTarget = subjectModel(password);
  const passwordForm = formView(password);

  function resetPassword(user: User) {
    if (password.isOpen.value) {
      return;
    }
    password.open(user);
  }

  function confirmResetPassword() {
    const user = password.subject.value;
    if (!user || password.isSaving.value) {
      return;
    }
    router.post(usersURL(`/${user.public_id}/password`), { password: password.values.password }, dialogHooks(password));
  }

  return {
    roleOptions,
    formError,
    actionError,
    showCreate,
    createForm,
    submitCreate,
    showInvite,
    headerActions,
    inviteForm,
    submitInvite,
    busyUserId,
    pendingToggle,
    togglingActive,
    toggleActive,
    confirmToggleActive,
    roleLabel,
    roleTarget,
    roleOpen,
    roleSaving,
    roleValue,
    requestRoleChange,
    closeRoleChange,
    submitRoleChange,
    userMenuItems,
    passwordTarget,
    passwordForm,
    resetPassword,
    confirmResetPassword,
  };
}
