import { computed, type ComputedRef, type WritableComputedRef } from "vue";

import { useDialog } from "./dialog";

/**
 * The transition table lives in `useDialog`; this adds the role fields and the
 * role-specific invariant.
 */
export interface RoleChangeSubject {
  id: string;
  name: string;
  currentRole: string;
}

export interface RoleChange {
  target: ComputedRef<RoleChangeSubject | null>;
  isOpen: ComputedRef<boolean>;
  isSaving: ComputedRef<boolean>;
  busyId: ComputedRef<string | null>;
  role: WritableComputedRef<string>;
  open(subject: RoleChangeSubject): void;
  close(): void;
  save(): void;
  committed(): void;
  failed(): void;
}

export function useRoleChange(): RoleChange {
  const dialog = useDialog<RoleChangeSubject, { role: string }>({
    label: "role change",
    initialValues: (subject) => ({ role: subject?.currentRole ?? "" }),
    idOf: (subject) => subject.id,
    assertSubject: (subject) => {
      if (subject.id === "" || subject.name === "") {
        throw new Error("role change: an open subject must be identified");
      }
    },
  });

  return {
    target: dialog.subject,
    isOpen: dialog.isOpen,
    isSaving: dialog.isSaving,
    busyId: dialog.busyId,
    role: computed({
      get: () => dialog.values.role,
      set: (role: string) => dialog.set("role", role),
    }),
    open: (subject) => dialog.open(subject),
    close: () => dialog.close(),
    save: () => dialog.save(),
    committed: () => dialog.committed(),
    failed: () => dialog.failed(),
  };
}
