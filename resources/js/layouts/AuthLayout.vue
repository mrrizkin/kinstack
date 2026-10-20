<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import AuthBrandPanel from "@/components/auth/AuthBrandPanel.vue";

defineProps<{ title?: string; subtitle?: string }>();

const { t } = useI18n();

const greeting = computed(() => {
  const hour = new Date().getHours();
  if (hour < 11) return t("shell.auth.greetingMorning");
  if (hour < 15) return t("shell.auth.greetingAfternoon");
  if (hour < 19) return t("shell.auth.greetingEvening");
  return t("shell.auth.greetingNight");
});
</script>

<template>
  <main
    id="main-content"
    tabindex="-1"
    class="bg-default text-default min-h-dvh lg:grid lg:grid-cols-[minmax(0,1fr)_clamp(380px,40vw,600px)]">
    <section class="relative flex min-h-dvh flex-col">
      <header class="flex items-center justify-between gap-4 px-5 pt-5 sm:px-8 lg:px-10 xl:px-14">
        <UButton
          to="/"
          icon="i-material-symbols:arrow-back-rounded"
          :label="t('shell.auth.back')"
          color="neutral"
          variant="ghost"
          size="sm"
          class="-ms-2" />
        <UColorModeButton color="neutral" variant="ghost" />
      </header>

      <div class="flex flex-1 items-center justify-center px-5 py-8 sm:px-8 lg:px-10 xl:px-14">
        <div class="w-full max-w-md">
          <div class="mb-8 flex flex-col items-center gap-3 text-center">
            <span
              class="bg-primary text-inverted grid size-11 place-items-center rounded-xl text-sm font-bold"
              aria-hidden="true">
              K
            </span>
            <p class="text-muted text-sm">
              <span class="text-highlighted font-semibold">{{ greeting }}</span
              >, {{ t("shell.auth.welcome") }}
            </p>
          </div>

          <div v-if="title" class="mb-6 text-center">
            <h1 class="text-highlighted text-2xl font-semibold tracking-tight">{{ title }}</h1>
            <p v-if="subtitle" class="text-muted mt-1.5 text-sm text-balance">{{ subtitle }}</p>
          </div>

          <slot />
        </div>
      </div>

      <footer class="text-dimmed px-5 pb-6 text-center text-xs leading-5 text-balance sm:px-8 lg:px-10 xl:px-14">
        {{ t("shell.auth.footer") }}
      </footer>
    </section>

    <AuthBrandPanel />
  </main>
</template>
