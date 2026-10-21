import { icons } from "./icons";
import { playNotificationSound, type NotificationSound } from "./sound";

/** Kinds that have a matching toast colour and chime. */
export type NotifyKind = Extract<NotificationSound, "success" | "error" | "info">;

const TOAST: Record<NotifyKind, { icon: string; color: "success" | "error" | "info"; duration: number }> = {
  success: { icon: icons.success, color: "success", duration: 4000 },
  error: { icon: icons.error, color: "error", duration: 6000 },
  info: { icon: icons.info, color: "info", duration: 5000 },
};

/**
 * Pairs a chime with a toast so every reported outcome uses one path. Call it
 * from `<script setup>`; the returned function may be used later.
 */
export function useNotify() {
  const toast = useToast();

  return function notify(kind: NotifyKind, title: string) {
    playNotificationSound(kind);
    const options = TOAST[kind];
    toast.add({
      title,
      icon: options.icon,
      color: options.color,
      duration: options.duration,
    });
  };
}
