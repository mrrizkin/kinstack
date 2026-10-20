import { onScopeDispose, watch, type WatchSource } from "vue";

/**
 * Debounced watcher for live filters. Accepts a single source (ref, reactive
 * object) or an array of sources; the callback runs once the source stops
 * changing for `delay` ms, so typing does not fire a request per keystroke.
 * The pending timer is cleared when the component scope is disposed.
 */
export function useDebouncedWatch(sources: unknown, fn: () => void, delay = 350) {
  let timer: number | undefined;
  const stop = watch(
    sources as WatchSource,
    () => {
      if (timer !== undefined) window.clearTimeout(timer);
      timer = window.setTimeout(() => {
        timer = undefined;
        fn();
      }, delay);
    },
    { deep: true },
  );
  onScopeDispose(() => {
    if (timer !== undefined) window.clearTimeout(timer);
    stop();
  });
}
