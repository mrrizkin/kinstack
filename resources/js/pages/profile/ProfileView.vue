<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import AppPageHeader from "@/components/AppPageHeader.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import SettingsLayout from "@/layouts/SettingsLayout.vue";

import ProfilePanel from "./components/ProfilePanel.vue";
import { useProfileForm } from "./composables/useProfileForm";
import type { Profile } from "./types";

defineOptions({ layout: [BaseLayout, SettingsLayout] });

const props = defineProps<{
  profile: Profile;
}>();

const { profileForm, fieldError, formError, submitProfile } = useProfileForm(props);
</script>

<template>
  <Head title="Profile" />
  <UDashboardPanel id="settings-profile">
    <template #header>
      <AppPageHeader title="Profile" description="Your account details and public profile." />
    </template>

    <template #body>
      <UAlert v-if="formError" color="error" variant="subtle" icon="i-bi-x-circle" :title="formError" class="mb-4" />

      <ProfilePanel
        :profile="profile"
        :profile-form="profileForm"
        :field-error="fieldError"
        :submit-profile="submitProfile" />
    </template>
  </UDashboardPanel>
</template>
