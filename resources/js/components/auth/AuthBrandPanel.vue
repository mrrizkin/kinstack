<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const features = computed(() => [
  t("shell.brand.featureRbac"),
  t("shell.brand.featureAudit"),
  t("shell.brand.featureMfa"),
]);

const year = new Date().getFullYear();
</script>

<template>
  <aside class="relative hidden overflow-hidden bg-neutral-950 text-neutral-50 lg:flex">
    <div class="auth-grid absolute inset-0 opacity-60" aria-hidden="true" />
    <div class="absolute -top-32 -right-24 size-96 rounded-full bg-white/10 blur-3xl" aria-hidden="true" />
    <div class="absolute -bottom-40 -left-24 size-96 rounded-full bg-white/5 blur-3xl" aria-hidden="true" />

    <div class="relative z-10 flex w-full flex-col justify-between p-10 xl:p-14">
      <div class="flex items-center gap-3">
        <span
          class="grid size-10 place-items-center rounded-xl bg-white text-sm font-bold text-neutral-950"
          aria-hidden="true">
          K
        </span>
        <span class="text-lg font-semibold tracking-tight">KinStack</span>
      </div>

      <div class="max-w-md">
        <p class="text-xs font-medium tracking-[0.2em] text-neutral-400 uppercase">{{ t("shell.brand.eyebrow") }}</p>
        <h2 class="mt-4 text-4xl leading-[1.05] font-semibold tracking-tight text-balance xl:text-5xl">
          {{ t("shell.brand.headline") }}
        </h2>
        <p class="mt-5 text-sm leading-6 text-neutral-400">
          {{ t("shell.brand.description") }}
        </p>

        <ul class="mt-8 space-y-3">
          <li v-for="feature in features" :key="feature" class="flex items-center gap-3 text-sm">
            <span class="grid size-5 shrink-0 place-items-center rounded-full bg-white/10">
              <UIcon name="i-material-symbols:check-rounded" class="size-3.5" />
            </span>
            <span class="text-neutral-300">{{ feature }}</span>
          </li>
        </ul>
      </div>

      <p class="text-xs text-neutral-500">{{ t("shell.brand.copyright", { year }) }}</p>
    </div>
  </aside>
</template>

<style scoped>
.auth-grid {
  background-image:
    linear-gradient(to right, rgb(255 255 255 / 0.06) 1px, transparent 1px),
    linear-gradient(to bottom, rgb(255 255 255 / 0.06) 1px, transparent 1px);
  background-size: 56px 56px;
  mask-image: radial-gradient(ellipse 90% 80% at 60% 40%, black 30%, transparent 75%);
}
</style>
