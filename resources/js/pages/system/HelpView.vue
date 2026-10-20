<script setup lang="ts">
import { Head, usePage } from "@inertiajs/vue3";
import { computed } from "vue";

import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import PublicLayout from "@/layouts/PublicLayout.vue";

defineOptions({
  layout: (props: { actor?: unknown }) => (props.actor ? [BaseLayout, AppLayout] : [BaseLayout, PublicLayout]),
});

interface Shortcut {
  key: string;
  action: string;
}

defineProps<{ topics: string[]; shortcuts: Shortcut[] }>();

const page = usePage();
const isAuthenticated = computed(() => Boolean(page.props.actor));
</script>

<template>
  <Head title="Help" />

  <UDashboardPanel v-if="isAuthenticated" id="help">
    <template #header>
      <UDashboardNavbar title="Help & support" description="A quick guide to using KinStack.">
        <template #leading>
          <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
        </template>
        <template #right>
          <AppNotificationBell />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="mx-auto max-w-3xl space-y-6">
        <UCard>
          <div class="flex items-start gap-3">
            <span class="bg-elevated text-toned ring-default grid size-10 shrink-0 place-items-center rounded-lg ring">
              <UIcon name="i-material-symbols:menu-book-outline-rounded" class="size-5" />
            </span>
            <div>
              <h1 class="text-highlighted text-lg font-semibold">Help center</h1>
              <p class="text-muted mt-1 text-sm leading-6">
                Find quick guides for managing accounts, access, audit, media, and notifications.
              </p>
            </div>
          </div>
        </UCard>

        <section>
          <h2 class="text-highlighted mb-3 text-sm font-semibold">Popular topics</h2>
          <div class="grid gap-3 sm:grid-cols-2">
            <UCard v-for="topic in topics" :key="topic" variant="subtle">
              <div class="flex items-start gap-3">
                <UIcon name="i-material-symbols:arrow-forward-rounded" class="text-dimmed mt-0.5 size-4 shrink-0" />
                <p class="text-toned text-sm leading-6">{{ topic }}</p>
              </div>
            </UCard>
          </div>
        </section>

        <UCard v-if="shortcuts.length">
          <h2 class="text-highlighted text-sm font-semibold">Keyboard shortcuts</h2>
          <dl class="mt-4 space-y-2">
            <div
              v-for="shortcut in shortcuts"
              :key="shortcut.key"
              class="border-default flex items-center justify-between gap-4 border-b py-2 last:border-0">
              <dt>
                <UKbd>{{ shortcut.key }}</UKbd>
              </dt>
              <dd class="text-muted text-sm">{{ shortcut.action }}</dd>
            </div>
          </dl>
        </UCard>

        <UAlert
          color="neutral"
          variant="subtle"
          icon="i-material-symbols:support-agent-outline-rounded"
          title="Need more help?"
          description="Contact your organization administrator. Activity audit and system status remain available from this panel." />
      </div>
    </template>
  </UDashboardPanel>

  <div v-else class="mx-auto max-w-3xl px-4 py-12 sm:py-16">
    <UCard>
      <div class="flex items-start gap-3">
        <span class="bg-elevated text-toned ring-default grid size-10 shrink-0 place-items-center rounded-lg ring">
          <UIcon name="i-material-symbols:menu-book-outline-rounded" class="size-5" />
        </span>
        <div>
          <h1 class="text-highlighted text-lg font-semibold">Help center</h1>
          <p class="text-muted mt-1 text-sm leading-6">
            A quick guide to using KinStack. Sign in to the panel to access all features.
          </p>
        </div>
      </div>
    </UCard>

    <section class="mt-6">
      <h2 class="text-highlighted mb-3 text-sm font-semibold">Popular topics</h2>
      <div class="grid gap-3 sm:grid-cols-2">
        <UCard v-for="topic in topics" :key="topic" variant="subtle">
          <div class="flex items-start gap-3">
            <UIcon name="i-material-symbols:arrow-forward-rounded" class="text-dimmed mt-0.5 size-4 shrink-0" />
            <p class="text-toned text-sm leading-6">{{ topic }}</p>
          </div>
        </UCard>
      </div>
    </section>

    <UCard v-if="shortcuts.length" class="mt-6">
      <h2 class="text-highlighted text-sm font-semibold">Keyboard shortcuts</h2>
      <dl class="mt-4 space-y-2">
        <div
          v-for="shortcut in shortcuts"
          :key="shortcut.key"
          class="border-default flex items-center justify-between gap-4 border-b py-2 last:border-0">
          <dt>
            <UKbd>{{ shortcut.key }}</UKbd>
          </dt>
          <dd class="text-muted text-sm">{{ shortcut.action }}</dd>
        </div>
      </dl>
    </UCard>

    <div class="mt-6 flex flex-wrap gap-2">
      <UButton to="/" color="neutral" variant="outline" icon="i-material-symbols:arrow-back-rounded" label="Back" />
      <UButton to="/login" color="primary" icon="i-material-symbols:login-rounded" label="Sign in" />
    </div>
  </div>
</template>
