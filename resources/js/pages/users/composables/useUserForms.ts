import { router, useForm } from "@inertiajs/vue3";
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, ref } from "vue";

import { useServerErrors } from "@/lib/errors";
import { icons } from "@/lib/icons";
import { PERMISSIONS, usePermissions } from "@/lib/permissions";
import { useRoleChange } from "@/lib/role-change";

import type { Role, User } from "../columns";

interface UsersProps {
  users: User[];
  roles: Role[];
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

  const showCreate = ref(false);
  const createForm = useForm({
    username: "",
    email: "",
    display_name: "",
    password: "",
    role: props.roles[0]?.code ?? "",
  });

  function submitCreate() {
    createForm.post(usersURL(), {
      onSuccess: () => {
        createForm.reset();
        showCreate.value = false;
      },
    });
  }

  const showInvite = ref(false);

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

  const inviteForm = useForm({
    email: "",
    username: "",
    display_name: "",
    role: props.roles[0]?.code ?? "",
  });

  function submitInvite() {
    inviteForm.post(usersURL("/invite"), {
      onSuccess: () => {
        inviteForm.reset();
        showInvite.value = false;
      },
    });
  }

  const busyUserId = ref<string | null>(null);
  const pendingToggle = ref<User | null>(null);
  const togglingActive = ref(false);

  function applyActive(user: User, isActive: boolean) {
    router.post(
      usersURL(`/${user.public_id}/active`),
      { is_active: isActive },
      {
        preserveScroll: true,
        onStart: () => (busyUserId.value = user.public_id),
        onFinish: () => (busyUserId.value = null),
      },
    );
  }

  function toggleActive(user: User) {
    if (busyUserId.value) return;
    if (!user.is_active) {
      applyActive(user, true);
      return;
    }
    pendingToggle.value = user;
  }

  function confirmToggleActive() {
    const user = pendingToggle.value;
    if (!user || togglingActive.value) return;
    togglingActive.value = true;
    router.post(
      usersURL(`/${user.public_id}/active`),
      { is_active: false },
      {
        preserveScroll: true,
        onStart: () => (busyUserId.value = user.public_id),
        onFinish: () => {
          togglingActive.value = false;
          busyUserId.value = null;
          pendingToggle.value = null;
        },
      },
    );
  }

  // ---- Change role (confirmed through a modal, never inline) ----
  const roleChange = useRoleChange();
  const roleTarget = roleChange.target;
  const roleOpen = roleChange.isOpen;
  const roleSaving = roleChange.isSaving;
  const roleValue = roleChange.role;

  function requestRoleChange(user: User) {
    if (busyUserId.value) return;
    roleChange.open({ id: user.public_id, name: user.display_name, currentRole: user.role });
  }

  function closeRoleChange() {
    roleChange.close();
  }

  function submitRoleChange() {
    const target = roleTarget.value;
    if (!target) return;
    const user = props.users.find((candidate) => candidate.public_id === target.id);
    if (!user) return;
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
        onStart: () => (busyUserId.value = user.public_id),
        onFinish: () => (busyUserId.value = null),
        onSuccess: () => roleChange.committed(),
        onError: () => roleChange.failed(),
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

  const passwordTarget = ref<User | null>(null);
  const passwordForm = useForm({ password: "" });

  function resetPassword(user: User) {
    passwordTarget.value = user;
    passwordForm.reset();
    passwordForm.clearErrors();
  }

  function confirmResetPassword() {
    const user = passwordTarget.value;
    if (!user) return;
    passwordForm.post(usersURL(`/${user.public_id}/password`), {
      onSuccess: () => {
        passwordTarget.value = null;
        passwordForm.reset();
      },
    });
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
