<script setup lang="ts">
/**
 * Dashboard page header.
 *
 * Wraps the shared navbar pattern: sidebar collapse control, title and
 * description, the notification bell, and page-specific actions. The page
 * still owns its `UDashboardPanel`; this component only fills its `#header`.
 */
import AppNotificationBell from "@/components/AppNotificationBell.vue";

defineProps<{
  title: string;
  description?: string;
}>();
</script>

<template>
  <UDashboardNavbar :title="title" :description="description">
    <template #leading>
      <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
    </template>
    <slot />
    <template #right>
      <AppNotificationBell />
      <slot name="actions" />
    </template>
  </UDashboardNavbar>
</template>
