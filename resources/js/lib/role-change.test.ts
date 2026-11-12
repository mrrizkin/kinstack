import { describe, expect, test } from "bun:test";

import { useRoleChange } from "./role-change";

const subject = { id: "user-1", name: "Alex", currentRole: "member" };

describe("role change state machine", () => {
  test("open starts editing with the current role", () => {
    const roleChange = useRoleChange();
    roleChange.open(subject);

    expect(roleChange.isOpen.value).toBe(true);
    expect(roleChange.isSaving.value).toBe(false);
    expect(roleChange.target.value).toEqual(subject);
    expect(roleChange.role.value).toBe("member");
  });

  test("saving is the only state that blocks closing", () => {
    const roleChange = useRoleChange();
    roleChange.open(subject);
    roleChange.save();

    expect(roleChange.isSaving.value).toBe(true);
    expect(roleChange.busyId.value).toBe(subject.id);

    roleChange.close();
    expect(roleChange.isOpen.value).toBe(true);

    roleChange.committed();
    expect(roleChange.isOpen.value).toBe(false);
    expect(roleChange.target.value).toBeNull();
  });

  test("a failed save returns to editing with the chosen role", () => {
    const roleChange = useRoleChange();
    roleChange.open(subject);
    roleChange.role.value = "admin";
    roleChange.save();
    roleChange.failed();

    expect(roleChange.isOpen.value).toBe(true);
    expect(roleChange.isSaving.value).toBe(false);
    expect(roleChange.role.value).toBe("admin");
  });

  test("opening while open is an illegal transition", () => {
    const roleChange = useRoleChange();
    roleChange.open(subject);

    expect(() => roleChange.open(subject)).toThrow("role change: open is not allowed from editing");
  });
});
