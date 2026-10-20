type UmamiConfig = {
  enabled?: boolean;
  script_url?: string;
  website_id?: string;
  domains?: string;
  auto_track?: boolean;
};

declare global {
  interface Window {
    umami?: {
      track: (
        event?: string | ((props: Record<string, unknown>) => Record<string, unknown>),
        data?: Record<string, unknown>,
      ) => void;
    };
  }
}

/**
 * Installs the Umami tracker once per document. The tracker observes the
 * History API, so Inertia navigations are recorded without an extra page-view
 * call (which would double count).
 */
export function installUmami(config?: UmamiConfig): void {
  if (!config?.enabled || !config.script_url || !config.website_id) {
    return;
  }
  if (document.querySelector("script[data-website-id]")) {
    return;
  }

  const script = document.createElement("script");
  script.async = true;
  script.src = config.script_url;
  script.dataset.websiteId = config.website_id;
  if (config.domains) {
    script.dataset.domains = config.domains;
  }
  if (config.auto_track === false) {
    script.dataset.autoTrack = "false";
  }
  document.head.appendChild(script);
}

/** Tracks a custom event when Umami is loaded. */
export function trackEvent(name: string, data?: Record<string, unknown>): void {
  window.umami?.track(name, data);
}
