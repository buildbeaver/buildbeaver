import React, { useState } from 'react';
import { TickContext } from './tick.context';

export function TickProvider(props: any): JSX.Element {
  const [tick, setTick] = useState(true);

  const flip = (): void => {
    setTick(!tick);
  };

  const providerValue = {
    tick,
    flip
  };

  return <TickContext.Provider value={providerValue}>{props.children}</TickContext.Provider>;
}
