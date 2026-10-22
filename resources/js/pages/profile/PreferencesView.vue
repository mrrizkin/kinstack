<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { useI18n } from "vue-i18n";

import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import SettingsLayout from "@/layouts/SettingsLayout.vue";
import { preferencesPageSchema } from "@/lib/contracts/profile";
import { icons } from "@/lib/icons";
import { usePageSchema } from "@/lib/page-props";

import PreferencesPanel from "./components/PreferencesPanel.vue";
import { usePreferencesForm } from "./composables/usePreferencesForm";

defineOptions({ layout: [BaseLayout, AppLayout, SettingsLayout] });

const props = usePageSchema(preferencesPageSchema);

const { t } = useI18n();

const { preferencesForm, themeOptions, rowsPerPageOptions, fieldError, formError, submitPreferences } =
  usePreferencesForm(props);
</script>

<template>
  <Head :title="t('shell.settings.sections.preferences.title')" />

  <UAlert v-if="formError" color="error" variant="subtle" :icon="icons.error" :title="formError" class="mb-4" />

  <PreferencesPanel
    :preferences-form="preferencesForm"
    :theme-options="themeOptions"
    :rows-per-page-options="rowsPerPageOptions"
    :field-error="fieldError"
    :submit-preferences="submitPreferences" />
</template>
