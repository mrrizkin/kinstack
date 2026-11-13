import { computed, shallowRef, type ComputedRef, type ShallowRef, type WritableComputedRef } from "vue";

import { errorMap } from "./errors";

/**
 * Dialog lifecycle. `closed` is terminal: open, busy, and errors are one state.
 *
 * | from     | event      | to      |
 * | closed   | open       | editing |
 * | editing  | save       | saving  |
 * | editing  | close      | closed  |
 * | editing  | change/invalid | editing |
 * | saving   | saved/committed/failed | editing/closed |
 * | saving   | close      | saving  |
 * | saving   | change/invalid/save | saving  |
 */
export type DialogErrors = Record<string, string>;

type DialogState<S, V> =
  | { status: "closed" }
  | { status: "editing"; subject: S | null; values: V; errors: DialogErrors }
  | { status: "saving"; subject: S | null; values: V; errors: DialogErrors };

type DialogEvent<S, V> =
  | { type: "open"; subject: S | null; values: V }
  | { type: "change"; values: V }
  | { type: "invalid"; errors: DialogErrors }
  | { type: "save" }
  | { type: "saved" }
  | { type: "committed" }
  | { type: "failed"; errors: DialogErrors }
  | { type: "close" };

function transition<S, V>(state: DialogState<S, V>, event: DialogEvent<S, V>, label: string): DialogState<S, V> {
  switch (event.type) {
    case "open":
      if (state.status !== "closed") {
        throw new Error(`${label}: open is not allowed from ${state.status}`);
      }
      return { status: "editing", subject: event.subject, values: event.values, errors: {} };
    case "change":
      return state.status === "closed" ? state : { ...state, values: event.values };
    case "invalid":
      return state.status === "closed" ? state : { ...state, errors: event.errors };
    case "save":
      return state.status === "editing"
        ? { status: "saving", subject: state.subject, values: state.values, errors: {} }
        : state;
    case "saved":
      return state.status === "saving"
        ? { status: "editing", subject: state.subject, values: state.values, errors: {} }
        : state;
    case "committed":
      return state.status === "saving" ? { status: "closed" } : state;
    case "failed":
      return state.status === "saving"
        ? { status: "editing", subject: state.subject, values: state.values, errors: event.errors }
        : state;
    case "close":
      return state.status === "editing" ? { status: "closed" } : state;
  }
}

export interface DialogOptions<S, V> {
  /** Prefix for thrown errors, e.g. "role change". */
  label: string;
  initialValues: (subject: S | null) => V;
  idOf?: (subject: S) => string;
  assertSubject?: (subject: S) => void;
}

export interface Dialog<S, V> {
  keys: readonly string[];
  subject: ComputedRef<S | null>;
  isOpen: ComputedRef<boolean>;
  isSaving: ComputedRef<boolean>;
  busyId: ComputedRef<string | null>;
  /** Writable view with stable identity across renders. */
  values: V;
  errors: ComputedRef<DialogErrors>;
  open(subject: S | null): void;
  close(): void;
  set(key: string, value: unknown): void;
  invalid(errors: DialogErrors): void;
  save(): void;
  /** Saved, dialog stays open. */
  saved(): void;
  /** Saved, dialog closes. */
  committed(): void;
  failed(errors?: DialogErrors): void;
}

function assertInvariants<S, V>(state: DialogState<S, V>, options: DialogOptions<S, V>): void {
  if (state.status === "closed") {
    return;
  }
  if (state.subject === null) {
    return;
  }
  options.assertSubject?.(state.subject);
  if (options.idOf && options.idOf(state.subject) === "") {
    throw new Error(`${options.label}: an open subject must be identified`);
  }
}

export function useDialog<S, V extends object>(options: DialogOptions<S, V>): Dialog<S, V> {
  const values: V = options.initialValues(null);
  const keys = Object.keys(values);
  const state: ShallowRef<DialogState<S, V>> = shallowRef({ status: "closed" });

  function send(event: DialogEvent<S, V>) {
    const next = transition(state.value, event, options.label);
    assertInvariants(next, options);
    state.value = next;
  }

  function currentValues(): V {
    return state.value.status === "closed" ? options.initialValues(null) : state.value.values;
  }

  function set(key: string, value: unknown): void {
    if (state.value.status === "closed") {
      throw new Error(`${options.label}: cannot change ${key} while closed`);
    }
    const next = { ...state.value.values };
    Reflect.set(next, key, value);
    send({ type: "change", values: next });
  }

  for (const key of keys) {
    Object.defineProperty(values, key, {
      enumerable: true,
      configurable: true,
      get: () => Reflect.get(currentValues(), key),
      set: (value: unknown) => set(key, value),
    });
  }

  return {
    keys,
    subject: computed(() => (state.value.status === "closed" ? null : state.value.subject)),
    isOpen: computed(() => state.value.status !== "closed"),
    isSaving: computed(() => state.value.status === "saving"),
    busyId: computed(() => {
      if (state.value.status !== "saving" || state.value.subject === null) {
        return null;
      }
      return options.idOf?.(state.value.subject) ?? null;
    }),
    values,
    errors: computed(() => (state.value.status === "closed" ? {} : state.value.errors)),
    open: (subject) => send({ type: "open", subject, values: options.initialValues(subject) }),
    close: () => send({ type: "close" }),
    set,
    invalid: (errors) => send({ type: "invalid", errors }),
    save: () => send({ type: "save" }),
    saved: () => send({ type: "saved" }),
    committed: () => send({ type: "committed" }),
    failed: (errors = {}) => send({ type: "failed", errors }),
  };
}

