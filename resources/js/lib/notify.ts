import type { Toast } from "@nuxt/ui/composables";

import { icons } from "./icons";
import { playNotificationSound, type NotificationSound } from "./sound";

/** Kinds that have a matching toast colour and chime. */
export type NotifyKind = Extract<NotificationSound, "success" | "error" | "info">;

const TOAST: Record<NotifyKind, { icon: string; color: "success" | "error" | "info"; duration: number }> = {
  success: { icon: icons.success, color: "success", duration: 4000 },
  error: { icon: icons.error, color: "error", duration: 6000 },
  info: { icon: icons.info, color: "info", duration: 5000 },
};

/** `duration: 0` keeps the toast open until the user acts. */
export interface NotifyOptions extends Pick<Partial<Toast>, "id" | "description" | "duration" | "actions" | "close"> {
  silent?: boolean;
}

/**
 * Pairs a chime with a toast so every reported outcome uses one path.
 */
export function useNotify() {
  const toast = useToast();

  return function notify(kind: NotifyKind, title: string, options: NotifyOptions = {}) {
    const defaults = TOAST[kind];
    if (!options.silent) {
      playNotificationSound(kind);
    }
    toast.add({
      id: options.id,
      title,
      description: options.description,
      actions: options.actions,
      close: options.close,
      icon: defaults.icon,
      color: defaults.color,
      duration: options.duration ?? defaults.duration,
    });
  };
}
