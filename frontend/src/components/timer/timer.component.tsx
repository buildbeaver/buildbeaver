import React, { useEffect, useState } from 'react';
import { ITimings } from '../../interfaces/timings.interface';
import { useTick } from '../../hooks/tick/tick.hook';
import { buildDuration } from '../../utils/build.utils';

interface Props {
  className?: string;
  timings: ITimings;
}

const isRunning = (timings: ITimings): boolean => {
  return !timings.canceled_at && !timings.finished_at;
};

/**
 * Optimistically ticks every second to simulate having real time data when only given a stale timings object.
 */
export function Timer(props: Props): JSX.Element {
  const { className, timings } = props;
  const [duration, setDuration] = useState<string>();
  const [shouldTick, setShouldTick] = useState(isRunning(timings));
  const tick = useTick(1000, shouldTick);

  useEffect(() => {
    setDuration(buildDuration(timings));
    setShouldTick(isRunning(timings));
  }, [tick]);

  return <span className={`whitespace-nowrap font-mono ${className}`}>{duration}</span>;
}
