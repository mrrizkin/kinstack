<script setup lang="ts">
interface Props {
  open: boolean;
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  color?: "error" | "primary" | "warning";
  loading?: boolean;
}

withDefaults(defineProps<Props>(), {
  description: "",
  confirmLabel: "Continue",
  cancelLabel: "Cancel",
  color: "error",
  loading: false,
});

const emit = defineEmits<{ "update:open": [boolean]; confirm: [] }>();
</script>

<template>
  <UModal
    :open="open"
    :title="title"
    :description="description"
    :ui="{ content: 'sm:max-w-md' }"
    @update:open="(value: boolean) => emit('update:open', value)">
    <template #body>
      <slot />
    </template>

    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" :label="cancelLabel" @click="emit('update:open', false)" />
        <UButton :color="color" :label="confirmLabel" :loading="loading" @click="emit('confirm')" />
      </div>
    </template>
  </UModal>
</template>
