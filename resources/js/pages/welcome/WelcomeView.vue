<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import BaseLayout from "@/layouts/BaseLayout.vue";
import PublicLayout from "@/layouts/PublicLayout.vue";

defineOptions({ layout: [BaseLayout, PublicLayout] });

const features = [
  {
    icon: "i-material-symbols:group-outline-rounded",
    title: "Users & invitations",
    description: "Manage accounts, invitations, activity status, and password resets without leaving the same panel.",
  },
  {
    icon: "i-material-symbols:shield-lock-outline-rounded",
    title: "RBAC + ABAC",
    description: "Roles, permissions, and per-user overrides with fail-closed route policies.",
  },
  {
    icon: "i-material-symbols:history-rounded",
    title: "Audit trail",
    description: "Every important change is recorded with the actor, time, and request context.",
  },
  {
    icon: "i-material-symbols:image-outline-rounded",
    title: "Centralized media",
    description: "Upload and manage assets through S3-compatible object storage.",
  },
  {
    icon: "i-material-symbols:notifications-outline-rounded",
    title: "Notifications",
    description: "A notification center and slide-over that stay in sync with recent activity.",
  },
  {
    icon: "i-material-symbols:passkey-rounded",
    title: "Account security",
    description: "Two-step verification, passkeys, active sessions, API tokens, and account recovery in one place.",
  },
];

const securityPoints = [
  {
    icon: "i-material-symbols:lock-outline-rounded",
    title: "Controlled sessions",
    description: "View and revoke active sessions from your profile page.",
  },
  {
    icon: "i-material-symbols:verified-user-outline-rounded",
    title: "Layered authorization",
    description: "Effective permissions are derived from roles and user overrides.",
  },
  {
    icon: "i-material-symbols:visibility-outline-rounded",
    title: "Transparent",
    description: "Audit and system status can be checked without database access.",
  },
];

const links = [
  {
    label: "Sign in to your account",
    to: "/login",
    trailingIcon: "i-material-symbols:arrow-forward-rounded",
    size: "lg" as const,
  },
  {
    label: "View features",
    to: "/#features",
    color: "neutral" as const,
    variant: "outline" as const,
    size: "lg" as const,
  },
];
</script>

<template>
  <Head title="Home" />
  <UPageHero
    title="Operational control, without the complexity."
    description="KinStack brings users, authorization, account security, and audit together in one clean panel. A ready-to-use foundation for growing teams."
    :links="links"
    :ui="{
      title: 'text-4xl sm:text-5xl lg:text-6xl tracking-tight text-balance',
      description: 'text-base sm:text-lg',
    }">
    <div class="relative mx-auto mt-8 w-full max-w-5xl">
      <div
        class="bg-elevated/60 pointer-events-none absolute -inset-x-8 -top-8 -bottom-10 -z-10 rounded-[2rem] blur-2xl"
        aria-hidden="true" />
      <UCard variant="subtle" :ui="{ root: 'overflow-hidden', body: 'p-0 sm:p-0' }">
        <div class="border-default flex items-center gap-2 border-b px-4 py-3">
          <span class="bg-accented size-2.5 rounded-full" />
          <span class="bg-accented size-2.5 rounded-full" />
          <span class="bg-accented size-2.5 rounded-full" />
          <span class="text-dimmed ml-2 text-xs">kinstack / dashboard</span>
        </div>

        <div class="grid gap-3 p-4 sm:grid-cols-3">
          <div class="bg-default ring-default rounded-lg p-4 ring">
            <p class="text-muted text-xs uppercase">Users</p>
            <p class="text-highlighted mt-2 text-2xl font-semibold">Active</p>
            <div class="mt-4 flex items-end gap-1">
              <span
                v-for="height in [40, 58, 46, 72, 64, 86, 78]"
                :key="height"
                class="bg-primary/80 w-full rounded-sm"
                :style="{ height: `${height * 0.4}px` }" />
            </div>
          </div>
          <div class="bg-default ring-default rounded-lg p-4 ring">
            <p class="text-muted text-xs uppercase">Access</p>
            <p class="text-highlighted mt-2 text-2xl font-semibold">RBAC</p>
            <div class="mt-4 space-y-2">
              <div class="bg-accented h-2 w-full rounded-full" />
              <div class="bg-accented h-2 w-4/5 rounded-full" />
              <div class="bg-accented h-2 w-2/3 rounded-full" />
            </div>
          </div>
          <div class="bg-default ring-default rounded-lg p-4 ring">
            <p class="text-muted text-xs uppercase">Audit</p>
            <p class="text-highlighted mt-2 text-2xl font-semibold">Tracked</p>
            <div class="mt-4 space-y-2">
              <div class="flex items-center gap-2">
                <span class="bg-primary size-1.5 rounded-full" />
                <div class="bg-accented h-2 flex-1 rounded-full" />
              </div>
              <div class="flex items-center gap-2">
                <span class="bg-primary/60 size-1.5 rounded-full" />
                <div class="bg-accented h-2 w-5/6 rounded-full" />
              </div>
              <div class="flex items-center gap-2">
                <span class="bg-primary/40 size-1.5 rounded-full" />
                <div class="bg-accented h-2 w-2/3 rounded-full" />
              </div>
            </div>
          </div>
        </div>
      </UCard>
    </div>
  </UPageHero>

  <UPageSection
    id="features"
    class="py-10 sm:py-14"
    headline="Features"
    title="A foundation ready to use"
    description="Everything you need to run an internal application, without rebuilding the basics.">
    <UPageGrid>
      <UPageCard v-for="feature in features" :key="feature.title" v-bind="feature" variant="subtle" spotlight />
    </UPageGrid>
  </UPageSection>

  <UPageSection
    id="security"
    class="py-10 sm:py-14"
    headline="Security"
    title="Clear access, complete trails"
    description="KinStack is built for organizations that need to know who did what, when, and from where."
    orientation="horizontal"
    :features="securityPoints">
    <div class="bg-elevated/50 ring-default rounded-2xl p-6 ring sm:p-10">
      <div class="mx-auto max-w-sm space-y-4">
        <div class="flex items-center justify-between">
          <span class="text-sm font-medium">Two-step verification</span>
          <UBadge color="success" variant="subtle">Active</UBadge>
        </div>
        <USeparator />
        <div class="flex items-center justify-between">
          <span class="text-sm font-medium">Passkey on this device</span>
          <UBadge color="neutral" variant="subtle">Registered</UBadge>
        </div>
        <USeparator />
        <div class="flex items-center justify-between">
          <span class="text-sm font-medium">Active sessions</span>
          <span class="text-muted text-sm">3 devices</span>
        </div>
        <USeparator />
        <div class="flex items-center justify-between">
          <span class="text-sm font-medium">Audit trail</span>
          <span class="text-muted text-sm">Real-time</span>
        </div>
      </div>
    </div>
  </UPageSection>

  <UPageCTA
    title="Ready to sign in?"
    description="Use the account provided by your administrator. Don't have access yet? Contact your organization admin."
    :links="[{ label: 'Sign in', to: '/login', trailingIcon: 'i-material-symbols:arrow-forward-rounded', size: 'lg' }]"
    variant="naked"
    class="overflow-hidden py-10 sm:py-14" />
</template>
