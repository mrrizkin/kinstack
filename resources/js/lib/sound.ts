import { ref, watch } from "vue";

import { parseSoundPreference } from "./contracts/preferences";

export type NotificationSound = "success" | "error" | "info" | "mute";

const STORAGE_KEY = "kinstack-sound";

const MARIMBA_MODULATION_RATIO = 4;
const MARIMBA_MODULATION_INDEX = 1;

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

/**
 * A wooden FM tone: a sine carrier modulated by its fourth harmonic with a
 * decaying index.
 */
function marimba(audio: AudioContext, frequency: number, offset: number, duration: number, peak = 0.06) {
  const start = audio.currentTime + offset;

  const carrier = audio.createOscillator();
  carrier.type = "sine";
  carrier.frequency.setValueAtTime(frequency, start);

  const modulator = audio.createOscillator();
  modulator.type = "sine";
  modulator.frequency.setValueAtTime(frequency * MARIMBA_MODULATION_RATIO, start);

  const modulation = audio.createGain();
  modulation.gain.setValueAtTime(frequency * MARIMBA_MODULATION_INDEX, start);
  modulation.gain.exponentialRampToValueAtTime(frequency * 0.01, start + duration * 0.7);
  modulator.connect(modulation);
  modulation.connect(carrier.frequency);

  const gain = audio.createGain();
  gain.gain.setValueAtTime(0.0001, start);
  gain.gain.exponentialRampToValueAtTime(peak, start + 0.004);
  gain.gain.exponentialRampToValueAtTime(0.0001, start + duration);

  carrier.connect(gain);
  gain.connect(audio.destination);

  carrier.start(start);
  modulator.start(start);
  carrier.stop(start + duration + 0.05);
  modulator.stop(start + duration + 0.05);
}

/**
 * Plays a short marimba chime for the given notification kind:
 * a rising pair for success, a single note for info, a falling pair for error.
 */
export function playNotificationSound(kind: NotificationSound = "info") {
  if (!enabled.value) return;
  previewNotificationSound(kind);
}

/**
 * Plays the chime regardless of the mute preference: used to preview the sound
 * when the user enables it from the settings page.
 */
export function previewNotificationSound(kind: NotificationSound = "info") {
  if (prefersReducedMotion()) return;
  const audio = resolveContext();
  if (!audio) return;

  if (kind === "success") {
    marimba(audio, 783.99, 0, 0.22, 0.06);
    marimba(audio, 1046.5, 0.11, 0.3, 0.055);
    return;
  }
  if (kind === "error") {
    marimba(audio, 440, 0, 0.22, 0.06);
    marimba(audio, 349.23, 0.11, 0.3, 0.055);
    return;
  }
  if (kind === "mute") {
    marimba(audio, 523.25, 0, 0.12, 0.05);
    return;
  }
  marimba(audio, 523.25, 0, 0.4, 0.07);
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
