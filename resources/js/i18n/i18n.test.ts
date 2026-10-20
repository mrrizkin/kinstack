import { describe, expect, test } from "bun:test";

import { i18n } from "./index";

describe("i18n", () => {
  test("uses English as the default locale", () => {
    expect(i18n.global.locale.value).toBe("en");
    expect(i18n.global.t("shell.nav.dashboard")).toBe("Dashboard");
    expect(i18n.global.t("shell.auth.back")).toBe("Back");
  });

  test("interpolates notification counts", () => {
    expect(i18n.global.t("shell.notifications.unread", { count: 3 })).toBe("3 unread");
  });
});
