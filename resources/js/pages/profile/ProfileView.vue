<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { useI18n } from "vue-i18n";

import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import SettingsLayout from "@/layouts/SettingsLayout.vue";
import { icons } from "@/lib/icons";

import ProfilePanel from "./components/ProfilePanel.vue";
import { useProfileForm } from "./composables/useProfileForm";
import type { Profile } from "./types";

defineOptions({ layout: [BaseLayout, AppLayout, SettingsLayout] });

const props = defineProps<{
  profile: Profile;
}>();

const { t } = useI18n();

const { profileForm, fieldError, formError, submitProfile } = useProfileForm(props);
</script>

<template>
  <Head :title="t('shell.settings.sections.profile.title')" />

  <UAlert v-if="formError" color="error" variant="subtle" :icon="icons.error" :title="formError" class="mb-4" />

  <ProfilePanel
    :profile="profile"
    :profile-form="profileForm"
    :field-error="fieldError"
    :submit-profile="submitProfile" />
</template>
