<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import AppPageHeader from "@/components/AppPageHeader.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import SettingsLayout from "@/layouts/SettingsLayout.vue";
import { icons } from "@/lib/icons";

import PreferencesPanel from "./components/PreferencesPanel.vue";
import { usePreferencesForm } from "./composables/usePreferencesForm";
import type { Preferences } from "./types";

defineOptions({ layout: [BaseLayout, SettingsLayout] });

const props = defineProps<{
  preferences: Preferences;
}>();

const { preferencesForm, themeOptions, fieldError, formError, submitPreferences } = usePreferencesForm(props);
</script>

<template>
  <Head title="Preferences" />
  <UDashboardPanel id="settings-preferences">
    <template #header>
      <AppPageHeader title="Preferences" description="Theme and notification preferences for your account." />
    </template>

    <template #body>
      <UAlert v-if="formError" color="error" variant="subtle" :icon="icons.error" :title="formError" class="mb-4" />

      <PreferencesPanel
        :preferences-form="preferencesForm"
        :theme-options="themeOptions"
        :field-error="fieldError"
        :submit-preferences="submitPreferences" />
    </template>
  </UDashboardPanel>
</template>