/**
 * Confirmation lifecycle. `start` runs immediately (activating a user);
 * `open` + `confirm` asks first (deleting). `closed` is terminal.
 *
 * | from       | event   | to         |
 * | closed     | open    | confirming |
 * | closed     | start   | running    |
 * | confirming | confirm | running    |
 * | confirming | close   | closed     |
 * | running    | done    | closed     |
 * | running    | close/confirm | running    |
 */
type ConfirmationState<S> =
  | { status: "closed" }
  | { status: "confirming"; subject: S }
  | { status: "running"; subject: S };

type ConfirmationEvent<S> =
  | { type: "open"; subject: S }
  | { type: "start"; subject: S }
  | { type: "confirm" }
  | { type: "done" }
  | { type: "close" };

function transitionConfirmation<S>(
  state: ConfirmationState<S>,
  event: ConfirmationEvent<S>,
  label: string,
): ConfirmationState<S> {
  switch (event.type) {
    case "open":
    case "start":
      if (state.status !== "closed") {
        throw new Error(`${label}: ${event.type} is not allowed from ${state.status}`);
      }
      return event.type === "open"
        ? { status: "confirming", subject: event.subject }
        : { status: "running", subject: event.subject };
    case "confirm":
      return state.status === "confirming" ? { status: "running", subject: state.subject } : state;
    case "done":
      return state.status === "running" ? { status: "closed" } : state;
    case "close":
      return state.status === "confirming" ? { status: "closed" } : state;
  }
}

export interface ConfirmationOptions<S> {
  label: string;
  idOf: (subject: S) => string;
  assertSubject?: (subject: S) => void;
}

export interface Confirmation<S> {
  subject: ComputedRef<S | null>;
  isOpen: ComputedRef<boolean>;
  isConfirming: ComputedRef<boolean>;
  isRunning: ComputedRef<boolean>;
  busyId: ComputedRef<string | null>;
  open(subject: S): void;
  start(subject: S): void;
  confirm(): void;
  done(): void;
  close(): void;
}

export function useConfirmation<S>(options: ConfirmationOptions<S>): Confirmation<S> {
  const state: ShallowRef<ConfirmationState<S>> = shallowRef({ status: "closed" });

  function send(event: ConfirmationEvent<S>) {
    const next = transitionConfirmation(state.value, event, options.label);
    if (next.status !== "closed") {
      options.assertSubject?.(next.subject);
      if (options.idOf(next.subject) === "") {
        throw new Error(`${options.label}: an open subject must be identified`);
      }
    }
    state.value = next;
  }

  return {
    subject: computed(() => (state.value.status === "closed" ? null : state.value.subject)),
    isOpen: computed(() => state.value.status !== "closed"),
    isConfirming: computed(() => state.value.status === "confirming"),
    isRunning: computed(() => state.value.status === "running"),
    busyId: computed(() => (state.value.status === "running" ? options.idOf(state.value.subject) : null)),
    open: (subject) => send({ type: "open", subject }),
    start: (subject) => send({ type: "start", subject }),
    confirm: () => send({ type: "confirm" }),
    done: () => send({ type: "done" }),
    close: () => send({ type: "close" }),
  };
}

/**
 * One-shot action lifecycle for work with no dialog: `start` returns false while
 * running so a double click fires once.
 */
export interface Action {
  isRunning: ComputedRef<boolean>;
  start(): boolean;
  settled(): void;
}

export function useAction(): Action {
  const running = shallowRef(false);
  return {
    isRunning: computed(() => running.value),
    start: () => {
      if (running.value) {
        return false;
      }
      running.value = true;
      return true;
    },
    settled: () => {
      running.value = false;
    },
  };
}

interface Openable<S> {
  subject: ComputedRef<S | null>;
  open(subject: S | null): void;
  close(): void;
}

/** `v-model` over a dialog's subject; repeated assignments are ignored. */
export function subjectModel<S>(dialog: Openable<S>): WritableComputedRef<S | null> {
  return computed({
    get: () => dialog.subject.value,
    set: (subject) => {
      if (subject === null) {
        dialog.close();
        return;
      }
      if (dialog.subject.value === subject) {
        return;
      }
      dialog.open(subject);
    },
  });
}

interface OpenableVoid {
  isOpen: ComputedRef<boolean>;
  open(subject: null): void;
  close(): void;
}

/** `v-model:open` over a subject-less dialog. */
export function openModel(dialog: OpenableVoid): WritableComputedRef<boolean> {
  return computed({
    get: () => dialog.isOpen.value,
    set: (open) => {
      if (open === dialog.isOpen.value) {
        return;
      }
      if (open) {
        dialog.open(null);
        return;
      }
      dialog.close();
    },
  });
}

export type FormView<V> = V & { errors: DialogErrors; processing: boolean };

export function formView<S, V extends object>(dialog: Dialog<S, V>): FormView<V> {
  const view: V = dialog.values;
  Object.defineProperty(view, "errors", { enumerable: true, get: () => dialog.errors.value });
  Object.defineProperty(view, "processing", { enumerable: true, get: () => dialog.isSaving.value });
  return view as FormView<V>;
}

export interface VisitHooks {
  onStart: () => void;
  onSuccess: () => void;
  onError: (errors: Record<string, unknown>) => void;
  onFinish: () => void;
}

/** `onFinish` is the safety net for a cancelled visit. */
export function dialogHooks<S, V extends object>(
  dialog: Dialog<S, V>,
  options: { keepOpen?: boolean } = {},
): VisitHooks {
  return {
    onStart: () => dialog.save(),
    onSuccess: () => (options.keepOpen ? dialog.saved() : dialog.committed()),
    onError: (errors) => dialog.failed(errorMap(errors)),
    onFinish: () => dialog.failed(),
  };
}
