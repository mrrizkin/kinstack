import { ref, watch } from "vue";

import { parseSoundPreference } from "./contracts/preferences";

export type NotificationSound = "success" | "error" | "info" | "mute";

const STORAGE_KEY = "kinstack-sound";

const enabled = ref(
  typeof window === "undefined" ? true : parseSoundPreference(window.localStorage.getItem(STORAGE_KEY)) === "on",
);

let context: AudioContext | undefined;

const prefersReducedMotion = () =>
  typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

function resolveContext(): AudioContext | undefined {
  if (typeof window === "undefined") return undefined;
  const Constructor =
    window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Constructor) return undefined;
  if (!context) {
    context = new Constructor();
  }
  if (context.state === "suspended") {
    void context.resume();
  }
  return context;
}

// Browsers only allow audio after a user gesture: warm the context on the first one.
if (typeof window !== "undefined") {
  const warm = () => {
    resolveContext();
    window.removeEventListener("pointerdown", warm);
    window.removeEventListener("keydown", warm);
  };
  window.addEventListener("pointerdown", warm, { once: true });
  window.addEventListener("keydown", warm, { once: true });
}

function tone(audio: AudioContext, frequency: number, offset: number, duration: number, peak = 0.05) {
  const oscillator = audio.createOscillator();
  const gain = audio.createGain();
  const start = audio.currentTime + offset;

  oscillator.type = "sine";
  oscillator.frequency.setValueAtTime(frequency, start);
  gain.gain.setValueAtTime(0.0001, start);
  gain.gain.exponentialRampToValueAtTime(peak, start + 0.015);
  gain.gain.exponentialRampToValueAtTime(0.0001, start + duration);

  oscillator.connect(gain);
  gain.connect(audio.destination);
  oscillator.start(start);
  oscillator.stop(start + duration + 0.05);
}

/**
 * Plays a short chime for the given notification kind (no-op when muted).
 */
export function playNotificationSound(kind: NotificationSound = "info") {
  if (!enabled.value) return;
  previewNotificationSound(kind);
}

/**
 * Plays the chime regardless of the mute preference: used to preview the sound
 * when the user toggles it in the user menu.
 */
function previewNotificationSound(kind: NotificationSound = "info") {
  if (prefersReducedMotion()) return;
  const audio = resolveContext();
  if (!audio) return;

  if (kind === "success") {
    tone(audio, 880, 0, 0.12);
    tone(audio, 1318.5, 0.09, 0.16);
    return;
  }
  if (kind === "error") {
    tone(audio, 320, 0, 0.18, 0.06);
    tone(audio, 240, 0.11, 0.22, 0.05);
    return;
  }
  if (kind === "mute") {
    tone(audio, 440, 0, 0.12, 0.04);
    return;
  }
  tone(audio, 659.3, 0, 0.14);
}

/** Shared mute preference for the notification chime. */
export function useNotificationSound() {
  watch(enabled, (value) => {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, value ? "on" : "off");
    }
  });

  return { enabled };
}
