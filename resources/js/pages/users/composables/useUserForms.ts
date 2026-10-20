import { router, useForm } from "@inertiajs/vue3";
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, ref } from "vue";

import { useServerErrors } from "@/lib/errors";
import { PERMISSIONS, usePermissions } from "@/lib/permissions";

import type { Role, User } from "../columns";

interface UsersProps {
  users: User[];
  roles: Role[];
}

export function useUserForms(props: UsersProps) {
  const roleOptions = computed(() => props.roles.map((role) => ({ label: role.name, value: role.code })));
  const usersURL = (suffix = "") => `/users${suffix}`;

  const { formError } = useServerErrors();
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
        icon: "i-bi-shield-lock",
        to: "/access",
      });
    }
    if (can(PERMISSIONS.usersManage)) {
      actions.push({
        label: "Invite",
        icon: "i-bi-envelope-plus",
        onSelect: () => {
          showInvite.value = !showInvite.value;
        },
      });
      actions.push({
        label: showCreate.value ? "Close" : "Add user",
        icon: showCreate.value ? "i-bi-x-lg" : "i-bi-person-plus",
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

  function toggleActive(user: User) {
    router.post(usersURL(`/${user.public_id}/active`), { is_active: !user.is_active });
  }

  function changeRole(user: User, role: string) {
    router.put(usersURL(`/${user.public_id}`), {
      email: user.email ?? "",
      display_name: user.display_name,
      role,
    });
  }

  function userMenuItems(user: User): DropdownMenuItem[] {
    return [
      {
        label: "Reset password",
        icon: "i-bi-key",
        onSelect: () => resetPassword(user),
      },
      {
        label: user.is_active ? "Deactivate" : "Activate",
        icon: user.is_active ? "i-bi-person-x" : "i-bi-person-check",
        color: user.is_active ? "error" : "success",
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
    showCreate,
    createForm,
    submitCreate,
    showInvite,
    headerActions,
    inviteForm,
    submitInvite,
    toggleActive,
    changeRole,
    userMenuItems,
    passwordTarget,
    passwordForm,
    resetPassword,
    confirmResetPassword,
  };
}
