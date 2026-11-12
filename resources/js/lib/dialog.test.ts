import { describe, expect, test } from "bun:test";

import { formView, subjectModel, useAction, useConfirmation, useDialog } from "./dialog";

interface Doc {
  title: string;
  rank: number;
}

const subject = { id: "role-1", name: "Supervisor" };

function makeDialog() {
  return useDialog<typeof subject, Doc>({
    label: "edit doc",
    initialValues: (target) => ({ title: target?.name ?? "", rank: 10 }),
    idOf: (target) => target.id,
  });
}

describe("dialog state machine", () => {
  test("open seeds the values from the subject and records the subject", () => {
    const dialog = makeDialog();
    dialog.open(subject);

    expect(dialog.isOpen.value).toBe(true);
    expect(dialog.isSaving.value).toBe(false);
    expect(dialog.subject.value).toEqual(subject);
    expect(dialog.values.title).toBe("Supervisor");
    expect(dialog.values.rank).toBe(10);
    expect(dialog.errors.value).toEqual({});
  });

  test("values are written through the state machine", () => {
    const dialog = makeDialog();
    dialog.open(subject);
    dialog.values.title = "Night shift";

    expect(dialog.values.title).toBe("Night shift");
    expect(dialog.subject.value).toEqual(subject);
  });

  test("a new open reseeds the values and clears errors", () => {
    const dialog = makeDialog();
    dialog.open(subject);
    dialog.values.title = "Night shift";
    dialog.invalid({ title: "Required" });
    dialog.save();
    dialog.failed({ title: "Taken" });
    dialog.close();

    dialog.open({ id: "role-2", name: "Operator" });
    expect(dialog.values.title).toBe("Operator");
    expect(dialog.errors.value).toEqual({});
  });

  test("saving is the only state that blocks closing", () => {
    const dialog = makeDialog();
    dialog.open(subject);
    dialog.save();

    expect(dialog.isSaving.value).toBe(true);
    expect(dialog.busyId.value).toBe(subject.id);

    dialog.close();
    expect(dialog.isOpen.value).toBe(true);

    dialog.committed();
    expect(dialog.isOpen.value).toBe(false);
    expect(dialog.subject.value).toBeNull();
    expect(dialog.busyId.value).toBeNull();
  });

  test("saved returns to editing and keeps the dialog open", () => {
    const dialog = makeDialog();
    dialog.open(subject);
    dialog.save();
    dialog.saved();

    expect(dialog.isOpen.value).toBe(true);
    expect(dialog.isSaving.value).toBe(false);
    expect(dialog.errors.value).toEqual({});
  });

  test("a failed save returns to editing with the server errors", () => {
    const dialog = makeDialog();
    dialog.open(subject);
    dialog.values.title = "Night shift";
    dialog.save();
    dialog.failed({ title: "Taken" });

    expect(dialog.isOpen.value).toBe(true);
    expect(dialog.isSaving.value).toBe(false);
    expect(dialog.values.title).toBe("Night shift");
    expect(dialog.errors.value).toEqual({ title: "Taken" });
  });

  test("a save clears the errors of the previous attempt", () => {
    const dialog = makeDialog();
    dialog.open(subject);
    dialog.invalid({ title: "Required" });
    expect(dialog.errors.value).toEqual({ title: "Required" });

    dialog.save();
    expect(dialog.errors.value).toEqual({});
  });

  test("opening while open is an illegal transition", () => {
    const dialog = makeDialog();
    dialog.open(subject);

    expect(() => dialog.open(subject)).toThrow("edit doc: open is not allowed from editing");
  });

  test("writing while closed is an illegal transition", () => {
    const dialog = makeDialog();

    expect(() => (dialog.values.title = "x")).toThrow("edit doc: cannot change title while closed");
  });

  test("terminal closed ignores late events", () => {
    const dialog = makeDialog();
    dialog.open(subject);
    dialog.save();
    dialog.committed();
    dialog.committed();
    dialog.saved();
    dialog.failed();

    expect(dialog.isOpen.value).toBe(false);
  });
});

describe("formView", () => {
  test("is a view over the machine, not a copy", () => {
    const dialog = makeDialog();
    const form = formView(dialog);

    dialog.open(subject);
    expect(form.title).toBe("Supervisor");
    expect(form.processing).toBe(false);
    expect(form.errors).toEqual({});

    form.title = "Night shift";
    expect(dialog.values.title).toBe("Night shift");

    dialog.save();
    expect(form.processing).toBe(true);

    dialog.failed({ title: "Taken" });
    expect(form.processing).toBe(false);
    expect(form.errors).toEqual({ title: "Taken" });

    // Stable identity: the component keeps the same object across reopenings.
    const identity = form;
    dialog.close();
    dialog.open({ id: "role-2", name: "Operator" });
    expect(identity.title).toBe("Operator");
  });
});

describe("subjectModel", () => {
  test("opens on a subject and closes on null", () => {
    const dialog = makeDialog();
    const model = subjectModel(dialog);

    expect(model.value).toBeNull();
    model.value = subject;
    expect(dialog.isOpen.value).toBe(true);
    expect(model.value).toEqual(subject);

    model.value = null;
    expect(dialog.isOpen.value).toBe(false);
  });

  test("reassigning the same subject does not trip the illegal-open guard", () => {
    const dialog = makeDialog();
    const model = subjectModel(dialog);
    model.value = subject;

    expect(() => (model.value = subject)).not.toThrow();
  });
});

describe("confirmation state machine", () => {
  const confirmation = () =>
    useConfirmation<{ id: string; name: string }>({ label: "delete", idOf: (target) => target.id });

  const target = { id: "role-1", name: "Supervisor" };

  test("asking is not running, and confirming starts the work", () => {
    const machine = confirmation();
    machine.open(target);

    expect(machine.isConfirming.value).toBe(true);
    expect(machine.isRunning.value).toBe(false);
    expect(machine.busyId.value).toBeNull();

    machine.confirm();
    expect(machine.isConfirming.value).toBe(false);
    expect(machine.isRunning.value).toBe(true);
    expect(machine.busyId.value).toBe(target.id);
  });

  test("running blocks closing until done", () => {
    const machine = confirmation();
    machine.open(target);
    machine.confirm();

    machine.close();
    expect(machine.isRunning.value).toBe(true);

    machine.done();
    expect(machine.isOpen.value).toBe(false);
    expect(machine.subject.value).toBeNull();
  });

  test("start runs immediately for actions that need no question", () => {
    const machine = confirmation();
    machine.start(target);

    expect(machine.isConfirming.value).toBe(false);
    expect(machine.isRunning.value).toBe(true);
  });

  test("closing while asking drops the subject", () => {
    const machine = confirmation();
    machine.open(target);
    machine.close();

    expect(machine.isOpen.value).toBe(false);
    expect(machine.subject.value).toBeNull();
    expect(machine.isRunning.value).toBe(false);
  });

  test("opening while asking is an illegal transition", () => {
    const machine = confirmation();
    machine.open(target);

    expect(() => machine.open(target)).toThrow("delete: open is not allowed from confirming");
  });

  test("an unidentifiable subject is rejected", () => {
    const machine = confirmation();

    expect(() => machine.open({ id: "", name: "Nobody" })).toThrow("delete: an open subject must be identified");
  });
});

describe("one-shot action", () => {
  test("runs once per start and settles back to idle", () => {
    const action = useAction();

    expect(action.isRunning.value).toBe(false);
    expect(action.start()).toBe(true);
    expect(action.isRunning.value).toBe(true);
    expect(action.start()).toBe(false);

    action.settled();
    expect(action.isRunning.value).toBe(false);
  });
});
