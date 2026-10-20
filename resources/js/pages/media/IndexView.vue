<script setup lang="ts">
import { Head, router, useForm } from "@inertiajs/vue3";
import { ref } from "vue";

import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { formatDateTime } from "@/lib/formatters";
import { usePageProps } from "@/lib/page-props";
import { PERMISSIONS, usePermissions } from "@/lib/permissions";

defineOptions({ layout: [BaseLayout, AppLayout] });

interface Asset {
  public_id: string;
  filename: string;
  content_type: string;
  size: number;
  url: string;
  created_at: string;
}

const props = defineProps<{ assets: Asset[] }>();

const { can } = usePermissions();
const { actor } = usePageProps();
const isSuperuser = Boolean(actor.value?.superuser);
const canManage = can(PERMISSIONS.mediaManage) && !isSuperuser;

const MAX_UPLOAD = 10 * 1024 * 1024;
const ACCEPTED = "image/jpeg,image/png,image/webp,image/gif,application/pdf,text/plain,text/csv";
const ACCEPTED_LABEL = "JPG, PNG, WebP, GIF, PDF, TXT, or CSV · up to 10 MB";

const form = useForm<{ file: File | null }>({ file: null });
const fileInput = ref<HTMLInputElement | null>(null);
const dragging = ref(false);
const clientError = ref("");

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
  form.post("/media", {
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
  if (!window.confirm(`Delete ${asset.filename}?`)) return;
  router.post(`/media/${asset.public_id}/delete`, undefined, { preserveScroll: true });
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
              icon="i-bi-cloud-arrow-up"
              label="Upload file"
              :loading="form.processing"
              @click="pickFile" />
          </div>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="space-y-6">
        <UAlert
          v-if="clientError || form.errors.file"
          color="error"
          variant="subtle"
          title="Upload failed"
          :description="clientError || String(form.errors.file)" />

        <div
          v-if="canManage"
          class="border-default bg-elevated/20 hover:border-primary/50 rounded-lg border border-dashed p-6 text-center transition-colors"
          :class="dragging ? 'border-primary bg-primary/5' : ''"
          @click="pickFile"
          @dragover.prevent="dragging = true"
          @dragleave.prevent="dragging = false"
          @drop.prevent="onDrop">
          <input ref="fileInput" type="file" class="hidden" :accept="ACCEPTED" @change="onFile" />
          <UIcon name="i-material-symbols:cloud-upload-outline-rounded" class="text-muted mx-auto size-8" />
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

        <div v-if="props.assets.length === 0" class="border-default rounded-lg border border-dashed py-16 text-center">
          <UIcon name="i-bi-images" class="text-muted mx-auto size-8" />
          <p class="text-muted mt-3 text-sm">No files yet.</p>
        </div>

        <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <UCard v-for="asset in props.assets" :key="asset.public_id" :ui="{ body: 'p-0' }">
            <div class="bg-elevated/50 flex h-40 items-center justify-center overflow-hidden">
              <img
                v-if="isImage(asset.content_type)"
                :src="asset.url"
                :alt="asset.filename"
                class="h-full w-full object-cover" />
              <UIcon v-else name="i-bi-file-earmark-text" class="text-muted size-10" />
            </div>
            <div class="space-y-1 p-4">
              <p class="text-highlighted truncate text-sm font-medium" :title="asset.filename">
                {{ asset.filename }}
              </p>
              <p class="text-muted text-xs">{{ humanSize(asset.size) }} · {{ formatDateTime(asset.created_at) }}</p>
              <div class="flex justify-end gap-2 pt-2">
                <UButton
                  :to="asset.url"
                  target="_blank"
                  size="xs"
                  color="neutral"
                  variant="outline"
                  icon="i-bi-download"
                  label="Open" />
                <UButton
                  v-if="canManage"
                  size="xs"
                  color="error"
                  variant="outline"
                  icon="i-bi-trash"
                  label="Delete"
                  @click="remove(asset)" />
              </div>
            </div>
          </UCard>
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
