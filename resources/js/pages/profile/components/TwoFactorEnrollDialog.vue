<script setup lang="ts">
import { computed } from "vue";

import { icons } from "@/lib/icons";

import type { TwoFactor, TwoFactorCodeForm } from "../types";

const open = defineModel<boolean>("open", { required: true });

const props = defineProps<{
  twoFactor: TwoFactor;
  twoFactorForm: TwoFactorCodeForm;
  qrDataUrl: string;
  beginningTwoFactor: boolean;
  fieldError: (field: string, local?: Record<string, unknown>) => string | undefined;
}>();

const emit = defineEmits<{ submit: [] }>();

// UPinInput is number[] while the form stores a string; the form stays the
// single source of truth for submission.
const code = computed<number[]>({
  get: () => (props.twoFactorForm.code ? Array.from(props.twoFactorForm.code, Number) : []),
  set: (value) => {
    props.twoFactorForm.code = value.join("");
  },
});

const codeError = computed(() => props.fieldError("code", props.twoFactorForm.errors));
const ready = computed(() => props.twoFactor.pending_secret.length > 0 || props.qrDataUrl.length > 0);
</script>

<template>
  <UModal
    v-model:open="open"
    title="Set up two-step verification"
    description="Scan the QR code with an authenticator app, then confirm the 6-digit code it shows."
    :ui="{ content: 'sm:max-w-lg' }">
    <template #body>
      <div v-if="!ready" class="flex items-center gap-3 py-6">
        <UIcon :name="icons.loading" class="text-muted size-5 animate-spin" />
        <p class="text-muted text-sm">Preparing your authenticator setup…</p>
      </div>

      <div v-else class="flex flex-col gap-5 sm:flex-row sm:items-start">
        <div class="border-default bg-default w-fit shrink-0 rounded-lg border p-3">
          <img v-if="props.qrDataUrl" :src="props.qrDataUrl" alt="Two-factor QR code" class="size-44" />
          <p v-else class="text-muted w-44 text-center text-xs">QR code unavailable</p>
        </div>

        <div class="min-w-0 flex-1 space-y-3">
          <div>
            <p class="text-muted text-xs">Manual key</p>
            <code class="text-highlighted font-mono text-xs break-all">{{ props.twoFactor.pending_secret }}</code>
          </div>

          <form id="confirm-two-factor" class="space-y-2" @submit.prevent="emit('submit')">
            <UFormField label="Verification code" :error="codeError">
              <UPinInput
                v-model="code"
                :length="6"
                type="number"
                otp
                :separator="3"
                size="lg"
                placeholder="○"
                :color="codeError ? 'error' : 'primary'"
                :highlight="Boolean(codeError)"
                :disabled="props.twoFactorForm.processing" />
            </UFormField>
            <p class="text-muted text-xs">Codes refresh every 30 seconds. Spaces are ignored.</p>
          </form>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" label="Cancel" @click="open = false" />
        <UButton
          type="submit"
          form="confirm-two-factor"
          color="primary"
          :icon="icons.check"
          label="Enable 2FA"
          :loading="props.twoFactorForm.processing"
          :disabled="!ready || code.length < 6" />
      </div>
    </template>
  </UModal>
</template>
