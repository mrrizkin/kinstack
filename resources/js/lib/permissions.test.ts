import { describe, expect, test } from "bun:test";

import { PERMISSIONS, permissionLabel } from "./permissions";

describe("permissions", () => {
  test("maps known permission codes to English labels", () => {
    expect(permissionLabel(PERMISSIONS.usersManage)).toBe("Manage users");
    expect(permissionLabel(PERMISSIONS.auditView)).toBe("View audit");
    expect(permissionLabel(PERMISSIONS.mediaManage)).toBe("Manage media");
  });

  test("falls back to the raw code for unknown permissions", () => {
    expect(permissionLabel("billing.export")).toBe("billing.export");
  });
});
