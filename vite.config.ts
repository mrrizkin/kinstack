import { fileURLToPath, URL } from "node:url";

import inertia from "@inertiajs/vite";
import { maizzle } from "@maizzle/framework";
import ui from "@nuxt/ui/vite";
import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import laravel from "laravel-vite-plugin";
import { defineConfig } from "vite";
import vueDevTools from "vite-plugin-vue-devtools";

import { icons } from "./resources/js/lib/icons.ts";

export default defineConfig({
  plugins: [
    laravel({ input: ["resources/js/app.ts"], refresh: false }),
    inertia(),
    vue(),
    ui({
      router: "inertia",
      icon: {
        clientBundle: {
          scan: {
            globInclude: ["resources/js/**/*.{vue,ts}", "vite.config.ts"],
          },
        },
      },
      ui: {
        colors: {
          primary: "fuchsia",
          neutral: "neutral",
        },
        // Every value references the registry; Nuxt UI validates these keys.
        icons: {
          arrowDown: icons.arrowDown,
          arrowLeft: icons.arrowLeft,
          arrowRight: icons.arrowRight,
          arrowUp: icons.arrowUp,
          caution: icons.alert,
          check: icons.check,
          chevronDoubleLeft: icons.chevronDoubleLeft,
          chevronDoubleRight: icons.chevronDoubleRight,
          chevronDown: icons.chevronDown,
          chevronLeft: icons.chevronLeft,
          chevronRight: icons.chevronRight,
          chevronUp: icons.chevronUp,
          close: icons.close,
          copy: icons.copy,
          copyCheck: icons.copied,
          dark: icons.dark,
          drag: icons.drag,
          ellipsis: icons.more,
          error: icons.error,
          external: icons.external,
          eye: icons.eye,
          eyeOff: icons.eyeOff,
          file: icons.file,
          folder: icons.folder,
          folderOpen: icons.folderOpen,
          hash: icons.tag,
          info: icons.info,
          light: icons.light,
          loading: icons.loading,
          menu: icons.menu,
          minus: icons.minus,
          panelClose: icons.panelClose,
          panelOpen: icons.panelOpen,
          plus: icons.add,
          reload: icons.reload,
          search: icons.search,
          stop: icons.stop,
          star: icons.star,
          success: icons.success,
          system: icons.system,
          tip: icons.tip,
          upload: icons.upload,
          warning: icons.warning,
        },
        card: { defaultVariants: { variant: "subtle" } },
        alert: { defaultVariants: { variant: "subtle" } },
        empty: { defaultVariants: { variant: "subtle" } },
        input: { defaultVariants: { variant: "subtle", size: "sm" } },
        select: { defaultVariants: { variant: "subtle", size: "sm" } },
        textarea: { defaultVariants: { variant: "subtle", size: "sm" } },
        selectMenu: { defaultVariants: { variant: "subtle", size: "sm" } },
        inputMenu: { defaultVariants: { variant: "subtle", size: "sm" } },
        inputNumber: { defaultVariants: { variant: "subtle", size: "sm" } },
        inputTags: { defaultVariants: { variant: "subtle", size: "sm" } },
        inputDate: { defaultVariants: { variant: "subtle", size: "sm" } },
        inputTime: { defaultVariants: { variant: "subtle", size: "sm" } },
        pinInput: { defaultVariants: { variant: "subtle", size: "sm" } },
        button: { defaultVariants: { size: "sm" } },
        badge: { defaultVariants: { size: "sm" } },
        inputRating: { defaultVariants: { size: "sm" } },
        tabs: { defaultVariants: { size: "sm" } },
        checkbox: { defaultVariants: { size: "sm" } },
        checkboxGroup: { defaultVariants: { size: "sm" } },
        radioGroup: { defaultVariants: { size: "sm" } },
        switch: { defaultVariants: { size: "sm" } },
        slider: { defaultVariants: { size: "sm" } },
        stepper: { defaultVariants: { size: "sm" } },
        calendar: { defaultVariants: { size: "sm" } },
        colorPicker: { defaultVariants: { size: "sm" } },
        fileUpload: { defaultVariants: { size: "sm" } },
        formField: { defaultVariants: { size: "sm" } },
        fieldGroup: { defaultVariants: { size: "sm" } },
        dropdownMenu: { defaultVariants: { size: "sm" } },
        contextMenu: { defaultVariants: { size: "sm" } },
        commandPalette: { defaultVariants: { size: "sm" } },
        listbox: { defaultVariants: { size: "sm" } },
      },
    }),
    tailwindcss(),
    maizzle({
      root: "resources/js/emails",
      content: ["*.vue"],
      output: {
        path: "assets/emails/dist",
        extension: "html",
      },
      server: {
        port: 3001,
      },
    }),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./resources/js", import.meta.url)),
    },
  },
});
