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
  if (typeof window === "undefined") {
    return undefined;
  }
  const Constructor =
    window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Constructor) {
    return undefined;
  }
  if (!context) {
    context = new Constructor();
  }
  if (context.state === "suspended") {
    void context.resume();
  }
  return context;
}

if (typeof window !== "undefined") {
  const warm = () => {
    resolveContext();
    window.removeEventListener("pointerdown", warm);
    window.removeEventListener("keydown", warm);
  };
  window.addEventListener("pointerdown", warm, { once: true });
  window.addEventListener("keydown", warm, { once: true });
}

interface ChimeNote {
  frequency: number;
  offset: number;
  duration: number;
  peak?: number;
}

/** A wooden FM tone: a sine carrier modulated by its fourth harmonic with a decaying index. */
function marimba(audio: AudioContext, note: ChimeNote) {
  const { frequency, offset, duration, peak = 0.06 } = note;
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

/** Plays the chime for a notification kind: a rising pair for success, a falling
 * pair for error. */
export function playNotificationSound(kind: NotificationSound = "info") {
  if (!enabled.value) {
    return;
  }
  previewNotificationSound(kind);
}

// The chime per kind, as data: a rising pair for success, a falling pair for
// error, one long note for info, one short note for mute.
const CHIMES: Record<NotificationSound, ChimeNote[]> = {
  success: [
    { frequency: 783.99, offset: 0, duration: 0.22 },
    { frequency: 1046.5, offset: 0.11, duration: 0.3, peak: 0.055 },
  ],
  error: [
    { frequency: 440, offset: 0, duration: 0.22 },
    { frequency: 349.23, offset: 0.11, duration: 0.3, peak: 0.055 },
  ],
  info: [{ frequency: 523.25, offset: 0, duration: 0.4, peak: 0.07 }],
  mute: [{ frequency: 523.25, offset: 0, duration: 0.12, peak: 0.05 }],
};

export function previewNotificationSound(kind: NotificationSound = "info") {
  if (prefersReducedMotion()) {
    return;
  }
  const audio = resolveContext();
  if (!audio) {
    return;
  }
  for (const note of CHIMES[kind]) {
    marimba(audio, note);
  }
}

export function useNotificationSound() {
  watch(enabled, (value) => {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, value ? "on" : "off");
    }
  });

  return { enabled };
}
