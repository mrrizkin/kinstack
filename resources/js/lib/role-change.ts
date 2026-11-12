import { computed, ref, type ComputedRef, type WritableComputedRef } from "vue";

/**
 * Role-change dialog lifecycle.
 *
 * | from    | event     | to      |
 * | closed  | open      | editing |
 * | editing | select    | editing |
 * | editing | save      | saving  |
 * | editing | close     | closed  |
 * | saving  | committed | closed  |
 * | saving  | failed    | editing |
 * | saving  | close     | saving  |
 * | saving  | select    | saving  |
 * | saving  | save      | saving  |
 *
 * `closed` is terminal. The subject only exists outside `closed`, so "which
 * user is being edited" is not a separate nullable ref that must agree with an
 * "is the dialog open" boolean.
 */
export interface RoleChangeSubject {
  id: string;
  name: string;
  currentRole: string;
}

type RoleChangeState =
  | { status: "closed" }
  | { status: "editing"; subject: RoleChangeSubject; role: string }
  | { status: "saving"; subject: RoleChangeSubject; role: string };

type RoleChangeEvent =
  | { type: "open"; subject: RoleChangeSubject }
  | { type: "select"; role: string }
  | { type: "save" }
  | { type: "committed" }
  | { type: "failed" }
  | { type: "close" };

function transition(state: RoleChangeState, event: RoleChangeEvent): RoleChangeState {
  switch (event.type) {
    case "open":
      if (state.status !== "closed") {
        throw new Error(`role change: open is not allowed from ${state.status}`);
      }
      return { status: "editing", subject: event.subject, role: event.subject.currentRole };
    case "close":
      return state.status === "editing" ? { status: "closed" } : state;
    case "select":
      return state.status === "editing" ? { ...state, role: event.role } : state;
    case "save":
      return state.status === "editing" ? { status: "saving", subject: state.subject, role: state.role } : state;
    case "committed":
      return state.status === "saving" ? { status: "closed" } : state;
    case "failed":
      return state.status === "saving" ? { status: "editing", subject: state.subject, role: state.role } : state;
  }
}

function assertInvariants(state: RoleChangeState): void {
  if (state.status === "closed") {
    return;
  }
  if (state.subject.id === "" || state.subject.name === "") {
    throw new Error("role change: an open subject must be identified");
  }
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
  const state = ref<RoleChangeState>({ status: "closed" });

  function send(event: RoleChangeEvent) {
    const next = transition(state.value, event);
    assertInvariants(next);
    state.value = next;
  }

  return {
    target: computed(() => (state.value.status === "closed" ? null : state.value.subject)),
    isOpen: computed(() => state.value.status !== "closed"),
    isSaving: computed(() => state.value.status === "saving"),
    busyId: computed(() => (state.value.status === "saving" ? state.value.subject.id : null)),
    role: computed({
      get: () => (state.value.status === "closed" ? "" : state.value.role),
      set: (role: string) => send({ type: "select", role }),
    }),
    open: (subject: RoleChangeSubject) => send({ type: "open", subject }),
    close: () => send({ type: "close" }),
    save: () => send({ type: "save" }),
    committed: () => send({ type: "committed" }),
    failed: () => send({ type: "failed" }),
  };
}
