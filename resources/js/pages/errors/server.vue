<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import BaseLayout from "@/layouts/BaseLayout.vue";
import { errorPageSchema } from "@/lib/contracts/pages";
import { usePageSchema } from "@/lib/page-props";

const { status, message, code } = usePageSchema(errorPageSchema);

defineOptions({ layout: [BaseLayout] });

function reload() {
  window.location.reload();
}
</script>

<template>
  <Head title="Something went wrong" />
  <main id="main-content" tabindex="-1" class="bg-elevated grid min-h-dvh place-items-center px-4">
    <div class="max-w-md text-center">
      <p class="text-error text-5xl font-bold">{{ status ?? 500 }}</p>
      <h1 class="text-highlighted mt-4 text-xl font-semibold">Something went wrong</h1>
      <p class="text-muted mt-2 text-sm">
        {{ message ?? "The request could not be processed. Please try again in a moment." }}
      </p>
      <UButton @click="reload" color="primary" class="mt-6">Reload</UButton>
    </div>
  </main>
</template>
