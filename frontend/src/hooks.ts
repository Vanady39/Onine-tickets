import { useCallback, useEffect, useRef, useState, type DependencyList } from 'react';
import { errorMessage } from './api';

export interface AsyncState<T> {
  data: T | undefined;
  error: string | null;
  loading: boolean;
  /** Loads again; keeps the current data visible while loading. */
  reload: () => void;
  setData: (d: T) => void;
}

/** useAsync runs `fn` when `deps` change and tracks loading/error state; stale results are dropped. */
export function useAsync<T>(fn: () => Promise<T>, deps: DependencyList): AsyncState<T> {
  const [data, setData] = useState<T | undefined>(undefined);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);
  const run = useRef(0);

  useEffect(() => {
    const id = ++run.current;
    setLoading(true);
    setError(null);
    fn()
      .then((d) => {
        if (id === run.current) setData(d);
      })
      .catch((e: unknown) => {
        if (id === run.current) setError(errorMessage(e));
      })
      .finally(() => {
        if (id === run.current) setLoading(false);
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick]);

  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { data, error, loading, reload, setData };
}

/** useNow re-renders every `intervalMs` and returns the current time in ms. */
export function useNow(intervalMs = 1000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(t);
  }, [intervalMs]);
  return now;
}

/** useTitle sets the browser tab title. */
export function useTitle(title: string | undefined): void {
  useEffect(() => {
    document.title = title ? `${title} · Online Tickets` : 'Online Tickets';
  }, [title]);
}
