import { describe, expect, test } from "bun:test";

import { accessPageSchema } from "./access";
import { apiTokensPageSchema } from "./apitoken";
import { auditPageSchema } from "./audit";
import { dashboardPageSchema } from "./dashboard";
import { mediaPageSchema } from "./media";
import { notificationsPageSchema } from "./notifications";
import { errorPageSchema, loginPageSchema, systemStatePageSchema } from "./pages";
import { passkeysPageSchema } from "./passkeys";
import { preferencesPageSchema, profilePageSchema, securityPageSchema } from "./profile";
import { usersPageSchema } from "./users";

const page = {
  page: 1,
  per_page: 5,
  total: 1,
  total_pages: 1,
  from: 1,
  to: 1,
  has_prev: false,
  has_next: false,
};

describe("page prop contracts", () => {
  test("parses the list pages", () => {
    expect(
      mediaPageSchema.parse({
        assets: [
          { public_id: "med_1", filename: "a.png", content_type: "image/png", size: 12, url: "/a", created_at: "" },
        ],
        page,
        extra: "kept",
      }).assets,
    ).toHaveLength(1);

    const tokens = apiTokensPageSchema.parse({
      tokens: [{ public_id: "tok_1", name: "CI", prefix: "ks_", scopes: ["users.view"], created_at: "" }],
      page,
      created_token: "ks_secret",
    });
    expect(tokens.created_token).toBe("ks_secret");

    const users = usersPageSchema.parse({
      users: [
        {
          public_id: "usr_1",
          username: "alex",
          display_name: "Alex",
          role: "staff",
          is_active: true,
          invited: false,
          created_at: "",
        },
      ],
      roles: [{ code: "staff", name: "Staff", description: "", is_system: true }],
      page,
    });
    expect(users.roles[0]?.code).toBe("staff");

    const access = accessPageSchema.parse({
      roles: [{ code: "staff", name: "Staff", description: "", rank: 10, is_system: true, permissions: [] }],
      role_options: [{ code: "staff", name: "Staff", description: "", rank: 10, is_system: true, permissions: [] }],
      permissions: [{ code: "users.view", category: "Users", label: "View", description: "" }],
      categories: ["Users"],
      users: [
        {
          public_id: "usr_1",
          username: "alex",
          display_name: "Alex",
          role: "staff",
          is_active: true,
          overrides: [{ permission: "users.view", label: "View", effect: "allow", reason: "", expired: false }],
        },
      ],
      page,
      roles_page: page,
    });
    expect(access.roles_page.total).toBe(1);

    const audit = auditPageSchema.parse({
      entries: [
        {
          id: 1,
          occurred_at: "",
          request_id: "r",
          actor_key: "usr_1",
          action: "http.read",
          resource: "/users",
          method: "GET",
          status: 200,
          outcome: "success",
        },
      ],
      filters: {},
      page,
    });
    expect(audit.filters.action).toBe("");

    const notifications = notificationsPageSchema.parse({ notifications: [], unread: 0, next_cursor: "" });
    expect(notifications.next_cursor).toBe("");

    expect(passkeysPageSchema.parse({ passkeys: [], page }).passkeys).toEqual([]);
  });

  test("parses the profile pages", () => {
    const profile = {
      public_id: "usr_1",
      username: "alex",
      display_name: "Alex",
      role: "staff",
      invited: false,
      last_login_at: "",
      created_at: "",
    };
    expect(profilePageSchema.parse({ profile }).profile.username).toBe("alex");
    expect(
      preferencesPageSchema.parse({ preferences: { theme: "dark", sound_enabled: true, page_size: 10 } }).preferences
        .page_size,
    ).toBe(10);

    const security = securityPageSchema.parse({
      profile,
      sessions: [
        { public_id: "ses_1", ip: "127.0.0.1", user_agent: "", login_at: "", last_seen_at: "", current: true },
      ],
      sessions_page: page,
      two_factor: { enabled: false },
    });
    expect(security.two_factor.recovery_codes).toEqual([]);
  });

  test("parses the auth, error and system pages", () => {
    expect(loginPageSchema.parse({ magic_link_enabled: true }).magic_link_enabled).toBe(true);
    expect(errorPageSchema.parse({ status: 404 }).message).toBeUndefined();
    expect(systemStatePageSchema.parse({ surface: "system-help" }).topics).toEqual([]);
    expect(
      dashboardPageSchema.parse({
        metrics: {
          users_total: 1,
          users_active: 1,
          roles_total: 1,
          audit_24h: 0,
          media_total: 0,
          notifications_unread: 0,
        },
      }).metrics.users_total,
    ).toBe(1);
  });
});
