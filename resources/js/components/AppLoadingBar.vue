<script setup lang="ts">
import { router } from "@inertiajs/vue3";
import { onMounted, onUnmounted, ref } from "vue";

const visible = ref(false);
const value = ref(0);
let timer: number | undefined;

function clearTimer() {
  if (timer !== undefined) {
    window.clearInterval(timer);
    timer = undefined;
  }
}

function start() {
  visible.value = true;
  value.value = value.value > 0 ? value.value : 12;
  clearTimer();
  timer = window.setInterval(() => {
    value.value = Math.min(92, value.value + Math.random() * 10);
  }, 200);
}

function finish() {
  clearTimer();
  value.value = 100;
  window.setTimeout(() => {
    visible.value = false;
    value.value = 0;
  }, 260);
}

onMounted(() => {
  router.on("start", start);
  router.on("finish", finish);
});

onUnmounted(clearTimer);
</script>

<template>
  <UProgress v-if="visible" v-model="value" size="xs" color="primary" class="fixed inset-x-0 top-0 z-50" />
</template>
