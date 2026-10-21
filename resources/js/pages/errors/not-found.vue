<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { computed } from "vue";

import BaseLayout from "@/layouts/BaseLayout.vue";
import { usePageProps } from "@/lib/page-props";

defineProps<{ status?: number; message?: string }>();

defineOptions({ layout: [BaseLayout] });

const { actor } = usePageProps();
const homeHref = computed(() => (actor.value ? "/dashboard" : "/"));
const homeLabel = computed(() => (actor.value ? "Go to dashboard" : "Go to home"));
</script>

<template>
  <Head title="404" />
  <main id="main-content" tabindex="-1" class="bg-elevated grid min-h-dvh place-items-center px-4">
    <div class="max-w-md text-center">
      <p class="text-primary text-5xl font-bold">404</p>
      <h1 class="text-highlighted mt-4 text-xl font-semibold">Not found</h1>
      <p class="text-muted mt-2 text-sm">
        {{ message ?? "The page you are looking for is not available." }}
      </p>
      <a
        :href="homeHref"
        class="bg-primary text-inverted hover:bg-primary/90 mt-6 inline-block rounded-md px-4 py-2 text-sm font-semibold">
        {{ homeLabel }}
      </a>
    </div>
  </main>
</template>
