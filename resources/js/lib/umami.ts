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
 * Installs the tracker once. It observes the History API, so Inertia navigations
 * are recorded without a second page-view call.
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
