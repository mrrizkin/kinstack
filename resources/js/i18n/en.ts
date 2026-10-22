import type { Locale } from "vue-i18n";

export const messages = {
  shell: {
    searchMenu: "Search menu…",
    help: "Help & support",
    groups: {
      main: "Main",
      administration: "Administration",
      system: "System",
    },
    nav: {
      dashboard: "Dashboard",
      users: "Users",
      access: "Access control",
      audit: "Audit",
      media: "Media",
      notifications: "Notifications",
    },
    workspace: {
      dashboard: "Dashboard",
      help: "Help & support",
    },
    settings: {
      title: "Settings",
      groups: {
        account: "Account",
        access: "Access",
      },
      sections: {
        profile: {
          title: "Profile",
          description: "Your account details and public profile.",
        },
        security: {
          title: "Security",
          description: "Password, two-step verification, and active sessions.",
        },
        preferences: {
          title: "Preferences",
          description: "Theme and notification preferences for your account.",
        },
        passkeys: {
          title: "Passkeys",
          description: "Sign in without a password using your fingerprint, face, or device PIN.",
        },
        apiTokens: {
          title: "API tokens",
          description: "Tokens for connecting integrations without a browser session.",
        },
      },
      nav: {
        profile: "Profile",
        security: "Security",
        preferences: "Preferences",
        passkeys: "Passkeys",
        apiTokens: "API tokens",
      },
    },
    userMenu: {
      settings: "Settings",
      appearance: "Appearance",
      light: "Light",
      dark: "Dark",
      system: "System",
      notifications: "Notifications",
      help: "Help & support",
      logout: "Sign out",
    },
    publicNav: {
      features: "Features",
      security: "Security",
      signIn: "Sign in",
    },
    filters: {
      close: "Close filters",
    },
    sidebar: {
      toggle: "Open or close sidebar",
    },
    pagination: {
      showing: "Showing {from}–{to} of {total}",
      rowsPerPage: "Rows per page",
      previous: "Previous",
      next: "Next",
      page: "Page {page} / {pages}",
    },
    table: {
      view: "View",
      columns: "Columns",
      showAll: "Show all",
    },
    notifications: {
      title: "Notifications",
      unread: "{count} unread",
      allRead: "All caught up",
      none: "No notifications",
      markAll: "Mark all as read",
      openCenter: "Open notification center",
      loadError: "Could not load notifications.",
      markError: "Could not mark the notification as read.",
      newOne: "New notification",
      newMany: "{count} new notifications",
      markAllError: "Could not mark all notifications as read.",
      open: "Open notifications",
    },
    auth: {
      back: "Back",
      skipToContent: "Skip to content",
      greetingMorning: "Good morning",
      greetingAfternoon: "Good afternoon",
      greetingEvening: "Good evening",
      greetingNight: "Good evening",
      welcome: "welcome to KinStack",
      footer: "Access is restricted to authorized users. Sign-in activity and data changes are recorded for audit.",
    },
    brand: {
      eyebrow: "Management system",
      headline: "One place to manage your team, access, and change history.",
      description: "A clean operational foundation: users, authorization, account security, and audit in one workflow.",
      featureRbac: "Fine-grained role and user permissions",
      featureAudit: "Traceable change history",
      featureMfa: "Two-factor authentication and passkeys",
      copyright: "© {year} KinStack. Internal access.",
    },
  },
} as const;

export const defaultLocale: Locale = "en";
export const fallbackLocale: Locale = "en";
