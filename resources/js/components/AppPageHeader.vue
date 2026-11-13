<script setup lang="ts">
/**
 * Dashboard page header: collapse control, title, notification bell, actions.
 */
import { useI18n } from "vue-i18n";

import AppNotificationBell from "@/components/AppNotificationBell.vue";

const { t } = useI18n();

defineProps<{
  title: string;
  description?: string;
}>();
</script>

<template>
  <UDashboardNavbar :title="title" :description="description">
    <template #leading>
      <UDashboardSidebarCollapse :aria-label="t('shell.sidebar.toggle')" />
      <slot name="leading" />
    </template>
    <slot />
    <template #right>
      <AppNotificationBell />
      <slot name="actions" />
    </template>
  </UDashboardNavbar>
</template>
