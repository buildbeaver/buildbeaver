import React from 'react';

interface Context {
  tick: boolean;
  flip: () => void;
}

export const TickContext = React.createContext<Context>({
  tick: true,
  flip: () => {}
});
