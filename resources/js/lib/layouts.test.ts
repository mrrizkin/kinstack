import { describe, expect, test } from "bun:test";
import { readdirSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";

/**
 * Structural regression tests for the Inertia layout migration:
 *
 * - `UApp` lives only in `BaseLayout.vue`.
 * - each page declares the layout chain it needs via `defineOptions`.
 */
const root = join(import.meta.dir, "..");
const pagesDir = join(root, "pages");
const layoutsDir = join(root, "layouts");

function vueFiles(dir: string): string[] {
  const found: string[] = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      found.push(...vueFiles(path));
    } else if (entry.name.endsWith(".vue")) {
      found.push(path);
    }
  }
  return found;
}

function source(path: string): string {
  return readFileSync(path, "utf8");
}

function relativeToResources(path: string): string {
  return relative(root, path).replaceAll("\\", "/");
}

function layoutDeclaration(path: string): string {
  const match = source(path).match(/defineOptions\(\{[\s\S]*?\}\);/);
  return match?.[0] ?? "";
}

function page(name: string): string {
  return join(pagesDir, name);
}

describe("layout structure", () => {
  test("only BaseLayout renders UApp", () => {
    const offenders = vueFiles(root)
      .filter((file) => relativeToResources(file) !== "layouts/BaseLayout.vue")
      .filter((file) => /<UApp\b/.test(source(file)))
      .map(relativeToResources);

    expect(offenders).toEqual([]);
  });

  test("BaseLayout owns the skip link", () => {
    const base = source(join(layoutsDir, "BaseLayout.vue"));
    expect(base).toContain("<UApp");
    expect(base).toContain('href="#main-content"');
  });

  test("auth pages use BaseLayout and AuthLayout", () => {
    for (const name of [
      "auth/LoginView.vue",
      "auth/TwoFactorView.vue",
      "auth/ForgotPasswordView.vue",
      "auth/ResetPasswordView.vue",
    ]) {
      const declaration = layoutDeclaration(page(name));
      expect(declaration, name).toContain("BaseLayout");
      expect(declaration, name).toContain("AuthLayout");
    }
  });

  test("app pages use BaseLayout and AppLayout", () => {
    for (const name of [
      "access/IndexView.vue",
      "audit/IndexView.vue",
      "dashboard/IndexView.vue",
      "media/IndexView.vue",
      "notifications/IndexView.vue",
      "users/IndexView.vue",
    ]) {
      const declaration = layoutDeclaration(page(name));
      expect(declaration, name).toContain("BaseLayout");
      expect(declaration, name).toContain("AppLayout");
    }
  });

  test("settings pages nest under BaseLayout, AppLayout and SettingsLayout in that order", () => {
    for (const name of [
      "apitoken/IndexView.vue",
      "passkeys/IndexView.vue",
      "profile/ProfileView.vue",
      "profile/SecurityView.vue",
      "profile/PreferencesView.vue",
    ]) {
      const declaration = layoutDeclaration(page(name));
      const order = ["BaseLayout", "AppLayout", "SettingsLayout"].map((layout) => declaration.indexOf(layout));

      expect(order, name).not.toContain(-1);
      expect(order, name).toEqual([...order].sort((a, b) => a - b));
    }
  });

  test("public pages use BaseLayout and PublicLayout", () => {
    const declaration = layoutDeclaration(page("welcome/WelcomeView.vue"));
    expect(declaration).toContain("BaseLayout");
    expect(declaration).toContain("PublicLayout");
  });

  test("error pages use BaseLayout only", () => {
    for (const name of ["errors/not-found.vue", "errors/server.vue"]) {
      const declaration = layoutDeclaration(page(name));
      expect(declaration, name).toContain("BaseLayout");
      expect(declaration, name).not.toContain("AppLayout");
      expect(declaration, name).not.toContain("PublicLayout");
    }
  });

  test("only AppLayout renders the dashboard group and primary sidebar", () => {
    const offenders = vueFiles(root)
      .filter((file) => !relativeToResources(file).startsWith("pages/"))
      .filter((file) => relativeToResources(file) !== "layouts/AppLayout.vue")
      .filter((file) => /<UDashboard(Group|Sidebar)\b/.test(source(file)))
      .map(relativeToResources);

    expect(offenders).toEqual([]);
  });

  test("SettingsLayout owns the settings panel and its vertical navigation", () => {
    const settings = source(join(layoutsDir, "SettingsLayout.vue"));
    expect(settings).toMatch(/<UDashboardPanel\b/);
    expect(settings).toContain('id="settings"');
    expect(settings).toMatch(/<UNavigationMenu[^>]*orientation="vertical"/s);

    for (const name of [
      "apitoken/IndexView.vue",
      "passkeys/IndexView.vue",
      "profile/ProfileView.vue",
      "profile/SecurityView.vue",
      "profile/PreferencesView.vue",
    ]) {
      expect(source(page(name)), name).not.toMatch(/<UDashboardPanel\b/);
      expect(source(page(name)), name).not.toMatch(/<AppPageHeader\b/);
    }
  });

  test("no page wraps a layout component in its template", () => {
    const offenders = vueFiles(pagesDir)
      .filter((file) => /<(AppLayout|AuthLayout|PublicLayout|SettingsLayout)\b/.test(source(file)))
      .map(relativeToResources);

    expect(offenders).toEqual([]);
  });
});
