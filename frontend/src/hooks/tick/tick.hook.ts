import { useEffect, useState } from 'react';

/**
 * Lightweight hook for running useEffect in a timed loop inside components.
 * @param intervalMs - Tick this often, in milliseconds.
 * @param shouldTick - Set this to false to pause ticking.
 */
export function useTick(intervalMs: number, shouldTick: boolean): boolean {
  const [tick, setTick] = useState(true);

  useEffect(() => {
    let interval: NodeJS.Timer;

    if (shouldTick) {
      interval = setInterval(() => {
        setTick(!tick);
      }, intervalMs);
    }

    return () => clearInterval(interval);
  }, [intervalMs, shouldTick, tick]);

  return tick;
}
