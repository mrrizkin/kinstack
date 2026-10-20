/**
 * Client-side entry point for the Inertia + Vue 3 application.
 *
 * The Laravel Vite plugin owns the module graph and HMR boundary. The resolver
 * below keeps the page directory and naming convention for KinStack.
 */
import { createInertiaApp, router } from "@inertiajs/vue3";
import nuxtUiPlugin from "@nuxt/ui/vue-plugin";
import { resolvePageComponent } from "laravel-vite-plugin/inertia-helpers";
import type { DefineComponent } from "vue";
import { createApp, h } from "vue";

import "../css/global.css";
import { i18n } from "./i18n";
import { captureException, installGlitchTip } from "./lib/glitchtip";
import { installUmami } from "./lib/umami";

const pages = import.meta.glob<DefineComponent>("./pages/**/*.vue");
let csrfToken = "";

type IntegrationProps = {
  csrf_token?: string;
  observability?: {
    glitchtip?: Parameters<typeof installGlitchTip>[1];
  };
  analytics?: {
    umami?: Parameters<typeof installUmami>[0];
  };
};

createInertiaApp({
  title: (title) => (title ? `${title} · KinStack` : "KinStack"),

  resolve: (name: string) => resolvePageComponent(`./pages/${name}.vue`, pages),

  setup({ el, App, props, plugin }) {
    const pageProps = props.initialPage.props as IntegrationProps;
    const initialToken = pageProps.csrf_token;
    if (typeof initialToken === "string") {
      csrfToken = initialToken;
    }
    const app = createApp({ render: () => h(App, props) })
      .use(plugin)
      .use(i18n)
      .use(nuxtUiPlugin);

    app.config.errorHandler = (err, instance, info) => {
      console.error("Vue error:", err, info, instance);
      captureException(err);
    };

    installUmami(pageProps.analytics?.umami);
    installGlitchTip(app, pageProps.observability?.glitchtip);

    app.mount(el);
  },

  defaults: {
    visitOptions: () => ({ viewTransition: true }),
  },
});

router.on("navigate", (event) => {
  const token = event.detail.page.props.csrf_token;
  if (typeof token === "string") {
    csrfToken = token;
  }
});

router.on("before", (event) => {
  if (csrfToken) {
    event.detail.visit.headers["X-XSRF-TOKEN"] = csrfToken;
  }
});
