<script setup lang="ts">
import { en as uiEn } from "@nuxt/ui/locale";
import { useI18n } from "vue-i18n";

import AppFlashToaster from "@/components/AppFlashToaster.vue";

const { t } = useI18n();
</script>

<template>
  <UApp :locale="uiEn" :toaster="{ position: 'top-center' }">
    <a
      href="#main-content"
      class="bg-default text-highlighted ring-default sr-only rounded-md px-3 py-2 text-sm font-medium ring focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50">
      {{ t("shell.auth.skipToContent") }}
    </a>
    <AppFlashToaster />
    <slot />
  </UApp>
</template>
