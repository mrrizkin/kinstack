<script setup lang="ts">
import type { PermissionCategory, Role, RoleCreateForm, RoleEditForm } from "../types";

const createOpen = defineModel<boolean>("createOpen", { required: true });
const editTarget = defineModel<Role | null>("editTarget", { required: true });
const editingRole = defineModel<Role | null>("editingRole", { required: true });

defineProps<{
  createForm: RoleCreateForm;
  editForm: RoleEditForm;
  permissionsByCategory: PermissionCategory[];
  selectedPermissions: string[];
  savingPermissions: boolean;
  submitCreate: () => void;
  submitEdit: () => void;
  togglePermission: (code: string, checked: boolean) => void;
  saveRolePermissions: () => void;
}>();
</script>

<template>
  <UModal v-model:open="createOpen" title="New role" description="A custom role with configurable permissions.">
    <template #body>
      <div class="space-y-4">
        <UFormField label="Code" :error="createForm.errors.code">
          <UInput v-model="createForm.code" placeholder="e.g. supervisor" class="w-full" />
        </UFormField>
        <UFormField label="Name" :error="createForm.errors.name">
          <UInput v-model="createForm.name" placeholder="e.g. Supervisor" class="w-full" />
        </UFormField>
        <UFormField label="Description" :error="createForm.errors.description">
          <UInput v-model="createForm.description" class="w-full" />
        </UFormField>
        <UFormField label="Priority" :error="createForm.errors.rank">
          <UInput v-model.number="createForm.rank" type="number" min="0" max="1000" class="w-full" />
        </UFormField>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton color="neutral" variant="ghost" label="Cancel" @click="createOpen = false" />
        <UButton label="Save" :loading="createForm.processing" @click="submitCreate" />
      </div>
    </template>
  </UModal>

  <UModal
    :open="editTarget !== null"
    title="Edit role"
    description="Role name, description, and priority. The role code is fixed."
    @update:open="(value: boolean) => (!value ? (editTarget = null) : null)">
    <template #body>
      <div class="space-y-4">
        <UFormField label="Code (fixed)">
          <UInput :model-value="editTarget?.code ?? ''" disabled class="w-full" />
        </UFormField>
        <UFormField label="Name" :error="editForm.errors.name">
          <UInput v-model="editForm.name" class="w-full" />
        </UFormField>
        <UFormField label="Description" :error="editForm.errors.description">
          <UInput v-model="editForm.description" class="w-full" />
        </UFormField>
        <UFormField label="Priority" :error="editForm.errors.rank">
          <UInput v-model.number="editForm.rank" type="number" min="0" max="1000" class="w-full" />
        </UFormField>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton color="neutral" variant="ghost" label="Cancel" @click="editTarget = null" />
        <UButton label="Save" :loading="editForm.processing" @click="submitEdit" />
      </div>
    </template>
  </UModal>

  <USlideover
    :open="editingRole !== null"
    :title="editingRole ? `Role permissions · ${editingRole.name}` : ''"
    @update:open="(value: boolean) => (!value ? (editingRole = null) : null)">
    <template #body>
      <div class="space-y-5">
        <div v-for="group in permissionsByCategory" :key="group.category" class="space-y-2">
          <p class="text-muted text-xs font-semibold tracking-wide uppercase">{{ group.category }}</p>
          <UCheckbox
            v-for="permission in group.items"
            :key="permission.code"
            :model-value="selectedPermissions.includes(permission.code)"
            :label="permission.label"
            @update:model-value="
              (value: boolean | 'indeterminate') => togglePermission(permission.code, value === true)
            " />
        </div>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton color="neutral" variant="ghost" label="Cancel" @click="editingRole = null" />
        <UButton label="Save permissions" :loading="savingPermissions" @click="saveRolePermissions" />
      </div>
    </template>
  </USlideover>
</template>
