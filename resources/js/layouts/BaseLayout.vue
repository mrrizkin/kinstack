<script setup lang="ts">
import { en as uiEn } from "@nuxt/ui/locale";

import AppFlashToaster from "@/components/AppFlashToaster.vue";
</script>

<template>
  <UApp :locale="uiEn" :toaster="{ position: 'top-center' }">
    <a
      href="#main-content"
      class="bg-default text-highlighted ring-default sr-only rounded-md px-3 py-2 text-sm font-medium ring focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50">
      Skip to content
    </a>
    <AppFlashToaster />
    <slot />
  </UApp>
</template>
