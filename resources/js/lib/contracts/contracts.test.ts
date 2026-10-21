import { describe, expect, test } from "bun:test";

import { notificationFeedSchema } from "./notifications";
import {
  actorSchema,
  authMethodsSchema,
  errorBagSchema,
  errorResponseSchema,
  sharedPagePropsSchema,
} from "./page-props";
import { passkeyOptionsSchema, passkeySummarySchema } from "./passkeys";
import { parseSoundPreference, parseThemePreference, themePreferenceSchema } from "./preferences";

describe("page props contracts", () => {
  test("parses a complete shared prop bag", () => {
    const props = sharedPagePropsSchema.parse({
      app_name: "KinStack",
      csrf_token: "token",
      media_enabled: true,
      auth_methods: { passkey: true, magic_link: false },
      actor: { public_id: "usr_1", username: "alex", superuser: true },
      authz_permissions: ["users.view"],
      errors: { email: ["Email is required"] },
      extra: "kept",
    });

    expect(props.actor?.username).toBe("alex");
    expect(props.actor?.superuser).toBe(true);
    expect(props.authz_permissions).toEqual(["users.view"]);
    expect(props.errors?.email).toEqual(["Email is required"]);
  });

  test("accepts an empty prop bag and a null actor", () => {
    const props = sharedPagePropsSchema.parse({ actor: null });
    expect(props.actor).toBeNull();
    expect(props.authz_permissions).toBeUndefined();
    expect(authMethodsSchema.parse({})).toEqual({});
  });

  test("rejects a wrong actor shape", () => {
    expect(() => actorSchema.parse({ username: 12 })).toThrow();
    expect(() => sharedPagePropsSchema.parse({ authz_permissions: "users.view" })).toThrow();
  });

  test("normalises nullable validation errors", () => {
    const errors = errorBagSchema.parse({ email: null, password: "Too short" });
    expect(errors.email).toBeUndefined();
    expect(errors.password).toBe("Too short");
  });

  test("parses the error envelope", () => {
    const payload = errorResponseSchema.parse({
      error: { status: 422, code: "validation_failed", kind: "validation", message: "Invalid input" },
    });
    expect(payload.error.status).toBe(422);
    expect(payload.error.retryable).toBeUndefined();
    expect(errorResponseSchema.safeParse({ message: "Invalid input" }).success).toBe(false);
  });
});

describe("notification contracts", () => {
  test("defaults a missing feed", () => {
    expect(notificationFeedSchema.parse({})).toEqual({ notifications: [], unread: 0 });
  });

  test("rejects a malformed notification", () => {
    const parsed = notificationFeedSchema.safeParse({
      notifications: [{ public_id: "n_1", kind: "system", title: "Hi", body: "There", read: "yes", created_at: "now" }],
      unread: 1,
    });
    expect(parsed.success).toBe(false);
  });
});

describe("passkey contracts", () => {
  test("accepts a generic public key envelope", () => {
    const options = passkeyOptionsSchema.parse({ publicKey: { challenge: "abc", rp: { id: "example.com" } } });
    expect(options.publicKey.challenge).toBe("abc");
  });

  test("requires publicKey", () => {
    expect(passkeyOptionsSchema.safeParse({}).success).toBe(false);
  });

  test("parses a passkey summary", () => {
    const summary = passkeySummarySchema.parse({ id: "pk_1", name: "Laptop", created_at: "2024-01-01T00:00:00Z" });
    expect(summary.name).toBe("Laptop");
    expect(summary.last_used_at).toBeUndefined();
  });
});

describe("preference contracts", () => {
  test("falls back to safe defaults", () => {
    expect(parseThemePreference("dark")).toBe("dark");
    expect(parseThemePreference("neon")).toBe("system");
    expect(parseThemePreference(null)).toBe("system");
    expect(parseSoundPreference("off")).toBe("off");
    expect(parseSoundPreference("maybe")).toBe("on");
    expect(themePreferenceSchema.options).toEqual(["light", "dark", "system"]);
  });
});
