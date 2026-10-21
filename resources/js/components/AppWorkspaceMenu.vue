<script setup lang="ts">
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { icons } from "@/lib/icons";

const props = defineProps<{ collapsed?: boolean }>();
const { t } = useI18n();

const workspace = {
  label: "KinStack",
  avatar: { alt: "K", size: "sm" as const },
};

const items = computed<DropdownMenuItem[][]>(() => {
  const groups: DropdownMenuItem[][] = [
    [
      {
        ...workspace,
        type: "label",
      },
    ],
    [
      {
        label: t("shell.workspace.dashboard"),
        icon: icons.dashboard,
        to: "/dashboard",
      },
    ],
    [
      {
        label: t("shell.workspace.help"),
        icon: icons.help,
        to: "/help",
      },
    ],
  ];

  return groups;
});
</script>

<template>
  <UDropdownMenu
    :items="items"
    :content="{ align: 'center', collisionPadding: 12 }"
    :ui="{ content: props.collapsed ? 'w-40' : 'w-(--reka-dropdown-menu-trigger-width)' }">
    <UButton
      v-bind="{
        ...workspace,
        label: props.collapsed ? undefined : workspace.label,
        trailingIcon: props.collapsed ? undefined : icons.unfoldMore,
      }"
      color="neutral"
      variant="ghost"
      block
      :square="props.collapsed"
      class="data-[state=open]:bg-elevated"
      :class="[!props.collapsed && 'py-2']"
      :ui="{ trailingIcon: 'text-dimmed' }" />
  </UDropdownMenu>
</template>
