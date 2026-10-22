<script setup lang="ts">
import { Head, router, useForm } from "@inertiajs/vue3";
import { computed, ref } from "vue";

import AppConfirmDialog from "@/components/AppConfirmDialog.vue";
import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppPagination from "@/components/AppPagination.vue";
import AppCellTime from "@/components/cells/AppCellTime.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { type Asset, mediaPageSchema } from "@/lib/contracts/media";
import { fieldMessage } from "@/lib/errors";
import { icons } from "@/lib/icons";
import { usePageProps, usePageSchema } from "@/lib/page-props";
import { PERMISSIONS, usePermissions } from "@/lib/permissions";
import { queryString } from "@/lib/query";

defineOptions({ layout: [BaseLayout, AppLayout] });

const props = usePageSchema(mediaPageSchema);

const { can } = usePermissions();
const { actor } = usePageProps();
const isSuperuser = Boolean(actor.value?.superuser);
const canManage = can(PERMISSIONS.mediaManage) && !isSuperuser;

const MAX_UPLOAD = 10 * 1024 * 1024;
const ACCEPTED = "image/jpeg,image/png,image/webp,image/gif,application/pdf,text/plain,text/csv";
const ACCEPTED_LABEL = "JPG, PNG, WebP, GIF, PDF, TXT, or CSV · up to 10 MB";

const pageQuery = computed(() => queryString({ page: props.page.page > 1 ? props.page.page : undefined }));

const form = useForm<{ file: File | null }>({ file: null });
const fileInput = ref<HTMLInputElement | null>(null);
const dragging = ref(false);
const clientError = ref("");
const deleteTarget = ref<Asset | null>(null);
const deleting = ref(false);

function pickFile() {
  fileInput.value?.click();
}

function validateFile(file: File): string {
  if (file.size > MAX_UPLOAD) return "File size exceeds the 10 MB limit.";
  const extension = file.name.split(".").pop()?.toLowerCase() ?? "";
  const allowedExtension = ["jpg", "jpeg", "png", "webp", "gif", "pdf", "txt", "csv"].includes(extension);
  const allowedType = ACCEPTED.split(",").includes(file.type);
  if (!allowedType && !allowedExtension) return "File type is not supported.";
  return "";
}

function selectFile(file: File | undefined) {
  clientError.value = "";
  if (!file) return;
  const error = validateFile(file);
  if (error) {
    clientError.value = error;
    return;
  }
  form.file = file;
  upload();
}

function onFile(event: Event) {
  const target = event.target as HTMLInputElement;
  selectFile(target.files?.[0]);
  target.value = "";
}

function onDrop(event: DragEvent) {
  dragging.value = false;
  selectFile(event.dataTransfer?.files?.[0]);
}

function upload() {
  if (!form.file) return;
  form.post(`/media${pageQuery.value}`, {
    forceFormData: true,
    preserveScroll: true,
    onSuccess: () => {
      form.reset();
      clientError.value = "";
    },
    onError: (errors) => {
      if (!("file" in errors)) {
        clientError.value = "Failed to upload the file. Make sure media storage is available, then try again.";
      }
    },
  });
}

function remove(asset: Asset) {
  deleteTarget.value = asset;
}

function confirmRemove() {
  const asset = deleteTarget.value;
  if (!asset || deleting.value) return;
  router.post(`/media/${asset.public_id}/delete${pageQuery.value}`, undefined, {
    preserveScroll: true,
    onStart: () => (deleting.value = true),
    onFinish: () => {
      deleting.value = false;
      deleteTarget.value = null;
    },
  });
}

function humanSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function isImage(contentType: string) {
  return contentType.startsWith("image/");
}
</script>

<template>
  <Head title="Media" />
  <UDashboardPanel id="media">
    <template #header>
      <UDashboardNavbar title="Media">
        <template #leading>
          <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
        </template>
        <span class="text-muted hidden text-sm lg:block">Uploaded files and images.</span>
        <template #right>
          <AppNotificationBell />
          <div v-if="canManage">
            <UButton
              color="primary"
              :icon="icons.upload"
              label="Upload file"
              :loading="form.processing"
              @click="pickFile" />
          </div>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="space-y-6">
        <UFormField v-if="canManage" label="Upload file" :error="clientError || fieldMessage(form.errors.file)">
          <div
            class="border-default bg-elevated/20 hover:border-primary/50 w-full rounded-lg border border-dashed p-6 text-center transition-colors"
            :class="dragging ? 'border-primary bg-primary/5' : ''"
            @click="pickFile"
            @dragover.prevent="dragging = true"
            @dragleave.prevent="dragging = false"
            @drop.prevent="onDrop">
            <input ref="fileInput" type="file" class="hidden" :accept="ACCEPTED" @change="onFile" />
            <UIcon :name="icons.cloudUpload" class="text-muted mx-auto size-8" />
            <p class="text-highlighted mt-3 text-sm font-medium">Drag a file here or click to choose one</p>
            <p class="text-muted mt-1 text-xs">{{ ACCEPTED_LABEL }}</p>
            <div
              v-if="form.processing"
              class="bg-accented mx-auto mt-4 h-1.5 w-full max-w-xs overflow-hidden rounded-full">
              <div
                class="bg-primary h-full rounded-full transition-all"
                :style="{ width: `${form.progress?.percentage ?? 0}%` }" />
            </div>
          </div>
        </UFormField>

        <UEmpty
          v-if="props.assets.length === 0"
          :icon="icons.media"
          title="No files yet."
          :description="
            canManage ? 'Upload an image, PDF, text, or CSV file to get started.' : 'No files have been uploaded yet.'
          "
          variant="naked"
          :actions="
            canManage
              ? [{ label: 'Upload a file', icon: icons.upload, onClick: pickFile }]
              : [{ label: 'Back to dashboard', icon: icons.arrowLeft, to: '/dashboard' }]
          "
          class="py-12" />

        <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <UCard v-for="asset in props.assets" :key="asset.public_id" :ui="{ body: 'p-0' }">
            <div class="bg-elevated/50 flex h-40 items-center justify-center overflow-hidden">
              <img
                v-if="isImage(asset.content_type)"
                :src="asset.url"
                :alt="asset.filename"
                class="h-full w-full object-cover" />
              <UIcon v-else :name="icons.file" class="text-muted size-10" />
            </div>
            <div class="space-y-1 p-4">
              <p class="text-highlighted truncate text-sm font-medium" :title="asset.filename">
                {{ asset.filename }}
              </p>
              <p class="text-muted text-xs">
                {{ humanSize(asset.size) }} · <AppCellTime :value="asset.created_at" inline />
              </p>
              <div class="flex justify-end gap-2 pt-2">
                <UButton
                  :to="asset.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  size="xs"
                  color="neutral"
                  variant="outline"
                  :icon="icons.download"
                  label="Open" />
                <UButton
                  v-if="canManage"
                  size="xs"
                  color="error"
                  variant="outline"
                  :icon="icons.delete"
                  label="Delete"
                  :loading="deleting && deleteTarget?.public_id === asset.public_id"
                  @click="remove(asset)" />
              </div>
            </div>
          </UCard>
        </div>
        <AppPagination :page="props.page" url="/media" />
      </div>
    </template>
  </UDashboardPanel>

  <AppConfirmDialog
    :open="deleteTarget !== null"
    title="Delete file"
    :description="deleteTarget ? `${deleteTarget.filename} will be permanently deleted.` : ''"
    confirm-label="Delete file"
    :loading="deleting"
    @update:open="
      (value: boolean) => {
        if (!value) deleteTarget = null;
      }
    "
    @confirm="confirmRemove" />
</template>
