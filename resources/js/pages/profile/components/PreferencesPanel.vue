<script setup lang="ts">
import { icons } from "@/lib/icons";
import { previewNotificationSound } from "@/lib/sound";

import type { PreferencesForm, ThemeOption } from "../types";

defineProps<{
  preferencesForm: PreferencesForm;
  themeOptions: ThemeOption[];
  fieldError: (field: string, local?: Record<string, unknown>) => string | undefined;
  submitPreferences: () => void;
}>();

function previewSound(enabled: boolean) {
  if (enabled) previewNotificationSound("info");
}
</script>

<template>
  <div class="max-w-2xl">
    <UCard>
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
            description="A preview plays when enabled."
            @update:model-value="previewSound" />
        </UFormField>

        <div class="flex justify-end">
          <UButton
            type="submit"
            color="primary"
            label="Save preferences"
            :icon="icons.check"
            :loading="preferencesForm.processing" />
        </div>
      </form>
    </UCard>
  </div>
</template>
