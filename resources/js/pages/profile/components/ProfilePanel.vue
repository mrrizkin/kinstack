<script setup lang="ts">
import { computed } from "vue";

import { avatarUrl } from "@/lib/avatar";
import { formatDateTime } from "@/lib/formatters";
import { icons } from "@/lib/icons";

import type { Profile, ProfileForm } from "../types";

const props = defineProps<{
  profile: Profile;
  profileForm: ProfileForm;
  fieldError: (field: string, local?: Record<string, unknown>) => string | undefined;
  submitProfile: () => void;
}>();

const roleLabel = computed(() => {
  const code = props.profile.role;
  if (!code) return "No role";
  return code.charAt(0).toUpperCase() + code.slice(1);
});
const roleColor = "neutral" as const;
const avatarSrc = computed(() => avatarUrl(props.profile, 128));
const lastLogin = computed(() =>
  props.profile.last_login_at ? formatDateTime(props.profile.last_login_at) : "No sign-in yet",
);
</script>

<template>
  <div class="max-w-2xl">
    <UCard>
      <div class="flex items-center gap-3">
        <UAvatar :src="avatarSrc" :alt="profile.display_name" size="lg" />
        <div class="min-w-0">
          <h2 class="text-highlighted truncate text-sm font-semibold">{{ profile.display_name }}</h2>
          <p class="text-muted text-xs">@{{ profile.username }}</p>
        </div>
      </div>

      <ul class="text-muted mt-4 flex flex-wrap items-center gap-x-2 gap-y-2 text-xs">
        <li>
          <UBadge :color="roleColor" variant="subtle" size="sm">{{ roleLabel }}</UBadge>
        </li>
        <li aria-hidden="true" class="text-dimmed">·</li>
        <li>Joined <AppCellTime :value="profile.created_at" inline /></li>
        <li aria-hidden="true" class="text-dimmed">·</li>
        <li>Last login {{ lastLogin }}</li>
      </ul>

      <USeparator class="my-4" />

      <form class="space-y-4" @submit.prevent="submitProfile">
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
            :icon="icons.check"
            :loading="profileForm.processing" />
        </div>
      </form>
    </UCard>
  </div>
</template>
