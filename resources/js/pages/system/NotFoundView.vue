<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import BaseLayout from "@/layouts/BaseLayout.vue";
import { errorPageSchema } from "@/lib/contracts/pages";
import { usePageSchema } from "@/lib/page-props";

const { status, message } = usePageSchema(errorPageSchema);
defineOptions({ layout: [BaseLayout] });
</script>

<template>
  <Head title="Not found" />
  <div class="bg-elevated grid min-h-dvh place-items-center px-4">
    <div class="max-w-md text-center">
      <p class="text-primary text-5xl font-bold">{{ status ?? 404 }}</p>
      <h1 class="text-highlighted mt-4 text-xl font-semibold">Page not found</h1>
      <p class="text-muted mt-2 text-sm">
        {{ message ?? "The page you are looking for is not available." }}
      </p>
      <a
        href="/"
        class="bg-primary hover:bg-primary/90 mt-6 inline-block rounded-md px-4 py-2 text-sm font-semibold text-white">
        Back to dashboard
      </a>
    </div>
  </div>
</template>
