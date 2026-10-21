<script setup lang="ts">
import AppInputDate from "@/components/AppInputDate.vue";
import { icons } from "@/lib/icons";

import type { Override, OverrideForm, UserAccess } from "../types";

const open = defineModel<boolean>("open", { required: true });

defineProps<{
  user: UserAccess | null;
  overrideForm: OverrideForm;
  permissionOptions: { label: string; value: string }[];
  submitting: boolean;
  fieldError: (field: string) => string | undefined;
  submitOverride: () => void;
  requestRemoveOverride: (user: UserAccess, override: Override) => void;
}>();

function focusPermissionForm() {
  document.getElementById("override-permission")?.focus();
}
</script>

<template>
  <USlideover
    :open="open"
    :title="user ? `Custom permissions · ${user.display_name}` : ''"
    @update:open="(value: boolean) => (!value ? (open = false) : null)">
    <template #body>
      <div v-if="user" class="space-y-5">
        <div class="space-y-2">
          <p class="text-muted text-xs font-semibold tracking-wide uppercase">Active permissions</p>
          <UEmpty
            v-if="user.overrides.length === 0"
            :icon="icons.deny"
            title="No custom permissions yet"
            description="This user only uses permissions from their role."
            :actions="[{ label: 'Add custom permission', icon: icons.add, onClick: focusPermissionForm }]" />
          <div v-else class="space-y-2">
            <div
              v-for="override in user.overrides"
              :key="override.permission"
              class="border-default flex items-start justify-between gap-3 rounded border p-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <UBadge
                    :color="override.expired ? 'neutral' : override.effect === 'allow' ? 'success' : 'error'"
                    variant="subtle"
                    size="sm">
                    {{ override.effect === "allow" ? "Allow" : "Deny" }}
                  </UBadge>
                  <span class="text-highlighted truncate text-sm font-medium">{{ override.label }}</span>
                </div>
                <p class="text-muted mt-1 text-xs">
                  {{ override.reason || "No reason" }}
                  <span v-if="override.expired"> · expired</span>
                  <span v-else-if="override.expires_at"> · until {{ override.expires_at.slice(0, 10) }}</span>
                </p>
              </div>
              <UButton
                size="xs"
                color="error"
                variant="link"
                label="Delete"
                @click="requestRemoveOverride(user, override)" />
            </div>
          </div>
        </div>

        <USeparator />

        <div class="space-y-4">
          <p id="override-permission-form" class="text-muted text-xs font-semibold tracking-wide uppercase">
            Add / edit permission
          </p>
          <UFormField label="Permission" :error="fieldError('permission')">
            <USelect
              id="override-permission"
              v-model="overrideForm.permission"
              :items="permissionOptions"
              class="w-full" />
          </UFormField>
          <UFormField label="Effect" :error="fieldError('effect')">
            <USelect
              v-model="overrideForm.effect"
              :items="[
                { label: 'Allow — add permission', value: 'allow' },
                { label: 'Deny — remove permission', value: 'deny' },
              ]"
              class="w-full" />
          </UFormField>
          <UFormField label="Reason" :error="fieldError('reason')">
            <UInput v-model="overrideForm.reason" placeholder="e.g. covering the night shift" class="w-full" />
          </UFormField>
          <UFormField label="Valid until (optional)" :error="fieldError('expires_at')">
            <AppInputDate v-model="overrideForm.expires" placeholder="2026-01-01" class="w-full" />
          </UFormField>
          <UButton label="Save custom permission" block :loading="submitting" @click="submitOverride" />
        </div>
      </div>
    </template>
  </USlideover>
</template>
