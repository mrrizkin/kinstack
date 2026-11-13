import type { App } from "vue";

type GlitchTipConfig = {
  dsn?: string;
  environment?: string;
  release?: string;
  traces_sample_rate?: number;
};

type SentryModule = typeof import("@sentry/vue");

let sentry: SentryModule | null = null;

/**
 * The SDK is loaded lazily so a build without a DSN does not pay for it in the
 * main bundle. An empty DSN disables error tracking.
 */
export async function installGlitchTip(app: App, config?: GlitchTipConfig): Promise<void> {
  const dsn = config?.dsn?.trim();
  if (!dsn) {
    return;
  }

  try {
    const module = await import("@sentry/vue");
    sentry = module;
    module.init({
      app,
      dsn,
      environment: config?.environment || undefined,
      release: config?.release || undefined,
      tracesSampleRate: config?.traces_sample_rate ?? 0,
      dataCollection: { userInfo: false, cookies: false, httpBodies: [] },
      integrations: [module.browserTracingIntegration()],
    });
  } catch (error) {
    // Observability must never break the application.
    console.error("GlitchTip could not be initialized:", error);
  }
}

export function captureException(error: unknown): void {
  sentry?.captureException(error);
}
