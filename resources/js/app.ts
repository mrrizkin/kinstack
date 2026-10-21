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
import { sharedPagePropsSchema } from "./lib/contracts/page-props";
import { captureException, installGlitchTip } from "./lib/glitchtip";
import { installUmami } from "./lib/umami";

const pages = import.meta.glob<DefineComponent>("./pages/**/*.vue");
let csrfToken = "";

createInertiaApp({
  title: (title) => (title ? `${title} · KinStack` : "KinStack"),

  resolve: (name: string) => resolvePageComponent(`./pages/${name}.vue`, pages),

  progress: {
    color: "var(--ui-primary)",
    showSpinner: false,
  },

  setup({ el, App, props, plugin }) {
    const pageProps = sharedPagePropsSchema.parse(props.initialPage.props);
    if (pageProps.csrf_token) {
      csrfToken = pageProps.csrf_token;
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
    // The Sentry SDK is imported lazily, so mount after it initializes; Vue
    // integrations must be installed before the app is mounted.
    void installGlitchTip(app, pageProps.observability?.glitchtip).finally(() => app.mount(el));
  },

  defaults: {
    visitOptions: () => ({ viewTransition: true }),
  },
});

router.on("navigate", (event) => {
  const parsed = sharedPagePropsSchema.safeParse(event.detail.page.props);
  if (parsed.success && parsed.data.csrf_token) {
    csrfToken = parsed.data.csrf_token;
  }
});

router.on("before", (event) => {
  if (csrfToken) {
    event.detail.visit.headers["X-XSRF-TOKEN"] = csrfToken;
  }
});
