<script setup lang="ts">
import { computed } from "vue";

import { formatDateTime } from "@/lib/formatters";

import type { PreferencesForm, Profile, ProfileForm, ThemeOption } from "../types";

const props = defineProps<{
  tab: string;
  profile: Profile;
  profileForm: ProfileForm;
  preferencesForm: PreferencesForm;
  themeOptions: ThemeOption[];
  fieldError: (field: string, local?: Record<string, unknown>) => string;
  submitProfile: () => void;
  submitPreferences: () => void;
}>();

const roleLabel = computed(() => {
  const code = props.profile.role;
  if (!code) return "No role";
  return code.charAt(0).toUpperCase() + code.slice(1);
});
const roleColor = "neutral" as const;
</script>

<template>
  <div v-if="tab === 'profile'" class="mt-4 grid gap-4 lg:grid-cols-3">
    <UCard class="lg:col-span-2">
      <h2 class="text-highlighted text-sm font-semibold">Account information</h2>
      <form class="mt-4 space-y-4" @submit.prevent="submitProfile">
        <UFormField label="Username" description="Username cannot be changed.">
          <UInput :model-value="profile.username" disabled class="w-full" />
        </UFormField>
        <UFormField
          label="Display name"
          hint="3-120 characters"
          :error="fieldError('display_name', profileForm.errors)">
          <UInput v-model="profileForm.display_name" class="w-full" />
        </UFormField>
        <div class="flex justify-end">
          <UButton
            type="submit"
            color="primary"
            label="Save profile"
            icon="i-bi-check2"
            :loading="profileForm.processing" />
        </div>
      </form>
    </UCard>

    <UCard>
      <h2 class="text-highlighted text-sm font-semibold">Overview</h2>
      <dl class="mt-4 space-y-3 text-sm">
        <div class="flex items-center justify-between gap-2">
          <dt class="text-muted">Role</dt>
          <dd>
            <UBadge :color="roleColor" variant="subtle" size="sm">{{ roleLabel }}</UBadge>
          </dd>
        </div>
        <div class="flex items-center justify-between gap-2">
          <dt class="text-muted">Last login</dt>
          <dd class="text-highlighted">{{ formatDateTime(profile.last_login_at) }}</dd>
        </div>
        <div class="flex items-center justify-between gap-2">
          <dt class="text-muted">Account created</dt>
          <dd class="text-highlighted">{{ formatDateTime(profile.created_at) }}</dd>
        </div>
      </dl>
    </UCard>
  </div>

  <div v-else class="mt-4 grid gap-4 lg:grid-cols-3">
    <UCard class="lg:col-span-2">
      <h2 class="text-highlighted text-sm font-semibold">Personalization</h2>
      <form class="mt-4 space-y-4" @submit.prevent="submitPreferences">
        <UFormField
          label="Display theme"
          description="Applies across all devices after saving."
          :error="fieldError('theme', preferencesForm.errors)">
          <USelect v-model="preferencesForm.theme" :items="themeOptions" class="w-full" />
        </UFormField>

        <UFormField label="Notification sound">
          <USwitch
            v-model="preferencesForm.sound_enabled"
            label="Play a chime when a notification appears"
            description="A preview plays when enabled." />
        </UFormField>

        <div class="flex justify-end">
          <UButton
            type="submit"
            color="primary"
            label="Save preferences"
            icon="i-bi-check2"
            :loading="preferencesForm.processing" />
        </div>
      </form>
    </UCard>
  </div>
</template>
