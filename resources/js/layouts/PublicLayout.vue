<script setup lang="ts">
import type { NavigationMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { useI18n } from "vue-i18n";

import AppLogo from "@/components/AppLogo.vue";

const { t } = useI18n();

const navigation: NavigationMenuItem[] = [
  { label: "Features", to: "/#features" },
  { label: "Security", to: "/#security" },
  { label: "Help", to: "/help" },
];

const year = new Date().getFullYear();
</script>

<template>
  <div>
    <UHeader>
      <template #left>
        <UButton to="/" color="neutral" variant="ghost" class="-ms-2 p-1">
          <AppLogo />
        </UButton>
      </template>

      <UNavigationMenu :items="navigation" variant="link" class="hidden lg:flex" />

      <template #right>
        <UColorModeButton color="neutral" variant="ghost" />
        <UButton
          to="/login"
          label="Sign in"
          color="neutral"
          trailing-icon="i-material-symbols:arrow-forward-rounded"
          class="hidden lg:inline-flex" />
      </template>

      <template #body>
        <UNavigationMenu :items="navigation" orientation="vertical" class="-mx-2.5" />
        <USeparator class="my-6" />
        <UButton to="/login" label="Sign in" block />
      </template>
    </UHeader>

    <UMain id="main-content" tabindex="-1">
      <slot />
    </UMain>

    <UFooter>
      <template #left>
        <p class="text-muted text-sm">© {{ year }} KinStack. Internal access.</p>
      </template>
      <template #right>
        <UButton to="/help" label="Help" color="neutral" variant="ghost" />
        <UButton to="/login" label="Sign in" color="neutral" variant="ghost" />
      </template>
    </UFooter>
  </div>
</template>
