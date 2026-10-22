<script setup lang="ts">
import { Head } from "@inertiajs/vue3";

import BaseLayout from "@/layouts/BaseLayout.vue";
import { systemStatePageSchema } from "@/lib/contracts/pages";
import { usePageSchema } from "@/lib/page-props";

const { surface, topics, shortcuts } = usePageSchema(systemStatePageSchema);
defineOptions({ layout: [BaseLayout] });
</script>

<template>
  <Head :title="surface" />
  <div class="mx-auto max-w-2xl px-4 py-16">
    <h1 class="text-highlighted text-2xl font-semibold capitalize">
      {{ surface.replace("system-", "").replace("-", " ") }}
    </h1>
    <ul class="mt-6 space-y-2">
      <li
        v-for="topic in topics"
        :key="topic"
        class="border-default bg-default text-toned rounded-lg border px-4 py-3 text-sm">
        {{ topic }}
      </li>
    </ul>
    <dl v-if="shortcuts.length" class="mt-8 space-y-2">
      <div v-for="shortcut in shortcuts" :key="shortcut.key" class="flex items-center justify-between text-sm">
        <dt class="text-muted font-mono">{{ shortcut.key }}</dt>
        <dd class="text-toned">{{ shortcut.action }}</dd>
      </div>
    </dl>
    <a href="/" class="text-primary mt-8 inline-block text-sm font-medium hover:underline"> Back to dashboard </a>
  </div>
</template>
