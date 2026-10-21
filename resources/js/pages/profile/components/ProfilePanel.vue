<script setup lang="ts">
import { computed } from "vue";

import { avatarUrl } from "@/lib/avatar";
import { formatDateTime } from "@/lib/formatters";

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
const avatarSrc = computed(() => avatarUrl(props.profile.username || props.profile.display_name, 128));
</script>

<template>
  <div class="grid gap-4 lg:grid-cols-3">
    <UCard class="lg:col-span-2">
      <div class="flex items-center gap-3">
        <UAvatar :src="avatarSrc" :alt="profile.display_name" size="lg" />
        <div class="min-w-0">
          <h2 class="text-highlighted truncate text-sm font-semibold">{{ profile.display_name }}</h2>
          <p class="text-muted text-xs">@{{ profile.username }}</p>
        </div>
      </div>
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
</template>
