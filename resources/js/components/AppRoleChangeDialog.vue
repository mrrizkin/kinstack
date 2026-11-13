<script setup lang="ts">
/**
 * Role picker for the users and access tables. A role change is never inline:
 * this modal opens from the row menu.
 */
import { icons } from "@/lib/icons";

defineProps<{
  open: boolean;
  target: string | null;
  roleOptions: { label: string; value: string }[];
  loading: boolean;
}>();

const role = defineModel<string>("role", { required: true });

const emit = defineEmits<{ confirm: []; close: [] }>();
</script>

<template>
  <UModal
    :open="open"
    title="Change role"
    :description="target ? `Choose a new role for ${target}.` : ''"
    :ui="{ content: 'sm:max-w-md' }"
    @update:open="(value: boolean) => (!value ? emit('close') : null)">
    <template #body>
      <UFormField label="Role">
        <USelectMenu
          v-model="role"
          :items="roleOptions"
          value-key="value"
          :search-input="{ placeholder: 'Search roles...', icon: icons.search }"
          class="w-full" />
      </UFormField>
    </template>

    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" label="Cancel" @click="emit('close')" />
        <UButton color="primary" label="Save role" :loading="loading" @click="emit('confirm')" />
      </div>
    </template>
  </UModal>
</template>
