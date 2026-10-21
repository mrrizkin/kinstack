<script setup lang="ts">
/**
 * Settings section shell. Owns the settings `UDashboardPanel` so nested pages
 * stay content-only and the section navigation never duplicates the main
 * sidebar.
 */
import { usePage } from "@inertiajs/vue3";
import type { NavigationMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import AppPageHeader from "@/components/AppPageHeader.vue";
import { icons } from "@/lib/icons";
import { normalizeSettingsPath, settingsNavOpen, useSettingsNav } from "@/lib/settings-nav";

const page = usePage();
const { t } = useI18n();
const { groups, items } = useSettingsNav();

const location = computed(() => normalizeSettingsPath(page.url));

const current = computed(
  () => items.value.find((item) => normalizeSettingsPath(item.to) === location.value) ?? items.value[0],
);

const links = computed<NavigationMenuItem[]>(() =>
  groups.value.flatMap((group) => [
    { type: "label", label: group.label },
    ...group.items.map((item) => ({
      label: item.label,
      icon: item.icon,
      to: item.to,
      active: normalizeSettingsPath(item.to) === location.value,
      onSelect: () => {
        settingsNavOpen.value = false;
      },
    })),
  ]),
);
</script>

<template>
  <UDashboardPanel id="settings">
    <template #header>
      <AppPageHeader :title="current?.title ?? t('shell.settings.title')" :description="current?.description">
        <template #leading>
          <UButton
            color="neutral"
            variant="ghost"
            :icon="icons.settings"
            :aria-label="t('shell.settings.title')"
            :title="t('shell.settings.title')"
            class="lg:hidden"
            @click="settingsNavOpen = true" />
        </template>
      </AppPageHeader>
    </template>

    <template #body>
      <div class="flex flex-col gap-6 lg:flex-row lg:gap-8">
        <nav class="hidden lg:block lg:w-56 lg:shrink-0" :aria-label="t('shell.settings.title')">
          <UNavigationMenu :items="links" orientation="vertical" />
        </nav>

        <div class="min-w-0 flex-1">
          <slot />
        </div>
      </div>
    </template>
  </UDashboardPanel>

  <UDrawer
    v-model:open="settingsNavOpen"
    direction="bottom"
    :title="t('shell.settings.title')"
    close
    :ui="{ content: 'lg:hidden', body: 'p-0' }">
    <template #body>
      <UNavigationMenu :items="links" orientation="vertical" />
    </template>
  </UDrawer>
</template>
