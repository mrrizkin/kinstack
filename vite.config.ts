import { fileURLToPath, URL } from "node:url";

import inertia from "@inertiajs/vite";
import { maizzle } from "@maizzle/framework";
import ui from "@nuxt/ui/vite";
import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import laravel from "laravel-vite-plugin";
import { defineConfig } from "vite";
import vueDevTools from "vite-plugin-vue-devtools";

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
        icons: {
          arrowDown: "i-material-symbols:arrow-downward-rounded",
          arrowLeft: "i-material-symbols:arrow-back-rounded",
          arrowRight: "i-material-symbols:arrow-forward-rounded",
          arrowUp: "i-material-symbols:arrow-upward-rounded",
          caution: "i-material-symbols:error-outline-rounded",
          check: "i-material-symbols:check-rounded",
          chevronDoubleLeft: "i-material-symbols:keyboard-double-arrow-left-rounded",
          chevronDoubleRight: "i-material-symbols:keyboard-double-arrow-right-rounded",
          chevronDown: "i-material-symbols:keyboard-arrow-down-rounded",
          chevronLeft: "i-material-symbols:chevron-left-rounded",
          chevronRight: "i-material-symbols:chevron-right-rounded",
          chevronUp: "i-material-symbols:keyboard-arrow-up-rounded",
          close: "i-material-symbols:close-rounded",
          copy: "i-material-symbols:content-copy-outline-rounded",
          copyCheck: "i-material-symbols:inventory-rounded",
          dark: "i-material-symbols:dark-mode-outline-rounded",
          drag: "i-material-symbols:drag-indicator",
          ellipsis: "i-material-symbols:more-horiz",
          error: "i-material-symbols:cancel-outline-rounded",
          external: "i-material-symbols:arrow-outward-rounded",
          eye: "i-material-symbols:visibility-outline-rounded",
          eyeOff: "i-material-symbols:visibility-off-outline-rounded",
          file: "i-material-symbols:description-outline-rounded",
          folder: "i-material-symbols:folder-outline-rounded",
          folderOpen: "i-material-symbols:folder-open-outline-rounded",
          hash: "i-material-symbols:tag-rounded",
          info: "i-material-symbols:info-outline-rounded",
          light: "i-material-symbols:light-mode-outline-rounded",
          loading: "i-material-symbols:progress-activity",
          menu: "i-material-symbols:menu-rounded",
          minus: "i-material-symbols:remove-rounded",
          panelClose: "i-material-symbols:left-panel-close-outline-rounded",
          panelOpen: "i-material-symbols:left-panel-open-outline-rounded",
          plus: "i-material-symbols:add-rounded",
          reload: "i-material-symbols:refresh-rounded",
          search: "i-material-symbols:search-rounded",
          stop: "i-material-symbols:stop-outline-rounded",
          star: "i-material-symbols:star-outline-rounded",
          success: "i-material-symbols:check-circle-outline-rounded",
          system: "i-material-symbols:desktop-windows-outline-rounded",
          tip: "i-material-symbols:lightbulb-outline-rounded",
          upload: "i-material-symbols:upload-rounded",
          warning: "i-material-symbols:warning-outline-rounded",
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
